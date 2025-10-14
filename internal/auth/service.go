package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	db "nucleus/db/sqlc"
	"nucleus/internal/utils"
	"nucleus/pkg/jwt"
	"nucleus/pkg/monitoring/logging"
	"nucleus/pkg/ratelimit"
	"nucleus/pkg/redis"
	"slices"
	"time"

	r "github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive       = errors.New("user is inactive")
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailNotVerified   = errors.New("email not verified")
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
		// userService: u,
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

// getUserPermissions retrieves all permissions for a user including role hierarchy
func (s *Service) getUserPermissions(ctx context.Context, userID, tenantID int32) ([]string, error) {
	// Get user permissions with hierarchy (includes inherited permissions from parent roles)
	permissions, err := s.queries.GetUserPermissionsWithHierarchy(ctx, db.GetUserPermissionsWithHierarchyParams{
		UserID:   userID,
		TenantID: tenantID,
	})
	if err != nil {
		// Check if it's a "no rows" error, which is acceptable for new users
		if err == sql.ErrNoRows {
			s.logger.Info("No permissions found for user", "userID", userID, "tenantID", tenantID)
			return []string{}, nil
		}
		s.logger.Error("Failed to get user permissions", "error", err, "userID", userID, "tenantID", tenantID)
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	// Extract permission codes from the slice of rows
	var permissionCodes []string
	for _, perm := range permissions {
		if perm.Code != "" { // Make sure we don't add empty permission codes
			permissionCodes = append(permissionCodes, perm.Code)
		}
	}

	// Remove duplicates (in case a user has the same permission through multiple roles)
	permissionCodes = removeDuplicates(permissionCodes)

	s.logger.Debug("Retrieved user permissions", "userID", userID, "tenantID", tenantID, "permissions", permissionCodes)
	return permissionCodes, nil
}

// Helper function to remove duplicate permission codes
func removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}

