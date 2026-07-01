package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/olympics-api/internal/errs"
	"github.com/maxmorhardt/olympics-api/internal/model"
	"github.com/maxmorhardt/olympics-api/internal/util"
)

// respondServiceError maps a service-layer sentinel error to an HTTP status.
func respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errs.ErrTournamentNotFound),
		errors.Is(err, errs.ErrMatchNotFound),
		errors.Is(err, errs.ErrParticipantNotFound),
		errors.Is(err, errs.ErrTeamNotFound):
		writeError(c, http.StatusNotFound, err)
	case errors.Is(err, errs.ErrUnauthorized), errors.Is(err, errs.ErrAdminRequired):
		writeError(c, http.StatusForbidden, err)
	case errors.Is(err, errs.ErrActiveTournament):
		writeError(c, http.StatusConflict, err)
	case errors.Is(err, errs.ErrInvalidRequestBody), errors.Is(err, errs.ErrInvalidSwap):
		writeError(c, http.StatusBadRequest, err)
	case errors.Is(err, errs.ErrNotEnoughParticipants),
		errors.Is(err, errs.ErrTeamsAlreadyGenerated),
		errors.Is(err, errs.ErrNoTeams),
		errors.Is(err, errs.ErrGroupsNotGenerated),
		errors.Is(err, errs.ErrGroupStageIncomplete),
		errors.Is(err, errs.ErrPlayoffsAlreadyExist),
		errors.Is(err, errs.ErrInvalidStatus),
		errors.Is(err, errs.ErrMatchAlreadyCompleted),
		errors.Is(err, errs.ErrTieNotAllowed),
		errors.Is(err, errs.ErrMatchNotReady):
		writeError(c, http.StatusConflict, err)
	default:
		writeError(c, http.StatusInternalServerError, errors.New("something went wrong"))
	}
}

func writeError(c *gin.Context, code int, err error) {
	c.JSON(code, model.NewAPIError(code, util.CapitalizeFirstLetter(err), c))
}

// actor returns the authenticated username and whether they are an olympics admin.
func actor(c *gin.Context) (user string, isAdmin bool) {
	user = c.GetString(model.UserKey)
	if claims := util.ClaimsFromContext(c.Request.Context()); claims != nil {
		isAdmin = claims.IsAdmin()
	}
	return user, isAdmin
}
