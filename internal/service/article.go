package service

import (
	"context"
	"errors"

	"wiki/internal/models"

	"gorm.io/gorm"
)

func (s *Service) CreateArticle(ctx context.Context, userID uint, title, slug, content string) (*models.Article, error) {
	article, err := s.repo.CreateArticle(ctx, &models.Article{
		Title:    title,
		Slug:     slug,
		Content:  content,
		Status:   "draft",
		AuthorID: userID,
	})
	if err != nil {
		return nil, err
	}
	return article, nil
}

func (s *Service) GetArticleByID(ctx context.Context, id uint) (*models.Article, error) {
 	article, err := s.repo.GetArticleByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound){
			return nil, ErrArticleNotFound
		}
		return nil, err
	}
	return article, nil
}
