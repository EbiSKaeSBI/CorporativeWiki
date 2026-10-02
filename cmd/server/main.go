package main

import (
	"log"

	_ "wiki/docs"

	"wiki/database"
	"wiki/internal/config"
	_ "wiki/internal/dto"
	"wiki/internal/handler"
	"wiki/internal/middleware"
	"wiki/internal/models"
	"wiki/internal/repository"
	"wiki/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

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
	err = db.AutoMigrate(&models.User{}, &models.Article{}, &models.ArticleRevision{}, &models.ArticlePermission{}, &models.AuditLog{})
	if err != nil {
		log.Println(err)
	}

	repo := repository.NewRepository(db)
	service := service.NewService(repo, conf)
	handler := handler.NewHandler(service)
	router := gin.Default()
	router.Use(cors.Default())
	protected := router.Group("/api")
	protected.Use(middleware.AuthMiddleware(conf.JwtSecret))
	admin := protected.Group("/admin")
	articles := protected.Group("/articles")

	admin.Use(middleware.RoleMiddleware("admin"))

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
	admin.GET("/users/:id", handler.GetUserByID)
	admin.GET("/audit-logs", handler.GetAuditLogs)

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
	protected.GET("/profile", handler.Profile)

	// @Summary List articles
	// @Tags Articles
	// @Success 200 {array} dto.ArticleResponse
	// @Security BearerAuth
	// @Router /api/articles [get]
	articles.GET("", handler.GetArticles)

	// @Summary Get article by ID
	// @Tags Articles
	// @Param id path int true "Article ID"
	// @Success 200 {object} dto.ArticleResponse
	// @Security BearerAuth
	// @Router /api/articles/{id} [get]
	articles.GET("/:id", handler.GetArticle)

	// @Summary Create article
	// @Tags Articles
	// @Accept json
	// @Param input body dto.CreateArticleRequest true "Article input"
	// @Success 201 {object} dto.ArticleResponse
	// @Security BearerAuth
	// @Router /api/articles [post]
	articles.POST("", handler.CreateArticle)

	// @Summary Update article (свою; editor/admin — чужие)
	// @Tags Articles
	// @Accept json
	// @Param id path int true "Article ID"
	// @Param input body dto.UpdateArticleRequest true "Article input"
	// @Success 200 {object} dto.ArticleResponse
	// @Security BearerAuth
	// @Router /api/articles/{id} [put]
	articles.PUT("/:id", handler.UpdateArticle)

	// @Summary Delete article (soft, только автор или admin)
	// @Tags Articles
	// @Param id path int true "Article ID"
	// @Success 204 "No Content"
	// @Security BearerAuth
	// @Router /api/articles/{id} [delete]
	articles.DELETE("/:id", handler.DeleteArticle)

	// @Summary Submit article for review (автор, draft -> pending)
	// @Tags Articles
	// @Param id path int true "Article ID"
	// @Success 200 {object} dto.ArticleResponse
	// @Security BearerAuth
	// @Router /api/articles/{id}/submit [post]
	articles.POST("/:id/submit", handler.SubmitArticle)

	// @Summary Approve article (pending -> published, editor/admin)
	// @Tags Articles
	// @Param id path int true "Article ID"
	// @Success 200 {object} dto.ArticleResponse
	// @Security BearerAuth
	// @Router /api/articles/{id}/approve [post]
	articles.POST("/:id/approve",
		middleware.RoleMiddleware("admin", "editor"),
		handler.ApproveArticle,
	)

	// @Summary Reject article (pending -> rejected, editor/admin)
	// @Tags Articles
	// @Param id path int true "Article ID"
	// @Success 200 {object} dto.ArticleResponse
	// @Security BearerAuth
	// @Router /api/articles/{id}/reject [post]
	articles.POST("/:id/reject",
		middleware.RoleMiddleware("admin", "editor"),
		handler.RejectArticle,
	)

	// @Summary Get article revision history (newest first)
	// @Tags Articles
	// @Param id path int true "Article ID"
	// @Success 200 {array} dto.ArticleRevisionResponse
	// @Security BearerAuth
	// @Router /api/articles/{id}/revisions [get]
	articles.GET("/:id/revisions", handler.GetArticleRevisions)

	// @Summary List article permissions
	// @Tags Articles
	// @Param id path int true "Article ID"
	// @Success 200 {array} dto.ArticlePermissionResponse
	// @Failure 400 {object} dto.ErrorResponse
	// @Failure 401 {object} dto.ErrorResponse
	// @Failure 404 {object} dto.ErrorResponse
	// @Failure 500 {object} dto.ErrorResponse
	// @Security BearerAuth
	// @Router /api/articles/{id}/permissions [get]
	articles.GET("/:id/permissions", handler.GetArticlePermissions)

	// @Summary Grant or update article permission (admin only)
	// @Tags Articles
	// @Param id path int true "Article ID"
	// @Param body body dto.CreateArticlePermissionRequest true "Permission payload"
	// @Success 201 {object} dto.ArticlePermissionResponse
	// @Failure 400 {object} dto.ErrorResponse
	// @Failure 401 {object} dto.ErrorResponse
	// @Failure 403 {object} dto.ErrorResponse
	// @Failure 404 {object} dto.ErrorResponse
	// @Failure 409 {object} dto.ErrorResponse
	// @Failure 500 {object} dto.ErrorResponse
	// @Security BearerAuth
	// @Router /api/articles/{id}/permissions [post]
	articles.POST("/:id/permissions", middleware.RoleMiddleware("admin"), handler.CreateArticlePermission)

	// @Summary Revoke article permission (admin only)
	// @Tags Articles
	// @Param id path int true "Article ID"
	// @Param userID path int true "User ID"
	// @Success 204 "No Content"
	// @Failure 400 {object} dto.ErrorResponse
	// @Failure 401 {object} dto.ErrorResponse
	// @Failure 403 {object} dto.ErrorResponse
	// @Failure 404 {object} dto.ErrorResponse
	// @Failure 500 {object} dto.ErrorResponse
	// @Security BearerAuth
	// @Router /api/articles/{id}/permissions/{userID} [delete]
	articles.DELETE("/:id/permissions/:userID", middleware.RoleMiddleware("admin"), handler.DeleteArticlePermission)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	log.Println("Swagger доступен по адресу: http://localhost:8080/swagger/index.html")
	router.Run(conf.Port)
}
