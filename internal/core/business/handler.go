package business

import (
	"database/sql"
	"encoding/json"
	"fmt"
	db "nucleus/db/sqlc"
	"nucleus/internal/auth"
	"nucleus/internal/config"
	"nucleus/internal/core/api"
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
	business := r.Group("/org")
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
	Name      string         `form:"name" example:"Palmwineexpress hotels" binding:"required"`
	Email     string         `form:"email" binding:"required" example:"admin@palmwinexpress.com"`
	Website   string         `form:"website" binding:"omitempty" example:"https://palmwinexpress.com"`
	TaxID     string         `form:"tax_id" binding:"omitempty" example:"123456789"`
	LogoUrl   string         `form:"logo_url" binding:"omitempty" example:"https://imgur.com/234343"`
	Motto     string         `form:"motto" binding:"omitempty"`
	VatNumber string         `form:"vat_number"`
	Country   string         `form:"country" binding:"required"`
	Metadata  map[string]any `form:"metadata" binding:"omitempty"`
}

// BusinessResponse represents the response returned after creating or fetching a business.
// It includes core business details, optional fields, metadata, and timestamps.
type BusinessResponse struct {
	ID       int32          `json:"id" example:"1"`                                         // Unique business ID
	Name     string         `json:"name" example:"Palmwineexpress Hotels"`                  // Business name
	Motto    string         `json:"motto,omitempty" example:"Your comfort, our pride"`      // Optional business motto
	Email    string         `json:"email,omitempty" example:"admin@palmwinexpress.com"`     // Business email address
	Website  string         `json:"website,omitempty" example:"https://palmwinexpress.com"` // Business website URL
	TaxID    string         `json:"tax_id,omitempty" example:"123456789"`                   // Tax Identification Number
	Country  string         `json:"country" example:"Nigeria"`                              // Country where the business is registered
	LogoUrl  string         `json:"logo_url,omitempty" example:"https://imgur.com/234343"`  // URL of the uploaded business logo
	Metadata map[string]any `json:"metadata,omitempty"`                                     // Custom metadata in JSON format
	CreateAt time.Time      `json:"created_at" example:"2025-09-29T12:00:00Z"`              // Timestamp of business creation (UTC)
}

type Branch struct {
	ID         int32          `json:"id"`
	BusinessID int32          `json:"business_id" example:"2" binding:"required"`
	Name       string         `json:"name" example:"Main branch" binding:"required"`
	Metadata   map[string]any `json:"metadata"`
}

type OrgWithBranchResponse struct {
	Org    BusinessResponse `json:"org"`
	Branch Branch           `json:"branch"`
}

