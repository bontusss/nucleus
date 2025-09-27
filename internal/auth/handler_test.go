package auth

// import (
// 	"bytes"
// 	"context"
// 	"errors"
// 	db "herp/db/sqlc"
// 	"herp/internal/config"
// 	"herp/pkg/monitoring/logging"
// 	"net/http/httptest"
// 	"testing"
// 	"time"

// 	"github.com/gin-gonic/gin"
// 	"github.com/stretchr/testify/assert"
// )

// type mockService struct {
// 	loginFunc              func(ctx context.Context, identifier, password, ip, ua string) (string, string, error)
// 	registerAdminFunc      func(ctx context.Context, username, email, password, first, last string) (db.Admin, error)
// 	setEmailVerification   func(ctx context.Context, id int32, code string, expiry time.Time) error
// 	verifyEmailCodeFunc    func(ctx context.Context, email, code string) (bool, error)
// 	forgotPasswordFunc     func(ctx context.Context, email string) (string, error)
// 	resetAdminPasswordFunc func(ctx context.Context, email, code, newPassword string) error
// 	refreshTokenFunc       func(ctx context.Context, refreshToken string) (string, string, error)
// 	logoutFunc             func(ctx context.Context, token string, expiry time.Duration) error
// }

// // Patch SendEmail to always succeed in tests
// // func init() {
// // 	utils.Plunk{}.SendEmail = func(to, subject, body string) error {
// // 		return nil
// // 	}
// // }

// func (m *mockService) Login(ctx context.Context, identifier, password, ip, ua string) (string, string, error) {
// 	return m.loginFunc(ctx, identifier, password, ip, ua)
// }
// func (m *mockService) RegisterAdmin(ctx context.Context, username, email, password, first, last string) (db.Admin, error) {
// 	return m.registerAdminFunc(ctx, username, email, password, first, last)
// }
// func (m *mockService) SetEmailVerification(ctx context.Context, id int32, code string, expiry time.Time) error {
// 	return m.setEmailVerification(ctx, id, code, expiry)
// }
// func (m *mockService) VerifyEmailCode(ctx context.Context, email, code string) (bool, error) {
// 	return m.verifyEmailCodeFunc(ctx, email, code)
// }
// func (m *mockService) ForgotPassword(ctx context.Context, email string) (string, error) {
// 	return m.forgotPasswordFunc(ctx, email)
// }
// func (m *mockService) ResetAdminPassword(ctx context.Context, email, code, newPassword string) error {
// 	return m.resetAdminPasswordFunc(ctx, email, code, newPassword)
// }
// func (m *mockService) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
// 	return m.refreshTokenFunc(ctx, refreshToken)
// }
// func (m *mockService) Logout(ctx context.Context, token string, expiry time.Duration) error {
// 	return m.logoutFunc(ctx, token, expiry)
// }

// // --- Helper to create handler and router ---
// func setupHandler(svc *mockService) (*Handler, *gin.Engine) {
// 	gin.SetMode(gin.TestMode)
// 	cfg := &config.Config{}
// 	logger := logging.NewLogger(cfg)
// 	h := NewHandler(svc, cfg, logger, "test")
// 	r := gin.New()
// 	return h, r
// }

// func TestHandler_Login_Success(t *testing.T) {
// 	svc := &mockService{
// 		loginFunc: func(ctx context.Context, identifier, password, ip, ua string) (string, string, error) {
// 			return "token", "refresh", nil
// 		},
// 	}
// 	h, r := setupHandler(svc)
// 	r.POST("/login", h.Login)

// 	body := `{"username":"admin","password":"pass"}`
// 	req := httptest.NewRequest("POST", "/login", bytes.NewBufferString(body))
// 	req.Header.Set("Content-Type", "application/json")
// 	w := httptest.NewRecorder()

// 	r.ServeHTTP(w, req)
// 	assert.Equal(t, 200, w.Code)
// 	assert.Contains(t, w.Body.String(), "token")
// }

// func TestHandler_Login_InvalidCredentials(t *testing.T) {
// 	svc := &mockService{
// 		loginFunc: func(ctx context.Context, identifier, password, ip, ua string) (string, string, error) {
// 			return "", "", ErrInvalidCredentials
// 		},
// 	}
// 	h, r := setupHandler(svc)
// 	r.POST("/login", h.Login)

// 	body := `{"username":"admin","password":"wrong"}`
// 	req := httptest.NewRequest("POST", "/login", bytes.NewBufferString(body))
// 	req.Header.Set("Content-Type", "application/json")
// 	w := httptest.NewRecorder()

// 	r.ServeHTTP(w, req)
// 	assert.Equal(t, 401, w.Code)
// 	assert.Contains(t, w.Body.String(), "invalid credentials")
// }

// func TestHandler_RegisterAdmin_Success(t *testing.T) {
// 	svc := &mockService{
// 		registerAdminFunc: func(ctx context.Context, username, email, password, first, last string) (db.Admin, error) {
// 			return db.Admin{
// 				ID:        1,
// 				Username:  username,
// 				Email:     email,
// 				FirstName: first,
// 				LastName:  last,
// 				IsActive:  true,
// 				RoleID:    1,
// 			}, nil
// 		},
// 		setEmailVerification: func(ctx context.Context, id int32, code string, expiry time.Time) error {
// 			return nil
// 		},
// 	}
// 	h, r := setupHandler(svc)
// 	r.POST("/register", h.RegisterAdmin)

// 	body := `{"first_name":"Admin","last_name":"User","username":"admin","email":"admin@hotel.com","password":"password123"}`
// 	req := httptest.NewRequest("POST", "/register", bytes.NewBufferString(body))
// 	req.Header.Set("Content-Type", "application/json")
// 	w := httptest.NewRecorder()

