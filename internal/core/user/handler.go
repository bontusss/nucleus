package user

import (
	"database/sql"
	"net/http"
	db "nucleus/db/sqlc"
	"nucleus/internal/auth"
	"nucleus/internal/config"
	"nucleus/internal/core/api"
	"nucleus/internal/key"
	"nucleus/internal/utils"
	"nucleus/pkg/jwt"
	"nucleus/pkg/monitoring/logging"
	"strconv"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service IUserService
	logger  *logging.Logger
	config  *config.Config
}

// NewHandler creates a new user handler
func NewHandler(service IUserService, logger *logging.Logger, config *config.Config) *UserHandler {
	return &UserHandler{
		service: service,
		logger:  logger,
		config:  config,
	}
}

// RegisterRoutes registers all user-related routes
func (h *UserHandler) RegisterRoutes(router *gin.RouterGroup, authService auth.ServiceInterface, apiKeyService *key.Service) {
	// User management routes
	users := router.Group("/users")
	users.Use(key.APIKeyMiddleware(apiKeyService, "user"))
	{
		// User CRUD
		users.POST("", h.CreateUser)
		users.GET("/:id", h.GetUser)
		users.PUT("/:id", h.UpdateUser)
		users.DELETE("/:id", h.DeactivateUser)
		users.GET("", h.ListUsers) // Query params: ?business_id=X&branch_id=Y&limit=10&offset=0

		// User role assignments
		users.POST("/:id/roles", h.AssignRoleToUser)
		users.DELETE("/:id/roles/:role_id", h.RemoveRoleFromUser)
		users.GET("/:id/roles", h.GetUserRoles)
		users.GET("/:id/permissions", h.GetUserPermissions)
	}

	// Role management routes
	roles := router.Group("/roles")
	roles.Use(key.APIKeyMiddleware(apiKeyService, "role"))
	{
		roles.POST("", h.CreateRole)
		roles.GET("/:id", h.GetRole)
		roles.PUT("/:id", h.UpdateRole)
		roles.DELETE("/:id", h.DeleteRole)
		roles.GET("", h.ListRoles)
		roles.GET("/hierarchy", h.GetRolesHierarchy)
		roles.GET("/:id/hierarchy", h.GetRoleHierarchy)
		roles.GET("/:id/children", h.GetChildRoles)
		roles.GET("/:id/users", h.GetRoleUsers)

		// Role permissions
		roles.POST("/:id/permissions", h.GrantPermissionToRole)
		roles.DELETE("/:id/permissions/:permission_id", h.RevokePermissionFromRole)
		roles.GET("/:id/permissions", h.GetRolePermissions)
	}

	// Permission management routes
	permissions := router.Group("/permissions")
	permissions.Use(key.APIKeyMiddleware(apiKeyService, "permission"))
	{
		permissions.POST("", h.CreatePermission)
		permissions.GET("/:id", h.GetPermission)
		permissions.PUT("/:id", h.UpdatePermission)
		permissions.GET("", h.ListPermissions) // Query params: ?module=inventory
		permissions.POST("/check", h.CheckPermission)
	}

	// Audit logs routes
	audit := router.Group("/audit")
	audit.Use(auth.AuthMiiddleware(authService.(*auth.Service)))
	{
		audit.GET("", h.GetAuditLogs)
		audit.GET("/users/:id", h.GetUserAuditLogs)
	}
}

// ========================================
// USER MANAGEMENT HANDLERS
// ========================================

// CreateUser godoc
// @Summary Create a new user
// @Description Create a new user under the current tenant
// @Tags Users
// @Accept json
// @Produce json
// @Param request body CreateUserRequest true "User creation request"
// @Success 201 {object} UserResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users [post]
// @Security BearerAuth
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	// Get tenant ID from auth context
	tenant, ok := key.GetTenantFromApikey(c)
	if !ok {
		api.ErrorResponse(c, 400, "error getting tenenat details from apikey")
	}

	params := db.CreateUserParams{
		TenantsID:    tenant.TenantID,
		BusinessID:   sql.NullInt32{Int32: req.BusinessID, Valid: req.BusinessID != 0},
		BranchID:     sql.NullInt32{Int32: req.BranchID, Valid: req.BranchID != 0},
		Email:        req.Email,
		PasswordHash: req.Password,
		FirstName:    sql.NullString{String: req.FirstName, Valid: req.FirstName != ""},
		LastName:     sql.NullString{String: req.LastName, Valid: req.LastName != ""},
		Phone:        sql.NullString{String: req.Phone, Valid: req.Phone != ""},
		Metadata:     utils.MarshalMetadata(req.Metadata),
		IsActive:     true,
	}

	h.logger.Infof("tenant from apikey: %v", params.TenantsID)
	user, err := h.service.CreateUser(c.Request.Context(), params)
	if err != nil {
		h.logger.Errorf("Failed to create user on tenant %d", tenant.TenantID)
		api.ErrorResponse(c, 500, err.Error())
		return
	}

	// TODO: uncomment after user login impl
	// Log audit trail
	// h.logAudit(c, "user_created", nil, &user.ID, nil, nil, map[string]interface{}{
	// 	"user_email":  user.Email,
	// 	"business_id": user.BusinessID,
	// 	"branch_id":   user.BranchID,
	// }, nil)

	// TODO: send email to created user and publish webhook

	api.SuccessResponse(c, 201, "user created", UserResponse{
		ID:         user.ID,
		BusinessID: &user.BusinessID.Int32,
		BranchID:   &user.BranchID.Int32,
		Email:      user.Email,
		FirstName:  &user.FirstName.String,
		LastName:   &user.LastName.String,
		Phone:      &user.Phone.String,
		Metadata:   utils.UnmarshalMetadata(user.Metadata),
	})
}

