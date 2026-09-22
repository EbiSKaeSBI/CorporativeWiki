package handler

import (
	"errors"
	"net/http"
	"strconv"

	"wiki/internal/dto"
	"wiki/internal/service"

	"github.com/gin-gonic/gin"
)

// CreateUser создаёт нового пользователя через сервисный слой
// @Summary Создать пользователя
// @Description Создаёт нового пользователя в системе (административный эндпоинт)
// @Tags users
// @Accept json
// @Produce json
// @Param request body dto.CreateUserRequest true "Данные пользователя"
// @Success 201 {object} dto.UserResponse
// @Failure 400 {object} dto.ErrorResponse "Некорректный запрос (невалидный JSON)"
// @Failure 409 {object} dto.ErrorResponse "Пользователь с таким email уже существует"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /users [post]
func (h *Handler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некорректный запрос",
		})
		return
	}
	user, err := h.service.CreateUser(
		c.Request.Context(),
		req.Name,
		req.Email,
		req.Password,
		req.Role,
	)
	if err != nil {
		if errors.Is(err, service.ErrUserAlreadyExists) {
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
	resp := dto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
	c.JSON(http.StatusCreated, resp)
}

// GetUserByID возвращает пользователя по идентификатору
// @Summary Получить пользователя по ID
// @Description Возвращает данные пользователя по его числовому идентификатору
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "ID пользователя"
// @Success 200 {object} dto.UserResponse
// @Failure 400 {object} dto.ErrorResponse "Некорректный ID пользователя"
// @Failure 404 {object} dto.ErrorResponse "Пользователь с таким ID не найден"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /users/:id [get]
func (h *Handler) GetUserByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 0)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некорректный ID пользователя",
		})
		return
	}

	user, err := h.service.GetUserByID(
		c.Request.Context(),
		uint(id),
	)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Пользователь с таким ID не существует",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	resp := dto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
	c.JSON(http.StatusOK, resp)
}
