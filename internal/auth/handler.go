package auth

import (
	"errors"
	"fmt"
	"herp/internal/config"
	"herp/internal/utils"
	"herp/pkg/jwt"
	"herp/pkg/monitoring/logging"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service ServiceInterface
	config  *config.Config
	logger  *logging.Logger
	env     string // remove
}

func NewHandler(service ServiceInterface, c *config.Config, l *logging.Logger, e string) *Handler {
	return &Handler{service, c, l, e}
}

// LoginRequest represents the login request payload
// @Description Login request payload
type LoginRequest struct {
	Username string `json:"username" example:"admin"`                          // Username for authentication (optional if email provided)
	Email    string `json:"email" example:"admin@hotel.com"`                   // Email for authentication (optional if username provided)
	Password string `json:"password" binding:"required" example:"password123"` // Password for authentication
}

// LoginResponse represents the login response payload
// @Description Login response payload
type LoginResponse struct {
	AccessToken  string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`    // JWT authentication token
	RefreshToken string `json:"refresh_token" example:"dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4..."` // JWT refresh token
	ExpiredAt    int64  `json:"expired_at" example:"1700000000"`                            // Token expiration timestamp in seconds
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required" example:"dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4..."` // JWT refresh token
}

type RefreshResponse struct {
	AccessToken  string `json:"accessToken" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."` // JWT authentication token
	RefreshToken string `json:"refreshToken" example:"dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4..."`     // JWT refresh token
	ExpiresIn    int    `json:"expiresIn" example:"3600"`                                      // Token expiration in seconds
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email" example:"user@example.com"` // User email address
}

type ResetAdminPasswordRequest struct {
	Email       string `json:"email" binding:"required,email" example:"admin@example.com"` // Admin email address
	Code        string `json:"code" binding:"required" example:"1234567"`                  // Password reset code
	NewPassword string `json:"new_password" binding:"required,min=8" example:"NewPassword123"`
}

// ErrorResponse represents an error response
// @Description Error response payload
type ErrorrResponse struct {
	Error string `json:"error" example:"Invalid credentials"` // Error message
}

type UnauthorizedResponse struct {
	Error string `json:"error" example:"Unauthorized"` // Error message
}

type BadRequestResponse struct {
	Error string `json:"error" example:"Bad request"` // Error message
}

type InternalServerErrorResponse struct {
	Error string `json:"error" example:"Internal server error"` // Error message
}

type RegisterResponse struct {
	ID              int32      `json:"id" example:"1"`
	Username        string     `json:"username" example:"admin"`
	Email           string     `json:"email" example:"admin@hotel.com"`
	FirstName       string     `json:"first_name" example:"Admin"`
	LastName        string     `json:"last_name" example:"Admin"`
	CreatedAt       *time.Time `json:"created_at,omitempty" example:"2021-01-01T00:00:00Z"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty" example:"2021-01-01T00:00:00Z"`
	IsActive        bool       `json:"is_active" example:"true"`
	RoleID          int32      `json:"role_id" example:"1"`
	IsEmailVerified bool       `json:"is_email_verified" example:"true"`
}

// Login godoc
// @Summary User login
// @Description Authenticate user with email or username and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param body body LoginRequest true "Login credentials (email or username)"
// @Success 200 {object} LoginResponse "Login successful"
// @Failure 400 {object} BadRequestResponse "Bad request"
// @Failure 401 {object} UnauthorizedResponse "Unauthorized"
// @Failure 500 {object} InternalServerErrorResponse "Internal server error"
// @Router /api/v1/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Println("Error binding JSON:", err)
		utils.ErrorResponse(c, 400, err.Error())
		return
	}

	// Determine which identifier to use (email or username)
	identifier := req.Email
	if identifier == "" {
		identifier = req.Username
	}

	// Validate that at least one identifier is provided
	if identifier == "" {
		utils.ErrorResponse(c, 400, "Either email or username must be provided")
		return
	}

	ip := utils.GetClientIP(c)

	token, refreshToken, err := h.service.Login(c, identifier, req.Password, ip, c.Request.UserAgent())
	if err != nil {
		// log.Printf("login error: %v", err)
		h.logger.Printf("login error: %v", err)
		status := http.StatusUnauthorized
		errorMsg := err.Error()
		if !errors.Is(err, ErrInvalidCredentials) && !errors.Is(err, ErrUserInactive) {
			status = http.StatusBadRequest
		} else if strings.Contains(errorMsg, "temporarily blocked") ||
			strings.Contains(errorMsg, "Account temporarily locked") ||
			strings.Contains(errorMsg, "Too many requests") {
			status = http.StatusTooManyRequests
		}
		utils.ErrorResponse(c, status, errorMsg)
		return
	}

	// Parse token to get expiry
	claims, _ := jwt.ParseToken(token, h.config.JWTSecret)
	expiry := time.Time{}
	if claims != nil {
		expiry = claims.ExpiresAt.Time
	}

	utils.SuccessResponse(c, 200, "login successful", LoginResponse{
		AccessToken:  token,
		RefreshToken: refreshToken,
		ExpiredAt:    expiry.Unix(),
	})

}

