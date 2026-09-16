package service

import (
	"wiki/internal/config"
	"wiki/internal/repository"
)

type Service struct {
	repo *repository.Repository
	conf *config.Config
}

func NewService(repo *repository.Repository, config *config.Config) *Service {
	return &Service{
		repo: repo,
		conf: config,
	}
}
