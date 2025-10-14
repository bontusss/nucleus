package utils

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sqlc-dev/pqtype"
)

// Get client IP from context
func GetClientIP(c context.Context) string {
	if ginCtx, ok := c.(*gin.Context); ok {
		// Try to get IP from X-Forwarded-For header (if behind proxy)
		if ip := ginCtx.GetHeader("X-Forwarded-For"); ip != "" {
			return strings.Split(ip, ",")[0] // First IP in chain
		}
		// Fall back to remote address
		return ginCtx.ClientIP()
	}
	return "unknown"
}

func WriteActivityDetails(username, email, action string, time time.Time) string {
	return fmt.Sprintf("User %s with email %s performed action: %s at %s", username, email, action, time)
}

// UploadFile validates and saves an uploaded file.
// Returns the relative URL path (e.g. /images/123_logo.png) or an error.
func UploadFile(c *gin.Context, fieldName string, saveDir string, maxSize int64) (string, error) {
	file, err := c.FormFile(fieldName)
	if err != nil {
		// No file provided
		return "", err
	}

	// Check file size
	if file.Size > maxSize {
		return "", fmt.Errorf("file too large, max %d bytes allowed", maxSize)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExt := map[string]bool{".jpg": true, ".jpeg": true, ".png": true}
	if !allowedExt[ext] {
		return "", fmt.Errorf("invalid file extension: only JPG/PNG allowed")
	}

	// Check MIME type
	openedFile, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("could not open uploaded file: %v", err)
	}
	defer openedFile.Close()

	buffer := make([]byte, 512)
	if _, err := openedFile.Read(buffer); err != nil {
		return "", fmt.Errorf("could not read uploaded file: %v", err)
	}

	contentType := http.DetectContentType(buffer)
	allowedMime := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
	}
	if !allowedMime[contentType] {
		return "", fmt.Errorf("invalid file type: only JPG/PNG allowed")
	}

	// Ensure save directory exists
	if _, statErr := os.Stat(saveDir); os.IsNotExist(statErr) {
		os.MkdirAll(saveDir, os.ModePerm)
	}

	// Generate unique filename
	filename := fmt.Sprintf("%d_%s", time.Now().Unix(), filepath.Base(file.Filename))
	filePath := filepath.Join(saveDir, filename)

	// Save file
	if saveErr := c.SaveUploadedFile(file, filePath); saveErr != nil {
		return "", fmt.Errorf("could not save file: %v", saveErr)
	}

	// Return relative URL for serving via Gin Static
	return "/" + filePath, nil
}

// ToNullString converts a pointer to a string to a sql.NullString.
func ToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{Valid: true, String: *s}
}

// ToNullInt32 converts a pointer to an int32 to a sql.NullInt32.
func ToNullInt32(i *int32) sql.NullInt32 {
	if i == nil {
		return sql.NullInt32{Valid: false}
	}
	return sql.NullInt32{Valid: true, Int32: *i}
}

// ToNullBool converts a pointer to a boolean to a sql.NullBool.
func ToNullBool(b *bool) sql.NullBool {
	if b == nil {
		return sql.NullBool{Valid: false}
	}
	return sql.NullBool{Valid: true, Bool: *b}
}

// Dereference string pointer or return empty string
func DerefOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// PatchString updates a sql.NullString if the field is present.
func PatchString(dest *sql.NullString, value *string) {
	if value == nil {
		return // not provided → leave unchanged
	}
	if *value == "" {
		*dest = sql.NullString{Valid: false}
	} else {
		*dest = sql.NullString{String: *value, Valid: true}
	}
}

// PatchInt32 updates a sql.NullInt32 if the field is present.
func PatchInt32(dest *sql.NullInt32, value *int32) {
	if value == nil {
		return
	}
	if *value == 0 {
		*dest = sql.NullInt32{Valid: false}
	} else {
		*dest = sql.NullInt32{Int32: *value, Valid: true}
	}
}

// PatchBool updates a sql.NullBool if the field is present.
func PatchBool(dest *sql.NullBool, value *bool) {
	if value == nil {
		return
	}
	*dest = sql.NullBool{Bool: *value, Valid: true}
}

