# Principalエンジニア 技術アーキテクチャ設計書 v3: AI English Conversation Trainer (MVP)

> FreeBSDレンタルサーバー単一構成。Vite + React + Go。自前認証 + ホワイトリスト。

**更新履歴**
- v3.1 (2026-03): MVP実装完了に伴い、実装状態を反映して更新（セキュリティ修正、発音解釈機能、動的ターン数、レベル表示、マイグレーション007、レート制限有効化）

---

## 1. 技術スタック

### フロントエンド

| カテゴリ | 技術 | 備考 |
|---|---|---|
| ビルドツール | **Vite** | 高速ビルド、HMR |
| フレームワーク | **React 19** | SPA |
| ルーティング | **React Router v7** | SPAルーティング |
| 言語 | **TypeScript** | |
| スタイル | **Tailwind CSS v4** + **shadcn/ui** | Radix UIベース |
| 状態管理 | **Zustand** | セッション状態 + 認証状態 |
| アニメーション | **Framer Motion** | 録音波形、ページ遷移 |
| API通信 | **fetch**（同一オリジン、相対パス） | CORS不要 |
| 配信 | **Goサーバーから静的配信** | Viteビルド成果物 |

### バックエンド

| カテゴリ | 技術 | 選定理由 |
|---|---|---|
| 言語 | **Go 1.23+** | セキュリティ、パフォーマンス |
| HTTPルーター | **Chi v5** | net/http互換、軽量 |
| クエリ | **pgx v5**（直接クエリ）/ sqlc設定あり | pgxで直接実行 |
| マイグレーション | **goose** | SQLベース |
| DB接続 | **pgx v5** | PostgreSQL専用、高性能 |
| OpenAI連携 | **sashabaranov/go-openai** | Chat/Whisper/TTS全対応 |
| 認証 | **golang.org/x/crypto/bcrypt** + **golang-jwt/jwt v5** | 自前認証 |
| バリデーション | **go-playground/validator v10** | |
| ログ | **log/slog** (標準) | 構造化ログ |
| 設定 | **caarlos0/env** | 環境変数バインド |

### インフラ

| カテゴリ | 技術 | 備考 |
|---|---|---|
| サーバー | **FreeBSD レンタルサーバー** | Docker不可 |
| リバースプロキシ | **nginx** | TLS終端、gzip |
| DB | **PostgreSQL 16** | ローカル |
| TLS | **Let's Encrypt (certbot)** | 自動更新 |
| プロセス管理 | **FreeBSD rc.d** | daemon(8) |
| デプロイ | **GitHub Actions → rsync/scp** | |

---

## 2. システムアーキテクチャ図

```
┌──────────────────────────────────────────────────────────┐
│                     ユーザー（ブラウザ）                      │
└───────────────────────┬──────────────────────────────────┘
                        │ HTTPS
                        ▼
┌──────────────────────────────────────────────────────────┐
│                  FreeBSD レンタルサーバー                     │
│                                                          │
│  ┌────────────────────────────────────────────────────┐  │
│  │  nginx (リバースプロキシ)                              │  │
│  │  - TLS終端 (Let's Encrypt)                          │  │
│  │  - gzip圧縮                                         │  │
│  │  - proxy_pass → 127.0.0.1:8080                     │  │
│  └──────────────────────┬─────────────────────────────┘  │
│                         │                                │
│  ┌────────────────────────────────────────────────────┐  │
│  │  Go バイナリ (casual-talker) :8080                   │  │
│  │                                                    │  │
│  │  ┌─ 静的ファイル配信 ─────────────────────────┐    │  │
│  │  │  / → ./static/ (Vite ビルド成果物)          │    │  │
│  │  │  SPAフォールバック → index.html              │    │  │
│  │  └───────────────────────────────────────────┘    │  │
│  │                                                    │  │
│  │  ┌─ API (Chi Router) ────────────────────────┐    │  │
│  │  │  /api/v1/auth/*     (自前認証)             │    │  │
│  │  │  /api/v1/sessions/* (セッション管理)        │    │  │
│  │  │  /api/v1/speech/*   (STT/TTS)              │    │  │
│  │  │  /api/v1/chat/*     (AI会話 SSE)           │    │  │
│  │  │  /api/v1/feedback/* (フィードバック)         │    │  │
│  │  └───────────────────────────────────────────┘    │  │
│  │                                                    │  │
│  │  ┌─ 自前認証 ────────────────────────────────┐    │  │
│  │  │  bcrypt (cost=12) + JWT HS256              │    │  │
│  │  │  allowed_emails ホワイトリスト               │    │  │
│  │  │  access_token (15min) + refresh_token (7d) │    │  │
│  │  └───────────────────────────────────────────┘    │  │
│  └──────────────────────┬─────────────────────────────┘  │
│                         │ pgx                            │
│  ┌──────────────────────┴─────────────────────────────┐  │
│  │  PostgreSQL 16  :5432 (localhost only)              │  │
│  └────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────┘
                        │ HTTPS
                        ▼
              ┌──────────────────┐
              │  OpenAI API       │
              │  - Whisper (STT)  │
              │  - GPT-4o-mini    │
              │  - TTS            │
              └──────────────────┘
```

