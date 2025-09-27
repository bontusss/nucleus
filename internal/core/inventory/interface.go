package inventory

import (
	"context"
	db "herp/db/sqlc"
)

type Querier interface {
	CreateBrand(ctx context.Context, params db.CreateBrandParams) (db.Brand, error)
	CreateCategory(ctx context.Context, params db.CreateCategoryParams) (db.Category, error)
	CreateItem(ctx context.Context, params db.CreateItemParams) (db.Item, error)
	// CreateItemImage(ctx context.Context, params db.CreateItemImageParams) (db.ItemImage, error)
	CreateVariation(ctx context.Context, params db.CreateVariationParams) (db.Variation, error)
	// // CreateInventory(ctx context.Context, params db.CreateI)
	// DeleteBrand(ctx context.Context, is int32) error
	// DeleteCategory(ctx context.Context, id int32) error
	// DeleteInventory(ctx context.Context, id int32) error
	// DeleteItem(ctx context.Context, id int32) error
	// DeleteItemImage(ctx context.Context, id int32) error
	// DeleteVariation(ctx context.Context, id int32) error
	GetBrand(ctx context.Context, id int32) (db.Brand, error)
	GetCategory(ctx context.Context, id int32) (db.Category, error)
	// GetInventoryByStore(ctx context.Context, storeID int32) ([]db.Inventory, error)
	// GetInventoryItem(ctx context.Context, params db.GetInventoryItemParams) (db.Inventory, error)
	GetItem(ctx context.Context, id int32) (db.Item, error)
	// GetItemImageByItem(ctx context.Context, itemID sql.NullInt32) ([]db.ItemImage, error)
	// GetItemImagesByVariation(ctx context.Context, variationID sql.NullInt32) ([]db.ItemImage, error)
	// GetVariation(ctx context.Context, id int32) (db.Variation, error)
	// ListBrand(ctx context.Context) ([]db.Brand, error)
	// ListCategories(ctx context.Context) ([]db.Category, error)
	// ListItems(ctx context.Context) ([]db.Item, error)
	// ListItemsByCategory(ctx context.Context, categoryID sql.NullInt32) ([]db.Item, error)
	// ListVariationsByItem(ctx context.Context, itemID int32) ([]db.Variation, error)
	// UpdateBrand(ctx context.Context, params db.UpdateBrandParams) ([]db.Brand, error)
	// UpdateCategory(ctx context.Context, params db.UpdateCategoryParams) ([]db.Category, error)
	// updateInventoryQuantity(ctx context.Context, params db.UpdateInventoryQuantityParams) (db.Inventory, error)
	// UpdateItem(ctx context.Context, params db.UpdateItemParams) (db.Item, error)
	// UpdateVariation(ctx context.Context, params db.UpdateVariationParams) (db.Variation, error)
	// UpsertInventory(ctx context.Context, param db.UpsertInventoryParams) (db.Inventory, error) // Create Inventory
	// UpdateUnit(ctx context.Context, args db.UpdateUnitParams) (db.Unit, error)
	CreateUnit(ctx context.Context, args db.CreateUnitParams) (db.Unit, error)
	GetUnitByID(ctx context.Context, id int32) (db.Unit, error)
	// ListUnits(ctx context.Context) ([]db.Unit, error)
	// DeleteUnit(ctx context.Context, id int32) (db.Unit, error)
	CreateColor(ctx context.Context, name string) (db.Color, error)
	GetColorByID(ctx context.Context, id int32) (db.Color, error)
	GetColorByName(ctx context.Context, name string) (db.Color, error)
	// ListColors(ctx context.Context) ([]db.Color, error)
	// UpdateColor(ctx context.Context, args db.UpdateColorParams) (db.Color, error)
	LogActivity(ctx context.Context, params db.LogActivityParams) (db.ActivityLog, error)
	// DeleteColor(ctx context.Context, id int32) (db.Color, error)
}

type InventoryInterface interface {
	CreateBrand(ctx context.Context, params db.CreateBrandParams) (db.Brand, error)
	LogActivity(ctx context.Context, params db.LogActivityParams) (db.ActivityLog, error)
	CreateCategory(ctx context.Context, params db.CreateCategoryParams) (db.Category, error)
	GetCategory(ctx context.Context, id int32) (db.Category, error)
	CreateItem(ctx context.Context, params db.CreateItemParams) (db.Item, error)
	CreateItemWithVariations(ctx context.Context, params db.CreateItemParams, defaultUnitID int32, defaultPrice string) (db.Item, db.Variation, error)
	GetBrand(ctx context.Context, id int32) (db.Brand, error)
	CreateVariation(ctx context.Context, params db.CreateVariationParams) (db.Variation, error)
	GetItem(ctx context.Context, id int32) (db.Item, error)
	CreateUnit(ctx context.Context, args db.CreateUnitParams) (db.Unit, error)
	GetUnitByID(ctx context.Context, id int32) (db.Unit, error)
	CreateColor(ctx context.Context, name string) (db.Color, error)
	GetColorByID(ctx context.Context, id int32) (db.Color, error)
	GetColorByName(ctx context.Context, name string) (db.Color, error)
}
