// Package docs contains the API documentation configuration
// and swagger specifications for the Hotel ERP system.
package docs

import (
	"github.com/swaggo/swag"
)

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {
            "name": "Hotel ERP API Support",
            "email": "support@hotel-erp.com"
        },
        "license": {
            "name": "MIT",
            "url": "https://opensource.org/licenses/MIT"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {},
    "definitions": {},
    "securityDefinitions": {
        "BearerAuth": {
            "description": "JWT Authorization header using the Bearer scheme. Example: \"Authorization: Bearer {token}\"",
            "type": "apiKey",
            "name": "Authorization",
            "in": "header"
        }
    }
}`

var doc = `{
    "schemes": ["http", "https"],
    "swagger": "2.0",
    "info": {
        "description": "This is the Hotel ERP API server. It provides endpoints for managing hotel operations including authentication, point of sale, inventory, and more.",
        "title": "Hotel ERP API",
        "contact": {
            "name": "Hotel ERP API Support",
            "email": "support@hotel-erp.com"
        },
        "license": {
            "name": "MIT",
            "url": "https://opensource.org/licenses/MIT"
        },
        "version": "1.0.0"
    },
    "host": "localhost:9000",
    "basePath": "/api",
    "paths": {
        "/auth/login": {
            "post": {
                "description": "Authenticate user and return JWT token",
                "consumes": ["application/json"],
                "produces": ["application/json"],
                "tags": ["auth"],
                "summary": "User login",
                "parameters": [
                    {
                        "description": "Login credentials",
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/LoginRequest"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Login successful",
                        "schema": {
                            "$ref": "#/definitions/LoginResponse"
                        }
                    },
                    "400": {
                        "description": "Bad request",
                        "schema": {
                            "$ref": "#/definitions/ErrorResponse"
                        }
                    },
                    "401": {
                        "description": "Unauthorized",
                        "schema": {
                            "$ref": "#/definitions/ErrorResponse"
                        }
                    }
                }
            }
        },
        "/health": {
            "get": {
                "description": "Check the health status of the API server",
                "produces": ["application/json"],
                "tags": ["health"],
                "summary": "Health check",
                "responses": {
                    "200": {
                        "description": "Service is healthy",
                        "schema": {
                            "$ref": "#/definitions/HealthResponse"
                        }
                    },
                    "500": {
                        "description": "Service is unhealthy",
                        "schema": {
                            "$ref": "#/definitions/ErrorResponse"
                        }
                    }
                }
            }
        },
        "/pos/sales": {
            "post": {
                "security": [{"BearerAuth": []}],
                "description": "Create a new sale transaction",
                "consumes": ["application/json"],
                "produces": ["application/json"],
                "tags": ["pos"],
                "summary": "Create sale",
                "parameters": [
                    {
                        "description": "Sale details",
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/CreateSaleRequest"
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Sale created successfully",
                        "schema": {
                            "$ref": "#/definitions/SaleResponse"
                        }
                    },
                    "400": {
                        "description": "Bad request",
                        "schema": {
                            "$ref": "#/definitions/ErrorResponse"
                        }
                    },
                    "401": {
                        "description": "Unauthorized",
                        "schema": {
                            "$ref": "#/definitions/ErrorResponse"
                        }
                    },
                    "403": {
                        "description": "Forbidden",
                        "schema": {
                            "$ref": "#/definitions/ErrorResponse"
                        }
                    }
                }
            }
        },
        "/pos/sales/history": {
            "get": {
                "security": [{"BearerAuth": []}],
                "description": "Get sales history with optional filters",
                "produces": ["application/json"],
                "tags": ["pos"],
                "summary": "Get sales history",
                "parameters": [
                    {
                        "type": "integer",
                        "description": "Page number",
                        "name": "page",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "Number of items per page",
                        "name": "limit",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Start date (YYYY-MM-DD)",
                        "name": "start_date",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "End date (YYYY-MM-DD)",
                        "name": "end_date",
                        "in": "query"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Sales history retrieved successfully",
                        "schema": {
                            "$ref": "#/definitions/SalesHistoryResponse"
                        }
                    },
                    "401": {
                        "description": "Unauthorized",
                        "schema": {
                            "$ref": "#/definitions/ErrorResponse"
                        }
                    },
                    "403": {
                        "description": "Forbidden",
                        "schema": {
                            "$ref": "#/definitions/ErrorResponse"
                        }
                    }
                }
            }
        },
        "/pos/items": {
            "post": {
                "security": [{"BearerAuth": []}],
                "description": "Create a new inventory item",
                "consumes": ["application/json"],
                "produces": ["application/json"],
                "tags": ["pos"],
                "summary": "Create item",
                "parameters": [
                    {
                        "description": "Item details",
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/CreateItemRequest"
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Item created successfully",
                        "schema": {
                            "$ref": "#/definitions/ItemResponse"
                        }
                    },
                    "400": {
                        "description": "Bad request",
                        "schema": {
                            "$ref": "#/definitions/ErrorResponse"
                        }
                    },
                    "401": {
                        "description": "Unauthorized",
                        "schema": {
                            "$ref": "#/definitions/ErrorResponse"
                        }
                    },
                    "403": {
                        "description": "Forbidden",
                        "schema": {
                            "$ref": "#/definitions/ErrorResponse"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "LoginRequest": {
            "type": "object",
            "required": ["username", "password"],
            "properties": {
                "username": {
                    "type": "string",
                    "example": "admin"
                },
                "password": {
                    "type": "string",
                    "example": "password123"
                }
            }
        },
        "LoginResponse": {
            "type": "object",
            "properties": {
                "token": {
                    "type": "string",
                    "example": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
                }
            }
        },
        "ErrorResponse": {
            "type": "object",
            "properties": {
                "error": {
                    "type": "string",
                    "example": "Invalid credentials"
                }
            }
        },
        "HealthResponse": {
            "type": "object",
            "properties": {
                "status": {
                    "type": "string",
                    "example": "healthy"
                }
            }
        },
        "CreateSaleRequest": {
            "type": "object",
            "required": ["items", "customer_id"],
            "properties": {
                "customer_id": {
                    "type": "integer",
                    "example": 1
                },
                "items": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/SaleItem"
                    }
                },
                "discount": {
                    "type": "number",
                    "format": "float",
                    "example": 10.5
                },
                "tax_rate": {
                    "type": "number",
                    "format": "float",
                    "example": 8.25
                }
            }
        },
        "SaleItem": {
            "type": "object",
            "required": ["item_id", "quantity", "price"],
            "properties": {
                "item_id": {
                    "type": "integer",
                    "example": 1
                },
                "quantity": {
                    "type": "integer",
                    "example": 2
                },
                "price": {
                    "type": "number",
                    "format": "float",
                    "example": 25.99
                }
            }
        },
        "SaleResponse": {
            "type": "object",
            "properties": {
                "id": {
                    "type": "integer",
                    "example": 1
                },
                "customer_id": {
                    "type": "integer",
                    "example": 1
                },
                "total_amount": {
                    "type": "number",
                    "format": "float",
                    "example": 56.23
                },
                "tax_amount": {
                    "type": "number",
                    "format": "float",
                    "example": 4.27
                },
                "discount_amount": {
                    "type": "number",
                    "format": "float",
                    "example": 10.5
                },
                "items": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/SaleItem"
                    }
                },
                "created_at": {
                    "type": "string",
                    "format": "date-time",
                    "example": "2024-01-15T10:30:00Z"
                }
            }
        },
        "SalesHistoryResponse": {
            "type": "object",
            "properties": {
                "sales": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/SaleResponse"
                    }
                },
                "pagination": {
                    "$ref": "#/definitions/PaginationResponse"
                }
            }
        },
        "PaginationResponse": {
            "type": "object",
            "properties": {
                "page": {
                    "type": "integer",
                    "example": 1
                },
                "limit": {
                    "type": "integer",
                    "example": 20
                },
                "total": {
                    "type": "integer",
                    "example": 100
                },
                "pages": {
                    "type": "integer",
                    "example": 5
                }
            }
        },
        "CreateItemRequest": {
            "type": "object",
            "required": ["name", "price", "category"],
            "properties": {
                "name": {
                    "type": "string",
                    "example": "Deluxe Room Service"
                },
                "description": {
                    "type": "string",
                    "example": "24-hour room service with premium menu"
                },
                "price": {
                    "type": "number",
                    "format": "float",
                    "example": 45.99
                },
                "category": {
                    "type": "string",
                    "example": "Room Service"
                },
                "sku": {
                    "type": "string",
                    "example": "RS-DELUXE-001"
                },
                "stock_quantity": {
                    "type": "integer",
                    "example": 100
                }
            }
        },
        "ItemResponse": {
            "type": "object",
            "properties": {
                "id": {
                    "type": "integer",
                    "example": 1
                },
                "name": {
                    "type": "string",
                    "example": "Deluxe Room Service"
                },
                "description": {
                    "type": "string",
                    "example": "24-hour room service with premium menu"
                },
                "price": {
                    "type": "number",
                    "format": "float",
                    "example": 45.99
                },
                "category": {
                    "type": "string",
                    "example": "Room Service"
                },
                "sku": {
                    "type": "string",
                    "example": "RS-DELUXE-001"
                },
                "stock_quantity": {
                    "type": "integer",
                    "example": 100
                },
                "created_at": {
                    "type": "string",
                    "format": "date-time",
                    "example": "2024-01-15T10:30:00Z"
                },
                "updated_at": {
                    "type": "string",
                    "format": "date-time",
                    "example": "2024-01-15T10:30:00Z"
                }
            }
        }
    },
    "securityDefinitions": {
        "BearerAuth": {
            "description": "JWT Authorization header using the Bearer scheme. Example: \"Authorization: Bearer {token}\"",
            "type": "apiKey",
            "name": "Authorization",
            "in": "header"
        }
    }
}`

// func init() {
// 	swag.Register("swagger", &s{})
// }

type s struct{}

func (s *s) ReadDoc() string {
	sInfo := swag.Spec{
		Version:          "1.0.0",
		Host:             "localhost:9000",
		BasePath:         "/api",
		Schemes:          []string{"http", "https"},
		Title:            "Hotel ERP API",
		Description:      "This is the Hotel ERP API server. It provides endpoints for managing hotel operations including authentication, point of sale, inventory, and more.",
		InfoInstanceName: "swagger",
		SwaggerTemplate:  docTemplate,
	}
	sInfo.Description = "This is the Hotel ERP API server. It provides endpoints for managing hotel operations including authentication, point of sale, inventory, and more."
	return doc
}
