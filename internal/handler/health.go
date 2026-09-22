package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Health возвращает статус приложения
// @Summary Проверка состояния
// @Description Возвращает статус приложения (alive, starting и т.д.)
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "Статус приложения"
// @Router /health [get]
func (h *Handler) Health(c *gin.Context) {
	status := h.service.Check()
	c.JSON(http.StatusOK, gin.H{
		"status": status,
	})
}
