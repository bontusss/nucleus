package store

import (
	"context"
	"database/sql"
	db "herp/db/sqlc"
)

type Querier interface {
	CreateStore(ctx context.Context, params db.CreateStoreParams) (db.Store, error)
	DeleteStore(ctx context.Context, id int32) error
	GetStoreByID(ctx context.Context, id int32) (db.Store, error)
	DeactivateStore(ctx context.Context, id int32) (db.Store, error)
	GetCentralStoreByBranch(ctx context.Context, branchID int32) (db.Store, error)
	GetStoresByBranch(ctx context.Context, branchID int32) ([]db.Store, error)
	ListStores(ctx context.Context) ([]db.Store, error)
	UpdateStore(ctx context.Context, params db.UpdateStoreParams) (db.Store, error)
	SearchStoresByName(ctx context.Context, name sql.NullString) ([]db.Store, error)
	LogActivity(ctx context.Context, params db.LogActivityParams) (db.ActivityLog, error)
}

type StoreInterface interface {
	CreateStore(ctx context.Context, params db.CreateStoreParams) (db.Store, error)
	DeleteStore(ctx context.Context, id int32) error
	GetStoreByID(ctx context.Context, id int32) (db.Store, error)
	UpdateStore(ctx context.Context, params db.UpdateStoreParams) (db.Store, error)
	LogActivity(ctx context.Context, params db.LogActivityParams) (db.ActivityLog, error)
}