// GetUser godoc
// @Summary Get user by ID
// @Description Get user information by ID
// @Tags Users
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} UserResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /users/{id} [get]
// @Security BearerAuth
func (h *UserHandler) GetUser(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		api.ErrorResponse(c, 400, err.Error())
	}

	tenant, ok := key.GetTenantFromApikey(c)
	if !ok {
		api.ErrorResponse(c, 400, "error getting tenenat details from apikey")
	}

	// Check permissions
	// if !h.hasPermission(c, "user:read", nil, nil) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	user, err := h.service.GetUserByID(c.Request.Context(), int32(userID), tenant.TenantID)
	if err != nil {
		api.ErrorResponse(c, 500, err.Error())
		return
	}

	api.SuccessResponse(c, 200, "", user)
}

// UpdateUser godoc
// @Summary Update user
// @Description Update user information
// @Tags Users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param request body UpdateUserRequest true "User update request"
// @Success 200 {object} UserResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /users/{id} [put]
// @Security BearerAuth
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	req.ID = int32(userID)
	tenant, ok := key.GetTenantFromApikey(c)
	if !ok {
		api.ErrorResponse(c, 400, "error getting tenenat details from apikey")
	}

	// Check permissions
	// if !h.hasPermission(c, "user:update", req.BusinessID, req.BranchID) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	// Check if business and branch exists and are active
	// if req.BranchID != 0 || req.BusinessID != 0 {
	// 	business, err := h.businessService.GetBusiness(c, db.GetBusinessParams{
	// 		TenantsID: tenant.TenantID,
	// 		ID:        req.BusinessID,
	// 	})
	// 	if err != nil {
	// 		api.ErrorResponse(c, 400, "business does not exist")
	// 		return
	// 	} else if !business.IsActive {
	// 		api.ErrorResponse(c, 400, "business is not active")
	// 		return
	// 	}

	// 	branch, err := h.businessService.GetBranch(c, db.GetBranchParams{
	// 		BusinessID: req.BusinessID,
	// 		ID:         req.BranchID,
	// 	})
	// 	if err != nil {
	// 		api.ErrorResponse(c, 400, "branch does not exist")
	// 		return
	// 	} else if !branch.IsActive {
	// 		api.ErrorResponse(c, 400, "branch is not active")
	// 	}
	// }

	// Get old user data for audit
	oldUser, _ := h.service.GetUserByID(c.Request.Context(), req.ID, tenant.TenantID)

	params := db.UpdateUserParams{
		ID:       int32(userID),
		TenantID: tenant.TenantID,
		Metadata: utils.MarshalMetadata(oldUser.Metadata),
	}

	utils.PatchNullString(&params.AvatarUrl, &req.AvatarURL)
	utils.PatchNullString(&params.FirstName, &req.FirstName)
	utils.PatchNullString(&params.LastName, &req.LastName)
	utils.PatchNullString(&params.Email, &req.Email)
	utils.PatchNullString(&params.Phone, &req.Phone)
	utils.PatchInt32(&params.BranchID, &req.BranchID)
	utils.PatchInt32(&params.BusinessID, &req.BusinessID)
	utils.PatchMetadata(&params.Metadata, req.Metadata)

	user, err := h.service.UpdateUser(c.Request.Context(), params)
	if err != nil {
		h.logger.Error("Failed to update user", "error", err, "user_id", userID)
		api.ErrorResponse(c, 500, err.Error())
		return
	}

	// Log audit trail
	// var oldValues map[string]interface{}
	// if oldUser != nil {
	// 	oldValues = map[string]interface{}{
	// 		"first_name": oldUser.FirstName,
	// 		"last_name":  oldUser.LastName,
	// 		"email":      oldUser.Email,
	// 		"is_active":  oldUser.IsActive,
	// 	}
	// }

	// h.logAudit(c, "user_updated", nil, &user.ID, nil, nil, map[string]interface{}{
	// 	"first_name": user.FirstName,
	// 	"last_name":  user.LastName,
	// 	"email":      user.Email,
	// 	"is_active":  user.IsActive,
	// }, oldValues)

	api.SuccessResponse(c, 200, "user updated", user)
}

