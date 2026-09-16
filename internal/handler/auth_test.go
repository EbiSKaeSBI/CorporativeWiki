package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"wiki/internal/dto"
)

func TestHandler_Register(t *testing.T) {
	r, _ := setupRouter(t)

	body, err := json.Marshal(dto.RegisterRequest{
		Name:     "Auth User",
		Email:    uniqueEmail(),
		Password: "secret",
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp dto.UserResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to parse response body")

	assert.NotZero(t, resp.ID)
	assert.Equal(t, "Auth User", resp.Name)
	assert.Equal(t, "viewer", resp.Role)
	assert.False(t, resp.CreatedAt.IsZero())
}

func TestHandler_Register_InvalidBody(t *testing.T) {
	r, _ := setupRouter(t)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString("{bad json"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error":"Некорректный запрос"}`, w.Body.String())
}

func TestHandler_Register_DuplicateEmail(t *testing.T) {
	r, _ := setupRouter(t)

	email := uniqueEmail()
	body, err := json.Marshal(dto.RegisterRequest{
		Name:     "Dup",
		Email:    email,
		Password: "pass",
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusConflict, w2.Code)
}

func TestHandler_Login(t *testing.T) {
	r, _ := setupRouter(t)

	email := uniqueEmail()
	body, err := json.Marshal(dto.RegisterRequest{
		Name:     "Login User",
		Email:    email,
		Password: "secret123",
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	loginBody, err := json.Marshal(dto.LoginRequest{
		Email:    email,
		Password: "secret123",
	})
	require.NoError(t, err)

	w2 := httptest.NewRecorder()
	req2, err := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	require.NoError(t, err)
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	var resp dto.LoginResponse
	err = json.Unmarshal(w2.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to parse response body")

	assert.Equal(t, "Login User", resp.User.Name)
	assert.Equal(t, email, resp.User.Email)
	assert.NotZero(t, resp.User.ID)
	assert.NotEmpty(t, resp.AccessToken, "expected access token in response")
}

func TestHandler_Login_InvalidPassword(t *testing.T) {
	r, _ := setupRouter(t)

	email := uniqueEmail()
	body, err := json.Marshal(dto.RegisterRequest{
		Name:     "Login User",
		Email:    email,
		Password: "correct",
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	loginBody, err := json.Marshal(dto.LoginRequest{
		Email:    email,
		Password: "wrong",
	})
	require.NoError(t, err)

	w2 := httptest.NewRecorder()
	req2, err := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	require.NoError(t, err)
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusUnauthorized, w2.Code)
	assert.JSONEq(t, `{"error":"invalid email or password"}`, w2.Body.String())
}

func TestHandler_Login_UserNotFound(t *testing.T) {
	r, _ := setupRouter(t)

	loginBody, err := json.Marshal(dto.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "any",
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
