package handler_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"wiki/database"
	"wiki/internal/config"
	"wiki/internal/handler"
	"wiki/internal/repository"
	"wiki/internal/service"
)

func newTestHandler(t *testing.T) (*handler.Handler, *repository.Repository) {
	conf := config.Load()
	db, err := database.Connect(conf)
	require.NoError(t, err, "failed to connect to database")
	repo := repository.NewRepository(db)
	return handler.NewHandler(service.NewService(repo)), repo
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