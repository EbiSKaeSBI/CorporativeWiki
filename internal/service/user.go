package service

import (
	"context"
	"errors"

	"wiki/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (s *Service) CreateUser(ctx context.Context, name, email, password, role string) (*models.User, error) {
	_, err := s.repo.GetUserByEmail(ctx, email)
	if err == nil {
		return nil, ErrUserAlreadyExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

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
	return s.repo.CreateUser(ctx, user)
}

func (s *Service) GetUserByID(ctx context.Context, id uint) (*models.User, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (s *Service) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user, err := s.repo.GetUserByEmail(ctx,email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound){
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}
