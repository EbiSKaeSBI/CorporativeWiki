package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"wiki/internal/models"
)

func TestCreateAuditLog(t *testing.T) {
	repo := newTestRepo(t)
	db := repo.GetDB()

	articleID, userID, slug := newPermissionFixture(t, repo)
	t.Cleanup(func() { cleanupPermissionsFixture(t, db, slug) })

	t.Cleanup(func() {
		require.NoError(t, db.Unscoped().Where("actor_id = ? OR target_user_id = ?", userID, userID).Delete(&models.AuditLog{}).Error)
	})

	log, err := repo.CreateAuditLog(context.Background(), &models.AuditLog{
		ActorID:      userID,
		Action:       "ARTICLE_CREATED",
		ArticleID:    &articleID,
		TargetUserID: nil,
		Details:      "title=Perm Target",
	})
	require.NoError(t, err)
	assert.NotZero(t, log.ID)
	assert.False(t, log.CreatedAt.IsZero())
	assert.Equal(t, userID, log.ActorID)
	assert.Equal(t, articleID, *log.ArticleID)
	assert.Nil(t, log.TargetUserID)
}

func TestCreateAuditLog_LoginNoRefs(t *testing.T) {
	repo := newTestRepo(t)
	db := repo.GetDB()

	_, userID, slug := newPermissionFixture(t, repo)
	t.Cleanup(func() { cleanupPermissionsFixture(t, db, slug) })

	t.Cleanup(func() {
		require.NoError(t, db.Unscoped().Where("actor_id = ?", userID).Delete(&models.AuditLog{}).Error)
	})

	log, err := repo.CreateAuditLog(context.Background(), &models.AuditLog{
		ActorID: userID,
		Action:  "USER_LOGIN",
	})
	require.NoError(t, err)
	assert.NotZero(t, log.ID)
	assert.Nil(t, log.ArticleID, "USER_LOGIN не должен иметь article_id")
	assert.Nil(t, log.TargetUserID)
}

func TestCreateAuditLog_UnknownActorFailsFK(t *testing.T) {
	repo := newTestRepo(t)

	_, err := repo.CreateAuditLog(context.Background(), &models.AuditLog{
		ActorID: 999999,
		Action:  "USER_CREATED",
	})
	require.Error(t, err, "FK actor_id должен отбить несуществующего пользователя")
}
