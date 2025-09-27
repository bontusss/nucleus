package inventory

import (
	"database/sql"
	"fmt"
	db "herp/db/sqlc"
	"herp/internal/auth"
	"herp/internal/utils"
	"herp/pkg/jwt"
	"herp/pkg/monitoring/logging"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	"github.com/lib/pq"
)

type Handler struct {
	service InventoryInterface
	logger  *logging.Logger
}

func NewInventoryHandler(service InventoryInterface, l *logging.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  l,
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authSvc *auth.Service) {
	inventory := r.Group("/inventory")
	inventory.Use(auth.AuthMiiddleware(authSvc))

	brand := inventory.Group("/brand")
	{
		brand.POST("", auth.PermissionMiddleware(authSvc, "inventory:create"), h.createBrand)
	}

	category := inventory.Group("/category")
	{
		category.POST("", auth.PermissionMiddleware(authSvc, "inventory:create"), h.createCategory)
	}

	// item := inventory.Group("/item")
	// {
	// 	// item.POST("", auth.PermissionMiddleware(authSvc, "inventory:create"), h.createItem)
	// }

	variation := inventory.Group("/variation")
	{
		variation.POST("", auth.PermissionMiddleware(authSvc, "inventory:create"), h.CreateVariation)
	}

	unit := inventory.Group("/unit")
	{
		unit.POST("", auth.PermissionMiddleware(authSvc, "inventory:create"), h.createUnit)
	}
}

type CreateBrandRequest struct {
	Name        string `form:"name" binding:"required" example:"Coca-Cola"`
	Description string `form:"description" binding:"omitempty" example:"..."`
	IsActive    bool   `form:"is_active" binding:"omitempty" example:"true" default:"true"`
}

type CreateBrandResponse struct {
	ID          int32  `json:"id"`
	Name        string `json:"name" binding:"omitempty" example:"Coca-Cola"`
	Description string `json:"description" binding:"omitempty" example:"..."`
	IsActive    bool   `json:"is_active" binding:"omitempty" example:"true"`
	Logo        string `json:"logo" binding:"omitempty"`
}

