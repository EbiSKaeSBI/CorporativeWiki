package repository

import (
	"context"

	"wiki/internal/models"
)

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("email=?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
