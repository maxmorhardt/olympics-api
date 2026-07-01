package model

import (
	"time"

	"github.com/google/uuid"
)

type WSType string

const (
	WSTournamentUpdated WSType = "tournament_updated"
	WSScoreRecorded     WSType = "score_recorded"
	WSTournamentDeleted WSType = "tournament_deleted"
)

// WSScore carries the detail needed to render a score popup on clients.
type WSScore struct {
	Stage      string `json:"stage"`
	GameType   string `json:"gameType"`
	TeamAName  string `json:"teamAName"`
	TeamBName  string `json:"teamBName"`
	TeamAScore int    `json:"teamAScore"`
	TeamBScore int    `json:"teamBScore"`
	WinnerName string `json:"winnerName"`
}

type WSMessage struct {
	Type         WSType           `json:"type"`
	TournamentID uuid.UUID        `json:"tournamentId"`
	Status       TournamentStatus `json:"status,omitempty"`
	Score        *WSScore         `json:"score,omitempty"`
	Timestamp    time.Time        `json:"timestamp"`
}

func NewTournamentUpdated(id uuid.UUID, status TournamentStatus) *WSMessage {
	return &WSMessage{
		Type:         WSTournamentUpdated,
		TournamentID: id,
		Status:       status,
		Timestamp:    time.Now(),
	}
}

func NewScoreRecorded(id uuid.UUID, status TournamentStatus, score *WSScore) *WSMessage {
	return &WSMessage{
		Type:         WSScoreRecorded,
		TournamentID: id,
		Status:       status,
		Score:        score,
		Timestamp:    time.Now(),
	}
}

func NewTournamentDeleted(id uuid.UUID) *WSMessage {
	return &WSMessage{
		Type:         WSTournamentDeleted,
		TournamentID: id,
		Timestamp:    time.Now(),
	}
}
