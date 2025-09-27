package logging

import (
	"bytes"
	"encoding/json"
	"herp/internal/config"
	"io"
	"log/syslog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	logrusSyslog "github.com/sirupsen/logrus/hooks/syslog"
)

type Logger struct {
	*logrus.Logger
	config *config.Config
}

type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func NewLogger(c *config.Config) *Logger {
	log := logrus.New()
	// Set log level
	if c.GinMode == "debug" {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}

	// Output only to stdout in debug mode
	if c.GinMode == "debug" {
		log.SetOutput(gin.DefaultWriter)
	} else {
		// discard stdout when not in debug
		log.SetOutput(io.Discard)
	}
	log.SetFormatter(&logrus.JSONFormatter{PrettyPrint: c.GinMode == "debug"})

	hook, err := logrusSyslog.NewSyslogHook("udp", c.PapertrailAddr, syslog.LOG_INFO, c.PapertrailAppName)
	if err == nil {
		log.AddHook(hook)
	} else {
		log.Warn("Failed to connect to Papertrail", err)
	}

	return &Logger{Logger: log, config: c}
}

func (l *Logger) LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		var reqBody []byte
		if c.Request.Body != nil {
			reqBody, _ = c.GetRawData()
			c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		w := &responseBodyWriter{
			body:           &bytes.Buffer{},
			ResponseWriter: c.Writer,
		}
		c.Writer = w

		c.Next()

		// Log after request is processed
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		var requestJson any
		var responseJson any
		err := json.Unmarshal(reqBody, &requestJson)
		if err != nil {
			l.Log(logrus.DebugLevel, "error unmarshalling requestBody, request may not be JSON")
		}

		err = json.Unmarshal(w.body.Bytes(), &responseJson)
		if err != nil {
			l.Log(logrus.DebugLevel, "error unmarshalling responseBody")
		}

		fields := logrus.Fields{
			"method":   c.Request.Method,
			"path":     c.Request.URL.Path,
			"status":   statusCode,
			"duration": duration,
			// "response_body": responseJson,
		}

		// Only log request body if it's small to avoid polluting logs with large payloads
		// that could impact log storage and make debugging more difficult
		if len(reqBody) < 250 {
			fields["request"] = requestJson
		}

		l.WithFields(fields).Info("Request-Response")
	}
}