// DeactivateUser godoc
// @Summary Deactivate user
// @Description Deactivate a user account
// @Tags Users
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} UserResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /users/{id} [delete]
// @Security BearerAuth
func (h *UserHandler) DeactivateUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	tenant, ok := key.GetTenantFromApikey(c)
	if !ok {
		api.ErrorResponse(c, 400, "error getting tenenat details from apikey")
	}

	// // Check permissions
	// if !h.hasPermission(c, "user:delete", nil, nil) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	user, err := h.service.DeactivateUser(c.Request.Context(), int32(userID), tenant.TenantID)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		h.logger.Error("Failed to deactivate user", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate user"})
		return
	}

	// Log audit trail
	// h.logAudit(c, "user_deactivated", nil, &user.ID, nil, nil, map[string]interface{}{
	// 	"email":     user.Email,
	// 	"is_active": false,
	// }, map[string]interface{}{
	// 	"is_active": true,
	// })

	c.JSON(http.StatusOK, user)
}

// ListUsers godoc
// @Summary List users
// @Description List users with optional filters
// @Tags Users
// @Produce json
// @Param business_id query int false "Business ID filter"
// @Param branch_id query int false "Branch ID filter"
// @Param limit query int false "Limit (default 50)"
// @Param offset query int false "Offset (default 0)"
// @Success 200 {array} UserResponse
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /users [get]
// @Security BearerAuth
func (h *UserHandler) ListUsers(c *gin.Context) {
	tenant, ok := key.GetTenantFromApikey(c)
	if !ok {
		api.ErrorResponse(c, 400, "error getting tenenat details from apikey")
	}

	// Check permissions
	// if !h.hasPermission(c, "user:read", nil, nil) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	// Parse query parameters
	businessID := h.parseOptionalInt32(c.Query("business_id"))
	branchID := h.parseOptionalInt32(c.Query("branch_id"))
	limit := h.parseInt32WithDefault(c.Query("limit"), 50)
	offset := h.parseInt32WithDefault(c.Query("offset"), 0)

	var users []*UserResponse
	var err error

	if businessID != nil {
		users, err = h.service.ListBusinessUsers(c.Request.Context(), tenant.TenantID, *businessID)
	} else if branchID != nil {
		users, err = h.service.ListBranchUsers(c.Request.Context(), tenant.TenantID, *branchID)
	} else {
		users, err = h.service.ListTenantUsers(c.Request.Context(), tenant.TenantID, limit, offset)
	}

	if err != nil {
		h.logger.Error("Failed to list users", "error", err, "tenant_id", tenant.TenantID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users":  users,
		"count":  len(users),
		"limit":  limit,
		"offset": offset,
	})
}

// ========================================
// ROLE MANAGEMENT HANDLERS
// ========================================

// CreateRole godoc
// @Summary Create a new role
// @Description Create a new role under the current tenant
// @Tags Roles
// @Accept json
// @Produce json
// @Param request body CreateRoleRequest true "Role creation request"
// @Success 201 {object} RoleResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /roles [post]
// @Security BearerAuth
func (h *UserHandler) CreateRole(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	tenant, ok := key.GetTenantFromApikey(c)
	if !ok {
		api.ErrorResponse(c, 400, "error getting tenenat details from apikey")
	}

	// Check permissions
	// if !h.hasPermission(c, "role:create", nil, nil) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	params := db.CreateTenantRoleParams{
		TenantID:     tenant.TenantID,
		Name:         req.Name,
		Description:  sql.NullString{String: req.Description, Valid: req.Description != ""},
		ParentRoleID: sql.NullInt32{Int32: req.ParentRoleID, Valid: req.ParentRoleID != 0},
		IsDefault:    sql.NullBool{Bool: req.IsDefault, Valid: !req.IsDefault},
		Metadata:     utils.MarshalMetadata(req.Metadata),
	}

	role, err := h.service.CreateTenantRole(c.Request.Context(), params)
	if err != nil {
		h.logger.Error("Failed to create role", "error", err, "tenant_id", tenant.TenantID)
		api.ErrorResponse(c, 500, err.Error())
		return
	}

	// Log audit trail
	// h.logAudit(c, "role_created", nil, nil, &role.ID, nil, map[string]interface{}{
	// 	"role_name":      role.Name,
	// 	"parent_role_id": role.ParentRoleID,
	// }, nil)

	api.SuccessResponse(c, 201, "role created", role)
}

// GetRole godoc
// @Summary Get role by ID
// @Description Get role information by ID
// @Tags Roles
// @Produce json
// @Param id path int true "Role ID"
// @Success 200 {object} RoleResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /roles/{id} [get]
// @Security BearerAuth
func (h *UserHandler) GetRole(c *gin.Context) {
	roleID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	tenant, ok := key.GetTenantFromApikey(c)
	if !ok {
		api.ErrorResponse(c, 400, "error getting tenenat details from apikey")
	}

	// Check permissions
	// if !h.hasPermission(c, "role:read", nil, nil) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	role, err := h.service.GetTenantRoleByID(c.Request.Context(), int32(roleID), tenant.TenantID)
	if err != nil {
		h.logger.Error("Failed to get role", "error", err, "role_id", roleID)
		api.ErrorResponse(c, 500, err.Error())
		return
	}

	api.SuccessResponse(c, 200, "", role)
}

