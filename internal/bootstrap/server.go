package bootstrap

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/olympics-api/internal/config"
	"github.com/maxmorhardt/olympics-api/internal/handler"
	"github.com/maxmorhardt/olympics-api/internal/middleware"
	"github.com/maxmorhardt/olympics-api/internal/repository"
	"github.com/maxmorhardt/olympics-api/internal/routes"
	"github.com/maxmorhardt/olympics-api/internal/service"
	"gorm.io/gorm"
)

type Dependencies struct {
	Config   *config.Config
	DB       *gorm.DB
	Verifier middleware.TokenVerifier
}

func BuildDependencies() (*Dependencies, error) {
	cfg, err := config.LoadEnv()
	if err != nil {
		return nil, err
	}

	db, err := config.InitDB(cfg)
	if err != nil {
		return nil, err
	}

	oidcVerifier, err := config.InitOIDC(cfg)
	if err != nil {
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		return nil, err
	}

	return &Dependencies{
		Config:   cfg,
		DB:       db,
		Verifier: middleware.NewOIDCTokenVerifier(oidcVerifier),
	}, nil
}

func NewServer(deps *Dependencies) *gin.Engine {
	r := gin.New()

	setupMiddleware(r, deps.Config)
	setupRoutes(r, deps)

	return r
}

func setupMiddleware(r *gin.Engine, cfg *config.Config) {
	r.Use(gin.Recovery())
	r.Use(middleware.CORSMiddleware(cfg.Server.AllowedOrigins))
	r.Use(middleware.LoggerMiddleware)
}

func setupRoutes(r *gin.Engine, deps *Dependencies) {
	db := deps.DB

	tournamentRepo := repository.NewTournamentRepository(db)
	matchRepo := repository.NewMatchRepository(db)

	tournamentService := service.NewTournamentService(tournamentRepo, matchRepo)
	matchService := service.NewMatchService(matchRepo, tournamentRepo)

	tournamentHandler := handler.NewTournamentHandler(tournamentService)
	matchHandler := handler.NewMatchHandler(matchService)
	healthHandler := handler.NewHealthHandler(db)

	routes.RegisterRootRoutes(r.Group(""), healthHandler)
	routes.RegisterTournamentRoutes(r.Group("/tournaments"), tournamentHandler, matchHandler, deps.Verifier)
	routes.RegisterMatchRoutes(r.Group("/matches"), matchHandler, deps.Verifier)
}
