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
	nodeRepo := repository.NewNodeRepo(pool)
	nodePermissionRepo := repository.NewNodePermissionRepo(pool)
	groupRepo := repository.NewGroupRepo(pool)

	authService := service.NewAuthService(userRepo, refreshTokenRepo, cfg)
	userService := service.NewUserService(userRepo)
	permissionService := service.NewPermissionService(userRepo, nodeRepo, nodePermissionRepo, groupRepo, pool)
	nodeService := service.NewNodeService(nodeRepo, userRepo, permissionService, pool)
	groupService := service.NewGroupService(groupRepo, userRepo, nodePermissionRepo, pool)

	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService, authService, hubManager)
	nodeHandler := handlers.NewNodeHandler(nodeService, permissionService)
	permissionHandler := handlers.NewPermissionHandler(permissionService, nodeService)
	groupHandler := handlers.NewGroupHandler(groupService)
	adminHandler := handlers.NewAdminHandler(userService, authService)
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

	nodes := api.Group("/nodes", requireAuth)
	{
		nodes.POST("", nodeHandler.CreateNode)
		nodes.GET("", nodeHandler.ListChildren)
		nodes.GET("/:id", nodeHandler.GetNode)
		nodes.PATCH("/:id", nodeHandler.UpdateNode)
		nodes.DELETE("/:id", nodeHandler.DeleteNode)
		nodes.POST("/:id/restore", nodeHandler.RestoreNode)
		nodes.GET("/:id/permissions", permissionHandler.GetNodePermissions)
		nodes.PUT("/:id/permissions", permissionHandler.SetNodePermissions)
	}

	admin := api.Group("/admin", requireAuth, middleware.RequireRole("manager"))
	{
		admin.POST("/users/:id/force-logout", userHandler.ForceLogout)
		admin.GET("/users", adminHandler.ListUsers)
		admin.PATCH("/users/:id/role", adminHandler.ChangeRole)
		admin.GET("/groups", groupHandler.ListGroups)
		admin.POST("/groups", groupHandler.CreateGroup)
		admin.PATCH("/groups/:id", groupHandler.UpdateGroup)
		admin.DELETE("/groups/:id", groupHandler.DeleteGroup)
		admin.GET("/groups/:id/members", groupHandler.ListMembers)
		admin.POST("/groups/:id/members", groupHandler.AddMember)
		admin.DELETE("/groups/:id/members/:uid", groupHandler.RemoveMember)
	}

	return r
}
