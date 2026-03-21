# Casual Talker - AI English Conversation Trainer

## Project Overview
English conversation training web app for Japanese beginners. AI-powered speaking sessions with voice input/output, pronunciation interpretation, and post-session feedback.

## Tech Stack

### Frontend (frontend/)
- Vite + React 19 + React Router v7 + TypeScript
- Tailwind CSS v4 + shadcn/ui (Radix UI based)
- Zustand (state management) + Framer Motion (animation)
- Static build served by Go backend (same origin, no CORS)

### Backend (backend/)
- Go 1.23+ / Chi v5 (HTTP router)
- pgx v5 (PostgreSQL driver, direct queries — sqlc is configured but repository layer uses pgx directly)
- goose (migrations)
- sashabaranov/go-openai (GPT-4o-mini, Whisper, TTS)
- Self-managed auth: bcrypt (cost=12) + JWT HS256 + email whitelist

### Infrastructure
- Single FreeBSD rental server: nginx (TLS) + Go binary + PostgreSQL 16
- Deploy: GitHub Actions → rsync/scp (planned)

## Implemented Features (as of 2026-03)

### Authentication
- register / login / refresh / logout
- Email whitelist enforcement
- bcrypt cost=12, JWT access token (15min) + refresh token (7d, stored as hash in DB)
- JWT validation enforces `type` claim ("access" / "refresh") to prevent token confusion

### Conversation
- Theme selection (8 themes, daily conversation course)
- AI conversation sessions via SSE streaming (GPT-4o-mini)
- Dynamic turn count: 6–20 turns based on user level (level × 2 + 4, capped at 20)
- Voice input: microphone recording → Whisper STT
- AI voice playback: OpenAI TTS
- Text input fallback when voice unavailable
- Pronunciation interpretation: auto-corrects Japanese-speaker errors (L/R confusion, etc.) displayed in chat bubble

### Session Support
- Stuck rescue: 3-stage hint display
- Session completion: auto-generates feedback via GPT-4o-mini

### Feedback & History
- Post-session feedback: achievements / natural expressions / improvement points / review phrases
- Pronunciation practice button: TTS playback of error phrases from feedback screen
- Conversation log: collapsible on feedback screen
- Level display + advice for next level
- Learning history screen

### Onboarding
- 3-step onboarding flow with microphone permission acquisition

### Security (applied)
- Rate limiting enabled (middleware/ratelimit.go) for auth and speech endpoints
- Path traversal prevention on static file serving
- Password length limit (max 72 bytes for bcrypt safety)
- JWT `type` claim validation (prevents refresh token → access token abuse)
- Minimum JWT_SECRET length validation at startup (32 bytes)

## Directory Structure
```
casual-talker/
├── CLAUDE.md
├── README.md
├── docs/                  # PRD, plans, CEO reviews
├── frontend/
│   ├── package.json
│   ├── vite.config.ts
│   ├── index.html
│   └── src/
│       ├── main.tsx
│       ├── App.tsx
│       ├── index.css
│       ├── routes/        # Home, Login, Register, Session, Feedback, History
│       ├── components/    # auth/, chat/, common/, feedback/, layout/, onboarding/
│       ├── hooks/         # useAudioRecorder, useChat, useTTS
│       ├── lib/           # api-client.ts
│       └── stores/        # auth-store, session-store
└── backend/
    ├── cmd/server/main.go
    ├── internal/
    │   ├── config/config.go
    │   ├── handler/       # auth, chat, feedback, health, session, speech
    │   ├── middleware/     # auth.go (JWT), ratelimit.go (rate limiting, ENABLED)
    │   ├── service/       # auth.go
    │   ├── repository/    # auth_repo.go, session_repo.go
    │   ├── openai/        # client.go, prompts.go
    │   └── domain/        # user.go, session.go
    ├── db/
    │   ├── migrations/    # 001–007
    │   └── queries/       # auth.sql, sessions.sql, themes.sql, progress.sql
    ├── deploy/config.env.example
    ├── sqlc.yaml
    ├── Makefile
    ├── go.mod
    └── go.sum
```

## Database Migrations

| # | File | Content |
|---|------|---------|
| 001 | create_users.sql | users, allowed_emails, refresh_tokens |
| 002 | create_sessions.sql | courses, themes, sessions, turns, feedbacks, phrase_progress |
| 003 | seed_themes.sql | 8 theme records for daily conversation course |
| 004 | add_level_feedback.sql | level_advice column on feedbacks |
| 005 | add_max_turns.sql | max_turns column on sessions |
| 006 | add_interpreted_text.sql | interpreted_text column on turns (pronunciation correction) |
| 007 | add_token_hash_index.sql | index on refresh_tokens(token_hash) for lookup performance |

## Development

### Backend
```sh
cd backend
cp deploy/config.env.example config.env  # Edit with your values
make migrate-up   # Run DB migrations (001–007)
make dev          # Run server (go run ./cmd/server/)
make sqlc         # Regenerate DB code from SQL queries
make test         # Run tests
make build        # Cross-compile for FreeBSD amd64
make lint         # golangci-lint
```

### Frontend
```sh
cd frontend
npm install
npm run dev       # Vite dev server (:5173) with API proxy to :8080
npm run build     # Production build to dist/
```

### Local Setup
1. PostgreSQL running locally (`brew services start postgresql@16`)
2. Create DB: `createdb casualtalker_dev`
3. Configure: `cd backend && cp deploy/config.env.example config.env` (edit values)
4. Migrate: `make migrate-up`
5. Start backend: `make dev`
6. Start frontend: `cd frontend && npm run dev`
7. Vite proxies `/api` requests to Go backend at localhost:8080

## API Routes

```
# Auth (no JWT required)
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
POST /api/v1/auth/logout

# Health
GET  /api/v1/health

# All routes below require JWT Bearer Token

GET  /api/v1/users/me
GET  /api/v1/courses
GET  /api/v1/courses/:id/themes
GET  /api/v1/themes/:id
POST /api/v1/sessions
GET  /api/v1/sessions
GET  /api/v1/sessions/:id
PUT  /api/v1/sessions/:id/complete
GET  /api/v1/sessions/:id/turns
GET  /api/v1/sessions/:id/feedback
POST /api/v1/speech/stt
POST /api/v1/speech/tts
POST /api/v1/chat/stream
POST /api/v1/chat/hint
POST /api/v1/chat/interpret
POST /api/v1/feedback/generate
```

## Conventions
- Code comments in English
- UI text in Japanese (English only for AI conversation content)
- Direct SQL queries in backend/db/queries/ (sqlc config exists but repository uses pgx directly)
- Migrations in backend/db/migrations/ (goose format)
- API: RESTful, all under /api/v1/
- Auth: JWT Bearer token in Authorization header
- Rate limiting is active for sensitive endpoints (auth, speech)
