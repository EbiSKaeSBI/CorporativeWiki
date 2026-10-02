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

func findAuditLog(t *testing.T, db *gorm.DB, actorID uint, action string) *models.AuditLog {
	var log models.AuditLog
	err := db.Where("actor_id = ? AND action = ?", actorID, action).Order("id desc").First(&log).Error
	require.NoError(t, err, "expected audit log %s for actor %d", action, actorID)
	return &log
}

func countAuditLogs(t *testing.T, db *gorm.DB, actorID uint, action string) int64 {
	var n int64
	require.NoError(t, db.Model(&models.AuditLog{}).Where("actor_id = ? AND action = ?", actorID, action).Count(&n).Error)
	return n
}

func TestAuditUserCreated(t *testing.T) {
	svc, db := newArticleService(t)
	ctx := context.Background()

	email := uniqueEmail()
	user, err := svc.Register(ctx, "Audit Register", email, "pass123")
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Unscoped().Where("actor_id = ?", user.ID).Delete(&models.AuditLog{})
	})

	log := findAuditLog(t, db, user.ID, service.AuditUserCreated)
	assert.Equal(t, user.ID, *log.TargetUserID)
	assert.Nil(t, log.ArticleID)
}

func TestAuditUserLogin(t *testing.T) {
	svc, db := newArticleService(t)
	ctx := context.Background()

	user, err := svc.Register(ctx, "Audit Login", uniqueEmail(), "pass123")
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Unscoped().Where("actor_id = ?", user.ID).Delete(&models.AuditLog{})
	})

	_, err = svc.Login(ctx, user.Email, "pass123")
	require.NoError(t, err)

	assert.Equal(t, int64(1), countAuditLogs(t, db, user.ID, service.AuditUserLogin))

	// неверный пароль — логировать ничего не должны
	_, err = svc.Login(ctx, user.Email, "wrong-password")
	require.ErrorIs(t, err, service.ErrInvalidCredentials)
	assert.Equal(t, int64(1), countAuditLogs(t, db, user.ID, service.AuditUserLogin),
		"неудачный вход не создаёт USER_LOGIN")
}

func TestAuditArticleAndWorkflowEvents(t *testing.T) {
	svc, db := newArticleService(t)
	ctx := context.Background()

	author, err := svc.CreateUser(ctx, "Audit Author", uniqueEmail(), "pass123", "viewer")
	require.NoError(t, err)
	editor, err := svc.CreateUser(ctx, "Audit Editor", uniqueEmail(), "pass123", "editor")
	require.NoError(t, err)

	slug := uniqueArticleSlug()
	article, err := svc.CreateArticle(ctx, author.ID, "Audit Title", slug, "v1")
	require.NoError(t, err)
	defer cleanupArticlesBySlugs(t, db, slug, slug+"-r")

	_, err = svc.UpdateArticle(ctx, article.ID, author.ID, models.Article{Title: "Audit Title", Slug: slug, Content: "v2"})
	require.NoError(t, err)

	_, err = svc.SubmitArticle(ctx, article.ID, author.ID)
	require.NoError(t, err)

	_, err = svc.ApproveArticle(ctx, article.ID, editor.ID)
	require.NoError(t, err)

	logCreated := findAuditLog(t, db, author.ID, service.AuditArticleCreated)
	assert.Equal(t, article.ID, *logCreated.ArticleID)
	assert.NotNil(t, findAuditLog(t, db, author.ID, service.AuditArticleUpdated))
	assert.NotNil(t, findAuditLog(t, db, author.ID, service.AuditArticleSubmitted))

	logApproved := findAuditLog(t, db, editor.ID, service.AuditArticleApproved)
	assert.Equal(t, article.ID, *logApproved.ArticleID)

	rejected, err := svc.CreateArticle(ctx, author.ID, "Audit Reject", slug+"-r", "body")
	require.NoError(t, err)
	_, err = svc.SubmitArticle(ctx, rejected.ID, author.ID)
	require.NoError(t, err)
	_, err = svc.RejectArticle(ctx, rejected.ID, editor.ID)
	require.NoError(t, err)
	logRejected := findAuditLog(t, db, editor.ID, service.AuditArticleRejected)
	assert.Equal(t, rejected.ID, *logRejected.ArticleID)

	require.NoError(t, svc.DeleteArticle(ctx, author.ID, article.ID))
	logDeleted := findAuditLog(t, db, author.ID, service.AuditArticleDeleted)
	assert.NotNil(t, logDeleted.ArticleID)
}

