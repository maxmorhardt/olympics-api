package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/olympics-api/internal/handler"
	"github.com/maxmorhardt/olympics-api/internal/middleware"
)

func RegisterTournamentRoutes(rg *gin.RouterGroup, h handler.TournamentHandler, m handler.MatchHandler, verifier middleware.TokenVerifier) {
	auth := middleware.AuthMiddleware(verifier)

	// public read access
	rg.GET("", h.GetTournaments)
	rg.GET("/:id", h.GetTournament)
	rg.GET("/:id/standings", h.GetStandings)
	rg.GET("/:id/bracket", h.GetBracket)
	rg.GET("/:id/matches", m.GetMatches)

	// mutations require a logged-in creator (or olympics admin)
	rg.POST("", auth, h.CreateTournament)
	rg.DELETE("/:id", auth, h.DeleteTournament)
	rg.POST("/:id/participants", auth, h.AddParticipants)
	rg.POST("/:id/participants/one", auth, h.AddParticipant)
	rg.PATCH("/:id/participants/:participantId", auth, h.UpdateParticipant)
	rg.DELETE("/:id/participants/:participantId", auth, h.DeleteParticipant)
	rg.POST("/:id/teams/generate", auth, h.GenerateTeams)
	rg.PATCH("/:id/teams/:teamId", auth, h.UpdateTeam)
	rg.POST("/:id/teams/swap", auth, h.SwapPlayers)
	rg.POST("/:id/groups/generate", auth, h.GenerateGroups)
	rg.POST("/:id/playoffs/generate", auth, h.GeneratePlayoffs)
}

func RegisterMatchRoutes(rg *gin.RouterGroup, h handler.MatchHandler, verifier middleware.TokenVerifier) {
	rg.PATCH("/:matchId/result", middleware.AuthMiddleware(verifier), h.RecordResult)
	rg.POST("/:matchId/rollback", middleware.AuthMiddleware(verifier), h.RollbackResult)
}
