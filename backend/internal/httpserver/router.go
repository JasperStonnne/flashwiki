package httpserver

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"fpgwiki/backend/internal/config"
	"fpgwiki/backend/internal/httpserver/handlers"
	"fpgwiki/backend/internal/httpserver/middleware"
	"fpgwiki/backend/internal/repository"
	"fpgwiki/backend/internal/service"
	"fpgwiki/backend/internal/ws"
)

func NewRouter(cfg config.Config, log zerolog.Logger, pool *pgxpool.Pool) *gin.Engine {
	r := gin.New()
	hubManager := ws.NewHubManager(log)

	userRepo := repository.NewUserRepo(pool)
	refreshTokenRepo := repository.NewRefreshTokenRepo(pool)

	authService := service.NewAuthService(userRepo, refreshTokenRepo, cfg)
	userService := service.NewUserService(userRepo)

	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService, authService, hubManager)
	requireAuth := middleware.RequireAuth(cfg, userRepo)

	r.Use(
		middleware.Recover(log),
		middleware.RequestID(),
		middleware.Logger(log),
		middleware.CORS(cfg),
	)

	r.GET("/ping", handlers.Health(pool))

	api := r.Group("/api")
	api.GET("/ws", ws.Handler(cfg, hubManager, log))

	auth := api.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", requireAuth, authHandler.Logout)
	}

	me := api.Group("/me", requireAuth)
	{
		me.GET("", userHandler.GetMe)
		me.PATCH("", userHandler.UpdateMe)
		me.POST("/password", authHandler.ChangePassword)
	}

	admin := api.Group("/admin", requireAuth, middleware.RequireRole("manager"))
	{
		admin.POST("/users/:id/force-logout", userHandler.ForceLogout)
	}

	return r
}
