package auth

import (
	"errors"
	"herp/pkg/jwt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	AuthorizationHeader = "Authorization"
	BearerPrefix        = "Bearer "
)

var (
	ErrNoAuthHeader      = errors.New("authorization header is missing")
	ErrInvalidAuthHeader = errors.New("invalid authorization header format")
	ErrInvalidToken      = errors.New("invalid token")
)

func AuthMiiddleware(authSvc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": ErrInvalidAuthHeader.Error()})
			return
		} 

		if !strings.HasPrefix(authHeader, BearerPrefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": ErrInvalidAuthHeader.Error()})
			return
		}

		token := strings.TrimPrefix(authHeader, BearerPrefix)
		claims, err := jwt.ParseToken(token, authSvc.jwtSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": ErrInvalidToken.Error()})
			return
		}

		// check blacklist
		blacklisted, err := authSvc.IsTokenBlacklisted(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if blacklisted {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": ErrInvalidToken.Error()})
			return
		}

		c.Set("claims", claims)
		c.Next()
	}
}

func PermissionMiddleware(authSvc *Service, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get("claims")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized to make this request"})
			return
		}

		jwtClaims, ok := claims.(*jwt.Claims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "invalid claim type"})
			return
		}

		if !authSvc.HasPermission(jwtClaims, permission) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}
		c.Next()
	}
}

func AdminMiddleware(authSvc *Service) gin.HandlerFunc {
	return PermissionMiddleware(authSvc, "admin:manage")
}
