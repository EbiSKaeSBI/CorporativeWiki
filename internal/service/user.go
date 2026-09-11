package service

import (
	"context"

	"wiki/internal/models"

	"golang.org/x/crypto/bcrypt"
)

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (s *Service) CreateUser(ctx context.Context, name, email, password, role string) (*models.User, error) {
	hashed, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	if role == "" {
		role = "viewer"
	}
	user := &models.User{
		Name:         name,
		Email:        email,
		PasswordHash: hashed,
		Role:         role,
	}
	return s.repo.CreateUser(user)
}

func (s *Service) GetUserByID(ctx context.Context, id uint) (*models.User, error) {
	user, err := s.repo.GetUserByID(id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user, err := s.repo.GetUserByEmail(email)
	if err != nil {
		return nil, err
	}
	return user, nil
}
