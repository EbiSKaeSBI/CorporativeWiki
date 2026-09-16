package service

import (
	"context"
	"errors"

	"wiki/internal/models"

	"golang.org/x/crypto/bcrypt"
)

func (s *Service) Register(ctx context.Context, name, email, password string) (*models.User, error) {
	return s.CreateUser(ctx, name, email, password, "viewer")
}

func (s *Service) Login(ctx context.Context, email, password string) (*models.User, error) {
	user, err := s.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return user, nil
}
