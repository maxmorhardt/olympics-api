# Olympics API Contribution Guide

This guide provides context for coding agents working in this repository. Olympics API is the backend for a family "olympics": a one-off, opinionated yard-games tournament. It takes a list of participants, randomly makes teams of 2, runs a round-robin group stage across three games (Darts, Bocce, Cornhole), then seeds a single-elimination playoff bracket. It is a Go + Gin + GORM service backed by PostgreSQL, fronted by OIDC auth, with an in-process WebSocket hub for real-time updates. It ships as a Docker image and is deployed via the sibling `charts/olympics-api` Helm chart.

The frontend is the sibling `olympics` workspace. Much of the architecture mirrors `squares-api`; when in doubt, look there for the canonical pattern, but this service is intentionally simpler (no tests, no metrics, no swagger, no NATS).

## The tournament format is fixed and opinionated

These are hardcoded constants in `internal/service`, not configurable per tournament:

- Teams of **2** (an odd leftover person makes one team of 3, never a team of 1).
- **6 teams per group**; groups are filled as evenly as possible.
- **Everyone advances** to the playoffs; seeding is purely by record (wins), equal-win ties broken randomly.
- Round-robin group stage: each team plays every other in its group **once**.
- Three games: **Darts, Bocce, Cornhole**, with **2 stations each** (six games can run at once). The scheduler packs each group's round-robin into equipment-aware rounds so the whole field plays simultaneously and each team's games stay balanced.

If a requirement seems to conflict with these, the constants in `tournament_service.go` / `groups.go` are the source of truth.

## Directory overview

- `cmd/main.go` – process entrypoint. Loads env, builds the bootstrap, starts the server with graceful shutdown.
- `internal/`
  - `bootstrap/server.go` – composition root. The only place concrete handler/service/repository structs and the WS hub are constructed and wired; routes are registered here.
  - `config/` – env loading (`env.go`), DB init (`db.go`), OIDC verifier (`oidc.go`), and schema migrations (`migrate.go` + embedded `migrations/*.sql`, run at startup, advisory-locked).
  - `errs/` – sentinel errors mapped to HTTP status codes in `handler/response.go`.
  - `handler/` – Gin handlers, one file per resource. Each defines its own interface. `response.go` holds the shared error→status mapping (`respondServiceError`) and the `actor(c)` helper (returns the authenticated username + isAdmin).
  - `middleware/` – `auth.go` (OIDC bearer verification), `cors.go`, `logger.go`.
  - `model/` – GORM entities (`tournament.go`, `participant.go`, `team.go`, `group.go`, `match.go`), request/response DTOs, `auth.go` (claims + `olympics-admin` group), `ws.go` (WS message types), `key.go` (context keys), `error.go` (`APIError`).
  - `repository/` – GORM data access; `tournament_repository.go` (tournament/participant/team/group) and `match_repository.go`. Each exposes an interface.
  - `routes/` – Gin route registration grouped by resource.
  - `service/` – business logic. `tournament_service.go` (create/get/delete, team generation, edits), `groups.go` (group + schedule generation), `playoffs.go` (bracket seeding/build), `standings.go` (computed standings), `match_service.go` (record results + bracket advancement), `ws_service.go` (the in-memory hub + `Broadcaster` interface).
  - `util/` – context/logger helpers, error capitalization.
- `Dockerfile` – multi-stage; the build stage cross-compiles the binary with `make build` for the target arch, the alpine runtime stage copies it in. `docker buildx` produces the `linux/amd64,linux/arm64` manifest.

## Tooling

- Language: **Go 1.26.x** (see [go.mod](go.mod)).
- **Prefer `make <target>`.** `make run`, `make build`, `make vet`, `make verify` (`go mod verify`), `make lint` (golangci-lint, config in [.golangci.yml](.golangci.yml)), `make tidy`.
- Migrations: **golang-migrate** SQL files in `internal/config/migrations/`, embedded via `go:embed` and applied on startup. Tracking uses a dedicated `olympics_schema_migrations` table (see `migrate.go`) so it never collides with another service sharing the database. Create a pair with `make migrate-create NAME=add_foo`; `make migrate-up`/`migrate-down` (set `DATABASE_URL`) for manual ops. **Change a model and add a migration** — models do not drive the schema.
- **No tests, no metrics/Prometheus, no swagger, no NATS** — this is a small single-replica app. Do not add these unless explicitly asked.
- CI runs `make verify`, `make vet`, `golangci-lint`, `make build`, then builds/pushes the Docker image; keep all of those green.

