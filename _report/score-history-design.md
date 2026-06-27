# スコア履歴 設計書

## 0. ステータス

- **状態**: 未実装の設計案（2026-06-27 時点）
- **対象リリース**: 初回リリース前（バックフィルなし）
- **関連仕様**:
  - プレイヤーデータ登録: `docs/player_data_registration_diff_specification.md`
  - 外部 API: `docs/API.md`（実装時に追記）

---

## 1. 目的

1譜面ごとに、スコアの伸びやランプ更新のたびに履歴を記録し、ユーザー（本人および公開設定に応じた他ユーザー）が参照できるようにする。

### 非目的

- 複数譜面の履歴を一括取得する API
- 導入以前のデータのバックフィル
- BAS / ADVANCED 難易度の履歴保存
- `slot` / `slot_order` の変化の履歴保存

---

## 2. 背景と課題

### 2.1 容量問題

`player_records` はユーザー数・譜面数に比例して肥大化する。履歴を無制限に保存すると、同規模の別サービス（18,000 ユーザーで DB 約 130GB）を超える可能性がある。

### 2.2 設計方針

| 方針 | 内容 |
| --- | --- |
| 別テーブル | `player_records` は現行どおり 1 譜面 1 行を維持し、履歴は専用テーブルへ分離 |
| 最新の非重複 | 常に最新 1 件は `player_records`（または `player_worldsend_records`）のみから取得 |
| 難易度フィルタ | 通常譜面は EXPERT / MASTER / ULTIMA のみ履歴対象 |
| 件数上限 | 譜面ごとに最大 50 件（初版から適用） |
| パーティション | `HASH(player_id)` で 4 テーブルすべてに適用 |
| 読み取り | 1 譜面ずつオンデマンド取得のみ |

### 2.3 ゲーム仕様の前提

- スコア・ランプは登録データ上、単調に改善する（ダウングレードは発生しない）
- 同一譜面の同時登録リクエストは通常発生しない（既存 `applyScores` と同じ前提）

---

## 3. ストレージモデル

### 3.1 責務分担

```
player_records / player_worldsend_records
  → 常に「現在のベスト」1 件

player_record_histories / player_worldsend_record_histories
  → 「過去のベスト」のみ（更新時に退避した before 状態）
```

最新レコードを histories に重複保存しない。API 表示時は `player_records` を先頭に、histories を時系列で連結する。

### 3.2 書き込みフロー

`PlayerDataUsecase.applyScores` の同一トランザクション内で実行する。

#### 初回プレイ（`change_type: new`）

1. `player_records`（または `player_worldsend_records`）へ UPSERT
2. histories には **書き込まない**

#### 2 回目以降の更新（`change_type: updated` かつ意味のある変化あり）

1. 対象難易度（後述）を満たす場合、**更新前状態（before）** を histories へ INSERT
   - `updated_at` には `before.UpdatedAt`（そのスコアを登録した日時）を保存
2. `player_records` を after で UPDATE
3. 対象譜面について histories が 51 件を超えたら、古い行を同期 DELETE（prune）

#### 意味のある変化の判定

既存の `playerRecordMeaningfullyChanged` / `worldsendRecordMeaningfullyChanged` をそのまま利用する。

- 対象: `score`, `clear_lamp_id`, `combo_lamp_id`, `full_chain_id`
- 非対象: `slot_id`, `slot_order`, `updated_at`

### 3.3 表示フロー（読み取り）

1. プライバシーチェック（後述）
2. `display_id` + 難易度（通常譜面）または `display_id`（WORLD'S END）から内部 chart ID を解決
3. `player_records`（または `player_worldsend_records`）を取得
   - 未プレイなら `404`
4. histories を `player_id` + `chart_id` で `updated_at DESC` 取得（最大 50 件）
5. レスポンス `entries` の先頭に現行レコード、続けて histories を並べる

### 3.4 タイムライン例

| 操作 | player_records | histories |
| --- | --- | --- |
| 初回 990,000 | 990,000 | （空） |
| 2 回目 1,000,000 | 1,000,000 | 990,000 |
| 3 回目 1,005,000 | 1,005,000 | 1,000,000, 990,000 |

表示（新しい順）: 1,005,000 → 1,000,000 → 990,000

---

## 4. テーブル設計

### 4.1 新規テーブル

#### `player_record_histories`

