-- name: CreateAPIKey :one
INSERT INTO api_keys (key_name, api_key, api_secret, user_id, allowed_modules, monthly_limit, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetAPIKeyByKey :one
SELECT * FROM api_keys WHERE api_key = $1;

-- name: GetAPIKeyByID :one
SELECT * FROM api_keys WHERE id = $1;

-- name: GetAPIKeysByUser :many
SELECT * FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC;

-- name: UpdateAPIKey :one
UPDATE api_keys
SET key_name = $2, allowed_modules = $3, monthly_limit = $4, expires_at = $5, is_active = $6, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteAPIKey :exec
DELETE FROM api_keys WHERE id = $1;

-- name: IncrementAPIKeyUsage :exec
UPDATE api_keys
SET current_month_requests = current_month_requests + 1, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: ResetMonthlyUsage :exec
UPDATE api_keys
SET current_month_requests = 0, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: RecordAPIKeyUsage :one
INSERT INTO api_key_usages (api_key_id, module_name, endpoint, request_method, request_size, response_status, ip_address, user_agent)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetAPIKeyUsageByDateRange :many
SELECT * FROM api_key_usages
WHERE api_key_id = $1 AND created_at BETWEEN $2 AND $3
ORDER BY created_at DESC;

-- name: GetAPIKeyUsageStats :one
SELECT
    COUNT(*) as total_requests,
    COALESCE(SUM(request_size), 0) as total_bytes,
    COUNT(DISTINCT endpoint) as unique_endpoints
FROM api_key_usages
WHERE api_key_id = $1 AND created_at BETWEEN $2 AND $3;