---

## 3. 音声パイプライン

### レイテンシバジェット

| 段階 | 目標 |
|------|------|
| STT (Whisper) | ~800ms |
| LLM first token (GPT-4o-mini) | ~300ms |
| TTS first chunk (tts-1) | ~400ms |
| ネットワーク往復 | ~200ms ※Gate 1で実測確認 |
| **合計** | **~1,700ms（目標3秒以内）** |

※ レンタルサーバー → OpenAI APIのネットワークレイテンシはGate 1で計測し、~200ms前提が維持できなければバジェット再計算する。

### 遅延削減の工夫
1. TTS先行リクエスト（LLM最初の文到達時点で発行、goroutineで並列化）
2. ストリーミングテキスト表示（TTS前にテキスト先行表示）

---

## 4. API設計

```
# 認証（JWT不要）
POST   /api/v1/auth/register         # ユーザー登録（ホワイトリスト検証）
POST   /api/v1/auth/login            # ログイン
POST   /api/v1/auth/refresh          # トークンリフレッシュ
POST   /api/v1/auth/logout           # ログアウト（refresh token revoke）

# ヘルスチェック（認証不要）
GET    /api/v1/health

# 以下は全て JWT Bearer Token 必要

# ユーザー
GET    /api/v1/users/me              # ユーザー情報

# コース・テーマ
GET    /api/v1/courses               # コース一覧
GET    /api/v1/courses/:id/themes    # テーマ一覧
GET    /api/v1/themes/:id            # テーマ詳細

# セッション
POST   /api/v1/sessions              # セッション作成
GET    /api/v1/sessions              # セッション一覧
GET    /api/v1/sessions/:id          # セッション詳細
PUT    /api/v1/sessions/:id/complete # セッション完了（+フィードバック自動生成）
GET    /api/v1/sessions/:id/turns    # ターン一覧
GET    /api/v1/sessions/:id/feedback # フィードバック取得

# 音声
POST   /api/v1/speech/stt            # 音声→テキスト (multipart)
POST   /api/v1/speech/tts            # テキスト→音声 (audio/mpeg stream)

# AI会話
POST   /api/v1/chat/stream           # AI会話 (SSE)
POST   /api/v1/chat/hint             # ヒント取得（3段階）
POST   /api/v1/chat/interpret        # 発音解釈（L/R混同等の日本語話者特有エラー補正）

# フィードバック
POST   /api/v1/feedback/generate     # フィードバック生成
```

> **変更点（v3.1）**: `/chat/example` を廃止し `/chat/interpret`（発音解釈）を追加。
> フィードバック取得は `/feedback/:session_id` から `/sessions/:id/feedback` に変更。

---

## 5. データベーススキーマ

