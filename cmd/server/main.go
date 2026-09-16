package main

import (
	"log"

	_ "wiki/docs"

	"wiki/database"
	"wiki/internal/config"
	"wiki/internal/handler"
	"wiki/internal/models"
	"wiki/internal/repository"
	"wiki/internal/service"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/gin-gonic/gin"
)

// @title Corporate Wiki API
// @version 1.0
// @description API для корпоративной Wiki (Auth, Users)
// @host localhost:8080
// @BasePath /v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	conf := config.Load()
	db, err := database.Connect(conf)
	if err != nil {
		log.Fatal(err)
	}
	err = db.AutoMigrate(&models.User{})
	if err != nil {
		log.Println(err)
	}

	repo := repository.NewRepository(db)
	service := service.NewService(repo)
	handler := handler.NewHandler(service)
	router := gin.Default()

	router.GET("/health", handler.Health)
	router.GET("/users/:id", handler.GetUserByID)
	router.POST("/auth/register", handler.Register)
	router.POST("/auth/login", handler.Login)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	log.Println("Swagger доступен по адресу: http://localhost:8080/swagger/index.html")
	router.Run(conf.Port)
}
