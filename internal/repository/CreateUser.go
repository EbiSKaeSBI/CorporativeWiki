package repository

import (
	"context"
	"errors"
	"fmt"

	"wiki/internal/models"

	"gorm.io/gorm"
)

func (r *Repository) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	_, err := r.GetUserByEmail(ctx, user.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
				return nil, err
			}
			return user, nil
		}
		return nil, err
	}
	return nil, fmt.Errorf("Пользователь с такой почтой  %s уже есть", user.Email)
}
