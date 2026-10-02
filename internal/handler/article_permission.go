package handler

import (
	"errors"
	"net/http"
	"strconv"

	"wiki/internal/dto"
	"wiki/internal/models"
	"wiki/internal/service"

	"github.com/gin-gonic/gin"
)

func currentUserIDAndRole(c *gin.Context) (uint, bool) {
	v, exists := c.Get("user_id")
	id, ok := v.(uint)
	return id, exists && ok
}

func toArticlePermissionResponse(p models.ArticlePermission) dto.ArticlePermissionResponse {
	return dto.ArticlePermissionResponse{
		ID:         p.ID,
		ArticleID:  p.ArticleID,
		UserID:     p.UserID,
		Permission: p.Permission,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
}

// @Summary Выдать или изменить право пользователя на статью
// @Description Только admin. Повторная выдача той же пары (article, user) обновляет permission, а soft-deleted запись восстанавливается
// @Tags articles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID статьи"
// @Param request body dto.CreateArticlePermissionRequest true "Кому и какое право"
// @Success 201 {object} dto.ArticlePermissionResponse
// @Failure 400 {object} dto.ErrorResponse "Некорректные данные или ID"
// @Failure 401 {object} dto.ErrorResponse "Невалидный или отсутствующий токен"
// @Failure 403 {object} dto.ErrorResponse "Только admin"
// @Failure 404 {object} dto.ErrorResponse "Статья или пользователь не найдены"
// @Failure 409 {object} dto.ErrorResponse "Гонка: право уже создаётся"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/articles/{id}/permissions [post]
func (h *Handler) CreateArticlePermission(c *gin.Context) {
	articleStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некорректный ID статьи",
		})
		return
	}

	var req dto.CreateArticlePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некорректные данные",
		})
		return
	}

	actorID, ok := currentUserIDAndRole(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	permission, err := h.service.CreateArticlePermission(c.Request.Context(), actorID, uint(articleID), req.UserID, req.Permission)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			c.JSON(http.StatusForbidden, gin.H{
				"error": err.Error(),
			})
		case errors.Is(err, service.ErrArticleNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
		case errors.Is(err, service.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
		case errors.Is(err, service.ErrInvalidArticlePermission):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
		case errors.Is(err, service.ErrArticlePermissionAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusCreated, toArticlePermissionResponse(*permission))
}

// @Summary Права на статью
// @Description Список permissions конкретной статьи для любого авторизованного пользователя
// @Tags articles
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID статьи"
// @Success 200 {array} dto.ArticlePermissionResponse
// @Failure 400 {object} dto.ErrorResponse "Некорректный ID статьи"
// @Failure 401 {object} dto.ErrorResponse "Невалидный или отсутствующий токен"
// @Failure 404 {object} dto.ErrorResponse "Статья не найдена"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/articles/{id}/permissions [get]
func (h *Handler) GetArticlePermissions(c *gin.Context) {
	articleStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некорректный ID статьи",
		})
		return
	}

	permissions, err := h.service.GetArticlePermissions(c.Request.Context(), uint(articleID))
	if err != nil {
		if errors.Is(err, service.ErrArticleNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	resp := make([]dto.ArticlePermissionResponse, 0, len(permissions))
	for _, p := range permissions {
		resp = append(resp, toArticlePermissionResponse(p))
	}
	c.JSON(http.StatusOK, resp)
}

// @Summary Отозвать право пользователя на статью
// @Description Только admin. Soft delete пары (article, user); успешный ответ — пустое тело
// @Tags articles
// @Security BearerAuth
// @Param id path int true "ID статьи"
// @Param userID path int true "ID пользователя"
// @Success 204 "Право отозвано"
// @Failure 400 {object} dto.ErrorResponse "Некорректный ID статьи или пользователя"
// @Failure 401 {object} dto.ErrorResponse "Невалидный или отсутствующий токен"
// @Failure 403 {object} dto.ErrorResponse "Только admin"
// @Failure 404 {object} dto.ErrorResponse "Статья или право не найдены"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/articles/{id}/permissions/{userID} [delete]
func (h *Handler) DeleteArticlePermission(c *gin.Context) {
	articleStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некорректный ID статьи",
		})
		return
	}

	userStr := c.Param("userID")
	targetUserID, err := strconv.ParseUint(userStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некорректный ID пользователя",
		})
		return
	}

	actorID, ok := currentUserIDAndRole(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	if err := h.service.DeleteArticlePermission(c.Request.Context(), actorID, uint(articleID), uint(targetUserID)); err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			c.JSON(http.StatusForbidden, gin.H{
				"error": err.Error(),
			})
		case errors.Is(err, service.ErrArticleNotFound),
			errors.Is(err, service.ErrArticlePermissionNotFound),
			errors.Is(err, service.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		}
		return
	}

	c.Status(http.StatusNoContent)
}