// CreateOrg godoc
// @Summary Create an organization
// @Description Create a new business/organization record and link it to the authenticated user as the owner.
// @Description
// @Description ### Features
// @Description - Accepts required and optional business details (name, email, website, tax ID, motto, etc.).
// @Description - Allows uploading a business logo (JPG/PNG, max 2MB).
// @Description - Supports attaching custom metadata as a JSON object.
// @Description - Automatically sets the authenticated user as the owner of the business.
// @Description - Logs the creation activity for auditing purposes.
// @Description
// @Description ### Notes
// @Description - A default branch called "Main" is created automatically when creating an org.
// @Description - The `metadata` field must be a valid JSON string. Example: {"industry":"Hospitality","branches":5}
// @Description - The `logo` field must be a JPG or PNG image not exceeding 2MB.
// @Description - On success, the response includes full business details, metadata, and timestamps.
// @Tags Organization
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param name formData string true "Business name"
// @Param email formData string false "Business email"
// @Param website formData string false "Business website"
// @Param tax_id formData string false "Tax ID"
// @Param motto formData string false "Motto"
// @Param country formData string true "Country of business"
// @Param vat_number formData string false "VAT Number"
// @Param logo formData file false "Business logo (JPG/PNG, max 2MB)"
// @Param metadata formData string false "Custom metadata in JSON format"
// @Description {
// @Description   "industry": "Hospitality",
// @Description   "branches": 5,
// @Description   "subscription": "premium"
// @Description }
// @Success 201 {object} OrgWithBranchResponse
// @Failure 400
// @Failure 401
// @Failure 403
// @Failure 500
// @Router /api/v1/org [post]
func (h *Handler) createBusiness(c *gin.Context) {
	claims, ok := jwt.GetUserFromContext(c)
	if !ok {
		h.logger.Errorf("could not get user from context")
		api.ErrorResponse(c, 500, api.SERVERERROR)
		return
	}

	// Parse form-data (multipart) instead of JSON
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil { // 10MB limit
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	var req CreateBusinessParams
	if err := c.ShouldBind(&req); err != nil {
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	// Handle file upload if present
	logoUrl, err := utils.UploadFile(c, "logo", "images", 2<<20) // 2MB max
	if err == nil && logoUrl != "" {
		req.LogoUrl = logoUrl
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
		api.ErrorResponse(c, 500, err.Error())
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

	h.logger.Infof("creating business %s", params.Name)

	business, branch, err := h.service.CreateBusinessWithBranch(c, params)
	if err != nil {
		h.logger.Errorf("error creating a business: %v", err)
		api.ErrorResponse(c, 500, err.Error())
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

	api.SuccessResponse(c, 201, "Business created", OrgWithBranchResponse{
		Org: BusinessResponse{
			ID:       business.ID,
			Name:     business.Name,
			Email:    business.Email.String,
			Website:  business.Website.String,
			TaxID:    business.TaxID.String,
			LogoUrl:  business.LogoUrl.String,
			Motto:    business.Motto.String,
			Country:  business.Country,
			Metadata: meta,
			CreateAt: business.CreatedAt.Time,
		},
		Branch: Branch{
			ID:         branch.ID,
			BusinessID: branch.BusinessID,
			Name:       branch.Name,
			Metadata:   utils.UnmarshalMetadata(branch.Metadata),
		},
	})

}

// GetOrg godoc
// @Summary Get an organization
// @Description Retrieve details of a specific business by its ID.
// @Description The authenticated user must be the owner of the business.
// @Tags Organization
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Business ID"
// @Success 200 {string} BusinessResponse "Business retrieved successfully"
// @Failure 400 {string} api.BADREQUEST "Invalid business ID supplied"
// @Failure 401 {string} api.UNAUTHORIZED "Unauthorized – missing or invalid JWT"
// @Failure 403 {string} api.FORBIDDEN "Forbidden – user does not have access"
// @Failure 404 {string} api.NOTFOUND "Business not found"
// @Failure 500 {string} api.SERVERERROR "Internal server error"
// @Router /api/v1/org/{id} [get]
func (h *Handler) getBusiness(c *gin.Context) {
	claims, ok := jwt.GetUserFromContext(c)
	if !ok {
		h.logger.Errorf("could not get user from context")
		api.ErrorResponse(c, 500, api.SERVERERROR)
		return
	}
	id := c.Param("id")
	bid, err := strconv.Atoi(id)
	if err != nil {
		h.logger.Errorf("get business id str conv err: %v", err)
		api.ErrorResponse(c, 400, err.Error())
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
		api.ErrorResponse(c, 500, err.Error())
		return
	}

	api.SuccessResponse(c, 200, "get business successful", BusinessResponse{
		ID:       business.ID,
		Name:     business.Name,
		Motto:    business.Motto.String,
		Email:    business.Email.String,
		Website:  business.Website.String,
		TaxID:    business.TaxID.String,
		LogoUrl:  business.LogoUrl.String,
		Metadata: utils.UnmarshalMetadata(business.Metadata),
		CreateAt: business.CreatedAt.Time,
	})
}

type UpdateBusinessRequest struct {
	Name     *string        `json:"name"`
	Motto    *string        `json:"motto"`
	Email    *string        `json:"email"`
	Website  *string        `json:"website"`
	TaxID    *string        `json:"tax_id"`
	LogoUrl  *string        `json:"logo_url"`
	Country  *string        `json:"country"`
	Metadata map[string]any `json:"metadata"`
}

type UpdateBusinessResponse struct {
	ID       int32          `json:"id"`
	Name     string         `json:"name"`
	Motto    string         `json:"motto"`
	Email    string         `json:"email"`
	Website  string         `json:"website"`
	TaxID    string         `json:"tax_id"`
	LogoUrl  string         `json:"logo_url"`
	Country  string         `json:"country"`
	Metadata map[string]any `json:"metadata"`
}

// UpdateOrg godoc
// @Summary Update an organization
// @Description Update an existing business by ID. Only the owner of the business can perform this action.
// Fields are optional — only provided fields will be updated.
// @Tags Organization
// @Accept json
// @Produce json
// @Param id path int true "Business ID"
// @Param business body UpdateBusinessRequest true "Business update payload"
// @Success 200 {object} UpdateBusinessResponse "Business successfully updated"
// @Failure 400 "Invalid request data"
// @Failure 403 "Forbidden: You do not own this business"
// @Failure 404 "Business not found"
// @Failure 500 "Internal server error"
// @Router /api/v1/org/{id} [patch]
// @Security BearerAuth
func (h *Handler) updateBusiness(c *gin.Context) {
	// Get current user
	claims, ok := jwt.GetUserFromContext(c)
	if !ok {
		h.logger.Errorf("could not get user from context")
		api.ErrorResponse(c, 500, "you are not logged in")
		return
	}

	// Parse business ID
	bid, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.logger.Errorf("invalid business id: %v", err)
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	// Bind request
	var req UpdateBusinessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("error binding business update: %v", err)
		api.ErrorResponse(c, 400, err.Error())
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
		api.ErrorResponse(c, 404, "Business not found or not owned by you")
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
	utils.PatchNullString(&updateParams.LogoUrl, req.LogoUrl)
	utils.PatchMetadata(&updateParams.Metadata, req.Metadata)
	// Update the business
	updatedBusiness, err := h.service.UpdateBusiness(c, updateParams)
	if err != nil {
		h.logger.Errorf("could not update business: %v", err)
		api.ErrorResponse(c, 500, err.Error())
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
	if err != nil {
		h.logger.Warnf("error logging activity: %v", err)
	}

	api.SuccessResponse(c, 200, "Business updated", UpdateBusinessResponse{
		ID:       updatedBusiness.ID,
		Name:     updatedBusiness.Name,
		Email:    updatedBusiness.Email.String,
		Country:  updatedBusiness.Country,
		Motto:    updatedBusiness.Motto.String,
		Website:  updatedBusiness.Website.String,
		TaxID:    updatedBusiness.TaxID.String,
		LogoUrl:  updatedBusiness.LogoUrl.String,
		Metadata: utils.UnmarshalMetadata(updatedBusiness.Metadata),
	})
}

// DeleteOrg godoc
// @Summary Delete an organization
// @Description Permanently delete a business owned by the authenticated user.
// Only the owner of the business can perform this action.
// @Tags Organization
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Business ID"
// @Success 200 {object} map[string]string "Business deleted successfully"
// @Failure 400 "Invalid business ID"
// @Failure 401 "Unauthorized: missing or invalid token"
// @Failure 403 "Forbidden: you do not own this business"
// @Failure 404 "Business not found"
// @Failure 500 "Internal server error"
// @Router /api/v1/org/{id} [delete]
func (h *Handler) deleteBusiness(c *gin.Context) {
	claims, ok := jwt.GetUserFromContext(c)
	if !ok {
		h.logger.Errorf("could not get user from context")
		api.ErrorResponse(c, 500, api.SERVERERROR)
		return
	}

	id := c.Param("id")
	bid, err := strconv.Atoi(id)
	if err != nil {
		h.logger.Errorf("get business id str conv err: %v", err)
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	params := db.DeleteBusinessParams{
		ID:      int32(bid),
		OwnerID: int32(claims.UserID),
	}

	business, err := h.service.DeleteBusiness(c, params)
	if err != nil {
		h.logger.Errorf("error deleteing business with is %d: %v", bid, err)
		api.ErrorResponse(c, 500, err.Error())
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

	api.SuccessResponse(c, 200, "business deleted", nil)
}

type ListBusinessResponse struct {
	ID        int32          `json:"id"`
	Name      string         `json:"name"`
	Motto     string         `json:"motto"`
	Email     string         `json:"email"`
	Website   string         `json:"website"`
	TaxID     string         `json:"tax_id"`
	Country   string         `json:"country"`
	LogoUrl   string         `json:"logo_url"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// ListOrgs godoc
// @Summary List organizations
// @Description Retrieve all businesses owned by the authenticated user.
// @Tags Organization
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} ListBusinessResponse "List of businesses"
// @Failure 400 "Invalid request"
// @Failure 401 "Unauthorized: missing or invalid token"
// @Failure 403 "Forbidden: you do not have access"
// @Failure 500 "Internal server error"
// @Router /api/v1/org [get]
func (h *Handler) listBusinesses(c *gin.Context) {
	claims, ok := jwt.GetUserFromContext(c)
	if !ok {
		h.logger.Errorf("could not get user from context")
		api.ErrorResponse(c, 500, api.SERVERERROR)
		return
	}

	businesses, err := h.service.ListBusinesses(c, int32(claims.UserID))
	if err != nil {
		h.logger.Errorf("error listing businesses: %v", err)
		api.ErrorResponse(c, 500, err.Error())
		return
	}

	var response []ListBusinessResponse
	for _, business := range businesses {

		response = append(response, ListBusinessResponse{
			ID:        business.ID,
			Name:      business.Name,
			Email:     business.Email.String,
			Website:   business.Website.String,
			Motto:     business.Motto.String,
			TaxID:     business.TaxID.String,
			LogoUrl:   business.LogoUrl.String,
			Country:   business.Country,
			Metadata:  utils.UnmarshalMetadata(business.Metadata),
			CreatedAt: business.CreatedAt.Time,
			UpdatedAt: business.UpdatedAt.Time,
		})
	}

	api.SuccessResponse(c, 200, "A list of your businesses", response)
}

type CreateBranchRequest struct {
	BusinessID int32                  `json:"business_id" example:"2" binding:"required"`
	Name       string                 `json:"name" example:"Main branch" binding:"required"`
	Address    string                 `json:"address" example:"..." binding:"omitempty"`
	Phone      string                 `json:"phone" example:"+2349028378964" binding:"omitempty"`
	Email      string                 `json:"email" example:"admin.mainbranch@gmail.com" binding:"omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
	IsActive   bool                   `json:"is_active" default:"true"`
}

type CreateBranchResponse struct {
	ID         int32                  `json:"id"`
	BusinessID int32                  `json:"business_id" example:"2"`
	Name       string                 `json:"name" example:"Main branch"`
	Address    string                 `json:"address" example:"1 Palmwine express"`
	Phone      string                 `json:"phone" example:"+2349028378964"`
	Email      string                 `json:"email" example:"admin.mainbranch@gmail.com"`
	Metadata   map[string]interface{} `json:"metadata"`
	IsActive   bool                   `json:"is_active"`
}

// CreateBranch godoc
// @Summary Create a branch
// @Description Create a branch. A business must have at least one branch.
// @Tags Organization/Branch
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param branch body CreateBranchRequest true "Branch details"
// @Success 201 {object} CreateBranchResponse "Branch created"
// @Failure 400 "Invalid input"
// @Failure 401 "Unauthorized"
// @Failure 403 "Forbidden"
// @Failure 500 "Internal server error"
// @Router /api/v1/org/branch [post]
func (h *Handler) createBranch(c *gin.Context) {
	claims, ok := jwt.GetUserFromContext(c)
	if !ok {
		h.logger.Errorf("could not get user from context")
		api.ErrorResponse(c, 400, api.SERVERERROR)
		return
	}

	var req CreateBranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("create branch request binding error: %v", err)
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	var params db.CreateBranchParams
	err := copier.Copy(&params, &req)
	if err != nil {
		h.logger.Errorf("error copying create branch request data: %v", err)
		api.ErrorResponse(c, 500, err.Error())
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
			api.ErrorResponse(c, 400, fmt.Sprintf("business with id %d does not exist", req.BusinessID))
			return
		}
		h.logger.Errorf("error getting business with id %d: %v", req.BusinessID, err)
		api.ErrorResponse(c, 500, err.Error())
		return
	}

	// convert map[string]any to JSON
	params.Metadata = utils.MarshalMetadata(req.Metadata)
	params.IsActive.Bool = req.IsActive
	params.IsActive.Valid = true

	// create branch
	branch, err := h.service.CreateBranch(c, params)
	if err != nil {
		h.logger.Errorf("error creating branch: %v", err)
		api.ErrorResponse(c, 500, err.Error())
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

	api.SuccessResponse(c, 201, "branch created", CreateBranchResponse{
		BusinessID: branch.BusinessID,
		Name:       branch.Name,
		Address:    branch.Address.String,
		Phone:      branch.Phone.String,
		Email:      branch.Email.String,
		IsActive:   branch.IsActive.Bool,
		Metadata:   utils.UnmarshalMetadata(branch.Metadata),
	})
}

// GetBranch godoc
// @Summary fetch a branch
// @Description Fetch a branch.
// @Tags Organization/Branch
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} CreateBranchResponse
// @Failure 400 "Invalid request data"
// @Failure 401 "Unauthorized"
// @Failure 403 "Forbidden"
// @Failure 500 "Internal server error"
// @Router /api/v1/org/branch/{id} [get]
func (h *Handler) getBranch(c *gin.Context) {
	id := c.Param("id")
	branchID, err := strconv.Atoi(id)
	if err != nil {
		h.logger.Errorf("get branch id str conv err: %v", err)
		api.ErrorResponse(c, 400, api.INVALID_REQUEST_DATA)
		return
	}
	branch, err := h.service.GetBranch(c, int32(branchID))
	if err != nil {
		h.logger.Errorf("error getting branch with is %d: %v", branchID, err)
		api.ErrorResponse(c, 500, err.Error())
		return
	}

	api.SuccessResponse(c, 200, "get branch successful", branch)
}

type UpdateBranchRequest struct {
	Name    string `json:"name" example:"Main branch" binding:"required"`
	Address string `json:"address" binding:"required" example:"..."`
	Country string `json:"country" binding:"required" example:"Nigeria"`
	Phone   string `json:"phone" binding:"omitempty" example:"+2349028378964"`
	Email   string `json:"email" binding:"omitempty" example:""`
	Website string `json:"website" binding:"omitempty" example:"https://"`
}

// UpdateBranch godoc
// @Summary Update an existing branch
// @Description Updates the details of a specific branch belonging to a business.
// @Description Only the branch owner (business owner) can perform this operation.
// @Description Fields not provided in the request will remain unchanged.
// @Tags Organization/Branch
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Branch ID (the unique identifier of the branch to update)"
// @Param branch body UpdateBranchRequest true "Branch details to update (only include fields you want to modify)"
// @Success 200 {object} CreateBranchResponse "Branch updated successfully"
// @Failure 400 "Invalid branch ID or bad request body"
// @Failure 401 "Unauthorized - missing or invalid authentication token"
// @Failure 403 "Forbidden - user does not have permission to update this branch"
// @Failure 404 "Branch not found"
// @Failure 500 "Internal server error"
// @Router /api/v1/org/branch/{id} [patch]
func (h *Handler) updateBranch(c *gin.Context) {
	claims, ok := jwt.GetUserFromContext(c)
	if !ok {
		h.logger.Errorf("could not get user from context")
		api.ErrorResponse(c, 500, api.SERVERERROR)
		return
	}

	id := c.Param("id")
	_, err := strconv.Atoi(id)
	if err != nil {
		h.logger.Errorf("get branch id str conv err: %v", err)
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	var req UpdateBranchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("update branch request binding error: %v", err)
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	updateParams := db.UpdateBranchParams{
		Name:    sql.NullString{String: req.Name, Valid: true},
		Phone:   sql.NullString{String: req.Phone, Valid: req.Phone != ""},
		Address: sql.NullString{String: req.Address},
		Email:   sql.NullString{String: req.Email, Valid: req.Email != ""},
	}

	branch, err := h.service.UpdateBranch(c, updateParams)
	if err != nil {
		h.logger.Errorf("error updating branch: %v", err)
		api.ErrorResponse(c, 500, err.Error())
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

	api.SuccessResponse(c, 200, "branch updated", branch)

}

// DeleteBranch godoc
// @Summary Delete a branch
// @Description Permanently deletes a specific branch of a business.
// @Description Only the business owner (or authorized user) can perform this operation.
// @Description Once deleted, the branch cannot be recovered.
// @Tags Organization/Branch
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Branch ID (the unique identifier of the branch to delete)"
// @Success 200 {string} string "Branch deleted successfully"
// @Failure 400 "Invalid branch ID supplied"
// @Failure 401 "Unauthorized - missing or invalid authentication token"
// @Failure 403 "Forbidden - user does not have permission to delete this branch"
// @Failure 404 "Branch not found"
// @Failure 500 "Internal server error"
// @Router /api/v1/org/branch/{id} [delete]
func (h *Handler) deleteBranch(c *gin.Context) {
	claims, ok := jwt.GetUserFromContext(c)
	if !ok {
		h.logger.Errorf("could not get user from context")
		api.ErrorResponse(c, 500, api.SERVERERROR)
		return
	}

	id := c.Param("id")
	bid, err := strconv.Atoi(id)
	if err != nil {
		h.logger.Errorf("get branch id str conv err: %v", err)
		api.ErrorResponse(c, 400, err.Error())
		return
	}
	branch, err := h.service.DeleteBranch(c, int32(bid))
	if err != nil {
		h.logger.Errorf("error deleting branch with is %d: %v", bid, err)
		api.ErrorResponse(c, 500, err.Error())
		return
	}

	api.SuccessResponse(c, 200, "branch deleted", nil)

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

	api.SuccessResponse(c, 200, "branch deleted", nil)

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
	CreatedAt  time.Time      `json:"created_at"`
}

// CreateSupplier godoc
// @Summary Create a new supplier
// @Description Registers a new supplier under a specific business.
// @Description The business must exist and belong to the authenticated user.
// @Description Each supplier must have a unique name within the same business.
// @Tags Organization/Supplier
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param supplier body Supplier true "Supplier details (name, phone, email, address, metadata, etc.)"
// @Success 201 {object} Supplier "Supplier successfully created"
// @Failure 400 "Invalid request data or supplier already exists"
// @Failure 401 "Unauthorized - missing or invalid authentication token"
// @Failure 403 "Forbidden - user does not have permission to create supplier for this business"
// @Failure 500 "Internal server error"
// @Router /api/v1/org/supplier [post]
func (h *Handler) createSupplier(c *gin.Context) {
	claims, ok := jwt.GetUserFromContext(c)
	if !ok {
		api.ErrorResponse(c, 500, "not logged in")
		return
	}

	var req Supplier
	if err := c.ShouldBind(&req); err != nil {
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	_, err := h.service.GetBusiness(c, db.GetBusinessParams{
		ID:      req.BusinessID,
		OwnerID: int32(claims.UserID),
	})

	if err != nil {
		api.ErrorResponse(c, 400, err.Error())
		return
	}

	var params db.CreateSupplierParams
	err = copier.Copy(&params, &req)
	if err != nil {
		api.ErrorResponse(c, 500, err.Error())
		return
	}

	params.Metadata = utils.MarshalMetadata(req.Metadata)

	supplier, err := h.service.CreateSupplier(c, params)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok {
			switch pgErr.Code {
			case "23505": // unique_violation
				api.ErrorResponse(c, 400, fmt.Sprintf("supplier with name %s already exists", req.Name))
				return
			}
		}
		api.ErrorResponse(c, 500, err.Error())
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

	api.SuccessResponse(c, 201, "Supplier created", Supplier{
		ID:         supplier.ID,
		Name:       supplier.Name,
		Phone:      supplier.Phone.String,
		Email:      supplier.Email.String,
		Address:    supplier.Address.String,
		BusinessID: supplier.BusinessID,
		Metadata:   meta,
		CreatedAt:  supplier.CreatedAt.Time,
	})

}
