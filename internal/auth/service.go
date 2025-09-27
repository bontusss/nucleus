// Service provides authentication and authorization functionalities for the hotel ERP system.
// It manages user registration, login, JWT token generation and refresh, session management,
// email verification, password reset, role and permission management, and activity logging.
//
// Fields:
//   - queries: Database queries interface for user, role, and token operations.
//   - jwtSecret: Secret key for signing JWT access tokens.
//   - jwtRefreshSecret: Secret key for signing JWT refresh tokens (can fallback to jwtSecret).
//   - accessExpiry: Duration for which access tokens are valid.
//   - refreshExpiry: Duration for which refresh tokens are valid.
//   - redis: Redis client for caching and token blacklisting.
//
// Main Methods:
//   - RegisterAdmin: Registers a new admin user with hashed password.
//   - SetEmailVerification: Sets email verification code and expiry for a user.
//   - VerifyEmailCode: Verifies the email code and marks email as verified if valid.
//   - Login: Authenticates a user by email or username, returns access and refresh tokens.
//   - RefreshToken: Rotates refresh tokens and issues new access tokens.
//   - Logout: Blacklists a JWT token until its expiry.
//   - IsTokenBlacklisted: Checks if a JWT token is blacklisted.
//   - HasPermission: Checks if a user has a required permission.
//   - RevokeAllUserSessions: Revokes all refresh tokens for a user and clears cache.
//   - User Management: CreateUser, UpdateUser, DeleteUser, ResetPassword.
//   - Role Management: CreateRole, UpdateRole, DeleteRole, AddPermissionToRole, RemovePermissionFromRole.
//   - GetUserByID, GetUserByEmail, GetUserByUsername: Fetches user details, with Redis caching.
//   - ListUsers, ListRoles, GetRolePermissions: Lists users, roles, and permissions.
//   - Logging: LogUserActivity, LogLogin for auditing user actions and login attempts.
//
// Internal Utilities:
//   - generateRefreshToken: Generates a secure random refresh token.
//   - cleanExpiredTokens: Periodically cleans up expired refresh tokens from the database.
//
// Error Handling:
//   - ErrInvalidCredentials: Returned when authentication fails.
//   - ErrUserInactive: Returned when a user is inactive.
//
// This service is designed to be thread-safe and efficient, leveraging Redis for caching and token blacklisting,
// and supports extensible role-based access control for fine-grained permission management.
package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	db "herp/db/sqlc"
	"herp/internal/utils"
	"herp/pkg/jwt"
	"herp/pkg/monitoring/logging"
	"herp/pkg/ratelimit"
	"herp/pkg/redis"
	"log"
	"slices"
	"time"

	r "github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive       = errors.New("user is inactive")
	ErrUserNotFound       = errors.New("user not found")
)

type Service struct {
	queries            Querier
	jwtSecret          string
	jwtRefreshSecret   string
	accessExpiry       time.Duration
	refreshExpiry      time.Duration
	redis              *redis.Redis
	rClient            *r.Client
	rateLimiter        *ratelimit.RateLimiter
	loginRateLimit     int
	loginRateWindow    time.Duration
	loginBlockDuration time.Duration
	ipRateLimit        int
	db                 *sql.DB
	logger             *logging.Logger
}

func NewService(queries Querier, jwtSecret, jwtRefreshSecret string, accessExpiry, refreshExpiry time.Duration, redis *redis.Redis, redisClient *r.Client, loginRateLimit, loginRateWindow, loginBlockDuration, ipRateLimit int, db *sql.DB, logger *logging.Logger) *Service {
	if jwtRefreshSecret == "" {
		jwtRefreshSecret = jwtSecret // Fallback to same secret if not provided
	}
	rateLimiter := ratelimit.NewRateLimit(redisClient)
	return &Service{
		queries:            queries,
		jwtSecret:          jwtSecret,
		accessExpiry:       accessExpiry,
		refreshExpiry:      refreshExpiry,
		jwtRefreshSecret:   jwtRefreshSecret,
		redis:              redis,
		rClient:            redisClient,
		rateLimiter:        rateLimiter,
		loginRateLimit:     loginRateLimit,
		loginRateWindow:    time.Duration(loginRateWindow) * time.Minute,
		loginBlockDuration: time.Duration(loginBlockDuration) * time.Minute,
		ipRateLimit:        ipRateLimit,
		db:                 db,
		logger:             logger,
	}
}

