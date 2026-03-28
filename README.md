# Casual Talker

**AI English Conversation Trainer — 初心者が安心して話せる英会話体験**

日本語話者の英会話初心者向けAI会話練習アプリ。マイクで話すだけで、AIと自然な英会話セッションができます。セッション後には自動フィードバックで学習の振り返りも可能です。

---

## 機能一覧

### 会話・音声
- **AI会話セッション** — GPT-4o-mini によるSSEストリーミング応答（レベルに応じた6〜20ターン）
- **音声入力** — マイク録音 → OpenAI Whisper STT
- **AI音声再生** — OpenAI TTS API による自然な音声
- **テキスト入力フォールバック** — 音声が使えない環境でもテキスト入力可

### 学習支援
- **テーマ選択** — 多言語（英語/イタリア語/韓国語/ポルトガル語）× 各8テーマ（計32テーマ）の日常会話コース
- **詰まり救済** — 3段階ヒント表示
- **発音解釈** — L/R混同等、日本語話者特有の発音エラーを自動補正してバブル内に表示
- **レベル表示** — 現在レベルと次レベルへのアドバイスを表示
- **多言語対応** — 英語・イタリア語・韓国語・ポルトガル語に対応。プロンプト・レベルガイドライン・発音エラーパターンが言語別に切り替わる

### フィードバック・履歴
- **セッション後フィードバック** — できたこと / 自然な表現 / 改善ポイント / 復習フレーズを自動生成
- **発音練習ボタン** — フィードバック画面で間違いフレーズをTTS再生
- **会話ログ表示** — フィードバック画面で折りたたみ式に表示
- **学習履歴画面** — 過去のセッション一覧
- **練習統計** — ストリーク（JST基準）、合計セッション数、練習時間、発話ターン数、発音修正回数、言語別セッション数をホーム画面に表示

### その他
- **自前認証** — bcrypt + JWT HS256 + メールホワイトリスト
- **オンボーディング** — 3ステップ、マイク権限取得フロー
- **モバイルファーストUI** — iPhone Safariを含むスマホ最適化

---

## 対応言語

| 言語 | フラグ | target_language 値 |
|------|--------|-------------------|
| 英語 | 🇬🇧 | `en` |
| イタリア語 | 🇮🇹 | `it` |
| 韓国語 | 🇰🇷 | `ko` |
| ポルトガル語 | 🇧🇷 | `pt` |

各言語につき8テーマ（計32テーマ）。プロンプト・レベルガイドライン・Whisper STT言語ヒント・発音エラーパターンが言語別に切り替わります。

---

## 技術スタック

### フロントエンド

| カテゴリ | 技術 |
|---|---|
| ビルドツール | Vite |
| フレームワーク | React 19 + React Router v7 |
| 言語 | TypeScript |
| スタイル | Tailwind CSS v4 + shadcn/ui (Radix UI) |
| 状態管理 | Zustand |
| アニメーション | Framer Motion |

### バックエンド

| カテゴリ | 技術 |
|---|---|
| 言語 | Go 1.23+ |
| HTTPルーター | Chi v5 |
| DBドライバー | pgx v5（直接クエリ） |
| マイグレーション | goose |
| OpenAI連携 | sashabaranov/go-openai |
| 認証 | bcrypt + golang-jwt v5 (HS256) |

### インフラ・AI

| カテゴリ | 技術 |
|---|---|
| DB | PostgreSQL 16 |
| AI会話 | OpenAI GPT-4o-mini |
| 音声認識 | OpenAI Whisper API |
| 音声合成 | OpenAI TTS API |
| サーバー | Rocky Linuxレンタルサーバー（nginx + Goバイナリ） |
| デプロイ | GitHub Actions → rsync/scp（予定） |

---

## セットアップ

### 前提条件

- Go 1.23+
- Node.js 20+
- PostgreSQL 16+
- goose（`go install github.com/pressly/goose/v3/cmd/goose@latest`）
- OpenAI APIキー

### 1. PostgreSQLのセットアップ

```sh
# macOS (Homebrew)
brew install postgresql@16
brew services start postgresql@16
createdb casualtalker_dev
```

### 2. バックエンドのセットアップ

```sh
cd backend

# 設定ファイルを作成
cp deploy/config.env.example config.env
# config.env を編集して各値を設定（後述）

# DBマイグレーション実行
make migrate-up

# 開発サーバー起動（:8080）
make dev
```

### 3. フロントエンドのセットアップ

```sh
cd frontend
npm install

# 開発サーバー起動（:5173 → /api は :8080 へプロキシ）
npm run dev
```

ブラウザで `http://localhost:5173` を開きます。

---

## 環境変数

`backend/config.env`（本番は環境変数として設定）:

| 変数名 | 説明 | 例 |
|---|---|---|
| `PORT` | サーバーポート番号 | `8080` |
| `DATABASE_URL` | PostgreSQL接続文字列 | `postgres://user:pass@localhost:5432/casualtalker_db?sslmode=disable` |
| `JWT_SECRET` | JWT署名キー（最低32バイトのランダム値） | `your-random-256bit-secret` |
| `OPENAI_API_KEY` | OpenAI APIキー | `sk-xxx` |
| `STATIC_DIR` | フロントエンドビルド成果物のディレクトリ | `./static` |

