package user

import (
	"context"
	db "nucleus/db/sqlc"
	"time"
)

// UserService defines the interface for user management operations
type IUserService interface {
	// User CRUD operations
	CreateUser(ctx context.Context, req db.CreateUserParams) (*db.User, error)
	GetUserByID(ctx context.Context, userID, tenantID int32) (*UserResponse, error)
	GetUserByEmail(ctx context.Context, email string, tenantID int32) (*UserResponse, error)
	ListTenantUsers(ctx context.Context, tenantID int32, limit, offset int32) ([]*UserResponse, error)
	ListBusinessUsers(ctx context.Context, tenantID, businessID int32) ([]*UserResponse, error)
	ListBranchUsers(ctx context.Context, tenantID, branchID int32) ([]*UserResponse, error)
	UpdateUser(ctx context.Context, req db.UpdateUserParams) (*UserResponse, error)
	DeactivateUser(ctx context.Context, userID, tenantID int32) (*UserResponse, error)
	CountTenantUsers(ctx context.Context, tenantID int32) (int64, error)

	// Role management
	CreateTenantRole(ctx context.Context, req db.CreateTenantRoleParams) (*RoleResponse, error)
	GetTenantRoleByID(ctx context.Context, roleID, tenantID int32) (*RoleResponse, error)
	GetTenantRoleByName(ctx context.Context, name string, tenantID int32) (*RoleResponse, error)
	ListTenantRoles(ctx context.Context, tenantID int32) ([]*RoleResponse, error)
	ListTenantRolesHierarchy(ctx context.Context, tenantID int32) ([]*RoleHierarchyResponse, error)
	UpdateTenantRole(ctx context.Context, req db.UpdateTenantRoleParams) (*RoleResponse, error)
	DeleteTenantRole(ctx context.Context, roleID, tenantID int32) error
	GetRoleHierarchy(ctx context.Context, roleID, tenantID int32) ([]*RoleHierarchyResponse, error)
	GetChildRoles(ctx context.Context, roleID, tenantID int32) ([]*RoleHierarchyResponse, error)

	// User role assignments
	AssignRoleToUser(ctx context.Context, req db.AssignRoleToUserParams) (*UserRoleAssignmentResponse, error)
	RemoveRoleFromUser(ctx context.Context, req db.RemoveRoleFromUserParams) error
	GetUserRoleAssignments(ctx context.Context, userID int32) ([]*UserRoleAssignmentResponse, error)
	GetRoleUsers(ctx context.Context, roleID int32) ([]*UserRoleAssignmentResponse, error)

	// Permission management
	CreatePermission(ctx context.Context, req db.CreatePermissionParams) (*PermissionResponse, error)
	GetPermissionByCode(ctx context.Context, code string) (*PermissionResponse, error)
	GetPermissionByID(ctx context.Context, id int32) (*PermissionResponse, error)
	ListPermissions(ctx context.Context) ([]*PermissionResponse, error)
	ListPermissionsByModule(ctx context.Context, module string) ([]*PermissionResponse, error)
	UpdatePermission(ctx context.Context, req *UpdatePermissionRequest) (*PermissionResponse, error)

	// Role-permission management
	GrantPermissionToRole(ctx context.Context, req db.GrantPermissionToRoleParams) (*RolePermissionResponse, error)
	RevokePermissionFromRole(ctx context.Context, req db.RevokePermissionFromRoleParams) error
	GetRolePermissions(ctx context.Context, roleID int32) ([]*RolePermissionResponse, error)

	// User permissions (aggregated with hierarchy)
	GetUserPermissions(ctx context.Context, userID, tenantID int32) ([]*UserPermissionResponse, error)
	GetUserPermissionsWithHierarchy(ctx context.Context, userID, tenantID int32) ([]*UserPermissionResponse, error)
	CheckUserPermission(ctx context.Context, req db.CheckUserPermissionParams) (bool, error)

	// Audit logging
	CreateAuditLog(ctx context.Context, req db.CreateAuditLogParams) (*AuditLogResponse, error)
	GetAuditLogs(ctx context.Context, tenantID int32, limit, offset int32) ([]*AuditLogResponse, error)
	GetUserAuditLogs(ctx context.Context, userID, tenantID int32, limit, offset int32) ([]*AuditLogResponse, error)

	// Utility methods
	UpdateUserLastLogin(ctx context.Context, userID int32) error
}

// Request/Response DTOs

type CreateUserRequest struct {
	BusinessID int32          `json:"business_id"`
	BranchID   int32          `json:"branch_id"`
	Email      string         `json:"email" validate:"required,email"`
	Password   string         `json:"password" validate:"required,min=8"`
	FirstName  string         `json:"first_name"`
	LastName   string         `json:"last_name"`
	Phone      string         `json:"phone"`
	AvatarURL  string         `json:"avatar_url"`
	Metadata   map[string]any `json:"metadata"`
}

