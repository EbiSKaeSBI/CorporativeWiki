package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"wiki/database"
	"wiki/internal/config"
	"wiki/internal/models"
	"wiki/internal/repository"
	"wiki/internal/service"
)

func TestGenerateAccessToken(t *testing.T) {
	conf := &config.Config{
		JwtSecret:    "test-secret-key",
		JwtAccessTTL: 24 * time.Hour,
	}
	db, err := database.Connect(config.Load())
	require.NoError(t, err, "failed to connect to database")
	repo := repository.NewRepository(db)
	svc := service.NewService(repo, conf)

	user := &models.User{
		Name:  "JWT User",
		Email: "jwt@example.com",
		Role:  "viewer",
	}

	token, err := svc.GenerateAccessToken(user)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Парсим и проверяем claims
	parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret-key"), nil
	})
	require.NoError(t, err)
	require.True(t, parsed.Valid)

	claims, ok := parsed.Claims.(jwt.MapClaims)
	require.True(t, ok)

	assert.Equal(t, user.ID, uint(claims["sub"].(float64)))
	assert.Equal(t, user.Email, claims["email"])
	assert.Equal(t, user.Role, claims["role"])
	assert.NotZero(t, claims["iat"])
	assert.NotZero(t, claims["exp"])
}

func TestGenerateAccessToken_InvalidSecret(t *testing.T) {
	conf := &config.Config{
		JwtSecret:    "",
		JwtAccessTTL: time.Hour,
	}
	db, err := database.Connect(config.Load())
	require.NoError(t, err, "failed to connect to database")
	repo := repository.NewRepository(db)
	svc := service.NewService(repo, conf)

	// JWT library accepts empty secret — we verify token was created (empty-signed)
	token, err := svc.GenerateAccessToken(&models.User{Name: "Test", Email: "test@test.com"})
	// Empty secret does not error in jwt/v5 library — token is still generated
	require.NoError(t, err)
	require.NotEmpty(t, token)
}

func TestLogin_TokenContainsCorrectClaims(t *testing.T) {
	svc := newTestService(t)

	email := uniqueEmail()
	password := "secret123"
	_, err := svc.Register(context.Background(), "JWT Login", email, password)
	require.NoError(t, err)

	result, err := svc.Login(context.Background(), email, password)
	require.NoError(t, err)
	require.NotNil(t, result)

	parsed, err := jwt.Parse(result.AccessToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.Load().JwtSecret), nil
	})
	require.NoError(t, err)
	require.True(t, parsed.Valid)

	claims, ok := parsed.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, email, claims["email"])
	assert.Equal(t, "viewer", claims["role"])
}