// Refresh godoc
// @Summary Refresh JWT token
// @Description Refresh JWT token using a valid refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param body body RefreshRequest true "Refresh token request"
// @Success 200 {object} RefreshResponse "Token refreshed successfully"
// @Failure 400 {object} BadRequestResponse "Bad request"
// @Failure 401 {object} UnauthorizedResponse "Unauthorized"
// @Failure 500 {object} InternalServerErrorResponse "Internal server error"
// @Router /api/v1/auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		utils.ErrorResponse(c, 401, "Missing or invalid refresh token")
		return
	}

	accessToken, refreshToken, err := h.service.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		status := http.StatusUnauthorized
		if !errors.Is(err, ErrInvalidCredentials) && !errors.Is(err, ErrUserInactive) {
			status = http.StatusInternalServerError
		}
		utils.ErrorResponse(c, status, err.Error())
		return
	}

	claims, _ := jwt.ParseToken(accessToken, h.config.JWTSecret)
	expiry := time.Time{}
	if claims != nil {
		expiry = claims.ExpiresAt.Time
	}

	utils.SuccessResponse(c, 200, "message string", RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(expiry.Unix()),
	})
}

// Logout godoc
// @Summary User logout
// @Description Logout user and invalidate JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 "Logout successful"
// @Failure 400 {object} BadRequestResponse "Bad request"
// @Failure 401 {object} UnauthorizedResponse "Unauthorized"
// @Failure 500 {object} InternalServerErrorResponse "Internal server error"
// @Router /api/v1/auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	authHeader := c.GetHeader(AuthorizationHeader)
	if authHeader == "" {
		utils.ErrorResponse(c, 401, "unauthorized")
		return
	}
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, BearerPrefix))
	if token == "" || token == authHeader { // No Bearer prefix found
		utils.ErrorResponse(c, 401, "invalid authorization header format")
		return
	}

	claims, exists := c.Get("claims")
	if !exists || claims == nil {
		utils.ErrorResponse(c, 401, "unauthorized")
		return
	}

	jwtClaims, ok := claims.(*jwt.Claims)
	if !ok {
		utils.ErrorResponse(c, 401, "invalid claims type")
		return
	}

	expiry := time.Until(jwtClaims.ExpiresAt.Time)
	if err := h.service.Logout(c.Request.Context(), token, expiry); err != nil {
		utils.ErrorResponse(c, 500, err.Error())
		return
	}
	utils.SuccessResponse(c, 200, "Logged out successfully", nil)
}

// RegisterAdminRequest represents the login request payload
// @Description Register admin request payload
type RegisterAdminRequest struct {
	FirstName string `json:"first_name" binding:"required,min=2"`
	LastName  string `json:"last_name" binding:"required,min=2"`
	Username  string `json:"username" binding:"required,min=3"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
}

// Admin Register godoc
// @Summary Admin Register
// @Description Create admin with email, username, password and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param body body RegisterAdminRequest true "Register credentials (email, username and password)"
// @Success 200 {object} RegisterResponse "Registration successful"
// @Failure 400 {object} BadRequestResponse "Bad request"
// @Failure 401 {object} UnauthorizedResponse "Unauthorized"
// @Failure 500 {object} InternalServerErrorResponse "Internal server error"
// @Failure 500 {string} string "Unable to send email at this time, request a new verification code for example@email.com"
// @Router /api/v1/auth/register [post]
func (h *Handler) RegisterAdmin(c *gin.Context) {
	var req RegisterAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, err.Error())
		return
	}
	// Generate verification code and expiry
	code := utils.GenerateOTP()
	expiry := time.Now().Add(10 * time.Minute)

	admin, err := h.service.RegisterAdmin(c, req.Username, req.Email, req.Password, req.FirstName, req.LastName)
	if err != nil {
		log.Printf("error registering admin: %v", err)
		utils.ErrorResponse(c, 500, err.Error())
		return
	}

	err = h.service.SetEmailVerification(c.Request.Context(), admin.ID, code, expiry)
	if err != nil {
		log.Printf("error saving verification code: %v", err)
		utils.ErrorResponse(c, 500, err.Error())
		return
	}

	// Send verification email
	emailBody, _ := utils.RenderEmailTemplate("templates/auth/verify_email.html", map[string]any{
		"Username": admin.Username,
		"Code":     code,
	})
	plunk := utils.Plunk{HttpClient: http.DefaultClient, Config: h.config}
	err = plunk.SendEmail(admin.Email, "Verify your Herp account", emailBody)
	if err != nil {
		log.Printf("error sending verification email: %v", err)
		utils.ErrorResponse(c, 500, fmt.Sprintf("Unable to send email at this time, request a new verification code for %s", admin.Email))
		return
	}
	utils.SuccessResponse(c, 200, "Registration successful", RegisterResponse{
		ID:              admin.ID,
		Username:        admin.Username,
		Email:           admin.Email,
		FirstName:       admin.FirstName,
		LastName:        admin.LastName,
		CreatedAt:       &admin.CreatedAt.Time,
		UpdatedAt:       &admin.UpdatedAt.Time,
		IsActive:        admin.IsActive,
		RoleID:          admin.RoleID,
		IsEmailVerified: admin.EmailVerified,
	})
}

// VerifyEmailRequest represents the login request payload
// @Description verify email request payload
type VerifyEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required"`
}

