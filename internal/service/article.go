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
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrArticleAlreadyExists
		}
		return nil, err
	}

	if _, err := s.CreateArticleRevision(ctx, article, userID); err != nil {
		return nil, err
	}

	if _, err := s.CreateAuditLog(ctx, userID, AuditArticleCreated, uintPtr(article.ID), nil, "title="+title); err != nil {
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
	canEdit := user.Role == "editor" || user.Role == "admin" || user.ID == article.AuthorID
	if !canEdit {
		hasPerm, err := s.HasArticlePermission(ctx, articleID, userID, PermissionEdit)
		if err != nil {
			return nil, err
		}
		canEdit = hasPerm
	}
	if canEdit {
		article.Title = articleData.Title
		article.Content = articleData.Content
		article.Slug = articleData.Slug
		article, err = s.repo.UpdateArticle(ctx, article)
		if err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return nil, ErrArticleAlreadyExists
			}
			return nil, err
		}

		if _, err := s.CreateArticleRevision(ctx, article, userID); err != nil {
			return nil, err
		}

		if _, err := s.CreateAuditLog(ctx, userID, AuditArticleUpdated, uintPtr(article.ID), nil, ""); err != nil {
			return nil, err
		}

		return article, nil
	} else {
		return nil, ErrForbidden
	}
}

func (s *Service) SubmitArticle(ctx context.Context, articleID, userID uint) (*models.Article, error) {
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

	if user.ID != article.AuthorID {
		return nil, ErrForbidden
	}

	if article.Status != "draft" {
		return nil, ErrInvalidArticleStatus
	}

	article.Status = "pending"
	article, err = s.repo.UpdateArticle(ctx, article)
	if err != nil {
		return nil, err
	}
	return article, nil
}

func (s *Service) ApproveArticle(ctx context.Context, articleID uint) (*models.Article, error) {
	article, err := s.repo.GetArticleByID(ctx, articleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrArticleNotFound
		}
		return nil, err
	}

	if article.Status != "pending" {
		return nil, ErrInvalidArticleStatus
	}

	article.Status = "published"
	article, err = s.repo.UpdateArticle(ctx, article)
	if err != nil {
		return nil, err
	}
	return article, nil
}

func (s *Service) RejectArticle(ctx context.Context, articleID uint) (*models.Article, error) {
	article, err := s.repo.GetArticleByID(ctx, articleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrArticleNotFound
		}
		return nil, err
	}

	if article.Status != "pending" {
		return nil, ErrInvalidArticleStatus
	}

	article.Status = "rejected"
	article, err = s.repo.UpdateArticle(ctx, article)
	if err != nil {
		return nil, err
	}
	return article, nil
}

func (s *Service) DeleteArticle(ctx context.Context, userID, articleID uint) error {
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
			return ErrUserNotFound
		}
		return err
	}
	if user.Role == "admin" || article.AuthorID == user.ID {
		if err := s.repo.DeleteArticle(ctx, articleID); err != nil {
			return err
		}

		if _, err := s.CreateAuditLog(ctx, userID, AuditArticleDeleted, uintPtr(articleID), nil, ""); err != nil {
			return err
		}

		return nil
	}
	return ErrForbidden
}

func (s *Service) CreateArticleRevision(ctx context.Context, article *models.Article, editorID uint) (*models.ArticleRevision, error) {
	revision := &models.ArticleRevision{
		ArticleID: article.ID,
		EditorID:  editorID,
		Title:     article.Title,
		Slug:      article.Slug,
		Content:   article.Content,
	}

	created, err := s.repo.CreateArticleRevision(ctx, revision)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *Service) GetArticleRevisions(ctx context.Context, articleID uint) ([]models.ArticleRevision, error) {
	if _, err := s.GetArticleByID(ctx, articleID); err != nil {
		return nil, err
	}

	revisions, err := s.repo.GetArticleRevisions(ctx, articleID)
	if err != nil {
		return nil, err
	}
	return revisions, nil
}
