package service

import (
	"context"
	cryptorand "crypto/rand"
	"fmt"
	"math"
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

	// split teams into balanced groups (6 per group), round-robin dealing
	numGroups := max((len(teams)+teamsPerGroup-1)/teamsPerGroup, 1)

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

	// schedule every group's round-robin into equipment-aware rounds
	matches := scheduleGroupMatches(id, groups, groupedTeams)

	if err := s.matchRepo.Create(ctx, matches); err != nil {
		log.Error("failed to persist group matches", "tournament_id", id, "error", err)
		return nil, err
	}

	if err := s.repo.UpdateStatus(ctx, id, model.TournamentStatusGroupStage); err != nil {
		return nil, err
	}

	s.broadcaster.Broadcast(id, model.NewTournamentUpdated(id, model.TournamentStatusGroupStage))
	log.Info("generated groups", "tournament_id", id, "groups", numGroups, "matches", len(matches))
	return s.repo.GetGroups(ctx, id)
}

// scheduleGroupMatches builds a single round-robin per group and aligns the groups
// so that every group plays its nth round in the same global round. A 6-team group
// is 5 rounds of 3 matches (everyone plays every round); two groups fill all six
// equipment stations (2 darts, 2 bocce, 2 cornhole) so the whole field plays at
// once. Within each round games are assigned to balance every team's games and
// respect equipment, so a team plays each game at least once across the stage.
func scheduleGroupMatches(tournamentID uuid.UUID, groups []*model.Group, groupedTeams [][]*model.Team) []*model.Match {
	// round-robin matchings (circle method) for each group
	groupMatchings := make([][][][2]uuid.UUID, len(groupedTeams))
	maxRounds := 0
	for gi, teams := range groupedTeams {
		ids := make([]uuid.UUID, len(teams))
		for i, t := range teams {
			ids[i] = t.ID
		}
		m := roundRobinMatchings(ids)
		groupMatchings[gi] = m
		if len(m) > maxRounds {
			maxRounds = len(m)
		}
	}

	teamGameCount := map[uuid.UUID]map[string]int{}
	gc := func(t uuid.UUID) map[string]int {
		if teamGameCount[t] == nil {
			teamGameCount[t] = map[string]int{}
		}
		return teamGameCount[t]
	}

	var matches []*model.Match
	matchNumber := 0

	for r := 0; r < maxRounds; r++ {
		roundUsage := map[string]int{}
		for gi := range groupedTeams {
			if r >= len(groupMatchings[gi]) {
				continue
			}
			groupID := groups[gi].ID
			for _, pr := range groupMatchings[gi][r] {
				a, b := pr[0], pr[1]
				game := pickGame(roundUsage, gc(a), gc(b))
				roundUsage[game]++
				gc(a)[game]++
				gc(b)[game]++

				teamA, teamB, grp := a, b, groupID
				matches = append(matches, &model.Match{
					ID:           uuid.New(),
					TournamentID: tournamentID,
					GroupID:      &grp,
					Stage:        model.MatchStageGroup,
					Round:        r + 1,
					MatchNumber:  matchNumber,
					GameType:     game,
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

// roundRobinMatchings returns a 1-factorization of the teams via the circle
// method: each inner slice is one round of simultaneous, disjoint matches, and
// every pair meets exactly once. An odd count gets a bye each round.
func roundRobinMatchings(ids []uuid.UUID) [][][2]uuid.UUID {
	arr := append([]uuid.UUID{}, ids...)
	if len(arr)%2 == 1 {
		arr = append(arr, uuid.Nil) // bye marker
	}
	n := len(arr)
	if n < 2 {
		return nil
	}

	rounds := make([][][2]uuid.UUID, 0, n-1)
	for r := 0; r < n-1; r++ {
		var ms [][2]uuid.UUID
		for i := 0; i < n/2; i++ {
			a, b := arr[i], arr[n-1-i]
			if a != uuid.Nil && b != uuid.Nil {
				ms = append(ms, [2]uuid.UUID{a, b})
			}
		}
		rounds = append(rounds, ms)

		// rotate all but the first element clockwise
		last := arr[n-1]
		copy(arr[2:], arr[1:n-1])
		arr[1] = last
	}
	return rounds
}

// pickGame chooses a game for a match within a round: it must have a free station
// this round, and among those it prefers the game both teams have played least so
// each team's three games stay balanced.
func pickGame(roundUsage, aCount, bCount map[string]int) string {
	best := ""
	bestScore := math.MaxInt
	for _, g := range groupGames {
		if roundUsage[g] >= gameCapacity[g] {
			continue
		}
		score := (aCount[g]+bCount[g])*100 + roundUsage[g]
		if score < bestScore {
			bestScore = score
			best = g
		}
	}

	// more matches this round than stations (3+ groups): fall back to least played
	if best == "" {
		for _, g := range groupGames {
			score := aCount[g] + bCount[g]
			if score < bestScore {
				bestScore = score
				best = g
			}
		}
	}

	return best
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
