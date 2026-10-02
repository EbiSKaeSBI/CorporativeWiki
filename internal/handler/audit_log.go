package handler

import (
	"errors"
	"net/http"
	"strconv"

	"wiki/internal/dto"
	"wiki/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetAuditLogs godoc
// @Summary Список audit-логов (админ)
// @Description Пагинация и фильтры: action, actor_id, article_id
// @Tags audit
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "страница (по умолчанию 1)"
// @Param limit query int false "размер страницы (по умолчанию 20, максимум 100)"
// @Param action query string false "фильтр по действию"
// @Param actor_id query int false "фильтр по актору"
// @Param article_id query int false "фильтр по статье"
// @Success 200 {object} dto.AuditLogListResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/admin/audit-logs [get]
func (h *Handler) GetAuditLogs(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page"})
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
		return
	}

	var filterActor, filterArticle uint64
	if v := c.Query("actor_id"); v != "" {
		filterActor, err = strconv.ParseUint(v, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid actor_id"})
			return
		}
	}
	if v := c.Query("article_id"); v != "" {
		filterArticle, err = strconv.ParseUint(v, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid article_id"})
			return
		}
	}

	logs, total, err := h.service.GetAuditLogs(c.Request.Context(), userID.(uint), c.Query("action"), uint(filterActor), uint(filterArticle), page, limit)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case errors.Is(err, service.ErrUserNotFound):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		case errors.Is(err, service.ErrInvalidAuditAction):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action"})
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	items := make([]dto.AuditLogResponse, 0, len(logs))
	for _, log := range logs {
		items = append(items, dto.AuditLogResponse{
			ID:           log.ID,
			ActorID:      log.ActorID,
			Action:       log.Action,
			ArticleID:    log.ArticleID,
			TargetUserID: log.TargetUserID,
			Details:      log.Details,
			CreatedAt:    log.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, dto.AuditLogListResponse{
		Items: items,
		Page:  page,
		Limit: limit,
		Total: total,
	})
}

