package repository

import (
	"context"

	"wiki/internal/models"
)

// Article-level thin passthrough: никаких бизнес-проверок (валидация
// permission, существование статьи/user'a, маппинг ErrRecordNotFound) здесь
// нет — это задача Service.

func (r *Repository) CreateArticlePermission(ctx context.Context, permission *models.ArticlePermission) (*models.ArticlePermission, error) {
	if err := r.db.WithContext(ctx).Create(permission).Error; err != nil {
		return nil, err
	}
	return permission, nil
}

// GetArticlePermission — конкретная связка (article, user). Если записи нет,
// возвращает gorm.ErrRecordNotFound как есть: Service сам решит, что это
// ErrArticlePermissionNotFound.
func (r *Repository) GetArticlePermission(ctx context.Context, articleID, userID uint) (*models.ArticlePermission, error) {
	var permission models.ArticlePermission
	if err := r.db.WithContext(ctx).
		Where("article_id = ? AND user_id = ?", articleID, userID).
		First(&permission).Error; err != nil {
		return nil, err
	}
	return &permission, nil
}

// GetArticlePermissions — все права на статью, старые первыми.
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

// DeleteArticlePermission — удаляет связку (article, user). С gorm.Model это
// soft delete: строка остаётся и продолжает занимать уникальный индекс
// uix_article_user — см. заметку Service-слоя про повторную выдачу права.
func (r *Repository) DeleteArticlePermission(ctx context.Context, articleID, userID uint) error {
	var permission models.ArticlePermission
	if err := r.db.WithContext(ctx).
		Where("article_id = ? AND user_id = ?", articleID, userID).
		Delete(&permission).Error; err != nil {
		return err
	}
	return nil
}
