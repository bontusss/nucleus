package middleware

import (
	"encoding/json"
	"fmt"
	"herp/internal/config"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

type logEntry struct {
	Timestamp    string   `json:"timestamp"`
	StatusCode   int      `json:"status"`
	Latency      string   `json:"latency"`
	Method       string   `json:"method"`
	Path         string   `json:"path"`
	BytesWritten int      `json:"bytes_written"`
	ClientIP     string   `json:"ip"`
	UserAgent    string   `json:"user_agent"`
	Errors       []string `json:"errors,omitempty"`
	RequestID    string   `json:"request_id,omitempty"`
}

// NewRequestLogger returns a Gin middleware that logs request/response details
// to both stdout and the specified file. The directory for the log file will be
// created if it does not exist.
func NewRequestLogger(logFilePath string, c *config.Config) gin.HandlerFunc {
	ginMode := c.GinMode
	var writer io.Writer

	if ginMode == "release" {
		// Send logs to papertrail
		papertrailAddr := c.PapertrailAddr
		if papertrailAddr == "" {
			log.Printf("Papertrail address not configured")
		}
		conn, err := net.Dial("udp", papertrailAddr)
		if err != nil {
			log.Printf("failed to connect to Papertrail: %v", err)
		}
		writer = conn
	} else {
		// local logging
		// Local logging: stdout + file
		dir := filepath.Dir(logFilePath)
		if dir != "." && dir != "" {
			_ = os.MkdirAll(dir, 0o755)
		}
		file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			writer = os.Stdout
		} else {
			writer = io.MultiWriter(os.Stdout, file)
		}
	}

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		writeJSONLog(writer, c, start)
	}

}

func writeJSONLog(w io.Writer, c *gin.Context, start time.Time) {
	latency := time.Since(start)
	statusCode := c.Writer.Status()
	clientIP := c.ClientIP()
	method := c.Request.Method
	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}
	userAgent := c.Request.UserAgent()
	bytesWritten := max(c.Writer.Size(), 0)

	// Capture error information
	var errorDetails []string

	// Capture Gin validation errors
	if len(c.Errors) > 0 {
		for _, err := range c.Errors {
			errorDetails = append(errorDetails, err.Error())
		}
	}

	// Capture HTTP status errors (4xx, 5xx)
	if statusCode >= 400 {
		// Try to capture response body for error details
		if c.Writer.Size() > 0 {
			// Note: We can't easily capture response body in middleware
			// but we can log the status code and any error messages
			errorDetails = append(errorDetails, fmt.Sprintf("HTTP %d", statusCode))
		}
	}

	// Capture specific error types based on status codes
	switch {
	case statusCode >= 500:
		errorDetails = append(errorDetails, "server_error")
	case statusCode == 404:
		errorDetails = append(errorDetails, "not_found")
	case statusCode == 401:
		errorDetails = append(errorDetails, "unauthorized")
	case statusCode == 403:
		errorDetails = append(errorDetails, "forbidden")
	case statusCode == 400:
		errorDetails = append(errorDetails, "bad_request")
	case statusCode == 422:
		errorDetails = append(errorDetails, "validation_error")
	}

	entry := logEntry{
		Timestamp:    time.Now().Format(time.RFC3339),
		StatusCode:   statusCode,
		Latency:      latency.String(),
		Method:       method,
		Path:         path,
		BytesWritten: bytesWritten,
		ClientIP:     clientIP,
		UserAgent:    userAgent,
		Errors:       errorDetails,
	}

	if reqID := c.GetHeader("X-Request-ID"); reqID != "" {
		entry.RequestID = reqID
	}

	enc := json.NewEncoder(w)
	_ = enc.Encode(entry)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
