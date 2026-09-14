package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"wiki/database"
	"wiki/internal/config"
	"wiki/internal/models"
	"wiki/internal/repository"
)

func newTestRepo(t *testing.T) *repository.Repository {
	conf := config.Load()
	db, err := database.Connect(conf)
	require.NoError(t, err, "failed to connect to database")
	return repository.NewRepository(db)
}

func uniqueEmail() string {
	return fmt.Sprintf("user_%d@example.com", time.Now().UnixNano())
}

// TestCreateUser проверяет, что пользователь создаётся и возвращается с ID.
func TestCreateUser(t *testing.T) {
	repo := newTestRepo(t)

	user := &models.User{
		Name:         "Test User",
		Email:        uniqueEmail(),
		PasswordHash: "hashed_password",
		Role:         "viewer",
	}

	created, err := repo.CreateUser(context.Background(), user)
	require.NoError(t, err, "CreateUser failed")

	assert.NotZero(t, created.ID, "expected non-zero ID")
	assert.Equal(t, user.Name, created.Name)
	assert.Equal(t, user.Email, created.Email)
	assert.Equal(t, user.Role, created.Role)
	assert.NotZero(t, created.CreatedAt)
}

// TestCreateUser_DuplicateEmail проверяет, что повторная вставка с тем же email возвращает ошибку.
func TestCreateUser_DuplicateEmail(t *testing.T) {
	repo := newTestRepo(t)

	email := uniqueEmail()

	user1 := &models.User{
		Name:         "User One",
		Email:        email,
		PasswordHash: "hash1",
		Role:         "viewer",
	}

	_, err := repo.CreateUser(context.Background(), user1)
	require.NoError(t, err, "first CreateUser failed")

	user2 := &models.User{
		Name:         "User Two",
		Email:        email,
		PasswordHash: "hash2",
		Role:         "viewer",
	}

	_, err = repo.CreateUser(context.Background(), user2)
	require.Error(t, err, "expected error for duplicate email")
}

// TestGetUserByID проверяет получение пользователя по ID.
func TestGetUserByID(t *testing.T) {
	repo := newTestRepo(t)

	user := &models.User{
		Name:         "Find Me",
		Email:        uniqueEmail(),
		PasswordHash: "hash",
		Role:         "editor",
	}

	created, err := repo.CreateUser(context.Background(), user)
	require.NoError(t, err, "CreateUser failed")

	found, err := repo.GetUserByID(context.Background(), created.ID)
	require.NoError(t, err, "GetUserByID failed")
	require.NotNil(t, found, "expected user, got nil")

	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, created.Email, found.Email)
	assert.Equal(t, created.Name, found.Name)
	assert.Equal(t, created.Role, found.Role)
}

// TestGetUserByID_NotFound проверяет, что запрос несуществующего ID возвращает ошибку.
func TestGetUserByID_NotFound(t *testing.T) {
	repo := newTestRepo(t)

	_, err := repo.GetUserByID(context.Background(), 999999)
	require.Error(t, err, "expected error for non-existent user")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// TestGetUserByEmail проверяет получение пользователя по email.
func TestGetUserByEmail(t *testing.T) {
	repo := newTestRepo(t)

	email := uniqueEmail()
	user := &models.User{
		Name:         "Email Finder",
		Email:        email,
		PasswordHash: "hash",
		Role:         "admin",
	}

	created, err := repo.CreateUser(context.Background(), user)
	require.NoError(t, err, "CreateUser failed")

	found, err := repo.GetUserByEmail(context.Background(), email)
	require.NoError(t, err, "GetUserByEmail failed")
	require.NotNil(t, found, "expected user, got nil")

	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, created.Email, found.Email)
}

// TestGetUserByEmail_NotFound проверяет запрос несуществующего email.
func TestGetUserByEmail_NotFound(t *testing.T) {
	repo := newTestRepo(t)

	_, err := repo.GetUserByEmail(context.Background(), "nonexistent@example.com")
	require.Error(t, err, "expected error for non-existent email")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
