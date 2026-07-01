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
	RollbackResult(ctx context.Context, matchID uuid.UUID, user string, isAdmin bool) (*model.Match, error)
}

type matchService struct {
	matchRepo      repository.MatchRepository
	tournamentRepo repository.TournamentRepository
	broadcaster    Broadcaster
}

func NewMatchService(matchRepo repository.MatchRepository, tournamentRepo repository.TournamentRepository, broadcaster Broadcaster) MatchService {
	return &matchService{
		matchRepo:      matchRepo,
		tournamentRepo: tournamentRepo,
		broadcaster:    broadcaster,
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

	// reload to capture preloaded team names and the (possibly transitioned) status
	updated, err := s.matchRepo.GetByID(ctx, matchID)
	if err != nil {
		return nil, err
	}
	s.broadcastScore(ctx, updated)

	log.Info("recorded match result", "match_id", matchID, "winner", match.WinnerTeamID)
	return updated, nil
}

func (s *matchService) broadcastScore(ctx context.Context, match *model.Match) {
	// default from the match stage so a DB error does not misreport the stage
	status := model.TournamentStatusGroupStage
	if match.Stage == model.MatchStagePlayoff {
		status = model.TournamentStatusPlayoffs
	}
	if t, err := s.tournamentRepo.GetByID(ctx, match.TournamentID); err == nil {
		status = t.Status
	}

	teamAName := teamNameOrTBD(match.TeamA)
	teamBName := teamNameOrTBD(match.TeamB)
	winnerName := teamAName
	if match.WinnerTeamID != nil && match.TeamBID != nil && *match.WinnerTeamID == *match.TeamBID {
		winnerName = teamBName
	}

	score := &model.WSScore{
		Stage:      string(match.Stage),
		GameType:   match.GameType,
		TeamAName:  teamAName,
		TeamBName:  teamBName,
		TeamAScore: match.TeamAScore,
		TeamBScore: match.TeamBScore,
		WinnerName: winnerName,
	}
	s.broadcaster.Broadcast(match.TournamentID, model.NewScoreRecorded(match.TournamentID, status, score))
}

func teamNameOrTBD(t *model.Team) string {
	if t == nil {
		return "TBD"
	}
	return t.Name
}

func (s *matchService) RollbackResult(ctx context.Context, matchID uuid.UUID, user string, isAdmin bool) (*model.Match, error) {
	log := util.LoggerFromContext(ctx)

	match, err := s.matchRepo.GetByID(ctx, matchID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrMatchNotFound
		}
		return nil, err
	}

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

	if match.Status != model.MatchStatusCompleted {
		return nil, errs.ErrMatchNotCompleted
	}
	// a bye has no entered result to undo
	if match.TeamAID == nil || match.TeamBID == nil {
		return nil, errs.ErrRollbackLocked
	}

	newStatus := tournament.Status
	if match.Stage == model.MatchStageGroup {
		// group scores lock once the bracket exists
		if tournament.Status != model.TournamentStatusGroupStage {
			return nil, errs.ErrRollbackLocked
		}
	} else if match.NextMatchID != nil {
		// cannot undo once the winner's next game has been played
		next, err := s.matchRepo.GetByID(ctx, *match.NextMatchID)
		if err != nil {
			return nil, err
		}
		if next.Status == model.MatchStatusCompleted {
			return nil, errs.ErrRollbackLocked
		}
		if match.NextSlot == "b" {
			next.TeamBID = nil
		} else {
			next.TeamAID = nil
		}
		if err := s.matchRepo.Update(ctx, next); err != nil {
			log.Error("failed to clear next match slot", "match_id", matchID, "error", err)
			return nil, err
		}
	} else if tournament.Status == model.TournamentStatusFinished {
		// undoing the final reopens the playoffs
		newStatus = model.TournamentStatusPlayoffs
		if err := s.tournamentRepo.UpdateStatus(ctx, tournament.ID, newStatus); err != nil {
			return nil, err
		}
	}

	match.Status = model.MatchStatusPending
	match.TeamAScore = 0
	match.TeamBScore = 0
	match.WinnerTeamID = nil
	if err := s.matchRepo.Update(ctx, match); err != nil {
		log.Error("failed to roll back match result", "match_id", matchID, "error", err)
		return nil, err
	}

	s.broadcaster.Broadcast(match.TournamentID, model.NewTournamentUpdated(match.TournamentID, newStatus))
	log.Info("rolled back match result", "match_id", matchID)
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
