package pos

import (
	"herp/internal/auth"
	"herp/internal/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// CreateSaleRequest represents the request payload for creating a sale
// @Description Create sale request payload
type CreateSaleRequest struct {
	CustomerID int        `json:"customer_id" binding:"required" example:"1"` // Customer ID
	Items      []SaleItem `json:"items" binding:"required"`                   // List of items in the sale
	Discount   float64    `json:"discount" example:"10.5"`                    // Discount amount
	TaxRate    float64    `json:"tax_rate" example:"8.25"`                    // Tax rate percentage
}

// SaleItem represents an item in a sale
// @Description Sale item details
type SaleItem struct {
	ItemID   int     `json:"item_id" binding:"required" example:"1"`   // Item ID
	Quantity int     `json:"quantity" binding:"required" example:"2"`  // Quantity of the item
	Price    float64 `json:"price" binding:"required" example:"25.99"` // Price per unit
}

// SaleResponse represents the response payload for a sale
// @Description Sale response payload
type SaleResponse struct {
	ID             int        `json:"id" example:"1"`                            // Sale ID
	CustomerID     int        `json:"customer_id" example:"1"`                   // Customer ID
	TotalAmount    float64    `json:"total_amount" example:"56.23"`              // Total amount after tax and discount
	TaxAmount      float64    `json:"tax_amount" example:"4.27"`                 // Tax amount
	DiscountAmount float64    `json:"discount_amount" example:"10.5"`            // Discount amount
	Items          []SaleItem `json:"items"`                                     // List of items in the sale
	CreatedAt      time.Time  `json:"created_at" example:"2024-01-15T10:30:00Z"` // Sale creation timestamp
}

// SalesHistoryResponse represents the response payload for sales history
// @Description Sales history response payload
type SalesHistoryResponse struct {
	Sales      []SaleResponse     `json:"sales"`      // List of sales
	Pagination PaginationResponse `json:"pagination"` // Pagination information
}

// PaginationResponse represents pagination information
// @Description Pagination response payload
type PaginationResponse struct {
	Page  int `json:"page" example:"1"`    // Current page number
	Limit int `json:"limit" example:"20"`  // Number of items per page
	Total int `json:"total" example:"100"` // Total number of items
	Pages int `json:"pages" example:"5"`   // Total number of pages
}

// CreateItemRequest represents the request payload for creating an item
// @Description Create item request payload
type CreateItemRequest struct {
	Name          string  `json:"name" binding:"required" example:"Deluxe Room Service"`        // Item name
	Description   string  `json:"description" example:"24-hour room service with premium menu"` // Item description
	Price         float64 `json:"price" binding:"required" example:"45.99"`                     // Item price
	Category      string  `json:"category" binding:"required" example:"Room Service"`           // Item category
	SKU           string  `json:"sku" example:"RS-DELUXE-001"`                                  // Item SKU
	StockQuantity int     `json:"stock_quantity" example:"100"`                                 // Stock quantity
}

// ItemResponse represents the response payload for an item
// @Description Item response payload
type ItemResponse struct {
	ID            int       `json:"id" example:"1"`                                               // Item ID
	Name          string    `json:"name" example:"Deluxe Room Service"`                           // Item name
	Description   string    `json:"description" example:"24-hour room service with premium menu"` // Item description
	Price         float64   `json:"price" example:"45.99"`                                        // Item price
	Category      string    `json:"category" example:"Room Service"`                              // Item category
	SKU           string    `json:"sku" example:"RS-DELUXE-001"`                                  // Item SKU
	StockQuantity int       `json:"stock_quantity" example:"100"`                                 // Stock quantity
	CreatedAt     time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`                    // Item creation timestamp
	UpdatedAt     time.Time `json:"updated_at" example:"2024-01-15T10:30:00Z"`                    // Item last update timestamp
}

// ErrorResponse represents an error response
// @Description Error response payload
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid request"` // Error message
}

func RegisterRoutes(r *gin.RouterGroup, authSvc *auth.Service) {
	pos := r.Group("/pos")
	pos.Use(auth.AuthMiiddleware(authSvc))

	// Sales endpoint
	sales := pos.Group("/sales")
	{
		sales.POST("", auth.PermissionMiddleware(authSvc, "pos:sell"), createSale)
		sales.GET("/history", auth.PermissionMiddleware(authSvc, "pos:view"), getSalesHistory)
	}

	// items endpoint
	items := pos.Group("/items")
	{
		items.POST("", auth.PermissionMiddleware(authSvc, "pos:manage_items"), createItem)
	}
}

// CreateSale godoc
// @Summary Create sale
// @Description Create a new sale transaction
// @Tags pos
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateSaleRequest true "Sale details"
// @Success 201 {object} SaleResponse "Sale created successfully"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /pos/sales [post]
func createSale(c *gin.Context) {
	var req CreateSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, err.Error())
		return
	}

	// TODO: Implement actual sale creation logic
	// For now, return a mock response
	response := SaleResponse{
		ID:             1,
		CustomerID:     req.CustomerID,
		TotalAmount:    calculateTotal(req.Items, req.Discount, req.TaxRate),
		TaxAmount:      calculateTax(req.Items, req.TaxRate),
		DiscountAmount: req.Discount,
		Items:          req.Items,
		CreatedAt:      time.Now(),
	}

	utils.SuccessResponse(c, 201, "", response)
}

