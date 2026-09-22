package handler

import (
	"errors"
	"net/http"

	"wiki/internal/dto"
	"wiki/internal/service"

	"github.com/gin-gonic/gin"
)

// Register создаёт нового пользователя (публичная регистрация)
// @Summary Регистрация
// @Description Создаёт нового пользователя через публичный эндпоинт регистрации
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "Данные регистрации"
// @Success 201 {object} dto.UserResponse
// @Failure 400 {object} dto.ErrorResponse "Некорректный запрос (невалидный JSON)"
// @Failure 409 {object} dto.ErrorResponse "Пользователь с таким email уже существует"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некорректный запрос",
		})
		return
	}
	user, err := h.service.Register(
		c.Request.Context(),
		req.Name,
		req.Email,
		req.Password,
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

// Login выполняет аутентификацию пользователя
// @Summary Вход в систему
// @Description Проверяет логин и пароль, возвращает JWT-токен и данные пользователя
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Учётные данные"
// @Success 200 {object} dto.LoginResponse
// @Failure 400 {object} dto.ErrorResponse "Некорректный запрос (невалидный JSON)"
// @Failure 401 {object} dto.ErrorResponse "Неверный email или пароль"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Некорректный запрос",
		})
		return
	}
	resp, err := h.service.Login(
		c.Request.Context(),
		req.Email,
		req.Password,
	)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, dto.LoginResponse{
		AccessToken: resp.AccessToken,
		User: dto.UserResponse{
			ID:        resp.User.ID,
			Name:      resp.User.Name,
			Email:     resp.User.Email,
			Role:      resp.User.Role,
			CreatedAt: resp.User.CreatedAt,
			UpdatedAt: resp.User.UpdatedAt,
		},
	})
}

// Profile возвращает профиль текущего авторизованного пользователя
// @Summary Профиль пользователя
// @Description Возвращает данные текущего пользователя по JWT-токену в заголовке Authorization
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.UserResponse
// @Failure 401 {object} dto.ErrorResponse "Невалидный или отсутствующий токен"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/profile [get]
func (h *Handler) Profile(c *gin.Context) {
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

	user, err := h.service.GetUserByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	})
}
