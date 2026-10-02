package models

import "gorm.io/gorm"

type ArticlePermission struct {
	gorm.Model

	ArticleID uint `gorm:"not null;uniqueIndex:uix_article_user"`
	UserID    uint `gorm:"not null;uniqueIndex:uix_article_user"`

	Permission string `gorm:"type:varchar(10);not null;default:'view';check:permission IN ('view','edit')"`

	Article Article
	User    User
}
