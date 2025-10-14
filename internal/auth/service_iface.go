package auth

import (
	"context"
	db "nucleus/db/sqlc"
	"nucleus/pkg/jwt"
	"time"
)

type ServiceInterface interface {
	Login(ctx context.Context, email, password, ip, ua string, tenantID int32) (string, string, error)
	RegisterTenant(ctx context.Context, email, password, orgName string) (*db.Tenant, error)
	SetEmailVerification(ctx context.Context, id int32, code string, expiry time.Time) error
	VerifyEmailCode(ctx context.Context, email, code string) (bool, error)
	IsTokenBlacklisted(ctx context.Context, token string) (bool, error)
	ForgotPassword(ctx context.Context, email string) (string, error)
	HasPermission(claims *jwt.Claims, requiredPermission string) bool
	ResetDeveloperPassword(ctx context.Context, email, code, newPassword string) error
	// RefreshToken(ctx context.Context, refreshToken string) (string, string, error)
	Logout(ctx context.Context, token string, expiry time.Duration) error
}

// Querier defines the database methods the Service depends on.
// Both *db.Queries and mocks in tests can implement this.
type Querier interface {
	CreateTenant(ctx context.Context, params db.CreateTenantParams) (db.Tenant, error)
	SetTenantEmailVerification(ctx context.Context, params db.SetTenantEmailVerificationParams) error
	GetTenantByEmail(ctx context.Context, email string) (db.GetTenantByEmailRow, error)
	MarkTenantEmailVerified(ctx context.Context, params db.MarkTenantEmailVerifiedParams) error
	GetTenantByID(ctx context.Context, id int32) (db.Tenant, error)
	LogLoginAttempt(ctx context.Context, params db.LogLoginAttemptParams) error
	ClearTenantResetCode(ctx context.Context, id int32) error
	SetTenantResetCode(ctx context.Context, params db.SetTenantResetCodeParams) error
	UpdateTenantPassword(ctx context.Context, params db.UpdateTenantPasswordParams) error
	GetLoginHistory(ctx context.Context, limit int32) ([]db.LoginHistory, error)
	GetUserByEmail(ctx context.Context, param db.GetUserByEmailParams) (db.GetUserByEmailRow, error)
	UpdateUserLastLogin(ctx context.Context, id int32) error
	GetUserPermissionsWithHierarchy(ctx context.Context, param db.GetUserPermissionsWithHierarchyParams) ([]db.GetUserPermissionsWithHierarchyRow, error)
}
