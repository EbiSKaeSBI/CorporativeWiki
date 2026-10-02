package models

import "gorm.io/gorm"

type ArticleRevision struct {
	gorm.Model

	ArticleID uint `gorm:"index;not null"`
	EditorID  uint `gorm:"index;not null"`

	Title   string `gorm:"type:varchar(255);not null"`
	Slug    string `gorm:"type:varchar(255);not null"`
	Content string `gorm:"type:text;not null"`

	Article Article
	Editor  User
}
