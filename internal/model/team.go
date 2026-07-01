package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Team struct {
	ID           uuid.UUID     `json:"id" gorm:"type:uuid;primaryKey"`
	TournamentID uuid.UUID     `json:"tournamentId" gorm:"type:uuid;index;not null"`
	GroupID      *uuid.UUID    `json:"groupId,omitempty" gorm:"type:uuid;index"`
	Name         string        `json:"name" gorm:"not null"`
	Seed         int           `json:"seed" gorm:"not null;default:0"`
	Members      []Participant `json:"members,omitempty" gorm:"foreignKey:TeamID"`
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedAt    time.Time     `json:"updatedAt"`
}

func (t *Team) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return
}