| カラム | 型 | 備考 |
| --- | --- | --- |
| `player_id` | MEDIUMINT UNSIGNED | FK → `players(id)` ON DELETE CASCADE |
| `chart_id` | MEDIUMINT UNSIGNED | FK → `charts(id)` ON DELETE CASCADE |
| `score` | MEDIUMINT UNSIGNED | CHECK 0〜1,010,000 |
| `clear_lamp_id` | TINYINT UNSIGNED | FK なし（INSERT 軽量化） |
| `combo_lamp_id` | TINYINT UNSIGNED | 同上 |
| `full_chain_id` | TINYINT UNSIGNED | 同上 |
| `updated_at` | TIMESTAMP | **不変**。`ON UPDATE CURRENT_TIMESTAMP` は付けない |

- **PRIMARY KEY**: `(player_id, chart_id, updated_at)`
- **パーティション**: `PARTITION BY HASH(player_id) PARTITIONS 8`

#### `player_worldsend_record_histories`

| カラム | 型 | 備考 |
| --- | --- | --- |
| `player_id` | MEDIUMINT UNSIGNED | FK → `players(id)` ON DELETE CASCADE |
| `worldsend_chart_id` | MEDIUMINT UNSIGNED | FK → `worldsend_charts(id)` ON DELETE CASCADE |
| `score` | MEDIUMINT UNSIGNED | CHECK 0〜1,010,000 |
| `clear_lamp_id` | TINYINT UNSIGNED | FK なし |
| `combo_lamp_id` | TINYINT UNSIGNED | FK なし |
| `full_chain_id` | TINYINT UNSIGNED | FK なし |
| `updated_at` | TIMESTAMP | 不変 |

- **PRIMARY KEY**: `(player_id, worldsend_chart_id, updated_at)`
- **パーティション**: `PARTITION BY HASH(player_id) PARTITIONS 8`

`slot_id` / `slot_order` は histories に含めない（差分判定対象外のため）。

### 4.2 既存テーブルへのパーティション追加

初版マイグレーション時に、以下にも同じパーティションを適用する。

| テーブル | 既存 PRIMARY KEY | パーティション |
| --- | --- | --- |
| `player_records` | `(player_id, chart_id)` | `HASH(player_id) PARTITIONS 8` |
| `player_worldsend_records` | `(player_id, worldsend_chart_id)` | `HASH(player_id) PARTITIONS 8` |

**根拠**: 現行の参照クエリはすべて `WHERE player_id = ?` で始まる。`idx_player_records_chart_id` を使う `chart_id` 単体検索は現コードベースに存在しない。

### 4.3 難易度フィルタ（書き込み時）

| 種別 | 履歴対象 |
| --- | --- |
| 通常譜面 | EXPERT / MASTER / ULTIMA のみ |
| WORLD'S END | 全件（難易度概念なし） |

BAS / ADVANCED は `player_records` への保存は既存どおり継続するが、histories には書き込まない。

判定は `applyScores` 内のマスタ（`masters.ChartsByID` の difficulty）で行い、DB JOIN は不要。

### 4.4 件数上限（prune）

- 譜面ごとに histories は最大 **50 件**
- 51 件目 INSERT 後、同一トランザクション内で最古行を DELETE
- 一時的に 51 件超えを許容しない（同期 prune）
- 50 件を超えた古い履歴は永久に失われる（仕様）

**ユーザー向け文言（案）**: 「各譜面の更新履歴は最大 50 件まで保存されます」

### 4.5 容量見積もり（参考）

1 行あたりおおよそ 20〜50 バイト（データ + InnoDB オーバーヘッド + 索引）。

| シナリオ | 規模感 |
| --- | --- |
| 現実的（30,000 ユーザー × 500 譜面 × 平均 5 改善） | 約 7,500 万行 → 数 GB〜十数 GB |
| 最悪（上限フル活用） | 数百 GB 級になりうる |

EXPERT+ フィルタ・50 件 cap・最新の非重複・改善時のみ INSERT の組み合わせで、無制限保存と比べて桁違いに抑制できる。

### 4.6 採用しない最適化

| 手法 | 理由 |
| --- | --- |
| TINYINT ビットパック | 節約量が少なく、ランプ種別追加時の拡張が煩雑 |
| バックフィル | リリース前のため不要 |

---

## 5. API 設計

### 5.1 パス設計方針

譜面統計 API（`GET /api/v1/songs/:displayid/stats/:difficulty`）と同様、**楽曲を主語**とする。`/users/:username/songs/...` のようなユーザー所有の階層は採用しない。

### 5.2 エンドポイント

#### 通常譜面

