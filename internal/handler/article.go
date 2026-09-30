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

// @Summary Создать статью
// @Description Создаёт новую статью в статусе draft; дубликат slug даёт 409. Первая ревизия истории создаётся автоматически
// @Tags articles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.CreateArticleRequest true "Данные статьи"
// @Success 201 {object} dto.ArticleResponse
// @Failure 400 {object} dto.ErrorResponse "Некорректные данные"
// @Failure 401 {object} dto.ErrorResponse "Невалидный или отсутствующий токен"
// @Failure 409 {object} dto.ErrorResponse "Статья с таким slug уже существует"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/articles [post]
func (h *Handler) CreateArticle(c *gin.Context) {
	var req dto.CreateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некорректные данные",
		})
		return
	}
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Пользователь не найден",
		})
		return
	}

	id, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Пользователь не найден",
		})
		return
	}
	article, err := h.service.CreateArticle(
		c.Request.Context(),
		id,
		req.Title,
		req.Slug,
		req.Content,
	)
	if err != nil {
		if errors.Is(err, service.ErrArticleAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	resp := dto.ArticleResponse{
		ID:        article.ID,
		Title:     article.Title,
		Slug:      article.Slug,
		Content:   article.Content,
		Status:    article.Status,
		AuthorID:  article.AuthorID,
		CreatedAt: article.CreatedAt,
		UpdatedAt: article.UpdatedAt,
	}
	c.JSON(http.StatusCreated, resp)
}

// @Summary Получить статью по ID
// @Tags articles
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID статьи"
// @Success 200 {object} dto.ArticleResponse
// @Failure 400 {object} dto.ErrorResponse "Некорректный ID статьи"
// @Failure 401 {object} dto.ErrorResponse "Невалидный или отсутствующий токен"
// @Failure 404 {object} dto.ErrorResponse "Статья не найдена"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/articles/{id} [get]
func (h *Handler) GetArticle(c *gin.Context) {
	articleStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некорректный ID статьи",
		})
		return
	}
	article, err := h.service.GetArticleByID(c.Request.Context(), uint(articleID))
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
	resp := dto.ArticleResponse{
		ID:        article.ID,
		Title:     article.Title,
		Slug:      article.Slug,
		Content:   article.Content,
		Status:    article.Status,
		AuthorID:  article.AuthorID,
		CreatedAt: article.CreatedAt,
		UpdatedAt: article.UpdatedAt,
	}
	c.JSON(http.StatusOK, resp)
}

