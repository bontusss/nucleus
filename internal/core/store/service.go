package store

import (
	"context"
	"database/sql"
	db "herp/db/sqlc"
)

type Store struct {
	db      *sql.DB
	queries Querier
}

func NewStore(db *sql.DB, queries Querier) *Store {
	return &Store{db, queries}
}

func (s *Store) CreateStore(ctx context.Context, params db.CreateStoreParams) (db.Store, error) {
	return s.queries.CreateStore(ctx, params)
}

func (s *Store) DeleteStore(ctx context.Context, id int32) error {
	return s.queries.DeleteStore(ctx, id)
}

func (s *Store) GetStoreByID(ctx context.Context, id int32) (db.Store, error) {
	return s.queries.GetStoreByID(ctx, id)
}

func (s *Store) UpdateStore(ctx context.Context, params db.UpdateStoreParams) (db.Store, error) {
	return s.queries.UpdateStore(ctx, params)
}

func (s *Store) LogActivity(ctx context.Context, params db.LogActivityParams) (db.ActivityLog, error) {
	return s.queries.LogActivity(ctx, params)
}