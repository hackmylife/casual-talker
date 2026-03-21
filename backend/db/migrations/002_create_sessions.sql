-- +goose Up
CREATE TABLE courses (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title       TEXT NOT NULL,
    description TEXT,
    sort_order  INTEGER DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE themes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id       UUID NOT NULL REFERENCES courses(id),
    title           TEXT NOT NULL,
    description     TEXT,
    target_phrases  JSONB DEFAULT '[]',
    base_vocabulary JSONB DEFAULT '[]',
    difficulty_min  INTEGER DEFAULT 1,
    difficulty_max  INTEGER DEFAULT 5,
    sort_order      INTEGER DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    theme_id    UUID NOT NULL REFERENCES themes(id),
    difficulty  INTEGER NOT NULL CHECK (difficulty BETWEEN 1 AND 5),
    status      TEXT NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active', 'completed', 'abandoned')),
    started_at  TIMESTAMPTZ DEFAULT now(),
    ended_at    TIMESTAMPTZ,
    turn_count  INTEGER DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_status ON sessions(status);

CREATE TABLE turns (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id     UUID NOT NULL REFERENCES sessions(id),
    turn_number    INTEGER NOT NULL,
    ai_text        TEXT NOT NULL,
    ai_audio_url   TEXT,
    user_text      TEXT,
    user_audio_url TEXT,
    hint_used      BOOLEAN DEFAULT FALSE,
    repeat_used    BOOLEAN DEFAULT FALSE,
    ja_help_used   BOOLEAN DEFAULT FALSE,
    example_used   BOOLEAN DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_turns_session_id ON turns(session_id);

CREATE TABLE feedbacks (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id           UUID UNIQUE NOT NULL REFERENCES sessions(id),
    achievements         JSONB DEFAULT '[]',
    natural_expressions  JSONB DEFAULT '[]',
    improvements         JSONB DEFAULT '[]',
    review_phrases       JSONB DEFAULT '[]',
    raw_llm_response     TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE phrase_progress (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    phrase          TEXT NOT NULL,
    translation_ja  TEXT,
    times_used      INTEGER DEFAULT 0,
    times_struggled INTEGER DEFAULT 0,
    last_used_at    TIMESTAMPTZ,
    mastery_level   INTEGER DEFAULT 0 CHECK (mastery_level BETWEEN 0 AND 3),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, phrase)
);
CREATE INDEX idx_phrase_progress_user_id ON phrase_progress(user_id);

-- +goose Down
DROP TABLE IF EXISTS phrase_progress;
DROP TABLE IF EXISTS feedbacks;
DROP TABLE IF EXISTS turns;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS themes;
DROP TABLE IF EXISTS courses;
