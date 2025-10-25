package middleware

import (
	"strings"

	"github.com/davbauer/knock-knock-portal/internal/auth"
	"github.com/davbauer/knock-knock-portal/internal/models"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates JWT tokens
func AuthMiddleware(jwtManager *auth.JWTManager, requiredType auth.TokenType) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, models.NewErrorResponse("Authorization header required", "MISSING_AUTH_HEADER"))
			c.Abort()
			return
		}

		// Parse "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(401, models.NewErrorResponse("Invalid authorization header format", "INVALID_AUTH_HEADER"))
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			c.JSON(401, models.NewErrorResponse("Invalid or expired token", "INVALID_TOKEN"))
			c.Abort()
			return
		}

		// Check token type
		if claims.TokenType != requiredType {
			c.JSON(403, models.NewErrorResponse("Insufficient permissions", "INVALID_TOKEN_TYPE"))
			c.Abort()
			return
		}

		// Store claims in context
		c.Set("jwt_claims", claims)
		c.Set("user_id", claims.UserID)
		c.Set("session_id", claims.SessionID)

		c.Next()
	}
}

// GetJWTClaims retrieves JWT claims from context
func GetJWTClaims(c *gin.Context) (*auth.JWTClaims, bool) {
	if claims, exists := c.Get("jwt_claims"); exists {
		if jwtClaims, ok := claims.(*auth.JWTClaims); ok {
			return jwtClaims, true
		}
	}
	return nil, false
}
