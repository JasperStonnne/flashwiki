package httpserver

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"fpgwiki/backend/internal/config"
	"fpgwiki/backend/internal/httpserver/handlers"
	"fpgwiki/backend/internal/httpserver/middleware"
	"fpgwiki/backend/internal/ws"
)

func NewRouter(cfg config.Config, log zerolog.Logger, pool *pgxpool.Pool) *gin.Engine {
	r := gin.New()
	hubManager := ws.NewHubManager(log)

	r.Use(
		middleware.Recover(log),
		middleware.RequestID(),
		middleware.Logger(log),
		middleware.CORS(cfg),
	)

	r.GET("/ping", handlers.Health(pool))

	api := r.Group("/api")
	api.GET("/ws", ws.Handler(cfg, hubManager, log))

	return r
}
