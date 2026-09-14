package service

import (
	"context"
	"errors"
	"fmt"

	"wiki/internal/models"

	"gorm.io/gorm"
)

func (s *Service) Register(ctx context.Context, name, email, password string) (*models.User, error) {
	_, err := s.repo.GetUserByEmail(ctx, email)
	if err == nil {
		return nil, fmt.Errorf("пользователь с такой почтой %s уже есть", email)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hashed, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Name:         name,
		Email:        email,
		PasswordHash: hashed,
		Role:         "viewer",
	}
	return s.repo.CreateUser(ctx, user)
}
