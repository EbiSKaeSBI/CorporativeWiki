package service

import (
	"context"
	"errors"

	"wiki/internal/models"

	"gorm.io/gorm"
)

const (
	PermissionView = "view"
	PermissionEdit = "edit"
)

func validArticlePermission(permission string) bool {
	return permission == PermissionView || permission == PermissionEdit
}

func (s *Service) CreateArticlePermission(ctx context.Context, actorID uint, articleID uint, userID uint, permission string) (*models.ArticlePermission, error) {
	actor, err := s.repo.GetUserByID(ctx, actorID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if actor.Role != "admin" {
		return nil, ErrForbidden
	}

	if _, err := s.repo.GetArticleByID(ctx, articleID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrArticleNotFound
		}
		return nil, err
	}

	if _, err := s.repo.GetUserByID(ctx, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if !validArticlePermission(permission) {
		return nil, ErrInvalidArticlePermission
	}

	existing, err := s.repo.GetArticlePermissionIncludeDeleted(ctx, articleID, userID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		created, err := s.repo.CreateArticlePermission(ctx, &models.ArticlePermission{
			ArticleID:  articleID,
			UserID:     userID,
			Permission: permission,
		})
		if err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return nil, ErrArticlePermissionAlreadyExists
			}
			return nil, err
		}
		return created, nil
	}

	existing.Permission = permission
	return s.repo.UpdateArticlePermission(ctx, existing)
}

func (s *Service) GetArticlePermission(ctx context.Context, articleID uint, userID uint) (*models.ArticlePermission, error) {
	permission, err := s.repo.GetArticlePermission(ctx, articleID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrArticlePermissionNotFound
		}
		return nil, err
	}
	return permission, nil
}

func (s *Service) GetArticlePermissions(ctx context.Context, articleID uint) ([]models.ArticlePermission, error) {
	if _, err := s.GetArticleByID(ctx, articleID); err != nil {
		return nil, err
	}

	permissions, err := s.repo.GetArticlePermissions(ctx, articleID)
	if err != nil {
		return nil, err
	}
	return permissions, nil
}

func (s *Service) DeleteArticlePermission(ctx context.Context, actorID uint, articleID uint, userID uint) error {
	actor, err := s.repo.GetUserByID(ctx, actorID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if actor.Role != "admin" {
		return ErrForbidden
	}

	if _, err := s.repo.GetArticleByID(ctx, articleID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrArticleNotFound
		}
		return err
	}

	if _, err := s.GetArticlePermission(ctx, articleID, userID); err != nil {
		return err
	}

	return s.repo.DeleteArticlePermission(ctx, articleID, userID)
}

func (s *Service) HasArticlePermission(ctx context.Context, articleID uint, userID uint, requiredPermission string) (bool, error) {
	if !validArticlePermission(requiredPermission) {
		return false, ErrInvalidArticlePermission
	}

	permission, err := s.repo.GetArticlePermission(ctx, articleID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	if requiredPermission == PermissionView {
		return permission.Permission == PermissionView || permission.Permission == PermissionEdit, nil
	}
	return permission.Permission == PermissionEdit, nil
}
