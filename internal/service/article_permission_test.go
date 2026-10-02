package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"wiki/internal/models"
	"wiki/internal/service"
)

type permFixture struct {
	svc        *service.Service
	admin      *models.User
	viewer     *models.User
	article    *models.Article
	otherAdmin *models.User
}

func adminPermissionFixture(t *testing.T) *permFixture {
	t.Helper()

	svc, db := newArticleService(t)

	admin, err := svc.CreateUser(context.Background(), "Perm Admin", uniqueEmail(), "pass123", "admin")
	require.NoError(t, err)
	otherAdmin, err := svc.CreateUser(context.Background(), "Perm Other Admin", uniqueEmail(), "pass123", "admin")
	require.NoError(t, err)
	viewer, err := svc.CreateUser(context.Background(), "Perm Viewer", uniqueEmail(), "pass123", "viewer")
	require.NoError(t, err)

	slug := uniqueArticleSlug()
	t.Cleanup(func() { cleanupArticlesBySlugs(t, db, slug) })

	article, err := svc.CreateArticle(context.Background(), admin.ID, "Perm Art "+slug, slug, "content")
	require.NoError(t, err)

	return &permFixture{svc: svc, admin: admin, viewer: viewer, article: article, otherAdmin: otherAdmin}
}

func TestArticlePermission_AdminCreatesView(t *testing.T) {
	f := adminPermissionFixture(t)

	perm, err := f.svc.CreateArticlePermission(context.Background(), f.admin.ID, f.article.ID, f.viewer.ID, service.PermissionView)
	require.NoError(t, err)
	assert.Equal(t, service.PermissionView, perm.Permission)

	got, err := f.svc.GetArticlePermission(context.Background(), f.article.ID, f.viewer.ID)
	require.NoError(t, err)
	assert.Equal(t, perm.ID, got.ID)
}

func TestArticlePermission_AdminCreatesEdit(t *testing.T) {
	f := adminPermissionFixture(t)

	perm, err := f.svc.CreateArticlePermission(context.Background(), f.admin.ID, f.article.ID, f.viewer.ID, service.PermissionEdit)
	require.NoError(t, err)
	assert.Equal(t, service.PermissionEdit, perm.Permission)
}

func TestArticlePermission_InvalidValue(t *testing.T) {
	f := adminPermissionFixture(t)

	_, err := f.svc.CreateArticlePermission(context.Background(), f.admin.ID, f.article.ID, f.viewer.ID, "delete")
	require.ErrorIs(t, err, service.ErrInvalidArticlePermission)
}

func TestArticlePermission_NonAdminForbidden(t *testing.T) {
	f := adminPermissionFixture(t)

	_, err := f.svc.CreateArticlePermission(context.Background(), f.viewer.ID, f.article.ID, f.admin.ID, service.PermissionEdit)
	require.ErrorIs(t, err, service.ErrForbidden)
}

func TestArticlePermission_ArticleNotFound(t *testing.T) {
	f := adminPermissionFixture(t)

	_, err := f.svc.CreateArticlePermission(context.Background(), f.admin.ID, 999999, f.viewer.ID, service.PermissionEdit)
	require.ErrorIs(t, err, service.ErrArticleNotFound)
}

func TestArticlePermission_TargetUserNotFound(t *testing.T) {
	f := adminPermissionFixture(t)

	_, err := f.svc.CreateArticlePermission(context.Background(), f.admin.ID, f.article.ID, 999999, service.PermissionEdit)
	require.ErrorIs(t, err, service.ErrUserNotFound)
}

