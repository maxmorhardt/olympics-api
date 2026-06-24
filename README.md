# Olympics API

Backend for a family "olympics" tournament: randomly pairs participants into teams,
runs a round-robin group stage across multiple games (cornhole, darts, etc.), then
seeds a single-elimination playoff bracket from the group standings.

Go + Gin + GORM + PostgreSQL, following the `handler → service → repository` layering
from `squares-api`.

## Tournament lifecycle

```
setup → teams_generated → group_stage → playoffs → finished
```

1. **Create** a tournament (configurable `teamSize`, `teamsPerGroup`, `advancePerGroup`, `gameTypes`).
2. **Add participants** (a list of names).
3. **Generate teams** — randomly shuffles participants into teams (default pairs; leftovers spread across teams so nobody is alone).
4. **Generate groups** — randomly splits teams into balanced groups and builds a round-robin schedule, cycling through the configured games.
5. Record results for each group match.
6. **Generate playoffs** — once every group match is played, seeds the top `advancePerGroup` teams per group into a single-elimination bracket (byes for the top seeds when the field isn't a power of two).
7. Record playoff results; winners auto-advance. Completing the final marks the tournament `finished`.

## Running locally

Copy `.env.example` to `.env` and fill in the Postgres connection, then:

```sh
make run
```

Migrations run automatically on startup (golang-migrate, embedded SQL in `internal/config/migrations`).

## Endpoints

| Method | Path | Description |
| ------ | ---- | ----------- |
| GET    | `/tournaments` | List tournaments |
| POST   | `/tournaments` | Create a tournament |
| GET    | `/tournaments/:id` | Get a tournament (participants, teams, groups) |
| POST   | `/tournaments/:id/participants` | Add participants (`{"names": [...]}`) |
| POST   | `/tournaments/:id/teams/generate` | Randomly generate teams |
| POST   | `/tournaments/:id/groups/generate` | Generate groups + round-robin matches |
| POST   | `/tournaments/:id/playoffs/generate` | Seed the playoff bracket |
| GET    | `/tournaments/:id/standings` | Group standings |
| GET    | `/tournaments/:id/bracket` | Playoff bracket |
| GET    | `/tournaments/:id/matches` | All matches |
| PATCH  | `/matches/:matchId/result` | Record a result (`{"teamAScore": x, "teamBScore": y}`) |
| GET    | `/health/live`, `/health/ready` | Health checks |
