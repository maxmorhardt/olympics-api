package service

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/google/uuid"
	"github.com/maxmorhardt/olympics-api/internal/errs"
	"github.com/maxmorhardt/olympics-api/internal/model"
	"github.com/maxmorhardt/olympics-api/internal/util"
)

func (s *tournamentService) GenerateGroups(ctx context.Context, id uuid.UUID, user string, isAdmin bool) ([]model.Group, error) {
	log := util.LoggerFromContext(ctx)

	tournament, err := s.GetTournament(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := authorizeTournament(tournament, user, isAdmin); err != nil {
		return nil, err
	}

	if tournament.Status != model.TournamentStatusTeamsGenerated {
		log.Warn("tournament not ready for group generation", "tournament_id", id, "status", tournament.Status)
		if tournament.Status == model.TournamentStatusSetup {
			return nil, errs.ErrNoTeams
		}
		return nil, errs.ErrInvalidStatus
	}

	teams, err := s.repo.GetTeams(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := shuffleTeams(teams); err != nil {
		return nil, err
	}

	// split teams into balanced groups, round-robin dealing for even sizing
	numGroups := (len(teams) + tournament.TeamsPerGroup - 1) / tournament.TeamsPerGroup
	if numGroups < 1 {
		numGroups = 1
	}

	groups := make([]*model.Group, numGroups)
	for i := range groups {
		groups[i] = &model.Group{
			ID:           uuid.New(),
			TournamentID: id,
			Name:         fmt.Sprintf("Group %c", 'A'+i),
		}
	}

	groupedTeams := make([][]*model.Team, numGroups)
	updatedTeams := make([]*model.Team, len(teams))
	for i := range teams {
		t := teams[i]
		gIdx := i % numGroups
		groupID := groups[gIdx].ID
		t.GroupID = &groupID
		updatedTeams[i] = &t
		groupedTeams[gIdx] = append(groupedTeams[gIdx], &t)
	}

	if err := s.repo.CreateGroupsAndAssign(ctx, groups, updatedTeams); err != nil {
		log.Error("failed to persist groups", "tournament_id", id, "error", err)
		return nil, err
	}

	// build round-robin matches within each group
	gameTypes := s.gameTypes(tournament)
	matches := buildGroupMatches(id, groups, groupedTeams, gameTypes)

	if err := s.matchRepo.Create(ctx, matches); err != nil {
		log.Error("failed to persist group matches", "tournament_id", id, "error", err)
		return nil, err
	}

	if err := s.repo.UpdateStatus(ctx, id, model.TournamentStatusGroupStage); err != nil {
		return nil, err
	}

	log.Info("generated groups", "tournament_id", id, "groups", numGroups, "matches", len(matches))
	return s.repo.GetGroups(ctx, id)
}

func buildGroupMatches(tournamentID uuid.UUID, groups []*model.Group, groupedTeams [][]*model.Team, gameTypes []string) []*model.Match {
	var matches []*model.Match
	matchNumber := 0
	gameIdx := 0

	for gIdx, teams := range groupedTeams {
		groupID := groups[gIdx].ID
		// round-robin: every team plays every other team in the group once
		for i := 0; i < len(teams); i++ {
			for j := i + 1; j < len(teams); j++ {
				teamA := teams[i].ID
				teamB := teams[j].ID
				gameType := ""
				if len(gameTypes) > 0 {
					gameType = gameTypes[gameIdx%len(gameTypes)]
					gameIdx++
				}

				matches = append(matches, &model.Match{
					ID:           uuid.New(),
					TournamentID: tournamentID,
					GroupID:      &groupID,
					Stage:        model.MatchStageGroup,
					MatchNumber:  matchNumber,
					GameType:     gameType,
					TeamAID:      &teamA,
					TeamBID:      &teamB,
					Status:       model.MatchStatusPending,
				})
				matchNumber++
			}
		}
	}

	return matches
}

func (s *tournamentService) gameTypes(tournament *model.Tournament) []string {
	var gameTypes []string
	if len(tournament.GameTypes) > 0 {
		_ = json.Unmarshal(tournament.GameTypes, &gameTypes)
	}
	if len(gameTypes) == 0 {
		gameTypes = defaultGameTypes
	}
	return gameTypes
}

func shuffleTeams(teams []model.Team) error {
	for i := len(teams) - 1; i > 0; i-- {
		n, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}
		j := int(n.Int64())
		teams[i], teams[j] = teams[j], teams[i]
	}
	return nil
}
