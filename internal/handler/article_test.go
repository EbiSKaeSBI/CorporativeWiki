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
	// Гарантируем схему до запуска тестов (см. комментарий в newTestHandler).
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.Article{}, &models.ArticleRevision{}, &models.ArticlePermission{}, &models.AuditLog{}), "failed to run migrations")
	repo := repository.NewRepository(db)
	return handler.NewHandler(service.NewService(repo, conf)), repo
}

// uniqueSlug — генерирует уникальный slug для тестов (префикс htest_ — под ним работает очистка).
func uniqueSlug() string {
	return fmt.Sprintf("htest_%d", time.Now().UnixNano())
}

// cleanupArticles — удаляет только тестовые статьи (htest_*) напрямую через GORM,
// обходя soft-delete. Таблица общая с тестами пакета service, которые идут
// параллельно: полная зачистка таблицы роняла их на середине работы.
func cleanupArticles(t *testing.T, db *gorm.DB) {
	// Ревизии, permissions и audit-логи висят FK на articles — чистим сначала их.
	err := db.Unscoped().Exec("DELETE FROM audit_logs WHERE article_id IN (SELECT id FROM articles WHERE slug LIKE ?)", "htest_%").Error
	require.NoError(t, err, "failed to cleanup audit logs")
	err = db.Unscoped().Exec("DELETE FROM article_permissions WHERE article_id IN (SELECT id FROM articles WHERE slug LIKE ?)", "htest_%").Error
	require.NoError(t, err, "failed to cleanup article permissions")
	err = db.Unscoped().Exec("DELETE FROM article_revisions WHERE article_id IN (SELECT id FROM articles WHERE slug LIKE ?)", "htest_%").Error
	require.NoError(t, err, "failed to cleanup article revisions")
	err = db.Unscoped().Where("slug LIKE ?", "htest_%").Delete(&models.Article{}).Error
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
		api.GET("/articles/:id/revisions", h.GetArticleRevisions)
		api.GET("/articles/:id/permissions", h.GetArticlePermissions)
		api.POST("/articles/:id/permissions", middleware.RoleMiddleware("admin"), h.CreateArticlePermission)
		api.DELETE("/articles/:id/permissions/:userID", middleware.RoleMiddleware("admin"), h.DeleteArticlePermission)
		api.POST("/articles/:id/submit", h.SubmitArticle)
		api.POST("/articles/:id/approve", h.ApproveArticle)
		api.POST("/articles/:id/reject", h.RejectArticle)
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
	assert.JSONEq(t, `{"error":"Некорректные данные"}`, w.Body.String())
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

	assert.Equal(t, http.StatusConflict, w2.Code)
	assert.JSONEq(t, `{"error":"article already exists"}`, w2.Body.String())
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
		Slug:     "htest_find-me",
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
	assert.Equal(t, "htest_find-me", resp.Slug)
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
		Title: "First", Slug: "htest_first", Content: "1", AuthorID: user.ID,
	})
	require.NoError(t, err)
	_, err = repo.CreateArticle(context.Background(), &models.Article{
		Title: "Second", Slug: "htest_second", Content: "2", AuthorID: user.ID,
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
		Title: "Old Title", Slug: "htest_old-slug", Content: "old", AuthorID: user.ID,
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

	updateBody, _ := json.Marshal(dto.UpdateArticleRequest{Title: "X Title", Slug: uniqueSlug(), Content: "body"})

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
		Title: "Admin Article", Slug: "htest_admin-art", Content: "x", AuthorID: admin.ID,
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

	body, _ := json.Marshal(dto.UpdateArticleRequest{Title: "Hacked", Slug: "htest_hacked-slug", Content: "hacked body"})

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
		Title: "To Delete", Slug: "htest_to-delete", Content: "x", AuthorID: user.ID,
	})
	require.NoError(t, err)

	token := testToken(t, user.ID)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/articles/%d", article.ID), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String(), "204 must have no body")

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
		Title: "Protected", Slug: "htest_protected", Content: "x", AuthorID: admin.ID,
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

// submitArticleReq — шлёт POST /api/articles/:id/submit с данным токеном.
func submitArticleReq(t *testing.T, r *gin.Engine, articleID uint, token string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/articles/%d/submit", articleID), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	return w
}

func TestHandler_SubmitArticle(t *testing.T) {
	r, repo := setupArticleRouter(t)
	defer cleanupArticles(t, repo.GetDB())

	user, err := repo.CreateUser(context.Background(), &models.User{
		Name: "Submitter", Email: uniqueEmail(), PasswordHash: "hash", Role: "viewer",
	})
	require.NoError(t, err)

	article, err := repo.CreateArticle(context.Background(), &models.Article{
		Title: "Ready", Slug: "htest_submit-ok", Content: "x", AuthorID: user.ID,
	})
	require.NoError(t, err)

	w := submitArticleReq(t, r, article.ID, testToken(t, user.ID))

	assert.Equal(t, http.StatusOK, w.Code)
	var resp dto.ArticleResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "pending", resp.Status)
	assert.Equal(t, article.ID, resp.ID)
}

