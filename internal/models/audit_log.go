package models

import "gorm.io/gorm"

type AuditLog struct {
	gorm.Model

	ActorID      uint   `gorm:"index;not null"`
	Action       string `gorm:"type:varchar(50);not null;index"`
	ArticleID    *uint  `gorm:"index"`
	TargetUserID *uint  `gorm:"index"`
	Details      string `gorm:"type:text"`

	Actor      User     `gorm:"foreignKey:ActorID"`
	Article    *Article `gorm:"foreignKey:ArticleID"`
	TargetUser *User    `gorm:"foreignKey:TargetUserID"`
}
