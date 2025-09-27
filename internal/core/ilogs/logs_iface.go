package logs

import (
	"context"
	db "herp/db/sqlc"
)

type Querier interface {
	GetActivityLogs(ctx context.Context, limit int32) ([]db.ActivityLog, error)
}

type LogsInterface interface {
	GetActivityLogs(ctx context.Context, limit int32) ([]db.ActivityLog, error)
}
