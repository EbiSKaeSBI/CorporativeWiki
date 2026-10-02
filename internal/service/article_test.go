package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"wiki/database"
	"wiki/internal/config"
	"wiki/internal/models"
	"wiki/internal/repository"
	"wiki/internal/service"
)

// newArticleService — сервис + доступ к db для очистки статей после теста.
func newArticleService(t *testing.T) (*service.Service, *gorm.DB) {
	conf := config.Load()
	db, err := database.Connect(conf)
	require.NoError(t, err, "failed to connect to database")
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.Article{}, &models.ArticleRevision{}, &models.ArticlePermission{}, &models.AuditLog{}), "failed to run migrations")
	return service.NewService(repository.NewRepository(db), conf), db
}

func uniqueArticleSlug() string {
	return fmt.Sprintf("art_%d", time.Now().UnixNano())
}

func cleanupArticlesBySlugs(t *testing.T, db *gorm.DB, slugs ...string) {
	// audit_logs и permissions висят FK на articles — сначала они, потом статьи.
	err := db.Unscoped().Exec("DELETE FROM audit_logs WHERE article_id IN (SELECT id FROM articles WHERE slug IN ?)", slugs).Error
	require.NoError(t, err, "failed to cleanup audit logs")
	err = db.Unscoped().Exec("DELETE FROM article_permissions WHERE article_id IN (SELECT id FROM articles WHERE slug IN ?)", slugs).Error
	require.NoError(t, err, "failed to cleanup article permissions")
	err = db.Unscoped().Exec("DELETE FROM article_revisions WHERE article_id IN (SELECT id FROM articles WHERE slug IN ?)", slugs).Error
	require.NoError(t, err, "failed to cleanup article revisions")
	err = db.Unscoped().Where("slug IN ?", slugs).Delete(&models.Article{}).Error
	require.NoError(t, err, "failed to cleanup articles")
}

func TestSubmitArticle(t *testing.T) {
	svc, db := newArticleService(t)
	slug := uniqueArticleSlug()
	defer cleanupArticlesBySlugs(t, db, slug)

	user, err := svc.CreateUser(context.Background(), "Author", uniqueEmail(), "pass123", "viewer")
	require.NoError(t, err)

	article, err := svc.CreateArticle(context.Background(), user.ID, "Submit Me", slug, "content")
	require.NoError(t, err)
	require.Equal(t, "draft", article.Status)

	submitted, err := svc.SubmitArticle(context.Background(), article.ID, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "pending", submitted.Status)
}

func TestSubmitArticle_NotAuthor(t *testing.T) {
	svc, db := newArticleService(t)
	slug := uniqueArticleSlug()
	defer cleanupArticlesBySlugs(t, db, slug)

	author, err := svc.CreateUser(context.Background(), "Author2", uniqueEmail(), "pass123", "viewer")
	require.NoError(t, err)
	other, err := svc.CreateUser(context.Background(), "Other", uniqueEmail(), "pass123", "admin")
	require.NoError(t, err)

	article, err := svc.CreateArticle(context.Background(), author.ID, "Protected", slug, "content")
	require.NoError(t, err)

	// даже admin не может отправить чужую статью — submit только для автора
	_, err = svc.SubmitArticle(context.Background(), article.ID, other.ID)
	assert.ErrorIs(t, err, service.ErrForbidden)
}

func TestSubmitArticle_NotDraft(t *testing.T) {
	svc, db := newArticleService(t)
	slug := uniqueArticleSlug()
	defer cleanupArticlesBySlugs(t, db, slug)

	user, err := svc.CreateUser(context.Background(), "Author3", uniqueEmail(), "pass123", "editor")
	require.NoError(t, err)
	article, err := svc.CreateArticle(context.Background(), user.ID, "Twice", slug, "content")
	require.NoError(t, err)

	_, err = svc.SubmitArticle(context.Background(), article.ID, user.ID)
	require.NoError(t, err)

	// повторный submit: pending → pending запрещён
	_, err = svc.SubmitArticle(context.Background(), article.ID, user.ID)
	assert.ErrorIs(t, err, service.ErrInvalidArticleStatus)
}

func TestSubmitArticle_NotFound(t *testing.T) {
	svc, _ := newArticleService(t)

	user, err := svc.CreateUser(context.Background(), "Author4", uniqueEmail(), "pass123", "viewer")
	require.NoError(t, err)

	_, err = svc.SubmitArticle(context.Background(), 999999, user.ID)
	assert.ErrorIs(t, err, service.ErrArticleNotFound)
}

func TestApproveArticle(t *testing.T) {
	svc, db := newArticleService(t)
	slug := uniqueArticleSlug()
	defer cleanupArticlesBySlugs(t, db, slug)

	user, err := svc.CreateUser(context.Background(), "Editor1", uniqueEmail(), "pass123", "editor")
	require.NoError(t, err)
	article, err := svc.CreateArticle(context.Background(), user.ID, "To Approve", slug, "content")
	require.NoError(t, err)

	// draft → approve запрещён
	_, err = svc.ApproveArticle(context.Background(), article.ID)
	assert.ErrorIs(t, err, service.ErrInvalidArticleStatus)

	// draft → pending → approve разрешён
	_, err = svc.SubmitArticle(context.Background(), article.ID, user.ID)
	require.NoError(t, err)

	published, err := svc.ApproveArticle(context.Background(), article.ID)
	require.NoError(t, err)
	assert.Equal(t, "published", published.Status)

	// повторный approve опубликованной статьи запрещён
	_, err = svc.ApproveArticle(context.Background(), article.ID)
	assert.ErrorIs(t, err, service.ErrInvalidArticleStatus)
}