- **Method**: `GET`
- **Path**: `/api/v1/songs/:displayid/score-history/:difficulty`
- **Query**: `username`（必須）— 履歴を参照する対象ユーザー名
- **Auth**: 任意（公開ユーザーの履歴は未認証でも可）

`:difficulty` は既存の `ParseDifficultyPath` と同じ変換規則を用いる（例: パス `master` → 内部 `MASTER`）。

#### WORLD'S END

- **Method**: `GET`
- **Path**: `/api/v1/worldsend-songs/:displayid/score-history`
- **Query**: `username`（必須）
- **Auth**: 任意

### 5.3 レスポンス

内部 ID（`chart_id`, `player_id` 等）は含めない。譜面の特定はリクエスト URL で完結する。

```json
{
  "entries": [
    {
      "score": 1009000,
      "clear_lamp": "ABSOLUTE",
      "combo_lamp": "ALL JUSTICE",
      "full_chain": "FULL CHAIN GOLD",
      "updated_at": "2026-04-27T12:34:56Z"
    },
    {
      "score": 1005000,
      "clear_lamp": "BRAVE",
      "combo_lamp": "FULL COMBO",
      "full_chain": null,
      "updated_at": "2026-03-15T08:00:00Z"
    }
  ]
}
```

| フィールド | 仕様 |
| --- | --- |
| `entries` | 新しい順。先頭は常に `player_records` の現行ベスト |
| `score` | 0〜1,010,000 |
| `clear_lamp` / `combo_lamp` / `full_chain` | マスタ `Name` を返す。`none` 相当・未設定は `null`（既存レコード API と同じ） |
| `updated_at` | RFC3339 |

`is_current` フラグは設けない（配列の先頭が現行ベスト）。

### 5.4 `include_noplay` について

**設けない。**

`include_noplay` はユーザーの全レコード一覧向けパラメータであり、1 譜面を指定する詳細 API とは用途が異なる。未プレイ譜面は `404` を返す。

### 5.5 プライバシー

既存のユーザープロフィール参照と同じパターンを用いる。

- `user.IsPrivate == true` かつ閲覧者が本人でない → `ErrUserPrivate`（HTTP 404, `user_not_found`）
- 存在しない `username` → `404`

### 5.6 エラーレスポンス

| 条件 | HTTP | code |
| --- | --- | --- |
| 対象ユーザーが private（本人以外） | 404 | `user_not_found` |
| ユーザー不存在 | 404 | `user_not_found` |
| 楽曲 `display_id` 不存在 | 404 | 既存の楽曲 API に準拠 |
| 未プレイ（現行レコードなし） | 404 | `resource_not_found`（新規または既存コードを利用） |
| 難易度パス不正 | 400 | `invalid_difficulty` |
| BAS / ADVANCED で履歴要求 | 400 | `score_history_unsupported_difficulty`（新規） |
| `username` 未指定 | 400 | `validation_failed` |

BAS / ADVANCED は **404 ではなく 400** とする。404 だと未プレイ・ユーザー不存在と区別できず、クライアントが履歴 UI の表示可否を判断しづらいため。

### 5.7 主キー衝突（`updated_at` 秒精度）

同一 `(player_id, chart_id, updated_at)` の衝突は現実的に発生しない前提とする。万が一発生した場合はユーザー側の異常操作とみなし、どちらの行を残してもよい（仕様上の許容）。

---

## 6. アーキテクチャ

Clean Architecture / DDD の依存規則に従う。

### 6.1 Domain

- 履歴エンティティ（または履歴行を表す値オブジェクト）
- 難易度が履歴対象かを判定するロジック（EXPERT / MASTER / ULTIMA）

### 6.2 Usecase

#### 書き込み（既存フローへの組み込み）

`PlayerDataUsecase.applyScores` 内:

1. 既存どおり `computeFullRecordChanges` / `computeWorldsendRecordChanges` で差分算出
2. `updated` かつ対象難易度のレコードについて、histories への INSERT 対象を収集
3. `SavePlayerData` の前後で histories INSERT + prune を実行（同一トランザクション）

#### 読み取り（新規）

`ScoreHistoryUsecase`（名称は実装時に確定）:

1. `username` からユーザーを解決しプライバシーチェック
2. `display_id` + 難易度から chart を解決
3. 難易度が BAS / ADVANCED なら `ErrScoreHistoryUnsupportedDifficulty`
4. 現行レコード + histories を取得して DTO 化

### 6.3 Repository

`internal/domain/repository` にインターフェースを定義。

