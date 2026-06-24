package model

type CreateTournamentRequest struct {
	Name            string   `json:"name"`
	TeamSize        int      `json:"teamSize"`
	TeamsPerGroup   int      `json:"teamsPerGroup"`
	AdvancePerGroup int      `json:"advancePerGroup"`
	GameTypes       []string `json:"gameTypes"`
}

type AddParticipantsRequest struct {
	Names []string `json:"names"`
}

type RecordMatchResultRequest struct {
	TeamAScore int `json:"teamAScore"`
	TeamBScore int `json:"teamBScore"`
}
