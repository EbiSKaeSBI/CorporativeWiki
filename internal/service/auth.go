package service

import (
	"context"

	"wiki/internal/models"
)

func (s *Service) Register(ctx context.Context, name, email, password string) (*models.User, error) {
	return s.CreateUser(ctx, name, email, password, "viewer")
}