// Generate random refresh token
func generateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Check and apply rate limiting
func (s *Service) checkRateLimits(ctx context.Context, username, ipAddress string) error {
	// Check IP-based rate limiting
	ipKey := fmt.Sprintf("rate_limit:ip:%s", ipAddress)
	blocked, ttl, err := s.rateLimiter.IsKeyBlocked(ctx, ipKey)
	if err != nil {
		return err
	}
	if blocked {
		return fmt.Errorf("IP temporarily blocked. Try again in %v", ttl)
	}

	// Check username-based rate limiting
	userKey := fmt.Sprintf("rate_limit:user:%s", username)
	blocked, ttl, err = s.rateLimiter.IsKeyBlocked(ctx, userKey)
	if err != nil {
		return err
	}
	if blocked {
		return fmt.Errorf("account temporarily locked. Try again in %v", ttl)
	}

	// Check IP rate limit for general requests
	exceeded, _, timeLeft, err := s.rateLimiter.Check(
		ctx,
		fmt.Sprintf("ip_requests:%s", ipAddress),
		s.ipRateLimit,
		time.Minute,
	)
	if err != nil {
		return err
	}
	if exceeded {
		return fmt.Errorf("too many requests. Try again in %v", timeLeft)
	}

	return nil
}

func (s *Service) RegisterAdmin(ctx context.Context, username, email, password, first_name, last_name string) (db.Admin, error) {
	log.Println("Registering new admin user:", username, email)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		s.logger.Error("error hashing password: ", err)
		return db.Admin{}, err
	}

	q, ok := s.queries.(*db.Queries)
	if !ok {
		return db.Admin{}, fmt.Errorf("invalid queries implementation")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.logger.Error("error starting transaction: ", err)
		return db.Admin{}, err
	}

	defer func() {
		if err != nil {
			s.logger.Error("rolling back transaction due to error: ", err)
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	log.Println("Creating admin user in database")
	txQueries := q.WithTx(tx)

	log.Println("Admin user details:", username, email, first_name, last_name)
	user, err := txQueries.CreateAdmin(ctx, db.CreateAdminParams{
		Username:     username,
		Email:        email,
		FirstName:    first_name,
		LastName:     last_name,
		PasswordHash: string(hashedPassword),
		RoleID:       1,
		IsActive:     true,
	})
	if err != nil {
		s.logger.Error("error creating admin user: ", err)
		return db.Admin{}, err
	}

	log.Println("loging activity for new admin user:", username)
	_, err = txQueries.LogActivity(ctx, db.LogActivityParams{
		UserID:     user.ID,
		Action:     "register_admin",
		Details:    utils.WriteActivityDetails(username, email, "Registered new admin user", user.CreatedAt.Time),
		EntityType: "admin",
		EntityID:   user.ID,
		IpAddress:  sql.NullString{Valid: true, String: utils.GetClientIP(ctx)},
		UserAgent:  sql.NullString{Valid: true, String: ""},
	})
	if err != nil {
		s.logger.Error("error logging activity: ", err)
		return db.Admin{}, err
	}
	return user, nil
}

// SetEmailVerification sets the verification code and expiry for a user.
func (a *Service) SetEmailVerification(ctx context.Context, userID int32, code string, expiry time.Time) error {
	return a.queries.SetAdminEmailVerification(ctx, db.SetAdminEmailVerificationParams{
		ID:                    userID,
		VerificationCode:      sql.NullString{Valid: code != "", String: code},
		VerificationExpiresAt: sql.NullTime{Valid: true, Time: expiry},
	})
}

// VerifyEmailCode checks the code and marks the email as verified if valid and not expired.
func (a *Service) VerifyEmailCode(ctx context.Context, email, code string) (bool, error) {
	admin, err := a.queries.GetAdminByEmail(ctx, email)
	if err != nil {
		return false, err
	}
	if admin.EmailVerified {
		return false, nil // Already verified
	}
	if admin.VerificationCode.String != code {
		return false, nil // Invalid code
	}
	if !admin.VerificationExpiresAt.Valid || admin.VerificationExpiresAt.Time.Before(time.Now()) {
		return false, nil // Expired
	}
	// Mark as verified and clear code
	err = a.queries.MarkAdminEmailVerified(ctx, db.MarkAdminEmailVerifiedParams{
		ID:            admin.ID,
		EmailVerified: true,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) recordFailedAttempt(ctx context.Context, username, ipAddress, reason string) {
	// Increment user attempt counter
	userAttemptsKey := fmt.Sprintf("login_attempts:user:%s", username)
	s.rateLimiter.Increment(ctx, userAttemptsKey, s.loginRateWindow)

	// Increment IP attempt counter
	ipAttemptsKey := fmt.Sprintf("login_attempts:ip:%s", ipAddress)
	s.rateLimiter.Increment(ctx, ipAttemptsKey, s.loginRateWindow)

	// Check if user should be blocked
	exceeded, _, _, err := s.rateLimiter.Check(
		ctx, userAttemptsKey, s.loginRateLimit, s.loginRateWindow,
	)
	if err == nil && exceeded {
		userBlockKey := fmt.Sprintf("rate_limit:ip:%s", username)
		s.rateLimiter.BlockKey(ctx, userBlockKey, s.loginBlockDuration)
	}

	// Check if IP should be blocked
	exceeded, _, _, err = s.rateLimiter.Check(
		ctx, ipAttemptsKey, s.loginRateLimit*2, s.loginRateWindow,
	)
	if err == nil && exceeded {
		ipBlockKey := fmt.Sprintf("rate_limit:ip:%s", ipAddress)
		s.rateLimiter.BlockKey(ctx, ipBlockKey, s.loginBlockDuration*2) // Longer block for IPs
	}

	// Log the failed attempt (you might want to store this in DB too)
	fmt.Printf("Failed login attempt - User: %s, IP: %s, Reason: %s\n", username, ipAddress, reason)
}

func (s *Service) resetLoginAttempts(ctx context.Context, username string) {
	// Remove user attempt counter
	userAttemptsKey := fmt.Sprintf("login_attempts:user:%s", username)
	s.redis.Delete(ctx, userAttemptsKey)
}

func (s *Service) logLoginAttempt(ctx context.Context, usernameOrEmail, ipAddress, userAgent string, success bool, reason string) {
	// Store in database
	if success {
		s.queries.LogLoginAttempt(ctx, db.LogLoginAttemptParams{
			UsernameOrEmail: usernameOrEmail,
			IpAddress:       sql.NullString{String: ipAddress, Valid: ipAddress != ""},
			UserAgent:       sql.NullString{String: userAgent, Valid: userAgent != ""},
			Success:         true,
		})
	}

	// Also cache recent attempts for quick checking
	attemptKey := fmt.Sprintf("recent_login:%s", usernameOrEmail)
	attemptData := fmt.Sprintf("%s|%t|%s", time.Now().Format(time.RFC3339), success, reason)
	s.rClient.LPush(ctx, attemptKey, attemptData)
	s.rClient.LTrim(ctx, attemptKey, 0, 9) // Keep only last 10 attempts
	s.rClient.Expire(ctx, attemptKey, 24*time.Hour)
}

func (s *Service) Login(ctx context.Context, emailOrUsername, password, ipAddress, userAgent string) (string, string, error) {
	// Check rate limits
	if err := s.checkRateLimits(ctx, emailOrUsername, ipAddress); err != nil {
		return "", "", err
	}

	// Increment IP request counter
	ipRequestKey := fmt.Sprintf("ip_requests:%s", ipAddress)
	s.rateLimiter.Increment(ctx, ipRequestKey, time.Minute)

	// Helper to handle successful login
	handleSuccess := func(userID int32, username, email, roleName, passwordHash string, isAdmin bool) (string, string, error) {
		if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
			s.recordFailedAttempt(ctx, emailOrUsername, ipAddress, "ErrInvalidCredentials")
			remaining, _ := s.rateLimiter.GetRemainingAttempts(ctx, fmt.Sprintf("login_attempts:user:%s", emailOrUsername), s.loginRateLimit, s.loginRateWindow)
			return "", "", fmt.Errorf("invalid credentials. %d attempts remaining", remaining)
		}
		s.resetLoginAttempts(ctx, emailOrUsername)
		var permissions []string
		var err error
		if isAdmin {
			permissions, err = s.queries.GetAdminPermissions(ctx, userID)
		} else {
			permissions, err = s.queries.GetUserPermissions(ctx, userID)
		}
		if err != nil {
			return "", "", err
		}
		token, err := jwt.GenerateToken(
			int(userID), username, email, roleName,
			s.jwtSecret, permissions, jwt.AccessToken, s.accessExpiry,
		)
		if err != nil {
			return "", "", err
		}
		refreshToken, err := generateRefreshToken()
		if err != nil {
			return "", "", err
		}
		expiresAt := time.Now().Add(s.refreshExpiry)
		_, err = s.queries.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
			UserID: userID, Token: refreshToken, ExpiresAt: expiresAt,
		})
		if err != nil {
			return "", "", err
		}
		s.logLoginAttempt(ctx, emailOrUsername, ipAddress, userAgent, true, "success")
		return token, refreshToken, nil
	}

	// Try user by email
	if userByEmail, err := s.queries.GetUserByEmail(ctx, sql.NullString{String: emailOrUsername, Valid: true}); err == nil {
		if !userByEmail.IsActive.Bool {
			s.recordFailedAttempt(ctx, emailOrUsername, ipAddress, "ErrUserInactive")
			return "", "", ErrUserInactive
		}
		return handleSuccess(userByEmail.ID, userByEmail.Username, userByEmail.Email.String, userByEmail.RoleName, userByEmail.PasswordHash, false)
	}

	// Try user by username
	if userByUsername, err := s.queries.GetUserByUsername(ctx, emailOrUsername); err == nil {
		if !userByUsername.IsActive.Bool {
			s.recordFailedAttempt(ctx, emailOrUsername, ipAddress, "ErrUserInactive")
			return "", "", ErrUserInactive
		}
		return handleSuccess(userByUsername.ID, userByUsername.Username, userByUsername.Email.String, userByUsername.RoleName, userByUsername.PasswordHash, false)
	}

	// Try admin by email
	if adminByEmail, err := s.queries.GetAdminByEmail(ctx, emailOrUsername); err == nil {
		if !adminByEmail.IsActive {
			s.recordFailedAttempt(ctx, emailOrUsername, ipAddress, "ErrUserInactive")
			return "", "", ErrUserInactive
		}
		return handleSuccess(adminByEmail.ID, adminByEmail.Username, adminByEmail.Email, adminByEmail.RoleName, adminByEmail.PasswordHash, true)
	}

	// Try admin by username
	if adminByUsername, err := s.queries.GetAdminByUsername(ctx, emailOrUsername); err == nil {
		if !adminByUsername.IsActive {
			s.recordFailedAttempt(ctx, emailOrUsername, ipAddress, "ErrUserInactive")
			return "", "", ErrUserInactive
		}
		return handleSuccess(adminByUsername.ID, adminByUsername.Username, adminByUsername.Email, adminByUsername.RoleName, adminByUsername.PasswordHash, true)
	}

	s.recordFailedAttempt(ctx, emailOrUsername, ipAddress, "ErrUserNotFound")
	return "", "", ErrInvalidCredentials
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	// Validate refresh token from database
	tokenRecord, err := s.queries.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", "", ErrInvalidCredentials
	}

	// Check if token is expired or revoked
	if tokenRecord.ExpiresAt.Before(time.Now()) {
		return "", "", ErrInvalidCredentials
	}

	// Get user information
	user, err := s.queries.GetUserByID(ctx, tokenRecord.UserID)
	if err != nil {
		return "", "", err
	}

	if !user.IsActive.Bool {
		return "", "", ErrUserInactive
	}

	permissions, err := s.queries.GetUserPermissions(ctx, user.ID)
	if err != nil {
		return "", "", err
	}

	// Generate new access token
	newAccessToken, err := jwt.GenerateToken(
		int(user.ID),
		user.Username,
		user.Email.String,
		user.RoleName,
		s.jwtSecret,
		permissions,
		jwt.AccessToken,
		s.accessExpiry,
	)
	if err != nil {
		return "", "", err
	}

	// Generate new refresh token (rotate refresh token)
	newRefreshToken, err := generateRefreshToken()
	if err != nil {
		return "", "", err
	}

	expiresAt := time.Now().Add(s.refreshExpiry)
	_, err = s.queries.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    int32(user.ID),
		Token:     newRefreshToken,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return "", "", err
	}

	// Revoke the old refresh token
	if err := s.queries.RevokeRefreshToken(ctx, refreshToken); err != nil {
		// Log error but continue
		fmt.Printf("Error revoking refresh token: %v\n", err)
	}

	return newAccessToken, newRefreshToken, nil
}

