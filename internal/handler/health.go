package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)


func (h *Handler) Health(c *gin.Context){
	status := h.service.Check()
	c.JSON(http.StatusOK, gin.H{
		"status": status, 
	})
}

