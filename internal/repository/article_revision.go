package repository

import (
	"context"

	"wiki/internal/models"
)

func (r *Repository) CreateArticleRevision(ctx context.Context, revision *models.ArticleRevision) (*models.ArticleRevision, error) {
	if err := r.db.WithContext(ctx).Create(revision).Error; err != nil {
		return nil, err
	}
	return revision, nil
}

// GetArticleRevisions — история статьи, свежие версии первыми.
func (r *Repository) GetArticleRevisions(ctx context.Context, articleID uint) ([]models.ArticleRevision, error) {
	var revisions []models.ArticleRevision
	if err := r.db.WithContext(ctx).
		Where("article_id = ?", articleID).
		Order("created_at DESC").
		Find(&revisions).Error; err != nil {
		return nil, err
	}
	return revisions, nil
}
