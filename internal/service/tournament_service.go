package service

import (
	"context"
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"github.com/google/uuid"
	"github.com/maxmorhardt/olympics-api/internal/errs"
	"github.com/maxmorhardt/olympics-api/internal/model"
	"github.com/maxmorhardt/olympics-api/internal/repository"
	"github.com/maxmorhardt/olympics-api/internal/util"
	"gorm.io/gorm"
)

type TournamentService interface {
	GetTournaments(ctx context.Context) ([]model.Tournament, error)
	GetTournament(ctx context.Context, id uuid.UUID) (*model.Tournament, error)

	CreateTournament(ctx context.Context, req *model.CreateTournamentRequest, user string, isAdmin bool) (*model.Tournament, error)
	AddParticipants(ctx context.Context, id uuid.UUID, names []string, user string, isAdmin bool) (*model.Tournament, error)
	AddParticipant(ctx context.Context, id uuid.UUID, name string, user string, isAdmin bool) (*model.Tournament, error)
	UpdateParticipant(ctx context.Context, id, participantID uuid.UUID, name string, user string, isAdmin bool) (*model.Tournament, error)
	DeleteParticipant(ctx context.Context, id, participantID uuid.UUID, user string, isAdmin bool) (*model.Tournament, error)

	GenerateTeams(ctx context.Context, id uuid.UUID, user string, isAdmin bool) ([]model.Team, error)
	GenerateGroups(ctx context.Context, id uuid.UUID, user string, isAdmin bool) ([]model.Group, error)
	GeneratePlayoffs(ctx context.Context, id uuid.UUID, user string, isAdmin bool) ([]model.Match, error)

	DeleteTournament(ctx context.Context, id uuid.UUID, user string, isAdmin bool) error
	UpdateTeam(ctx context.Context, id, teamID uuid.UUID, name string, user string, isAdmin bool) (*model.Tournament, error)
	SwapPlayers(ctx context.Context, id, participantAID, participantBID uuid.UUID, user string, isAdmin bool) (*model.Tournament, error)

	GetStandings(ctx context.Context, id uuid.UUID) ([]model.GroupStandings, error)
	GetBracket(ctx context.Context, id uuid.UUID) ([]model.Match, error)
}

type tournamentService struct {
	repo        repository.TournamentRepository
	matchRepo   repository.MatchRepository
	broadcaster Broadcaster
}

func NewTournamentService(repo repository.TournamentRepository, matchRepo repository.MatchRepository, broadcaster Broadcaster) TournamentService {
	return &tournamentService{
		repo:        repo,
		matchRepo:   matchRepo,
		broadcaster: broadcaster,
	}
}

// The tournament format is fixed and opinionated.
const (
	teamSize      = 2 // pairs (leftover odd person makes one team of 3)
	teamsPerGroup = 6
)

// two of every station, so up to six games run at once
var groupGames = []string{"Darts", "Bocce", "Cornhole"}

var gameCapacity = map[string]int{
	"Darts":    2,
	"Bocce":    2,
	"Cornhole": 2,
}

func (s *tournamentService) GetTournaments(ctx context.Context) ([]model.Tournament, error) {
	log := util.LoggerFromContext(ctx)

	tournaments, err := s.repo.GetAll(ctx)
	if err != nil {
		log.Error("failed to get tournaments", "error", err)
		return nil, err
	}

	log.Info("retrieved tournaments", "count", len(tournaments))
	return tournaments, nil
}

func (s *tournamentService) GetTournament(ctx context.Context, id uuid.UUID) (*model.Tournament, error) {
	log := util.LoggerFromContext(ctx)

	tournament, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrTournamentNotFound
		}
		log.Error("failed to get tournament", "tournament_id", id, "error", err)
		return nil, err
	}

	log.Info("retrieved tournament", "tournament_id", id)
	return tournament, nil
}

func authorizeTournament(t *model.Tournament, user string, isAdmin bool) error {
	if isAdmin || t.CreatedBy == user {
		return nil
	}
	return errs.ErrUnauthorized
}

