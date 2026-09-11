package repository

import "wiki/internal/models"

func (r *Repository) GetUserByEmail(email string) (*models.User, error) {
	var user *models.User
	if err := r.db.Where("email=?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return user, nil
}