type UpdateUserRequest struct {
	ID         int32          `json:"id" validate:"required"`
	FirstName  string         `json:"first_name"`
	LastName   string         `json:"last_name"`
	Email      string         `json:"email" validate:"omitempty,email"`
	Phone      string         `json:"phone"`
	BusinessID int32          `json:"business_id"`
	BranchID   int32          `json:"branch_id"`
	AvatarURL  string         `json:"avatar_url"`
	Metadata   map[string]any `json:"metadata"`
}

type UserResponse struct {
	ID         int32   `json:"id"`
	BusinessID *int32  `json:"business_id"`
	BranchID   *int32  `json:"branch_id"`
	Email      string  `json:"email"`
	FirstName  *string `json:"first_name"`
	LastName   *string `json:"last_name"`
	Phone      *string `json:"phone"`
	AvatarURL  *string `json:"avatar_url"`
	// Timezone      string                 `json:"timezone"`
	// Language      string                 `json:"language"`
	IsActive      bool                   `json:"is_active"`
	LastLoginAt   *time.Time             `json:"last_login_at"`
	EmailVerified bool                   `json:"email_verified"`
	TenantName    string                 `json:"tenant_name"`
	BusinessName  *string                `json:"business_name"`
	BranchName    *string                `json:"branch_name"`
	Metadata      map[string]interface{} `json:"metadata"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	Roles         []*RoleResponse        `json:"roles,omitempty"`
}

type CreateRoleRequest struct {
	Name         string                 `json:"name" validate:"required"`
	Description  string                 `json:"description"`
	ParentRoleID int32                  `json:"parent_role_id"`
	IsDefault    bool                   `json:"is_default"`
	Metadata     map[string]interface{} `json:"metadata"`
}

type UpdateRoleRequest struct {
	ID           int32                  `json:"id" validate:"required"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	ParentRoleID int32                  `json:"parent_role_id"`
	IsDefault    bool                   `json:"is_default"`
	IsActive     bool                   `json:"is_active"`
	Metadata     map[string]interface{} `json:"metadata"`
}

type RoleResponse struct {
	ID           int32                  `json:"id"`
	TenantID     int32                  `json:"tenant_id"`
	Name         string                 `json:"name"`
	Description  *string                `json:"description"`
	ParentRoleID *int32                 `json:"parent_role_id"`
	Level        int32                  `json:"level"`
	IsDefault    bool                   `json:"is_default"`
	IsActive     bool                   `json:"is_active"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type RoleHierarchyResponse struct {
	ID           int32                  `json:"id"`
	TenantID     int32                  `json:"tenant_id"`
	Name         string                 `json:"name"`
	Description  *string                `json:"description"`
	ParentRoleID *int32                 `json:"parent_role_id"`
	Level        int32                  `json:"level"`
	IsDefault    bool                   `json:"is_default"`
	IsActive     bool                   `json:"is_active"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Depth        int32                  `json:"depth"` // For hierarchy queries
}

type AssignRoleRequest struct {
	TenantRoleID int32                  `json:"tenant_role_id" validate:"required"`
	BusinessID   int32                  `json:"business_id"`
	BranchID     int32                  `json:"branch_id"`
	AssignedBy   int32                  `json:"assigned_by"`
	ExpiresAt    time.Time              `json:"expires_at"`
	Metadata     map[string]interface{} `json:"metadata"`
}

type RemoveRoleRequest struct {
	UserID       int32 `json:"user_id" validate:"required"`
	TenantRoleID int32 `json:"tenant_role_id" validate:"required"`
	BusinessID   int32 `json:"business_id"`
	BranchID     int32 `json:"branch_id"`
}

type UserRoleAssignmentResponse struct {
	ID              int32                  `json:"id"`
	UserID          int32                  `json:"user_id"`
	TenantRoleID    int32                  `json:"tenant_role_id"`
	BusinessID      *int32                 `json:"business_id"`
	BranchID        *int32                 `json:"branch_id"`
	AssignedBy      *int32                 `json:"assigned_by"`
	AssignedAt      time.Time              `json:"assigned_at"`
	ExpiresAt       *time.Time             `json:"expires_at"`
	IsActive        bool                   `json:"is_active"`
	RoleName        string                 `json:"role_name"`
	RoleDescription *string                `json:"role_description"`
	BusinessName    *string                `json:"business_name"`
	BranchName      *string                `json:"branch_name"`
	UserEmail       *string                `json:"user_email,omitempty"`
	UserFirstName   *string                `json:"user_first_name,omitempty"`
	UserLastName    *string                `json:"user_last_name,omitempty"`
	Metadata        map[string]interface{} `json:"metadata"`
}

type CreatePermissionRequest struct {
	Code        string `json:"code" validate:"required"`
	Description string `json:"description"`
	Module      string `json:"module" validate:"required"`
	Action      string `json:"action" validate:"required"`
	Resource    string `json:"resource"`
	Scope       string `json:"scope" validate:"required,oneof=global tenant business branch"`
}

