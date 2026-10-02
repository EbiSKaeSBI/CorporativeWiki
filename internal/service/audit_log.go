package service

import (
	"context"
	"errors"

	"wiki/internal/models"

	"gorm.io/gorm"
)

func uintPtr(u uint) *uint { return &u }

func (s *Service) CreateAuditLog(ctx context.Context, actorID uint, action string, articleID, targetUserID *uint, details string) (*models.AuditLog, error) {
	_, err := s.repo.GetUserByID(ctx, actorID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	log := &models.AuditLog{
		ActorID:      actorID,
		Action:       action,
		ArticleID:    articleID,
		TargetUserID: targetUserID,
		Details:      details,
	}

	return s.repo.CreateAuditLog(ctx, log)
}
