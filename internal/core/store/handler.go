package store

import (
	"database/sql"
	"fmt"
	db "herp/db/sqlc"
	"herp/internal/auth"
	"herp/internal/utils"
	"herp/pkg/jwt"
	"herp/pkg/monitoring/logging"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

type Handler struct {
	service StoreInterface
	logger  *logging.Logger
}

func NewHandler(service StoreInterface, logger *logging.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authSvc *auth.Service) {
	store := r.Group("/store")
	store.Use(auth.AdminMiddleware(authSvc))
	{
		store.POST("/", h.CreateStore)
		store.GET("/:id", h.GetStoreByID)
		store.PUT("/", h.UpdateStore)
		store.DELETE("/:id", h.DeleteStore)
	}
}

type storeParams struct {
	BranchID        int32  `json:"branch_id" binding:"required" example:"1"`
	Description     string `json:"description"`
	Name            string `json:"name" binding:"required" example:"Main Street Store"`
	Address         string `json:"address" binding:"required" example:"123 Main St, Cityville"`
	Phone           string `json:"phone" binding:"required" example:"+1234567890"`
	Email           string `json:"email" binding:"required,email" example:""`
	StoreCode       string `json:"store_code" binding:"required" example:"STR001"`
	IsCentral       bool   `json:"is_central" binding:"omitempty" example:"false"`
	IsActive        bool   `json:"is_active" example:"true"`
	AssignedUser    int32  `json:"assigned_user" example:"1"`
	AssignedManager int32  `json:"assigned_manager" example:"1"`
}

// CreateStore godoc
// @Summary Create a store
// @Tags store
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body storeParams true "store details"
// @Success 201 {object} storeParams
// @Failure 400
// @Failure 401
// @Failure 403
// @Failure 500
// @Router /store [post]
func (h *Handler) CreateStore(c *gin.Context) {
	claims, ok := jwt.GetUserFromContext(c)
	if !ok {
		h.logger.Errorf("could not get user from context")
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	var req storeParams
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("Failed to bind create store request error: %v", err)
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	// TODO: auto generate store names

	storeType := "sub-store"
	if req.IsCentral {
		storeType = "central"
	}

	params := db.CreateStoreParams{
		BranchID:  req.BranchID,
		Name:      req.Name,
		Address:   req.Address,
		Phone:     req.Phone,
		Email:     req.Email,
		StoreType: storeType,
		StoreCode: req.StoreCode,
	}

	store, err := h.service.CreateStore(c.Request.Context(), params)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				if pqErr.Constraint == "unique_central_store_per_branch" {
					utils.ErrorResponse(c, 400, "A branch can only have one central store")
					return
				}
			}
		}

		h.logger.Errorf("Failed to create store error: %v", err)
		c.JSON(500, gin.H{"error": "Failed to create store"})
		return
	}

	// Log activity
	_, err = h.service.LogActivity(c, db.LogActivityParams{
		UserID:     int32(claims.UserID),
		Action:     "Created Store",
		EntityType: "Store",
		EntityID:   store.ID,
		Details:    utils.WriteActivityDetails(claims.Username, claims.Email, fmt.Sprintf("Created store %s", store.Name), store.CreatedAt.Time),
		IpAddress:  sql.NullString{Valid: true, String: utils.GetClientIP(c)},
		UserAgent:  sql.NullString{Valid: true, String: c.Request.UserAgent()},
	})

	if err != nil {
		h.logger.Warnf("error logging activity: %v", err)
		// not returning error to user as business and branch have been created successfully
	}

	utils.SuccessResponse(c, 201, "store created", storeParams{
		Name:            store.Name,
		BranchID:        store.BranchID,
		Address:         store.Address,
		Phone:           store.Phone,
		Email:           store.Email,
		StoreCode:       store.StoreCode,
		IsCentral:       storeType == "central",
		IsActive:        store.IsActive.Bool,
		AssignedUser:    store.AssignedUser.Int32,
		AssignedManager: store.ManagerID.Int32,
	})
}

func (h *Handler) GetStoreByID(c *gin.Context) {
	idParam := c.Param("id")
	var id int32
	_, err := fmt.Sscan(idParam, &id)
	if err != nil {
		h.logger.Errorf("Invalid store ID error: %v", err)
		c.JSON(400, gin.H{"error": "Invalid store ID"})
		return
	}

	store, err := h.service.GetStoreByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Errorf("Failed to get store: %v", err)
		c.JSON(500, gin.H{"error": "Failed to get store"})
		return
	}

	c.JSON(200, store)
}

func (h *Handler) UpdateStore(c *gin.Context) {}

func (h *Handler) DeleteStore(c *gin.Context) {}