```sql
-- ユーザー（自前認証）
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    display_name  TEXT,
    level         INTEGER NOT NULL DEFAULT 1 CHECK (level BETWEEN 1 AND 5),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- メールホワイトリスト
CREATE TABLE allowed_emails (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT UNIQUE NOT NULL,
    invited_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- リフレッシュトークン
CREATE TABLE refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id),
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked    BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- コース
CREATE TABLE courses (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title       TEXT NOT NULL,
    description TEXT,
    sort_order  INTEGER DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- テーマ
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

-- セッション
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

-- セッション
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
    max_turns   INTEGER DEFAULT 10,  -- 動的ターン数（レベル × 2 + 4、上限20）
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ターン
CREATE TABLE turns (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id       UUID NOT NULL REFERENCES sessions(id),
    turn_number      INTEGER NOT NULL,
    ai_text          TEXT NOT NULL,
    ai_audio_url     TEXT,
    user_text        TEXT,
    interpreted_text TEXT,  -- 発音補正後テキスト（L/R混同等）
    user_audio_url   TEXT,
    hint_used        BOOLEAN DEFAULT FALSE,
    repeat_used      BOOLEAN DEFAULT FALSE,
    ja_help_used     BOOLEAN DEFAULT FALSE,
    example_used     BOOLEAN DEFAULT FALSE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- フィードバック
CREATE TABLE feedbacks (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id           UUID UNIQUE NOT NULL REFERENCES sessions(id),
    achievements         JSONB DEFAULT '[]',
    natural_expressions  JSONB DEFAULT '[]',
    improvements         JSONB DEFAULT '[]',
    review_phrases       JSONB DEFAULT '[]',
    level_advice         TEXT,  -- 次レベルへのアドバイス（migration 004で追加）
    raw_llm_response     TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- フレーズ進捗
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

-- インデックス
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_status ON sessions(status);
CREATE INDEX idx_turns_session_id ON turns(session_id);
CREATE INDEX idx_phrase_progress_user_id ON phrase_progress(user_id);
```

---

## 6. 認証設計

### フロー
```
登録: POST /api/v1/auth/register
  → allowed_emails検証 → bcrypt hash → users INSERT → JWT発行

ログイン: POST /api/v1/auth/login
  → email検索 → bcrypt比較 → JWT発行

リフレッシュ: POST /api/v1/auth/refresh
  → refresh_token検証 → 新access_token発行

ログアウト: POST /api/v1/auth/logout
  → refresh_token revoke
```

### JWT設計
- アルゴリズム: HS256
- JWT_SECRET: 環境変数管理、最低256ビット（32バイト）のランダム値
- access_token: 有効期限15分、claims: sub(userID), email, exp, iat, type("access")
- refresh_token: 有効期限7日、claims: sub(userID), exp, iat, type("refresh")
- refresh_tokenはDBにハッシュ保存（revoke対応）

### パスワードリセット
- MVPでは省略。管理者がDB直接操作で対応

---

## 7. ディレクトリ構成（実装済み）

```
casual-talker/
├── CLAUDE.md
├── README.md
├── docs/
├── frontend/
│   ├── package.json
│   ├── vite.config.ts
│   ├── index.html
│   └── src/
│       ├── main.tsx
│       ├── App.tsx                 # React Router ルート定義
│       ├── index.css
│       ├── routes/
│       │   ├── Home.tsx
│       │   ├── Login.tsx
│       │   ├── Register.tsx
│       │   ├── Session.tsx
│       │   ├── Feedback.tsx
│       │   └── History.tsx
│       ├── components/
│       │   ├── auth/AuthGuard.tsx
│       │   ├── chat/               # ChatBubble, VoiceInputButton, RescuePanel, HintModal, TypingIndicator
│       │   ├── common/LoadingSpinner.tsx
│       │   ├── feedback/ExpressionCard.tsx
│       │   ├── layout/             # AppShell, BottomNav, Header
│       │   └── onboarding/OnboardingFlow.tsx
│       ├── hooks/
│       │   ├── useChat.ts          # SSEストリーミング
│       │   ├── useAudioRecorder.ts
│       │   └── useTTS.ts
│       ├── lib/
│       │   └── api-client.ts       # 相対パスfetch
│       └── stores/
│           ├── auth-store.ts
│           └── session-store.ts
└── backend/
    ├── cmd/server/main.go
    ├── internal/
    │   ├── config/config.go
    │   ├── handler/
    │   │   ├── auth.go             # 認証ハンドラ
    │   │   ├── chat.go             # chat/stream, chat/hint, chat/interpret
    │   │   ├── feedback.go
    │   │   ├── health.go
    │   │   ├── session.go
    │   │   └── speech.go
    │   ├── middleware/
    │   │   ├── auth.go             # JWT検証（typeクレーム検証含む）
    │   │   └── ratelimit.go        # レート制限（有効化済み）
    │   ├── service/
    │   │   └── auth.go             # 認証ロジック
    │   ├── repository/
    │   │   ├── auth_repo.go
    │   │   └── session_repo.go
    │   ├── openai/
    │   │   ├── client.go
    │   │   └── prompts.go
    │   └── domain/
    │       ├── user.go
    │       └── session.go
    ├── db/
    │   ├── migrations/             # 001〜007
    │   └── queries/                # auth.sql, sessions.sql, themes.sql, progress.sql
    ├── deploy/
    │   └── config.env.example
    ├── sqlc.yaml
    ├── Makefile
    ├── go.mod
    └── go.sum
```