`JWT_SECRET` は必ず十分に長いランダム値を使用してください:

```sh
openssl rand -base64 32
```

---

## Makefileコマンド

```sh
cd backend

make dev          # 開発サーバー起動（go run）
make build        # Rocky Linux向けクロスコンパイル
make migrate-up   # DBマイグレーション適用
make migrate-down # DBマイグレーション1件ロールバック
make sqlc         # SQLクエリからGoコード生成
make test         # テスト実行
make lint         # golangci-lint実行
```

---

## API一覧

すべて `/api/v1/` プレフィックス。`*` はJWT Bearer Token必須。

```
# 認証（JWT不要）
POST /api/v1/auth/register      ユーザー登録（ホワイトリスト検証）
POST /api/v1/auth/login         ログイン
POST /api/v1/auth/refresh       トークンリフレッシュ
POST /api/v1/auth/logout        ログアウト

# ヘルスチェック
GET  /api/v1/health

# ユーザー *
GET  /api/v1/users/me           自分の情報
GET  /api/v1/users/me/stats     練習統計（ストリーク、セッション数、練習時間、発話ターン数、発音修正回数、言語別統計）

# コース・テーマ *
GET  /api/v1/courses            コース一覧
GET  /api/v1/courses/:id/themes テーマ一覧
GET  /api/v1/themes/:id         テーマ詳細

# セッション *
POST /api/v1/sessions                      セッション作成
GET  /api/v1/sessions                      セッション一覧
GET  /api/v1/sessions/:id                  セッション詳細
PUT  /api/v1/sessions/:id/complete         セッション完了（フィードバック自動生成）
GET  /api/v1/sessions/:id/turns            ターン一覧
GET  /api/v1/sessions/:id/feedback         フィードバック取得

# 音声 *
POST /api/v1/speech/stt         音声→テキスト（Whisper）multipart
POST /api/v1/speech/tts         テキスト→音声（TTS）audio/mpegストリーム

# AI会話 *
POST /api/v1/chat/stream        AI会話ストリーミング（SSE）
POST /api/v1/chat/hint          ヒント取得
POST /api/v1/chat/interpret     発音解釈（L/R混同等の補正）

# フィードバック *
POST /api/v1/feedback/generate  フィードバック生成
```

---

## プロジェクト構成

```
casual-talker/
├── CLAUDE.md               AIアシスタント向け開発ガイド
├── README.md               本ファイル
├── docs/
│   ├── prd.md              プロダクト要件定義書
│   ├── plan-pm.md          PM実装計画
│   ├── plan-principal.md   技術アーキテクチャ設計書
│   ├── plan-uiux.md        UI/UX設計書
│   └── plan-qa.md          QA・品質保証計画
├── frontend/
│   ├── src/
│   │   ├── routes/         ページコンポーネント（Home, Login, Session, Feedback, History）
│   │   ├── components/     UIコンポーネント（chat, feedback, layout, onboarding）
│   │   ├── hooks/          カスタムフック（useChat, useAudioRecorder, useTTS）
│   │   ├── lib/            APIクライアント
│   │   └── stores/         Zustandストア（auth, session）
│   └── public/
└── backend/
    ├── cmd/server/         エントリーポイント
    ├── internal/
    │   ├── config/         設定読み込み
    │   ├── handler/        HTTPハンドラ
    │   ├── middleware/     認証・レート制限ミドルウェア
    │   ├── service/        ビジネスロジック（auth）
    │   ├── repository/     DBアクセス層
    │   ├── openai/         OpenAIクライアント・プロンプト
    │   └── domain/         ドメイン型定義
    ├── db/
    │   ├── migrations/     gooseマイグレーション（001〜007）
    │   └── queries/        SQLクエリ定義（sqlc用）
    ├── deploy/             デプロイ設定テンプレート
    └── Makefile
```

---

## デプロイ（Rocky Linux向け概要）

### サーバー構成

```
nginx（TLS終端 / Let's Encrypt）
  ↓ proxy_pass
Go バイナリ（:8080）
  ↓
PostgreSQL 16（localhost:5432）
```

### デプロイ手順概要

```sh
# 1. バイナリのクロスコンパイル
cd backend && make build
# → casual-talker（Rocky Linux amd64バイナリ）が生成される

# 2. フロントエンドのビルド
cd frontend && npm run build
# → dist/ に成果物が生成される

# 3. サーバーへ転送
scp backend/casual-talker user@server:/usr/local/bin/
rsync -avz frontend/dist/ user@server:/usr/local/www/casual-talker/static/
rsync -avz backend/db/migrations/ user@server:/usr/local/www/casual-talker/migrations/

# 4. マイグレーション実行
ssh user@server "goose -dir /usr/local/www/casual-talker/migrations postgres '...' up"

# 5. サービス再起動
ssh user@server "service casualtalker restart"
```

プロセス管理は systemd を使用。unit ファイルは `backend/deploy/casual-talker.service` を参照してください。

---

## ライセンス

MIT