func (s *Service) cleanExpiredTokens(ctx context.Context) {
	// Run cleanup every hour
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.queries.CleanExpiredRefreshTokens(ctx); err != nil {
				fmt.Printf("Error cleaning expired tokens: %v\n", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) RevokeAllUserSessions(ctx context.Context, userID int) error {
	// Revoke all refresh tokens for user
	if err := s.queries.RevokeAllUserRefreshTokens(ctx, int32(userID)); err != nil {
		return err
	}

	// Add user's tokens to blacklist (you might want to track user's active tokens)
	cacheKey := fmt.Sprintf("user:%d:active_tokens", userID)
	return s.redis.Delete(ctx, cacheKey)
}

func (s *Service) Logout(ctx context.Context, token string, expiry time.Duration) error {
	claims, err := jwt.ParseToken(token, s.jwtSecret)
	if err != nil {
		return err
	}

	remainingTime := time.Until(claims.ExpiresAt.Time)
	if remainingTime > 0 {
		// add to blacklist until token expires
		err := s.redis.Set(ctx, fmt.Sprintf("jwt:blacklist:%s", token), "1", remainingTime)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	exists, err := s.redis.Exists(ctx, fmt.Sprintf("jwt:blacklist:%s", token))
	return exists, err
}

func (s *Service) HasPermission(claims *jwt.Claims, requiredPermission string) bool {
	return slices.Contains(claims.Permissions, requiredPermission)
}

// Admin user management functions
func (s *Service) CreateUser(ctx context.Context, params db.CreateUserParams) (db.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(params.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		return db.User{}, err
	}

	params.PasswordHash = string(hashedPassword)
	return s.queries.CreateUser(ctx, params)
}

func (s *Service) UpdateUser(ctx context.Context, params db.UpdateUserParams) (db.User, error) {
	updatedUser, err := s.queries.UpdateUser(ctx, params)
	if err != nil {
		return db.User{}, err
	}
	// Invalidate cache
	cacheKey := fmt.Sprintf("user:%d", params.ID)
	s.redis.Delete(ctx, cacheKey)
	return updatedUser, nil
}

func (s *Service) DeleteUser(ctx context.Context, id int32) error {
	return s.queries.DeleteUser(ctx, id)
}

func (s *Service) ResetPassword(ctx context.Context, params db.UpdateUserPasswordParams) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(params.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	params.PasswordHash = string(hashedPassword)
	return s.queries.UpdateUserPassword(ctx, params)
}

// Role management functions
func (s *Service) CreateRole(ctx context.Context, params db.CreateRoleParams) (db.Role, error) {
	return s.queries.CreateRole(ctx, params)
}

func (s *Service) UpdateRole(ctx context.Context, params db.UpdateRoleParams) (db.Role, error) {
	return s.queries.UpdateRole(ctx, params)
}

func (s *Service) DeleteRole(ctx context.Context, id int32) error {
	return s.queries.DeleteRole(ctx, id)
}

func (s *Service) AddPermissionToRole(ctx context.Context, params db.AddPermissionToRoleParams) error {
	return s.queries.AddPermissionToRole(ctx, params)
}

func (s *Service) RemovePermissionFromRole(ctx context.Context, params db.RemovePermissionFromRoleParams) error {
	return s.queries.RemovePermissionFromRole(ctx, params)
}

func (s *Service) GetUserByID(ctx context.Context, id int32) (db.GetUserByIDRow, error) {
	cacheKey := fmt.Sprintf("user:%d", id)

	// Try cache first
	if cachedUser, err := s.redis.Get(ctx, cacheKey); err == nil {
		var user db.GetUserByIDRow
		err = json.Unmarshal([]byte(cachedUser), &user)
		if err != nil {
			return db.GetUserByIDRow{}, err
		}
		return user, nil
	}

	// Fetch from database if no cache
	user, err := s.queries.GetUserByID(ctx, id)
	if err != nil {
		return db.GetUserByIDRow{}, err
	}

	// Cache for 30 minutes
	jsonUser, _ := json.Marshal(user)
	s.redis.Set(ctx, cacheKey, jsonUser, 30*time.Minute)
	return user, nil
}

func (s *Service) GetUserByEmail(ctx context.Context, email string) (db.GetUserByEmailRow, error) {
	cacheKey := fmt.Sprintf("user:email:%s", email)

	// Try cache first
	if cachedUser, err := s.redis.Get(ctx, cacheKey); err == nil {
		var user db.GetUserByEmailRow
		err = json.Unmarshal([]byte(cachedUser), &user)
		if err != nil {
			return db.GetUserByEmailRow{}, err
		}
		return user, nil
	}

	// Fetch from database if no cache
	user, err := s.queries.GetUserByEmail(ctx, sql.NullString{String: email, Valid: true})
	if err != nil {
		return db.GetUserByEmailRow{}, err
	}

	// Cache for 30 minutes
	jsonUser, _ := json.Marshal(user)
	s.redis.Set(ctx, cacheKey, jsonUser, 30*time.Minute)
	return user, nil
}

func (s *Service) GetUserByUsername(ctx context.Context, username string) (db.GetUserByUsernameRow, error) {
	cachedKey := fmt.Sprintf("user_by_username:%s", username)
	if cachedUser, err := s.redis.Get(ctx, cachedKey); err == nil {
		var user db.GetUserByUsernameRow
		err = json.Unmarshal([]byte(cachedUser), &user)
		if err != nil {
			return db.GetUserByUsernameRow{}, err
		}
		return user, nil
	}

	// Fetch from database if no cache
	user, err := s.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return db.GetUserByUsernameRow{}, err
	}

	// Cache for 30 minutes
	jsonUser, _ := json.Marshal(user)
	s.redis.Set(ctx, cachedKey, jsonUser, 30*time.Minute)
	return user, nil
}

func (s *Service) ListUsers(ctx context.Context) ([]db.ListUsersRow, error) {
	return s.queries.ListUsers(ctx)
}

func (s *Service) ListRoles(ctx context.Context) ([]db.Role, error) {
	return s.queries.ListRoles(ctx)
}

func (s *Service) GetRolePermissions(ctx context.Context, roleID int32) ([]db.Permission, error) {
	return s.queries.GetRolePermissions(ctx, roleID)
}

func (s *Service) LogActivity(ctx context.Context, params db.LogActivityParams) error {
	_, err := s.queries.LogActivity(ctx, params)
	return err
}

// func (s *Service) LogLogin(ctx context.Context, username, email string, ip, userAgent string, success bool, errorReason string) error {
// 	err := s.queries.LogLoginAttempt(ctx, db.LogLoginAttemptParams{
// 		Username:    username,
// 		Email:       sql.NullString{String: email, Valid: email != ""},
// 		IpAddress:   sql.NullString{String: ip, Valid: ip != ""},
// 		UserAgent:   sql.NullString{String: userAgent, Valid: userAgent != ""},
// 		ErrorReason: sql.NullString{String: errorReason, Valid: errorReason != ""},
// 	})
// 	return err
// }

// ForgotPassword: generates a reset code and expiry, stores it for user/admin
func (s *Service) ForgotPassword(ctx context.Context, email string) (string, error) {
	admin, err := s.queries.GetAdminByEmail(ctx, email)
	if err == nil {
		code := utils.GenerateOTP()
		fmt.Println("Generated code:", code)
		expiry := time.Now().Add(15 * time.Minute)
		err := s.queries.SetAdminResetCode(ctx, db.SetAdminResetCodeParams{
			ID:                 admin.ID,
			ResetCode:          sql.NullString{String: code, Valid: true},
			ResetCodeExpiresAt: sql.NullTime{Time: expiry, Valid: true},
		})
		if err != nil {
			return "", err
		}
		return code, nil
	}

	return "", errors.New("email not found")
}

// ResetPassword: verifies code and sets new password for user/admin
func (s *Service) ResetAdminPassword(ctx context.Context, email, code, newPassword string) error {
	admin, err := s.queries.GetAdminByEmail(ctx, email)
	if err == nil {
		if !admin.ResetCode.Valid || admin.ResetCode.String != code || !admin.ResetCodeExpiresAt.Valid || admin.ResetCodeExpiresAt.Time.Before(time.Now()) {
			return errors.New("invalid or expired code")
		}
		hashed, _ := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		err := s.queries.UpdateAdminPassword(ctx, db.UpdateAdminPasswordParams{
			ID:           admin.ID,
			PasswordHash: string(hashed),
		})
		if err != nil {
			return err
		}
		// Clear reset code
		_ = s.queries.ClearAdminResetCode(ctx, admin.ID)
		return nil
	}

	return errors.New("email not found")
}