// UpdateRole godoc
// @Summary Update role
// @Description Update role information
// @Tags Roles
// @Accept json
// @Produce json
// @Param id path int true "Role ID"
// @Param request body UpdateRoleRequest true "Role update request"
// @Success 200 {object} RoleResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /roles/{id} [put]
// @Security BearerAuth
func (h *UserHandler) UpdateRole(c *gin.Context) {
	roleID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}

	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	req.ID = int32(roleID)
	tenant, ok := key.GetTenantFromApikey(c)
	if !ok {
		api.ErrorResponse(c, 400, "error getting tenenat details from apikey")
	}

	// Check permissions
	// if !h.hasPermission(c, "role:update", nil, nil) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	r, err := h.service.GetTenantRoleByID(c, req.ID, tenant.TenantID)
	if err != nil {
		api.ErrorResponse(c, 500, err.Error())
		return
	}

	params := db.UpdateTenantRoleParams{
		ID:       req.ID,
		TenantID: tenant.TenantID,
		Metadata: utils.MarshalMetadata(r.Metadata),
	}

	utils.PatchNullString(&params.Name, &req.Name)
	utils.PatchNullString(&params.Description, &req.Description)
	utils.PatchInt32(&params.ParentRoleID, &req.ParentRoleID)
	utils.PatchNullBool(&params.IsActive, &req.IsActive)
	utils.PatchNullBool(&params.IsDefault, &req.IsDefault)
	utils.PatchMetadata(&params.Metadata, req.Metadata)

	role, err := h.service.UpdateTenantRole(c.Request.Context(), params)
	if err != nil {
		if err.Error() == "role not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
			return
		}
		h.logger.Error("Failed to update role", "error", err, "role_id", roleID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update role"})
		return
	}

	// Log audit trail
	// h.logAudit(c, "role_updated", nil, nil, &role.ID, nil, map[string]interface{}{
	// 	"role_name":   role.Name,
	// 	"description": role.Description,
	// }, nil)

	c.JSON(http.StatusOK, role)
}

// DeleteRole godoc
// @Summary Delete role
// @Description Delete a role
// @Tags Roles
// @Produce json
// @Param id path int true "Role ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /roles/{id} [delete]
// @Security BearerAuth
func (h *UserHandler) DeleteRole(c *gin.Context) {
	roleID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}

	claims, _ := c.Get("claims")
	tenantID := int32(claims.(*jwt.Claims).TenantID)

	// Check permissions
	// if !h.HasPermission(c, "role:delete", nil, nil) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	// Get role info for audit before deletion
	_, _ = h.service.GetTenantRoleByID(c.Request.Context(), int32(roleID), tenantID)

	err = h.service.DeleteTenantRole(c.Request.Context(), int32(roleID), tenantID)
	if err != nil {
		h.logger.Error("Failed to delete role", "error", err, "role_id", roleID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete role"})
		return
	}

	// Log audit trail
	// rid := int32(roleID)
	// h.logAudit(c, "role_deleted", nil, nil, &rid, nil, nil, map[string]interface{}{
	// 	"role_name": role.Name,
	// 	"tenant_id": role.TenantID,
	// })

	c.Status(http.StatusNoContent)
}

// ListRoles godoc
// @Summary List roles
// @Description List all roles for the current tenant
// @Tags Roles
// @Produce json
// @Success 200 {array} RoleResponse
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /roles [get]
// @Security BearerAuth
func (h *UserHandler) ListRoles(c *gin.Context) {
	claims, _ := c.Get("claims")
	tenantID := int32(claims.(*jwt.Claims).TenantID)

	// Check permissions
	// if !h.HasPermission(c, "role:read", nil, nil) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	roles, err := h.service.ListTenantRoles(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error("Failed to list roles", "error", err, "tenant_id", tenantID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list roles"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"roles": roles,
		"count": len(roles),
	})
}

// ========================================
// USER ROLE ASSIGNMENT HANDLERS
// ========================================

// AssignRoleToUser godoc
// @Summary Assign role to user
// @Description Assign a role to a user
// @Tags Users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param request body AssignRoleRequest true "Role assignment request"
// @Success 201 {object} UserRoleAssignmentResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api/v1/users/{id}/roles [post]
// @Security BearerAuth
func (h *UserHandler) AssignRoleToUser(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	var req AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	// Check permissions
	// if !h.HasPermission(c, "user:assign_roles", req.BusinessID, req.BranchID) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	// Get current user ID for assignment tracking
	// if actorID := h.getCurrentUserID(c); actorID != 0 {
	// 	req.AssignedBy = &actorID
	// }

	params := db.AssignRoleToUserParams{
		UserID:       int32(userID),
		TenantRoleID: req.TenantRoleID,
		BusinessID:   sql.NullInt32{Int32: req.BusinessID, Valid: req.BusinessID != 0},
		BranchID:     sql.NullInt32{Int32: req.BranchID, Valid: req.BranchID != 0},
		AssignedBy:   sql.NullInt32{Int32: req.AssignedBy, Valid: req.AssignedBy != 0},
		ExpiresAt:    sql.NullTime{Time: req.ExpiresAt, Valid: !req.ExpiresAt.IsZero()},
		Metadata:     utils.MarshalMetadata(req.Metadata),
	}

	assignment, err := h.service.AssignRoleToUser(c.Request.Context(), params)
	if err != nil {
		h.logger.Error("Failed to assign role to user", "error: ", err, "user_id: ", userID, "role_id: ", req.TenantRoleID)
		api.ErrorResponse(c, 500, err.Error())
		return
	}

	// uid := int32(userID)

	// Log audit trail
	// h.logAudit(c, "role_assigned", nil, &uid, &req.TenantRoleID, nil, map[string]interface{}{
	// 	"role_id":     req.TenantRoleID,
	// 	"business_id": req.BusinessID,
	// 	"branch_id":   req.BranchID,
	// }, nil)

	api.SuccessResponse(c, 201, "role assigned", assignment)
}

