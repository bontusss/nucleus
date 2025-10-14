package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"math/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	db "nucleus/db/sqlc"
	"time"
)

var (
	ErrWebhookNotFound = errors.New("webhook not found")
	ErrWebhookInactive = errors.New("webhook is inactive")
	ErrEventNotAllowed = errors.New("event type not allowed for this webhook")
)

type CreateWebhookParams struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Events      []string `json:"events"`
	TenantID    int      `json:"tenant_id"`
	MaxRetries  int      `json:"max_retries"`
	TimeoutMs   int      `json:"timeout_ms"`
}

type Service struct {
	queries *db.Queries
	client  *http.Client
}

func NewService(queries *db.Queries) *Service {
	return &Service{
		queries: queries,
		client: &http.Client{
			Timeout: 30 * time.Second, // Default timeout
		},
	}
}

// WebhookEvent represents an event that can trigger webhooks
type WebhookEvent struct {
	Type      string    `json:"type"`
	Data      any       `json:"data"`
	Timestamp time.Time `json:"timestamp"`
	TenantID  int       `json:"tenant_id,omitempty"`
	Module    string    `json:"module"`
}

// TriggerEvent sends webhook notifications for an event
func (s *Service) TriggerEvent(ctx context.Context, event WebhookEvent) error {
	// require tenant scoping
	if event.TenantID == 0 {
		return fmt.Errorf("tenant id required for event %q", event.Type)
	}

	// Get active webhooks that listen for this event type
	webhooks, err := s.queries.GetWebhooksByEvent(ctx, db.GetWebhooksByEventParams{
		TenantID: int32(event.TenantID),
		Events:   []string{event.Type},
	})
	if err != nil {
		return err
	}

	// Send webhooks in background goroutines
	for _, webhook := range webhooks {
		if !webhook.IsActive.Bool {
			continue
		}

		go s.sendWebhook(ctx, webhook, event)
	}

	return nil
}

// sendWebhook delivers a webhook to a specific URL
func (s *Service) sendWebhook(ctx context.Context, webhook db.Webhook, event WebhookEvent) {
	payload := map[string]any{
		"event":      event.Type,
		"data":       event.Data,
		"timestamp":  event.Timestamp.Format(time.RFC3339),
		"webhook_id": webhook.ID,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		s.recordDeliveryFailure(ctx, webhook.ID, event.Type, payload, 0, err.Error(), 1)
		return
	}

	// Create request with timeout from webhook settings
	timeout := time.Duration(webhook.TimeoutMs.Int32) * time.Millisecond
	if timeout == 0 {
		timeout = 5 * time.Second // Default timeout
	}

	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, "POST", webhook.Url, bytes.NewReader(payloadBytes))
	if err != nil {
		s.recordDeliveryFailure(ctx, webhook.ID, event.Type, payload, 0, err.Error(), 1)
		return
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Nucleus-Webhooks/1.0")
	req.Header.Set("X-Webhook-Event", event.Type)
	req.Header.Set("X-Webhook-ID", fmt.Sprintf("%d", webhook.ID))

	// Add signature for verification
	timestamp := time.Now().Format(time.RFC3339)
	signature := s.generateSignature(webhook.Secret, payloadBytes, timestamp)
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Timestamp", timestamp)

	// Record delivery attempt
	delivery, err := s.queries.RecordWebhookDelivery(ctx, db.RecordWebhookDeliveryParams{
		WebhookID:     webhook.ID,
		EventType:     event.Type,
		Payload:       payloadBytes,
		AttemptNumber: sql.NullInt32{Int32: 1, Valid: true},
	})
	if err != nil {
		// set error on the original attempt before retry
		s.queries.UpdateWebhookDelivery(ctx, db.UpdateWebhookDeliveryParams{
			ID:           delivery.ID,
			ErrorMessage: sql.NullString{String: err.Error(), Valid: true},
		})
		s.handleRetry(ctx, webhook, event, payload, int64(delivery.ID), err.Error(), 1)
		return
	}
	req.Header.Set("X_Webhook-Delivery-ID", fmt.Sprintf("%d", delivery.ID))
	payload["delivery_id"] = delivery.ID
	payloadBytes, _ = json.Marshal(payload)

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		s.handleRetry(ctx, webhook, event, payload, int64(delivery.ID), err.Error(), 1)
		return
	}
	defer resp.Body.Close()

	// Read response body
	responseBody, _ := io.ReadAll(resp.Body)

	// Update delivery record
	s.queries.UpdateWebhookDelivery(ctx, db.UpdateWebhookDeliveryParams{
		ID:             delivery.ID,
		ResponseStatus: sql.NullInt32{Int32: int32(resp.StatusCode), Valid: true},
		ResponseBody:   sql.NullString{String: string(responseBody), Valid: true},
		DeliveredAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})

	// Handle non-2xx responses with retry logic
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.queries.UpdateWebhookDelivery(ctx, db.UpdateWebhookDeliveryParams{
			ID:           delivery.ID,
			ErrorMessage: sql.NullString{String: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(responseBody)), Valid: true},
		})
		s.handleRetry(ctx, webhook, event, payload, int64(delivery.ID),
			fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(responseBody)), 1)
	}
}

