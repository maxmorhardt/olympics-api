package service

import (
	"context"
	"sort"

	"github.com/google/uuid"
	"github.com/maxmorhardt/olympics-api/internal/model"
	"github.com/maxmorhardt/olympics-api/internal/util"
)

func (s *tournamentService) GetStandings(ctx context.Context, id uuid.UUID) ([]model.GroupStandings, error) {
	log := util.LoggerFromContext(ctx)

	tournament, err := s.GetTournament(ctx, id)
	if err != nil {
		return nil, err
	}

	matches, err := s.matchRepo.GetByTournamentAndStage(ctx, id, model.MatchStageGroup)
	if err != nil {
		log.Error("failed to get group matches for standings", "tournament_id", id, "error", err)
		return nil, err
	}

	standings := computeStandings(tournament.Groups, matches)
	log.Info("retrieved standings", "tournament_id", id, "groups", len(standings))
	return standings, nil
}

func computeStandings(groups []model.Group, matches []model.Match) []model.GroupStandings {
	result := make([]model.GroupStandings, 0, len(groups))

	for _, group := range groups {
		// seed a standing row for every team in the group
		rows := make(map[uuid.UUID]*model.TeamStanding, len(group.Teams))
		for _, team := range group.Teams {
			rows[team.ID] = &model.TeamStanding{
				TeamID:   team.ID,
				TeamName: team.Name,
			}
		}

		// fold completed group matches into the standing rows
		for i := range matches {
			m := matches[i]
			if m.GroupID == nil || *m.GroupID != group.ID || m.Status != model.MatchStatusCompleted {
				continue
			}
			if m.TeamAID == nil || m.TeamBID == nil {
				continue
			}

			a, aok := rows[*m.TeamAID]
			b, bok := rows[*m.TeamBID]
			if !aok || !bok {
				continue
			}

			a.Played++
			b.Played++
			a.PointsFor += m.TeamAScore
			a.PointsAgainst += m.TeamBScore
			b.PointsFor += m.TeamBScore
			b.PointsAgainst += m.TeamAScore

			if m.TeamAScore > m.TeamBScore {
				a.Wins++
				b.Losses++
			} else {
				b.Wins++
				a.Losses++
			}
		}

		standings := make([]model.TeamStanding, 0, len(rows))
		for _, row := range rows {
			row.PointDiff = row.PointsFor - row.PointsAgainst
			standings = append(standings, *row)
		}

		sortStandings(standings)

		result = append(result, model.GroupStandings{
			GroupID:   group.ID,
			GroupName: group.Name,
			Standings: standings,
		})
	}

	return result
}

func sortStandings(standings []model.TeamStanding) {
	sort.SliceStable(standings, func(i, j int) bool {
		if standings[i].Wins != standings[j].Wins {
			return standings[i].Wins > standings[j].Wins
		}
		if standings[i].PointDiff != standings[j].PointDiff {
			return standings[i].PointDiff > standings[j].PointDiff
		}
		return standings[i].PointsFor > standings[j].PointsFor
	})
}