// RemoveRoleFromUser godoc
// @Summary Remove role from user
// @Description Remove a role assignment from a user
// @Tags Users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param role_id path int true "Role ID"
// @Param business_id query int false "Business ID scope"
// @Param branch_id query int false "Branch ID scope"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /users/{id}/roles/{role_id} [delete]
// @Security BearerAuth
func (h *UserHandler) RemoveRoleFromUser(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.logger.Errorf("userid conversion error: %s", err.Error())
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	roleID, err := strconv.Atoi(c.Param("role_id"))
	if err != nil {
		h.logger.Errorf("roleid conversion error: %s", err.Error())
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	business_id, err := strconv.Atoi(c.DefaultQuery("business_id", "0"))
	if err != nil {
		h.logger.Errorf("businessid conversion error: %s", err.Error())
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	branch_id, err := strconv.Atoi(c.DefaultQuery("branch_id", "0"))
	if err != nil {
		h.logger.Errorf("branchid conversion error: %s", err.Error())
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	req := db.RemoveRoleFromUserParams{
		UserID:       int32(userID),
		TenantRoleID: int32(roleID),
		BusinessID:   sql.NullInt32{Int32: int32(business_id), Valid: business_id != 0},
		BranchID:     sql.NullInt32{Int32: int32(branch_id), Valid: branch_id != 0},
	}

	// Check permissions
	// if !h.HasPermission(c, "user:assign_roles", req.BusinessID, req.BranchID) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	err = h.service.RemoveRoleFromUser(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Failed to remove role from user", "error", err, "user_id", userID, "role_id", roleID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove role"})
		return
	}

	// uid := int32(userID)

	// // Log audit trail
	// h.logAudit(c, "role_removed", nil, &uid, &req.TenantRoleID, nil, nil, map[string]interface{}{
	// 	"role_id":     req.TenantRoleID,
	// 	"business_id": req.BusinessID,
	// 	"branch_id":   req.BranchID,
	// })

	api.SuccessResponse(c, 200, "role removed from user", nil)
}

// GetUserRoles godoc
// @Summary Get user roles
// @Description Get all role assignments for a user
// @Tags Users
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {array} UserRoleAssignmentResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /users/{id}/roles [get]
// @Security BearerAuth
func (h *UserHandler) GetUserRoles(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	// Check permissions
	// if !h.HasPermission(c, "user:read", nil, nil) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	assignments, err := h.service.GetUserRoleAssignments(c.Request.Context(), int32(userID))
	if err != nil {
		h.logger.Error("Failed to get user roles", "error", err, "user_id", userID)
		api.ErrorResponse(c, 500, err.Error())
		return
	}

	api.SuccessResponse(c, 200, "", gin.H{
		"assignments": assignments,
		"count":       len(assignments),
	})
}

// GetUserPermissions godoc
// @Summary Get user permissions
// @Description Get all effective permissions for a user (including inherited from roles)
// @Tags Users
// @Produce json
// @Param id path int true "User ID"
// @Param hierarchy query bool false "Include role hierarchy (default: true)"
// @Success 200 {array} UserPermissionResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /users/{id}/permissions [get]
// @Security BearerAuth
func (h *UserHandler) GetUserPermissions(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	tenant, ok := key.GetTenantFromApikey(c)
	if !ok {
		api.ErrorResponse(c, 400, "error getting tenant id from apikey")
	}

	// Check permissions
	// if !h.HasPermission(c, "user:read", nil, nil) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	// Parse hierarchy parameter
	includeHierarchy := c.DefaultQuery("hierarchy", "true") == "true"

	var permissions []*UserPermissionResponse
	if includeHierarchy {
		permissions, err = h.service.GetUserPermissionsWithHierarchy(c.Request.Context(), int32(userID), tenant.TenantID)
	} else {
		permissions, err = h.service.GetUserPermissions(c.Request.Context(), int32(userID), tenant.TenantID)
	}

	if err != nil {
		h.logger.Error("Failed to get user permissions", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user permissions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"permissions": permissions,
		"count":       len(permissions),
	})
}

// GetRolesHierarchy godoc
// @Summary Get roles hierarchy
// @Description Get hierarchical view of all roles
// @Tags Roles
// @Produce json
// @Success 200 {array} RoleHierarchyResponse
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /roles/hierarchy [get]
// @Security BearerAuth
func (h *UserHandler) GetRolesHierarchy(c *gin.Context) {
	claims, _ := c.Get("claims")
	tenantID := int32(claims.(*jwt.Claims).TenantID)

	// Check permissions
	// if !h.HasPermission(c, "role:read", nil, nil) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	roles, err := h.service.ListTenantRolesHierarchy(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Error("Failed to get roles hierarchy", "error", err, "tenant_id", tenantID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get roles hierarchy"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"roles": roles,
		"count": len(roles),
	})
}