// Verify Email godoc
// @Summary Verify Admin Email
// @Description Verify admin email with email and code
// @Tags auth
// @Accept json
// @Produce json
// @Param body body VerifyEmailRequest true "Verify Email Request"
// @Success 200 "Email verified successfully"
// @Failure 400 {object} BadRequestResponse "Bad request"
// @Failure 401 {object} UnauthorizedResponse "Unauthorized"
// @Failure 500 {object} InternalServerErrorResponse "Internal server error"
// @Router /api/v1/auth/verify-email [post]
func (h *Handler) VerifyEmail(c *gin.Context) {
	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, err.Error())
		return
	}

	ok, err := h.service.VerifyEmailCode(c.Request.Context(), req.Email, req.Code)
	if err != nil {
		utils.ErrorResponse(c, 500, err.Error())
		return
	}
	if !ok {
		utils.ErrorResponse(c, 400, "Invalid or expired code")
		return
	}
	utils.SuccessResponse(c, 200, "Email verified successfully", nil)
}

// Forgot Password godoc
// @Summary Forgot Password
// @Description Initiate password reset by sending a reset code to the user's email
// @Tags auth
// @Accept json
// @Produce json
// @Param body body ForgotPasswordRequest true "Forgot Password Request"
// @Success 200 "Reset code sent to email"
// @Failure 400 {object} BadRequestResponse "Bad request"
// @Failure 404 {object} UnauthorizedResponse "User not found"
// @Failure 500 {object} InternalServerErrorResponse "Internal server error"
// @Router /api/v1/auth/forgot-password [post]
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, err.Error())
		return
	}
	code, err := h.service.ForgotPassword(c.Request.Context(), req.Email)
	if err != nil {
		utils.ErrorResponse(c, 404, err.Error())
		return
	}
	// Send verification email
	emailBody, _ := utils.RenderEmailTemplate("templates/auth/forgot_password.html", map[string]any{
		"Code": code,
	})
	plunk := utils.Plunk{HttpClient: http.DefaultClient, Config: h.config}
	err = plunk.SendEmail(req.Email, "Reset your password", emailBody)
	if err != nil {
		log.Printf("error sending verification email: %v", err)
		utils.ErrorResponse(c, 500, err.Error())
		return
	}
	utils.SuccessResponse(c, 200, "Reset code sent to email", nil)
}

// Reset Password godoc
// @Summary Reset Password
// @Description Reset password using email, reset code, and new password
// @Tags auth
// @Accept json
// @Produce json
// @Param body body ResetAdminPasswordRequest true "Reset Password Request"
// @Success 200 "Password reset successful"
// @Failure 400 {object} BadRequestResponse "Bad request or invalid code"
// @Failure 404 {object} UnauthorizedResponse "User not found"
// @Failure 500 {object} InternalServerErrorResponse "Internal server error"
// @Router /api/v1/auth/reset-password [post]
func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetAdminPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, err.Error())
		return
	}
	err := h.service.ResetAdminPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword)
	if err != nil {
		utils.ErrorResponse(c, 400, err.Error())
		return
	}
	utils.SuccessResponse(c, 200, "Password reset successful", nil)
}
