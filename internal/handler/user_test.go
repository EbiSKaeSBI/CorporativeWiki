package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"wiki/internal/dto"
	"wiki/internal/models"
)

func TestHandler_CreateUser(t *testing.T) {
	r, _ := setupRouter(t)

	body, err := json.Marshal(dto.CreateUserRequest{
		Name:     "Handler User",
		Email:    uniqueEmail(),
		Password: "secret",
		Role:     "editor",
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp dto.UserResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to parse response body")

	assert.NotZero(t, resp.ID)
	assert.Equal(t, "Handler User", resp.Name)
	assert.Equal(t, "editor", resp.Role)
	assert.False(t, resp.CreatedAt.IsZero())
}

func TestHandler_CreateUser_InvalidBody(t *testing.T) {
	r, _ := setupRouter(t)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/users", bytes.NewBufferString("{not a json"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error":"Некорректный запрос"}`, w.Body.String())
}

func TestHandler_CreateUser_DuplicateEmail(t *testing.T) {
	r, repo := setupRouter(t)

	email := uniqueEmail()
	_, err := repo.CreateUser(context.Background(), &models.User{
		Name:         "Existing",
		Email:        email,
		PasswordHash: "hash",
		Role:         "viewer",
	})
	require.NoError(t, err)

	body, err := json.Marshal(dto.CreateUserRequest{
		Name:     "Duplicate",
		Email:    email,
		Password: "secret",
		Role:     "viewer",
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_GetUserByID(t *testing.T) {
	r, repo := setupRouter(t)

	created, err := repo.CreateUser(context.Background(), &models.User{
		Name:         "Lookup",
		Email:        uniqueEmail(),
		PasswordHash: "hash",
		Role:         "viewer",
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/users/%d", created.ID), nil)
	require.NoError(t, err)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.UserResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, created.ID, resp.ID)
	assert.Equal(t, created.Email, resp.Email)
	assert.Equal(t, created.Name, resp.Name)
}

func TestHandler_GetUserByID_NotFound(t *testing.T) {
	r, _ := setupRouter(t)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/users/999999", nil)
	require.NoError(t, err)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetUserByID_InvalidID(t *testing.T) {
	r, _ := setupRouter(t)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/users/abc", nil)
	require.NoError(t, err)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
