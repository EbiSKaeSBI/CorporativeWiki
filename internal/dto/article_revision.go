package dto

import "time"

type ArticleRevisionResponse struct {
	ID        uint      `json:"id"`
	ArticleID uint      `json:"article_id"`
	EditorID  uint      `json:"editor_id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