---

## 8. ローカル開発環境

```sh
# macOS での開発構成

# PostgreSQL（Homebrewで直接起動）
brew install postgresql@16
brew services start postgresql@16
createdb casualtalker_dev

# バックエンド
cd backend
cp ../deploy/config.env.example .env
go run ./cmd/server/

# フロントエンド（HMR + APIプロキシ）
cd frontend
npm install
npm run dev
```

Vite devサーバーからGoバックエンドへプロキシ:
```typescript
// vite.config.ts
export default defineConfig({
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
});
```

---

## 9. サーバーリソース配分

レンタルサーバーの利用可能メモリに対する配分目安:

| プロセス | 配分 | 設定方法 |
|---------|------|---------|
| PostgreSQL | 40% | shared_buffers, work_mem |
| Go | 30% | GOMEMLIMIT環境変数 |
| OS + nginx | 30% | （残り） |

---

## 10. バックアップ方針

```sh
# deploy/backup-db.sh
#!/bin/sh
DATE=$(date +%Y%m%d_%H%M%S)
pg_dump -U casualtalker casualtalker_db | gzip > /var/backups/casualtalker/db_${DATE}.sql.gz

# 7日以上前のバックアップを削除
find /var/backups/casualtalker -name "*.sql.gz" -mtime +7 -delete
```

cron設定（日次3:00AM）:
```
0 3 * * * /usr/local/etc/casual-talker/backup-db.sh
```

外部転送はrsyncまたはscpで開発者のローカルマシンに日次同期。

---

## 11. デプロイ・ロールバック

### デプロイスクリプト (deploy/deploy.sh)
```sh
#!/bin/sh
set -e

SERVER=$1
USER=$2

# バックアップ
ssh ${USER}@${SERVER} "cp /usr/local/bin/casual-talker /usr/local/bin/casual-talker.prev"

# バイナリ転送
scp backend/casual-talker ${USER}@${SERVER}:/usr/local/bin/casual-talker.new

# フロントビルド転送
rsync -avz --delete frontend/dist/ ${USER}@${SERVER}:/usr/local/www/casual-talker/static/

# マイグレーション転送・実行
rsync -avz backend/db/migrations/ ${USER}@${SERVER}:/usr/local/www/casual-talker/migrations/
ssh ${USER}@${SERVER} "cd /usr/local/www/casual-talker && goose -dir ./migrations postgres 'postgres://casualtalker:xxx@localhost/casualtalker_db?sslmode=disable' up"

# バイナリ入れ替え・再起動
ssh ${USER}@${SERVER} "mv /usr/local/bin/casual-talker.new /usr/local/bin/casual-talker && service casualtalker restart"
```

### ロールバックスクリプト (deploy/rollback.sh)
```sh
#!/bin/sh
set -e
SERVER=$1
USER=$2
ssh ${USER}@${SERVER} "mv /usr/local/bin/casual-talker.prev /usr/local/bin/casual-talker && service casualtalker restart"
# マイグレーションのdown: goose -dir ./migrations postgres '...' down
```

---

## 12. セキュリティ

### 設計時の対策

| 項目 | 対策 |
|------|------|
| PostgreSQL | pg_hba.confでlocalhost接続のみ |
| SSH | 鍵認証のみ、パスワード認証無効化 |
| 認証 | bcrypt cost=12、JWT有効期限短い、refresh revoke対応 |
| JWT_SECRET | 最低256ビットランダム値、環境変数管理 |
| nginx | rate limiting設定 |
| Go | 専用ユーザー(www)で実行、最小権限 |
| 同一オリジン | CORS不要、CSRF対策容易 |

