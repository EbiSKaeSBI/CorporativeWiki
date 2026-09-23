package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(secret string) gin.HandlerFunc {
	if secret == "" {
		panic("AuthMiddleware: JWT secret must not be empty")
	}
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header",
			})
			c.Abort()
			return
		}
		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
			})
			c.Abort()
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Claims not found exception",
			})
			c.Abort()
			return
		}
		sub, ok := claims["sub"].(float64)
		if !ok || sub <= 0 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid user id",
			})
			c.Abort()
			return
		}
		userID := uint(sub)
		role, ok := claims["role"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid role",
			})
			c.Abort()
			return
		}
		c.Set("user_id", userID)
		c.Set("role", role)
		c.Next()
	}
}

func RoleMiddleware(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Роль не найдена",
			})
			c.Abort()
			return
		}
		roleUser, ok := roleValue.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid role",
			})
			c.Abort()
			return
		}
		for _, role := range roles {
			if roleUser == role {
				c.Next()
				return 
			} 
		}
		c.JSON(http.StatusForbidden, gin.H{
			"error": "У вас недостаточно прав",
		})
		c.Abort()
	}
}
