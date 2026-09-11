package repository_test

import (
	"fmt"
	"testing"
	"time"

	"wiki/database"
	"wiki/internal/config"
	"wiki/internal/models"
	"wiki/internal/repository"
)

func newTestRepo(t *testing.T) *repository.Repository {
	conf := config.Load()
	db, err := database.Connect(conf)
	if err != nil {
		t.Fatal(err)
	}
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

	created, err := repo.CreateUser(user)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if created.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if created.Name != user.Name {
		t.Errorf("expected name %q, got %q", user.Name, created.Name)
	}
	if created.Email != user.Email {
		t.Errorf("expected email %q, got %q", user.Email, created.Email)
	}
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

	if _, err := repo.CreateUser(user1); err != nil {
		t.Fatalf("first CreateUser failed: %v", err)
	}

	user2 := &models.User{
		Name:         "User Two",
		Email:        email,
		PasswordHash: "hash2",
		Role:         "viewer",
	}

	_, err := repo.CreateUser(user2)
	if err == nil {
		t.Error("expected error for duplicate email, got nil")
	}
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

	created, err := repo.CreateUser(user)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	found, err := repo.GetUserByID(created.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}

	if found == nil {
		t.Fatal("expected user, got nil")
	}
	if found.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, found.ID)
	}
	if found.Email != created.Email {
		t.Errorf("expected email %q, got %q", created.Email, found.Email)
	}
}

// TestGetUserByID_NotFound проверяет, что запрос несуществующего ID возвращает ошибку.
func TestGetUserByID_NotFound(t *testing.T) {
	repo := newTestRepo(t)

	_, err := repo.GetUserByID(999999)
	if err == nil {
		t.Error("expected error for non-existent user, got nil")
	}
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

	created, err := repo.CreateUser(user)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	found, err := repo.GetUserByEmail(email)
	if err != nil {
		t.Fatalf("GetUserByEmail failed: %v", err)
	}

	if found == nil {
		t.Fatal("expected user, got nil")
	}
	if found.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, found.ID)
	}
}

// TestGetUserByEmail_NotFound проверяет запрос несуществующего email.
func TestGetUserByEmail_NotFound(t *testing.T) {
	repo := newTestRepo(t)

	_, err := repo.GetUserByEmail("nonexistent@example.com")
	if err == nil {
		t.Error("expected error for non-existent email, got nil")
	}
}