// GetRoleHierarchy godoc
// @Summary Get role hierarchy
// @Description Get hierarchy for a specific role
// @Tags Roles
// @Produce json
// @Param id path int true "Role ID"
// @Success 200 {array} RoleHierarchyResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /roles/{id}/hierarchy [get]
// @Security BearerAuth
func (h *UserHandler) GetRoleHierarchy(c *gin.Context) {
	roleID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}

	claims, _ := c.Get("claims")
	tenantID := int32(claims.(*jwt.Claims).TenantID)

	// Check permissions
	// if !h.HasPermission(c, "role:read", nil, nil) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	roles, err := h.service.GetRoleHierarchy(c.Request.Context(), int32(roleID), tenantID)
	if err != nil {
		h.logger.Error("Failed to get role hierarchy", "error", err, "role_id", roleID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get role hierarchy"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"roles": roles,
		"count": len(roles),
	})
}

// GetChildRoles godoc
// @Summary Get child roles
// @Description Get all child roles for a specific role
// @Tags Roles
// @Produce json
// @Param id path int true "Role ID"
// @Success 200 {array} RoleHierarchyResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /roles/{id}/children [get]
// @Security BearerAuth
func (h *UserHandler) GetChildRoles(c *gin.Context) {
	roleID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}

	claims, _ := c.Get("claims")
	tenantID := int32(claims.(*jwt.Claims).TenantID)

	// Check permissions
	// if !h.HasPermission(c, "role:read", nil, nil) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	roles, err := h.service.GetChildRoles(c.Request.Context(), int32(roleID), tenantID)
	if err != nil {
		h.logger.Error("Failed to get child roles", "error", err, "role_id", roleID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get child roles"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"roles": roles,
		"count": len(roles),
	})
}

// GetRoleUsers godoc
// @Summary Get role users
// @Description Get all users assigned to a specific role
// @Tags Roles
// @Produce json
// @Param id path int true "Role ID"
// @Success 200 {array} UserRoleAssignmentResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /roles/{id}/users [get]
// @Security BearerAuth
func (h *UserHandler) GetRoleUsers(c *gin.Context) {
	roleID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}

	// Check permissions
	// if !h.HasPermission(c, "role:read", nil, nil) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	users, err := h.service.GetRoleUsers(c.Request.Context(), int32(roleID))
	if err != nil {
		h.logger.Error("Failed to get role users", "error", err, "role_id", roleID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get role users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"count": len(users),
	})
}

// ========================================
// PERMISSION MANAGEMENT HANDLERS
// ========================================

// CreatePermission godoc
// @Summary Create a new permission
// @Description Create a new permission
// @Tags Permissions
// @Accept json
// @Produce json
// @Param request body CreatePermissionRequest true "Permission creation request"
// @Success 201 {object} PermissionResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /permissions [post]
// @Security BearerAuth
func (h *UserHandler) CreatePermission(c *gin.Context) {
	var req CreatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Check permissions - only system admins can create permissions
	// if !h.HasPermission(c, "tenant:manage", nil, nil) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	params := db.CreatePermissionParams{
		Code:        req.Code,
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
		Module:      req.Module,
		Action:      req.Action,
		Resource:    sql.NullString{String: req.Resource, Valid: req.Resource != ""},
		Scope:       req.Scope,
	}

	permission, err := h.service.CreatePermission(c.Request.Context(), params)
	if err != nil {
		h.logger.Error("Failed to create permission", "error", err)
		api.ErrorResponse(c, 500, err.Error())
		return
	}

	api.SuccessResponse(c, 201, "permission created", permission)
}

// GetPermission godoc
// @Summary Get permission by ID
// @Description Get permission information by ID
// @Tags Permissions
// @Produce json
// @Param id path int true "Permission ID"
// @Success 200 {object} PermissionResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /permissions/{id} [get]
// @Security BearerAuth
func (h *UserHandler) GetPermission(c *gin.Context) {
	permissionID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	// Check permissions
	// if !h.HasPermission(c, "role:read", nil, nil) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	permission, err := h.service.GetPermissionByID(c.Request.Context(), int32(permissionID))
	if err != nil {
		h.logger.Error("Failed to get permission", "error", err, "permission_id", permissionID)
		api.ErrorResponse(c, 404, err.Error())
		return
	}

	api.SuccessResponse(c, 200, "", permission)
}

// UpdatePermission godoc
// @Summary Update permission
// @Description Update permission information
// @Tags Permissions
// @Accept json
// @Produce json
// @Param id path int true "Permission ID"
// @Param request body UpdatePermissionRequest true "Permission update request"
// @Success 200 {object} PermissionResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /permissions/{id} [put]
// @Security BearerAuth
func (h *UserHandler) UpdatePermission(c *gin.Context) {
	permissionID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid permission ID"})
		return
	}

	var req UpdatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	req.ID = int32(permissionID)

	// Check permissions - only system admins can update permissions
	// if !h.HasPermission(c, "tenant:manage", nil, nil) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	permission, err := h.service.UpdatePermission(c.Request.Context(), &req)
	if err != nil {
		if err.Error() == "permission not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Permission not found"})
			return
		}
		h.logger.Error("Failed to update permission", "error", err, "permission_id", permissionID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update permission"})
		return
	}

	c.JSON(http.StatusOK, permission)
}

