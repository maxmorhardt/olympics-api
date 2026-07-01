package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TournamentStatus string

const (
	TournamentStatusSetup          TournamentStatus = "setup"
	TournamentStatusTeamsGenerated TournamentStatus = "teams_generated"
	TournamentStatusGroupStage     TournamentStatus = "group_stage"
	TournamentStatusPlayoffs       TournamentStatus = "playoffs"
	TournamentStatusFinished       TournamentStatus = "finished"
)

type Tournament struct {
	ID           uuid.UUID        `json:"id" gorm:"type:uuid;primaryKey"`
	Name         string           `json:"name"`
	Status       TournamentStatus `json:"status" gorm:"not null;default:setup"`
	CreatedBy    string           `json:"createdBy" gorm:"index"`
	Participants []Participant    `json:"participants,omitempty" gorm:"foreignKey:TournamentID;constraint:OnDelete:CASCADE"`
	Teams        []Team           `json:"teams,omitempty" gorm:"foreignKey:TournamentID;constraint:OnDelete:CASCADE"`
	Groups       []Group          `json:"groups,omitempty" gorm:"foreignKey:TournamentID;constraint:OnDelete:CASCADE"`
	CreatedAt    time.Time        `json:"createdAt"`
	UpdatedAt    time.Time        `json:"updatedAt"`
}

func (t *Tournament) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return
}
