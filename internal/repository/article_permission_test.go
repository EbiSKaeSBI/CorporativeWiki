package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"wiki/internal/models"

	"gorm.io/gorm"
)

func uniquePermSlug() string {
	return fmt.Sprintf("rtest_%d", time.Now().UnixNano())
}

// newPermissionFixture — реальные user + article (FK не дают вставить мусор)
// и возвращает пару id'ов. Очистка — cleanupPermissionsFixture.
func newPermissionFixture(t *testing.T, repo interface {
	CreateUser(ctx context.Context, user *models.User) (*models.User, error)
	CreateArticle(ctx context.Context, article *models.Article) (*models.Article, error)
}) (articleID, userID uint, slug string) {
	t.Helper()

	user, err := repo.CreateUser(context.Background(), &models.User{
		Name:         "Perm Subject",
		Email:        uniqueEmail(),
		PasswordHash: "hash",
		Role:         "viewer",
	})
	require.NoError(t, err)

	article, err := repo.CreateArticle(context.Background(), &models.Article{
		Title:    "Perm Target",
		Slug:     uniquePermSlug(),
		Content:  "body",
		AuthorID: user.ID,
		Status:   "draft",
	})
	require.NoError(t, err)

	return article.ID, user.ID, article.Slug
}

// cleanupPermissionsFixture — жёстко снимает сначала permissions (FK на
// articles), потом ревизии, потом статью; скоуп — только rtest_ slug'и.
func cleanupPermissionsFixture(t *testing.T, db *gorm.DB, slug string) {
	t.Helper()
	require.NoError(t, db.Unscoped().
		Exec("DELETE FROM article_permissions WHERE article_id IN (SELECT id FROM articles WHERE slug LIKE 'rtest_%')").Error)
	require.NoError(t, db.Unscoped().
		Exec("DELETE FROM article_revisions WHERE article_id IN (SELECT id FROM articles WHERE slug LIKE 'rtest_%')").Error)
	require.NoError(t, db.Unscoped().Where("slug LIKE ?", "rtest_%").Delete(&models.Article{}).Error)
}

func TestRepository_CreateArticlePermission(t *testing.T) {
	repo := newTestRepo(t)
	db := repo.GetDB()

	articleID, userID, slug := newPermissionFixture(t, repo)
	defer cleanupPermissionsFixture(t, db, slug)

	created, err := repo.CreateArticlePermission(context.Background(), &models.ArticlePermission{
		ArticleID:  articleID,
		UserID:     userID,
		Permission: "edit",
	})
	require.NoError(t, err)
	assert.NotZero(t, created.ID)
	assert.Equal(t, articleID, created.ArticleID)
	assert.Equal(t, userID, created.UserID)
	assert.Equal(t, "edit", created.Permission)
}

func TestRepository_CreateArticlePermission_Duplicate(t *testing.T) {
	repo := newTestRepo(t)
	db := repo.GetDB()

	articleID, userID, slug := newPermissionFixture(t, repo)
	defer cleanupPermissionsFixture(t, db, slug)

	_, err := repo.CreateArticlePermission(context.Background(), &models.ArticlePermission{
		ArticleID: articleID, UserID: userID, Permission: "view",
	})
	require.NoError(t, err)

	// вторая запись той же пары (article, user) — падает на UNIQUE индексе
	_, err = repo.CreateArticlePermission(context.Background(), &models.ArticlePermission{
		ArticleID: articleID, UserID: userID, Permission: "edit",
	})
	assert.ErrorIs(t, err, gorm.ErrDuplicatedKey)
}

func TestRepository_CreateArticlePermission_InvalidValue(t *testing.T) {
	repo := newTestRepo(t)
	db := repo.GetDB()

	articleID, userID, slug := newPermissionFixture(t, repo)
	defer cleanupPermissionsFixture(t, db, slug)

	// CHECK в БД держит множество {view, edit} — Repository не фильтрует сам
	_, err := repo.CreateArticlePermission(context.Background(), &models.ArticlePermission{
		ArticleID: articleID, UserID: userID, Permission: "delete",
	})
	assert.Error(t, err, "CHECK constraint must reject non-view/edit values")
}

func TestRepository_GetArticlePermission(t *testing.T) {
	repo := newTestRepo(t)
	db := repo.GetDB()

	articleID, userID, slug := newPermissionFixture(t, repo)
	defer cleanupPermissionsFixture(t, db, slug)

	_, err := repo.CreateArticlePermission(context.Background(), &models.ArticlePermission{
		ArticleID: articleID, UserID: userID, Permission: "view",
	})
	require.NoError(t, err)

	got, err := repo.GetArticlePermission(context.Background(), articleID, userID)
	require.NoError(t, err)
	assert.Equal(t, "view", got.Permission)

	// чужая пара — ошибка наверх как есть (Repository НЕ маппит её)
	_, err = repo.GetArticlePermission(context.Background(), articleID, 0xC0FFEE)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestRepository_GetArticlePermissions(t *testing.T) {
	repo := newTestRepo(t)
	db := repo.GetDB()

	articleID, userID, slug := newPermissionFixture(t, repo)
	defer cleanupPermissionsFixture(t, db, slug)

	// разные пользователи на одну статью
	other, err := repo.CreateUser(context.Background(), &models.User{
		Name: "Second Subject", Email: uniqueEmail(), PasswordHash: "hash", Role: "viewer",
	})
	require.NoError(t, err)

	_, err = repo.CreateArticlePermission(context.Background(), &models.ArticlePermission{
		ArticleID: articleID, UserID: userID, Permission: "view",
	})
	require.NoError(t, err)
	_, err = repo.CreateArticlePermission(context.Background(), &models.ArticlePermission{
		ArticleID: articleID, UserID: other.ID, Permission: "edit",
	})
	require.NoError(t, err)

	list, err := repo.GetArticlePermissions(context.Background(), articleID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	// ORDER BY created_at ASC: при равных created_at (микросекунды могут
	// совпасть) tie-break берём по id — вторая вставка обязана быть второй.
	assert.Less(t, list[0].ID, list[1].ID)
}

func TestRepository_DeleteArticlePermission(t *testing.T) {
	repo := newTestRepo(t)
	db := repo.GetDB()

	articleID, userID, slug := newPermissionFixture(t, repo)
	defer cleanupPermissionsFixture(t, db, slug)

	_, err := repo.CreateArticlePermission(context.Background(), &models.ArticlePermission{
		ArticleID: articleID, UserID: userID, Permission: "edit",
	})
	require.NoError(t, err)

	require.NoError(t, repo.DeleteArticlePermission(context.Background(), articleID, userID))

	_, err = repo.GetArticlePermission(context.Background(), articleID, userID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "soft-deleted row must be invisible to Get")

	// задокументированная тонкость: удалённая строка занимает uix_article_user —
	// повторная выдача той же пары падает на дубликате (лечится в Service)
	_, err = repo.CreateArticlePermission(context.Background(), &models.ArticlePermission{
		ArticleID: articleID, UserID: userID, Permission: "view",
	})
	assert.ErrorIs(t, err, gorm.ErrDuplicatedKey)
}
