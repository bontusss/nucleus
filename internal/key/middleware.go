package key

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	APIKeyHeader = "X-API-Key"
	LOGIN        = "/api/v1/auth/login"
	REFRESH      = "/api/v1/auth/refresh"
)

// APIKeyMiddleware validates API keys for module access
func APIKeyMiddleware(apiKeyService *Service, module string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip API key check for authentication endpoints
		if c.Request.URL.Path == LOGIN || c.Request.URL.Path == REFRESH {
			c.Next()
			return
		}

		apiKey := c.GetHeader(APIKeyHeader)
		if apiKey == "" {
			// Try to get from query parameter
			apiKey = c.Query("api_key")
		}

		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			c.Abort()
			return
		}

		// Validate API key
		key, err := apiKeyService.ValidateAPIKey(c.Request.Context(), apiKey, module)
		if err != nil {
			var status int
			switch err {
			case ErrAPIKeyNotFound:
				status = http.StatusUnauthorized
			case ErrAPIKeyInactive, ErrAPIKeyExpired:
				status = http.StatusForbidden
			case ErrMonthlyLimitExceeded:
				status = http.StatusTooManyRequests
			case ErrModuleNotAllowed:
				status = http.StatusForbidden
			default:
				status = http.StatusInternalServerError
			}

			c.JSON(status, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		// Store API key info in context
		c.Set("apiKey", key)
		c.Set("apiKeyId", key.ID)

		// Record usage after request completes
		c.Next()

		// Record usage (after response is sent to not block the request)
		go func() {
			// Use background context to avoid cancellation
			ctx := context.Background()

			apiKeyService.RecordUsage(ctx, RecordUsageParams{
				APIKeyID:       key.ID,
				ModuleName:     module,
				Endpoint:       c.Request.URL.Path,
				RequestMethod:  c.Request.Method,
				RequestSize:    int32(c.Request.ContentLength),
				ResponseStatus: int32(c.Writer.Status()),
				IPAddress:      c.ClientIP(),
				UserAgent:      c.Request.UserAgent(),
			})
		}()
	}
}

// HMACValidationMiddleware for WEBHOOKS only
func HMACValidationMiddleware(apiKeyService *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		signature := c.GetHeader("X-Webhook-Signature")
		timestamp := c.GetHeader("X-Webhook-Timestamp")

		if signature == "" || timestamp == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "webhook signature and timestamp required"})
			c.Abort()
			return
		}

		// Validate timestamp
		if !validateTimestamp(timestamp) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid timestamp"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func requiresHMACValidation(method string) bool {
	// Require HMAC for write operations
	writeMethods := []string{"POST", "PUT", "PATCH", "DELETE"}
	for _, m := range writeMethods {
		if method == m {
			return true
		}
	}
	return false
}

func validateTimestamp(timestamp string) bool {
	ts, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return false
	}

	// Allow 5-minute window for clock skew
	now := time.Now()
	fiveMinutesAgo := now.Add(-5 * time.Minute)
	fiveMinutesFromNow := now.Add(5 * time.Minute)

	return ts.After(fiveMinutesAgo) && ts.Before(fiveMinutesFromNow)
}

func getRequestBody(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}

	// Read body into buffer
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		// On read error, return empty string
		return ""
	}

	// Restore body so downstream handlers can still read it
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return string(bodyBytes)
}