func (s *tournamentService) CreateTournament(ctx context.Context, req *model.CreateTournamentRequest, user string, isAdmin bool) (*model.Tournament, error) {
	log := util.LoggerFromContext(ctx)

	// only olympics admins may start the games
	if !isAdmin {
		log.Warn("non-admin attempted to create tournament", "user", user)
		return nil, errs.ErrAdminRequired
	}

	// enforce a single tournament at a time (a finished one may be superseded)
	active, err := s.repo.CountActive(ctx)
	if err != nil {
		log.Error("failed to count active tournaments", "error", err)
		return nil, err
	}
	if active > 0 {
		log.Warn("active tournament already exists", "user", user)
		return nil, errs.ErrActiveTournament
	}

	tournament := &model.Tournament{
		Name:      req.Name,
		Status:    model.TournamentStatusSetup,
		CreatedBy: user,
	}

	if err := s.repo.Create(ctx, tournament); err != nil {
		log.Error("failed to create tournament", "error", err)
		return nil, err
	}

	log.Info("created tournament", "tournament_id", tournament.ID, "name", tournament.Name)
	return tournament, nil
}

func (s *tournamentService) AddParticipants(ctx context.Context, id uuid.UUID, names []string, user string, isAdmin bool) (*model.Tournament, error) {
	log := util.LoggerFromContext(ctx)

	tournament, err := s.GetTournament(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := authorizeTournament(tournament, user, isAdmin); err != nil {
		return nil, err
	}

	if tournament.Status != model.TournamentStatusSetup {
		log.Warn("cannot add participants after setup", "tournament_id", id, "status", tournament.Status)
		return nil, errs.ErrInvalidStatus
	}

	participants := make([]*model.Participant, 0, len(names))
	for _, name := range names {
		participants = append(participants, &model.Participant{
			TournamentID: id,
			Name:         name,
		})
	}

	if err := s.repo.AddParticipants(ctx, participants); err != nil {
		log.Error("failed to add participants", "tournament_id", id, "error", err)
		return nil, err
	}

	log.Info("added participants", "tournament_id", id, "count", len(participants))
	return s.GetTournament(ctx, id)
}

func (s *tournamentService) GenerateTeams(ctx context.Context, id uuid.UUID, user string, isAdmin bool) ([]model.Team, error) {
	log := util.LoggerFromContext(ctx)

	tournament, err := s.GetTournament(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := authorizeTournament(tournament, user, isAdmin); err != nil {
		return nil, err
	}

	if tournament.Status != model.TournamentStatusSetup {
		log.Warn("teams already generated or tournament past setup", "tournament_id", id, "status", tournament.Status)
		return nil, errs.ErrTeamsAlreadyGenerated
	}

	participants, err := s.repo.GetParticipants(ctx, id)
	if err != nil {
		return nil, err
	}

	if len(participants) < 2 {
		return nil, errs.ErrNotEnoughParticipants
	}

	// shuffle then chunk into teams, spreading any leftover across existing teams
	if err := shuffleParticipants(participants); err != nil {
		return nil, err
	}

	teams, assigned := buildTeams(id, participants, teamSize)

	if err := s.repo.CreateTeamsAndAssign(ctx, teams, assigned); err != nil {
		log.Error("failed to persist generated teams", "tournament_id", id, "error", err)
		return nil, err
	}

	if err := s.repo.UpdateStatus(ctx, id, model.TournamentStatusTeamsGenerated); err != nil {
		return nil, err
	}

	s.broadcaster.Broadcast(id, model.NewTournamentUpdated(id, model.TournamentStatusTeamsGenerated))
	log.Info("generated teams", "tournament_id", id, "teams", len(teams))
	return s.repo.GetTeams(ctx, id)
}

func buildTeams(tournamentID uuid.UUID, participants []model.Participant, teamSize int) ([]*model.Team, []*model.Participant) {
	if teamSize < 1 {
		teamSize = 2
	}

	numTeams := len(participants) / teamSize
	if numTeams == 0 {
		numTeams = 1
	}

	teams := make([]*model.Team, numTeams)
	for i := range teams {
		teams[i] = &model.Team{
			ID:           uuid.New(),
			TournamentID: tournamentID,
			Name:         fmt.Sprintf("Team %d", i+1),
		}
	}

	// round-robin deal participants onto teams so leftovers spread evenly
	assigned := make([]*model.Participant, len(participants))
	for i := range participants {
		p := participants[i]
		teamID := teams[i%numTeams].ID
		p.TeamID = &teamID
		assigned[i] = &p
	}

	return teams, assigned
}

func shuffleParticipants(participants []model.Participant) error {
	for i := len(participants) - 1; i > 0; i-- {
		n, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}
		j := int(n.Int64())
		participants[i], participants[j] = participants[j], participants[i]
	}
	return nil
}

func (s *tournamentService) DeleteTournament(ctx context.Context, id uuid.UUID, user string, isAdmin bool) error {
	log := util.LoggerFromContext(ctx)

	tournament, err := s.GetTournament(ctx, id)
	if err != nil {
		return err
	}
	if err := authorizeTournament(tournament, user, isAdmin); err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		log.Error("failed to delete tournament", "tournament_id", id, "error", err)
		return err
	}

	s.broadcaster.Broadcast(id, model.NewTournamentDeleted(id))
	log.Info("deleted tournament", "tournament_id", id)
	return nil
}

