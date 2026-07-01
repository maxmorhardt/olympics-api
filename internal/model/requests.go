package model

type CreateTournamentRequest struct {
	Name string `json:"name"`
}

type AddParticipantsRequest struct {
	Names []string `json:"names"`
}

type RecordMatchResultRequest struct {
	TeamAScore int `json:"teamAScore"`
	TeamBScore int `json:"teamBScore"`
}

type UpdateNameRequest struct {
	Name string `json:"name"`
}

type SwapPlayersRequest struct {
	ParticipantAId string `json:"participantAId"`
	ParticipantBId string `json:"participantBId"`
}
