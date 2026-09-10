package handler

import "wiki/internal/service"

type Handler struct {
	service *service.Service
}

func NewHandler(serv *service.Service) *Handler {
	return &Handler {
		service: serv,
	}
}

