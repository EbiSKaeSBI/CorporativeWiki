package dto

import "time"

type CreateArticlePermissionRequest struct {
	UserID     uint   `json:"user_id" binding:"required"`
	Permission string `json:"permission" binding:"required,oneof=view edit"`
}

type ArticlePermissionResponse struct {
	ID         uint      `json:"id"`
	ArticleID  uint      `json:"article_id"`
	UserID     uint      `json:"user_id"`
	Permission string    `json:"permission"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