// GetSalesHistory godoc
// @Summary Get sales history
// @Description Get sales history with optional filters
// @Tags pos
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number"
// @Param limit query int false "Number of items per page"
// @Param start_date query string false "Start date (YYYY-MM-DD)"
// @Param end_date query string false "End date (YYYY-MM-DD)"
// @Success 200 {object} SalesHistoryResponse "Sales history retrieved successfully"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /pos/sales/history [get]
func getSalesHistory(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// TODO: Implement actual sales history retrieval logic
	// For now, return a mock response
	response := SalesHistoryResponse{
		Sales: []SaleResponse{
			{
				ID:             1,
				CustomerID:     1,
				TotalAmount:    56.23,
				TaxAmount:      4.27,
				DiscountAmount: 10.5,
				Items: []SaleItem{
					{ItemID: 1, Quantity: 2, Price: 25.99},
				},
				CreatedAt: time.Now().Add(-24 * time.Hour),
			},
		},
		Pagination: PaginationResponse{
			Page:  page,
			Limit: limit,
			Total: 1,
			Pages: 1,
		},
	}

	// Log filters for debugging
	if startDate != "" || endDate != "" {
		// TODO: Apply date filters
		_ = startDate
		_ = endDate
	}

	c.JSON(http.StatusOK, response)
}

// CreateItem godoc
// @Summary Create item
// @Description Create a new inventory item
// @Tags pos
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateItemRequest true "Item details"
// @Success 201 {object} ItemResponse "Item created successfully"
// @Failure 400 {object} ErrorResponse "Bad request"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /pos/items [post]
func createItem(c *gin.Context) {
	var req CreateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// TODO: Implement actual item creation logic
	// For now, return a mock response
	response := ItemResponse{
		ID:            1,
		Name:          req.Name,
		Description:   req.Description,
		Price:         req.Price,
		Category:      req.Category,
		SKU:           req.SKU,
		StockQuantity: req.StockQuantity,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	c.JSON(http.StatusCreated, response)
}

// Helper functions for calculations
func calculateTotal(items []SaleItem, discount, taxRate float64) float64 {
	subtotal := 0.0
	for _, item := range items {
		subtotal += item.Price * float64(item.Quantity)
	}

	discountedTotal := subtotal - discount
	if discountedTotal < 0 {
		discountedTotal = 0
	}

	tax := discountedTotal * (taxRate / 100)
	return discountedTotal + tax
}

func calculateTax(items []SaleItem, taxRate float64) float64 {
	subtotal := 0.0
	for _, item := range items {
		subtotal += item.Price * float64(item.Quantity)
	}
	return subtotal * (taxRate / 100)
}
