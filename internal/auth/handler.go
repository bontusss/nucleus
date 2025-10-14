package auth

import (
	"fmt"
	"log"
	"net/http"
	"nucleus/internal/config"
	"nucleus/internal/core/api"
	"nucleus/internal/utils"
	"nucleus/pkg/jwt"
	"nucleus/pkg/monitoring/logging"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service ServiceInterface
	config  *config.Config
	logger  *logging.Logger
}

func NewHandler(service ServiceInterface, c *config.Config, l *logging.Logger) *Handler {
	return &Handler{service, c, l}
}

// LoginRequest represents the login request payload
// @Description Login request payload
type LoginRequest struct {
	Email    string `json:"email" binding:"required" example:"admin@hotel.com"` // Email for authentication (optional if username provided)
	Password string `json:"password" binding:"required" example:"password123"`  // Password for authentication
	TenantID int32  `json:"tenant_id" binding:"omitempty" example:"1"`
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
	Email           string     `json:"email" example:"admin@hotel.com"`
	OrgName         string     `json:"organization" example:"org name"`
	CreatedAt       *time.Time `json:"created_at,omitempty" example:"2021-01-01T00:00:00Z"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty" example:"2021-01-01T00:00:00Z"`
	IsActive        bool       `json:"is_active" example:"true"`
	IsEmailVerified bool       `json:"is_email_verified" example:"true"`
}

// Login godoc
// @Summary Tenant login
// @Tags Tenants
// @Accept json
// @Produce json
// @Param body body LoginRequest true "Login credentials (email)"
// @Success 200 {object} LoginResponse "Login successful"
// @Failure 400 {object} BadRequestResponse "Bad request"
// @Failure 401 {object} UnauthorizedResponse "Unauthorized"
// @Failure 500 {object} InternalServerErrorResponse "Internal server error"
// @Router /api/v1/tenant/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Println("Error binding JSON:", err)
		api.ErrorResponse(c, 400, err.Error())
		return
	}
	var token, refreshToken string
	var err error
	// For tenant login (organization admin)
	if req.TenantID == 0 {
		// This is a tenant admin login
		token, refreshToken, err = h.service.Login(c.Request.Context(), req.Email, req.Password,
			c.ClientIP(), c.GetHeader("User-Agent"), 0)
		if err != nil {
			api.ErrorResponse(c, 401, err.Error())
			return
		}

		// Return success response...
	} else {
		// This is a user login under a specific tenant
		token, refreshToken, err = h.service.Login(c.Request.Context(), req.Email, req.Password,
			c.ClientIP(), c.GetHeader("User-Agent"), req.TenantID)
		if err != nil {
			api.ErrorResponse(c, 401, err.Error())
			return
		}
	}

	api.SuccessResponse(c, 200, "login successful", LoginResponse{
		AccessToken:  token,
		RefreshToken: refreshToken,
	})

}

// Refresh godoc
// @Summary Refresh JWT token
// @Description Refresh JWT token using a valid refresh token
// @Tags Tenants
// @Accept json
// @Produce json
// @Param body body RefreshRequest true "Refresh token request"
// @Success 200 {object} RefreshResponse "Token refreshed successfully"
// @Failure 400 {object} BadRequestResponse "Bad request"
// @Failure 401 {object} UnauthorizedResponse "Unauthorized"
// @Failure 500 {object} InternalServerErrorResponse "Internal server error"
// @Router /api/v1/auth/refresh [post]
// func (h *Handler) Refresh(c *gin.Context) {
// 	var req RefreshRequest
// 	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
// 		api.ErrorResponse(c, 401, "Missing or invalid refresh token")
// 		return
// 	}

// 	accessToken, refreshToken, err := h.service.RefreshToken(c.Request.Context(), req.RefreshToken)
// 	if err != nil {
// 		status := http.StatusUnauthorized
// 		if !errors.Is(err, ErrInvalidCredentials) && !errors.Is(err, ErrUserInactive) {
// 			status = http.StatusInternalServerError
// 		}
// 		api.ErrorResponse(c, status, err.Error())
// 		return
// 	}

// 	claims, _ := jwt.ParseToken(accessToken, h.config.JWTSecret)
// 	expiry := time.Time{}
// 	if claims != nil {
// 		expiry = claims.ExpiresAt.Time
// 	}

// 	api.SuccessResponse(c, 200, "message string", RefreshResponse{
// 		AccessToken:  accessToken,
// 		RefreshToken: refreshToken,
// 		ExpiresIn:    int(expiry.Unix()),
// 	})
// }

// Logout godoc
// @Summary Tenant logout
// @Description Logout tenant and invalidate JWT token
// @Tags Tenants
// @Accept json
// @Produce json
// @Success 200 "Logout successful"
// @Failure 400 {object} BadRequestResponse "Bad request"
// @Failure 401 {object} UnauthorizedResponse "Unauthorized"
// @Failure 500 {object} InternalServerErrorResponse "Internal server error"
// @Router /api/v1/tenant/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	authHeader := c.GetHeader(AuthorizationHeader)
	if authHeader == "" {
		api.ErrorResponse(c, 401, "unauthorized")
		return
	}
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, BearerPrefix))
	if token == "" || token == authHeader { // No Bearer prefix found
		api.ErrorResponse(c, 401, "invalid authorization header format")
		return
	}

	claims, exists := c.Get("claims")
	if !exists || claims == nil {
		api.ErrorResponse(c, 401, "unauthorized")
		return
	}

	jwtClaims, ok := claims.(*jwt.Claims)
	if !ok {
		api.ErrorResponse(c, 401, "invalid claims type")
		return
	}

	expiry := time.Until(jwtClaims.ExpiresAt.Time)
	if err := h.service.Logout(c.Request.Context(), token, expiry); err != nil {
		api.ErrorResponse(c, 500, err.Error())
		return
	}
	api.SuccessResponse(c, 200, "Logged out successfully", nil)
}