// CreateBrand godoc
// @Summary Create a brand
// @Description Create a new brand.
// @Tags inventory
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param name formData string true "Brand name"
// @Param description formData string false "Brand description"
// @Param isActive formData bool false "is brand active?"
// @Param logo formData file false "brand logo"
// @Success 201 {object} CreateBrandResponse
// @Router /api/v1/inventory/brand [post]
func (h *Handler) createBrand(c *gin.Context) {
	claims, ok := jwt.GetUserFromContext(c)
	if !ok {
		h.logger.Errorf("could not get user from context")
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	// Parse form-data (multipart) instead of JSON
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil { // 10MB limit
		h.logger.Errorf("multipart parse error: %v", err)
		utils.ErrorResponse(c, 400, utils.INVALID_REQUEST_DATA)
		return
	}

	var req CreateBrandRequest
	if err := c.ShouldBind(&req); err != nil {
		h.logger.Errorf("error binding creating brand request data: %v", err)
		utils.ErrorResponse(c, 400, utils.INVALID_REQUEST_DATA)
		return
	}

	// Handle logo file separately
	var logoUrl string
	if url, err := utils.UploadFile(c, "logo", "images", 2<<20); err == nil && url != "" {
		logoUrl = url
	}

	var params db.CreateBrandParams
	err := copier.Copy(&params, &req)
	if err != nil {
		h.logger.Errorf("error copying create brand request data: %v", err)
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	if logoUrl != "" {
		params.Logo = sql.NullString{String: logoUrl, Valid: true}
	}

	brand, err := h.service.CreateBrand(c, params)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok {
			switch pgErr.Code {
			case "23505": // unique_violation
				utils.ErrorResponse(c, 400, fmt.Sprintf("brand with name %s already exists", req.Name))
				return
			}
		}

		h.logger.Errorf("error creating a brand: %v", err)
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	h.service.LogActivity(c, db.LogActivityParams{
		UserID:     int32(claims.UserID),
		EntityID:   brand.ID,
		Action:     "Created Brand",
		EntityType: "Brand",
		Details:    utils.WriteActivityDetails(claims.Username, claims.Email, fmt.Sprintf("Created brand %s", brand.Name), brand.CreatedAt.Time),
		IpAddress:  sql.NullString{Valid: true, String: utils.GetClientIP(c)},
		UserAgent:  sql.NullString{Valid: true, String: c.Request.UserAgent()},
	})

	utils.SuccessResponse(c, 201, "brand created", CreateBrandResponse{
		ID:          brand.ID,
		Name:        brand.Name,
		Description: brand.Description.String,
		IsActive:    brand.IsActive.Bool,
		Logo:        brand.Logo.String,
	})
}

type Category struct {
	Name        string `json:"name" binding:"required"`
	ParentID    *int32 `json:"parent_id"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active" example:"true" default:"true"`
}

type CategoryResponse struct {
	ID          int32  `json:"id"`
	Name        string `json:"name"`
	ParentID    *int32 `json:"parent_id"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
}

// CreateCategory godoc
// @Summary Create Category
// @Description Create a category
// @Tags inventory
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 201 {object} CategoryResponse
// @Param body body Category true "category details"
// @Failure 400
// @Failure 401
// @Failure 403
// @Failure 500
// @Router /api/v1/inventory/category [post]
func (h *Handler) createCategory(c *gin.Context) {
	claims, ok := jwt.GetUserFromContext(c)
	if !ok {
		h.logger.Errorf("could not get user from context")
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	var req Category
	if err := c.ShouldBind(&req); err != nil {
		h.logger.Errorf("error binding create category request data: %v", err)
		utils.ErrorResponse(c, 400, utils.INVALID_REQUEST_DATA)
		return
	}

	// Validate parent if provided
	if req.ParentID != nil {
		_, err := h.service.GetCategory(c, *req.ParentID)
		if err == sql.ErrNoRows {
			utils.ErrorResponse(c, 400, fmt.Sprintf("parent category with id %d does not exist", *req.ParentID))
			return
		} else if err != nil {
			h.logger.Errorf("error checking parent category: %v", err)
			utils.ErrorResponse(c, 500, utils.SERVERERROR)
			return
		}
	}

	var params db.CreateCategoryParams
	err := copier.Copy(&params, &req)
	if err != nil {
		h.logger.Errorf("error copying create category request data: %v", err)
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	category, err := h.service.CreateCategory(c, params)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok {
			switch pgErr.Code {
			case "23505": // unique_violation
				utils.ErrorResponse(c, 400, fmt.Sprintf("category with name %s already exists", req.Name))
				return
			}
		}

		h.logger.Errorf("error creating a category: %v", err)
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	h.service.LogActivity(c, db.LogActivityParams{
		UserID:     int32(claims.UserID),
		EntityID:   category.ID,
		Action:     "Created Category",
		EntityType: "Category",
		Details:    utils.WriteActivityDetails(claims.Username, claims.Email, fmt.Sprintf("Created category %s", category.Name), category.CreatedAt.Time),
		IpAddress:  sql.NullString{Valid: true, String: utils.GetClientIP(c)},
		UserAgent:  sql.NullString{Valid: true, String: c.Request.UserAgent()},
	})

	utils.SuccessResponse(c, 201, "created category", CategoryResponse{
		ID:          category.ID,
		Name:        category.Name,
		ParentID:    &category.ParentID.Int32,
		Description: category.Description.String,
		IsActive:    category.IsActive.Bool,
	})
}

type ItemRequest struct {
	BrandID      *int32 `json:"brand_id" binding:"omitempty" example:"3"`
	CategoryID   int32  `json:"category_id" binding:"required" example:"1"`
	Name         string `json:"name" binding:"required" example:"Shoes"`
	Description  string `json:"description"`
	IsActive     bool   `json:"is_active" default:"true" example:"true"`
	UnitID       int32  `json:"unit_id"`
	DefaultPrice string `json:"default_price"`
}

type ItemResponse struct {
	ID            int32        `json:"id"`
	BrandID       int32        `json:"brand_id" binding:"required" example:"3"`
	CategoryID    int32        `json:"category_id" binding:"required" example:"1"`
	Name          string       `json:"name" binding:"required" example:"Shoes"`
	Description   string       `json:"description"`
	Has_variation bool         `json:"has_variation"`
	IsActive      bool         `json:"is_active"`
	Variation     db.Variation `json:"variation"`
}

// CreateItem godoc
// @Summary Create Item
// @Description An ite
// @Tags inventory
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 201 {object} ItemResponse
// @Param body body ItemRequest true "item details"
// @Failure 400
// @Failure 401
// @Failure 403
// @Failure 500
// @Router /api/v1/inventory/item [post]
// func (h *Handler) createItem(c *gin.Context) {
// 	claims, ok := jwt.GetUserFromContext(c)
// 	if !ok {
// 		h.logger.Errorf("could not get user from context")
// 		utils.ErrorResponse(c, 500, utils.SERVERERROR)
// 		return
// 	}

// 	var req ItemRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		h.logger.Errorf("create item binding error: %v", err)
// 		utils.ErrorResponse(c, 400, utils.INVALID_REQUEST_DATA)
// 		return
// 	}

// 	var params db.CreateItemParams
// 	err := copier.Copy(&params, &req)
// 	if err != nil {
// 		h.logger.Errorf("error copying create item request data: %v", err)
// 		utils.ErrorResponse(c, 500, utils.SERVERERROR)
// 		return
// 	}

// 	_, err = h.service.GetBrand(c, params.BrandID.Int32)
// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			utils.ErrorResponse(c, 400, fmt.Sprintf("brand with id %d does not exist", req.BrandID))
// 			return
// 		}

// 		h.logger.Errorf("error getting brand with id %d: %v", req.BrandID, err)
// 		utils.ErrorResponse(c, 500, utils.SERVERERROR)
// 		return
// 	}

// 	_, err = h.service.GetCategory(c, params.BrandID.Int32)
// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			utils.ErrorResponse(c, 400, fmt.Sprintf("category with id %d does not exist", req.CategoryID))
// 			return
// 		}

// 		h.logger.Errorf("error getting category with id %d: %v", req.CategoryID, err)
// 		utils.ErrorResponse(c, 500, utils.SERVERERROR)
// 		return
// 	}

// 	item, variations, err := h.service.CreateItemWithVariations(c, params)
// 	if err != nil {
// 		h.logger.Errorf("error creating item: %v", err)
// 		utils.ErrorResponse(c, 500, utils.SERVERERROR)
// 		return
// 	}

// 	// add activity log here
// 	_, err = h.service.LogActivity(c, db.LogActivityParams{
// 		UserID:     int32(claims.UserID),
// 		Action:     "Created Item",
// 		EntityType: "Item",
// 		EntityID:   item.ID,
// 		Details:    utils.WriteActivityDetails("system", "system", fmt.Sprintf("Created item %s", item.Name), item.CreatedAt.Time),
// 		IpAddress:  sql.NullString{Valid: true, String: utils.GetClientIP(c)},
// 		UserAgent:  sql.NullString{Valid: true, String: c.Request.UserAgent()},
// 	})

// 	if err != nil {
// 		h.logger.Warnf("error logging create item activity: %v", err)
// 		// not returning error to user as branch has been created successfully
// 	}

// 	utils.SuccessResponse(c, 201, "item created", ItemResponse{
// 		ID:          item.ID,
// 		Name:        item.Name,
// 		BrandID:     item.BrandID.Int32,
// 		CategoryID:  item.CategoryID,
// 		Description: item.Description.String,
// 		IsActive:    item.IsActive.Bool,
// 		Variation:   variations, // TODO: serialize
// 	})
// }

type UnitRequest struct {
	Name      string `json:"name" binding:"required" example:"kg"`
	ShortCode string `json:"short_code"`
}

type UnitResponse struct {
	ID        int32  `json:"id"`
	Name      string `json:"name"`
	ShortCode string `json:"short_code"`
}

// CreateUnit godoc
// @Summary Create a unit
// @Description Create an inventory unit.
// @Tags inventory
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 201 {object} UnitResponse
// @Param body body UnitRequest true "unit details"
// @Failure 400
// @Failure 401
// @Failure 403
// @Failure 500
// @Router /api/v1/inventory/unit [post]
func (h *Handler) createUnit(c *gin.Context) {
	var req UnitRequest
	if err := c.ShouldBind(&req); err != nil {
		h.logger.Errorf("error binding creating unit request data: %v", err)
		utils.ErrorResponse(c, 400, utils.INVALID_REQUEST_DATA)
		return
	}

	params := db.CreateUnitParams{
		Name:      req.Name,
		ShortCode: sql.NullString{String: req.ShortCode, Valid: req.ShortCode != ""},
	}

	unit, err := h.service.CreateUnit(c, params)
	if err != nil {
		h.logger.Errorf("error creating a unit: %v", err)
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	utils.SuccessResponse(c, 201, "unit created", UnitResponse{
		Name:      unit.Name,
		ShortCode: unit.ShortCode.String,
		ID:        unit.ID,
	})
}

type VariationRequest struct {
	ItemID       int32  `json:"item_id" binding:"required" example:"1"`
	Sku          string `json:"sku" binding:"omitempty" example:"GTR30l"`
	Name         string `json:"name" binding:"required" example:"...."`
	UnitID       int32  `json:"unit_id" binding:"required" example:"1"`
	Size         string `json:"size" binding:"omitempty" example:"xl"`
	ColorID      int32  `json:"color" binding:"omitempty" example:"1"`
	Barcode      string `json:"barcode" binding:"omitempty" example:"..."`
	ReorderLevel int32  `json:"reorder_level" binding:"omitempty" example:"5" default:"5"`
	BasePrice    string `json:"base_price" binding:"required" example:"10.99"`
	IsActive     bool   `json:"is_active" binding:"omitempty" default:"true"`
	IsDefault    bool   `json:"is_default" binding:"omitempty" default:"false" example:"false"`
}

type VariationResponse struct {
	ID           int32  `json:"id"`
	ItemID       int32  `json:"item_id"`
	Sku          string `json:"sku"`
	Name         string `json:"name"`
	UnitID       int32  `json:"unit_id"`
	Size         string `json:"size"`
	ColorID      int32  `json:"color_id"`
	Barcode      string `json:"barcode"`
	IsActive     bool   `json:"is_active"`
	ReorderLevel int32  `json:"reorder_level"`
	BasePrice    string `json:"base_price"`
}

func safePrefix(s string, length int) string {
	if len(s) < length {
		return s
	}
	return s[:length]
}

// CreateVariant godoc
// @Summary Create a variant
// @Description Create an item variation. If sku is empty, system autogenerates it.
// @Tags inventory
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 201 {object} VariationResponse
// @Param body body VariationRequest true "variation details"
// @Failure 400
// @Failure 401
// @Failure 403
// @Failure 500
// @Router /api/v1/inventory/variation [post]
func (h *Handler) CreateVariation(c *gin.Context) {
	claims, ok := jwt.GetUserFromContext(c)
	if !ok {
		h.logger.Errorf("could not get user from context")
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	var req VariationRequest
	if err := c.ShouldBind(&req); err != nil {
		h.logger.Errorf("error binding creating business request data: %v", err)
		utils.ErrorResponse(c, 400, utils.INVALID_REQUEST_DATA)
		return
	}

	_, err := h.service.GetItem(c, req.ItemID)
	if err == sql.ErrNoRows {
		utils.ErrorResponse(c, 400, fmt.Sprintf("item with id %d does not exist", req.ItemID))
		return
	} else if err != nil {
		h.logger.Errorf("error fetching item in create variation: %v", err)
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	// INFO: sku will be auto generated if empty
	if req.Sku == "" {
		var brand db.Brand
		item, err := h.service.GetItem(c, req.ItemID)
		if err != nil {
			utils.ErrorResponse(c, 500, err.Error())
			return
		}

		category, err := h.service.GetCategory(c, item.CategoryID)
		if err != nil {
			utils.ErrorResponse(c, 500, err.Error())
			return
		}

		if item.BrandID.Valid && item.BrandID.Int32 != 0 {
			brand, err = h.service.GetBrand(c, item.BrandID.Int32)
			if err != nil {
				h.logger.Errorf("error fetching brand in create variation: %v", err)
				utils.ErrorResponse(c, 500, err.Error())
				return
			}

			req.Sku = fmt.Sprintf("%s-%s-%s-%s",
				safePrefix(category.Name, 3),
				safePrefix(brand.Name, 2),
				safePrefix(item.Name, 2),
				safePrefix(req.Name, 2),
			)
		} else {
			req.Sku = fmt.Sprintf("%s-%s-%s",
				safePrefix(category.Name, 3),
				safePrefix(item.Name, 2),
				safePrefix(req.Name, 2),
			)
		}
	}

	params := db.CreateVariationParams{
		Name:         req.Name,
		ItemID:       req.ItemID,
		Sku:          req.Sku,
		UnitID:       req.UnitID,
		Size:         sql.NullString{String: req.Size, Valid: req.Size != ""},
		ColorID:      sql.NullInt32{Int32: req.ColorID, Valid: req.ColorID != 0},
		Barcode:      sql.NullString{String: req.Barcode, Valid: req.Barcode != ""},
		ReorderLevel: sql.NullInt32{Int32: req.ReorderLevel, Valid: req.ReorderLevel != 0},
		BasePrice:    req.BasePrice,
	}

	variant, err := h.service.CreateVariation(c, params)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok {
			switch pgErr.Code {
			case "23505": // unique_violation
				utils.ErrorResponse(c, 400, fmt.Sprintf("variant with name %s already exists", req.Name))
				return
			}
		}

		h.logger.Errorf("error creating a variant: %v", err)
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	h.service.LogActivity(c, db.LogActivityParams{
		UserID:     int32(claims.UserID),
		EntityID:   variant.ID,
		Action:     "Created Variant",
		EntityType: "Variation",
		Details:    utils.WriteActivityDetails(claims.Username, claims.Email, fmt.Sprintf("Created variant %s", variant.Name), variant.CreatedAt.Time),
		IpAddress:  sql.NullString{Valid: true, String: utils.GetClientIP(c)},
		UserAgent:  sql.NullString{Valid: true, String: c.Request.UserAgent()},
	})

	utils.SuccessResponse(c, 201, "variant created", VariationResponse{
		ID:        variant.ID,
		ItemID:    variant.ItemID,
		Sku:       variant.Sku,
		Name:      variant.Name,
		UnitID:    variant.UnitID,
		Size:      variant.Size.String,
		ColorID:   variant.ColorID.Int32,
		Barcode:   variant.Barcode.String,
		BasePrice: variant.BasePrice,
		IsActive:  variant.IsActive.Bool,
	})
}