// ListPermissions godoc
// @Summary List permissions
// @Description List all permissions with optional module filter
// @Tags Permissions
// @Produce json
// @Param module query string false "Module filter"
// @Success 200 {array} PermissionResponse
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /permissions [get]
// @Security BearerAuth
func (h *UserHandler) ListPermissions(c *gin.Context) {
	// Check permissions
	// if !h.HasPermission(c, "role:read", nil, nil) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	module := c.Query("module")
	var permissions []*PermissionResponse
	var err error

	if module != "" {
		permissions, err = h.service.ListPermissionsByModule(c.Request.Context(), module)
	} else {
		permissions, err = h.service.ListPermissions(c.Request.Context())
	}

	if err != nil {
		h.logger.Error("Failed to list permissions", "error", err, "module", module)
		api.ErrorResponse(c, 500, err.Error())
		return
	}

	api.SuccessResponse(c, http.StatusOK, "", gin.H{
		"permissions": permissions,
		"count":       len(permissions),
	})
}

// CheckPermission godoc
// @Summary Check user permission
// @Description Check if a user has a specific permission
// @Tags Permissions
// @Accept json
// @Produce json
// @Param request body CheckUserPermissionRequest true "Permission check request"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /permissions/check [post]
// @Security BearerAuth
func (h *UserHandler) CheckPermission(c *gin.Context) {
	var req CheckUserPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	t, ok := key.GetTenantFromApikey(c)
	if !ok {
		api.ErrorResponse(c, 400, "error getting teneant details from apikey")
	}

	params := db.CheckUserPermissionParams{
		UserID:   req.UserID,
		TenantID: t.TenantID,
		Code:     req.Code,
		ScopeID:  sql.NullInt32{Int32: req.ScopeID, Valid: req.ScopeID != 0},
		BranchID: sql.NullInt32{Int32: req.BranchID, Valid: req.BranchID != 0},
	}

	HasPermission, err := h.service.CheckUserPermission(c.Request.Context(), params)
	if err != nil {
		h.logger.Error("Failed to check user permission", "error", err, "user_id", req.UserID, "permission", req.Code)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check permission"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"has_permission": HasPermission,
		"user_id":        req.UserID,
		"code":           req.Code,
	})
}

// ========================================
// ROLE PERMISSION HANDLERS
// ========================================

// GrantPermissionToRole godoc
// @Summary Grant permission to role
// @Description Grant a permission to a role
// @Tags Roles
// @Accept json
// @Produce json
// @Param id path int true "Role ID"
// @Param request body GrantPermissionRequest true "Permission grant request"
// @Success 201 {object} RolePermissionResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /roles/{id}/permissions [post]
// @Security BearerAuth
func (h *UserHandler) GrantPermissionToRole(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	var req GrantPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Check permissions
	// if !h.HasPermission(c, "role:assign_permissions", nil, nil) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	// Get current user ID for assignment tracking
	// if actorID := h.getCurrentUserID(c); actorID != 0 {
	// 	req.GrantedBy = &actorID
	// }

	params := db.GrantPermissionToRoleParams{
		TenantRoleID: int32(roleID),
		PermissionID: req.PermissionID,
		ScopeType:    sql.NullString{String: req.ScopeType, Valid: req.ScopeType != ""},
		ScopeID:      sql.NullInt32{Int32: req.ScopeID, Valid: req.ScopeID != 0},
		GrantedBy:    sql.NullInt32{Int32: req.GrantedBy, Valid: req.GrantedBy != 0},
		Metadata:     utils.MarshalMetadata(req.Metadata),
	}

	rolePermission, err := h.service.GrantPermissionToRole(c.Request.Context(), params)
	if err != nil {
		h.logger.Error("Failed to grant permission to role", "error", err, "role_id", roleID, "permission_id", req.PermissionID)
		api.ErrorResponse(c, 500, err.Error())
		return
	}

	// Log audit trail
	// h.logAudit(c, "permission_granted", nil, nil, &req.TenantRoleID, &req.PermissionID, map[string]interface{}{
	// 	"scope_type": req.ScopeType,
	// 	"scope_id":   req.ScopeID,
	// }, nil)

	api.SuccessResponse(c, 201, "permission granted", rolePermission)
}

