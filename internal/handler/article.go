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

func (h *Handler) CreateArticle(c *gin.Context) {
	var req dto.CreateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некорректный JSON",
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

func (h *Handler) GetArticle(c *gin.Context) {
	articleStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleStr, 10, 0)
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

func (h *Handler) UpdateArticle(c *gin.Context) {
	var req dto.UpdateArticleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некорректный JSON",
		})
		return
	}
	articleStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleStr, 10, 0)
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

func (h *Handler) DeleteArticle(c *gin.Context) {
	articleStr := c.Param("id")
	articleID, err := strconv.ParseUint(articleStr, 10, 0)
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

	c.JSON(http.StatusOK, gin.H{
		"message": "Статья удалена",
	})
}
