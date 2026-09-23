package service

import "errors"

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrArticleNotFound = errors.New("article not found")
	ErrForbidden = errors.New("forbidden")
)