// RevokePermissionFromRole [N] godoc
// @Summary Revoke permission from role
// @Description Revoke a permission from a role
// @Tags Roles
// @Accept json
// @Produce json
// @Param id path int true "Role ID"
// @Param permission_id path int true "Permission ID"
// @Param scope_type query string true "Scope type"
// @Param scope_id query int false "Scope ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /roles/{id}/permissions/{permission_id} [delete]
// @Security BearerAuth
func (h *UserHandler) RevokePermissionFromRole(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	permissionID, err := strconv.Atoi(c.Param("permission_id"))
	if err != nil {
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	scopeType := c.DefaultQuery("scope_type", "0")
	scopeID, _ := strconv.Atoi(c.DefaultQuery("scope_id", "0"))

	req := db.RevokePermissionFromRoleParams{
		TenantRoleID: int32(roleID),
		PermissionID: int32(permissionID),
		ScopeType:    sql.NullString{String: scopeType, Valid: scopeType != ""},
		ScopeID:      sql.NullInt32{Int32: int32(scopeID), Valid: scopeID != 0},
	}

	// Check permissions
	// if !h.HasPermission(c, "role:assign_permissions", nil, nil) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	err = h.service.RevokePermissionFromRole(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Failed to revoke permission from role", "error", err, "role_id", roleID, "permission_id", permissionID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke permission"})
		return
	}

	// Log audit trail
	// h.logAudit(c, "permission_revoked", nil, nil, &req.TenantRoleID, &req.PermissionID, nil, map[string]interface{}{
	// 	"scope_type": req.ScopeType,
	// 	"scope_id":   req.ScopeID,
	// })

	api.SuccessResponse(c, 200, "permission revoked from role", nil)
}

// GetRolePermissions godoc
// @Summary Get role permissions
// @Description Get all permissions assigned to a role
// @Tags Roles
// @Produce json
// @Param id path int true "Role ID"
// @Success 200 {array} RolePermissionResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /roles/{id}/permissions [get]
// @Security BearerAuth
func (h *UserHandler) GetRolePermissions(c *gin.Context) {
	roleID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}

	// Check permissions
	// if !h.HasPermission(c, "role:read", nil, nil) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	permissions, err := h.service.GetRolePermissions(c.Request.Context(), int32(roleID))
	if err != nil {
		h.logger.Error("Failed to get role permissions", "error", err, "role_id", roleID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get role permissions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"permissions": permissions,
		"count":       len(permissions),
	})
}

// ========================================
// AUDIT LOG HANDLERS
// ========================================

// GetAuditLogs [N] godoc
// @Summary Get audit logs
// @Description Get audit logs for the current tenant
// @Tags Audit
// @Produce json
// @Param limit query int false "Limit (default 50)"
// @Param offset query int false "Offset (default 0)"
// @Success 200 {array} AuditLogResponse
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /audit [get]
// @Security BearerAuth
func (h *UserHandler) GetAuditLogs(c *gin.Context) {
	claims, _ := c.Get("claims")
	tenantID := int32(claims.(*jwt.Claims).TenantID)

	// Check permissions
	// if !h.HasPermission(c, "logs:read", nil, nil) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	limit := h.parseInt32WithDefault(c.Query("limit"), 50)
	offset := h.parseInt32WithDefault(c.Query("offset"), 0)

	logs, err := h.service.GetAuditLogs(c.Request.Context(), tenantID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get audit logs", "error", err, "tenant_id", tenantID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get audit logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":   logs,
		"count":  len(logs),
		"limit":  limit,
		"offset": offset,
	})
}

// GetUserAuditLogs godoc
// @Summary Get user audit logs
// @Description Get audit logs for a specific user
// @Tags Audit
// @Produce json
// @Param id path int true "User ID"
// @Param limit query int false "Limit (default 50)"
// @Param offset query int false "Offset (default 0)"
// @Success 200 {array} AuditLogResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /audit/users/{id} [get]
// @Security BearerAuth
func (h *UserHandler) GetUserAuditLogs(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	claims, _ := c.Get("claims")
	tenantID := int32(claims.(*jwt.Claims).TenantID)

	// Check permissions
	// if !h.HasPermission(c, "logs:read", nil, nil) {
	// 	c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	// 	return
	// }

	limit := h.parseInt32WithDefault(c.Query("limit"), 50)
	offset := h.parseInt32WithDefault(c.Query("offset"), 0)

	logs, err := h.service.GetUserAuditLogs(c.Request.Context(), int32(userID), tenantID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get user audit logs", "error", err, "user_id", userID, "tenant_id", tenantID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user audit logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":   logs,
		"count":  len(logs),
		"limit":  limit,
		"offset": offset,
	})
}

// ========================================
// HELPER FUNCTIONS
// ========================================

func (h *UserHandler) LogAudit(c *gin.Context, user *jwt.Claims, action string, targetUserID int32, targetRoleID int32, permissionID int32, newValues, oldValues map[string]any) {

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
	go func() {
		_, err := h.service.CreateAuditLog(c.Request.Context(), req)
		if err != nil {
			h.logger.Error("Failed to create audit log", "error", err, "action", action)
			return
		}
	}()
}

func (h *UserHandler) parseOptionalInt32(value string) *int32 {
	if value == "" {
		return nil
	}
	if intVal, err := strconv.Atoi(value); err == nil {
		result := int32(intVal)
		return &result
	}
	return nil
}

func (h *UserHandler) parseInt32WithDefault(value string, defaultValue int32) int32 {
	if value == "" {
		return defaultValue
	}
	if intVal, err := strconv.Atoi(value); err == nil {
		return int32(intVal)
	}
	return defaultValue
}
