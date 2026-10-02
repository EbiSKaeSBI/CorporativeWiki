package service

import (
	"context"
	"errors"

	"wiki/internal/repository"

	"wiki/internal/models"

	"gorm.io/gorm"
)

var auditActions = map[string]bool{
	AuditUserCreated:       true,
	AuditUserLogin:         true,
	AuditArticleCreated:    true,
	AuditArticleUpdated:    true,
	AuditArticleDeleted:    true,
	AuditArticleSubmitted:  true,
	AuditArticleApproved:   true,
	AuditArticleRejected:   true,
	AuditPermissionGranted: true,
	AuditPermissionRevoked: true,
}

func (s *Service) GetAuditLogs(ctx context.Context, adminID uint, action string, filterActorID, filterArticleID uint, page, limit int) ([]models.AuditLog, int64, error) {
	user, err := s.repo.GetUserByID(ctx, adminID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, ErrUserNotFound
		}
		return nil, 0, err
	}
	if user.Role != "admin" {
		return nil, 0, ErrForbidden
	}

	if action != "" && !auditActions[action] {
		return nil, 0, ErrInvalidAuditAction
	}

	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return s.repo.GetAuditLogs(ctx, repository.AuditLogFilter{
		Action:    action,
		ActorID:   filterActorID,
		ArticleID: filterArticleID,
	}, page, limit)
}

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
