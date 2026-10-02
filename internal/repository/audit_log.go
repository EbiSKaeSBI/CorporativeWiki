package repository

import (
	"context"

	"wiki/internal/models"
)

func (r *Repository) CreateAuditLog(ctx context.Context, log *models.AuditLog) (*models.AuditLog, error) {
	err := r.db.WithContext(ctx).Create(log).Error
	if err != nil {
		return nil, err
	}
	return log, nil
}