// 	r.ServeHTTP(w, req)
// 	assert.Equal(t, 200, w.Code)
// 	assert.Contains(t, w.Body.String(), "Registration successful")
// }

// func TestHandler_RegisterAdmin_BadRequest(t *testing.T) {
// 	svc := &mockService{}
// 	h, r := setupHandler(svc)
// 	r.POST("/register", h.RegisterAdmin)

// 	body := `{"first_name":"","last_name":"","username":"","email":"not-an-email","password":""}`
// 	req := httptest.NewRequest("POST", "/register", bytes.NewBufferString(body))
// 	req.Header.Set("Content-Type", "application/json")
// 	w := httptest.NewRecorder()

// 	r.ServeHTTP(w, req)
// 	assert.Equal(t, 400, w.Code)
// }

// // func TestHandler_VerifyEmail_Success(t *testing.T) {
// // 	svc := &mockService{
// // 		verifyEmailCodeFunc: func(ctx context.Context, email, code string) (bool, error) {
// // 			return true, nil
// // 		},
// // 	}
// // 	h, r := setupHandler(svc)
// // 	r.POST("/verify-email", h.VerifyEmail)

// // 	body := `{"email":"admin@hotel.com","code":"123456"}`
// // 	req := httptest.NewRequest("POST", "/verify-email", bytes.NewBufferString(body))
// // 	req.Header.Set("Content-Type", "application/json")
// // 	w := httptest.NewRecorder()

// // 	r.ServeHTTP(w, req)
// // 	assert.Equal(t, 200, w.Code)
// // 	assert.Contains(t, w.Body.String(), "Email verified successfully")
// // }

// // func TestHandler_VerifyEmail_InvalidCode(t *testing.T) {
// // 	svc := &mockService{
// // 		verifyEmailCodeFunc: func(ctx context.Context, email, code string) (bool, error) {
// // 			return false, nil
// // 		},
// // 	}
// // 	h, r := setupHandler(svc)
// // 	r.POST("/verify-email", h.VerifyEmail)

// // 	body := `{"email":"admin@hotel.com","code":"wrong"}`
// // 	req := httptest.NewRequest("POST", "/verify-email", bytes.NewBufferString(body))
// // 	req.Header.Set("Content-Type", "application/json")
// // 	w := httptest.NewRecorder()

// // 	r.ServeHTTP(w, req)
// // 	assert.Equal(t, 400, w.Code)
// // 	assert.Contains(t, w.Body.String(), "Invalid or expired code")
// // }

// func TestHandler_ForgotPassword_Success(t *testing.T) {
// 	svc := &mockService{
// 		forgotPasswordFunc: func(ctx context.Context, email string) (string, error) {
// 			return "resetcode", nil
// 		},
// 	}
// 	h, r := setupHandler(svc)
// 	r.POST("/forgot-password", h.ForgotPassword)

// 	body := `{"email":"user@example.com"}`
// 	req := httptest.NewRequest("POST", "/forgot-password", bytes.NewBufferString(body))
// 	req.Header.Set("Content-Type", "application/json")
// 	w := httptest.NewRecorder()

// 	r.ServeHTTP(w, req)
// 	assert.Equal(t, 200, w.Code)
// 	assert.Contains(t, w.Body.String(), "Reset code sent to email")
// }

// func TestHandler_ForgotPassword_NotFound(t *testing.T) {
// 	svc := &mockService{
// 		forgotPasswordFunc: func(ctx context.Context, email string) (string, error) {
// 			return "", errors.New("user not found")
// 		},
// 	}
// 	h, r := setupHandler(svc)
// 	r.POST("/forgot-password", h.ForgotPassword)

// 	body := `{"email":"user@example.com"}`
// 	req := httptest.NewRequest("POST", "/forgot-password", bytes.NewBufferString(body))
// 	req.Header.Set("Content-Type", "application/json")
// 	w := httptest.NewRecorder()

// 	r.ServeHTTP(w, req)
// 	assert.Equal(t, 404, w.Code)
// 	assert.Contains(t, w.Body.String(), "user not found")
// }

// func TestHandler_ResetPassword_Success(t *testing.T) {
// 	svc := &mockService{
// 		resetAdminPasswordFunc: func(ctx context.Context, email, code, newPassword string) error {
// 			return nil
// 		},
// 	}
// 	h, r := setupHandler(svc)
// 	r.POST("/reset-password", h.ResetPassword)

// 	body := `{"email":"admin@hotel.com","code":"123456","new_password":"NewPassword123"}`
// 	req := httptest.NewRequest("POST", "/reset-password", bytes.NewBufferString(body))
// 	req.Header.Set("Content-Type", "application/json")
// 	w := httptest.NewRecorder()

// 	r.ServeHTTP(w, req)
// 	assert.Equal(t, 200, w.Code)
// 	assert.Contains(t, w.Body.String(), "Password reset successful")
// }

// func TestHandler_ResetPassword_BadRequest(t *testing.T) {
// 	svc := &mockService{
// 		resetAdminPasswordFunc: func(ctx context.Context, email, code, newPassword string) error {
// 			return errors.New("invalid code")
// 		},
// 	}
// 	h, r := setupHandler(svc)
// 	r.POST("/reset-password", h.ResetPassword)

// 	body := `{"email":"admin@hotel.com","code":"wrong","new_password":"NewPassword123"}`
// 	req := httptest.NewRequest("POST", "/reset-password", bytes.NewBufferString(body))
// 	req.Header.Set("Content-Type", "application/json")
// 	w := httptest.NewRecorder()

// 	r.ServeHTTP(w, req)
// 	assert.Equal(t, 400, w.Code)
// 	assert.Contains(t, w.Body.String(), "invalid code")
// }
