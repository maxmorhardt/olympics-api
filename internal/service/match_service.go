package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/maxmorhardt/olympics-api/internal/errs"
	"github.com/maxmorhardt/olympics-api/internal/model"
	"github.com/maxmorhardt/olympics-api/internal/repository"
	"github.com/maxmorhardt/olympics-api/internal/util"
	"gorm.io/gorm"
)

type MatchService interface {
	GetMatches(ctx context.Context, tournamentID uuid.UUID) ([]model.Match, error)
	RecordResult(ctx context.Context, matchID uuid.UUID, req *model.RecordMatchResultRequest, user string, isAdmin bool) (*model.Match, error)
}

type matchService struct {
	matchRepo      repository.MatchRepository
	tournamentRepo repository.TournamentRepository
}

func NewMatchService(matchRepo repository.MatchRepository, tournamentRepo repository.TournamentRepository) MatchService {
	return &matchService{
		matchRepo:      matchRepo,
		tournamentRepo: tournamentRepo,
	}
}

func (s *matchService) GetMatches(ctx context.Context, tournamentID uuid.UUID) ([]model.Match, error) {
	return s.matchRepo.GetByTournament(ctx, tournamentID)
}

func (s *matchService) RecordResult(ctx context.Context, matchID uuid.UUID, req *model.RecordMatchResultRequest, user string, isAdmin bool) (*model.Match, error) {
	log := util.LoggerFromContext(ctx)

	match, err := s.matchRepo.GetByID(ctx, matchID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrMatchNotFound
		}
		return nil, err
	}

	// only the tournament creator or an olympics admin may record results
	tournament, err := s.tournamentRepo.GetByID(ctx, match.TournamentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrTournamentNotFound
		}
		return nil, err
	}
	if err := authorizeTournament(tournament, user, isAdmin); err != nil {
		return nil, err
	}

	if match.Status == model.MatchStatusCompleted {
		return nil, errs.ErrMatchAlreadyCompleted
	}

	if match.TeamAID == nil || match.TeamBID == nil {
		return nil, errs.ErrMatchNotReady
	}

	if req.TeamAScore == req.TeamBScore {
		return nil, errs.ErrTieNotAllowed
	}

	// record the score and resolve the winner
	match.TeamAScore = req.TeamAScore
	match.TeamBScore = req.TeamBScore
	if req.TeamAScore > req.TeamBScore {
		match.WinnerTeamID = match.TeamAID
	} else {
		match.WinnerTeamID = match.TeamBID
	}
	match.Status = model.MatchStatusCompleted

	if err := s.matchRepo.Update(ctx, match); err != nil {
		log.Error("failed to record match result", "match_id", matchID, "error", err)
		return nil, err
	}

	// advance the winner in the playoff bracket
	if match.Stage == model.MatchStagePlayoff {
		if err := s.advanceWinner(ctx, match); err != nil {
			log.Error("failed to advance playoff winner", "match_id", matchID, "error", err)
			return nil, err
		}
	}

	log.Info("recorded match result", "match_id", matchID, "winner", match.WinnerTeamID)
	return s.matchRepo.GetByID(ctx, matchID)
}

func (s *matchService) advanceWinner(ctx context.Context, match *model.Match) error {
	// the final has no next match: completing it finishes the tournament
	if match.NextMatchID == nil {
		return s.tournamentRepo.UpdateStatus(ctx, match.TournamentID, model.TournamentStatusFinished)
	}

	next, err := s.matchRepo.GetByID(ctx, *match.NextMatchID)
	if err != nil {
		return err
	}

	if match.NextSlot == "b" {
		next.TeamBID = match.WinnerTeamID
	} else {
		next.TeamAID = match.WinnerTeamID
	}

	return s.matchRepo.Update(ctx, next)
}
