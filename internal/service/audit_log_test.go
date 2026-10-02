package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"wiki/internal/models"
	"wiki/internal/service"

	"gorm.io/gorm"
)

func cleanupAuditLogsByActors(t *testing.T, db *gorm.DB, actorIDs ...uint) {
	t.Helper()
	err := db.Unscoped().Where("actor_id IN ? OR target_user_id IN ?", actorIDs, actorIDs).
		Delete(&models.AuditLog{}).Error
	require.NoError(t, err, "failed to cleanup audit logs")
}

func TestCreateAuditLog_ArticleCreated(t *testing.T) {
	svc, db := newArticleService(t)

	user, err := svc.CreateUser(context.Background(), "Audit Author", uniqueEmail(), "pass123", "viewer")
	require.NoError(t, err)
	slug := uniqueArticleSlug()
	article, err := svc.CreateArticle(context.Background(), user.ID, "Audit Art", slug, "content")
	require.NoError(t, err)
	defer cleanupArticlesBySlugs(t, db, slug)
	defer cleanupAuditLogsByActors(t, db, user.ID)

	created, err := svc.CreateAuditLog(context.Background(), user.ID, service.AuditArticleCreated, &article.ID, nil, "")
	require.NoError(t, err)
	assert.NotZero(t, created.ID)

	var stored models.AuditLog
	require.NoError(t, db.First(&stored, created.ID).Error)
	assert.Equal(t, user.ID, stored.ActorID)
	assert.Equal(t, service.AuditArticleCreated, stored.Action)
	require.NotNil(t, stored.ArticleID)
	assert.Equal(t, article.ID, *stored.ArticleID)
	assert.Nil(t, stored.TargetUserID)
}

func TestCreateAuditLog_UserLoginNoRefs(t *testing.T) {
	svc, db := newArticleService(t)

	user, err := svc.CreateUser(context.Background(), "Audit Login", uniqueEmail(), "pass123", "viewer")
	require.NoError(t, err)
	defer cleanupAuditLogsByActors(t, db, user.ID)

	created, err := svc.CreateAuditLog(context.Background(), user.ID, service.AuditUserLogin, nil, nil, "")
	require.NoError(t, err)

	var stored models.AuditLog
	require.NoError(t, db.First(&stored, created.ID).Error)
	assert.Nil(t, stored.ArticleID, "USER_LOGIN без article_id")
	assert.Nil(t, stored.TargetUserID)
}

func TestCreateAuditLog_PermissionGrantedFullFields(t *testing.T) {
	svc, db := newArticleService(t)

	admin, err := svc.CreateUser(context.Background(), "Audit Admin", uniqueEmail(), "pass123", "admin")
	require.NoError(t, err)
	target, err := svc.CreateUser(context.Background(), "Audit Target", uniqueEmail(), "pass123", "viewer")
	require.NoError(t, err)
	slug := uniqueArticleSlug()
	article, err := svc.CreateArticle(context.Background(), admin.ID, "Audit Perm Art", slug, "content")
	require.NoError(t, err)
	defer cleanupArticlesBySlugs(t, db, slug)
	defer cleanupAuditLogsByActors(t, db, admin.ID, target.ID)

	created, err := svc.CreateAuditLog(context.Background(), admin.ID, service.AuditPermissionGranted,
		&article.ID, &target.ID, "permission=edit")
	require.NoError(t, err)

	var stored models.AuditLog
	require.NoError(t, db.First(&stored, created.ID).Error)
	require.NotNil(t, stored.ArticleID)
	require.NotNil(t, stored.TargetUserID)
	assert.Equal(t, admin.ID, stored.ActorID)
	assert.Equal(t, article.ID, *stored.ArticleID)
	assert.Equal(t, target.ID, *stored.TargetUserID)
	assert.Equal(t, service.AuditPermissionGranted, stored.Action)
	assert.Equal(t, "permission=edit", stored.Details)
}

func TestCreateAuditLog_UnknownActor(t *testing.T) {
	svc, db := newArticleService(t)

	_, err := svc.CreateAuditLog(context.Background(), 999999, service.AuditUserLogin, nil, nil, "")
	require.ErrorIs(t, err, service.ErrUserNotFound)

	var count int64
	require.NoError(t, db.Model(&models.AuditLog{}).Where("actor_id = ?", 999999).Count(&count).Error)
	assert.Zero(t, count, "лог несуществующего actor не должен создаваться")
}

func TestCreateAuditLog_RespectsContext(t *testing.T) {
	svc, db := newArticleService(t)

	user, err := svc.CreateUser(context.Background(), "Audit Ctx", uniqueEmail(), "pass123", "viewer")
	require.NoError(t, err)
	defer cleanupAuditLogsByActors(t, db, user.ID)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = svc.CreateAuditLog(ctx, user.ID, service.AuditUserLogin, nil, nil, "")
	require.Error(t, err, "отменённый context должен прервать запись лога")
}
