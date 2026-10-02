package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"wiki/internal/dto"
	"wiki/internal/models"
	"wiki/internal/service"
)

func serveReq(t *testing.T, router *gin.Engine, method, path, token string, body []byte) *httptest.ResponseRecorder {
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
	router.ServeHTTP(w, req)
	return w
}

func itoa(u uint) string { return strconv.FormatUint(uint64(u), 10) }

func auditGet(t *testing.T, s *permSetup, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	return permReq(t, s, http.MethodGet, path, token, nil)
}

func TestHandler_RegisterWritesAuditLog(t *testing.T) {
	r, repo := setupArticleRouter(t)
	db := repo.GetDB()
	t.Cleanup(func() { cleanupArticles(t, db) })

	body := []byte(`{"name":"HAudit Reg","email":"` + uniqueEmail() + `","password":"pass123"}`)
	w := serveReq(t, r, http.MethodPost, "/auth/register", "", body)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var user struct {
		ID uint `json:"id"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &user))
	require.NotZero(t, user.ID)
	t.Cleanup(func() {
		db.Unscoped().Where("actor_id = ?", user.ID).Delete(&models.AuditLog{})
	})

	var log models.AuditLog
	require.NoError(t, db.Where("actor_id = ? AND action = ?", user.ID, service.AuditUserCreated).First(&log).Error,
		"POST /auth/register должен оставить USER_CREATED")
	assert.Equal(t, user.ID, *log.TargetUserID)
	assert.Nil(t, log.ArticleID)
}

func TestHandler_LoginWritesAuditLog(t *testing.T) {
	r, repo := setupArticleRouter(t)
	db := repo.GetDB()
	t.Cleanup(func() { cleanupArticles(t, db) })

	email := uniqueEmail()
	regBody := []byte(`{"name":"HAudit Login","email":"` + email + `","password":"pass123"}`)
	w := serveReq(t, r, http.MethodPost, "/auth/register", "", regBody)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	w = serveReq(t, r, http.MethodPost, "/auth/login", "", []byte(`{"email":"`+email+`","password":"pass123"}`))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var user models.User
	require.NoError(t, db.Where("email = ?", email).First(&user).Error)
	t.Cleanup(func() {
		db.Unscoped().Where("actor_id = ?", user.ID).Delete(&models.AuditLog{})
	})

	var n int64
	require.NoError(t, db.Model(&models.AuditLog{}).Where("actor_id = ? AND action = ?", user.ID, service.AuditUserLogin).Count(&n).Error)
	assert.Equal(t, int64(1), n, "успешный логин пишет ровно один USER_LOGIN")
}

func TestHandler_AuditLogsAccess(t *testing.T) {
	s := newPermSetup(t)

	w := auditGet(t, s, "/api/admin/audit-logs", testTokenRole(t, s.adminID, "admin"))
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = auditGet(t, s, "/api/admin/audit-logs", testTokenRole(t, s.viewerID, "editor"))
	assert.Equal(t, http.StatusForbidden, w.Code)

	w = auditGet(t, s, "/api/admin/audit-logs", testTokenRole(t, s.viewerID, "viewer"))
	assert.Equal(t, http.StatusForbidden, w.Code)

	w = auditGet(t, s, "/api/admin/audit-logs", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_AuditLogsPaginationAndFilters(t *testing.T) {
	s := newPermSetup(t)
	ctx := context.Background()
	db := s.repo.GetDB()
	t.Cleanup(func() {
		db.Unscoped().Where("actor_id = ?", s.viewerID).Delete(&models.AuditLog{})
	})

	articleID := s.articleID
	for i := 0; i < 3; i++ {
		_, err := s.repo.CreateAuditLog(ctx, &models.AuditLog{
			ActorID: s.viewerID, Action: service.AuditArticleUpdated, ArticleID: &articleID,
		})
		require.NoError(t, err)
	}
	_, err := s.repo.CreateAuditLog(ctx, &models.AuditLog{
		ActorID: s.viewerID, Action: service.AuditUserLogin,
	})
	require.NoError(t, err)

	adminTok := testTokenRole(t, s.adminID, "admin")

	w := auditGet(t, s, "/api/admin/audit-logs?actor_id="+itoa(s.viewerID)+"&limit=2", adminTok)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var page1 dto.AuditLogListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &page1))
	assert.Len(t, page1.Items, 2)
	assert.EqualValues(t, 4, page1.Total)
	assert.Equal(t, 1, page1.Page)
	assert.Equal(t, 2, page1.Limit)

	w = auditGet(t, s, "/api/admin/audit-logs?actor_id="+itoa(s.viewerID)+"&limit=2&page=2", adminTok)
	var page2 dto.AuditLogListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &page2))
	assert.Len(t, page2.Items, 2)

	w = auditGet(t, s, "/api/admin/audit-logs?actor_id="+itoa(s.viewerID)+"&action="+service.AuditArticleUpdated, adminTok)
	var byAction dto.AuditLogListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &byAction))
	assert.Len(t, byAction.Items, 3)
	for _, item := range byAction.Items {
		assert.Equal(t, service.AuditArticleUpdated, item.Action)
	}

	w = auditGet(t, s, "/api/admin/audit-logs?actor_id="+itoa(s.viewerID)+"&action="+service.AuditUserLogin, adminTok)
	var byLogin dto.AuditLogListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &byLogin))
	assert.Len(t, byLogin.Items, 1)
	assert.Nil(t, byLogin.Items[0].ArticleID)

	w = auditGet(t, s, "/api/admin/audit-logs?actor_id="+itoa(s.viewerID)+"&article_id="+itoa(s.articleID), adminTok)
	var combo dto.AuditLogListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &combo))
	assert.Len(t, combo.Items, 3)

	w = auditGet(t, s, "/api/admin/audit-logs?actor_id=999999", adminTok)
	var empty dto.AuditLogListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &empty))
	assert.Empty(t, empty.Items)
	assert.EqualValues(t, 0, empty.Total)

	w = auditGet(t, s, "/api/admin/audit-logs?action=NOT_REAL", adminTok)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = auditGet(t, s, "/api/admin/audit-logs?page=abc", adminTok)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	for _, bad := range []string{"page=0", "page=-1", "limit=0", "limit=101", "limit=-5"} {
		w = auditGet(t, s, "/api/admin/audit-logs?"+bad, adminTok)
		assert.Equal(t, http.StatusBadRequest, w.Code, "ожидался 400 для ?"+bad)
	}
}
