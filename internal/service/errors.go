package service

import "errors"

var (
	ErrInvalidAuditAction             = errors.New("неизвестное действие аудита")
	ErrUserNotFound                   = errors.New("user not found")
	ErrUserAlreadyExists              = errors.New("user already exists")
	ErrInvalidCredentials             = errors.New("invalid email or password")
	ErrArticleNotFound                = errors.New("article not found")
	ErrArticleAlreadyExists           = errors.New("article already exists")
	ErrInvalidArticleStatus           = errors.New("invalid article status")
	ErrForbidden                      = errors.New("forbidden")
	ErrArticlePermissionNotFound      = errors.New("article permission not found")
	ErrArticlePermissionAlreadyExists = errors.New("article permission already exists")
	ErrInvalidArticlePermission       = errors.New("invalid article permission")
)
