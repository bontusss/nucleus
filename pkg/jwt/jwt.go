package jwt

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

type Claims struct {
	UserID      int       `json:"userId"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	Permissions []string  `json:"permissions"`
	TokenType   TokenType `json:"tokenType"`
	jwt.RegisteredClaims
}

func GenerateToken(userID int, username, email, role, secret string, permissions []string, tokenType TokenType, expiry time.Duration) (string, error) {
	expirationTime := time.Now().Add(expiry)

	claims := &Claims{
		UserID:      userID,
		Email:       email,
		Role:        role,
		Permissions: permissions,
		Username:    username,
		TokenType:   tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseToken(tokenString, secret string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	return claims, nil
}

// ValidateTokenType checks if the token is of the expected type
func ValidateTokenType(claims *Claims, expectedType TokenType) error {
	if claims.TokenType != expectedType {
		return jwt.ErrTokenInvalidClaims
	}
	return nil
}

func GetUserFromContext(c *gin.Context) (*Claims, bool) {
	claims, exists := c.Get("claims")
	if !exists {
		return nil, false
	}

	userClaims, ok := claims.(*Claims)
	if !ok {
		return nil, false
	}

	return userClaims, true
}
