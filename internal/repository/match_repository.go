package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/maxmorhardt/olympics-api/internal/model"
	"gorm.io/gorm"
)

type MatchRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Match, error)
	GetByTournament(ctx context.Context, tournamentID uuid.UUID) ([]model.Match, error)
	GetByTournamentAndStage(ctx context.Context, tournamentID uuid.UUID, stage model.MatchStage) ([]model.Match, error)
	CountPendingByStage(ctx context.Context, tournamentID uuid.UUID, stage model.MatchStage) (int64, error)

	Create(ctx context.Context, matches []*model.Match) error
	Update(ctx context.Context, match *model.Match) error
}

type matchRepository struct {
	db *gorm.DB
}

func NewMatchRepository(db *gorm.DB) MatchRepository {
	return &matchRepository{
		db: db,
	}
}

func (r *matchRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Match, error) {
	var match model.Match
	err := r.db.WithContext(ctx).
		Preload("TeamA.Members").
		Preload("TeamB.Members").
		First(&match, "id = ?", id).Error
	return &match, err
}

func (r *matchRepository) GetByTournament(ctx context.Context, tournamentID uuid.UUID) ([]model.Match, error) {
	var matches []model.Match
	err := r.db.WithContext(ctx).
		Preload("TeamA").
		Preload("TeamB").
		Where("tournament_id = ?", tournamentID).
		Order("stage ASC, round ASC, match_number ASC").
		Find(&matches).Error
	return matches, err
}

func (r *matchRepository) GetByTournamentAndStage(ctx context.Context, tournamentID uuid.UUID, stage model.MatchStage) ([]model.Match, error) {
	var matches []model.Match
	err := r.db.WithContext(ctx).
		Preload("TeamA").
		Preload("TeamB").
		Where("tournament_id = ? AND stage = ?", tournamentID, stage).
		Order("round ASC, match_number ASC").
		Find(&matches).Error
	return matches, err
}

func (r *matchRepository) CountPendingByStage(ctx context.Context, tournamentID uuid.UUID, stage model.MatchStage) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Match{}).
		Where("tournament_id = ? AND stage = ? AND status != ?", tournamentID, stage, model.MatchStatusCompleted).
		Count(&count).Error
	return count, err
}

func (r *matchRepository) Create(ctx context.Context, matches []*model.Match) error {
	if len(matches) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(matches).Error
}

func (r *matchRepository) Update(ctx context.Context, match *model.Match) error {
	return r.db.WithContext(ctx).Save(match).Error
}