// @Summary Список статей
// @Tags articles
// @Produce json
// @Security BearerAuth
// @Success 200 {array} dto.ArticleResponse
// @Failure 401 {object} dto.ErrorResponse "Невалидный или отсутствующий токен"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/articles [get]
func (h *Handler) GetArticles(c *gin.Context) {
	articles, err := h.service.GetArticles(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	resp := make([]dto.ArticleResponse, 0, len(articles))
	for _, a := range articles {
		resp = append(resp, dto.ArticleResponse{
			ID:        a.ID,
			Title:     a.Title,
			Slug:      a.Slug,
			Content:   a.Content,
			Status:    a.Status,
			AuthorID:  a.AuthorID,
			CreatedAt: a.CreatedAt,
			UpdatedAt: a.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, resp)
}

// @Summary Обновить статью
// @Description Автор может менять свою статью, editor/admin — любую. После успешного изменения создаётся ревизия с новым содержимым
// @Tags articles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID статьи"
// @Param request body dto.UpdateArticleRequest true "Новые данные статьи"
// @Success 200 {object} dto.ArticleResponse
// @Failure 400 {object} dto.ErrorResponse "Некорректные данные или ID"
// @Failure 401 {object} dto.ErrorResponse "Невалидный или отсутствующий токен"
// @Failure 403 {object} dto.ErrorResponse "Нет прав на изменение этой статьи"
// @Failure 404 {object} dto.ErrorResponse "Статья не найдена"
// @Failure 409 {object} dto.ErrorResponse "Статья с таким slug уже существует"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/articles/{id} [put]
func (h *Handler) UpdateArticle(c *gin.Context) {
	var req dto.UpdateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некорректные данные",
		})
		return
	}
	articleStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некорректный ID статьи",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Пользователь не найден",
		})
		return
	}

	id, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Пользователь не найден",
		})
		return
	}

	article, err := h.service.UpdateArticle(
		c.Request.Context(),
		uint(articleID),
		id,
		models.Article{
			Title:   req.Title,
			Slug:    req.Slug,
			Content: req.Content,
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrArticleNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
		case errors.Is(err, service.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
		case errors.Is(err, service.ErrForbidden):
			c.JSON(http.StatusForbidden, gin.H{
				"error": err.Error(),
			})
		case errors.Is(err, service.ErrArticleAlreadyExists):
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

	resp := dto.ArticleResponse{
		ID:        article.ID,
		Title:     article.Title,
		Slug:      article.Slug,
		Content:   article.Content,
		Status:    article.Status,
		AuthorID:  article.AuthorID,
		CreatedAt: article.CreatedAt,
		UpdatedAt: article.UpdatedAt,
	}
	c.JSON(http.StatusOK, resp)
}

// @Summary Удалить статью
// @Description Soft-delete: автор или admin. Возвращает пустое тело
// @Tags articles
// @Security BearerAuth
// @Param id path int true "ID статьи"
// @Success 204 "Статья удалена"
// @Failure 400 {object} dto.ErrorResponse "Некорректный ID статьи"
// @Failure 401 {object} dto.ErrorResponse "Невалидный или отсутствующий токен"
// @Failure 403 {object} dto.ErrorResponse "Нет прав на удаление этой статьи"
// @Failure 404 {object} dto.ErrorResponse "Статья не найдена"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/articles/{id} [delete]
func (h *Handler) DeleteArticle(c *gin.Context) {
	articleStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некорректный ID статьи",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Пользователь не найден",
		})
		return
	}

	id, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Пользователь не найден",
		})
		return
	}

	if err := h.service.DeleteArticle(c.Request.Context(), id, uint(articleID)); err != nil {
		switch {
		case errors.Is(err, service.ErrArticleNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
		case errors.Is(err, service.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
		case errors.Is(err, service.ErrForbidden):
			c.JSON(http.StatusForbidden, gin.H{
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

// @Summary Отправить статью на модерацию
// @Description Только автор и только из статуса draft: draft -> pending
// @Tags articles
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID статьи"
// @Success 200 {object} dto.ArticleResponse
// @Failure 400 {object} dto.ErrorResponse "Некорректный ID статьи"
// @Failure 401 {object} dto.ErrorResponse "Невалидный или отсутствующий токен"
// @Failure 403 {object} dto.ErrorResponse "Отправить на модерацию может только автор"
// @Failure 404 {object} dto.ErrorResponse "Статья или пользователь не найдены"
// @Failure 409 {object} dto.ErrorResponse "Статья не в статусе draft"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/articles/{id}/submit [post]
func (h *Handler) SubmitArticle(c *gin.Context) {
	articleStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некорректный ID статьи",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Пользователь не найден",
		})
		return
	}

	id, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Пользователь не найден",
		})
		return
	}

	article, err := h.service.SubmitArticle(
		c.Request.Context(),
		uint(articleID),
		id,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrArticleNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
		case errors.Is(err, service.ErrUserNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
		case errors.Is(err, service.ErrForbidden):
			c.JSON(http.StatusForbidden, gin.H{
				"error": err.Error(),
			})
		case errors.Is(err, service.ErrInvalidArticleStatus):
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

	resp := dto.ArticleResponse{
		ID:        article.ID,
		Title:     article.Title,
		Slug:      article.Slug,
		Content:   article.Content,
		Status:    article.Status,
		AuthorID:  article.AuthorID,
		CreatedAt: article.CreatedAt,
		UpdatedAt: article.UpdatedAt,
	}
	c.JSON(http.StatusOK, resp)
}

// @Summary Одобрить статью
// @Description Editor/admin: pending -> published. Других переходов Service не допускает
// @Tags articles
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID статьи"
// @Success 200 {object} dto.ArticleResponse
// @Failure 400 {object} dto.ErrorResponse "Некорректный ID статьи"
// @Failure 401 {object} dto.ErrorResponse "Невалидный или отсутствующий токен"
// @Failure 403 {object} dto.ErrorResponse "Только editor/admin"
// @Failure 404 {object} dto.ErrorResponse "Статья не найдена"
// @Failure 409 {object} dto.ErrorResponse "Статья не в статусе pending"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/articles/{id}/approve [post]
func (h *Handler) ApproveArticle(c *gin.Context) {
	articleStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некорректный ID статьи",
		})
		return
	}

	article, err := h.service.ApproveArticle(c.Request.Context(), uint(articleID))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrArticleNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
		case errors.Is(err, service.ErrInvalidArticleStatus):
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

	resp := dto.ArticleResponse{
		ID:        article.ID,
		Title:     article.Title,
		Slug:      article.Slug,
		Content:   article.Content,
		Status:    article.Status,
		AuthorID:  article.AuthorID,
		CreatedAt: article.CreatedAt,
		UpdatedAt: article.UpdatedAt,
	}
	c.JSON(http.StatusOK, resp)
}

