package utils

import (
	"os"

	"github.com/gin-gonic/gin"
)
const(
	INVALID_REQUEST_DATA = "invalid request data"
	SERVERERROR = "an error ocurred, try again"
)
// Response structure for both success and error responses
type APIResponse struct {
	Version string `json:"version"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func getVersion() string {
    version := os.Getenv("API_VERSION")
    if version == "" {
        return "1.0.0" // Fallback version if not set
    }
    return version
}

// SuccessResponse sends a success response with a status code, message, and optional data
func SuccessResponse(c *gin.Context, statusCode int, message string, data any) {
	c.JSON(statusCode, APIResponse{
		Version: getVersion(),
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

// ErrorResponse sends an error response with a status code and error message
func ErrorResponse(c *gin.Context, statusCode int, errorMsg string) {
	c.JSON(statusCode, APIResponse{
		Version: getVersion(),
		Status:  "error",
		Error:   errorMsg,
	})
}