```go
// 概略（実装時に詳細化）
type PlayerRecordHistoryRepository interface {
    BulkInsertStandard(ctx context.Context, exec Executor, rows []PlayerRecordHistoryRow) error
    BulkInsertWorldsend(ctx context.Context, exec Executor, rows []WorldsendRecordHistoryRow) error
    PruneStandardOverLimit(ctx context.Context, exec Executor, playerID int, chartIDs []int, limit int) error
    PruneWorldsendOverLimit(ctx context.Context, exec Executor, playerID int, chartIDs []int, limit int) error
    FindStandardByPlayerAndChart(ctx context.Context, exec Executor, playerID, chartID int) ([]PlayerRecordHistoryRow, error)
    FindWorldsendByPlayerAndChart(ctx context.Context, exec Executor, playerID, worldsendChartID int) ([]WorldsendRecordHistoryRow, error)
}
```

- `SELECT *` 禁止（明示カラム指定）
- prune は「今回 INSERT した譜面 ID」のみを対象にバッチ実行し、INSERT ごとの DELETE を避ける

### 6.4 Infra

- `internal/infra/models` に DB モデル（`ToEntity` / `FromEntity`）
- `internal/infra/repository` に実装

### 6.5 Presentation

- `internal/app/handler/api_v1` にハンドラ追加
- `internal/app/router.go` にルート登録
- `internal/dto/api_v1` にレスポンス DTO

---

## 7. パフォーマンス設計

### 7.1 書き込み

- histories INSERT は登録 1 回あたり変更譜面数分のバルク INSERT
- prune は変更譜面 ID に限定
- 登録処理のクリティカルパスへの影響を最小化する

### 7.2 読み取り

- 1 リクエストあたり DB アクセス目標: **3 回以内**
  1. ユーザー解決
  2. 現行レコード 1 件
  3. histories 最大 50 件
- 複数譜面の一括取得 API は設けない
- `(player_id, chart_id, updated_at)` PK により追加索引は不要

### 7.3 パーティション

`HASH(player_id) PARTITIONS 8` により、`player_id` 指定クエリでパーティションプルーニングが効く。テーブル作成時に適用する（後からの `ALTER` は全件再構築が必要なため）。

---

## 8. 実装タスク分解

1. マイグレーション
   - `player_record_histories` / `player_worldsend_record_histories` 作成
   - `player_records` / `player_worldsend_records` へのパーティション適用
2. Domain: 履歴行・難易度判定
3. Repository: INSERT / prune / SELECT
4. `applyScores` への書き込み組み込み
5. `ScoreHistoryUsecase` + 読み取り Repository 呼び出し
6. Handler / Router / DTO
7. API エラーコード追加（`score_history_unsupported_difficulty`）
8. `docs/API.md` 追記
9. テスト
   - 初回のみプレイ → histories 0 件
   - 2 回目更新 → histories 1 件（before が保存される）
   - EXPERT+ 以外は histories に書き込まれない
   - WORLD'S END も同様に動作
   - 51 件目で prune
   - private ユーザーは他人から 404
   - BAS/ADV API 要求で 400
   - 未プレイで 404

---

## 9. テスト観点

### 9.1 書き込み（`applyScores`）

| ケース | 期待 |
| --- | --- |
| 初回プレイ（EXPERT+） | `player_records` のみ更新、histories 不変 |
| 2 回目改善 | before が histories に 1 件、player_records が after |
| 変化なしの再登録 | histories 不変 |
| slot のみ変化 | histories 不変 |
| BAS / ADV 更新 | player_records は更新、histories 不変 |
| 51 回目の改善 | 最古 1 件が削除され 50 件維持 |

### 9.2 読み取り API

| ケース | 期待 |
| --- | --- |
| 初回のみプレイ済み | `entries` 1 件（現行のみ） |
| 複数回更新済み | 先頭が現行、以降 histories |
| 未プレイ | 404 |
| BAS/ADV で要求 | 400 `score_history_unsupported_difficulty` |
| private 本人以外 | 404 |

---

## 10. 未決事項

現時点で実装開始に必要な決定は完了している。実装フェーズで詰める細部は以下。

| 項目 | 備考 |
| --- | --- |
| `resource_not_found` vs 専用 code（未プレイ） | 既存エラーコードの再利用可否を実装時に確認 |
| internal API の要否 | 初版は外部 v1 のみ。必要になれば後追い |
| `docs/er_diagram.puml` 更新 | マイグレーション実装時に合わせて更新 |