# CEOレビュー v3: FreeBSDレンタルサーバー単一構成への移行（最終版）

## 総合判定

**APPROVED（条件付き）**

---

## 1. Next.js → Vite + React への変更

**判定: 適切。承認。**

FreeBSDでの静的配信が前提。Next.jsの利点（SSR, RSC, Vercel統合）が全て使えないため、Viteの方がシンプルでビルドが高速。shadcn/ui、Zustand、Framer Motionはそのまま使用可能。

注意: React Router v7のAPI安定性を確認すること。問題がある場合はTanStack Routerも候補。

---

## 2. FreeBSD 1台構成のリスク

**判定: MVPとして許容。**

許容理由:
- ホワイトリスト制の少数ユーザー
- レイテンシの主要因はOpenAI API
- 数秒のダウンタイムはMVPで許容

リスクと対策:

**(a) 単一障害点:** PostgreSQL/Go/nginxが1台に同居。メモリ競合の可能性あり。
→ リソース配分を事前設計（PostgreSQL 40%, Go 30%, OS+nginx 30%）

**(b) FreeBSD固有:** GoクロスコンパイルのCGO_ENABLED=0前提でbcryptが動作するか。
→ Gate 1で実際にFreeBSD上での動作を確認

**(c) バックアップ:** pg_dump日次 + 外部転送のcronを設定すること。Gate 1の完了条件に含める。

---

## 3. 自前認証 + ホワイトリストの設計

**判定: MVPとして適切。**

- bcrypt cost=12: 適切
- access_token 15分 + refresh_token 7日: 標準的
- allowed_emailsテーブルでDB管理: 運用しやすい
- refresh_tokensでrevoke対応: セキュリティ上正しい
- パスワードリセット省略: MVPで妥当

追加条件: JWT_SECRETは最低256ビット（32バイト）のランダム値。環境変数管理。設計書に明記すること。

---

## 4. 21日への見積もり増

**判定: 妥当。21日を上限とする。**

- 認証自前実装: +2日 → 妥当
- Next.js→Vite: +1日 → 妥当
- サーバー構築: +1日 → 妥当
- Supabase廃止/CORS不要: -1日 → 妥当な相殺

**22日目以降に未完了機能がある場合はスコープカットで対応。延伸は認めない。**

---

## 5. デプロイフローの安全性

**判定: 概ね妥当。改善1点。**

改善: ロールバック手順を明確にすること。
- デプロイ前のバイナリバックアップ
- マイグレーション失敗時のdown手順
- デプロイスクリプトを1本にまとめること

---

## 6. 前回条件の反映状況

| 条件 | 反映状況 |
|---|---|
| アナリティクス | Phase 5で実装 → OK |
| オンボーディング | Phase 5で実装 → OK |
| UI日本語化 | フロント実装時に遵守 → OK |
| iOS Safari手動テスト | Phase 6で実施 → OK |
| レイテンシ3秒以内 | バジェット~1,700ms → OK |
| CEOレビューゲート | Gate 2 (Day 9) → OK |

---

## 7. マイルストーン（21日間）

### Gate 1: Day 4
- Go API起動、PostgreSQL接続、自前認証疎通
- STT/TTS curlレベル疎通
- **FreeBSDでのバイナリ起動確認**
- **PostgreSQLバックアップcron動作確認**
- **OpenAI APIネットワークレイテンシ計測**

### Gate 2: Day 9（CEOレビューゲート）
- 音声パイプライン一気通貫デモ
- レイテンシ実測値（10回の平均・中央値・p95）
- 3秒超過の場合はアーキテクチャ見直し

### Gate 3: Day 15
- フィードバック、履歴、オンボーディング動作確認
- アナリティクス動作確認

### Gate 4: Day 21（リリース判定）
- FreeBSD本番動作確認
- iOS Safari手動テスト完了
- **デプロイ/ロールバックスクリプト動作確認**
- E2Eハッピーパス

---

## 8. 承認条件（実装着手前に必須）

1. **PostgreSQLバックアップ方針:** pg_dump日次 + 外部転送。Gate 1完了条件に含める
2. **ローカル開発環境の構成を明記:** Homebrew PostgreSQL + go run + vite devの構成を文書化
3. **レンタルサーバー → OpenAI APIレイテンシ計測:** Gate 1に含める
4. **デプロイ/ロールバックスクリプトの一本化:** Gate 4完了条件に含める
5. **JWT_SECRET要件の明記:** 最低256ビットランダム値、環境変数管理
6. **リソース配分の事前設計:** PostgreSQL 40% / Go 30% / OS+nginx 30%

---

## 9. 最終確定の技術スタック

| レイヤー | 技術 |
|---------|------|
| フロントエンド | Vite + React 19 + React Router v7 + TypeScript + Tailwind CSS v4 + shadcn/ui + Zustand + Framer Motion |
| バックエンド | Go 1.23+ / Chi v5 / sqlc / pgx v5 / goose / bcrypt / golang-jwt |
| DB | PostgreSQL 16（FreeBSD ローカル） |
| AI/音声 | go-openai (GPT-4o-mini + Whisper + TTS) |
| 認証 | 自前認証 (bcrypt + JWT HS256) + メールホワイトリスト |
| インフラ | FreeBSDレンタルサーバー1台（nginx + Goバイナリ + PostgreSQL） |
| デプロイ | GitHub Actions → rsync/scp |

---

## 総評

Supabase/Vercel/Cloud RunからFreeBSD 1台への集約は、コスト削減と運用簡素化として理解できる。Next.js→Viteの判断も技術的に正しい。自前認証+ホワイトリストの設計は堅実。

主な懸念は「1台構成の運用安定性」と「バックアップ」だが、ホワイトリスト制の限定ユーザーMVPであれば許容範囲。

上記6条件を反映の上、実装を開始すること。
