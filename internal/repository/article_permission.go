package repository

import (
	"context"
	"time"

	"wiki/internal/models"

	"gorm.io/gorm"
)

func (r *Repository) CreateArticlePermission(ctx context.Context, permission *models.ArticlePermission) (*models.ArticlePermission, error) {
	if err := r.db.WithContext(ctx).Create(permission).Error; err != nil {
		return nil, err
	}
	return permission, nil
}

func (r *Repository) GetArticlePermission(ctx context.Context, articleID, userID uint) (*models.ArticlePermission, error) {
	var permission models.ArticlePermission
	if err := r.db.WithContext(ctx).
		Where("article_id = ? AND user_id = ?", articleID, userID).
		First(&permission).Error; err != nil {
		return nil, err
	}
	return &permission, nil
}

func (r *Repository) GetArticlePermissions(ctx context.Context, articleID uint) ([]models.ArticlePermission, error) {
	var permissions []models.ArticlePermission
	if err := r.db.WithContext(ctx).
		Where("article_id = ?", articleID).
		Order("created_at ASC").
		Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

func (r *Repository) DeleteArticlePermission(ctx context.Context, articleID, userID uint) error {
	var permission models.ArticlePermission
	if err := r.db.WithContext(ctx).
		Where("article_id = ? AND user_id = ?", articleID, userID).
		Delete(&permission).Error; err != nil {
		return err
	}
	return nil
}

func (r *Repository) GetArticlePermissionIncludeDeleted(ctx context.Context, articleID, userID uint) (*models.ArticlePermission, error) {
	var permission models.ArticlePermission
	if err := r.db.WithContext(ctx).Unscoped().
		Where("article_id = ? AND user_id = ?", articleID, userID).
		First(&permission).Error; err != nil {
		return nil, err
	}
	return &permission, nil
}

func (r *Repository) UpdateArticlePermission(ctx context.Context, permission *models.ArticlePermission) (*models.ArticlePermission, error) {
	if err := r.db.WithContext(ctx).Unscoped().
		Model(&models.ArticlePermission{}).
		Where("id = ?", permission.ID).
		Updates(map[string]any{
			"permission": permission.Permission,
			"deleted_at": nil,
			"updated_at": time.Now(),
		}).Error; err != nil {
		return nil, err
	}
	permission.DeletedAt = gorm.DeletedAt{}
	return permission, nil
}
