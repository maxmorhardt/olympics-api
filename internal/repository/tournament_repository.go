package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/maxmorhardt/olympics-api/internal/model"
	"gorm.io/gorm"
)

type TournamentRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Tournament, error)
	GetAll(ctx context.Context) ([]model.Tournament, error)
	CountActive(ctx context.Context) (int64, error)
	GetParticipants(ctx context.Context, tournamentID uuid.UUID) ([]model.Participant, error)
	GetTeams(ctx context.Context, tournamentID uuid.UUID) ([]model.Team, error)
	GetGroups(ctx context.Context, tournamentID uuid.UUID) ([]model.Group, error)

	Create(ctx context.Context, tournament *model.Tournament) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.TournamentStatus) error
	Delete(ctx context.Context, id uuid.UUID) error
	AddParticipants(ctx context.Context, participants []*model.Participant) error
	UpdateParticipantName(ctx context.Context, participantID uuid.UUID, name string) error
	DeleteParticipant(ctx context.Context, participantID uuid.UUID) error
	UpdateTeamName(ctx context.Context, teamID uuid.UUID, name string) error
	SwapPlayers(ctx context.Context, aID, aTeamID, bID, bTeamID uuid.UUID) error
	ApplyRosterChange(ctx context.Context, tournamentID uuid.UUID, ch RosterChange) error

	CreateTeamsAndAssign(ctx context.Context, teams []*model.Team, participants []*model.Participant) error
	CreateGroupsAndAssign(ctx context.Context, groups []*model.Group, teams []*model.Team) error
}

type RosterChange struct {
	Wipe              bool
	NewTeam           *model.Team
	NewParticipant    *model.Participant
	DeleteParticipant *uuid.UUID
	DeleteTeam        *uuid.UUID
	Reassign          map[uuid.UUID]uuid.UUID
}

type tournamentRepository struct {
	db *gorm.DB
}

func NewTournamentRepository(db *gorm.DB) TournamentRepository {
	return &tournamentRepository{
		db: db,
	}
}

func (r *tournamentRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Tournament, error) {
	var tournament model.Tournament
	err := r.db.WithContext(ctx).
		Preload("Participants").
		Preload("Teams.Members").
		Preload("Groups.Teams.Members").
		First(&tournament, "id = ?", id).Error

	return &tournament, err
}

func (r *tournamentRepository) GetAll(ctx context.Context) ([]model.Tournament, error) {
	var tournaments []model.Tournament
	err := r.db.WithContext(ctx).Order("created_at DESC").Find(&tournaments).Error
	return tournaments, err
}

func (r *tournamentRepository) CountActive(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Tournament{}).
		Where("status != ?", model.TournamentStatusFinished).
		Count(&count).Error
	return count, err
}

func (r *tournamentRepository) GetParticipants(ctx context.Context, tournamentID uuid.UUID) ([]model.Participant, error) {
	var participants []model.Participant
	err := r.db.WithContext(ctx).Where("tournament_id = ?", tournamentID).Find(&participants).Error
	return participants, err
}

func (r *tournamentRepository) GetTeams(ctx context.Context, tournamentID uuid.UUID) ([]model.Team, error) {
	var teams []model.Team
	err := r.db.WithContext(ctx).
		Preload("Members").
		Where("tournament_id = ?", tournamentID).
		Order("seed ASC").
		Find(&teams).Error
	return teams, err
}

func (r *tournamentRepository) GetGroups(ctx context.Context, tournamentID uuid.UUID) ([]model.Group, error) {
	var groups []model.Group
	err := r.db.WithContext(ctx).
		Preload("Teams.Members").
		Where("tournament_id = ?", tournamentID).
		Order("name ASC").
		Find(&groups).Error
	return groups, err
}

func (r *tournamentRepository) Create(ctx context.Context, tournament *model.Tournament) error {
	return r.db.WithContext(ctx).Create(tournament).Error
}