func TestApproveArticle_NotFound(t *testing.T) {
	svc, _ := newArticleService(t)

	_, err := svc.ApproveArticle(context.Background(), 999999)
	assert.ErrorIs(t, err, service.ErrArticleNotFound)
}

func TestRejectArticle(t *testing.T) {
	svc, db := newArticleService(t)
	slug := uniqueArticleSlug()
	defer cleanupArticlesBySlugs(t, db, slug)

	user, err := svc.CreateUser(context.Background(), "Editor2", uniqueEmail(), "pass123", "editor")
	require.NoError(t, err)
	article, err := svc.CreateArticle(context.Background(), user.ID, "To Reject", slug, "content")
	require.NoError(t, err)

	// draft → reject запрещён
	_, err = svc.RejectArticle(context.Background(), article.ID)
	assert.ErrorIs(t, err, service.ErrInvalidArticleStatus)

	// draft → pending → reject разрешён
	_, err = svc.SubmitArticle(context.Background(), article.ID, user.ID)
	require.NoError(t, err)

	rejected, err := svc.RejectArticle(context.Background(), article.ID)
	require.NoError(t, err)
	assert.Equal(t, "rejected", rejected.Status)

	// повторный reject запрещён
	_, err = svc.RejectArticle(context.Background(), article.ID)
	assert.ErrorIs(t, err, service.ErrInvalidArticleStatus)
}

func TestRejectArticle_NotFound(t *testing.T) {
	svc, _ := newArticleService(t)

	_, err := svc.RejectArticle(context.Background(), 999999)
	assert.ErrorIs(t, err, service.ErrArticleNotFound)
}

func TestArticleRevision_CreatedOnCreate(t *testing.T) {
	svc, db := newArticleService(t)
	slug := uniqueArticleSlug()
	defer cleanupArticlesBySlugs(t, db, slug)

	user, err := svc.CreateUser(context.Background(), "RevAuthor", uniqueEmail(), "pass123", "viewer")
	require.NoError(t, err)

	article, err := svc.CreateArticle(context.Background(), user.ID, "Initial Title", slug, "initial content")
	require.NoError(t, err)

	revisions, err := svc.GetArticleRevisions(context.Background(), article.ID)
	require.NoError(t, err)
	require.Len(t, revisions, 1, "создание статьи должно давать Revision #1")

	rev := revisions[0]
	assert.Equal(t, article.ID, rev.ArticleID)
	assert.Equal(t, user.ID, rev.EditorID)
	assert.Equal(t, "Initial Title", rev.Title)
	assert.Equal(t, slug, rev.Slug)
	assert.Equal(t, "initial content", rev.Content)
}

func TestArticleRevision_CreatedOnUpdate(t *testing.T) {
	svc, db := newArticleService(t)
	slug := uniqueArticleSlug()
	newSlug := slug + "-v2"
	defer cleanupArticlesBySlugs(t, db, slug, newSlug)

	user, err := svc.CreateUser(context.Background(), "RevEditor", uniqueEmail(), "pass123", "editor")
	require.NoError(t, err)

	article, err := svc.CreateArticle(context.Background(), user.ID, "Go", slug, "Основы Go")
	require.NoError(t, err)

	updated, err := svc.UpdateArticle(context.Background(), article.ID, user.ID, models.Article{
		Title:   "Go Backend",
		Slug:    newSlug,
		Content: "Основы backend-разработки на Go",
	})
	require.NoError(t, err)

	revisions, err := svc.GetArticleRevisions(context.Background(), updated.ID)
	require.NoError(t, err)
	require.Len(t, revisions, 2)

	// ORDER BY created_at DESC: самая свежая — первой, и она хранит НОВОЕ состояние.
	assert.Equal(t, "Go Backend", revisions[0].Title)
	assert.Equal(t, newSlug, revisions[0].Slug)
	assert.Equal(t, "Основы backend-разработки на Go", revisions[0].Content)
	assert.Equal(t, user.ID, revisions[0].EditorID)

	assert.Equal(t, "Go", revisions[1].Title)
	assert.Equal(t, slug, revisions[1].Slug)
}

func TestArticleRevision_NotCreatedOnWorkflow(t *testing.T) {
	svc, db := newArticleService(t)
	slug := uniqueArticleSlug()
	defer cleanupArticlesBySlugs(t, db, slug)

	user, err := svc.CreateUser(context.Background(), "FlowUser", uniqueEmail(), "pass123", "editor")
	require.NoError(t, err)

	article, err := svc.CreateArticle(context.Background(), user.ID, "Flow", slug, "body")
	require.NoError(t, err)

	// workflow-переходы (submit/approve) — это смена состояния, не содержимого:
	// новых ревизий быть не должно (AuditLog будет позже).
	_, err = svc.SubmitArticle(context.Background(), article.ID, user.ID)
	require.NoError(t, err)
	_, err = svc.ApproveArticle(context.Background(), article.ID)
	require.NoError(t, err)

	revisions, err := svc.GetArticleRevisions(context.Background(), article.ID)
	require.NoError(t, err)
	assert.Len(t, revisions, 1, "submit/approve не должны создавать ревизии содержимого")
}

func TestGetArticleRevisions_NotFound(t *testing.T) {
	svc, _ := newArticleService(t)

	_, err := svc.GetArticleRevisions(context.Background(), 99999999)
	assert.ErrorIs(t, err, service.ErrArticleNotFound)
}