// RegisterTenantRequest represents the login request payload
// @Description Register admin request payload
type RegisterTenantRequest struct {
	OrgName  string `json:"organization" binding:"required,min=2"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// Tenant Register godoc
// @Summary Tenant Register
// @Description Create tenant with email, password and return JWT token
// @Tags Tenants
// @Accept json
// @Produce json
// @Param body body RegisterTenantRequest true "Register credentials (email and password)"
// @Success 200 {object} RegisterResponse "Registration successful"
// @Failure 400 {object} BadRequestResponse "Bad request"
// @Failure 401 {object} UnauthorizedResponse "Unauthorized"
// @Failure 500 {object} InternalServerErrorResponse "Internal server error"
// @Failure 500 {string} string "Unable to send email at this time, request a new verification code for example@email.com"
// @Router /api/v1/tenant/register [post]
func (h *Handler) RegisterTenant(c *gin.Context) {
	var req RegisterTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ErrorResponse(c, 400, err.Error())
		return
	}
	// Generate verification code and expiry
	code := utils.GenerateOTP()
	expiry := time.Now().Add(10 * time.Minute)

	dev, err := h.service.RegisterTenant(c, req.Email, req.Password, req.OrgName)
	if err != nil {
		log.Printf("error registering dev: %v", err)
		api.ErrorResponse(c, 500, err.Error())
		return
	}

	err = h.service.SetEmailVerification(c.Request.Context(), dev.ID, code, expiry)
	if err != nil {
		log.Printf("error saving verification code: %v", err)
		api.ErrorResponse(c, 500, err.Error())
		return
	}

	// Send verification email
	emailBody, _ := utils.RenderEmailTemplate("templates/auth/verify_email.html", map[string]any{
		"Username": dev.Organization,
		"Code":     code,
	})
	plunk := utils.Plunk{HttpClient: http.DefaultClient, Config: h.config}
	err = plunk.SendEmail(dev.Email, "Verify your Herp account", emailBody)
	if err != nil {
		log.Printf("error sending verification email: %v", err)
		api.ErrorResponse(c, 500, fmt.Sprintf("Unable to send email at this time, request a new verification code for %s", dev.Email))
		return
	}
	api.SuccessResponse(c, 200, "Registration successful", RegisterResponse{
		ID:              dev.ID,
		OrgName:         dev.Organization,
		Email:           dev.Email,
		CreatedAt:       &dev.CreatedAt.Time,
		UpdatedAt:       &dev.UpdatedAt.Time,
		IsActive:        dev.IsActive,
		IsEmailVerified: dev.EmailVerified,
	})
}

// VerifyEmailRequest represents the login request payload
// @Description verify email request payload
type VerifyEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required"`
}

// Verify Email godoc
// @Summary Verify Tenant Email
// @Description Verify admin email with email and code
// @Tags Tenants
// @Accept json
// @Produce json
// @Param body body VerifyEmailRequest true "Verify Email Request"
// @Success 200 "Email verified successfully"
// @Failure 400 {object} BadRequestResponse "Bad request"
// @Failure 401 {object} UnauthorizedResponse "Unauthorized"
// @Failure 500 {object} InternalServerErrorResponse "Internal server error"
// @Router /api/v1/tenant/verify-email [post]
func (h *Handler) VerifyEmail(c *gin.Context) {
	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	ok, err := h.service.VerifyEmailCode(c.Request.Context(), req.Email, req.Code)
	if err != nil {
		api.ErrorResponse(c, 500, err.Error())
		return
	}
	if !ok {
		api.ErrorResponse(c, 400, "Invalid or expired code")
		return
	}
	api.SuccessResponse(c, 200, "Email verified successfully", nil)
}

// Forgot Password godoc
// @Summary Forgot Password
// @Description Initiate password reset by sending a reset code to the user's email
// @Tags Tenants
// @Accept json
// @Produce json
// @Param body body ForgotPasswordRequest true "Forgot Password Request"
// @Success 200 "Reset code sent to email"
// @Failure 400 {object} BadRequestResponse "Bad request"
// @Failure 404 {object} UnauthorizedResponse "User not found"
// @Failure 500 {object} InternalServerErrorResponse "Internal server error"
// @Router /api/v1/tenant/forgot-password [post]
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ErrorResponse(c, 400, err.Error())
		return
	}
	code, err := h.service.ForgotPassword(c.Request.Context(), req.Email)
	if err != nil {
		api.ErrorResponse(c, 404, err.Error())
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
		api.ErrorResponse(c, 500, err.Error())
		return
	}
	api.SuccessResponse(c, 200, "Reset code sent to email", nil)
}

// Reset Password godoc
// @Summary Reset Password
// @Description Reset password using email, reset code, and new password
// @Tags Tenants
// @Accept json
// @Produce json
// @Param body body ResetAdminPasswordRequest true "Reset Password Request"
// @Success 200 "Password reset successful"
// @Failure 400 {object} BadRequestResponse "Bad request or invalid code"
// @Failure 404 {object} UnauthorizedResponse "User not found"
// @Failure 500 {object} InternalServerErrorResponse "Internal server error"
// @Router /api/v1/tenant/reset-password [post]
func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetAdminPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ErrorResponse(c, 400, err.Error())
		return
	}
	err := h.service.ResetDeveloperPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword)
	if err != nil {
		api.ErrorResponse(c, 400, err.Error())
		return
	}
	api.SuccessResponse(c, 200, "Password reset successful", nil)
}