func TestArticlePermission_UpdateInsteadOfDuplicate(t *testing.T) {
	f := adminPermissionFixture(t)

	first, err := f.svc.CreateArticlePermission(context.Background(), f.admin.ID, f.article.ID, f.viewer.ID, service.PermissionView)
	require.NoError(t, err)

	second, err := f.svc.CreateArticlePermission(context.Background(), f.admin.ID, f.article.ID, f.viewer.ID, service.PermissionEdit)
	require.NoError(t, err)

	// та же строка, а не вторая: право обновляется на месте
	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, service.PermissionEdit, second.Permission)

	list, err := f.svc.GetArticlePermissions(context.Background(), f.article.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, service.PermissionEdit, list[0].Permission)

	// другое право от другого админа — тоже не новая запись
	third, err := f.svc.CreateArticlePermission(context.Background(), f.otherAdmin.ID, f.article.ID, f.viewer.ID, service.PermissionView)
	require.NoError(t, err)
	assert.Equal(t, first.ID, third.ID)
	list, err = f.svc.GetArticlePermissions(context.Background(), f.article.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestArticlePermission_ResurrectAfterSoftDelete(t *testing.T) {
	f := adminPermissionFixture(t)

	_, err := f.svc.CreateArticlePermission(context.Background(), f.admin.ID, f.article.ID, f.viewer.ID, service.PermissionEdit)
	require.NoError(t, err)

	require.NoError(t, f.svc.DeleteArticlePermission(context.Background(), f.admin.ID, f.article.ID, f.viewer.ID))

	_, err = f.svc.GetArticlePermission(context.Background(), f.article.ID, f.viewer.ID)
	require.ErrorIs(t, err, service.ErrArticlePermissionNotFound)

	restored, err := f.svc.CreateArticlePermission(context.Background(), f.admin.ID, f.article.ID, f.viewer.ID, service.PermissionView)
	require.NoError(t, err)
	assert.Equal(t, service.PermissionView, restored.Permission)

	list, err := f.svc.GetArticlePermissions(context.Background(), f.article.ID)
	require.NoError(t, err)
	require.Len(t, list, 1, "восстановление не должно плодить вторую запись")
}

func TestArticlePermission_DeleteMissing(t *testing.T) {
	f := adminPermissionFixture(t)

	err := f.svc.DeleteArticlePermission(context.Background(), f.admin.ID, f.article.ID, f.viewer.ID)
	require.ErrorIs(t, err, service.ErrArticlePermissionNotFound)
}

func TestArticlePermission_DeleteNonAdmin(t *testing.T) {
	f := adminPermissionFixture(t)

	_, err := f.svc.CreateArticlePermission(context.Background(), f.admin.ID, f.article.ID, f.viewer.ID, service.PermissionEdit)
	require.NoError(t, err)

	err = f.svc.DeleteArticlePermission(context.Background(), f.viewer.ID, f.article.ID, f.viewer.ID)
	require.ErrorIs(t, err, service.ErrForbidden)
}

func TestHasArticlePermission_Matrix(t *testing.T) {
	f := adminPermissionFixture(t)

	has, err := f.svc.HasArticlePermission(context.Background(), f.article.ID, f.viewer.ID, service.PermissionView)
	require.NoError(t, err)
	assert.False(t, has, "нет права — false, не ошибка")
	has, err = f.svc.HasArticlePermission(context.Background(), f.article.ID, f.viewer.ID, service.PermissionEdit)
	require.NoError(t, err)
	assert.False(t, has)

	_, err = f.svc.CreateArticlePermission(context.Background(), f.admin.ID, f.article.ID, f.viewer.ID, service.PermissionView)
	require.NoError(t, err)
	has, err = f.svc.HasArticlePermission(context.Background(), f.article.ID, f.viewer.ID, service.PermissionView)
	require.NoError(t, err)
	assert.True(t, has)
	has, err = f.svc.HasArticlePermission(context.Background(), f.article.ID, f.viewer.ID, service.PermissionEdit)
	require.NoError(t, err)
	assert.False(t, has, "view не даёт edit")

	_, err = f.svc.CreateArticlePermission(context.Background(), f.admin.ID, f.article.ID, f.viewer.ID, service.PermissionEdit)
	require.NoError(t, err)
	has, err = f.svc.HasArticlePermission(context.Background(), f.article.ID, f.viewer.ID, service.PermissionView)
	require.NoError(t, err)
	assert.True(t, has, "edit включает view")
	has, err = f.svc.HasArticlePermission(context.Background(), f.article.ID, f.viewer.ID, service.PermissionEdit)
	require.NoError(t, err)
	assert.True(t, has)

	_, err = f.svc.HasArticlePermission(context.Background(), f.article.ID, f.viewer.ID, "delete")
	require.ErrorIs(t, err, service.ErrInvalidArticlePermission)
}

func TestArticlePermission_SoftDeletedNotVisible(t *testing.T) {
	f := adminPermissionFixture(t)

	_, err := f.svc.CreateArticlePermission(context.Background(), f.admin.ID, f.article.ID, f.viewer.ID, service.PermissionEdit)
	require.NoError(t, err)
	require.NoError(t, f.svc.DeleteArticlePermission(context.Background(), f.admin.ID, f.article.ID, f.viewer.ID))

	has, err := f.svc.HasArticlePermission(context.Background(), f.article.ID, f.viewer.ID, service.PermissionEdit)
	require.NoError(t, err)
	assert.False(t, has, "soft-deleted право не должно давать доступ")
}

func TestUpdateArticle_EditPermissionOpensForeignArticle(t *testing.T) {
	f := adminPermissionFixture(t)

	_, err := f.svc.UpdateArticle(context.Background(), f.article.ID, f.viewer.ID,
		models.Article{Title: "No Perm", Slug: f.article.Slug, Content: "x"})
	require.ErrorIs(t, err, service.ErrForbidden)

	_, err = f.svc.CreateArticlePermission(context.Background(), f.admin.ID, f.article.ID, f.viewer.ID, service.PermissionEdit)
	require.NoError(t, err)

	updated, err := f.svc.UpdateArticle(context.Background(), f.article.ID, f.viewer.ID,
		models.Article{Title: "Perm Edit", Slug: f.article.Slug, Content: "x"})
	require.NoError(t, err)
	assert.Equal(t, "Perm Edit", updated.Title)
}
