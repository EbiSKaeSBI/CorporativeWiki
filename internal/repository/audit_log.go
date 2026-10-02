package repository

import (
	"context"

	"wiki/internal/models"

	"gorm.io/gorm"
)

func (r *Repository) CreateAuditLog(ctx context.Context, log *models.AuditLog) (*models.AuditLog, error) {
	err := r.db.WithContext(ctx).Create(log).Error
	if err != nil {
		return nil, err
	}
	return log, nil
}

type AuditLogFilter struct {
	Action    string
	ActorID   uint
	ArticleID uint
}

func (r *Repository) GetAuditLogs(ctx context.Context, f AuditLogFilter, page, limit int) ([]models.AuditLog, int64, error) {
	build := func() *gorm.DB {
		q := r.db.WithContext(ctx).Model(&models.AuditLog{})
		if f.Action != "" {
			q = q.Where("action = ?", f.Action)
		}
		if f.ActorID != 0 {
			q = q.Where("actor_id = ?", f.ActorID)
		}
		if f.ArticleID != 0 {
			q = q.Where("article_id = ?", f.ArticleID)
		}
		return q
	}

	var total int64
	if err := build().Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []models.AuditLog
	err := build().Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
