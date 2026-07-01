package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/maxmorhardt/olympics-api/internal/errs"
	"github.com/maxmorhardt/olympics-api/internal/model"
	"github.com/maxmorhardt/olympics-api/internal/service"
)

type TournamentHandler interface {
	GetTournaments(c *gin.Context)
	GetTournament(c *gin.Context)
	CreateTournament(c *gin.Context)
	AddParticipants(c *gin.Context)
	GenerateTeams(c *gin.Context)
	GenerateGroups(c *gin.Context)
	GeneratePlayoffs(c *gin.Context)
	DeleteTournament(c *gin.Context)
	UpdateTeam(c *gin.Context)
	SwapPlayers(c *gin.Context)
	GetStandings(c *gin.Context)
	GetBracket(c *gin.Context)
}

type tournamentHandler struct {
	tournamentService service.TournamentService
}

func NewTournamentHandler(tournamentService service.TournamentService) TournamentHandler {
	return &tournamentHandler{
		tournamentService: tournamentService,
	}
}

func (h *tournamentHandler) GetTournaments(c *gin.Context) {
	tournaments, err := h.tournamentService.GetTournaments(c.Request.Context())
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, tournaments)
}

func (h *tournamentHandler) GetTournament(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	tournament, err := h.tournamentService.GetTournament(c.Request.Context(), id)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, tournament)
}

func (h *tournamentHandler) CreateTournament(c *gin.Context) {
	var req model.CreateTournamentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, errs.ErrInvalidRequestBody)
		return
	}

	user, isAdmin := actor(c)
	tournament, err := h.tournamentService.CreateTournament(c.Request.Context(), &req, user, isAdmin)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, tournament)
}

func (h *tournamentHandler) AddParticipants(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	var req model.AddParticipantsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, errs.ErrInvalidRequestBody)
		return
	}

	user, isAdmin := actor(c)
	tournament, err := h.tournamentService.AddParticipants(c.Request.Context(), id, req.Names, user, isAdmin)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, tournament)
}

func (h *tournamentHandler) GenerateTeams(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	user, isAdmin := actor(c)
	teams, err := h.tournamentService.GenerateTeams(c.Request.Context(), id, user, isAdmin)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, teams)
}

func (h *tournamentHandler) GenerateGroups(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	user, isAdmin := actor(c)
	groups, err := h.tournamentService.GenerateGroups(c.Request.Context(), id, user, isAdmin)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, groups)
}

func (h *tournamentHandler) GeneratePlayoffs(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	user, isAdmin := actor(c)
	bracket, err := h.tournamentService.GeneratePlayoffs(c.Request.Context(), id, user, isAdmin)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, bracket)
}

func (h *tournamentHandler) GetStandings(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	standings, err := h.tournamentService.GetStandings(c.Request.Context(), id)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, standings)
}

func (h *tournamentHandler) GetBracket(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	bracket, err := h.tournamentService.GetBracket(c.Request.Context(), id)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, bracket)
}

func (h *tournamentHandler) DeleteTournament(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	user, isAdmin := actor(c)
	if err := h.tournamentService.DeleteTournament(c.Request.Context(), id, user, isAdmin); err != nil {
		respondServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *tournamentHandler) UpdateTeam(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	teamID, ok := parseID(c, "teamId")
	if !ok {
		return
	}

	var req model.UpdateNameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, errs.ErrInvalidRequestBody)
		return
	}

	user, isAdmin := actor(c)
	tournament, err := h.tournamentService.UpdateTeam(c.Request.Context(), id, teamID, req.Name, user, isAdmin)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, tournament)
}

func (h *tournamentHandler) SwapPlayers(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}

	var req model.SwapPlayersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, errs.ErrInvalidRequestBody)
		return
	}

	aID, err := uuid.Parse(req.ParticipantAId)
	if err != nil {
		writeError(c, http.StatusBadRequest, errs.ErrInvalidRequestBody)
		return
	}
	bID, err := uuid.Parse(req.ParticipantBId)
	if err != nil {
		writeError(c, http.StatusBadRequest, errs.ErrInvalidRequestBody)
		return
	}

	user, isAdmin := actor(c)
	tournament, err := h.tournamentService.SwapPlayers(c.Request.Context(), id, aID, bID, user, isAdmin)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, tournament)
}

func parseID(c *gin.Context, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid ID format", c))
		return uuid.Nil, false
	}
	return id, true
}
