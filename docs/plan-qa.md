# QAテスト戦略・品質保証計画: AI English Conversation Trainer (MVP)

**更新履歴**
- v1.1 (2026-03): MVP実装後のセキュリティレビュー結果と修正済み問題を追記。

## 1. テスト戦略の全体像

### テストピラミッド構成（MVPスリム版）

| レイヤー | 比率 | 対象 |
|---|---|---|
| **単体テスト** | 60% | ビジネスロジック層のみ（プロンプト構築、難易度計算、救済タイミング判定） |
| **結合テスト** | 30% | APIルートのリクエスト/レスポンス形式検証 |
| **E2Eテスト** | 5% | ハッピーパス1本のみ |
| **手動テスト** | 5% | 音声品質、AI人格、iOS Safari |

> 注: 正式リリース前に拡張テスト計画（UIコンポーネント単体テスト、k6負荷テスト、Lighthouse CI等）を段階的に追加する

---

## 2. テストフレームワーク・ツール選定

### フロントエンド

| 用途 | ツール | 選定理由 |
|---|---|---|
| 単体テスト | **Vitest** | Vite親和性、Jest互換、高速 |
| コンポーネントテスト | **Testing Library** | ユーザー視点テスト |
| E2Eテスト | **Playwright** | クロスブラウザ、モバイルエミュレーション |

### バックエンド (Go)

| 用途 | ツール | 選定理由 |
|---|---|---|
| 単体/結合テスト | **Go標準 testing パッケージ** | 外部依存不要 |
| テストヘルパー | **testify** | アサーション、モック |
| APIテスト | **net/http/httptest** | Go標準のHTTPテスト |
| モック | **gomock** or **testify/mock** | インターフェースベースモック |

### 共通

| 用途 | ツール |
|---|---|
| API モック（フロント用） | **MSW (Mock Service Worker)** |
| CI | **GitHub Actions** |
| セキュリティ静的解析 | **npm audit** (frontend), **govulncheck** (backend) |

---

## 3. 音声機能のテスト方法

### 3.1 STT/TTS のテスト戦略

**Go バックエンド側:**
- OpenAI クライアントをインターフェース化し、テスト時はスタブに差し替え
- Whisper API / TTS API のレスポンス形式を固定JSONで返すモック実装

```go
// テストケース例
func TestSTTHandler(t *testing.T) {
    // マルチパートで音声ファイルをPOST → テキスト応答の検証
}
func TestSTTHandler_InvalidAudioFormat(t *testing.T) {
    // 不正な音声形式 → 400エラーの検証
}
func TestTTSHandler(t *testing.T) {
    // テキストPOST → audio/mpegストリームの検証
}
```

**フロントエンド側:**
- `useAudioRecorder` hookのテスト: MediaRecorder APIをモック
- `useTTS` hookのテスト: Audio APIをモック
- テキスト入力フォールバックのテスト

### 3.2 マイク権限テスト

| テストケース | 方法 |
|---|---|
| 権限許可後に録音が開始される | Playwright (grantPermissions) |
| 権限拒否時にテキスト入力UIが表示される | Playwright (denyPermissions) |
| iOS Safari固有のマイク権限挙動 | **手動テスト（実機）** |

---

## 4. AI応答品質のテスト方法

### 4.1 Go側の構造化テスト

```go
// テストケース例
func TestSystemPrompt_Level1(t *testing.T) {
    // Lv1のプロンプトが「単語・超短文」の指示を含むか
}
func TestChatStream_ResponseFormat(t *testing.T) {
    // SSEフォーマットが正しいか
}
func TestFeedbackGeneration_Structure(t *testing.T) {
    // achievements, improvements, reviewPhrases の各フィールドが存在するか
    // improvementsが1-2個以下か
    // reviewPhrasesが最大3つか
}
```

### 4.2 プロンプト品質の手動検証

- 難易度レベルごとに最低10パターンの入力で手動テスト
- 「否定的表現を含まないか」「難しすぎないか」をチェックリストで確認
- Phase 3（AI会話コア）完了時に実施

---

## 5. パフォーマンステスト計画

### 5.1 レイテンシ目標値

| メトリクス | 目標値 | 測定方法 |
|---|---|---|
| 音声パイプライン合計 | **< 3秒** | フロントのタイムスタンプ計測 |
| STT (Whisper) | < 800ms | Go側タイマー |
| LLM first token | < 300ms | Go側タイマー |
| TTS first chunk | < 400ms | Go側タイマー |
| ネットワーク往復 | < 200ms | フロント計測 |
| ページ初期ロード (LCP) | < 2.5秒 | Lighthouse手動実行 |

### 5.2 レイテンシ計測方法

Gate 2（Day 8）で以下を提出:
- 音声パイプライン各段階のタイムスタンプログ
- 合計レイテンシの実測値（10回の平均・中央値・p95）

---

## 6. クロスブラウザ・クロスデバイステスト

### 対象マトリクス

| 優先度 | ブラウザ/デバイス | テスト方法 |
|---|---|---|
| **P0** | iOS Safari (iPhone 13以降) | **手動テスト（実機）** |
| **P0** | Android Chrome (Pixel系) | Playwright Chromium |
| **P1** | デスクトップ Chrome | Playwright Chromium |
| **P2** | デスクトップ Safari | Playwright WebKit |

### iOS Safari 手動テストチェックリスト

- [ ] マイク権限ダイアログが正しく表示される
- [ ] 録音開始/停止が正常に動作する
- [ ] 音声認識結果がテキスト表示される
- [ ] AI音声が自動再生される（ユーザージェスチャー後）
- [ ] セッション全体（5ターン）を完了できる
- [ ] 結果画面が正しく表示される
- [ ] Safe Areaが正しく適用される（ノッチ、ホームインジケータ）
- [ ] テキスト入力フォールバックが動作する

