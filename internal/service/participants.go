package service

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/google/uuid"
	"github.com/maxmorhardt/olympics-api/internal/errs"
	"github.com/maxmorhardt/olympics-api/internal/model"
	"github.com/maxmorhardt/olympics-api/internal/repository"
	"github.com/maxmorhardt/olympics-api/internal/util"
)

func (s *tournamentService) UpdateParticipant(ctx context.Context, id, participantID uuid.UUID, name string, user string, isAdmin bool) (*model.Tournament, error) {
	tournament, err := s.GetTournament(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := authorizeTournament(tournament, user, isAdmin); err != nil {
		return nil, err
	}

	if findParticipant(tournament, participantID) == nil {
		return nil, errs.ErrParticipantNotFound
	}

	if err := s.repo.UpdateParticipantName(ctx, participantID, name); err != nil {
		return nil, err
	}

	s.broadcaster.Broadcast(id, model.NewTournamentUpdated(id, tournament.Status))
	return s.GetTournament(ctx, id)
}

func (s *tournamentService) AddParticipant(ctx context.Context, id uuid.UUID, name string, user string, isAdmin bool) (*model.Tournament, error) {
	log := util.LoggerFromContext(ctx)

	tournament, err := s.GetTournament(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := authorizeTournament(tournament, user, isAdmin); err != nil {
		return nil, err
	}
	if tournament.Status == model.TournamentStatusFinished {
		return nil, errs.ErrInvalidStatus
	}

	participant := &model.Participant{ID: uuid.New(), TournamentID: id, Name: name}

	// before teams exist this is just another name on the list
	if tournament.Status == model.TournamentStatusSetup {
		if err := s.repo.AddParticipants(ctx, []*model.Participant{participant}); err != nil {
			return nil, err
		}
		s.broadcaster.Broadcast(id, model.NewTournamentUpdated(id, tournament.Status))
		return s.GetTournament(ctx, id)
	}

	ch := addToRoster(id, tournament.Teams, participant)
	if err := s.repo.ApplyRosterChange(ctx, id, ch); err != nil {
		log.Error("failed to add participant", "tournament_id", id, "error", err)
		return nil, err
	}

	s.broadcaster.Broadcast(id, model.NewTournamentUpdated(id, model.TournamentStatusTeamsGenerated))
	log.Info("added participant and rebuilt roster", "tournament_id", id)
	return s.GetTournament(ctx, id)
}

func (s *tournamentService) DeleteParticipant(ctx context.Context, id, participantID uuid.UUID, user string, isAdmin bool) (*model.Tournament, error) {
	log := util.LoggerFromContext(ctx)

	tournament, err := s.GetTournament(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := authorizeTournament(tournament, user, isAdmin); err != nil {
		return nil, err
	}
	if tournament.Status == model.TournamentStatusFinished {
		return nil, errs.ErrInvalidStatus
	}

	participant := findParticipant(tournament, participantID)
	if participant == nil {
		return nil, errs.ErrParticipantNotFound
	}

	// before teams exist just drop the name
	if tournament.Status == model.TournamentStatusSetup {
		if err := s.repo.DeleteParticipant(ctx, participantID); err != nil {
			return nil, err
		}
		s.broadcaster.Broadcast(id, model.NewTournamentUpdated(id, tournament.Status))
		return s.GetTournament(ctx, id)
	}

	ch := removeFromRoster(tournament.Teams, participant)
	if err := s.repo.ApplyRosterChange(ctx, id, ch); err != nil {
		log.Error("failed to delete participant", "tournament_id", id, "error", err)
		return nil, err
	}

	s.broadcaster.Broadcast(id, model.NewTournamentUpdated(id, model.TournamentStatusTeamsGenerated))
	log.Info("deleted participant and rebuilt roster", "tournament_id", id)
	return s.GetTournament(ctx, id)
}

func addToRoster(tournamentID uuid.UUID, teams []model.Team, participant *model.Participant) repository.RosterChange {
	ch := repository.RosterChange{Wipe: true, NewParticipant: participant}

	if short := teamWithFewerThan(teams, teamSize); short != nil {
		teamID := short.ID
		participant.TeamID = &teamID
		return ch
	}

	if big := teamWithAtLeast(teams, teamSize+1); big != nil {
		// pull a random member out to pair with the newcomer
		member := big.Members[rand.IntN(len(big.Members))]
		newTeam := &model.Team{ID: uuid.New(), TournamentID: tournamentID, Name: nextTeamName(teams)}
		participant.TeamID = &newTeam.ID
		ch.NewTeam = newTeam
		ch.Reassign = map[uuid.UUID]uuid.UUID{member.ID: newTeam.ID}
		return ch
	}

	// every team is a full pair: make one a trio
	target := teams[rand.IntN(len(teams))]
	participant.TeamID = &target.ID
	return ch
}

func removeFromRoster(teams []model.Team, participant *model.Participant) repository.RosterChange {
	ch := repository.RosterChange{Wipe: true, DeleteParticipant: &participant.ID}

	if participant.TeamID == nil {
		return ch
	}
	team := teamByID(teams, *participant.TeamID)
	if team == nil {
		return ch
	}

	if len(team.Members) >= teamSize+1 {
		// trio stays intact as a pair
		return ch
	}

	remaining := otherMember(team, participant.ID)
	if remaining == nil {
		// nothing left on the team, remove the empty shell
		ch.DeleteTeam = &team.ID
		return ch
	}

	if other := randomTeamExcept(teams, team.ID); other != nil {
		ch.Reassign = map[uuid.UUID]uuid.UUID{remaining.ID: other.ID}
		ch.DeleteTeam = &team.ID
	}
	return ch
}

func teamWithFewerThan(teams []model.Team, size int) *model.Team {
	for i := range teams {
		if len(teams[i].Members) < size {
			return &teams[i]
		}
	}
	return nil
}

func teamWithAtLeast(teams []model.Team, size int) *model.Team {
	for i := range teams {
		if len(teams[i].Members) >= size {
			return &teams[i]
		}
	}
	return nil
}

func teamByID(teams []model.Team, id uuid.UUID) *model.Team {
	for i := range teams {
		if teams[i].ID == id {
			return &teams[i]
		}
	}
	return nil
}

func otherMember(team *model.Team, excludeID uuid.UUID) *model.Participant {
	for i := range team.Members {
		if team.Members[i].ID != excludeID {
			return &team.Members[i]
		}
	}
	return nil
}

func randomTeamExcept(teams []model.Team, excludeID uuid.UUID) *model.Team {
	others := make([]*model.Team, 0, len(teams))
	for i := range teams {
		if teams[i].ID != excludeID {
			others = append(others, &teams[i])
		}
	}
	if len(others) == 0 {
		return nil
	}
	return others[rand.IntN(len(others))]
}

func nextTeamName(teams []model.Team) string {
	return fmt.Sprintf("Team %d", len(teams)+1)
}
