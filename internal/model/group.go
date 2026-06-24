package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Group struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	TournamentID uuid.UUID `json:"tournamentId" gorm:"type:uuid;index;not null"`
	Name         string    `json:"name" gorm:"not null"`
	Teams        []Team    `json:"teams,omitempty" gorm:"foreignKey:GroupID"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (g *Group) BeforeCreate(tx *gorm.DB) (err error) {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return
}
