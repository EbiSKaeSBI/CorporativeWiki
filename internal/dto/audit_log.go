package dto

import "time"

type AuditLogResponse struct {
	ID           uint      `json:"id"`
	ActorID      uint      `json:"actor_id"`
	Action       string    `json:"action"`
	ArticleID    *uint     `json:"article_id"`
	TargetUserID *uint     `json:"target_user_id"`
	Details      string    `json:"details"`
	CreatedAt    time.Time `json:"created_at"`
}

type AuditLogListResponse struct {
	Items []AuditLogResponse `json:"items"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
	Total int64              `json:"total"`
}
