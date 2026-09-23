package models

import "gorm.io/gorm"

type Article struct {
	gorm.Model
	Title    string `gorm:"type:varchar(255);not null"`
	Slug     string `gorm:"type:varchar(255);not null;unique"`
	Content  string `gorm:"type:text;not null"`
	AuthorID uint   `gorm:"index;not null"`
	Status   string `gorm:"type:varchar(20);not null;default:'draft';check:status IN ('draft','pending','published','rejected')"`
}