func TestHandler_SubmitArticle_Forbidden(t *testing.T) {
	r, repo := setupArticleRouter(t)
	defer cleanupArticles(t, repo.GetDB())

	author, err := repo.CreateUser(context.Background(), &models.User{
		Name: "Real Author", Email: uniqueEmail(), PasswordHash: "hash", Role: "editor",
	})
	require.NoError(t, err)
	other, err := repo.CreateUser(context.Background(), &models.User{
		Name: "Not Author", Email: uniqueEmail(), PasswordHash: "hash", Role: "admin",
	})
	require.NoError(t, err)

	article, err := repo.CreateArticle(context.Background(), &models.Article{
		Title: "Mine", Slug: "htest_submit-forbidden", Content: "x", AuthorID: author.ID,
	})
	require.NoError(t, err)

	w := submitArticleReq(t, r, article.ID, testToken(t, other.ID))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_SubmitArticle_InvalidStatus(t *testing.T) {
	r, repo := setupArticleRouter(t)
	defer cleanupArticles(t, repo.GetDB())

	user, err := repo.CreateUser(context.Background(), &models.User{
		Name: "Twice Author", Email: uniqueEmail(), PasswordHash: "hash", Role: "editor",
	})
	require.NoError(t, err)

	article, err := repo.CreateArticle(context.Background(), &models.Article{
		Title: "Already Pending", Slug: "htest_submit-twice", Content: "x",
		AuthorID: user.ID, Status: "pending",
	})
	require.NoError(t, err)

	w := submitArticleReq(t, r, article.ID, testToken(t, user.ID))
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.JSONEq(t, `{"error":"invalid article status"}`, w.Body.String())
}

func TestHandler_SubmitArticle_NotFound(t *testing.T) {
	r, _ := setupArticleRouter(t)

	w := submitArticleReq(t, r, 999999, testToken(t, 1))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_SubmitArticle_InvalidID(t *testing.T) {
	r, _ := setupArticleRouter(t)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/api/articles/abc/submit", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+testToken(t, 1))
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SubmitArticle_NoAuth(t *testing.T) {
	r, _ := setupArticleRouter(t)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/api/articles/1/submit", nil)
	require.NoError(t, err)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// approveArticleReq — шлёт POST /api/articles/:id/approve с данным токеном.
func approveArticleReq(t *testing.T, r *gin.Engine, articleID uint, token string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/articles/%d/approve", articleID), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	return w
}

func TestHandler_ApproveArticle(t *testing.T) {
	r, repo := setupArticleRouter(t)
	defer cleanupArticles(t, repo.GetDB())

	user, err := repo.CreateUser(context.Background(), &models.User{
		Name: "Approver", Email: uniqueEmail(), PasswordHash: "hash", Role: "editor",
	})
	require.NoError(t, err)

	article, err := repo.CreateArticle(context.Background(), &models.Article{
		Title: "Waiting", Slug: "htest_approve-ok", Content: "x", AuthorID: user.ID, Status: "pending",
	})
	require.NoError(t, err)

	w := approveArticleReq(t, r, article.ID, testToken(t, user.ID))

	assert.Equal(t, http.StatusOK, w.Code)
	var resp dto.ArticleResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "published", resp.Status)
}

func TestHandler_ApproveArticle_WrongStatus(t *testing.T) {
	r, repo := setupArticleRouter(t)
	defer cleanupArticles(t, repo.GetDB())

	user, err := repo.CreateUser(context.Background(), &models.User{
		Name: "Draft Author", Email: uniqueEmail(), PasswordHash: "hash", Role: "editor",
	})
	require.NoError(t, err)

	article, err := repo.CreateArticle(context.Background(), &models.Article{
		Title: "Still Draft", Slug: "htest_approve-draft", Content: "x", AuthorID: user.ID,
	})
	require.NoError(t, err)

	w := approveArticleReq(t, r, article.ID, testToken(t, user.ID))
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.JSONEq(t, `{"error":"invalid article status"}`, w.Body.String())
}

func TestHandler_ApproveArticle_NotFound(t *testing.T) {
	r, _ := setupArticleRouter(t)

	w := approveArticleReq(t, r, 999999, testToken(t, 1))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_ApproveArticle_InvalidID(t *testing.T) {
	r, _ := setupArticleRouter(t)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/api/articles/abc/approve", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+testToken(t, 1))
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ApproveArticle_NoAuth(t *testing.T) {
	r, _ := setupArticleRouter(t)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/api/articles/1/approve", nil)
	require.NoError(t, err)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// rejectArticleReq — шлёт POST /api/articles/:id/reject с данным токеном.
func rejectArticleReq(t *testing.T, r *gin.Engine, articleID uint, token string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/articles/%d/reject", articleID), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	return w
}

func TestHandler_RejectArticle(t *testing.T) {
	r, repo := setupArticleRouter(t)
	defer cleanupArticles(t, repo.GetDB())

	user, err := repo.CreateUser(context.Background(), &models.User{
		Name: "Rejector", Email: uniqueEmail(), PasswordHash: "hash", Role: "editor",
	})
	require.NoError(t, err)

	article, err := repo.CreateArticle(context.Background(), &models.Article{
		Title: "Waiting2", Slug: "htest_reject-ok", Content: "x", AuthorID: user.ID, Status: "pending",
	})
	require.NoError(t, err)

	w := rejectArticleReq(t, r, article.ID, testToken(t, user.ID))

	assert.Equal(t, http.StatusOK, w.Code)
	var resp dto.ArticleResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "rejected", resp.Status)
}

func TestHandler_RejectArticle_WrongStatus(t *testing.T) {
	r, repo := setupArticleRouter(t)
	defer cleanupArticles(t, repo.GetDB())

	user, err := repo.CreateUser(context.Background(), &models.User{
		Name: "Draft Author2", Email: uniqueEmail(), PasswordHash: "hash", Role: "editor",
	})
	require.NoError(t, err)

	article, err := repo.CreateArticle(context.Background(), &models.Article{
		Title: "Still Draft2", Slug: "htest_reject-draft", Content: "x", AuthorID: user.ID,
	})
	require.NoError(t, err)

	w := rejectArticleReq(t, r, article.ID, testToken(t, user.ID))
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.JSONEq(t, `{"error":"invalid article status"}`, w.Body.String())
}

func TestHandler_RejectArticle_NotFound(t *testing.T) {
	r, _ := setupArticleRouter(t)

	w := rejectArticleReq(t, r, 999999, testToken(t, 1))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_RejectArticle_NoAuth(t *testing.T) {
	r, _ := setupArticleRouter(t)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/api/articles/1/reject", nil)
	require.NoError(t, err)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_GetArticleRevisions(t *testing.T) {
	r, repo := setupArticleRouter(t)
	defer cleanupArticles(t, repo.GetDB())

	user, err := repo.CreateUser(context.Background(), &models.User{
		Name:         "Revision Watcher",
		Email:        uniqueEmail(),
		PasswordHash: "hash",
		Role:         "editor",
	})
	require.NoError(t, err)
	token := testToken(t, user.ID)

	// Создаём статью (Revision #1) и обновляем её (Revision #2).
	slug := uniqueSlug()
	body, err := json.Marshal(dto.CreateArticleRequest{
		Title:   "History Article",
		Slug:    slug,
		Content: "first version",
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, "/api/articles", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created dto.ArticleResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))

	updateBody, err := json.Marshal(dto.UpdateArticleRequest{
		Title:   "History Article v2",
		Slug:    slug + "-b",
		Content: "second version",
	})
	require.NoError(t, err)

	w = httptest.NewRecorder()
	req, err = http.NewRequest(http.MethodPut, "/api/articles/"+fmt.Sprint(created.ID), bytes.NewReader(updateBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req, err = http.NewRequest(http.MethodGet, "/api/articles/"+fmt.Sprint(created.ID)+"/revisions", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var revisions []dto.ArticleRevisionResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &revisions))
	require.Len(t, revisions, 2)

	// свежие первыми; каждая ревизия — снимок, а не ссылка на текущую статью
	assert.Equal(t, "History Article v2", revisions[0].Title)
	assert.Equal(t, "second version", revisions[0].Content)
	assert.Equal(t, created.ID, revisions[0].ArticleID)
	assert.Equal(t, user.ID, revisions[0].EditorID)

	assert.Equal(t, "History Article", revisions[1].Title)
	assert.Equal(t, slug, revisions[1].Slug)
	assert.Equal(t, "first version", revisions[1].Content)
}

func TestHandler_GetArticleRevisions_NotFound(t *testing.T) {
	r, repo := setupArticleRouter(t)
	defer cleanupArticles(t, repo.GetDB())

	user, err := repo.CreateUser(context.Background(), &models.User{
		Name:         "No Revisions",
		Email:        uniqueEmail(),
		PasswordHash: "hash",
		Role:         "viewer",
	})
	require.NoError(t, err)
	token := testToken(t, user.ID)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/api/articles/99999999/revisions", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetArticleRevisions_InvalidID(t *testing.T) {
	r, repo := setupArticleRouter(t)
	defer cleanupArticles(t, repo.GetDB())

	user, err := repo.CreateUser(context.Background(), &models.User{
		Name:         "Bad ID",
		Email:        uniqueEmail(),
		PasswordHash: "hash",
		Role:         "viewer",
	})
	require.NoError(t, err)
	token := testToken(t, user.ID)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/api/articles/abc/revisions", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetArticleRevisions_NoAuth(t *testing.T) {
	r, _ := setupArticleRouter(t)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/api/articles/1/revisions", nil)
	require.NoError(t, err)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