// @Summary Отклонить статью
// @Description Editor/admin: pending -> rejected. Причина отклонения — позже с AuditLog
// @Tags articles
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID статьи"
// @Success 200 {object} dto.ArticleResponse
// @Failure 400 {object} dto.ErrorResponse "Некорректный ID статьи"
// @Failure 401 {object} dto.ErrorResponse "Невалидный или отсутствующий токен"
// @Failure 403 {object} dto.ErrorResponse "Только editor/admin"
// @Failure 404 {object} dto.ErrorResponse "Статья не найдена"
// @Failure 409 {object} dto.ErrorResponse "Статья не в статусе pending"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/articles/{id}/reject [post]
func (h *Handler) RejectArticle(c *gin.Context) {
	articleStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некорректный ID статьи",
		})
		return
	}

	article, err := h.service.RejectArticle(c.Request.Context(), uint(articleID))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrArticleNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
		case errors.Is(err, service.ErrInvalidArticleStatus):
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

	resp := dto.ArticleResponse{
		ID:        article.ID,
		Title:     article.Title,
		Slug:      article.Slug,
		Content:   article.Content,
		Status:    article.Status,
		AuthorID:  article.AuthorID,
		CreatedAt: article.CreatedAt,
		UpdatedAt: article.UpdatedAt,
	}
	c.JSON(http.StatusOK, resp)
}

// @Summary История изменений статьи
// @Description Ревизии содержимого, свежие первыми. Доступна всем авторизованным до появления ArticlePermission
// @Tags articles
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID статьи"
// @Success 200 {array} dto.ArticleRevisionResponse
// @Failure 400 {object} dto.ErrorResponse "Некорректный ID статьи"
// @Failure 401 {object} dto.ErrorResponse "Невалидный или отсутствующий токен"
// @Failure 404 {object} dto.ErrorResponse "Статья не найдена"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/articles/{id}/revisions [get]
func (h *Handler) GetArticleRevisions(c *gin.Context) {
	articleStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некорректный ID статьи",
		})
		return
	}

	revisions, err := h.service.GetArticleRevisions(c.Request.Context(), uint(articleID))
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

	resp := make([]dto.ArticleRevisionResponse, 0, len(revisions))
	for _, rev := range revisions {
		resp = append(resp, dto.ArticleRevisionResponse{
			ID:        rev.ID,
			ArticleID: rev.ArticleID,
			EditorID:  rev.EditorID,
			Title:     rev.Title,
			Slug:      rev.Slug,
			Content:   rev.Content,
			CreatedAt: rev.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, resp)
}
