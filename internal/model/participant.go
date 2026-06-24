package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Participant struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	TournamentID uuid.UUID  `json:"tournamentId" gorm:"type:uuid;index;not null"`
	TeamID       *uuid.UUID `json:"teamId,omitempty" gorm:"type:uuid;index"`
	Name         string     `json:"name" gorm:"not null"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

func (p *Participant) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return
}