// handleRetry manages webhook delivery retries
func (s *Service) handleRetry(ctx context.Context, webhook db.Webhook, event WebhookEvent,
	payload map[string]any, deliveryID int64, errorMsg string, attempt int) {

	maxRetries := int(webhook.MaxRetries.Int32)
	if maxRetries <= 0 {
		maxRetries = 3
	}

	if attempt >= maxRetries {
		// Max retries reached, mark as failed
		s.queries.UpdateWebhookDelivery(ctx, db.UpdateWebhookDeliveryParams{
			ID:           int32(deliveryID),
			ErrorMessage: sql.NullString{String: fmt.Sprintf("Failed after %d attempts: %s", attempt, errorMsg), Valid: true},
		})
		return
	}

	// capped exponential backoff with jitter
	base := time.Second * time.Duration(attempt*attempt)
	if base > 30*time.Second {
		base = 30 * time.Second
	}
	jitter := time.Duration(rand.Int63n(int64(time.Second))) // 0–1s
	delay := base + jitter

	time.AfterFunc(delay, func() {
		s.retryWebhook(ctx, webhook, event, payload, deliveryID, attempt+1)
	})
}

// retryWebhook attempts to resend a failed webhook
func (s *Service) retryWebhook(ctx context.Context, webhook db.Webhook, event WebhookEvent,
	payload map[string]any, originalDeliveryID int64, attempt int) {

	payloadBytes, _ := json.Marshal(payload)

	// Record new delivery attempt
	delivery, err := s.queries.RecordWebhookDelivery(ctx, db.RecordWebhookDeliveryParams{
		WebhookID:     webhook.ID,
		EventType:     event.Type,
		Payload:       payloadBytes,
		AttemptNumber: sql.NullInt32{Int32: int32(attempt), Valid: true},
	})
	if err != nil {
		return
	}

	// Similar sending logic as sendWebhook but with retry context
	client := &http.Client{Timeout: time.Duration(webhook.TimeoutMs.Int32) * time.Millisecond}
	req, err := http.NewRequest("POST", webhook.Url, bytes.NewReader(payloadBytes))
	if err != nil {
		s.queries.UpdateWebhookDelivery(ctx, db.UpdateWebhookDeliveryParams{
			ID:           delivery.ID,
			ErrorMessage: sql.NullString{String: err.Error(), Valid: true},
		})
		return
	}

	// Add headers (same as original)
	timestamp := time.Now().Format(time.RFC3339)
	signature := s.generateSignature(webhook.Secret, payloadBytes, timestamp)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Event", event.Type)
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Timestamp", timestamp)

	resp, err := client.Do(req)
	if err != nil {
		s.handleRetry(ctx, webhook, event, payload, int64(delivery.ID), err.Error(), attempt)
		return
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(resp.Body)
	s.queries.UpdateWebhookDelivery(ctx, db.UpdateWebhookDeliveryParams{
		ID:             delivery.ID,
		ResponseStatus: sql.NullInt32{Int32: int32(resp.StatusCode), Valid: true},
		ResponseBody:   sql.NullString{String: string(responseBody), Valid: true},
		DeliveredAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.handleRetry(ctx, webhook, event, payload, int64(delivery.ID),
			fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(responseBody)), attempt)
	}
}

// Generate signature for webhook verification
func (s *Service) generateSignature(secret string, payload []byte, timestamp string) string {
	message := fmt.Sprintf("%s.%s", timestamp, string(payload))
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifySignature validates an incoming webhook signature
func (s *Service) VerifySignature(secret, signature, timestamp string, payload []byte) bool {
	expectedSignature := s.generateSignature(secret, payload, timestamp)
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// Record delivery failure
func (s *Service) recordDeliveryFailure(ctx context.Context, webhookID int32, eventType string,
	payload map[string]any, statusCode int, errorMsg string, attempt int) {

	payloadBytes, _ := json.Marshal(payload)
	s.queries.RecordWebhookDelivery(ctx, db.RecordWebhookDeliveryParams{
		WebhookID:     webhookID,
		EventType:     eventType,
		Payload:       payloadBytes,
		AttemptNumber: sql.NullInt32{Int32: int32(attempt), Valid: true},
	})
}

// GetAvailableEvents returns all registered webhook events
func (s *Service) GetAvailableEvents(ctx context.Context) ([]db.WebhookEvent, error) {
	return s.queries.GetWebhookEvents(ctx)
}

// CreateWebhook creates a new webhook subscription
func (s *Service) CreateWebhook(ctx context.Context, params CreateWebhookParams) (*db.Webhook, error) {
	// Generate webhook secret
	secret, err := generateRandomString(32)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(params.URL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, fmt.Errorf("invalid webhook URL")
	}
	if params.TimeoutMs <= 0 || params.TimeoutMs > 30000 {
		params.TimeoutMs = 5000
	}
	if params.MaxRetries < 0 || params.MaxRetries > 10 {
		params.MaxRetries = 3
	}

	webhook, err := s.queries.CreateWebhook(ctx, db.CreateWebhookParams{
		Name:        params.Name,
		Description: sql.NullString{String: params.Description, Valid: params.Description != ""},
		Url:         params.URL,
		Secret:      secret,
		Events:      params.Events,
		TenantID:    int32(params.TenantID),
		MaxRetries:  sql.NullInt32{Int32: int32(params.MaxRetries), Valid: params.MaxRetries != 0},
		TimeoutMs:   sql.NullInt32{Int32: int32(params.TimeoutMs), Valid: params.TimeoutMs != 0},
	})
	if err != nil {
		return nil, err
	}

	return &webhook, nil
}

// Helper function to generate random strings
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
