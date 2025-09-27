package business

import (
	"database/sql"
	"encoding/json"
	"fmt"
	db "nucleus/db/sqlc"
	"nucleus/internal/auth"
	"nucleus/internal/config"
	"nucleus/internal/utils"
	"nucleus/pkg/jwt"
	"nucleus/pkg/monitoring/logging"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	"github.com/lib/pq"
	"github.com/sqlc-dev/pqtype"
)

type Handler struct {
	service BusinessInterface
	config  *config.Config
	logger  *logging.Logger
}

func NewBusinessHandler(service BusinessInterface, c *config.Config, l *logging.Logger) *Handler {
	return &Handler{
		service: service,
		config:  c,
		logger:  l,
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authSvc *auth.Service) {
	business := r.Group("/business")
	business.Use(auth.AdminMiddleware(authSvc))
	// Business endpoints
	{
		business.POST("", auth.PermissionMiddleware(authSvc, "business:create"), h.createBusiness)
		business.GET("/:id", auth.PermissionMiddleware(authSvc, "business:view"), h.getBusiness)
		business.PATCH("/:id", auth.PermissionMiddleware(authSvc, "business:update"), h.updateBusiness)
		business.DELETE("/:id", auth.PermissionMiddleware(authSvc, "business:delete"), h.deleteBusiness)
		business.GET("", auth.PermissionMiddleware(authSvc, "business:view"), h.listBusinesses)
	}

	branch := business.Group("/branch")
	{
		branch.POST("", auth.PermissionMiddleware(authSvc, "business:create"), h.createBranch)
		branch.GET("/:id", auth.PermissionMiddleware(authSvc, "business:view"), h.getBranch)
		branch.PUT("/:id", auth.PermissionMiddleware(authSvc, "business:update"), h.updateBranch)
		branch.DELETE("/:id", auth.PermissionMiddleware(authSvc, "business:delete"), h.deleteBranch)
		branch.GET("", auth.PermissionMiddleware(authSvc, "business:view"), h.listBranches)
	}

	supplier := business.Group("/supplier")
	{
		supplier.POST("", auth.PermissionMiddleware(authSvc, "supplier:create"), h.createSupplier)
	}
}

type CreateBusinessParams struct {
	Name              string  `form:"name" example:"Palmwineexpress hotels" binding:"required"`
	Email             *string `form:"email" binding:"omitempty" example:"admin@palmwinexpress.com"`
	Website           *string `form:"website" binding:"omitempty" example:"https://palmwinexpress.com"`
	TaxID             *string `form:"tax_id" binding:"omitempty" example:"123456789"`
	TaxRate           *string `form:"tax_rate" binding:"omitempty" example:"12"`
	LogoUrl           *string `form:"logo_url" binding:"omitempty" example:"https://imgur.com/234343"`
	Rounding          *string `form:"rounding" binding:"omitempty" example:"nearest"`
	Currency          *string `form:"currency" binding:"omitempty" example:"NGN"`
	Timezone          *string `form:"timezone" binding:"omitempty" example:"UTC +1"`
	Language          *string `form:"language" binding:"omitempty" example:"en"`
	LowStockThreshold *int    `form:"low_stock_threshold" binding:"omitempty" example:"5"`
	AllowOverselling  *bool   `form:"allow_overselling" binding:"omitempty" example:"false"`
	Font              *string `form:"font" binding:"omitempty"`
	PrimaryColor      *string `form:"primary_color" binding:"omitempty"`
	Motto             *string `form:"motto" binding:"omitempty"`
	Country           *string `form:"country" binding:"omitempty" example:"Nigeria"`
	// You can extend this params by creating an abitrary number of custom data.
	// "metadata" : { "key": "value"}
	Metadata map[string]any `form:"metadata" binding:"omitempty"`
}

type CreateBusinesswithBranchResponse struct {
	ID                int32  `json:"id"`
	Name              string `json:"name" example:"Palmwineexpress hotels" binding:"required"`
	Email             string `json:"email" binding:"omitempty" example:"admin@palmwinexpress.com"`
	Website           string `json:"website" binding:"omitempty" example:"https://palmwinexpress.com"`
	TaxID             string `json:"tax_id" binding:"omitempty" example:"123456789"`
	TaxRate           string `json:"tax_rate" binding:"omitempty" example:"12"`
	LogoUrl           string `json:"logo_url" binding:"omitempty" example:"https://imgur.com/234343"`
	Rounding          string `json:"rounding" binding:"omitempty" example:"nearest"`
	Currency          string `json:"currency" binding:"omitempty" example:"NGN"`
	Timezone          string `json:"timezone" binding:"omitempty" example:"UTC +1"`
	Language          string `json:"language" binding:"omitempty" example:"en"`
	LowStockThreshold int32  `json:"low_stock_threshold" binding:"omitempty" example:"5"`
	AllowOverselling  bool   `json:"allow_overselling" binding:"omitempty" example:"false"`
	Font              string `json:"font" binding:"omitempty"`
	PrimaryColor      string `json:"primary_color" binding:"omitempty"`
	Motto             string `json:"motto" binding:"omitempty"`
	Country           string `json:"country" binding:"omitempty" example:"Nigeria"`
	Branch            Branch `json:"branch"`
}

type BusinessResponse struct {
	ID                int32          `json:"id"`
	Name              string         `json:"name" example:"Palmwineexpress hotels" binding:"required"`
	Email             string         `json:"email" binding:"omitempty" example:"admin@palmwinexpress.com"`
	Website           string         `json:"website" binding:"omitempty" example:"https://palmwinexpress.com"`
	TaxID             string         `json:"tax_id" binding:"omitempty" example:"123456789"`
	TaxRate           string         `json:"tax_rate" binding:"omitempty" example:"12"`
	LogoUrl           string         `json:"logo_url" binding:"omitempty" example:"https://imgur.com/234343"`
	Rounding          string         `json:"rounding" binding:"omitempty" example:"nearest"`
	Currency          string         `json:"currency" binding:"omitempty" example:"NGN"`
	Timezone          string         `json:"timezone" binding:"omitempty" example:"UTC +1"`
	Language          string         `json:"language" binding:"omitempty" example:"en"`
	LowStockThreshold int32          `json:"low_stock_threshold" binding:"omitempty" example:"5"`
	AllowOverselling  bool           `json:"allow_overselling" binding:"omitempty" example:"false"`
	Font              string         `json:"font" binding:"omitempty"`
	PrimaryColor      string         `json:"primary_color" binding:"omitempty"`
	Motto             string         `json:"motto" binding:"omitempty"`
	Country           string         `json:"country" binding:"omitempty" example:"Nigeria"`
	Metadata          map[string]any `json:"metadata"`
	CreateAt          time.Time      `json:"created_at"`
	UpdateAt          time.Time      `json:"updated_at"`
}

type Branch struct {
	ID         int32  `json:"id"`
	BusinessID int32  `json:"business_id" example:"2" binding:"required"`
	Name       string `json:"name" example:"Main branch" binding:"required"`
	// You can extend this params by creating an abitrary number of custom data.
	// "metadata" : { "key": "value"}
	Metadata map[string]any `json:"metadata"`
}

// CreateBusiness godoc
// @Summary Create a business
// @Description Create a new business with optional logo upload.
// @Tags business
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param name formData string true "Business name"
// @Param email formData string false "Business email"
// @Param website formData string false "Business website"
// @Param tax_id formData string false "Tax ID"
// @Param tax_rate formData string false "Tax Rate"
// @Param currency formData string false "Currency code (e.g. NGN)"
// @Param timezone formData string false "Timezone (e.g. UTC+1)"
// @Param country formData string false "Country"
// @Param payment_type formData []string false "Accepted payment types (e.g. cash,pos,room_charge,transfer)"
// @Param low_stock_threshold formData int false "Low stock threshold"
// @Param allow_overselling formData bool false "Allow overselling"
// @Param motto formData string false "Business motto"
// @Param rounding formData string false "Rounding method (e.g. nearest, up, down)"
// @Param language formData string false "Language (e.g. en, fr, es)"
// @Param logo formData file false "Business logo (JPG/PNG, max 2MB)"
// @Param metadata formData object false "custom data"
// @Success 201 {object} BusinessResponse
// @Failure 400
// @Failure 401
// @Failure 403
// @Failure 500
// @Router /api/v1/business [post]
func (h *Handler) createBusiness(c *gin.Context) {
	claims, ok := jwt.GetUserFromContext(c)
	if !ok {
		h.logger.Errorf("could not get user from context")
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	// Parse form-data (multipart) instead of JSON
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil { // 10MB limit
		utils.ErrorResponse(c, 400, err.Error())
		return
	}

	var req CreateBusinessParams
	if err := c.ShouldBind(&req); err != nil {
		utils.ErrorResponse(c, 400, err.Error())
		return
	}

	// Handle file upload if present
	logoUrl, err := utils.UploadFile(c, "logo", "images", 2<<20) // 2MB max
	if err == nil && logoUrl != "" {
		req.LogoUrl = &logoUrl
	}

	// --- Handle Metadata ---
	if rawMeta := c.PostForm("metadata"); rawMeta != "" {
		var meta map[string]any
		if err := json.Unmarshal([]byte(rawMeta), &meta); err == nil {
			req.Metadata = meta
		} else {
			h.logger.Warnf("invalid metadata json: %v", err)
		}
	}

	var params db.CreateBusinessParams
	err = copier.Copy(&params, &req)
	if err != nil {
		h.logger.Errorf("error copying business request data: %v", err)
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	params.OwnerID = int32(claims.UserID)

	// Marshal metadata for DB
	if req.Metadata != nil {
		if bytes, err := json.Marshal(req.Metadata); err == nil {
			params.Metadata = pqtype.NullRawMessage{
				Valid:      true,
				RawMessage: bytes,
			}
		}
	}

	business, err := h.service.CreateBusiness(c, params)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok {
			switch pgErr.Code {
			case "23505": // unique_violation
				utils.ErrorResponse(c, 400, fmt.Sprintf("business with name %s already exists", req.Name))
				return
			}
		}
		h.logger.Errorf("error creating a business: %v", err)
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	// Log activity
	_, err = h.service.LogActivity(c, db.LogActivityParams{
		UserID:     int32(claims.UserID),
		Action:     "Created business",
		EntityType: "Business",
		EntityID:   business.ID,
		Details:    utils.WriteActivityDetails(claims.Username, claims.Email, fmt.Sprintf("Created business %s", business.Name), business.CreatedAt.Time),
		IpAddress:  sql.NullString{Valid: true, String: utils.GetClientIP(c)},
		UserAgent:  sql.NullString{Valid: true, String: c.Request.UserAgent()},
	})

	if err != nil {
		h.logger.Warnf("error logging activity: %v", err)
		// not returning error to user as business and branch have been created successfully
	}

	var meta map[string]any
	if business.Metadata.Valid {
		_ = json.Unmarshal(business.Metadata.RawMessage, &meta)
	}

	utils.SuccessResponse(c, 201, "Business created", BusinessResponse{
		ID:                business.ID,
		Name:              business.Name,
		Email:             business.Email.String,
		Website:           business.Website.String,
		TaxID:             business.TaxID.String,
		TaxRate:           business.TaxRate.String,
		LogoUrl:           business.LogoUrl.String,
		Rounding:          business.Rounding.String,
		Currency:          business.Currency.String,
		Timezone:          business.Timezone.String,
		Language:          business.Language.String,
		LowStockThreshold: business.LowStockThreshold.Int32,
		AllowOverselling:  business.AllowOverselling.Bool,
		Motto:             business.Motto.String,
		Country:           business.Country,
		Metadata:          meta,
	})

}

// GetBusiness godoc
// @Summary Get business
// @Description Get a business
// @Tags business
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} BusinessResponse
// @Failure 400
// @Failure 401
// @Failure 403
// @Failure 500
// @Router /api/v1/business/:id [get]
func (h *Handler) getBusiness(c *gin.Context) {
	claims, ok := jwt.GetUserFromContext(c)
	if !ok {
		h.logger.Errorf("could not get user from context")
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}
	id := c.Param("id")
	bid, err := strconv.Atoi(id)
	if err != nil {
		h.logger.Errorf("get business id str conv err: %v", err)
		utils.ErrorResponse(c, 400, utils.INVALID_REQUEST_DATA)
		return
	}

	params := db.GetBusinessParams{
		ID:      int32(bid),
		OwnerID: int32(claims.UserID),
	}

	fmt.Printf("business id: %d and owner id: %d", bid, claims.UserID)
	business, err := h.service.GetBusiness(c, params)
	if err != nil {
		h.logger.Errorf("error getting business with is %d: %v", bid, err)
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	utils.SuccessResponse(c, 200, "get business successful", BusinessResponse{
		ID:       business.ID,
		Name:     business.Name,
		Motto:    business.Motto.String,
		Email:    business.Email.String,
		Website:  business.Website.String,
		TaxID:    business.TaxID.String,
		TaxRate:  business.TaxRate.String,
		LogoUrl:  business.LogoUrl.String,
		Rounding: business.Rounding.String,
		Currency: business.Currency.String,
		Timezone: business.Timezone.String,
		Language: business.Language.String,
		Metadata: utils.UnmarshalMetadata(business.Metadata),
		CreateAt: business.CreatedAt.Time,
		UpdateAt: business.UpdatedAt.Time,
	})
}

type UpdateBusinessRequest struct {
	Name              *string        `json:"name"`
	Motto             *string        `json:"motto"`
	Email             *string        `json:"email"`
	Website           *string        `json:"website"`
	TaxID             *string        `json:"tax_id"`
	TaxRate           *string        `json:"tax_rate"`
	LogoUrl           *string        `json:"logo_url"`
	Rounding          *string        `json:"rounding"`
	Currency          *string        `json:"currency"`
	Timezone          *string        `json:"timezone"`
	Language          *string        `json:"language"`
	LowStockThreshold *int32         `json:"low_stock_threshold"`
	AllowOverselling  *bool          `json:"allow_overselling"`
	Country           *string        `json:"country"`
	Metadata          map[string]any `json:"metadata"`
}

type UpdateBusinessResponse struct {
	ID                int32          `json:"id"`
	Name              string         `json:"name"`
	Motto             string         `json:"motto"`
	Email             string         `json:"email"`
	Website           string         `json:"website"`
	TaxID             string         `json:"tax_id"`
	TaxRate           string         `json:"tax_rate"`
	LogoUrl           string         `json:"logo_url"`
	Rounding          string         `json:"rounding"`
	Currency          string         `json:"currency"`
	Timezone          string         `json:"timezone"`
	Language          string         `json:"language"`
	LowStockThreshold int32          `json:"low_stock_threshold"`
	AllowOverselling  bool           `json:"allow_overselling"`
	Country           string         `json:"country"`
	Metadata          map[string]any `json:"metadata"`
}

// UpdateBusiness godoc
// @Summary Update a business
// @Description Update a business
// @Tags business
// @Accept json
// @Produce json
// @Param id path int true "Business ID"
// @Param business body UpdateBusinessRequest true "Business"
// @Success 200 {object} UpdateBusinessResponse
// @Failure 400
// @Failure 403
// @Failure 404
// @Failure 500
// @Router /business/{id} [patch]
func (h *Handler) updateBusiness(c *gin.Context) {
	// Get current user
	claims, ok := jwt.GetUserFromContext(c)
	if !ok {
		h.logger.Errorf("could not get user from context")
		utils.ErrorResponse(c, 500, "you are not logged in")
		return
	}

	// Parse business ID
	bid, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.logger.Errorf("invalid business id: %v", err)
		utils.ErrorResponse(c, 400, err.Error())
		return
	}

	// Bind request
	var req UpdateBusinessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("error binding business update: %v", err)
		utils.ErrorResponse(c, 400, err.Error())
		return
	}

	// Ensure the business exists and belongs to this user
	getParams := db.GetBusinessParams{
		ID:      int32(bid),
		OwnerID: int32(claims.UserID),
	}
	b, err := h.service.GetBusiness(c, getParams)
	if err != nil {
		h.logger.Errorf("get business by id err: %v", err)
		utils.ErrorResponse(c, 404, "Business not found or not owned by you")
		return
	}

	updateParams := db.UpdateBusinessParams{
		ID:       int32(bid),
		OwnerID:  int32(claims.UserID),
		Metadata: b.Metadata,
	}

	// Patch optional fields
	utils.PatchNullString(&updateParams.Name, req.Name)
	utils.PatchNullString(&updateParams.Motto, req.Motto)
	utils.PatchNullString(&updateParams.Email, req.Email)
	utils.PatchNullString(&updateParams.Website, req.Website)
	utils.PatchNullString(&updateParams.TaxID, req.TaxID)
	utils.PatchNullString(&updateParams.TaxRate, req.TaxRate)
	utils.PatchNullString(&updateParams.LogoUrl, req.LogoUrl)
	utils.PatchNullString(&updateParams.Rounding, req.Rounding)
	utils.PatchNullString(&updateParams.Currency, req.Currency)
	utils.PatchNullString(&updateParams.Timezone, req.Timezone)
	utils.PatchNullString(&updateParams.Language, req.Language)
	utils.PatchNullInt32(&updateParams.LowStockThreshold, req.LowStockThreshold)
	utils.PatchNullBool(&updateParams.AllowOverselling, req.AllowOverselling)
	utils.PatchMetadata(&updateParams.Metadata, req.Metadata)
	// Update the business
	updatedBusiness, err := h.service.UpdateBusiness(c, updateParams)
	if err != nil {
		h.logger.Errorf("could not update business: %v", err)
		utils.ErrorResponse(c, 500, err.Error())
		return
	}

	// Log activity
	_, err = h.service.LogActivity(c, db.LogActivityParams{
		UserID:    int32(claims.UserID),
		Action:    "update_business",
		Details:   utils.WriteActivityDetails(claims.Username, claims.Email, "update business", updatedBusiness.CreatedAt.Time),
		IpAddress: sql.NullString{Valid: true, String: utils.GetClientIP(c)},
		UserAgent: sql.NullString{Valid: true, String: c.Request.UserAgent()},
	})

	utils.SuccessResponse(c, 200, "Business updated", UpdateBusinessResponse{
		ID:                updatedBusiness.ID,
		Name:              updatedBusiness.Name,
		Email:             updatedBusiness.Email.String,
		Country:           updatedBusiness.Country,
		Timezone:          updatedBusiness.Timezone.String,
		Language:          updatedBusiness.Language.String,
		LowStockThreshold: updatedBusiness.LowStockThreshold.Int32,
		AllowOverselling:  updatedBusiness.AllowOverselling.Bool,
		Motto:             updatedBusiness.Motto.String,
		Website:           updatedBusiness.Website.String,
		TaxID:             updatedBusiness.TaxID.String,
		TaxRate:           updatedBusiness.TaxRate.String,
		LogoUrl:           updatedBusiness.LogoUrl.String,
		Rounding:          updatedBusiness.Rounding.String,
		Currency:          updatedBusiness.Currency.String,
		Metadata:          utils.UnmarshalMetadata(updatedBusiness.Metadata),
	})
}

// DeleteBusiness godoc
// @Summary Delete business
// @Description Delete a new business
// @Tags business
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {string} string "business deleted"
// @Failure 400
// @Failure 401
// @Failure 403
// @Failure 500
// @Router /api/v1/business/:id [delete]
func (h *Handler) deleteBusiness(c *gin.Context) {
	claims, ok := jwt.GetUserFromContext(c)
	if !ok {
		h.logger.Errorf("could not get user from context")
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	id := c.Param("id")
	bid, err := strconv.Atoi(id)
	if err != nil {
		h.logger.Errorf("get business id str conv err: %v", err)
		utils.ErrorResponse(c, 400, utils.INVALID_REQUEST_DATA)
		return
	}

	params := db.DeleteBusinessParams{
		ID:      int32(bid),
		OwnerID: int32(claims.UserID),
	}

	business, err := h.service.DeleteBusiness(c, params)
	if err != nil {
		h.logger.Errorf("error deleteing business with is %d: %v", bid, err)
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	// Log activity
	_, err = h.service.LogActivity(c, db.LogActivityParams{
		UserID:     int32(claims.UserID),
		Action:     "Deleted business",
		EntityType: "Business",
		EntityID:   business.ID,
		Details:    utils.WriteActivityDetails(claims.Username, claims.Email, fmt.Sprintf("Deleted business %s", business.Name), business.CreatedAt.Time),
		IpAddress:  sql.NullString{Valid: true, String: utils.GetClientIP(c)},
		UserAgent:  sql.NullString{Valid: true, String: c.Request.UserAgent()},
	})

	if err != nil {
		h.logger.Warnf("error logging activity: %v", err)
		// not returning error to user as business and branch have been created successfully
	}

	utils.SuccessResponse(c, 200, "business deleted", nil)
}

type ListBusinessResponse struct {
	ID                int32          `json:"id"`
	Name              string         `json:"name"`
	Motto             string         `json:"motto"`
	Email             string         `json:"email"`
	Website           string         `json:"website"`
	TaxID             string         `json:"tax_id"`
	TaxRate           string         `json:"tax_rate"`
	Country           string         `json:"country"`
	LogoUrl           string         `json:"logo_url"`
	Rounding          string         `json:"rounding"`
	Currency          string         `json:"currency"`
	Timezone          string         `json:"timezone"`
	Language          string         `json:"language"`
	LowStockThreshold int32          `json:"low_stock_threshold"`
	AllowOverselling  bool           `json:"allow_overselling"`
	Metadata          map[string]any `json:"metadata"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

// ListBusinesses godoc
// @Summary Get a list of businesses
// @Description Get a list of businesses
// @Tags business
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} []ListBusinessResponse
// @Failure 400
// @Failure 401
// @Failure 403
// @Failure 500
// @Router /api/v1/business [get]
func (h *Handler) listBusinesses(c *gin.Context) {
	claims, ok := jwt.GetUserFromContext(c)
	if !ok {
		h.logger.Errorf("could not get user from context")
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	businesses, err := h.service.ListBusinesses(c, int32(claims.UserID))
	if err != nil {
		h.logger.Errorf("error listing businesses: %v", err)
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	var response []ListBusinessResponse
	for _, business := range businesses {

		response = append(response, ListBusinessResponse{
			ID:                business.ID,
			Name:              business.Name,
			Email:             business.Email.String,
			Website:           business.Website.String,
			Motto:             business.Motto.String,
			TaxID:             business.TaxID.String,
			TaxRate:           business.TaxRate.String,
			LogoUrl:           business.LogoUrl.String,
			Language:          business.Language.String,
			Currency:          business.Currency.String,
			Rounding:          business.Rounding.String,
			Timezone:          business.Timezone.String,
			LowStockThreshold: business.LowStockThreshold.Int32,
			AllowOverselling:  business.AllowOverselling.Bool,
			Country:           business.Country,
			Metadata:          utils.UnmarshalMetadata(business.Metadata),
			CreatedAt:         business.CreatedAt.Time,
			UpdatedAt:         business.UpdatedAt.Time,
		})
	}

	utils.SuccessResponse(c, 200, "A list of your businesses", response)
}

type CreateBranchRequest struct {
	BusinessID int32          `json:"business_id" example:"2" binding:"required"`
	Name       string         `json:"name" example:"Main branch" binding:"required"`
	AddressOne string         `json:"address_one" binding:"required" example:"..."`
	AddresTwo  string         `json:"addres_two" binding:"omitempty" example:"1 Plamwine express"`
	Country    string         `json:"country" binding:"required" example:"Nigeria"`
	Phone      string         `json:"phone" binding:"omitempty" example:"+2349028378964"`
	Email      string         `json:"email" binding:"omitempty" example:"admin.mainbranch@gmail.com"`
	Website    string         `json:"website" binding:"omitempty" example:"https://"`
	City       string         `json:"city" binding:"omitempty" example:"aba"`
	State      string         `json:"state" binding:"omitempty" example:"abia"`
	ZipCode    string         `json:"zip_code" binding:"omitempty" example:"..."`
	Metadata   map[string]any `json:"metadata"`
}

type CreateBranchResponse struct {
	ID         int32          `json:"id"`
	BusinessID int32          `json:"business_id" example:"2" binding:"required"`
	Name       string         `json:"name" example:"Main branch" binding:"required"`
	AddressOne string         `json:"address_one" binding:"required" example:"..."`
	AddresTwo  string         `json:"addres_two" binding:"omitempty" example:"1 Plamwine express"`
	Country    string         `json:"country" binding:"required" example:"Nigeria"`
	Phone      string         `json:"phone" binding:"omitempty" example:"+2349028378964"`
	Email      string         `json:"email" binding:"omitempty" example:""`
	Website    string         `json:"website" binding:"omitempty" example:"https://"`
	City       string         `json:"city" binding:"omitempty" example:"aba"`
	State      string         `json:"state" binding:"omitempty" example:"abia"`
	ZipCode    string         `json:"zip_code" binding:"omitempty" example:"..."`
	Metadata   map[string]any `json:"metadata"`
}

// CreateBranch godoc
// @Summary Create a branch
// @Description Create a branch. A business must have atleast one branch.
// @Tags business
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param business body CreateBranchRequest true "Branch details"
// @Success 200 {object} CreateBranchResponse
// @Failure 400
// @Failure 401
// @Failure 403
// @Failure 500
// @Router /api/v1/business/branch [post]
func (h *Handler) createBranch(c *gin.Context) {
	claims, ok := jwt.GetUserFromContext(c)
	if !ok {
		h.logger.Errorf("could not get user from context")
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	var req CreateBranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("create branch request binding error: %v", err)
		utils.ErrorResponse(c, 400, utils.INVALID_REQUEST_DATA)
		return
	}

	var params db.CreateBranchParams
	err := copier.Copy(&params, &req)
	if err != nil {
		h.logger.Errorf("error copying create branch request data: %v", err)
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	getParams := db.GetBusinessParams{
		ID:      int32(req.BusinessID),
		OwnerID: int32(claims.UserID),
	}

	// check if business exists
	_, err = h.service.GetBusiness(c, getParams)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.ErrorResponse(c, 400, fmt.Sprintf("business with id %d does not exist", req.BusinessID))
			return
		}
		h.logger.Errorf("error getting business with id %d: %v", req.BusinessID, err)
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	// convert map[string]any to JSON
	params.Metadata = utils.MarshalMetadata(req.Metadata)

	branch, err := h.service.CreateBranch(c, params)
	if err != nil {
		h.logger.Errorf("error creating branch: %v", err)
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	// add activity log here
	_, err = h.service.LogActivity(c, db.LogActivityParams{
		UserID:     int32(claims.UserID),
		Action:     "Created branch",
		EntityType: "Branch",
		EntityID:   branch.ID,
		Details:    utils.WriteActivityDetails("system", "system", fmt.Sprintf("Created branch %s for business id %d", branch.Name, branch.BusinessID), branch.CreatedAt.Time),
		IpAddress:  sql.NullString{Valid: true, String: utils.GetClientIP(c)},
		UserAgent:  sql.NullString{Valid: true, String: c.Request.UserAgent()},
	})

	if err != nil {
		h.logger.Warnf("error logging activity: %v", err)
		// not returning error to user as branch has been created successfully
	}

	utils.SuccessResponse(c, 201, "branch created", CreateBranchResponse{
		BusinessID: branch.BusinessID,
		Name:       branch.Name,
		AddressOne: branch.AddressOne.String,
		AddresTwo:  branch.AddressTwo.String,
		Country:    branch.Country.String,
		Phone:      branch.Phone.String,
		Email:      branch.Email.String,
		City:       branch.City.String,
		State:      branch.State.String,
		ZipCode:    branch.ZipCode.String,
		Metadata:   utils.UnmarshalMetadata(branch.Metadata),
	})
}

// GetBranch godoc
// @Summary fetch a branch
// @Description Fetch a branch.
// @Tags business
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200
// @Failure 400
// @Failure 401
// @Failure 403
// @Failure 500
// @Router /api/v1/business/:id [get]
func (h *Handler) getBranch(c *gin.Context) {
	id := c.Param("id")
	bid, err := strconv.Atoi(id)
	if err != nil {
		h.logger.Errorf("get branch id str conv err: %v", err)
		utils.ErrorResponse(c, 400, utils.INVALID_REQUEST_DATA)
		return
	}

	branch, err := h.service.GetBranch(c, int32(bid))
	if err != nil {
		h.logger.Errorf("error getting branch with is %d: %v", bid, err)
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	utils.SuccessResponse(c, 200, "get branch successful", branch)
}

type UpdateBranchRequest struct {
	Name       string `json:"name" example:"Main branch" binding:"required"`
	AddressOne string `json:"address_one" binding:"required" example:"..."`
	AddresTwo  string `json:"addres_two" binding:"omitempty" example:"1 Plamwine express"`
	Country    string `json:"country" binding:"required" example:"Nigeria"`
	Phone      string `json:"phone" binding:"omitempty" example:"+2349028378964"`
	Email      string `json:"email" binding:"omitempty" example:""`
	Website    string `json:"website" binding:"omitempty" example:"https://"`
	City       string `json:"city" binding:"omitempty" example:"aba"`
	State      string `json:"state" binding:"omitempty" example:"abia"`
	ZipCode    string `json:"zip_code" binding:"omitempty" example:"..."`
}

// UpdateBranch godoc
// @Summary Update a branch
// @Description Update a branch
// @Tags business
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Branch ID"
// @Param branch body UpdateBranchRequest true "Branch details"
// @Success 200 {object} CreateBranchResponse
// @Failure 400
// @Failure 401
// @Failure 403
// @Failure 500
// @Router /api/v1/business/branch/{id} [put]
func (h *Handler) updateBranch(c *gin.Context) {
	claims, ok := jwt.GetUserFromContext(c)
	if !ok {
		h.logger.Errorf("could not get user from context")
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	id := c.Param("id")
	_, err := strconv.Atoi(id)
	if err != nil {
		h.logger.Errorf("get branch id str conv err: %v", err)
		utils.ErrorResponse(c, 400, utils.INVALID_REQUEST_DATA)
		return
	}

	var req UpdateBranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("update branch request binding error: %v", err)
		utils.ErrorResponse(c, 400, utils.INVALID_REQUEST_DATA)
		return
	}

	updateParams := db.UpdateBranchParams{
		Name:    sql.NullString{String: req.Name, Valid: true},
		Country: sql.NullString{String: req.Country, Valid: true},
		Phone:   sql.NullString{String: req.Phone, Valid: req.Phone != ""},
		Email:   sql.NullString{String: req.Email, Valid: req.Email != ""},
		City:    sql.NullString{String: req.City, Valid: req.City != ""},
		State:   sql.NullString{String: req.State, Valid: req.State != ""},
		ZipCode: sql.NullString{String: req.ZipCode, Valid: req.ZipCode != ""},
	}

	branch, err := h.service.UpdateBranch(c, updateParams)
	if err != nil {
		h.logger.Errorf("error updating branch: %v", err)
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	// add activity log here
	_, err = h.service.LogActivity(c, db.LogActivityParams{
		UserID:     int32(claims.UserID),
		Action:     "Updated branch",
		EntityType: "Branch",
		EntityID:   branch.ID,
		Details:    utils.WriteActivityDetails(claims.Username, claims.Email, fmt.Sprintf("Updated branch %s for business id %d", branch.Name, branch.BusinessID), branch.UpdatedAt.Time),
		IpAddress:  sql.NullString{Valid: true, String: utils.GetClientIP(c)},
		UserAgent:  sql.NullString{Valid: true, String: c.Request.UserAgent()},
	})

	if err != nil {
		h.logger.Warnf("error logging activity: %v", err)
		// not returning error to user as branch has been created successfully
	}

	utils.SuccessResponse(c, 200, "branch updated", branch)

}

// DeleteBranch godoc
// @Summary Delete a branch
// @Description Delete a branch.
// @Tags business
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param business body CreateBranchRequest true "Branch details"
// @Success 200 {object} CreateBranchResponse
// @Failure 400
// @Failure 401
// @Failure 403
// @Failure 500
// @Router /api/v1/business/branch [post]
func (h *Handler) deleteBranch(c *gin.Context) {
	claims, ok := jwt.GetUserFromContext(c)
	if !ok {
		h.logger.Errorf("could not get user from context")
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	id := c.Param("id")
	bid, err := strconv.Atoi(id)
	if err != nil {
		h.logger.Errorf("get branch id str conv err: %v", err)
		utils.ErrorResponse(c, 400, utils.INVALID_REQUEST_DATA)
		return
	}

	branch, err := h.service.DeleteBranch(c, int32(bid))
	if err != nil {
		h.logger.Errorf("error deleting branch with is %d: %v", bid, err)
		utils.ErrorResponse(c, 500, utils.SERVERERROR)
		return
	}

	utils.SuccessResponse(c, 200, "branch deleted", nil)

	// Log activity
	_, err = h.service.LogActivity(c, db.LogActivityParams{
		UserID:     int32(claims.UserID),
		Action:     "Deleted branch",
		EntityType: "Branch",
		EntityID:   branch.ID,
		Details:    utils.WriteActivityDetails(claims.Username, claims.Email, fmt.Sprintf("Deleted branch %s", branch.Name), branch.CreatedAt.Time),
		IpAddress:  sql.NullString{Valid: true, String: utils.GetClientIP(c)},
		UserAgent:  sql.NullString{Valid: true, String: c.Request.UserAgent()},
	})

	if err != nil {
		h.logger.Warnf("error logging activity: %v", err)
		// not returning error to user as business and branch have been created successfully
	}

	utils.SuccessResponse(c, 200, "branch deleted", nil)

}

func (h *Handler) listBranches(c *gin.Context) {
	// Implementation goes here
}

type Supplier struct {
	ID         int32          `json:"id"`
	Name       string         `json:"name" binding:"required" example:"..."`
	Phone      string         `json:"phone"`
	Email      string         `json:"email"`
	Address    string         `json:"address"`
	Metadata   map[string]any `json:"metadata"`
	BusinessID int32          `json:"business_id"`
	// CreatedAt  time.Time      `json:"created_at"`
	// UpdatedAt  time.Time      `json:"updated_at"`
}

// createSupplier godoc
// @Summary Create a supplier
// @Description Create a supplier
// @Tags supplier
// @Accept json
// @Produce json
// @Security BearerAuth
// @Params body body {object} Supplier true "supplier details"
// @Success 200 {object} Supplier
// @Failure 400
// @Failure 401
// @Failure 403
// @Failure 500
// @Router /api/v1/business/supplier [post]
func (h *Handler) createSupplier(c *gin.Context) {
	claims, ok := jwt.GetUserFromContext(c)
	if !ok {
		utils.ErrorResponse(c, 500, "not logged in")
		return
	}

	var req Supplier
	if err := c.ShouldBind(&req); err != nil {
		utils.ErrorResponse(c, 400, err.Error())
		return
	}

	_, err := h.service.GetBusiness(c, db.GetBusinessParams{
		ID:      req.BusinessID,
		OwnerID: int32(claims.UserID),
	})

	if err != nil {
		utils.ErrorResponse(c, 400, err.Error())
		return
	}

	var params db.CreateSupplierParams
	err = copier.Copy(&params, &req)
	if err != nil {
		utils.ErrorResponse(c, 500, err.Error())
		return
	}

	params.Metadata = utils.MarshalMetadata(req.Metadata)

	supplier, err := h.service.CreateSupplier(c, params)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok {
			switch pgErr.Code {
			case "23505": // unique_violation
				utils.ErrorResponse(c, 400, fmt.Sprintf("supplier with name %s already exists", req.Name))
				return
			}
		}
		utils.ErrorResponse(c, 500, err.Error())
		return
	}

	_, err = h.service.LogActivity(c, db.LogActivityParams{
		UserID:     int32(claims.UserID),
		Action:     "Created supplier",
		EntityType: "Supplier",
		EntityID:   supplier.ID,
		Details:    utils.WriteActivityDetails(claims.Username, claims.Email, fmt.Sprintf("Created supplier %s", supplier.Name), supplier.CreatedAt.Time),
		IpAddress:  sql.NullString{Valid: true, String: utils.GetClientIP(c)},
		UserAgent:  sql.NullString{Valid: true, String: c.Request.UserAgent()},
	})

	if err != nil {
		h.logger.Warnf("error logging activity: %v", err)
	}

	meta := utils.UnmarshalMetadata(supplier.Metadata)

	utils.SuccessResponse(c, 201, "Supplier created", Supplier{
		ID:         supplier.ID,
		Name:       supplier.Name,
		Phone:      supplier.Phone.String,
		Email:      supplier.Email.String,
		Address:    supplier.Address.String,
		BusinessID: supplier.BusinessID,
		Metadata:   meta,
	})

}
