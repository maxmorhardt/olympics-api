CREATE TABLE IF NOT EXISTS tournaments (
    id uuid PRIMARY KEY,
    name text NOT NULL,
    status text NOT NULL DEFAULT 'setup',
    team_size integer NOT NULL DEFAULT 2,
    teams_per_group integer NOT NULL DEFAULT 4,
    advance_per_group integer NOT NULL DEFAULT 2,
    game_types jsonb,
    created_by text,
    created_at timestamptz,
    updated_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_tournaments_created_by ON tournaments (created_by);

CREATE TABLE IF NOT EXISTS groups (
    id uuid PRIMARY KEY,
    tournament_id uuid NOT NULL REFERENCES tournaments (id) ON DELETE CASCADE,
    name text NOT NULL,
    created_at timestamptz,
    updated_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_groups_tournament_id ON groups (tournament_id);

CREATE TABLE IF NOT EXISTS teams (
    id uuid PRIMARY KEY,
    tournament_id uuid NOT NULL REFERENCES tournaments (id) ON DELETE CASCADE,
    group_id uuid REFERENCES groups (id) ON DELETE SET NULL,
    name text NOT NULL,
    seed integer NOT NULL DEFAULT 0,
    created_at timestamptz,
    updated_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_teams_tournament_id ON teams (tournament_id);
CREATE INDEX IF NOT EXISTS idx_teams_group_id ON teams (group_id);

CREATE TABLE IF NOT EXISTS participants (
    id uuid PRIMARY KEY,
    tournament_id uuid NOT NULL REFERENCES tournaments (id) ON DELETE CASCADE,
    team_id uuid REFERENCES teams (id) ON DELETE SET NULL,
    name text NOT NULL,
    created_at timestamptz,
    updated_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_participants_tournament_id ON participants (tournament_id);
CREATE INDEX IF NOT EXISTS idx_participants_team_id ON participants (team_id);

CREATE TABLE IF NOT EXISTS matches (
    id uuid PRIMARY KEY,
    tournament_id uuid NOT NULL REFERENCES tournaments (id) ON DELETE CASCADE,
    group_id uuid REFERENCES groups (id) ON DELETE CASCADE,
    stage text NOT NULL,
    round integer NOT NULL DEFAULT 0,
    match_number integer NOT NULL DEFAULT 0,
    game_type text,
    team_a_id uuid REFERENCES teams (id) ON DELETE SET NULL,
    team_b_id uuid REFERENCES teams (id) ON DELETE SET NULL,
    team_a_score integer NOT NULL DEFAULT 0,
    team_b_score integer NOT NULL DEFAULT 0,
    winner_team_id uuid REFERENCES teams (id) ON DELETE SET NULL,
    status text NOT NULL DEFAULT 'pending',
    next_match_id uuid REFERENCES matches (id) ON DELETE SET NULL,
    next_slot text,
    created_at timestamptz,
    updated_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_matches_tournament_id ON matches (tournament_id);
CREATE INDEX IF NOT EXISTS idx_matches_group_id ON matches (group_id);
