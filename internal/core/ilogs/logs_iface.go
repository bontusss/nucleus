package logs

import (
	"context"
	db "nucleus/db/sqlc"
)

type Querier interface {
	GetActivityLog(ctx context.Context, params db.GetActivityLogParams) (db.ActivityLog, error)
}

type LogsInterface interface {
	GetActivityLogs(ctx context.Context, params db.GetActivityLogParams) (db.ActivityLog, error)
}
