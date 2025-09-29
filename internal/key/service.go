package key

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	db "nucleus/db/sqlc"
	"time"
)

var (
	ErrAPIKeyNotFound       = errors.New("API key not found")
	ErrAPIKeyInactive       = errors.New("API key is inactive")
	ErrAPIKeyExpired        = errors.New("API key has expired")
	ErrMonthlyLimitExceeded = errors.New("monthly request limit exceeded")
	ErrModuleNotAllowed     = errors.New("module not allowed for this API key")
	ErrInvalidSignature     = errors.New("invalid request signature")
)

type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{
		queries: queries,
	}
}

type GenerateAPIKeyParams struct {
	KeyName        string    `json:"keyName"`
	UserID         int       `json:"userId"`
	AllowedModules []string  `json:"allowedModules"`
	MonthlyLimit   int       `json:"monthlyLimit"`
	ExpiresAt      time.Time `json:"expiresAt"`
}

type APIKeyWithSecret struct {
	APIKey db.ApiKey `json:"apiKey"`
	Secret string    `json:"secret"` // Only returned during creation
}

type RecordUsageParams struct {
	APIKeyID       int32  `json:"apiKeyId"`
	ModuleName     string `json:"moduleName"`
	Endpoint       string `json:"endpoint"`
	RequestMethod  string `json:"requestMethod"`
	RequestSize    int32  `json:"requestSize"`
	ResponseStatus int32  `json:"responseStatus"`
	IPAddress      string `json:"ipAddress"`
	UserAgent      string `json:"userAgent"`
}

type MonthlyUsage struct {
	TotalRequests int32     `json:"totalRequests"`
	TotalBytes    int64     `json:"totalBytes"`
	Month         time.Time `json:"month"`
}

// Helper function to generate random strings
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateAPIKey creates a new API key with secret
func (s *Service) GenerateAPIKey(ctx context.Context, params GenerateAPIKeyParams) (*APIKeyWithSecret, error) {
	// Generate API key (public identifier)
	apiKey, err := generateRandomString(32)
	if err != nil {
		return nil, err
	}

	// Generate secret for signing
	secret, err := generateRandomString(32)
	if err != nil {
		return nil, err
	}

	// Create API key record
	key, err := s.queries.CreateAPIKey(ctx, db.CreateAPIKeyParams{
		KeyName:        params.KeyName,
		ApiKey:         apiKey,
		ApiSecret:      secret,
		UserID:         int32(params.UserID),
		AllowedModules: params.AllowedModules,
		MonthlyLimit:   int32(params.MonthlyLimit),
		ExpiresAt:      params.ExpiresAt,
	})
	if err != nil {
		return nil, err
	}

	return &APIKeyWithSecret{
		APIKey: key,
		Secret: secret, // Only returned once during creation
	}, nil
}

// ValidateAPIKey checks if the API key is valid for the requested module
func (s *Service) ValidateAPIKey(ctx context.Context, apiKey, module string) (*db.ApiKey, error) {
	key, err := s.queries.GetAPIKeyByKey(ctx, apiKey)
	if err != nil {
		return nil, ErrAPIKeyNotFound
	}

	if !key.IsActive {
		return nil, ErrAPIKeyInactive
	}

	if key.ExpiresAt.Before(time.Now()) {
		return nil, ErrAPIKeyExpired
	}

	// Check if module is allowed
	moduleAllowed := false
	for _, allowedModule := range key.AllowedModules {
		if allowedModule == module {
			moduleAllowed = true
			break
		}
	}

	if !moduleAllowed {
		return nil, ErrModuleNotAllowed
	}

	// Check monthly limit (reset on 1st of each month)
	now := time.Now()
	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	if key.UpdatedAt.Time.Before(firstOfMonth) {
		// Reset counter for new month
		key.CurrentMonthRequests = 0
		if err := s.queries.ResetMonthlyUsage(ctx, key.ID); err != nil {
			return nil, err
		}
	}

	if key.CurrentMonthRequests >= key.MonthlyLimit {
		return nil, ErrMonthlyLimitExceeded
	}

	return &key, nil
}

// RecordUsage records an API call and increments the usage counter
func (s *Service) RecordUsage(ctx context.Context, params RecordUsageParams) error {
	// Increment usage counter
	if err := s.queries.IncrementAPIKeyUsage(ctx, params.APIKeyID); err != nil {
		return err
	}

	// Record detailed usage
	_, err := s.queries.RecordAPIKeyUsage(ctx, db.RecordAPIKeyUsageParams{
		ApiKeyID:       int32(params.APIKeyID),
		ModuleName:     params.ModuleName,
		Endpoint:       params.Endpoint,
		RequestMethod:  params.RequestMethod,
		RequestSize:    int32(params.RequestSize),
		ResponseStatus: int32(params.ResponseStatus),
		IpAddress:      params.IPAddress,
		UserAgent:      params.UserAgent,
	})
	return err
}

// GetAPIKeyUsage returns usage statistics
func (s *Service) GetAPIKeyUsage(ctx context.Context, apiKeyID int, startDate, endDate time.Time) ([]db.ApiKeyUsage, error) {
	return s.queries.GetAPIKeyUsageByDateRange(ctx, db.GetAPIKeyUsageByDateRangeParams{
		ApiKeyID:    int32(apiKeyID),
		CreatedAt:   sql.NullTime{Time: startDate, Valid: true},
		CreatedAt_2: sql.NullTime{Time: endDate, Valid: true},
	})
}

// GetMonthlyUsage returns current month's usage
func (s *Service) GetMonthlyUsage(ctx context.Context, apiKeyID int) (*MonthlyUsage, error) {
	now := time.Now()
	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	firstOfNextMonth := firstOfMonth.AddDate(0, 1, 0)

	usage, err := s.queries.GetAPIKeyUsageByDateRange(ctx, db.GetAPIKeyUsageByDateRangeParams{
		ApiKeyID:    int32(apiKeyID),
		CreatedAt:   sql.NullTime{Time: firstOfMonth, Valid: true},
		CreatedAt_2: sql.NullTime{Time: firstOfNextMonth, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	var totalRequests int32
	var totalBytes int64
	for _, u := range usage {
		totalRequests++
		totalBytes += int64(u.RequestSize)
	}

	return &MonthlyUsage{
		TotalRequests: totalRequests,
		TotalBytes:    totalBytes,
		Month:         firstOfMonth,
	}, nil
}