func (s *tournamentService) UpdateTeam(ctx context.Context, id, teamID uuid.UUID, name string, user string, isAdmin bool) (*model.Tournament, error) {
	tournament, err := s.GetTournament(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := authorizeTournament(tournament, user, isAdmin); err != nil {
		return nil, err
	}

	if findTeam(tournament, teamID) == nil {
		return nil, errs.ErrTeamNotFound
	}

	if err := s.repo.UpdateTeamName(ctx, teamID, name); err != nil {
		return nil, err
	}

	s.broadcaster.Broadcast(id, model.NewTournamentUpdated(id, tournament.Status))
	return s.GetTournament(ctx, id)
}

func (s *tournamentService) SwapPlayers(ctx context.Context, id, participantAID, participantBID uuid.UUID, user string, isAdmin bool) (*model.Tournament, error) {
	tournament, err := s.GetTournament(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := authorizeTournament(tournament, user, isAdmin); err != nil {
		return nil, err
	}

	// swapping only rearranges people within existing teams, so results stay valid
	if tournament.Status == model.TournamentStatusSetup || tournament.Status == model.TournamentStatusFinished {
		return nil, errs.ErrInvalidStatus
	}

	a := findParticipant(tournament, participantAID)
	b := findParticipant(tournament, participantBID)
	if a == nil || b == nil {
		return nil, errs.ErrParticipantNotFound
	}
	if a.TeamID == nil || b.TeamID == nil || *a.TeamID == *b.TeamID {
		return nil, errs.ErrInvalidSwap
	}

	if err := s.repo.SwapPlayers(ctx, a.ID, *b.TeamID, b.ID, *a.TeamID); err != nil {
		return nil, err
	}

	s.broadcaster.Broadcast(id, model.NewTournamentUpdated(id, tournament.Status))
	return s.GetTournament(ctx, id)
}

func findParticipant(t *model.Tournament, participantID uuid.UUID) *model.Participant {
	for i := range t.Participants {
		if t.Participants[i].ID == participantID {
			return &t.Participants[i]
		}
	}
	// participants assigned to teams are loaded under Teams once generated
	for i := range t.Teams {
		for j := range t.Teams[i].Members {
			if t.Teams[i].Members[j].ID == participantID {
				return &t.Teams[i].Members[j]
			}
		}
	}
	return nil
}

func findTeam(t *model.Tournament, teamID uuid.UUID) *model.Team {
	for i := range t.Teams {
		if t.Teams[i].ID == teamID {
			return &t.Teams[i]
		}
	}
	return nil
}