## Architecture

Strict **handler → service → repository** layering:

- **Handlers** parse requests, read identity via `actor(c)`, call exactly one service method, and translate domain errors into HTTP status via `respondServiceError`. They never touch the DB.
- **Services** own business logic and orchestration, depend on repository interfaces, and broadcast real-time events through the `service.Broadcaster` interface (the WS hub). Each service declares its interface alongside its implementation.
- **Repositories** wrap GORM, take/return models and a `context.Context`, and never know about HTTP.
- **Bootstrap** wires everything, including the `WebSocketService` (which implements `Broadcaster`).

## Authentication & authorization

- OIDC config in `config/oidc.go`; `middleware.AuthMiddleware` validates the bearer token and stores `model.Claims` + username under context keys (`model/key.go`).
- **Reads are public** (a shareable link is handed out); **all writes require auth**. Only an **olympics admin** (JWT group `olympics-admin`) may create a tournament. Every other mutation requires the tournament **creator or an admin** — enforced by `authorizeTournament` in the service layer.
- In handlers, get identity with `actor(c)` → `(user string, isAdmin bool)` and pass them down as plain values; services never depend on `*gin.Context`.

## Real-time (WebSockets, single replica, no NATS)

- `internal/service/ws_service.go` is an **in-process hub** keyed by tournament ID (`map[uuid]map[*client]bool` + mutex, per-client send channel, ping/pong). It exposes `Register(tournamentID, conn)` (blocks for the connection lifetime) and `Broadcast(tournamentID, msg)`, and satisfies `Broadcaster`.
- `handler/ws_handler.go` upgrades `GET /ws/tournaments/:id` (public, no auth) and calls `Register`.
- Domain services call `Broadcast` after mutations. Message types (`model/ws.go`): `tournament_updated` (lifecycle/status + team/participant edits → clients reload and advance stages), `score_recorded` (drives the score popup), `tournament_deleted`.

## The group-stage scheduler (`internal/service/groups.go`)

This is the trickiest code. `scheduleGroupMatches` builds a round-robin **1-factorization** per group via the circle method (`roundRobinMatchings`), then aligns groups so every group plays its nth round in the same global round. `pickGame` assigns a game to each match respecting per-round equipment capacity (`gameCapacity`) and balancing each team's game counts. A 6-team group is 5 rounds of 3 matches; two groups fill all six stations so all 12 teams play each round. If you change team/group sizes or equipment, re-check this function.

## Code style

- Lowercase, single-package files; one logical resource per file.
- Define dependencies as **interfaces**; constructors are `NewXxx(...)` returning the interface.
- `context.Context` is the first parameter of anything crossing a layer.
- Pull a `*slog.Logger` from context with `util.LoggerFromContext(ctx)` / `util.LoggerFromGinContext(c)`.
- Return sentinel errors from `internal/errs`; wrap lower-level errors with `fmt.Errorf("...: %w", err)` only when adding context. Services translate `gorm.ErrRecordNotFound` to a domain error.
- Avoid comments unless genuinely non-obvious. No `any`/`interface{}` for domain data.

## Deployment

- Image built from the [Dockerfile](Dockerfile) and pushed by CI on tag push. Runtime config via env (see `.env.example`): DB connection + `OIDC_CLIENT_ID` (`OIDC_ISSUER` defaults to the olympics Authentik app), supplied in-cluster via the `olympics-api-env` secret.
- Single replica (`charts/olympics-api`, namespace `apps`). No HPA/PDB/metrics. Don't change the chart from this repo unless asked — coordinate via the `charts` workspace.

## Commit conventions

Conventional commits. Types: `feat`, `fix`, `refactor`, `chore`, `ci`, `docs`, `style`. Scopes (optional): `handler`, `service`, `repository`, `routes`, `middleware`, `model`, `bootstrap`, `ws`, `auth`, `build`, `deploy`.

Always run `make vet`, `make lint`, and `go build ./...` before committing.
