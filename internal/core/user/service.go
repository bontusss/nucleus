package user

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	db "nucleus/db/sqlc"
	"nucleus/internal/core/api"
	"nucleus/internal/utils"
	"nucleus/pkg/jwt"
	"nucleus/pkg/monitoring/logging"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	db      *sql.DB
	queries *db.Queries
	logger  *logging.Logger
}

// NewService creates a new user service
func NewService(database *sql.DB, queries *db.Queries) *UserService {
	return &UserService{
		db:      database,
		queries: queries,
	}
}

// ========================================
// USER CRUD OPERATIONS
// ========================================

func (s *UserService) CreateUser(ctx context.Context, param db.CreateUserParams) (*db.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(param.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	param.PasswordHash = string(hashedPassword)

	user, err := s.queries.CreateUser(ctx, param)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

func (s *UserService) GetUserByID(ctx context.Context, userID, tenantID int32) (*UserResponse, error) {
	user, err := s.queries.GetUserByID(ctx, db.GetUserByIDParams{
		ID:        userID,
		TenantsID: tenantID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return s.mapUserRowToResponse(user)
}

func (s *UserService) GetUserByEmail(ctx context.Context, email string, tenantID int32) (*UserResponse, error) {
	user, err := s.queries.GetUserByEmail(ctx, db.GetUserByEmailParams{
		Email:     email,
		TenantsID: tenantID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return s.mapUserByEmailRowToResponse(user)
}

func (s *UserService) ListTenantUsers(ctx context.Context, tenantID int32, limit, offset int32) ([]*UserResponse, error) {
	users, err := s.queries.ListTenantUsers(ctx, db.ListTenantUsersParams{
		TenantsID: tenantID,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list tenant users: %w", err)
	}

	var responses []*UserResponse
	for _, user := range users {
		resp, err := s.mapTenantByUsersRowResponse(user)
		if err != nil {
			continue // Skip invalid users
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

func (s *UserService) ListBusinessUsers(ctx context.Context, tenantID, businessID int32) ([]*UserResponse, error) {
	users, err := s.queries.ListBusinessUsers(ctx, db.ListBusinessUsersParams{
		TenantsID:  tenantID,
		BusinessID: sql.NullInt32{Int32: businessID, Valid: businessID != 0},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list business users: %w", err)
	}

	var responses []*UserResponse
	for _, user := range users {
		resp, err := s.mapBusinessByUsersRowResponse(user)
		if err != nil {
			continue // Skip invalid users
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

func (s *UserService) ListBranchUsers(ctx context.Context, tenantID, branchID int32) ([]*UserResponse, error) {
	users, err := s.queries.ListBranchUsers(ctx, db.ListBranchUsersParams{
		TenantsID: tenantID,
		BranchID:  sql.NullInt32{Int32: branchID, Valid: branchID != 0},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list branch users: %w", err)
	}

	var responses []*UserResponse
	for _, user := range users {
		resp, err := s.mapBranchByUserToResponse(user)
		if err != nil {
			continue // Skip invalid users
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

func (s *UserService) UpdateUser(ctx context.Context, req db.UpdateUserParams) (*UserResponse, error) {

	user, err := s.queries.UpdateUser(ctx, req)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return s.mapUserToResponse(user)
}

func (s *UserService) DeactivateUser(ctx context.Context, userID, tenantID int32) (*UserResponse, error) {
	user, err := s.queries.DeactivateUser(ctx, db.DeactivateUserParams{
		ID:        userID,
		TenantsID: tenantID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to deactivate user: %w", err)
	}

	return s.mapUserToResponse(user)
}

func (s *UserService) CountTenantUsers(ctx context.Context, tenantID int32) (int64, error) {
	count, err := s.queries.CountTenantUsers(ctx, tenantID)
	if err != nil {
		return 0, fmt.Errorf("failed to count tenant users: %w", err)
	}
	return count, nil
}

func (s *UserService) UpdateUserLastLogin(ctx context.Context, userID int32) error {
	return s.queries.UpdateUserLastLogin(ctx, userID)
}

// ========================================
// ROLE MANAGEMENT
// ========================================

func (s *UserService) CreateTenantRole(ctx context.Context, req db.CreateTenantRoleParams) (*RoleResponse, error) {

	role, err := s.queries.CreateTenantRole(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant role: %w", err)
	}

	return s.mapRoleToResponse(role)
}

func (s *UserService) GetTenantRoleByID(ctx context.Context, roleID, tenantID int32) (*RoleResponse, error) {
	role, err := s.queries.GetTenantRoleByID(ctx, db.GetTenantRoleByIDParams{
		ID:       roleID,
		TenantID: tenantID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("role not found")
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return s.mapRoleToResponse(role)
}

func (s *UserService) GetTenantRoleByName(ctx context.Context, name string, tenantID int32) (*RoleResponse, error) {
	role, err := s.queries.GetTenantRoleByName(ctx, db.GetTenantRoleByNameParams{
		Name:     name,
		TenantID: tenantID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("role not found")
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return s.mapRoleToResponse(role)
}

func (s *UserService) ListTenantRoles(ctx context.Context, tenantID int32) ([]*RoleResponse, error) {
	roles, err := s.queries.ListTenantRoles(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenant roles: %w", err)
	}

	var responses []*RoleResponse
	for _, role := range roles {
		resp, err := s.mapRoleToResponse(role)
		if err != nil {
			continue
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

func (s *UserService) ListTenantRolesHierarchy(ctx context.Context, tenantID int32) ([]*RoleHierarchyResponse, error) {
	roles, err := s.queries.ListTenantRolesHierarchy(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenant roles hierarchy: %w", err)
	}

	var responses []*RoleHierarchyResponse
	for _, role := range roles {
		resp, err := s.mapRoleHierarchyToResponse(role)
		if err != nil {
			continue
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

func (s *UserService) UpdateTenantRole(ctx context.Context, req db.UpdateTenantRoleParams) (*RoleResponse, error) {
	role, err := s.queries.UpdateTenantRole(ctx, req)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("role not found")
		}
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	return s.mapRoleToResponse(role)
}

func (s *UserService) DeleteTenantRole(ctx context.Context, roleID, tenantID int32) error {
	return s.queries.DeleteTenantRole(ctx, db.DeleteTenantRoleParams{
		ID:       roleID,
		TenantID: tenantID,
	})
}

func (s *UserService) GetRoleHierarchy(ctx context.Context, roleID, tenantID int32) ([]*RoleHierarchyResponse, error) {
	roles, err := s.queries.GetRoleHierarchy(ctx, db.GetRoleHierarchyParams{
		ID:       roleID,
		TenantID: tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get role hierarchy: %w", err)
	}

	var responses []*RoleHierarchyResponse
	for _, role := range roles {
		resp, err := s.mapGetRoleHierarchyRowToResponse(role)
		if err != nil {
			continue
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

func (s *UserService) GetChildRoles(ctx context.Context, roleID, tenantID int32) ([]*RoleHierarchyResponse, error) {
	roles, err := s.queries.GetChildRoles(ctx, db.GetChildRolesParams{
		ID:       roleID,
		TenantID: tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get child roles: %w", err)
	}

	var responses []*RoleHierarchyResponse
	for _, role := range roles {
		resp, err := s.mapGetChildRolesRowToResponse(role)
		if err != nil {
			continue
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

// ========================================
// USER ROLE ASSIGNMENTS
// ========================================

func (s *UserService) AssignRoleToUser(ctx context.Context, req db.AssignRoleToUserParams) (*UserRoleAssignmentResponse, error) {

	assignment, err := s.queries.AssignRoleToUser(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to assign role to user: %w", err)
	}

	return s.mapUserRoleAssignmentToResponse(assignment)
}

func (s *UserService) RemoveRoleFromUser(ctx context.Context, req db.RemoveRoleFromUserParams) error {
	return s.queries.RemoveRoleFromUser(ctx, req)
}

func (s *UserService) GetUserRoleAssignments(ctx context.Context, userID int32) ([]*UserRoleAssignmentResponse, error) {
	assignments, err := s.queries.GetUserRoleAssignments(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user role assignments: %w", err)
	}

	var responses []*UserRoleAssignmentResponse
	for _, assignment := range assignments {
		resp, err := s.mapUserRoleAssignmentRowToResponse(assignment)
		if err != nil {
			continue
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

func (s *UserService) GetRoleUsers(ctx context.Context, roleID int32) ([]*UserRoleAssignmentResponse, error) {
	assignments, err := s.queries.GetRoleUsers(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role users: %w", err)
	}

	var responses []*UserRoleAssignmentResponse
	for _, assignment := range assignments {
		resp, err := s.mapRoleUsersRowToResponse(assignment)
		if err != nil {
			continue
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

// ========================================
// PERMISSION MANAGEMENT
// ========================================

func (s *UserService) CreatePermission(ctx context.Context, req db.CreatePermissionParams) (*PermissionResponse, error) {
	permission, err := s.queries.CreatePermission(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create permission: %w", err)
	}

	return s.mapPermissionToResponse(permission)
}

func (s *UserService) GetPermissionByCode(ctx context.Context, code string) (*PermissionResponse, error) {
	permission, err := s.queries.GetPermissionByCode(ctx, code)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("permission not found")
		}
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	return s.mapPermissionToResponse(permission)
}

func (s *UserService) GetPermissionByID(ctx context.Context, id int32) (*PermissionResponse, error) {
	permission, err := s.queries.GetPermissionByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("permission not found")
		}
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	return s.mapPermissionToResponse(permission)
}

func (s *UserService) ListPermissions(ctx context.Context) ([]*PermissionResponse, error) {
	permissions, err := s.queries.ListPermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions: %w", err)
	}

	var responses []*PermissionResponse
	for _, permission := range permissions {
		resp, err := s.mapPermissionToResponse(permission)
		if err != nil {
			continue
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

func (s *UserService) ListPermissionsByModule(ctx context.Context, module string) ([]*PermissionResponse, error) {
	permissions, err := s.queries.ListPermissionsByModule(ctx, module)
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions by module: %w", err)
	}

	var responses []*PermissionResponse
	for _, permission := range permissions {
		resp, err := s.mapPermissionToResponse(permission)
		if err != nil {
			continue
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

func (s *UserService) UpdatePermission(ctx context.Context, req *UpdatePermissionRequest) (*PermissionResponse, error) {
	permission, err := s.queries.UpdatePermission(ctx, db.UpdatePermissionParams{
		ID:          req.ID,
		Description: sql.NullString{String: *req.Description, Valid: req.Description != nil},
		Module:      sql.NullString{String: *req.Module, Valid: req.Module != nil},
		Action:      sql.NullString{String: *req.Action, Valid: req.Action != nil},
		Resource:    sql.NullString{String: *req.Resource, Valid: req.Resource != nil},
		Scope:       sql.NullString{String: *req.Scope, Valid: req.Scope != nil},
		IsActive:    sql.NullBool{Bool: *req.IsActive, Valid: req.IsActive != nil},
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("permission not found")
		}
		return nil, fmt.Errorf("failed to update permission: %w", err)
	}

	return s.mapPermissionToResponse(permission)
}

// ========================================
// ROLE-PERMISSION MANAGEMENT
// ========================================

func (s *UserService) GrantPermissionToRole(ctx context.Context, req db.GrantPermissionToRoleParams) (*RolePermissionResponse, error) {

	rolePermission, err := s.queries.GrantPermissionToRole(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to grant permission to role: %w", err)
	}

	return s.mapRolePermissionToResponse(rolePermission)
}

func (s *UserService) RevokePermissionFromRole(ctx context.Context, req db.RevokePermissionFromRoleParams) error {
	return s.queries.RevokePermissionFromRole(ctx, req)
}

func (s *UserService) GetRolePermissions(ctx context.Context, roleID int32) ([]*RolePermissionResponse, error) {
	permissions, err := s.queries.GetRolePermissions(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}

	var responses []*RolePermissionResponse
	for _, permission := range permissions {
		resp, err := s.mapRolePermissionRowToResponse(permission)
		if err != nil {
			continue
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

// ========================================
// USER PERMISSIONS (AGGREGATED)
// ========================================

func (s *UserService) GetUserPermissions(ctx context.Context, userID, tenantID int32) ([]*UserPermissionResponse, error) {
	permissions, err := s.queries.GetUserPermissions(ctx, db.GetUserPermissionsParams{
		ID:        userID,
		TenantsID: tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	var responses []*UserPermissionResponse
	for _, permission := range permissions {
		resp, err := s.mapUserPermissionToResponse(permission)
		if err != nil {
			continue
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

func (s *UserService) GetUserPermissionsWithHierarchy(ctx context.Context, userID, tenantID int32) ([]*UserPermissionResponse, error) {
	permissions, err := s.queries.GetUserPermissionsWithHierarchy(ctx, db.GetUserPermissionsWithHierarchyParams{
		UserID:   userID,
		TenantID: tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions with hierarchy: %w", err)
	}

	var responses []*UserPermissionResponse
	for _, permission := range permissions {
		resp, err := s.mapUserPermissionHierarchyToResponse(permission)
		if err != nil {
			continue
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

func (s *UserService) CheckUserPermission(ctx context.Context, req db.CheckUserPermissionParams) (bool, error) {
	result, err := s.queries.CheckUserPermission(ctx, req)
	if err != nil {
		return false, fmt.Errorf("failed to check user permission: %w", err)
	}

	return result, nil
}

// ========================================
// AUDIT LOGGING
// ========================================

func (s *UserService) CreateAuditLog(ctx context.Context, req db.CreateAuditLogParams) (*AuditLogResponse, error) {
	auditLog, err := s.queries.CreateAuditLog(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create audit log: %w", err)
	}

	return s.mapAuditLogToResponse(auditLog)
}

func (s *UserService) GetAuditLogs(ctx context.Context, tenantID int32, limit, offset int32) ([]*AuditLogResponse, error) {
	logs, err := s.queries.GetAuditLogs(ctx, db.GetAuditLogsParams{
		TenantID: tenantID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}

	var responses []*AuditLogResponse
	for _, log := range logs {
		resp, err := s.mapAuditLogRowToResponse(log)
		if err != nil {
			continue
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

func (s *UserService) GetUserAuditLogs(ctx context.Context, userID, tenantID int32, limit, offset int32) ([]*AuditLogResponse, error) {
	logs, err := s.queries.GetUserAuditLogs(ctx, db.GetUserAuditLogsParams{
		TargetUserID: sql.NullInt32{Int32: userID, Valid: userID != 0},
		TenantID:     tenantID,
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user audit logs: %w", err)
	}

	var responses []*AuditLogResponse
	for _, log := range logs {
		resp, err := s.mapUserAuditLogRowToResponse(log)
		if err != nil {
			continue
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

// ========================================
// MAPPING FUNCTIONS
// ========================================

func (s *UserService) mapUserToResponse(user db.User) (*UserResponse, error) {

	return &UserResponse{
		ID:            user.ID,
		BusinessID:    &user.BusinessID.Int32,
		BranchID:      &user.BranchID.Int32,
		Email:         user.Email,
		FirstName:     &user.FirstName.String,
		LastName:      &user.LastName.String,
		Phone:         &user.Phone.String,
		AvatarURL:     &user.AvatarUrl.String,
		IsActive:      user.IsActive,
		LastLoginAt:   &user.LastLoginAt.Time,
		EmailVerified: utils.GetBoolOrDefault(&user.EmailVerified.Bool, false),
		Metadata:      utils.UnmarshalMetadata(user.Metadata),
		CreatedAt:     user.CreatedAt.Time,
		UpdatedAt:     user.UpdatedAt.Time,
	}, nil
}

func (s *UserService) mapUserByEmailRowToResponse(user db.GetUserByEmailRow) (*UserResponse, error) {

	return &UserResponse{
		ID:            user.ID,
		BusinessID:    &user.BusinessID.Int32,
		BranchID:      &user.BranchID.Int32,
		Email:         user.Email,
		FirstName:     &user.FirstName.String,
		LastName:      &user.LastName.String,
		Phone:         &user.Phone.String,
		AvatarURL:     &user.AvatarUrl.String,
		IsActive:      user.IsActive,
		LastLoginAt:   &user.LastLoginAt.Time,
		EmailVerified: utils.GetBoolOrDefault(&user.EmailVerified.Bool, false),
		TenantName:    utils.GetStringOrDefault(&user.TenantName, ""),
		BusinessName:  &user.BusinessName.String,
		BranchName:    &user.BranchName.String,
		Metadata:      utils.UnmarshalMetadata(user.Metadata),
		CreatedAt:     user.CreatedAt.Time,
		UpdatedAt:     user.UpdatedAt.Time,
	}, nil
}

func (s *UserService) mapTenantByUsersRowResponse(user db.ListTenantUsersRow) (*UserResponse, error) {

	return &UserResponse{
		ID:            user.ID,
		BusinessID:    &user.BusinessID.Int32,
		BranchID:      &user.BranchID.Int32,
		Email:         user.Email,
		FirstName:     &user.FirstName.String,
		LastName:      &user.LastName.String,
		Phone:         &user.Phone.String,
		AvatarURL:     &user.AvatarUrl.String,
		IsActive:      user.IsActive,
		LastLoginAt:   &user.LastLoginAt.Time,
		EmailVerified: utils.GetBoolOrDefault(&user.EmailVerified.Bool, false),
		TenantName:    utils.GetStringOrDefault(&user.TenantName, ""),
		BusinessName:  &user.BusinessName.String,
		BranchName:    &user.BranchName.String,
		Metadata:      utils.UnmarshalMetadata(user.Metadata),
		CreatedAt:     user.CreatedAt.Time,
		UpdatedAt:     user.UpdatedAt.Time,
	}, nil
}

func (s *UserService) mapBusinessByUsersRowResponse(user db.ListBusinessUsersRow) (*UserResponse, error) {

	return &UserResponse{
		ID:            user.ID,
		BusinessID:    &user.BusinessID.Int32,
		BranchID:      &user.BranchID.Int32,
		Email:         user.Email,
		FirstName:     &user.FirstName.String,
		LastName:      &user.LastName.String,
		Phone:         &user.Phone.String,
		AvatarURL:     &user.AvatarUrl.String,
		IsActive:      user.IsActive,
		LastLoginAt:   &user.LastLoginAt.Time,
		EmailVerified: utils.GetBoolOrDefault(&user.EmailVerified.Bool, false),
		TenantName:    utils.GetStringOrDefault(&user.TenantName, ""),
		BusinessName:  &user.BusinessName.String,
		BranchName:    &user.BranchName.String,
		Metadata:      utils.UnmarshalMetadata(user.Metadata),
		CreatedAt:     user.CreatedAt.Time,
		UpdatedAt:     user.UpdatedAt.Time,
	}, nil
}

func (s *UserService) mapBranchByUserToResponse(user db.ListBranchUsersRow) (*UserResponse, error) {

	return &UserResponse{
		ID:            user.ID,
		BusinessID:    &user.BusinessID.Int32,
		BranchID:      &user.BranchID.Int32,
		Email:         user.Email,
		FirstName:     &user.FirstName.String,
		LastName:      &user.LastName.String,
		Phone:         &user.Phone.String,
		AvatarURL:     &user.AvatarUrl.String,
		IsActive:      user.IsActive,
		LastLoginAt:   &user.LastLoginAt.Time,
		EmailVerified: utils.GetBoolOrDefault(&user.EmailVerified.Bool, false),
		TenantName:    utils.GetStringOrDefault(&user.TenantName, ""),
		BusinessName:  &user.BusinessName.String,
		BranchName:    &user.BranchName.String,
		Metadata:      utils.UnmarshalMetadata(user.Metadata),
		CreatedAt:     user.CreatedAt.Time,
		UpdatedAt:     user.UpdatedAt.Time,
	}, nil
}

func (s *UserService) mapUserRowToResponse(user db.GetUserByIDRow) (*UserResponse, error) {

	return &UserResponse{
		ID:            user.ID,
		BusinessID:    &user.BusinessID.Int32,
		BranchID:      &user.BranchID.Int32,
		Email:         user.Email,
		FirstName:     &user.FirstName.String,
		LastName:      &user.LastName.String,
		Phone:         &user.Phone.String,
		AvatarURL:     &user.AvatarUrl.String,
		IsActive:      user.IsActive,
		LastLoginAt:   &user.LastLoginAt.Time,
		EmailVerified: utils.GetBoolOrDefault(&user.EmailVerified.Bool, false),
		TenantName:    utils.GetStringOrDefault(&user.TenantName, ""),
		BusinessName:  &user.BusinessName.String,
		BranchName:    &user.BranchName.String,
		Metadata:      utils.UnmarshalMetadata(user.Metadata),
		CreatedAt:     user.CreatedAt.Time,
		UpdatedAt:     user.UpdatedAt.Time,
	}, nil
}

func (s *UserService) mapRoleToResponse(role db.TenantRole) (*RoleResponse, error) {

	return &RoleResponse{
		ID:           role.ID,
		TenantID:     role.TenantID,
		Name:         role.Name,
		Description:  &role.Description.String,
		ParentRoleID: &role.ParentRoleID.Int32,
		Level:        role.Level,
		IsDefault:    role.IsDefault.Bool,
		IsActive:     role.IsActive.Bool,
		Metadata:     utils.UnmarshalMetadata(role.Metadata),
		CreatedAt:    role.CreatedAt.Time,
		UpdatedAt:    role.UpdatedAt.Time,
	}, nil
}

func (s *UserService) mapRoleHierarchyToResponse(role db.ListTenantRolesHierarchyRow) (*RoleHierarchyResponse, error) {
	return &RoleHierarchyResponse{
		ID:           role.ID,
		TenantID:     role.TenantID,
		Name:         role.Name,
		Description:  &role.Description.String,
		ParentRoleID: &role.ParentRoleID.Int32,
		Level:        role.Level,
		IsDefault:    role.IsDefault.Bool,
		IsActive:     role.IsActive.Bool,
		Metadata:     utils.UnmarshalMetadata(role.Metadata),
		CreatedAt:    role.CreatedAt.Time,
		UpdatedAt:    role.UpdatedAt.Time,
		Depth:        role.Depth,
	}, nil
}

func (s *UserService) mapGetChildRolesRowToResponse(role db.GetChildRolesRow) (*RoleHierarchyResponse, error) {
	return &RoleHierarchyResponse{
		ID:           role.ID,
		TenantID:     role.TenantID,
		Name:         role.Name,
		ParentRoleID: &role.ParentRoleID.Int32,
		Level:        role.Level,
		Depth:        role.Depth,
	}, nil
}

func (s *UserService) mapGetRoleHierarchyRowToResponse(role db.GetRoleHierarchyRow) (*RoleHierarchyResponse, error) {
	return &RoleHierarchyResponse{
		ID:           role.ID,
		TenantID:     role.TenantID,
		Name:         role.Name,
		ParentRoleID: &role.ParentRoleID.Int32,
		Level:        role.Level,
		Depth:        role.Depth,
	}, nil
}

func (s *UserService) mapUserRoleAssignmentToResponse(assignment db.UserRoleAssignment) (*UserRoleAssignmentResponse, error) {

	return &UserRoleAssignmentResponse{
		ID:           assignment.ID,
		UserID:       assignment.UserID,
		TenantRoleID: assignment.TenantRoleID,
		BusinessID:   &assignment.BusinessID.Int32,
		BranchID:     &assignment.BranchID.Int32,
		AssignedBy:   &assignment.AssignedBy.Int32,
		AssignedAt:   assignment.AssignedAt.Time,
		ExpiresAt:    &assignment.ExpiresAt.Time,
		IsActive:     assignment.IsActive.Bool,
		Metadata:     utils.UnmarshalMetadata(assignment.Metadata),
	}, nil
}

func (s *UserService) mapUserRoleAssignmentRowToResponse(assignment db.GetUserRoleAssignmentsRow) (*UserRoleAssignmentResponse, error) {

	return &UserRoleAssignmentResponse{
		ID:              assignment.ID,
		UserID:          assignment.UserID,
		TenantRoleID:    assignment.TenantRoleID,
		BusinessID:      &assignment.BusinessID.Int32,
		BranchID:        &assignment.BranchID.Int32,
		AssignedBy:      &assignment.AssignedBy.Int32,
		AssignedAt:      assignment.AssignedAt.Time,
		ExpiresAt:       &assignment.ExpiresAt.Time,
		IsActive:        assignment.IsActive.Bool,
		RoleName:        assignment.RoleName,
		RoleDescription: &assignment.RoleDescription.String,
		BusinessName:    &assignment.BusinessName.String,
		BranchName:      &assignment.BranchName.String,
		Metadata:        utils.UnmarshalMetadata(assignment.Metadata),
	}, nil
}

func (s *UserService) mapRoleUsersRowToResponse(assignment db.GetRoleUsersRow) (*UserRoleAssignmentResponse, error) {

	return &UserRoleAssignmentResponse{
		ID:            assignment.ID,
		UserID:        assignment.UserID,
		TenantRoleID:  assignment.TenantRoleID,
		BusinessID:    &assignment.BusinessID.Int32,
		BranchID:      &assignment.BranchID.Int32,
		AssignedBy:    &assignment.AssignedBy.Int32,
		AssignedAt:    assignment.AssignedAt.Time,
		ExpiresAt:     &assignment.ExpiresAt.Time,
		IsActive:      assignment.IsActive.Bool,
		UserEmail:     &assignment.Email,
		UserFirstName: &assignment.FirstName.String,
		UserLastName:  &assignment.LastName.String,
		BusinessName:  &assignment.BusinessName.String,
		BranchName:    &assignment.BranchName.String,
		Metadata:      utils.UnmarshalMetadata(assignment.Metadata),
	}, nil
}

func (s *UserService) mapPermissionToResponse(permission db.Permission) (*PermissionResponse, error) {
	return &PermissionResponse{
		ID:          permission.ID,
		Code:        permission.Code,
		Description: &permission.Description.String,
		Module:      permission.Module,
		Action:      permission.Action,
		Resource:    &permission.Resource.String,
		Scope:       permission.Scope,
		IsActive:    utils.GetBoolOrDefault(&permission.IsActive.Bool, true),
		CreatedAt:   utils.GetTimeOrDefault(&permission.CreatedAt.Time),
	}, nil
}

func (s *UserService) mapRolePermissionToResponse(rp db.TenantRolePermission) (*RolePermissionResponse, error) {
	return &RolePermissionResponse{
		ID:           rp.ID,
		TenantRoleID: rp.TenantRoleID,
		PermissionID: rp.PermissionID,
		ScopeType:    rp.ScopeType.String,
		ScopeID:      &rp.ScopeID.Int32,
		GrantedBy:    &rp.GrantedBy.Int32,
		GrantedAt:    rp.GrantedAt.Time,
		IsActive:     rp.IsActive.Bool,
		Metadata:     utils.UnmarshalMetadata(rp.Metadata),
	}, nil
}

func (s *UserService) mapRolePermissionRowToResponse(rp db.GetRolePermissionsRow) (*RolePermissionResponse, error) {

	return &RolePermissionResponse{
		ID:           rp.ID,
		TenantRoleID: rp.TenantRoleID,
		PermissionID: rp.PermissionID,
		ScopeType:    rp.ScopeType.String,
		ScopeID:      &rp.ScopeID.Int32,
		GrantedBy:    &rp.GrantedBy.Int32,
		GrantedAt:    rp.GrantedAt.Time,
		IsActive:     rp.IsActive.Bool,
		Code:         rp.Code,
		Description:  &rp.Description.String,
		Module:       rp.Module,
		Action:       rp.Action,
		Resource:     &rp.Resource.String,
		Metadata:     utils.UnmarshalMetadata(rp.Metadata),
	}, nil
}

func (s *UserService) mapUserPermissionToResponse(up db.GetUserPermissionsRow) (*UserPermissionResponse, error) {
	return &UserPermissionResponse{
		ID:          up.ID,
		Code:        up.Code,
		Description: &up.Description.String,
		Module:      up.Module,
		Action:      up.Action,
		Resource:    &up.Resource.String,
		ScopeType:   up.ScopeType.String,
		ScopeID:     &up.ScopeID.Int32,
		RoleName:    up.RoleName,
	}, nil
}

func (s *UserService) mapUserPermissionHierarchyToResponse(up db.GetUserPermissionsWithHierarchyRow) (*UserPermissionResponse, error) {
	return &UserPermissionResponse{
		ID:          up.ID,
		Code:        up.Code,
		Description: &up.Description.String,
		Module:      up.Module,
		Action:      up.Action,
		Resource:    &up.Resource.String,
		ScopeType:   up.ScopeType.String,
		ScopeID:     &up.ScopeID.Int32,
		RoleName:    up.RoleName,
	}, nil
}

func (s *UserService) mapAuditLogToResponse(al db.RbacAuditLog) (*AuditLogResponse, error) {
	var oldValues, newValues map[string]interface{}

	if al.OldValues.Valid {
		if err := json.Unmarshal(al.OldValues.RawMessage, &oldValues); err != nil {
			oldValues = nil
		}
	}

	if al.NewValues.Valid {
		if err := json.Unmarshal(al.NewValues.RawMessage, &newValues); err != nil {
			newValues = nil
		}
	}

	return &AuditLogResponse{
		ID:           al.ID,
		Action:       al.Action,
		ActorID:      &al.ActorID,
		TargetUserID: &al.TargetUserID.Int32,
		TargetRoleID: &al.TargetRoleID.Int32,
		PermissionID: &al.PermissionID.Int32,
		OldValues:    oldValues,
		NewValues:    newValues,
		Reason:       &al.Reason.String,
		IPAddress:    &al.IpAddress,
		UserAgent:    &al.UserAgent,
		TenantID:     al.TenantID,
		CreatedAt:    al.CreatedAt.Time,
	}, nil
}

func (s *UserService) mapAuditLogRowToResponse(al db.GetAuditLogsRow) (*AuditLogResponse, error) {
	var oldValues, newValues map[string]interface{}

	if al.OldValues.Valid {
		if err := json.Unmarshal(al.OldValues.RawMessage, &oldValues); err != nil {
			oldValues = nil
		}
	}

	if al.NewValues.Valid {
		if err := json.Unmarshal(al.NewValues.RawMessage, &newValues); err != nil {
			newValues = nil
		}
	}

	return &AuditLogResponse{
		ID:              al.ID,
		Action:          al.Action,
		ActorID:         &al.ActorID,
		TargetUserID:    &al.TargetUserID.Int32,
		TargetRoleID:    &al.TargetRoleID.Int32,
		PermissionID:    &al.PermissionID.Int32,
		OldValues:       oldValues,
		NewValues:       newValues,
		Reason:          &al.Reason.String,
		IPAddress:       &al.IpAddress,
		UserAgent:       &al.UserAgent,
		TenantID:        al.TenantID,
		ActorEmail:      &al.ActorEmail.String,
		ActorFirstName:  &al.ActorFirstName.String,
		ActorLastName:   &al.ActorLastName.String,
		TargetUserEmail: &al.TargetUserEmail.String,
		TargetFirstName: &al.TargetFirstName.String,
		TargetLastName:  &al.TargetLastName.String,
		TargetRoleName:  &al.TargetRoleName.String,
		PermissionCode:  &al.PermissionCode.String,
		CreatedAt:       al.CreatedAt.Time,
	}, nil
}

func (s *UserService) mapUserAuditLogRowToResponse(al db.GetUserAuditLogsRow) (*AuditLogResponse, error) {
	var oldValues, newValues map[string]interface{}

	if al.OldValues.Valid {
		if err := json.Unmarshal(al.OldValues.RawMessage, &oldValues); err != nil {
			oldValues = nil
		}
	}

	if al.NewValues.Valid {
		if err := json.Unmarshal(al.NewValues.RawMessage, &newValues); err != nil {
			newValues = nil
		}
	}

	return &AuditLogResponse{
		ID:             al.ID,
		Action:         al.Action,
		ActorID:        &al.ActorID,
		TargetUserID:   &al.TargetUserID.Int32,
		TargetRoleID:   &al.TargetRoleID.Int32,
		PermissionID:   &al.PermissionID.Int32,
		OldValues:      oldValues,
		NewValues:      newValues,
		Reason:         &al.Reason.String,
		IPAddress:      &al.IpAddress,
		UserAgent:      &al.UserAgent,
		TenantID:       al.TenantID,
		ActorEmail:     &al.ActorEmail.String,
		ActorFirstName: &al.ActorFirstName.String,
		ActorLastName:  &al.ActorLastName.String,
		TargetRoleName: &al.TargetRoleName.String,
		PermissionCode: &al.PermissionCode.String,
		CreatedAt:      al.CreatedAt.Time,
	}, nil
}

func (h *UserService) HasPermission(c *gin.Context, code string, scopeID, branchID int32) bool {
	user, ok := jwt.GetUserFromContext(c)
	if !ok {
		api.ErrorResponse(c, 400, "error getting user from jwt claims")
	}

	if user.UserID == 0 || user.TenantID == 0 {
		return false
	}

	req := db.CheckUserPermissionParams{
		UserID:   int32(user.UserID),
		TenantID: int32(user.TenantID),
		Code:     code,
		ScopeID:  sql.NullInt32{Int32: scopeID, Valid: scopeID != 0},
		BranchID: sql.NullInt32{Int32: branchID, Valid: branchID != 0},
	}

	hasPermission, err := h.CheckUserPermission(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Failed to check permission", "error", err, "user_id", user.UserID, "permission", code)
		return false
	}

	return hasPermission
}

func (h *UserService) LogAudit(c *gin.Context, user *jwt.Claims, action string, targetUserID int32, targetRoleID int32, permissionID int32, newValues, oldValues map[string]any) {

	req := db.CreateAuditLogParams{
		Action:       action,
		ActorID:      int32(user.UserID),
		TargetUserID: sql.NullInt32{Int32: targetUserID, Valid: targetUserID != 0},
		TargetRoleID: sql.NullInt32{Int32: targetRoleID, Valid: targetRoleID != 0},
		PermissionID: sql.NullInt32{Int32: permissionID, Valid: permissionID != 0},
		OldValues:    utils.MarshalMetadata(oldValues),
		NewValues:    utils.MarshalMetadata(newValues),
		IpAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
		TenantID:     int32(user.TenantID),
	}

	// Log audit in background - don't block the response
	// go func() {
	fmt.Printf("creating audit log")
	_, err := h.CreateAuditLog(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Failed to create audit log", "error", err, "action", action)
		return
	}
	fmt.Printf("created audit log")
	// }()
}