// updateUserLastLogin updates the user's last login timestamp
func (s *Service) updateUserLastLogin(ctx context.Context, userID int32) error {
	return s.queries.UpdateUserLastLogin(ctx, userID)
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

func (s *Service) RegisterTenant(ctx context.Context, email, password, organization string) (*db.Tenant, error) {
	log.Println("Registering new admin user:", organization, email)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		s.logger.Error("error hashing password: ", err)
		return nil, err
	}

	q, ok := s.queries.(*db.Queries)
	if !ok {
		return nil, fmt.Errorf("invalid queries implementation")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.logger.Error("error starting transaction: ", err)
		return nil, err
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

	log.Println("Admin user details:", email, organization)
	user, err := txQueries.CreateTenant(ctx, db.CreateTenantParams{
		Organization: organization,
		Email:        email,
		PasswordHash: string(hashedPassword),
	})
	if err != nil {
		s.logger.Error("error creating admin user: ", err)
		return nil, err
	}

	return &user, nil
}

// SetEmailVerification sets the verification code and expiry for a user.
func (a *Service) SetEmailVerification(ctx context.Context, userID int32, code string, expiry time.Time) error {
	return a.queries.SetTenantEmailVerification(ctx, db.SetTenantEmailVerificationParams{
		ID:                    userID,
		VerificationCode:      sql.NullString{Valid: code != "", String: code},
		VerificationExpiresAt: sql.NullTime{Valid: true, Time: expiry},
	})
}

// VerifyEmailCode checks the code and marks the email as verified if valid and not expired.
func (a *Service) VerifyEmailCode(ctx context.Context, email, code string) (bool, error) {
	admin, err := a.queries.GetTenantByEmail(ctx, email)
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
	err = a.queries.MarkTenantEmailVerified(ctx, db.MarkTenantEmailVerifiedParams{
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

func (s *Service) Login(ctx context.Context, email, password, ipAddress, userAgent string, tenantID int32) (string, string, error) {
	// Check rate limits
	if err := s.checkRateLimits(ctx, email, ipAddress); err != nil {
		return "", "", err
	}

	// Increment IP request counter
	ipRequestKey := fmt.Sprintf("ip_requests:%s", ipAddress)
	s.rateLimiter.Increment(ctx, ipRequestKey, time.Minute)

	// Helper to handle successful login
	handleSuccess := func(userID, tenantID int32, passwordHash, email, organization, userType string) (string, string, error) {
		if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
			s.recordFailedAttempt(ctx, email, ipAddress, "ErrInvalidCredentials")
			remaining, _ := s.rateLimiter.GetRemainingAttempts(ctx, fmt.Sprintf("login_attempts:user:%s", email), s.loginRateLimit, s.loginRateWindow)
			return "", "", fmt.Errorf("invalid credentials. %d attempts remaining", remaining)
		}

		s.resetLoginAttempts(ctx, email)

		var permissions []string
		var err error

		if userType == "user" && userID > 0 {
			// Get permissions for regular users from RBAC system
			permissions, err = s.getUserPermissions(ctx, userID, tenantID)
			if err != nil {
				s.logger.Error("failed to get user permission during login", "error", err)
				// Continue with empty permissions rather than failing login
				permissions = []string{}
			}

			// update last login timestamp
			if err := s.updateUserLastLogin(ctx, userID); err != nil {
				s.logger.Error("Failed to update last login", "error", err)
			}
		} else if userType == "tenant" {
			permissions = []string{"tenant:manage"}
		}

		token, err := jwt.GenerateToken(
			int(userID), int(tenantID), email, organization, "", userType, s.jwtSecret, permissions, jwt.AccessToken, s.accessExpiry)
		if err != nil {
			return "", "", fmt.Errorf("failed to generate token: %w", err)
		}

		refreshToken, err := generateRefreshToken()
		if err != nil {
			return "", "", err
		}
		// expiresAt := time.Now().Add(s.refreshExpiry)
		// _, err = s.queries.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		// 	UserID: userID, Token: refreshToken, ExpiresAt: expiresAt,
		// })

		s.logLoginAttempt(ctx, email, ipAddress, userAgent, true, "success")
		return token, refreshToken, nil
	}

	if tenant, err := s.queries.GetTenantByEmail(ctx, email); err == nil {
		if !tenant.IsActive {
			s.recordFailedAttempt(ctx, email, ipAddress, "ErrUserInactive")
			return "", "", ErrUserInactive
		}

		if !tenant.EmailVerified {
			s.recordFailedAttempt(ctx, email, ipAddress, "ErrEmailNotVerified")
			return "", "", ErrEmailNotVerified
		}

		return handleSuccess(0, tenant.ID, tenant.PasswordHash, tenant.Email, tenant.Organization, "tenant")
	}

	_, err := s.queries.GetTenantByID(ctx, tenantID)
	if err != nil {
		return "", "", err
	}

	if user, err := s.queries.GetUserByEmail(ctx, db.GetUserByEmailParams{
		Email:     email,
		TenantsID: tenantID,
	}); err == nil {
		if !user.IsActive {
			s.recordFailedAttempt(ctx, email, ipAddress, "ErrUserInactive")
			return "", "", ErrUserInactive
		}

		return handleSuccess(user.ID, user.TenantsID, user.PasswordHash, user.Email, "", "user")
	}

	s.recordFailedAttempt(ctx, email, ipAddress, "ErrUserNotFound")
	return "", "", ErrInvalidCredentials
}

// func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
// 	// Validate refresh token from database
// 	tokenRecord, err := s.queries.GetRefreshToken(ctx, refreshToken)
// 	if err != nil {
// 		return "", "", ErrInvalidCredentials
// 	}

// 	// Check if token is expired or revoked
// 	if tokenRecord.ExpiresAt.Before(time.Now()) {
// 		return "", "", ErrInvalidCredentials
// 	}

// 	// Get user information
// 	user, err := s.queries.GetDeveloperByID(ctx, tokenRecord.UserID)
// 	if err != nil {
// 		return "", "", err
// 	}

// 	if !user.IsActive.Bool {
// 		return "", "", ErrUserInactive
// 	}

// 	// Generate new access token
// 	newAccessToken, err := jwt.GenerateDevToken(
// 		int(user.ID),
// 		user.Email,
// 		user.Organization,
// 		s.jwtSecret,
// 		jwt.AccessToken,
// 		s.accessExpiry,
// 	)
// 	if err != nil {
// 		return "", "", err
// 	}

// 	// Generate new refresh token (rotate refresh token)
// 	newRefreshToken, err := generateRefreshToken()
// 	if err != nil {
// 		return "", "", err
// 	}

// 	expiresAt := time.Now().Add(s.refreshExpiry)
// 	_, err = s.queries.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
// 		UserID:    int32(user.ID),
// 		Token:     newRefreshToken,
// 		ExpiresAt: expiresAt,
// 	})
// 	if err != nil {
// 		return "", "", err
// 	}

// 	// Revoke the old refresh token
// 	if err := s.queries.RevokeRefreshToken(ctx, refreshToken); err != nil {
// 		// Log error but continue
// 		fmt.Printf("Error revoking refresh token: %v\n", err)
// 	}

// 	return newAccessToken, newRefreshToken, nil
// }

// func (s *Service) cleanExpiredTokens(ctx context.Context) {
// 	// Run cleanup every hour
// 	ticker := time.NewTicker(time.Hour)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-ticker.C:
// 			if err := s.queries.CleanExpiredRefreshTokens(ctx); err != nil {
// 				fmt.Printf("Error cleaning expired tokens: %v\n", err)
// 			}
// 		case <-ctx.Done():
// 			return
// 		}
// 	}
// }

// func (s *Service) RevokeAllUserSessions(ctx context.Context, userID int) error {
// 	// Revoke all refresh tokens for user
// 	if err := s.queries.RevokeAllUserRefreshTokens(ctx, int32(userID)); err != nil {
// 		return err
// 	}

// 	// Add user's tokens to blacklist (you might want to track user's active tokens)
// 	cacheKey := fmt.Sprintf("user:%d:active_tokens", userID)
// 	return s.redis.Delete(ctx, cacheKey)
// }

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

// ForgotPassword: generates a reset code and expiry, stores it for user/admin
func (s *Service) ForgotPassword(ctx context.Context, email string) (string, error) {
	admin, err := s.queries.GetTenantByEmail(ctx, email)
	if err == nil {
		code := utils.GenerateOTP()
		fmt.Println("Generated code:", code)
		expiry := time.Now().Add(15 * time.Minute)
		err := s.queries.SetTenantResetCode(ctx, db.SetTenantResetCodeParams{
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
func (s *Service) ResetDeveloperPassword(ctx context.Context, email, code, newPassword string) error {
	admin, err := s.queries.GetTenantByEmail(ctx, email)
	if err == nil {
		if !admin.ResetCode.Valid || admin.ResetCode.String != code || !admin.ResetCodeExpiresAt.Valid || admin.ResetCodeExpiresAt.Time.Before(time.Now()) {
			return errors.New("invalid or expired code")
		}
		hashed, _ := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		err := s.queries.UpdateTenantPassword(ctx, db.UpdateTenantPasswordParams{
			ID:           admin.ID,
			PasswordHash: string(hashed),
		})
		if err != nil {
			return err
		}
		// Clear reset code
		_ = s.queries.ClearTenantResetCode(ctx, admin.ID)
		return nil
	}

	return errors.New("email not found")
}
