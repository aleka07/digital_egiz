package api

import (
	"net/http"

	"github.com/digital-egiz/backend/internal/api/controllers"
	"github.com/digital-egiz/backend/internal/api/middleware"
	"github.com/digital-egiz/backend/internal/config"
	"github.com/digital-egiz/backend/internal/db"
	"github.com/digital-egiz/backend/internal/services"
	"github.com/digital-egiz/backend/internal/utils"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Router manages the API routes and controllers
type Router struct {
	engine             *gin.Engine
	logger             *utils.Logger
	config             *config.Config
	authMiddleware     *middleware.AuthMiddleware
	serviceProvider    *services.ServiceProvider
	db                 *db.Database
	apiV1              *gin.RouterGroup
	userController     *controllers.UserController
	projectController  *controllers.ProjectController
	twinTypeController *controllers.TwinTypeController
	twinController     *controllers.TwinController
	historyController  *controllers.HistoryController
}

// NewRouter creates a new Router instance
func NewRouter(
	config *config.Config,
	logger *utils.Logger,
	db *db.Database,
	serviceProvider *services.ServiceProvider,
) *Router {
	// Set Gin mode based on environment
	if config.Server.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	// Use the logger and recovery middleware
	engine.Use(gin.Recovery())
	engine.Use(middleware.LoggingMiddleware(logger))

	// Configure CORS
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowCredentials = true
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Authorization", "Content-Type", "Origin"}
	engine.Use(cors.New(corsConfig))

	// Create JWT auth middleware
	authMiddleware := middleware.NewAuthMiddleware(&config.JWT)

	return &Router{
		engine:          engine,
		logger:          logger.Named("router"),
		config:          config,
		authMiddleware:  authMiddleware,
		serviceProvider: serviceProvider,
		db:              db,
	}
}

// SetupRoutes configures all API routes
func (r *Router) SetupRoutes() {
	// Health check endpoint (no auth required)
	r.engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// API version group - all main API routes are under /api/v1
	r.apiV1 = r.engine.Group("/api/v1")

	// Setup services
	userService := services.NewUserService(r.db, r.logger)
	projectService := services.NewProjectService(r.db, r.logger)
	twinTypeService := services.NewTwinTypeService(r.db, r.logger)
	twinService := services.NewTwinService(r.db, r.logger)
	historyService := r.serviceProvider.GetHistoryService()

	// Setup controllers
	authController := controllers.NewAuthController(userService, &r.config.JWT, r.logger)
	r.userController = controllers.NewUserController(userService, r.logger)
	r.projectController = controllers.NewProjectController(projectService, r.logger)
	r.twinTypeController = controllers.NewTwinTypeController(twinTypeService, r.logger)
	r.twinController = controllers.NewTwinController(twinService, r.logger)
	r.historyController = controllers.NewHistoryController(historyService, r.logger)

	// Register auth routes (no auth required)
	authController.RegisterRoutes(r.engine.Group("/api"))

	// Routes that require authentication
	authorizedRoutes := r.apiV1.Group("")
	authorizedRoutes.Use(r.authMiddleware.RequireAuth())

	// Register routes that require authentication
	r.userController.RegisterRoutes(authorizedRoutes)
	r.projectController.RegisterRoutes(authorizedRoutes)
	r.twinTypeController.RegisterRoutes(authorizedRoutes)

	// Group for twin endpoints
	twinsRoutes := authorizedRoutes.Group("/twins")
	r.twinController.RegisterRoutes(twinsRoutes)

	// Register history routes under each twin
	twinHistoryRoutes := twinsRoutes.Group("/:id/history")
	r.historyController.RegisterRoutes(twinHistoryRoutes)

	// Admin-only routes
	adminRoutes := authorizedRoutes.Group("/admin")
	adminRoutes.Use(r.authMiddleware.RequireAdmin())

	// Add Swagger documentation if not in production
	if !r.config.Server.IsProduction() {
		r.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	r.logger.Info("API routes setup completed")
}

// GetEngine returns the Gin engine
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}
