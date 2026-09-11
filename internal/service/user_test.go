package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"wiki/database"
	"wiki/internal/config"
	"wiki/internal/repository"
	"wiki/internal/service"
)

func newTestService(t *testing.T) *service.Service {
	conf := config.Load()
	db, err := database.Connect(conf)
	if err != nil {
		t.Fatal(err)
	}
	repo := repository.NewRepository(db)
	return service.NewService(repo)
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
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if user.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if user.Email == "" {
		t.Error("expected non-empty email")
	}
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
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Пароль не должен храниться в открытом виде
	if user.PasswordHash == password {
		t.Error("password should be hashed, not stored as plain text")
	}

	// bcrypt хеш начинается с $2a$, $2b$ или $2y$
	if len(user.PasswordHash) < 3 {
		t.Fatal("password hash too short")
	}
	prefix := user.PasswordHash[:3]
	if prefix != "$2a" && prefix != "$2b" && prefix != "$2y" {
		t.Errorf("expected bcrypt hash (prefix $2a/$2b/$2y), got prefix %s", prefix)
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	svc := newTestService(t)

	email := uniqueEmail()

	_, err := svc.CreateUser(context.Background(), "User One", email, "password1", "viewer")
	if err != nil {
		t.Fatalf("first CreateUser failed: %v", err)
	}

	_, err = svc.CreateUser(context.Background(), "User Two", email, "password2", "viewer")
	if err == nil {
		t.Error("expected error for duplicate email, got nil")
	}
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
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Если роль не указана, должна быть viewer (или то, что настроено по умолчанию)
	expectedRole := "viewer"
	if user.Role != expectedRole {
		t.Errorf("expected role %q, got %q", expectedRole, user.Role)
	}
}