func (r *tournamentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.TournamentStatus) error {
	return r.db.WithContext(ctx).
		Model(&model.Tournament{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *tournamentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// FK cascades remove participants, teams, groups, and matches
	return r.db.WithContext(ctx).Delete(&model.Tournament{}, "id = ?", id).Error
}

func (r *tournamentRepository) UpdateTeamName(ctx context.Context, teamID uuid.UUID, name string) error {
	return r.db.WithContext(ctx).
		Model(&model.Team{}).
		Where("id = ?", teamID).
		Update("name", name).Error
}

func (r *tournamentRepository) SwapPlayers(ctx context.Context, aID, aTeamID, bID, bTeamID uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Participant{}).Where("id = ?", aID).Update("team_id", aTeamID).Error; err != nil {
			return err
		}
		return tx.Model(&model.Participant{}).Where("id = ?", bID).Update("team_id", bTeamID).Error
	})
}

func (r *tournamentRepository) AddParticipants(ctx context.Context, participants []*model.Participant) error {
	if len(participants) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(participants).Error
}

func (r *tournamentRepository) UpdateParticipantName(ctx context.Context, participantID uuid.UUID, name string) error {
	return r.db.WithContext(ctx).
		Model(&model.Participant{}).
		Where("id = ?", participantID).
		Update("name", name).Error
}

func (r *tournamentRepository) DeleteParticipant(ctx context.Context, participantID uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Participant{}, "id = ?", participantID).Error
}

func (r *tournamentRepository) ApplyRosterChange(ctx context.Context, tournamentID uuid.UUID, ch RosterChange) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if ch.Wipe {
			// clear the schedule and unlink teams from their (now stale) groups
			if err := tx.Where("tournament_id = ?", tournamentID).Delete(&model.Match{}).Error; err != nil {
				return err
			}
			if err := tx.Where("tournament_id = ?", tournamentID).Delete(&model.Group{}).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.Team{}).Where("tournament_id = ?", tournamentID).
				Updates(map[string]any{"group_id": nil, "seed": 0}).Error; err != nil {
				return err
			}
		}

		// create a new team before anything is reassigned onto it
		if ch.NewTeam != nil {
			if err := tx.Create(ch.NewTeam).Error; err != nil {
				return err
			}
		}
		for participantID, teamID := range ch.Reassign {
			if err := tx.Model(&model.Participant{}).Where("id = ?", participantID).Update("team_id", teamID).Error; err != nil {
				return err
			}
		}
		if ch.NewParticipant != nil {
			if err := tx.Create(ch.NewParticipant).Error; err != nil {
				return err
			}
		}
		if ch.DeleteParticipant != nil {
			if err := tx.Delete(&model.Participant{}, "id = ?", *ch.DeleteParticipant).Error; err != nil {
				return err
			}
		}
		// delete an emptied team only after its members have moved off it
		if ch.DeleteTeam != nil {
			if err := tx.Delete(&model.Team{}, "id = ?", *ch.DeleteTeam).Error; err != nil {
				return err
			}
		}

		if ch.Wipe {
			return tx.Model(&model.Tournament{}).Where("id = ?", tournamentID).
				Update("status", model.TournamentStatusTeamsGenerated).Error
		}
		return nil
	})
}

func (r *tournamentRepository) CreateTeamsAndAssign(ctx context.Context, teams []*model.Team, participants []*model.Participant) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(teams).Error; err != nil {
			return err
		}

		// assign each participant to its generated team
		for _, p := range participants {
			if err := tx.Model(&model.Participant{}).Where("id = ?", p.ID).Update("team_id", p.TeamID).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *tournamentRepository) CreateGroupsAndAssign(ctx context.Context, groups []*model.Group, teams []*model.Team) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(groups).Error; err != nil {
			return err
		}

		// assign each team to its group and persist its seed
		for _, t := range teams {
			if err := tx.Model(&model.Team{}).Where("id = ?", t.ID).
				Updates(map[string]any{"group_id": t.GroupID, "seed": t.Seed}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
