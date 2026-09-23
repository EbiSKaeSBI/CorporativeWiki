package models

import (
	"gorm.io/gorm"
)


type User struct {
	gorm.Model
	Name         string   `gorm:"type:varchar(255);not null"`
	Email        string   `gorm:"type:varchar(255);unique;not null;index"`
	PasswordHash string   `gorm:"type:varchar(255);not null"`
	Role string `gorm:"type:varchar(20);not null;default:'viewer';check:role IN ('admin','editor','viewer')"`
	Articles []Article `gorm:"foreignKey:AuthorID"`
}
