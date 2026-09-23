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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrArticleNotFound
		}
		return nil, err
	}
	return article, nil
}

func (s *Service) GetArticles(ctx context.Context) ([]models.Article, error) {
	articles, err := s.repo.GetArticles(ctx)
	if err != nil {
		return nil, err
	}
	return articles, nil
}

func (s *Service) UpdateArticle(ctx context.Context, articleID, userID uint, articleData models.Article) (*models.Article, error) {
	article, err := s.repo.GetArticleByID(ctx, articleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrArticleNotFound
		}
		return nil, err
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if user.Role == "editor" || user.Role == "admin" || user.ID == article.AuthorID {
		article.Title = articleData.Title
		article.Content = articleData.Content
		article.Slug = articleData.Slug
		article, err = s.repo.UpdateArticle(ctx, article)
		if err != nil {
			return nil, err
		}
		return article, nil
	} else {
		return nil, ErrForbidden
	}
}

func (s *Service) DeleteArticle(ctx context.Context,userID, articleID uint) error {
article, err := s.repo.GetArticleByID(ctx, articleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrArticleNotFound
		}
		return err
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return  ErrUserNotFound
		}
		return err
	}
	if user.Role == "admin" || article.AuthorID == user.ID {
		err := s.repo.DeleteArticle(ctx, articleID)
		return err

	}
		return ErrForbidden
}
