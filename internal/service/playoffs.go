package service

import (
	"context"
	"math/bits"
	"math/rand/v2"
	"sort"

	"github.com/google/uuid"
	"github.com/maxmorhardt/olympics-api/internal/errs"
	"github.com/maxmorhardt/olympics-api/internal/model"
	"github.com/maxmorhardt/olympics-api/internal/util"
)

func (s *tournamentService) GeneratePlayoffs(ctx context.Context, id uuid.UUID, user string, isAdmin bool) ([]model.Match, error) {
	log := util.LoggerFromContext(ctx)

	tournament, err := s.GetTournament(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := authorizeTournament(tournament, user, isAdmin); err != nil {
		return nil, err
	}

	if tournament.Status != model.TournamentStatusGroupStage {
		log.Warn("tournament not in group stage", "tournament_id", id, "status", tournament.Status)
		if tournament.Status == model.TournamentStatusPlayoffs || tournament.Status == model.TournamentStatusFinished {
			return nil, errs.ErrPlayoffsAlreadyExist
		}
		return nil, errs.ErrGroupsNotGenerated
	}

	// every group match must be played before seeding the bracket
	pending, err := s.matchRepo.CountPendingByStage(ctx, id, model.MatchStageGroup)
	if err != nil {
		return nil, err
	}
	if pending > 0 {
		return nil, errs.ErrGroupStageIncomplete
	}

	groupMatches, err := s.matchRepo.GetByTournamentAndStage(ctx, id, model.MatchStageGroup)
	if err != nil {
		return nil, err
	}

	standings := computeStandings(tournament.Groups, groupMatches)
	qualifiers := seedQualifiers(standings)
	if len(qualifiers) < 2 {
		return nil, errs.ErrInvalidStatus
	}

	matches := buildBracket(id, qualifiers, groupGames)

	if err := s.matchRepo.Create(ctx, matches); err != nil {
		log.Error("failed to persist playoff matches", "tournament_id", id, "error", err)
		return nil, err
	}

	if err := s.repo.UpdateStatus(ctx, id, model.TournamentStatusPlayoffs); err != nil {
		return nil, err
	}

	s.broadcaster.Broadcast(id, model.NewTournamentUpdated(id, model.TournamentStatusPlayoffs))
	log.Info("generated playoffs", "tournament_id", id, "qualifiers", len(qualifiers), "matches", len(matches))
	return s.matchRepo.GetByTournamentAndStage(ctx, id, model.MatchStagePlayoff)
}

func seedQualifiers(standings []model.GroupStandings) []uuid.UUID {
	var rows []model.TeamStanding
	for _, gs := range standings {
		rows = append(rows, gs.Standings...)
	}

	// shuffle first so equal-win ties resolve randomly under the stable sort
	rand.Shuffle(len(rows), func(i, j int) {
		rows[i], rows[j] = rows[j], rows[i]
	})
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].Wins > rows[j].Wins
	})

	qualifiers := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		qualifiers[i] = r.TeamID
	}
	return qualifiers
}

func buildBracket(tournamentID uuid.UUID, qualifiers []uuid.UUID, gameTypes []string) []*model.Match {
	q := len(qualifiers)
	bracketSize := 1
	for bracketSize < q {
		bracketSize <<= 1
	}
	rounds := bits.TrailingZeros(uint(bracketSize)) // number of rounds

	gameIdx := 0
	pickGame := func() string {
		if len(gameTypes) == 0 {
			return ""
		}
		g := gameTypes[gameIdx%len(gameTypes)]
		gameIdx++
		return g
	}

	// create every match, round by round (round 1 = first round, rounds = final)
	matchesByRound := make([][]*model.Match, rounds+1)
	for r := 1; r <= rounds; r++ {
		count := bracketSize >> r
		matchesByRound[r] = make([]*model.Match, count)
		for m := 0; m < count; m++ {
			matchesByRound[r][m] = &model.Match{
				ID:           uuid.New(),
				TournamentID: tournamentID,
				Stage:        model.MatchStagePlayoff,
				Round:        r,
				MatchNumber:  m,
				GameType:     pickGame(),
				Status:       model.MatchStatusPending,
			}
		}
	}

	// link each match to the slot it feeds in the next round
	for r := 1; r < rounds; r++ {
		for m, match := range matchesByRound[r] {
			next := matchesByRound[r+1][m/2]
			nextID := next.ID
			match.NextMatchID = &nextID
			if m%2 == 0 {
				match.NextSlot = "a"
			} else {
				match.NextSlot = "b"
			}
		}
	}

	// fill the first round from seed positions, then resolve byes
	positions := seedPositions(bracketSize)
	teamForSeed := func(seed int) *uuid.UUID {
		if seed > q {
			return nil
		}
		teamID := qualifiers[seed-1]
		return &teamID
	}

	for m, match := range matchesByRound[1] {
		match.TeamAID = teamForSeed(positions[2*m])
		match.TeamBID = teamForSeed(positions[2*m+1])

		// a bye: the present team auto-advances to the next round
		var winner *uuid.UUID
		switch {
		case match.TeamAID != nil && match.TeamBID == nil:
			winner = match.TeamAID
		case match.TeamBID != nil && match.TeamAID == nil:
			winner = match.TeamBID
		}

		if winner != nil {
			match.WinnerTeamID = winner
			match.Status = model.MatchStatusCompleted
			if rounds > 1 {
				next := matchesByRound[2][m/2]
				if m%2 == 0 {
					next.TeamAID = winner
				} else {
					next.TeamBID = winner
				}
			}
		}
	}

	// flatten final-round-first so each match is inserted after the one it references
	var matches []*model.Match
	for r := rounds; r >= 1; r-- {
		matches = append(matches, matchesByRound[r]...)
	}

	return matches
}

func seedPositions(size int) []int {
	seeds := []int{1}
	for len(seeds) < size {
		n := len(seeds) * 2
		next := make([]int, 0, n)
		for _, s := range seeds {
			next = append(next, s)
			next = append(next, n+1-s)
		}
		seeds = next
	}
	return seeds
}

func (s *tournamentService) GetBracket(ctx context.Context, id uuid.UUID) ([]model.Match, error) {
	if _, err := s.GetTournament(ctx, id); err != nil {
		return nil, err
	}
	return s.matchRepo.GetByTournamentAndStage(ctx, id, model.MatchStagePlayoff)
}
