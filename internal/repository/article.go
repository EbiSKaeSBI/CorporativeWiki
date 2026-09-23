package repository

import (
	"context"
	"wiki/internal/models"
)


func (r *Repository) CreateArticle(ctx context.Context, article *models.Article) (*models.Article, error) {
	if err := r.db.WithContext(ctx).Create(article).Error; err != nil {
		return nil, err	
	}
	return article, nil
}

func (r *Repository) GetArticleByID(ctx context.Context, id uint) (*models.Article, error) {
	var article models.Article
	if err := r.db.WithContext(ctx).First(&article, id).Error; err != nil {
		return nil, err
	}
	return &article, nil
}

func (r *Repository) GetArticles(ctx context.Context) ([]models.Article, error) {
	var article []models.Article
	if err := r.db.WithContext(ctx).Find(&article).Error; err != nil {
		return  nil, err
	}
	return article, nil
}

func (r *Repository) UpdateArticle(ctx context.Context,article *models.Article) (*models.Article, error) {
	if err := r.db.WithContext(ctx).Where("id = ?", article.ID).Updates(article).Error; err != nil {
		return nil, err
	}

	return article, nil
}

func (r *Repository) DeleteArticle(ctx context.Context, id uint) error {
	var article models.Article
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&article).Error; err != nil {
		return  err
	}
	return nil
}
