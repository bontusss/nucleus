// Copyright 2025 The HERP Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package business

import (
	"context"
	"database/sql"
	"fmt"
	db "nucleus/db/sqlc"
)

type Business struct {
	db      *sql.DB
	queries Querier
}

func NewBusiness(queries Querier, db *sql.DB) *Business {
	return &Business{
		queries: queries,
		db:      db,
	}
}

// CreateBusiness creates a new business with a default branch.
func (c *Business) CreateBusinessWithBranch(ctx context.Context, params db.CreateBusinessParams) (db.Business, db.Branch, error) {
	q, ok := c.queries.(*db.Queries)
	if !ok {
		return db.Business{}, db.Branch{}, nil
	}

	// Start a transaction
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return db.Business{}, db.Branch{}, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	txQueries := q.WithTx(tx)

	// Create the business
	business, err := txQueries.CreateBusiness(ctx, params)
	if err != nil {
		return db.Business{}, db.Branch{}, err
	}

	// Create a default branch for the business
	branchParams := db.CreateBranchParams{
		Name:       "Main",
		BusinessID: business.ID,
	}

	fmt.Printf("creating branch %s", branchParams.Name)
	branch, err := txQueries.CreateBranch(ctx, branchParams)
	if err != nil {
		return db.Business{}, db.Branch{}, err
	}

	fmt.Printf("branch created. id %d and name %s", branch.ID, branch.Name)

	return business, branch, nil
}

func (c *Business) CreateBusiness(ctx context.Context, params db.CreateBusinessParams) (db.Business, error) {
	return c.queries.CreateBusiness(ctx, params)
}

// GetBusiness retrieves a business by its ID.
func (c *Business) GetBusiness(ctx context.Context, args db.GetBusinessParams) (db.Business, error) {
	return c.queries.GetBusiness(ctx, args)
}

// UpdateBusiness updates an existing business.
func (c *Business) UpdateBusiness(ctx context.Context, params db.UpdateBusinessParams) (db.Business, error) {
	return c.queries.UpdateBusiness(ctx, params)
}

// DeleteBusiness deletes a business by its ID.
func (c *Business) DeleteBusiness(ctx context.Context, args db.DeleteBusinessParams) (db.Business, error) {
	return c.queries.DeleteBusiness(ctx, args)
}

// ListBusinesses lists all businesses.
func (c *Business) ListBusinesses(ctx context.Context, ownerID int32) ([]db.Business, error) {
	return c.queries.ListBusinesses(ctx, ownerID)
}

// --------Branch Methods-------- //

// CreateBranch creates a new branch.
func (c *Business) CreateBranch(ctx context.Context, params db.CreateBranchParams) (db.Branch, error) {
	return c.queries.CreateBranch(ctx, params)
}

// GetBranch retrieves a branch by its ID.
func (c *Business) GetBranch(ctx context.Context, id int32) (db.Branch, error) {
	return c.queries.GetBranch(ctx, id)
}

// UpdateBranch updates an existing branch.
func (c *Business) UpdateBranch(ctx context.Context, params db.UpdateBranchParams) (db.Branch, error) {
	return c.queries.UpdateBranch(ctx, params)
}

// DeleteBranch deletes a branch by its ID.
func (c *Business) DeleteBranch(ctx context.Context, id int32) (db.Branch, error) {
	return c.queries.DeleteBranch(ctx, id)
}

// ListBranch lists branches
func (c *Business) ListBranches(ctx context.Context, id int32) ([]db.Branch, error) {
	return c.queries.ListBranches(ctx, id)
}

func (c *Business) LogActivity(ctx context.Context, params db.LogActivityParams) (db.ActivityLog, error) {
	return c.queries.LogActivity(ctx, params)
}

// ------ Suppliers ----
func (c *Business) CreateSupplier(ctx context.Context, params db.CreateSupplierParams) (db.Supplier, error) {
	return c.queries.CreateSupplier(ctx, params)
}
func (c *Business) GetSupplier(ctx context.Context, params db.GetSupplierParams) (db.Supplier, error) {
	return c.queries.GetSupplier(ctx, params)
}
func (c *Business) ListSuppliers(ctx context.Context, businessID int32) ([]db.Supplier, error) {
	return c.queries.ListSuppliers(ctx, businessID)
}
func (c *Business) UpdateSupplier(ctx context.Context, params db.UpdateSupplierParams) (db.Supplier, error) {
	return c.queries.UpdateSupplier(ctx, params)
}

func (c *Business) DeleteSupplier(ctx context.Context, params db.DeleteSupplierParams) (db.Supplier, error) {
	return c.queries.DeleteSupplier(ctx, params)
}
