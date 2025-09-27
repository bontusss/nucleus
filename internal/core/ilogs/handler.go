package logs

import (
	"herp/internal/auth"
	"herp/internal/utils"
	"herp/pkg/monitoring/logging"
	"time"

	"github.com/gin-gonic/gin"
)

type LogsHandler struct {
	service LogsInterface
	logger  *logging.Logger
}

func NewLogsHandler(service LogsInterface, logger *logging.Logger) *LogsHandler {
	return &LogsHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers the logs-related routes to the given router group.
func (h *LogsHandler) RegisterRoutes(rg *gin.RouterGroup, authSvc *auth.Service) {
	logs := rg.Group("/logs")
	logs.Use(auth.AdminMiddleware(authSvc))
	{
		logs.GET("/", auth.PermissionMiddleware(authSvc, "logs:activity_logs"), h.GetActivityLogs)
	}
}

type LogsResponse struct {
	ID         int32     `json:"id"`
	UserID     int32     `json:"user_id"`
	Action     string    `json:"action"`
	Details    string    `json:"details"`
	EntityID   int32     `json:"entity_id"`
	EntityType string    `json:"entity_type"`
	IpAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	CreatedAt  time.Time `json:"created_at"`
}

// GetLogs godoc
// @Summary Fetches 100 system logs
// @Description Gets 100 system log entries.
// @Tags Logs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 201 {object} LogsResponse
// @Failure 401
// @Failure 403
// @Failure 500
// @Router /logs [get]
func (h *LogsHandler) GetActivityLogs(c *gin.Context) {
	logs, err := h.service.GetActivityLogs(c, 100)
	if err != nil {
		h.logger.Error("Failed to fetch logs: ", err)
		utils.ErrorResponse(c, 500, "Failed to fetch logs")
		return
	}

	var logsResponse []LogsResponse
	for _, log := range logs {
		logsResponse = append(logsResponse, LogsResponse{
			ID:         log.ID,
			UserID:     log.UserID,
			Action:     log.Action,
			Details:    log.Details,
			EntityID:   log.EntityID,
			EntityType: log.EntityType,
			IpAddress:  log.IpAddress.String,
			UserAgent:  log.UserAgent.String,
			CreatedAt:  log.CreatedAt.Time,
		})
	}

	utils.SuccessResponse(c, 200, "Logs fetched successfully", logsResponse)
}
