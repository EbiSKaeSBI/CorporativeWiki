package repository

import (
	"errors"
	"fmt"

	"wiki/internal/models"

	"gorm.io/gorm"
)

func (r *Repository) CreateUser(user *models.User) (*models.User, error) {
	_, err := r.GetUserByEmail(user.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := r.db.Create(user).Error; err != nil {
				return nil, err
			}
			return user, nil
		}
		return nil, err
	}
	return nil, fmt.Errorf("Пользователь с такой почтой  %s уже есть", user.Email)
}
