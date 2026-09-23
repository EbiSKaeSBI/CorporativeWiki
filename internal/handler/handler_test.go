package handler_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"

	"wiki/database"
	"wiki/internal/config"
	"wiki/internal/handler"
	"wiki/internal/middleware"
	"wiki/internal/repository"
	"wiki/internal/service"
)

func init() {
	_ = godotenv.Load()
}

func newTestHandler(t *testing.T) (*handler.Handler, *repository.Repository) {
	conf := config.Load()
	db, err := database.Connect(conf)
	require.NoError(t, err, "failed to connect to database")
	repo := repository.NewRepository(db)
	return handler.NewHandler(service.NewService(repo, conf)), repo
}

// setupRouter собирает роутер так же, как в cmd/server/main.go.
func setupRouter(t *testing.T) (*gin.Engine, *repository.Repository) {
	gin.SetMode(gin.TestMode)
	h, repo := newTestHandler(t)

	r := gin.New()
	r.GET("/health", h.Health)
	r.GET("/users/:id", h.GetUserByID)
	r.POST("/users", h.CreateUser)
	r.POST("/auth/register", h.Register)
	r.POST("/auth/login", h.Login)
	return r, repo
}

func uniqueEmail() string {
	return fmt.Sprintf("user_%d@example.com", time.Now().UnixNano())
}

// setupProtectedRouter собирает роутер с JWT-защитой для Profile.
func setupProtectedRouter(t *testing.T) (*gin.Engine, *repository.Repository) {
	gin.SetMode(gin.TestMode)
	h, repo := newTestHandler(t)

	conf := config.Load()
	r := gin.New()
	r.GET("/health", h.Health)
	r.POST("/auth/register", h.Register)
	r.POST("/auth/login", h.Login)

	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware(conf.JwtSecret))
	protected.GET("/profile", h.Profile)

	return r, repo
}

// testToken генерирует JWT с указанным user_id для тестов.
func testToken(t *testing.T, userID uint) string {
	conf := config.Load()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  float64(userID),
		"role": "admin",
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(conf.JwtSecret))
	require.NoError(t, err)
	return tokenString
}