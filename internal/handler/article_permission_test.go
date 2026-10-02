package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"wiki/internal/config"
	"wiki/internal/dto"
	"wiki/internal/models"
	"wiki/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// testTokenRole — JWT с произвольной ролью (testToken жёстко админский).
func testTokenRole(t *testing.T, userID uint, role string) string {
	t.Helper()
	conf := config.Load()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  float64(userID),
		"role": role,
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	s, err := token.SignedString([]byte(conf.JwtSecret))
	require.NoError(t, err)
	return s
}

type permSetup struct {
	r         *gin.Engine
	repo      *repository.Repository
	adminID   uint
	viewerID  uint
	articleID uint
	slug      string
}

// newPermSetup — роутер + admin/viewer/статья в htest_ скоупе, очистка на Cleanup.
func newPermSetup(t *testing.T) *permSetup {
	t.Helper()

	r, repo := setupArticleRouter(t)
	db := repo.GetDB()

	admin, err := repo.CreateUser(context.Background(), &models.User{
		Name: "HPerm Admin", Email: uniqueEmail(), PasswordHash: "hash", Role: "admin",
	})
	require.NoError(t, err)
	viewer, err := repo.CreateUser(context.Background(), &models.User{
		Name: "HPerm Viewer", Email: uniqueEmail(), PasswordHash: "hash", Role: "viewer",
	})
	require.NoError(t, err)

	slug := uniqueSlug()
	article, err := repo.CreateArticle(context.Background(), &models.Article{
		Title: "HPerm Article", Slug: slug, Content: "x", AuthorID: admin.ID, Status: "draft",
	})
	require.NoError(t, err)

	t.Cleanup(func() { cleanupArticles(t, db) })

	return &permSetup{r: r, repo: repo, adminID: admin.ID, viewerID: viewer.ID, articleID: article.ID, slug: slug}
}

func permReq(t *testing.T, s *permSetup, method, path, token string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	if body == nil {
		body = []byte{}
	}
	req, err := http.NewRequest(method, path, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	s.r.ServeHTTP(w, req)
	return w
}

func permURL(id uint, tail ...uint) string {
	url := fmt.Sprintf("/api/articles/%d/permissions", id)
	for _, u := range tail {
		url += fmt.Sprintf("/%d", u)
	}
	return url
}

func TestHandler_CreateArticlePermission(t *testing.T) {
	s := newPermSetup(t)
	body, _ := json.Marshal(dto.CreateArticlePermissionRequest{UserID: s.viewerID, Permission: "edit"})

	w := permReq(t, s, http.MethodPost, permURL(s.articleID), testToken(t, s.adminID), body)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var resp dto.ArticlePermissionResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, s.articleID, resp.ArticleID)
	assert.Equal(t, s.viewerID, resp.UserID)
	assert.Equal(t, "edit", resp.Permission)
	assert.False(t, resp.CreatedAt.IsZero())
}

func TestHandler_GetArticlePermissions(t *testing.T) {
	s := newPermSetup(t)
	body, _ := json.Marshal(dto.CreateArticlePermissionRequest{UserID: s.viewerID, Permission: "view"})
	require.Equal(t, http.StatusCreated,
		permReq(t, s, http.MethodPost, permURL(s.articleID), testToken(t, s.adminID), body).Code)

	w := permReq(t, s, http.MethodGet, permURL(s.articleID), testToken(t, s.adminID), nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var list []dto.ArticlePermissionResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	require.Len(t, list, 1)
	assert.Equal(t, "view", list[0].Permission)
}

func TestHandler_DeleteArticlePermission(t *testing.T) {
	s := newPermSetup(t)
	body, _ := json.Marshal(dto.CreateArticlePermissionRequest{UserID: s.viewerID, Permission: "edit"})
	require.Equal(t, http.StatusCreated,
		permReq(t, s, http.MethodPost, permURL(s.articleID), testToken(t, s.adminID), body).Code)

	w := permReq(t, s, http.MethodDelete, permURL(s.articleID, s.viewerID), testToken(t, s.adminID), nil)
	require.Equal(t, http.StatusNoContent, w.Code, w.Body.String())

	w = permReq(t, s, http.MethodGet, permURL(s.articleID), testToken(t, s.adminID), nil)
	require.Equal(t, http.StatusOK, w.Code)
	var list []dto.ArticlePermissionResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	assert.Len(t, list, 0, "после DELETE право не должно висеть в списке")
}

func TestHandler_ArticlePermission_Errors(t *testing.T) {
	s := newPermSetup(t)
	adminToken := testToken(t, s.adminID)
	viewerToken := testTokenRole(t, s.viewerID, "viewer")
	body, _ := json.Marshal(dto.CreateArticlePermissionRequest{UserID: s.viewerID, Permission: "edit"})

	cases := []struct {
		name   string
		method string
		path   string
		token  string
		body   []byte
		want   int
	}{
		{"bad article id", http.MethodPost, "/api/articles/abc/permissions", adminToken, body, http.StatusBadRequest},
		{"bad user id", http.MethodDelete, "/api/articles/" + fmt.Sprint(s.articleID) + "/permissions/abc", adminToken, nil, http.StatusBadRequest},
		{"wrong json types", http.MethodPost, permURL(s.articleID), adminToken, []byte(`{"user_id": "x"}`), http.StatusBadRequest},
		{"missing fields", http.MethodPost, permURL(s.articleID), adminToken, []byte(`{}`), http.StatusBadRequest},
		{"invalid permission value", http.MethodPost, permURL(s.articleID), adminToken,
			[]byte(`{"user_id": 1, "permission": "delete"}`), http.StatusBadRequest},
		{"no auth", http.MethodPost, permURL(s.articleID), "", body, http.StatusUnauthorized},
		{"non-admin forbidden", http.MethodPost, permURL(s.articleID), viewerToken, body, http.StatusForbidden},
		{"non-admin delete forbidden", http.MethodDelete, permURL(s.articleID, s.viewerID), viewerToken, nil, http.StatusForbidden},
		{"article not found", http.MethodPost, "/api/articles/999999/permissions", adminToken, body, http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := permReq(t, s, tc.method, tc.path, tc.token, tc.body)
			assert.Equal(t, tc.want, w.Code, w.Body.String())
		})
	}
}

func TestHandler_UpdateArticleWithPermission(t *testing.T) {
	s := newPermSetup(t)
	viewerToken := testTokenRole(t, s.viewerID, "viewer")

	updateBody, _ := json.Marshal(dto.UpdateArticleRequest{
		Title: "Edited by Viewer", Slug: s.slug, Content: "new",
	})

	w := permReq(t, s, http.MethodPut, fmt.Sprintf("/api/articles/%d", s.articleID), viewerToken, updateBody)
	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

	grant, _ := json.Marshal(dto.CreateArticlePermissionRequest{UserID: s.viewerID, Permission: "edit"})
	require.Equal(t, http.StatusCreated,
		permReq(t, s, http.MethodPost, permURL(s.articleID), testToken(t, s.adminID), grant).Code)

	w = permReq(t, s, http.MethodPut, fmt.Sprintf("/api/articles/%d", s.articleID), viewerToken, updateBody)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp dto.ArticleResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Edited by Viewer", resp.Title)
}
