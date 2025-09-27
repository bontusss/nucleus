package auth

import (
	"context"
	"database/sql"
	db "herp/db/sqlc"
	"time"
)

type ServiceInterface interface {
	Login(ctx context.Context, identifier, password, ip, ua string) (string, string, error)
	RegisterAdmin(ctx context.Context, username, email, password, first, last string) (db.Admin, error)
	SetEmailVerification(ctx context.Context, id int32, code string, expiry time.Time) error
	VerifyEmailCode(ctx context.Context, email, code string) (bool, error)
	ForgotPassword(ctx context.Context, email string) (string, error)
	ResetAdminPassword(ctx context.Context, email, code, newPassword string) error
	RefreshToken(ctx context.Context, refreshToken string) (string, string, error)
	Logout(ctx context.Context, token string, expiry time.Duration) error
}

// Querier defines the database methods the Service depends on.
// Both *db.Queries and mocks in tests can implement this.
type Querier interface {
	CreateAdmin(ctx context.Context, params db.CreateAdminParams) (db.Admin, error)
	SetAdminEmailVerification(ctx context.Context, params db.SetAdminEmailVerificationParams) error
	GetAdminByEmail(ctx context.Context, email string) (db.GetAdminByEmailRow, error)
	MarkAdminEmailVerified(ctx context.Context, params db.MarkAdminEmailVerifiedParams) error
	LogLoginAttempt(ctx context.Context, params db.LogLoginAttemptParams) error
	GetUserPermissions(ctx context.Context, userID int32) ([]string, error)
	GetAdminPermissions(ctx context.Context, adminID int32) ([]string, error)
	CreateRefreshToken(ctx context.Context, params db.CreateRefreshTokenParams) (db.RefreshToken, error)
	GetUserByEmail(ctx context.Context, email sql.NullString) (db.GetUserByEmailRow, error)
	GetUserByUsername(ctx context.Context, username string) (db.GetUserByUsernameRow, error)
	GetAdminByUsername(ctx context.Context, username string) (db.GetAdminByUsernameRow, error)
	GetRefreshToken(ctx context.Context, token string) (db.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, token string) error
	CleanExpiredRefreshTokens(ctx context.Context) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID int32) error
	CreateUser(ctx context.Context, params db.CreateUserParams) (db.User, error)
	UpdateUser(ctx context.Context, params db.UpdateUserParams) (db.User, error)
	DeleteUser(ctx context.Context, id int32) error
	UpdateUserPassword(ctx context.Context, params db.UpdateUserPasswordParams) error
	CreateRole(ctx context.Context, params db.CreateRoleParams) (db.Role, error)
	UpdateRole(ctx context.Context, params db.UpdateRoleParams) (db.Role, error)
	DeleteRole(ctx context.Context, id int32) error
	AddPermissionToRole(ctx context.Context, params db.AddPermissionToRoleParams) error
	RemovePermissionFromRole(ctx context.Context, params db.RemovePermissionFromRoleParams) error
	ListUsers(ctx context.Context) ([]db.ListUsersRow, error)
	ListRoles(ctx context.Context) ([]db.Role, error)
	GetRolePermissions(ctx context.Context, roleID int32) ([]db.Permission, error)
	SetAdminResetCode(ctx context.Context, params db.SetAdminResetCodeParams) error
	UpdateAdminPassword(ctx context.Context, params db.UpdateAdminPasswordParams) error
	ClearAdminResetCode(ctx context.Context, adminID int32) error
	GetUserByID(ctx context.Context, ID int32) (db.GetUserByIDRow, error)
	GetRoleByID(ctx context.Context, id int32) (db.Role, error)
	GetLoginHistory(ctx context.Context, limit int32) ([]db.LoginHistory, error)
	LogActivity(ctx context.Context, params db.LogActivityParams) (db.ActivityLog, error)
}
