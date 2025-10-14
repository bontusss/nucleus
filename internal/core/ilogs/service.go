package logs

import (
	"context"
	"database/sql"
	db "nucleus/db/sqlc"
)

type Logs struct {
	db      *sql.DB
	queries Querier
}

func NewLogs(db *sql.DB, queries Querier) *Logs {
	return &Logs{db, queries}
}

func (l *Logs) GetActivityLogs(ctx context.Context, params db.GetActivityLogParams) (db.ActivityLog, error) {
	return l.queries.GetActivityLog(ctx, params)
}
