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

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"wiki/database"
	"wiki/internal/config"
	"wiki/internal/dto"
	"wiki/internal/handler"
	"wiki/internal/middleware"
	"wiki/internal/models"
	"wiki/internal/repository"
	"wiki/internal/service"
)

// newArticleHandler — хелпер для создания handler с репозиторием.
func newArticleHandler(t *testing.T, conf *config.Config) (*handler.Handler, *repository.Repository) {
	db, err := database.Connect(conf)
	require.NoError(t, err, "failed to connect to database")
	repo := repository.NewRepository(db)
	return handler.NewHandler(service.NewService(repo, conf)), repo
}

// uniqueSlug — генерирует уникальный slug для тестов.
func uniqueSlug() string {
	return fmt.Sprintf("slug_%d", time.Now().UnixNano())
}

// cleanupArticles — удаляет статью напрямую через GORM (обходит soft-delete).
func cleanupArticles(t *testing.T, db *gorm.DB) {
	err := db.Unscoped().Where("1 = 1").Delete(&models.Article{}).Error
	require.NoError(t, err, "failed to cleanup articles")
}

// setupArticleRouter — собирает роутер с JWT-защитой для статей.
func setupArticleRouter(t *testing.T) (*gin.Engine, *repository.Repository) {
	gin.SetMode(gin.TestMode)
	conf := config.Load()
	h, repo := newArticleHandler(t, conf)

	r := gin.New()
	r.POST("/auth/register", h.Register)
	r.POST("/auth/login", h.Login)

	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware(conf.JwtSecret))
	{
		api.POST("/articles", h.CreateArticle)
		api.GET("/articles", h.GetArticles)
		api.GET("/articles/:id", h.GetArticle)
		api.PUT("/articles/:id", h.UpdateArticle)
		api.DELETE("/articles/:id", h.DeleteArticle)
	}

	return r, repo
}

func TestHandler_CreateArticle(t *testing.T) {
	r, repo := setupArticleRouter(t)
	defer cleanupArticles(t, repo.GetDB())

	user, err := repo.CreateUser(context.Background(), &models.User{
		Name:         "Article Author",
		Email:        uniqueEmail(),
		PasswordHash: "hash",
		Role:         "editor",
	})
	require.NoError(t, err)
	token := testToken(t, user.ID)

	body, err := json.Marshal(dto.CreateArticleRequest{
		Title:   "Test Article",
		Slug:    uniqueSlug(),
		Content: "Hello, World!",
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/api/articles", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp dto.ArticleResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to parse response body")

	assert.NotZero(t, resp.ID)
	assert.Equal(t, "Test Article", resp.Title)
	assert.NotEmpty(t, resp.Slug)
	assert.Equal(t, "Hello, World!", resp.Content)
	assert.Equal(t, "draft", resp.Status)
	assert.Equal(t, user.ID, resp.AuthorID)
	assert.False(t, resp.CreatedAt.IsZero())
}

func TestHandler_CreateArticle_NoAuth(t *testing.T) {
	r, _ := setupArticleRouter(t)

	body, _ := json.Marshal(dto.CreateArticleRequest{
		Title: "No Auth",
		Slug:  uniqueSlug(),
	})

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/api/articles", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CreateArticle_InvalidBody(t *testing.T) {
	r, _ := setupArticleRouter(t)
	token := testToken(t, 1)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/api/articles", bytes.NewBufferString("{not json"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error":"Некорректный JSON"}`, w.Body.String())
}

func TestHandler_CreateArticle_DuplicateSlug(t *testing.T) {
	r, repo := setupArticleRouter(t)
	defer cleanupArticles(t, repo.GetDB())

	user, err := repo.CreateUser(context.Background(), &models.User{
		Name:         "Duper",
		Email:        uniqueEmail(),
		PasswordHash: "hash",
		Role:         "editor",
	})
	require.NoError(t, err)
	token := testToken(t, user.ID)

	slug := uniqueSlug()
	body, _ := json.Marshal(dto.CreateArticleRequest{Title: "Dup", Slug: slug, Content: "x"})

	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/articles", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w1, req1)
	require.Equal(t, http.StatusCreated, w1.Code)

	body2, _ := json.Marshal(dto.CreateArticleRequest{Title: "Dup2", Slug: slug, Content: "y"})
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/articles", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusInternalServerError, w2.Code)
}

func TestHandler_GetArticle(t *testing.T) {
	r, repo := setupArticleRouter(t)
	defer cleanupArticles(t, repo.GetDB())

	user, err := repo.CreateUser(context.Background(), &models.User{
		Name:         "Getter",
		Email:        uniqueEmail(),
		PasswordHash: "hash",
		Role:         "editor",
	})
	require.NoError(t, err)

	article, err := repo.CreateArticle(context.Background(), &models.Article{
		Title:    "Find Me",
		Slug:     "find-me",
		Content:  "content",
		AuthorID: user.ID,
	})
	require.NoError(t, err)

	token := testToken(t, user.ID)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/articles/%d", article.ID), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.ArticleResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, article.ID, resp.ID)
	assert.Equal(t, "Find Me", resp.Title)
	assert.Equal(t, "find-me", resp.Slug)
	assert.Equal(t, user.ID, resp.AuthorID)
}

func TestHandler_GetArticle_NotFound(t *testing.T) {
	r, _ := setupArticleRouter(t)
	token := testToken(t, 1)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/api/articles/999999", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetArticle_InvalidID(t *testing.T) {
	r, _ := setupArticleRouter(t)
	token := testToken(t, 1)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/api/articles/abc", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error":"Некорректный ID статьи"}`, w.Body.String())
}

