package errs

import "errors"

var (
	ErrTournamentNotFound = errors.New("tournament not found")
	ErrMatchNotFound      = errors.New("match not found")
	ErrInvalidRequestBody = errors.New("invalid request body")
	ErrUnauthorized       = errors.New("only the tournament creator or an olympics admin can perform this action")
	ErrAdminRequired      = errors.New("only an olympics admin can create a tournament")
	ErrActiveTournament   = errors.New("a tournament is already in progress; only one is allowed at a time")
)

var (
	ErrNotEnoughParticipants = errors.New("at least two participants are required to generate teams")
	ErrTeamsAlreadyGenerated = errors.New("teams have already been generated for this tournament")
	ErrNoTeams               = errors.New("teams must be generated before this action")
	ErrGroupsNotGenerated    = errors.New("groups must be generated before this action")
	ErrGroupStageIncomplete  = errors.New("all group-stage matches must be completed before generating playoffs")
	ErrPlayoffsAlreadyExist  = errors.New("playoffs have already been generated for this tournament")
	ErrInvalidStatus         = errors.New("tournament is not in a valid state for this action")
)

var (
	ErrMatchAlreadyCompleted = errors.New("match result has already been recorded")
	ErrTieNotAllowed         = errors.New("matches cannot end in a tie")
	ErrMatchNotReady         = errors.New("both teams must be set before recording a result")
)