func TestAuditPermissionEvents(t *testing.T) {
	svc, db := newArticleService(t)
	ctx := context.Background()

	admin, err := svc.CreateUser(ctx, "Audit Admin", uniqueEmail(), "pass123", "admin")
	require.NoError(t, err)
	viewer, err := svc.CreateUser(ctx, "Audit Target", uniqueEmail(), "pass123", "viewer")
	require.NoError(t, err)

	slug := uniqueArticleSlug()
	article, err := svc.CreateArticle(ctx, admin.ID, "Audit Perm", slug, "body")
	require.NoError(t, err)
	defer cleanupArticlesBySlugs(t, db, slug)

	_, err = svc.CreateArticlePermission(ctx, admin.ID, article.ID, viewer.ID, service.PermissionView)
	require.NoError(t, err)

	logGranted := findAuditLog(t, db, admin.ID, service.AuditPermissionGranted)
	assert.Equal(t, article.ID, *logGranted.ArticleID)
	assert.Equal(t, viewer.ID, *logGranted.TargetUserID)
	assert.Equal(t, "permission=view", logGranted.Details)

	require.NoError(t, svc.DeleteArticlePermission(ctx, admin.ID, article.ID, viewer.ID))

	logRevoked := findAuditLog(t, db, admin.ID, service.AuditPermissionRevoked)
	assert.Equal(t, article.ID, *logRevoked.ArticleID)
	assert.Equal(t, viewer.ID, *logRevoked.TargetUserID)
	assert.Equal(t, "permission=view", logRevoked.Details)
}

func TestGetAuditLogsAdminOnly(t *testing.T) {
	svc, db := newArticleService(t)
	ctx := context.Background()

	admin, err := svc.CreateUser(ctx, "List Admin", uniqueEmail(), "pass123", "admin")
	require.NoError(t, err)
	viewer, err := svc.CreateUser(ctx, "List Viewer", uniqueEmail(), "pass123", "viewer")
	require.NoError(t, err)
	defer db.Unscoped().Where("actor_id IN ?", []uint{admin.ID, viewer.ID}).Delete(&models.AuditLog{})

	_, _, err = svc.GetAuditLogs(ctx, viewer.ID, "", 0, 0, 1, 20)
	assert.ErrorIs(t, err, service.ErrForbidden)

	slogs, total, err := svc.GetAuditLogs(ctx, admin.ID, "", 0, 0, 1, 5)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(slogs), 5)
	assert.GreaterOrEqual(t, total, int64(0))
}

func TestGetAuditLogsFiltersAndPagination(t *testing.T) {
	svc, db := newArticleService(t)
	ctx := context.Background()

	admin, err := svc.CreateUser(ctx, "Filter Admin", uniqueEmail(), "pass123", "admin")
	require.NoError(t, err)
	target, err := svc.CreateUser(ctx, "Filter Target", uniqueEmail(), "pass123", "viewer")
	require.NoError(t, err)
	defer db.Unscoped().Where("actor_id IN ?", []uint{admin.ID, target.ID}).Delete(&models.AuditLog{})

	for i := 0; i < 3; i++ {
		_, err = svc.CreateAuditLog(ctx, target.ID, service.AuditArticleUpdated, nil, nil, "")
		require.NoError(t, err)
	}
	_, err = svc.CreateAuditLog(ctx, target.ID, service.AuditUserLogin, nil, nil, "")
	require.NoError(t, err)

	slogs, total, err := svc.GetAuditLogs(ctx, admin.ID, service.AuditArticleUpdated, target.ID, 0, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	require.Len(t, slogs, 2)
	for _, l := range slogs {
		assert.Equal(t, service.AuditArticleUpdated, l.Action)
		assert.Equal(t, target.ID, l.ActorID)
	}

	page2, total2, err := svc.GetAuditLogs(ctx, admin.ID, service.AuditArticleUpdated, target.ID, 0, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total2)
	require.Len(t, page2, 1)

	_, _, err = svc.GetAuditLogs(ctx, admin.ID, "NOT_A_REAL_ACTION", 0, 0, 1, 20)
	assert.ErrorIs(t, err, service.ErrInvalidAuditAction)

	empty, emptyTotal, err := svc.GetAuditLogs(ctx, admin.ID, "", target.ID, 999999, 1, 20)
	require.NoError(t, err)
	assert.Empty(t, empty)
	assert.Equal(t, int64(0), emptyTotal)
}