---

## 7. CI/CDパイプライン

### GitHub Actions構成

```
PR作成/更新時:
  ├── [frontend] Lint (ESLint + Prettier)       ~30秒
  ├── [frontend] 型チェック (TypeScript)          ~30秒
  ├── [frontend] 単体テスト (Vitest)             ~1分
  ├── [backend]  Lint (golangci-lint)            ~30秒
  ├── [backend]  単体/結合テスト (go test)        ~1分
  ├── [backend]  govulncheck                     ~15秒
  ├── [frontend] npm audit                       ~15秒
  └── [frontend] E2Eテスト (Playwright) ※1本のみ  ~3分

mainマージ時:
  ├── 上記全て
  ├── フロント: Vercelデプロイ
  └── バックエンド: Cloud Runデプロイ
```

### 品質ゲート（PRマージ必須条件）
- 全テストがグリーン
- govulncheck / npm audit に高重大度の脆弱性がない

---

## 8. テストカバレッジ目標

| 対象 | カバレッジ目標 |
|---|---|
| **全体** | >= 50% |
| **ビジネスロジック層 (Go service/)** | >= 70% |
| **APIハンドラ (Go handler/)** | >= 60% |
| **フロントエンド hooks** | >= 50% |

### カバレッジ除外対象
- ブラウザ音声API直接呼び出し部分
- OpenAI API直接呼び出し部分
- 設定ファイル、型定義ファイル
- sqlc生成コード

---

## 9. セキュリティテスト

### 9.1 音声データ

| リスク | テスト方法 |
|---|---|
| HTTPS強制 | E2Eテスト（HTTPリダイレクト検証） |
| 音声Blobの破棄 | フロント単体テスト |
| 不要な音声のサーバー保存 | コードレビュー |

### 9.2 ユーザーデータ

| リスク | テスト方法 |
|---|---|
| XSS（ユーザー発話テキスト） | フロント単体テスト |
| 他ユーザーデータへのアクセス | Go結合テスト（別ユーザーJWTで403確認） |
| APIレート制限 | Go結合テスト |
| SQLインジェクション | sqlcにより構造的に排除（コードレビューで確認） |

---

## 10. 手動テストチェックリスト（MVP出荷前）

### 音声体験
- [ ] AIの音声が自然に聞こえるか
- [ ] 静かな環境での音声認識精度
- [ ] イヤホン使用時の音声入出力

### AI人格
- [ ] やさしい・否定しない・急かさない・褒めるトーンか
- [ ] 初学者が萎縮しないフィードバックか

### UX
- [ ] オンボーディングで操作方法が理解できるか
- [ ] セッション全体の体験がスムーズか
- [ ] 詰まり救済機能の存在に気づけるか

---

## 11. テスト実装の優先順位

| Phase | 対象 |
|-------|------|
| Phase 0 (Day 1-2) | Vitest/Playwright/Go testセットアップ、MSWセットアップ、CI基盤 |
| Phase 2-3 (Day 4-8) | Go: ビジネスロジック単体テスト、API結合テスト |
| Phase 5 (Day 9-14) | フロント: hooks単体テスト |
| Phase 6 (Day 15-18) | E2Eハッピーパス1本、iOS Safari手動テスト、セキュリティテスト |

---

---

## 12. セキュリティレビュー結果（MVP実装後、2026-03実施）

MVP実装完了後にセキュリティレビューを実施し、以下の問題を発見・修正した。

### 発見・修正した問題

| 重大度 | 問題 | 修正内容 | ファイル |
|--------|------|---------|---------|
| 高 | レート制限が実装されていたが有効化されていなかった | 認証・音声エンドポイントにレート制限ミドルウェアを適用 | `middleware/ratelimit.go`, `cmd/server/main.go` |
| 高 | 静的ファイルサーバーでパストラバーサルが可能 | `filepath.Clean` と先頭`.`チェックで防止 | `cmd/server/main.go` |
| 中 | パスワード長の上限なし（bcryptは72バイト超を切り捨て） | 入力バリデーションで最大72バイトを強制 | `handler/auth.go` |
| 中 | JWTの`type`クレームを検証していない | アクセストークンとリフレッシュトークンの混用を防ぐ検証を追加 | `middleware/auth.go`, `service/auth.go` |
| 中 | JWT_SECRETの最低長チェックなし | 起動時に32バイト未満のシークレットを検出してエラー終了 | `config/config.go` |
| 低 | refresh_tokens(token_hash)にインデックスなし | インデックスを追加（migration 007） | `db/migrations/007_add_token_hash_index.sql` |

### 確認して問題なかった点

- SQLインジェクション: pgxのプレースホルダーバインドを使用しており構造的に排除
- JWTのアルゴリズム混同攻撃: HS256固定指定で防止済み
- bcryptのcostが低い: cost=12で設定済み（標準的な値）
- refresh tokenのハッシュ保存: SHA-256ハッシュでDB保存済み

---

## 正式リリース前 拡張テスト計画（MVP後に段階的追加）

以下はMVP後、正式リリースに向けて追加するテスト:

- UIコンポーネント単体テスト（Testing Library）
- k6負荷テスト（同時接続10/50/100）
- Lighthouse CI（PRごとのスコア計測）
- ビジュアルリグレッション（Playwrightスクリーンショット）
- LLM応答品質バッチテスト（週次）
- カバレッジ目標の引き上げ（全体70%、ビジネスロジック85%）
