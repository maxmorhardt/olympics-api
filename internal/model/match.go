package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MatchStage string

const (
	MatchStageGroup   MatchStage = "group"
	MatchStagePlayoff MatchStage = "playoff"
)

type MatchStatus string

const (
	MatchStatusPending   MatchStatus = "pending"
	MatchStatusCompleted MatchStatus = "completed"
)

// Match models both a group-stage round-robin game and a single-elimination
// playoff game. For playoff games NextMatchID/NextSlot link the winner forward.
type Match struct {
	ID           uuid.UUID   `json:"id" gorm:"type:uuid;primaryKey"`
	TournamentID uuid.UUID   `json:"tournamentId" gorm:"type:uuid;index;not null"`
	GroupID      *uuid.UUID  `json:"groupId,omitempty" gorm:"type:uuid;index"`
	Stage        MatchStage  `json:"stage" gorm:"not null"`
	Round        int         `json:"round" gorm:"not null;default:0"`
	MatchNumber  int         `json:"matchNumber" gorm:"not null;default:0"`
	GameType     string      `json:"gameType"`
	TeamAID      *uuid.UUID  `json:"teamAId,omitempty" gorm:"type:uuid"`
	TeamBID      *uuid.UUID  `json:"teamBId,omitempty" gorm:"type:uuid"`
	TeamA        *Team       `json:"teamA,omitempty" gorm:"foreignKey:TeamAID"`
	TeamB        *Team       `json:"teamB,omitempty" gorm:"foreignKey:TeamBID"`
	TeamAScore   int         `json:"teamAScore" gorm:"not null;default:0"`
	TeamBScore   int         `json:"teamBScore" gorm:"not null;default:0"`
	WinnerTeamID *uuid.UUID  `json:"winnerTeamId,omitempty" gorm:"type:uuid"`
	Status       MatchStatus `json:"status" gorm:"not null;default:pending"`
	NextMatchID  *uuid.UUID  `json:"nextMatchId,omitempty" gorm:"type:uuid"`
	NextSlot     string      `json:"nextSlot,omitempty"`
	CreatedAt    time.Time   `json:"createdAt"`
	UpdatedAt    time.Time   `json:"updatedAt"`
}

func (m *Match) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return
}