func TestHandler_GetArticle_NoAuth(t *testing.T) {
	r, _ := setupArticleRouter(t)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/api/articles/1", nil)
	require.NoError(t, err)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetArticles_Empty(t *testing.T) {
	r, repo := setupArticleRouter(t)
	defer cleanupArticles(t, repo.GetDB())

	token := testToken(t, 1)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/api/articles", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `[]`, w.Body.String())
}

func TestHandler_GetArticles_Multiple(t *testing.T) {
	r, repo := setupArticleRouter(t)
	defer cleanupArticles(t, repo.GetDB())

	user, err := repo.CreateUser(context.Background(), &models.User{
		Name:         "Multi",
		Email:        uniqueEmail(),
		PasswordHash: "hash",
		Role:         "editor",
	})
	require.NoError(t, err)

	_, err = repo.CreateArticle(context.Background(), &models.Article{
		Title: "First", Slug: "first", Content: "1", AuthorID: user.ID,
	})
	require.NoError(t, err)
	_, err = repo.CreateArticle(context.Background(), &models.Article{
		Title: "Second", Slug: "second", Content: "2", AuthorID: user.ID,
	})
	require.NoError(t, err)

	token := testToken(t, user.ID)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/api/articles", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []dto.ArticleResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp, 2)
}

func TestHandler_UpdateArticle(t *testing.T) {
	r, repo := setupArticleRouter(t)
	defer cleanupArticles(t, repo.GetDB())

	user, err := repo.CreateUser(context.Background(), &models.User{
		Name:         "Updater",
		Email:        uniqueEmail(),
		PasswordHash: "hash",
		Role:         "editor",
	})
	require.NoError(t, err)

	article, err := repo.CreateArticle(context.Background(), &models.Article{
		Title: "Old Title", Slug: "old-slug", Content: "old", AuthorID: user.ID,
	})
	require.NoError(t, err)

	token := testToken(t, user.ID)

	newSlug := uniqueSlug()
	updateBody, err := json.Marshal(dto.UpdateArticleRequest{
		Title:   "New Title",
		Slug:    newSlug,
		Content: "new content",
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/articles/%d", article.ID), bytes.NewReader(updateBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.ArticleResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, article.ID, resp.ID)
	assert.Equal(t, "New Title", resp.Title)
	assert.Equal(t, newSlug, resp.Slug)
	assert.Equal(t, "new content", resp.Content)
}

func TestHandler_UpdateArticle_NotFound(t *testing.T) {
	r, _ := setupArticleRouter(t)
	token := testToken(t, 1)

	updateBody, _ := json.Marshal(dto.UpdateArticleRequest{Title: "X"})

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPut, "/api/articles/999999", bytes.NewReader(updateBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_UpdateArticle_InvalidBody(t *testing.T) {
	r, _ := setupArticleRouter(t)
	token := testToken(t, 1)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPut, "/api/articles/1", bytes.NewBufferString("not json"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateArticle_NoAuth(t *testing.T) {
	r, _ := setupArticleRouter(t)

	body, _ := json.Marshal(dto.UpdateArticleRequest{Title: "X"})

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPut, "/api/articles/1", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_UpdateArticle_Forbidden(t *testing.T) {
	r, repo := setupArticleRouter(t)
	defer cleanupArticles(t, repo.GetDB())

	admin, err := repo.CreateUser(context.Background(), &models.User{
		Name:         "Admin",
		Email:        uniqueEmail(),
		PasswordHash: "hash",
		Role:         "admin",
	})
	require.NoError(t, err)

	article, err := repo.CreateArticle(context.Background(), &models.Article{
		Title: "Admin Article", Slug: "admin-art", Content: "x", AuthorID: admin.ID,
	})
	require.NoError(t, err)

	viewer, err := repo.CreateUser(context.Background(), &models.User{
		Name:         "Viewer",
		Email:        uniqueEmail(),
		PasswordHash: "hash",
		Role:         "viewer",
	})
	require.NoError(t, err)
	viewerToken := testToken(t, viewer.ID)

	body, _ := json.Marshal(dto.UpdateArticleRequest{Title: "Hacked"})

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/articles/%d", article.ID), bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+viewerToken)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_GetArticles_NoAuth(t *testing.T) {
	r, _ := setupArticleRouter(t)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/api/articles", nil)
	require.NoError(t, err)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_DeleteArticle(t *testing.T) {
	r, repo := setupArticleRouter(t)
	defer cleanupArticles(t, repo.GetDB())

	user, err := repo.CreateUser(context.Background(), &models.User{
		Name:         "Deleter",
		Email:        uniqueEmail(),
		PasswordHash: "hash",
		Role:         "admin",
	})
	require.NoError(t, err)

	article, err := repo.CreateArticle(context.Background(), &models.Article{
		Title: "To Delete", Slug: "to-delete", Content: "x", AuthorID: user.ID,
	})
	require.NoError(t, err)

	token := testToken(t, user.ID)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/articles/%d", article.ID), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"message":"Статья удалена"}`, w.Body.String())

	_, err = repo.GetArticleByID(context.Background(), article.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestHandler_DeleteArticle_NoAuth(t *testing.T) {
	r, _ := setupArticleRouter(t)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodDelete, "/api/articles/1", nil)
	require.NoError(t, err)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_DeleteArticle_NotFound(t *testing.T) {
	r, _ := setupArticleRouter(t)
	token := testToken(t, 1)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodDelete, "/api/articles/999999", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_DeleteArticle_Forbidden(t *testing.T) {
	r, repo := setupArticleRouter(t)
	defer cleanupArticles(t, repo.GetDB())

	admin, err := repo.CreateUser(context.Background(), &models.User{
		Name:         "Admin2",
		Email:        uniqueEmail(),
		PasswordHash: "hash",
		Role:         "admin",
	})
	require.NoError(t, err)

	article, err := repo.CreateArticle(context.Background(), &models.Article{
		Title: "Protected", Slug: "protected", Content: "x", AuthorID: admin.ID,
	})
	require.NoError(t, err)

	viewer, err := repo.CreateUser(context.Background(), &models.User{
		Name:         "Viewer2",
		Email:        uniqueEmail(),
		PasswordHash: "hash",
		Role:         "viewer",
	})
	require.NoError(t, err)
	viewerToken := testToken(t, viewer.ID)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/articles/%d", article.ID), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+viewerToken)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_DeleteArticle_InvalidID(t *testing.T) {
	r, _ := setupArticleRouter(t)
	token := testToken(t, 1)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodDelete, "/api/articles/abc", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error":"Некорректный ID статьи"}`, w.Body.String())
}
