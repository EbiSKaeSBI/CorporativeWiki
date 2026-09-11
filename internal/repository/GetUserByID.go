package repository

import (
	"wiki/internal/models"
)

func (r *Repository) GetUserByID(id uint) (*models.User, error) {
	var user *models.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return user, nil
}