func PatchNullString(field *sql.NullString, value *string) {
	if value == nil {
		return // don't change
	}
	if *value == "" {
		field.Valid = false
		field.String = ""
	} else {
		field.Valid = true
		field.String = *value
	}
}

func PatchNullInt32(field *sql.NullInt32, value *int32) {
	if value == nil {
		return
	}
	field.Valid = true
	field.Int32 = *value
}

func PatchNullBool(field *sql.NullBool, value *bool) {
	if value == nil {
		return
	}
	field.Valid = true
	field.Bool = *value
}

// MarshalMetadata converts map[string]any → pqtype.NullRawMessage
func MarshalMetadata(meta map[string]any) pqtype.NullRawMessage {
	if meta == nil {
		return pqtype.NullRawMessage{Valid: false}
	}

	raw, err := json.Marshal(meta)
	if err != nil {
		return pqtype.NullRawMessage{Valid: false}
	}

	return pqtype.NullRawMessage{
		RawMessage: raw,
		Valid:      true,
	}
}

// UnmarshalMetadata converts pqtype.NullRawMessage → map[string]any
func UnmarshalMetadata(raw pqtype.NullRawMessage) map[string]any {
	if !raw.Valid || len(raw.RawMessage) == 0 {
		return nil
	}

	var meta map[string]any
	if err := json.Unmarshal(raw.RawMessage, &meta); err != nil {
		return nil
	}

	return meta
}

// PatchMetadata sets metadata if a new map is provided.
func PatchMetadata(dest *pqtype.NullRawMessage, src map[string]any) {
	if src == nil {
		return
	}

	var existing map[string]any
	if dest.Valid && len(dest.RawMessage) > 0 {
		_ = json.Unmarshal(dest.RawMessage, &existing)
	} else {
		existing = make(map[string]any)
	}

	// merge: overwrite keys from src into existing
	for k, v := range src {
		existing[k] = v
	}

	b, err := json.Marshal(existing)
	if err == nil {
		dest.Valid = true
		dest.RawMessage = b
	}
}

// GetStringOrDefault returns the value if not nil, otherwise returns the default value
func GetStringOrDefault(value *string, defaultValue string) string {
	if value == nil {
		return defaultValue
	}
	return *value
}

// GetBoolOrDefault returns the value if not nil, otherwise returns the default value
func GetBoolOrDefault(value *bool, defaultValue bool) bool {
	if value == nil {
		return defaultValue
	}
	return *value
}

// GetInt32OrDefault returns the value if not nil, otherwise returns the default value
func GetInt32OrDefault(value *int32, defaultValue int32) int32 {
	if value == nil {
		return defaultValue
	}
	return *value
}

// GetTimeOrDefault returns the time value if not nil, otherwise returns current time
func GetTimeOrDefault(value *time.Time) time.Time {
	if value == nil {
		return time.Now()
	}
	return *value
}

// StringPtr returns a pointer to the string value
func StringPtr(s string) *string {
	return &s
}

// BoolPtr returns a pointer to the bool value
func BoolPtr(b bool) *bool {
	return &b
}

// Int32Ptr returns a pointer to the int32 value
func Int32Ptr(i int32) *int32 {
	return &i
}

// TimePtr returns a pointer to the time value
func TimePtr(t time.Time) *time.Time {
	return &t
}

// IsValidEmail performs basic email validation
func IsValidEmail(email string) bool {
	// Basic email validation - can be enhanced with regex
	return len(email) > 0 &&
		len(email) < 255 &&
		containsChar(email, '@') &&
		containsChar(email, '.')
}

// containsChar checks if string contains a specific character
func containsChar(s string, char rune) bool {
	for _, c := range s {
		if c == char {
			return true
		}
	}
	return false
}

// TruncateString truncates a string to a maximum length
func TruncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength]
}

// Contains checks if a slice contains a specific string
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ContainsInt32 checks if a slice contains a specific int32
func ContainsInt32(slice []int32, item int32) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// RemoveDuplicateStrings removes duplicate strings from a slice
func RemoveDuplicateStrings(slice []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}

// RemoveDuplicateInt32 removes duplicate int32s from a slice
func RemoveDuplicateInt32(slice []int32) []int32 {
	keys := make(map[int32]bool)
	var result []int32

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}