type UpdatePermissionRequest struct {
	ID          int32   `json:"id" validate:"required"`
	Description *string `json:"description"`
	Module      *string `json:"module"`
	Action      *string `json:"action"`
	Resource    *string `json:"resource"`
	Scope       *string `json:"scope" validate:"omitempty,oneof=global tenant business branch"`
	IsActive    *bool   `json:"is_active"`
}

type PermissionResponse struct {
	ID          int32     `json:"id"`
	Code        string    `json:"code"`
	Description *string   `json:"description"`
	Module      string    `json:"module"`
	Action      string    `json:"action"`
	Resource    *string   `json:"resource"`
	Scope       string    `json:"scope"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

type GrantPermissionRequest struct {
	PermissionID int32          `json:"permission_id" validate:"required"`
	ScopeType    string         `json:"scope_type" validate:"required,oneof=global tenant business branch"`
	ScopeID      int32          `json:"scope_id"`
	GrantedBy    int32          `json:"granted_by"`
	Metadata     map[string]any `json:"metadata"`
}

type RevokePermissionRequest struct {
	TenantRoleID int32  `json:"tenant_role_id" validate:"required"`
	PermissionID int32  `json:"permission_id" validate:"required"`
	ScopeType    string `json:"scope_type" validate:"required,oneof=global tenant business branch"`
	ScopeID      *int32 `json:"scope_id"`
}

type RolePermissionResponse struct {
	ID           int32                  `json:"id"`
	TenantRoleID int32                  `json:"tenant_role_id"`
	PermissionID int32                  `json:"permission_id"`
	ScopeType    string                 `json:"scope_type"`
	ScopeID      *int32                 `json:"scope_id"`
	GrantedBy    *int32                 `json:"granted_by"`
	GrantedAt    time.Time              `json:"granted_at"`
	IsActive     bool                   `json:"is_active"`
	Code         string                 `json:"code"`
	Description  *string                `json:"description"`
	Module       string                 `json:"module"`
	Action       string                 `json:"action"`
	Resource     *string                `json:"resource"`
	RoleName     *string                `json:"role_name,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
}

type UserPermissionResponse struct {
	ID          int32   `json:"id"`
	Code        string  `json:"code"`
	Description *string `json:"description"`
	Module      string  `json:"module"`
	Action      string  `json:"action"`
	Resource    *string `json:"resource"`
	ScopeType   string  `json:"scope_type"`
	ScopeID     *int32  `json:"scope_id"`
	RoleName    string  `json:"role_name"`
}

type CheckUserPermissionRequest struct {
	UserID   int32  `json:"user_id" validate:"required"`
	TenantID int32  `json:"tenant_id" validate:"required"`
	Code     string `json:"code" validate:"required"`
	ScopeID  int32  `json:"business_id"`
	BranchID int32  `json:"branch_id"`
}

type CreateAuditLogRequest struct {
	Action       string                 `json:"action" validate:"required"`
	ActorID      *int32                 `json:"actor_id"`
	TargetUserID *int32                 `json:"target_user_id"`
	TargetRoleID *int32                 `json:"target_role_id"`
	PermissionID *int32                 `json:"permission_id"`
	OldValues    map[string]interface{} `json:"old_values"`
	NewValues    map[string]interface{} `json:"new_values"`
	Reason       *string                `json:"reason"`
	IPAddress    *string                `json:"ip_address"`
	UserAgent    *string                `json:"user_agent"`
	TenantID     int32                  `json:"tenant_id" validate:"required"`
}

type AuditLogResponse struct {
	ID              int32                  `json:"id"`
	Action          string                 `json:"action"`
	ActorID         *int32                 `json:"actor_id"`
	TargetUserID    *int32                 `json:"target_user_id"`
	TargetRoleID    *int32                 `json:"target_role_id"`
	PermissionID    *int32                 `json:"permission_id"`
	OldValues       map[string]interface{} `json:"old_values"`
	NewValues       map[string]interface{} `json:"new_values"`
	Reason          *string                `json:"reason"`
	IPAddress       *string                `json:"ip_address"`
	UserAgent       *string                `json:"user_agent"`
	TenantID        int32                  `json:"tenant_id"`
	ActorEmail      *string                `json:"actor_email"`
	ActorFirstName  *string                `json:"actor_first_name"`
	ActorLastName   *string                `json:"actor_last_name"`
	TargetUserEmail *string                `json:"target_user_email"`
	TargetFirstName *string                `json:"target_first_name"`
	TargetLastName  *string                `json:"target_last_name"`
	TargetRoleName  *string                `json:"target_role_name"`
	PermissionCode  *string                `json:"permission_code"`
	CreatedAt       time.Time              `json:"created_at"`
}
