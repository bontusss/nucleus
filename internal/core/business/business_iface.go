package business

import (
	"context"
	db "nucleus/db/sqlc"
	"nucleus/pkg/jwt"
)

type Querier interface {
	CreateBusiness(ctx context.Context, params db.CreateBusinessParams) (db.Business, error)
	GetBusiness(ctx context.Context, params db.GetBusinessParams) (db.Business, error)
	UpdateBusiness(ctx context.Context, params db.UpdateBusinessParams) (db.Business, error)
	DeleteBusiness(ctx context.Context, params db.DeleteBusinessParams) (db.Business, error)
	ListBusinesses(ctx context.Context, id int32) ([]db.Business, error)
	CreateBranch(ctx context.Context, params db.CreateBranchParams) (db.Branch, error)
	GetBranch(ctx context.Context, params db.GetBranchParams) (db.Branch, error)
	UpdateBranch(ctx context.Context, params db.UpdateBranchParams) (db.Branch, error)
	DeleteBranch(ctx context.Context, id int32) (db.Branch, error)
	ListBranches(ctx context.Context, businessID int32) ([]db.Branch, error)
	CreateStore(ctx context.Context, params db.CreateStoreParams) (db.Store, error)
	CreateSupplier(ctx context.Context, params db.CreateSupplierParams) (db.Supplier, error)
	GetSupplier(ctx context.Context, params db.GetSupplierParams) (db.Supplier, error)
	ListSuppliers(ctx context.Context, businessID int32) ([]db.Supplier, error)
	UpdateSupplier(ctx context.Context, params db.UpdateSupplierParams) (db.Supplier, error)
	DeleteSupplier(ctx context.Context, params db.DeleteSupplierParams) (db.Supplier, error)
	CreateActivityLog(ctx context.Context, params db.CreateActivityLogParams) (db.ActivityLog, error)
}

type BusinessInterface interface {
	CreateBusinessWithBranch(ctx context.Context, claims *jwt.Claims, params db.CreateBusinessParams, ipAddress, userAgent string) (*db.Business, *db.Branch, error)
	CreateBusiness(ctx context.Context, params db.CreateBusinessParams) (db.Business, error)
	GetBusiness(ctx context.Context, params db.GetBusinessParams) (db.Business, error)
	UpdateBusiness(ctx context.Context, params db.UpdateBusinessParams, c *jwt.Claims, ipAddress, userAgent string) (*db.Business, error)
	DeleteBusiness(ctx context.Context, params db.DeleteBusinessParams) (db.Business, error)
	ListBusinesses(ctx context.Context, ownerID int32) ([]db.Business, error)
	CreateBranch(ctx context.Context, params db.CreateBranchParams, claims *jwt.Claims, ip, agent string) (*db.Branch, error)
	GetBranch(ctx context.Context, params db.GetBranchParams) (db.Branch, error)
	UpdateBranch(ctx context.Context, params db.UpdateBranchParams) (db.Branch, error)
	DeleteBranch(ctx context.Context, id int32) (db.Branch, error)
	ListBranches(ctx context.Context, businessID int32) ([]db.Branch, error)
	CreateSupplier(ctx context.Context, params db.CreateSupplierParams) (db.Supplier, error)
	GetSupplier(ctx context.Context, params db.GetSupplierParams) (db.Supplier, error)
	ListSuppliers(ctx context.Context, businessID int32) ([]db.Supplier, error)
	DeleteSupplier(ctx context.Context, params db.DeleteSupplierParams) (db.Supplier, error)
	UpdateSupplier(ctx context.Context, params db.UpdateSupplierParams) (db.Supplier, error)
	CreateActivityLog(ctx context.Context, params db.CreateActivityLogParams) (db.ActivityLog, error)
}
