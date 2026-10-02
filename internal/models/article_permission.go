package models

import "gorm.io/gorm"

// ArticlePermission — точечное право конкретного пользователя на конкретную статью.
// Роль в системе (viewer/editor/admin) остаётся глобальной; это дополнительный
// layer: «Владимир → edit на статью "Инструкция по Docker"».
//
// Permission — открытое множество из двух значений; ограничения держат
// БД (check) и Service (валидация на записи), а не enum-тип Go:
// добавлять третье право ("comment") позже — только новая строка в check.
type ArticlePermission struct {
	gorm.Model

	// composite unique: одна запись (article_id, user_id) — перекладываем
	// permission через UPDATE, а не плодим вторую строку.
	ArticleID uint `gorm:"not null;uniqueIndex:uix_article_user"`
	UserID    uint `gorm:"not null;uniqueIndex:uix_article_user"`

	Permission string `gorm:"type:varchar(10);not null;default:'view';check:permission IN ('view','edit')"`

	Article Article
	User    User
}
