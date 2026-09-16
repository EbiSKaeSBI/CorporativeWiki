package main

import (
	"log"

	_ "wiki/docs"

	"wiki/database"
	"wiki/internal/config"
	_ "wiki/internal/dto"
	"wiki/internal/handler"
	"wiki/internal/models"
	"wiki/internal/repository"
	"wiki/internal/service"

	"github.com/gin-contrib/cors"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/joho/godotenv"

	"github.com/gin-gonic/gin"
)

// @title Corporate Wiki API
// @version 1.0
// @description API для корпоративной Wiki (Auth, Users)
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}
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
	service := service.NewService(repo, conf)
	handler := handler.NewHandler(service)
	router := gin.Default()
	router.Use(cors.Default())

	// @Summary Health check
	// @Tags Health
	// @Success 200 {object} map[string]string
	// @Router /health [get]
	router.GET("/health", handler.Health)

	// @Summary Get user by ID
	// @Tags Users
	// @Param id path string true "User ID"
	// @Success 200 {object} models.User
	// @Router /users/{id} [get]
	router.GET("/users/:id", handler.GetUserByID)

	// @Summary Register new user
	// @Tags Auth
	// @Accept json
	// @Param input body dto.RegisterRequest true "Register input"
	// @Success 201 {object} models.User
	// @Router /auth/register [post]
	router.POST("/auth/register", handler.Register)

	// @Summary Login
	// @Tags Auth
	// @Accept json
	// @Param input body dto.LoginRequest true "Login input"
	// @Success 200 {object} map[string]string
	// @Router /auth/login [post]
	router.POST("/auth/login", handler.Login)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	log.Println("Swagger доступен по адресу: http://localhost:8080/swagger/index.html")
	router.Run(conf.Port)
}
