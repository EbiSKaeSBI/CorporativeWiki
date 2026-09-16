package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"wiki/internal/service"
)

func TestRegister(t *testing.T) {
	svc := newTestService(t)

	user, err := svc.Register(
		context.Background(),
		"Ali",
		uniqueEmail(),
		"secret123",
	)
	require.NoError(t, err)
	require.NotNil(t, user)

	assert.NotZero(t, user.ID, "expected non-zero ID")
	assert.Equal(t, "Ali", user.Name)
	assert.Equal(t, "viewer", user.Role)
}

func TestRegister_PasswordIsHashed(t *testing.T) {
	svc := newTestService(t)

	password := "mysecretpassword"
	user, err := svc.Register(
		context.Background(),
		"Hash Check",
		uniqueEmail(),
		password,
	)
	require.NoError(t, err)
	require.NotNil(t, user)

	assert.NotEqual(t, password, user.PasswordHash, "password should be hashed")
	assert.Len(t, user.PasswordHash, 60, "expected bcrypt hash of 60 chars")
	assert.Regexp(t, `^\$2[aby]\$`, user.PasswordHash, "expected bcrypt hash prefix")
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc := newTestService(t)

	email := uniqueEmail()

	_, err := svc.Register(context.Background(), "First", email, "password1")
	require.NoError(t, err)

	_, err = svc.Register(context.Background(), "Second", email, "password2")
	require.Error(t, err, "expected error for duplicate email")
}

func TestLogin(t *testing.T) {
	svc := newTestService(t)

	email := uniqueEmail()
	password := "secret123"
	_, err := svc.Register(context.Background(), "Login User", email, password)
	require.NoError(t, err)

	user, err := svc.Login(context.Background(), email, password)
	require.NoError(t, err)
	require.NotNil(t, user)

	assert.Equal(t, "Login User", user.Name)
	assert.Equal(t, email, user.Email)
}

func TestLogin_InvalidPassword(t *testing.T) {
	svc := newTestService(t)

	email := uniqueEmail()
	_, err := svc.Register(context.Background(), "Login User", email, "correct")
	require.NoError(t, err)

	_, err = svc.Login(context.Background(), email, "wrong")
	require.Error(t, err)
	assert.ErrorIs(t, err, service.ErrInvalidCredentials)
}

func TestLogin_UserNotFound(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Login(context.Background(), "nonexistent@example.com", "any")
	require.Error(t, err)
	assert.ErrorIs(t, err, service.ErrInvalidCredentials)
}