### 実装時に発見・修正したセキュリティ問題（v3.1）

| 問題 | 修正内容 |
|------|---------|
| レート制限が無効化されていた | `middleware/ratelimit.go` を有効化、認証・音声エンドポイントに適用済み |
| 静的ファイルサーバーのパストラバーサル | `filepath.Clean` + `.` チェックで防止 |
| パスワード長制限なし | bcrypt上限72バイトを超える入力を拒否するバリデーションを追加 |
| JWT `type` クレーム未検証 | アクセストークンとリフレッシュトークンの混用を防ぐ `type` クレーム検証を追加 |
| JWT_SECRET最低長チェックなし | 起動時に32バイト未満のシークレットを検出してエラー終了する検証を追加 |
| refresh_tokensのlookupインデックスなし | `token_hash` カラムにインデックスを追加（migration 007） |

---

## 13. 実装ステップ（21日間）

> **注（v3.1）**: MVP実装完了。以下は実施済みの記録として保持。

### Phase 0: プロジェクト基盤 (Day 1-2) ✅
- Go module初期化
- Vite + React + React Router プロジェクト初期化
- Tailwind CSS v4 + shadcn/ui セットアップ
- ローカルPostgreSQL + Go起動確認
- CI基盤（GitHub Actions: lint + test）

### Phase 1: 認証 + DB基盤 (Day 3-6) ✅
- DBマイグレーション（goose）001〜003: 全テーブル + テーマシード
- Go: 自前認証実装（register/login/refresh/logout/ホワイトリスト）
- React: ログイン/登録画面、auth-store
- React: api-client（同一オリジン、相対パス）
- STT/TTSエンドポイント疎通確認

### Phase 2: 音声パイプライン (Day 7-9) ✅
- Go: Whisper API連携、TTS API連携
- React: useAudioRecorder、useTTS
- 音声→STT→テキスト→TTS→再生の貫通テスト

### Phase 3: AI会話コア (Day 10-13) ✅
- Go: SSEストリーミング、セッション管理
- Go: システムプロンプト設計（openai/prompts.go）
- React: useChat hook、セッション画面UI
- **動的ターン数実装**: レベル × 2 + 4（上限20）

### Phase 4: フィードバック + 履歴 (Day 14-15) ✅
- Go: フィードバック生成API、学習履歴API
- React: フィードバック画面（会話ログ折りたたみ、発音練習ボタン）
- React: 履歴画面
- **レベル表示 + 次レベルへのアドバイス実装**（migration 004）

### Phase 5: ヒント・救済 + テーマ + オンボーディング (Day 16-17) ✅
- Go: ヒントAPI（3段階）、発音解釈API（/chat/interpret）
- React: 救済UI、テーマ選択、ホーム画面
- React: オンボーディングフロー（3ステップ、マイク権限取得）
- **発音解釈機能**: L/R混同等の日本語話者エラーを自動補正、バブル内表示（migration 006）

### Phase 6: 品質・セキュリティ修正 ✅
- セキュリティレビューで発見した問題を修正（詳細はplan-qa.md参照）
- migration 005: max_turnsカラム追加
- migration 007: refresh_tokens(token_hash)インデックス追加
- レート制限の有効化
- パストラバーサル防止、パスワード長制限、JWT検証強化

---

## 14. Gate

| Gate | Day | 確認事項 |
|------|-----|---------|
| Gate 1 | Day 4 | Go API起動、PostgreSQL接続、自前認証疎通、STT/TTS curl疎通、**FreeBSDバイナリ起動確認**、**OpenAI APIレイテンシ計測**、**DBバックアップcron動作** |
| **Gate 2** | **Day 9** | **CEOレビュー: 音声パイプライン一気通貫デモ + レイテンシ実測値（10回平均・中央値・p95）** |
| Gate 3 | Day 15 | フィードバック、履歴、オンボーディング、アナリティクス動作確認 |
| Gate 4 | Day 21 | FreeBSD本番動作確認、iOS Safari手動テスト、**デプロイ/ロールバックスクリプト動作確認**、E2Eハッピーパス |
