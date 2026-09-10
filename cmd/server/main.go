package main

import (
	"log"

	"wiki/database"
	"wiki/internal/config"
	"wiki/internal/handler"
	"wiki/internal/repository"
	"wiki/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	conf := config.Load()
	db, err := database.Connect(conf)
	if err != nil {
		log.Fatal(err)
	}
	repo := repository.NewRepository(db)

	service := service.NewService(repo)
	handler := handler.NewHandler(service)
	router := gin.Default()

	router.GET("/health", handler.Health)
	router.Run(conf.Port)
}
