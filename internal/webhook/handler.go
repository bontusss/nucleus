package webhook

import (
	"net/http"
	db "nucleus/db/sqlc"
	"nucleus/internal/core/api"
	"nucleus/internal/key"
	"nucleus/pkg/jwt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, apiKeySvc *key.Service) {
	webhooks := r.Group("/webhooks")
	webhooks.Use(key.APIKeyMiddleware(apiKeySvc, "webhook"))
	{
		webhooks.POST("", h.CreateWebhook)
		webhooks.GET("", h.ListWebhooks)
		webhooks.GET("/:id", h.GetWebhook)
		webhooks.PATCH("/:id", h.UpdateWebhook)
		webhooks.POST("/:id/test", h.TestWebhook)
		webhooks.POST("/:id/deliveries", h.GetWebhookDeliveries)
		webhooks.GET("/events/available", h.GetAvailableEvents)
	}
}

type CreateWebhookRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	URL         string   `json:"url" binding:"required,url"`
	Events      []string `json:"events" binding:"required"`
	MaxRetries  int      `json:"max_retries" binding:"omitempty,min=0,max=10"`
	TimeoutMs   int      `json:"timeout_ms" binding:"omitempty,min=1000,max=30000"`
}

type TestWebhookRequest struct {
	EventType string `json:"event_type" binding:"required"`
	TestData  any    `json:"test_data"`
}

func (h *Handler) CreateWebhook(c *gin.Context) {
	var req CreateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	t, exists := key.GetTenantFromApikey(c)
	if !exists {
		api.ErrorResponse(c, 401, "error getting tenant details from apikey")
		return
	}


	// Set defaults
	if req.MaxRetries == 0 {
		req.MaxRetries = 3
	}
	if req.TimeoutMs == 0 {
		req.TimeoutMs = 5000
	}
	webhook, err := h.service.CreateWebhook(c.Request.Context(), CreateWebhookParams{
		Name:        req.Name,
		Description: req.Description,
		URL:         req.URL,
		Events:      req.Events,
		TenantID:    int(t.TenantID),
		MaxRetries:  req.MaxRetries,
		TimeoutMs:   req.TimeoutMs,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, webhook)
}

func (h *Handler) ListWebhooks(c *gin.Context) {
	claims, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID := claims.(*jwt.Claims).UserID

	webhooks, err := h.service.queries.GetWebhooksByUser(c.Request.Context(), int32(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, webhooks)
}

func (h *Handler) GetWebhook(c *gin.Context) {
	webhookID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook ID"})
		return
	}

	// Verify ownership
	t, exists := key.GetTenantFromApikey(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	webhook, err := h.service.queries.GetWebhookByID(c.Request.Context(), int32(webhookID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Webhook not found"})
		return
	}
	if webhook.TenantID != int32(t.TenantID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, webhook)
}

func (h *Handler) UpdateWebhook(c *gin.Context) {
	webhookID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook ID"})
		return
	}

	var req struct {
		Name        *string  `json:"name"`
		Description *string  `json:"description"`
		URL         *string  `json:"url"`
		Events      []string `json:"events"`
		IsActive    *bool    `json:"is_active"`
		MaxRetries  *int     `json:"max_retries"`
		TimeoutMs   *int     `json:"timeout_ms"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Verify ownership
	t, exists := key.GetTenantFromApikey(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	webhook, err := h.service.queries.GetWebhookByID(c.Request.Context(), int32(webhookID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Webhook not found"})
		return
	}

	if webhook.TenantID != int32(t.TenantID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}
	updateParams := db.UpdateWebhookParams{ID: int32(webhookID)}

	if req.Name != nil {
		updateParams.Name = *req.Name
	} else {
		updateParams.Name = webhook.Name
	}

	if req.Description != nil {
		updateParams.Description.String = *req.Description
	} else {
		updateParams.Description = webhook.Description
	}

	if req.URL != nil {
		updateParams.Url = *req.URL
	} else {
		updateParams.Url = webhook.Url
	}

	if req.Events != nil {
		updateParams.Events = req.Events
	} else {
		updateParams.Events = webhook.Events
	}
	if req.IsActive != nil {
		updateParams.IsActive.Bool = *req.IsActive
	} else {
		updateParams.IsActive = webhook.IsActive
	}

	if req.MaxRetries != nil {
		updateParams.MaxRetries.Int32 = int32(*req.MaxRetries)
	} else {
		updateParams.MaxRetries = webhook.MaxRetries
	}

	if req.TimeoutMs != nil {
		updateParams.TimeoutMs.Int32 = int32(*req.TimeoutMs)
	} else {
		updateParams.TimeoutMs = webhook.TimeoutMs
	}

	updatedWebhook, err := h.service.queries.UpdateWebhook(c.Request.Context(), updateParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedWebhook)
}

func (h *Handler) TestWebhook(c *gin.Context) {
	webhookID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook ID"})
		return
	}

	var req TestWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify ownership
	t, exists := key.GetTenantFromApikey(c)
	if !exists {
		api.ErrorResponse(c, 401, "error getting tenant details from apikey")
		return
	}

	webhook, err := h.service.queries.GetWebhookByID(c.Request.Context(), int32(webhookID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Webhook not found"})
		return
	}

	if webhook.TenantID != int32(t.TenantID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Trigger test event
	testEvent := WebhookEvent{
		Type:      req.EventType,
		Data:      req.TestData,
		Timestamp: time.Now(),
		TenantID:    int(t.TenantID),
		Module:    "test",
	}

	// Send test webhook (synchronously for testing)
	go h.service.sendWebhook(c.Request.Context(), webhook, testEvent)

	api.SuccessResponse(c, 200, "test webhook sent", testEvent)
}

func (h *Handler) GetWebhookDeliveries(c *gin.Context) {
	webhookID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook ID"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit > 100 {
		limit = 100
	}

	// Verify ownership
	t, exists := key.GetTenantFromApikey(c)
	if !exists {
		api.ErrorResponse(c, 401, "invalid apikey")
		return
	}

	webhook, err := h.service.queries.GetWebhookByID(c.Request.Context(), int32(webhookID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Webhook not found"})
		return
	}

	if webhook.TenantID != int32(t.TenantID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	deliveries, err := h.service.queries.GetWebhookDeliveries(c.Request.Context(),
		db.GetWebhookDeliveriesParams{
			WebhookID: int32(webhookID),
			Limit:     int32(limit),
		})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, deliveries)
}

func (h *Handler) GetAvailableEvents(c *gin.Context) {
	events, err := h.service.GetAvailableEvents(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, events)
}
