package model

import "github.com/google/uuid"
type TeamStanding struct {
	TeamID        uuid.UUID `json:"teamId"`
	TeamName      string    `json:"teamName"`
	Played        int       `json:"played"`
	Wins          int       `json:"wins"`
	Losses        int       `json:"losses"`
	PointsFor     int       `json:"pointsFor"`
	PointsAgainst int       `json:"pointsAgainst"`
	PointDiff     int       `json:"pointDiff"`
}

type GroupStandings struct {
	GroupID   uuid.UUID      `json:"groupId"`
	GroupName string         `json:"groupName"`
	Standings []TeamStanding `json:"standings"`
}
