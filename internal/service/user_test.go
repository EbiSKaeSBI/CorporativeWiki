package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"wiki/database"
	"wiki/internal/config"
	"wiki/internal/repository"
	"wiki/internal/service"
)

func init() {
	_ = godotenv.Load()
}

func newTestService(t *testing.T) *service.Service {
	conf := config.Load()
	db, err := database.Connect(conf)
	require.NoError(t, err, "failed to connect to database")
	repo := repository.NewRepository(db)
	return service.NewService(repo, conf)
}

func uniqueEmail() string {
	return fmt.Sprintf("user_%d@example.com", time.Now().UnixNano())
}

func TestCreateUser(t *testing.T) {
	svc := newTestService(t)

	user, err := svc.CreateUser(
		context.Background(),
		"Test User",
		uniqueEmail(),
		"password123",
		"viewer",
	)
	require.NoError(t, err, "CreateUser failed")
	require.NotNil(t, user)

	assert.NotZero(t, user.ID, "expected non-zero ID")
	assert.Equal(t, "Test User", user.Name)
	assert.NotEmpty(t, user.Email)
	assert.Equal(t, "viewer", user.Role)
}

func TestCreateUser_PasswordIsHashed(t *testing.T) {
	svc := newTestService(t)

	password := "mysecretpassword"
	user, err := svc.CreateUser(
		context.Background(),
		"Hash Test",
		uniqueEmail(),
		password,
		"viewer",
	)
	require.NoError(t, err, "CreateUser failed")
	require.NotNil(t, user)

	// Пароль не должен храниться в открытом виде
	assert.NotEqual(t, password, user.PasswordHash, "password should be hashed, not stored as plain text")

	// bcrypt хеш начинается с $2a$, $2b$ или $2y$
	assert.Len(t, user.PasswordHash, 60, "expected bcrypt hash of 60 chars")
	assert.Regexp(t, `^\$2[aby]\$`, user.PasswordHash, "expected bcrypt hash prefix")
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	svc := newTestService(t)

	email := uniqueEmail()

	_, err := svc.CreateUser(context.Background(), "User One", email, "password1", "viewer")
	require.NoError(t, err, "first CreateUser failed")

	_, err = svc.CreateUser(context.Background(), "User Two", email, "password2", "viewer")
	require.Error(t, err, "expected error for duplicate email")
	assert.ErrorIs(t, err, service.ErrUserAlreadyExists)
}

func TestCreateUser_DefaultRole(t *testing.T) {
	svc := newTestService(t)

	user, err := svc.CreateUser(
		context.Background(),
		"Default Role Test",
		uniqueEmail(),
		"password",
		"", // пустая роль
	)
	require.NoError(t, err, "CreateUser failed")
	require.NotNil(t, user)

	// Если роль не указана, должна быть viewer (или то, что настроено по умолчанию)
	assert.Equal(t, "viewer", user.Role)
}

func TestGetUserByID(t *testing.T) {
	svc := newTestService(t)

	created, err := svc.CreateUser(
		context.Background(),
		"Lookup By ID",
		uniqueEmail(),
		"password",
		"viewer",
	)
	require.NoError(t, err, "CreateUser failed")

	found, err := svc.GetUserByID(context.Background(), created.ID)
	require.NoError(t, err, "GetUserByID failed")
	require.NotNil(t, found)

	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, created.Email, found.Email)
}

func TestGetUserByID_NotFound(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.GetUserByID(context.Background(), 999999)
	require.Error(t, err, "expected error for non-existent user")
	assert.ErrorIs(t, err, service.ErrUserNotFound)
}

func TestGetUserByEmail(t *testing.T) {
	svc := newTestService(t)

	email := uniqueEmail()
	created, err := svc.CreateUser(
		context.Background(),
		"Lookup By Email",
		email,
		"password",
		"viewer",
	)
	require.NoError(t, err, "CreateUser failed")

	found, err := svc.GetUserByEmail(context.Background(), email)
	require.NoError(t, err, "GetUserByEmail failed")
	require.NotNil(t, found)

	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, created.Email, found.Email)
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.GetUserByEmail(context.Background(), "nonexistent@example.com")
	require.Error(t, err, "expected error for non-existent email")
	assert.ErrorIs(t, err, service.ErrUserNotFound)
}

