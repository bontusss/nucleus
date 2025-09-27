package logs

import (
	"context"
	"database/sql"
	db "herp/db/sqlc"
)

type Logs struct {
	db      *sql.DB
	queries Querier
}

func NewLogs(db *sql.DB, queries Querier) *Logs {
	return &Logs{db, queries}
}

func(l *Logs) GetActivityLogs(ctx context.Context, limit int32) ([]db.ActivityLog, error) {
	return l.queries.GetActivityLogs(ctx, limit)
}