-- name: CreateWebhook :one
INSERT INTO webhooks (name, description, url, secret, events, tenant_id, max_retries, timeout_ms)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetWebhooksByUser :many
SELECT * FROM webhooks WHERE tenant_id = $1 ORDER BY created_at DESC;

-- name: GetWebhookByID :one
SELECT * FROM webhooks WHERE id = $1;

-- name: GetWebhooksByEvent :many
SELECT * FROM webhooks
WHERE tenant_id = $1 AND $2 = ANY(events) AND is_active = true
ORDER BY created_at DESC;

-- name: UpdateWebhook :one
UPDATE webhooks
SET name = $2, description = $3, url = $4, events = $5, is_active = $6,
    max_retries = $7, timeout_ms = $8, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND tenant_id = $2
RETURNING *;

-- name: DeleteWebhook :exec
DELETE FROM webhooks WHERE id = $1 AND tenant_id = $2;

-- name: RecordWebhookDelivery :one
INSERT INTO webhook_deliveries (webhook_id, event_type, payload, attempt_number)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateWebhookDelivery :exec
UPDATE webhook_deliveries
SET response_status = $2, response_body = $3, error_message = $4, delivered_at = $5
WHERE id = $1;

-- name: GetWebhookDeliveries :many
SELECT wd.*, w.name as webhook_name, w.url as webhook_url
FROM webhook_deliveries wd
JOIN webhooks w ON wd.webhook_id = w.id
WHERE wd.webhook_id = $1
ORDER BY wd.created_at DESC
LIMIT $2;

-- name: GetWebhookEvents :many
SELECT * FROM webhook_events ORDER BY module, event_type;

-- name: GetDeliveryStats :one
SELECT
    COUNT(*) as total_deliveries,
    COUNT(CASE WHEN response_status BETWEEN 200 AND 299 THEN 1 END) as successful_deliveries,
    COUNT(CASE WHEN response_status IS NULL OR response_status NOT BETWEEN 200 AND 299 THEN 1 END) as failed_deliveries,
    AVG(EXTRACT(EPOCH FROM (delivered_at - created_at))) as avg_delivery_time_seconds
FROM webhook_deliveries
WHERE webhook_id = $1 AND created_at >= $2 AND created_at <= $3;
