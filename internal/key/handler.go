package key

import (
	"net/http"
	db "nucleus/db/sqlc"
	"nucleus/internal/auth"
	"nucleus/internal/config"
	"nucleus/internal/core/api"
	"nucleus/pkg/jwt"
	"nucleus/pkg/monitoring/logging"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
	logger  *logging.Logger
	config  *config.Config
}

func NewHandler(service *Service, logger *logging.Logger, config *config.Config) *Handler {
	return &Handler{service: service, logger: logger, config: config}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authSvc *auth.Service) {
	key := r.Group("/apikey")
	key.Use(auth.DeveloperMiddleware(authSvc))
	key.POST("", h.CreateAPIKey)
}

type CreateAPIKeyRequest struct {
	KeyName        string    `json:"key_name" binding:"required"`
	AllowedModules []string  `json:"allowed_modules" binding:"required"`
	MonthlyLimit   int       `json:"monthly_limit" binding:"omitempty"`
	ExpiresAt      time.Time `json:"expiresAt"`
}

type CreateAPIKeyResponse struct {
	APIKey db.ApiKey `json:"apiKey"`
	Secret string    `json:"secret"`
}

type APIKeyUsageResponse struct {
	APIKey        db.ApiKey     `json:"apiKey"`
	MonthlyUsage  *MonthlyUsage `json:"monthlyUsage"`
	TotalRequests int           `json:"totalRequests"`
	Remaining     int           `json:"remainingRequests"`
}

func (h *Handler) CreateAPIKey(c *gin.Context) {
	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (from JWT auth)
	claims, ok := jwt.GetUserFromContext(c)
	if !ok {
		h.logger.Errorf("could not get user from context")
		api.ErrorResponse(c, 500, api.SERVERERROR)
		return
	}
	userID := claims.TenantID

	// TODO: Check for usage, if inactive for more than 30 days,
	if req.ExpiresAt.IsZero() {
		req.ExpiresAt = time.Now().AddDate(1, 0, 0)
	}

	// Set default monthly limit if not provided
	limit, err := strconv.Atoi(h.config.APIKEY_MONTHLY_REQUEST_COUNT)
	if err != nil {
		h.logger.Errorf("could not parse monthly limit from config: %v", err)
		api.ErrorResponse(c, 500, err.Error())
		return
	}
	if req.MonthlyLimit == 0 {
		req.MonthlyLimit = limit
	}

	// Generate API key
	result, err := h.service.GenerateAPIKey(c.Request.Context(), GenerateAPIKeyParams{
		KeyName:        req.KeyName,
		TenantID:         userID,
		AllowedModules: req.AllowedModules,
		MonthlyLimit:   req.MonthlyLimit,
		ExpiresAt:      req.ExpiresAt,
	})
	if err != nil {
		api.ErrorResponse(c, 500, err.Error())
		return
	}

	api.SuccessResponse(c, 201, "api key created", CreateAPIKeyResponse{
		APIKey: result.APIKey,
		Secret: result.Secret,
	})
}

func (h *Handler) ListAPIKeys(c *gin.Context) {
	claims, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID := claims.(*jwt.Claims).UserID

	keys, err := h.service.queries.GetAPIKeysByTenant(c.Request.Context(), int32(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, keys)
}

func (h *Handler) GetAPIKeyUsage(c *gin.Context) {
	apiKeyID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid API key ID"})
		return
	}

	// Verify ownership
	claims, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID := claims.(*jwt.Claims).UserID

	apiKey, err := h.service.queries.GetAPIKeyByID(c.Request.Context(), int32(apiKeyID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	if apiKey.TenantID != int32(userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	monthlyUsage, err := h.service.GetMonthlyUsage(c.Request.Context(), apiKeyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := APIKeyUsageResponse{
		APIKey:        apiKey,
		MonthlyUsage:  monthlyUsage,
		TotalRequests: int(monthlyUsage.TotalRequests),
		Remaining:     int(apiKey.MonthlyLimit) - int(monthlyUsage.TotalRequests),
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) UpdateAPIKey(c *gin.Context) {
	apiKeyID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid API key ID"})
		return
	}

	var req struct {
		KeyName        *string    `json:"keyName"`
		AllowedModules []string   `json:"allowedModules"`
		MonthlyLimit   *int       `json:"monthlyLimit"`
		ExpiresAt      *time.Time `json:"expiresAt"`
		IsActive       *bool      `json:"isActive"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify ownership
	claims, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID := claims.(*jwt.Claims).UserID

	apiKey, err := h.service.queries.GetAPIKeyByID(c.Request.Context(), int32(apiKeyID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	if apiKey.TenantID != int32(userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}
	updateParams := db.UpdateAPIKeyParams{ID: int32(apiKeyID)}
	if req.KeyName != nil {
		updateParams.KeyName = *req.KeyName
	} else {
		updateParams.KeyName = apiKey.KeyName
	}

	if req.AllowedModules != nil {
		updateParams.AllowedModules = req.AllowedModules
	} else {
		updateParams.AllowedModules = apiKey.AllowedModules
	}

	if req.MonthlyLimit != nil {
		updateParams.MonthlyLimit = int32(*req.MonthlyLimit)
	} else {
		updateParams.MonthlyLimit = apiKey.MonthlyLimit
	}

	if req.ExpiresAt != nil {
		updateParams.ExpiresAt = *req.ExpiresAt
	} else {
		updateParams.ExpiresAt = apiKey.ExpiresAt
	}

	if req.IsActive != nil {
		updateParams.IsActive = *req.IsActive
	} else {
		updateParams.IsActive = apiKey.IsActive
	}

	updatedKey, err := h.service.queries.UpdateAPIKey(c.Request.Context(), updateParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedKey)
}

func (h *Handler) DeleteAPIKey(c *gin.Context) {
	apiKeyID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid API key ID"})
		return
	}

	// Verify ownership
	claims, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID := claims.(*jwt.Claims).UserID

	apiKey, err := h.service.queries.GetAPIKeyByID(c.Request.Context(), int32(apiKeyID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}
	if apiKey.TenantID != int32(userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.service.queries.DeleteAPIKey(c.Request.Context(), int32(apiKeyID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
