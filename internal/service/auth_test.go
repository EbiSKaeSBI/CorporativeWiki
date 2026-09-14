package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
