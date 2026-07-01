package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/olympics-api/internal/errs"
	"github.com/maxmorhardt/olympics-api/internal/model"
	"github.com/maxmorhardt/olympics-api/internal/service"
)

type MatchHandler interface {
	GetMatches(c *gin.Context)
	RecordResult(c *gin.Context)
	RollbackResult(c *gin.Context)
}

type matchHandler struct {
	matchService service.MatchService
}

func NewMatchHandler(matchService service.MatchService) MatchHandler {
	return &matchHandler{
		matchService: matchService,
	}
}

func (h *matchHandler) GetMatches(c *gin.Context) {
	tournamentID, ok := parseID(c, "id")
	if !ok {
		return
	}

	matches, err := h.matchService.GetMatches(c.Request.Context(), tournamentID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, matches)
}

func (h *matchHandler) RecordResult(c *gin.Context) {
	matchID, ok := parseID(c, "matchId")
	if !ok {
		return
	}

	var req model.RecordMatchResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, errs.ErrInvalidRequestBody)
		return
	}

	user, isAdmin := actor(c)
	match, err := h.matchService.RecordResult(c.Request.Context(), matchID, &req, user, isAdmin)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, match)
}

func (h *matchHandler) RollbackResult(c *gin.Context) {
	matchID, ok := parseID(c, "matchId")
	if !ok {
		return
	}

	user, isAdmin := actor(c)
	match, err := h.matchService.RollbackResult(c.Request.Context(), matchID, user, isAdmin)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, match)
}
