# スコア履歴 設計書

## 0. ステータス

- **状態**: 未実装の設計案（2026-06-27 時点、レビュー反映済み）
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
| 索引 | `player_id` + 譜面 ID + `updated_at` の複合主キーで参照と prune を効率化 |
| 読み取り | 1 譜面ずつオンデマンド取得のみ |

### 2.3 ゲーム仕様の前提

- スコア・ランプは登録データ上、単調に改善する（ダウングレードは発生しない）
- 同一譜面の同時登録リクエストは通常発生しないが、更新対象の現行レコードをロックして履歴と現行値の整合性を保証する

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

> **補足**: 同日に BAS で初プレイした後、EXPERT で初プレイした場合、EXPERT 側の histories は空のまま（EXPERT の `change_type: new` のため）。別難易度の成長は見えないが、BAS / ADV を履歴対象外とする非目的に合致する。

#### 2 回目以降の更新（`change_type: updated` かつ意味のある変化あり）

同一トランザクション内で、次の順序を守る。

1. 対象となる現行レコードを `SELECT ... FOR UPDATE` でロックし、`before` 状態を取得
2. 対象難易度（後述）を満たす場合、**更新前状態（before）** を histories へ INSERT
   - `updated_at` には `before.UpdatedAt`（そのスコアを登録した日時）を保存
3. `SavePlayerData` で `player_records` を after で UPSERT
4. 今回 histories に INSERT した譜面について prune を実行

#### 意味のある変化の判定

既存の `playerRecordMeaningfullyChanged` / `worldsendRecordMeaningfullyChanged` をそのまま利用する。

- 対象: `score`, `clear_lamp_id`, `combo_lamp_id`, `full_chain_id`
- 非対象: `slot_id`, `slot_order`, `updated_at`

### 3.3 表示フロー（読み取り）

1. プライバシーチェック（後述）
2. `display_id` + 難易度（通常譜面）または `display_id`（WORLD'S END）から内部 chart ID を解決
3. `player_records`（または `player_worldsend_records`）を取得
   - 行がなければ `404`（`score_history_not_found`）。histories のみ存在する不整合時もフォールバックせず同じく `404`
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
| `updated_at` | TIMESTAMP | **不変**。`ON UPDATE CURRENT_TIMESTAMP` は付けない。セマンティクスは「そのスコアを登録した日時」（`player_records.updated_at` の退避値）であり、histories 行が INSERT された日時ではない |

- **PRIMARY KEY**: `(player_id, chart_id, updated_at)`

#### `player_worldsend_record_histories`

| カラム | 型 | 備考 |
| --- | --- | --- |
| `player_id` | MEDIUMINT UNSIGNED | FK → `players(id)` ON DELETE CASCADE |
| `worldsend_chart_id` | MEDIUMINT UNSIGNED | FK → `worldsend_charts(id)` ON DELETE CASCADE |
| `score` | MEDIUMINT UNSIGNED | CHECK 0〜1,010,000 |
| `clear_lamp_id` | TINYINT UNSIGNED | FK なし |
| `combo_lamp_id` | TINYINT UNSIGNED | FK なし |
| `full_chain_id` | TINYINT UNSIGNED | FK なし |
| `updated_at` | TIMESTAMP | 不変。意味は通常譜面側と同様（スコア登録日時の退避値） |

- **PRIMARY KEY**: `(player_id, worldsend_chart_id, updated_at)`

`slot_id` / `slot_order` は **通常譜面の histories のみ** 対象外（差分判定対象外のため）。WORLD'S END は元テーブルに `slot` 系カラムが存在しない。

### 4.2 パーティション

**採用しない。** MySQL のパーティション化 InnoDB テーブルは外部キーを利用できず、既存の参照整合性と両立しないためである。履歴の取得と prune は、`player_id`、譜面 ID、`updated_at` からなる複合主キーを利用する。

### 4.3 難易度フィルタ（書き込み時）

| 種別 | 履歴対象 |
| --- | --- |
| 通常譜面 | EXPERT / MASTER / ULTIMA のみ |
| WORLD'S END | 全件（難易度概念なし） |

BAS / ADVANCED は `player_records` への保存は既存どおり継続するが、histories には書き込まない。

判定は `applyScores` 内のマスタ（`masters.ChartsByID` の difficulty）で行い、DB JOIN は不要。

### 4.4 件数上限（prune）

- 譜面ごとに histories は最大 **50 件**（定数は `internal/info` に定義。例: `MaxScoreHistoryEntriesPerChart = 50`）
- 51 件目 INSERT 後、同一トランザクション内で 51 件目以降を DELETE
- INSERT から prune までのトランザクション内では一時的に 51 件以上になりうるが、コミット時点では必ず 50 件以内とする
- 50 件を超えた古い履歴は永久に失われる（仕様）
- 新しい順の判定は `updated_at DESC` とする。主キー制約により、同一譜面で同一 `updated_at` の履歴は複数存在しない

#### 同時登録時の競合

同一譜面への登録は、現行レコードを `SELECT ... FOR UPDATE` でロックして直列化する。後続トランザクションは先行トランザクションのコミット後に最新の before を取得し、履歴 INSERT、現行レコード更新、prune を実行するため、コミット時点で 50 件以内を維持する。

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
| `clear_lamp` / `combo_lamp` / `full_chain` | マスタ `Name` を返す。`none` 相当・未設定・マスタ未解決は `null`（`playerDataLampNamePtr` / 既存レコード API と同じ。`"NONE"` 文字列は返さない） |
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
| 楽曲 `display_id` 不存在 | 404 | `song_not_found` |
| 楽曲は存在するが指定難易度の譜面がない | 404 | `chart_not_found` |
| 未プレイ（現行レコードなし） | 404 | `score_history_not_found`（新規） |
| 難易度パス不正 | 400 | `invalid_difficulty` |
| BAS / ADVANCED で履歴要求 | 400 | `score_history_unsupported_difficulty`（新規） |
| `username` 未指定 | 400 | `validation_failed` |

BAS / ADVANCED は **404 ではなく 400** とする。404 だと未プレイ・ユーザー不存在と区別できず、クライアントが履歴 UI の表示可否を判断しづらいため。

### 5.7 主キー衝突（`updated_at` 秒精度）

アプリケーションの登録フロー上、同一譜面が秒間 2 回更新されることはないため、通常利用では同一 `(player_id, chart_id, updated_at)` の衝突は発生しない前提とする。

万が一衝突した場合は異常な登録操作として扱い、既存履歴を上書き・無視せず、履歴 INSERT の主キー重複エラーによって登録トランザクション全体をロールバックする。Repository はこの重複を `ErrScoreHistoryTimestampConflict` へ変換し、呼び出し元は登録失敗として明示的にエラーを返す。

---

## 6. アーキテクチャ

Clean Architecture / DDD の依存規則に従う。

### 6.1 Domain

- 履歴エンティティ（または履歴行を表す値オブジェクト）
- 難易度が履歴対象かを判定するロジック（EXPERT / MASTER / ULTIMA）
- 件数上限定数: `internal/info.MaxScoreHistoryEntriesPerChart`（Repository 実装が参照。Usecase から `limit` を注入しない）

### 6.2 Usecase

#### 書き込み（既存フローへの組み込み）

`PlayerDataUsecase.applyScores` 内:

1. 登録対象に対応する既存の現行レコードを `SELECT ... FOR UPDATE` で取得し、トランザクション終了までロック
2. ロック下で取得した before を使い、既存どおり `computeFullRecordChanges` / `computeWorldsendRecordChanges` で差分算出
3. `updated` かつ対象難易度のレコードについて、histories への INSERT 対象を収集
4. histories へバルク INSERT（`before` 状態）
5. `SavePlayerData` で現行レコードを UPSERT
6. 今回 INSERT した譜面 ID に対して prune（同一トランザクション）

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
    FindStandardStatesForUpdate(ctx context.Context, exec Executor, playerID int, chartIDs []int) (map[int]PlayerRecordState, error)
    FindWorldsendStatesForUpdate(ctx context.Context, exec Executor, playerID int, chartIDs []int) (map[int]WorldsendRecordState, error)
    BulkInsertStandard(ctx context.Context, exec Executor, rows []PlayerRecordHistoryRow) error
    BulkInsertWorldsend(ctx context.Context, exec Executor, rows []WorldsendRecordHistoryRow) error
    PruneStandardOverLimit(ctx context.Context, exec Executor, playerID int, chartIDs []int) error
    PruneWorldsendOverLimit(ctx context.Context, exec Executor, playerID int, chartIDs []int) error
    FindStandardByPlayerAndChart(ctx context.Context, exec Executor, playerID, chartID int) ([]PlayerRecordHistoryRow, error)
    FindWorldsendByPlayerAndChart(ctx context.Context, exec Executor, playerID, worldsendChartID int) ([]WorldsendRecordHistoryRow, error)
}
```

- 本プロジェクトでは RDB 以外の永続化方式を採用しないため、実装コストを抑える目的で `Executor` の Repository インターフェースへの露出を限定的に許容する
- `Executor` はトランザクションを共有する必要があるメソッドにのみ渡し、Usecase から SQL 文や `sqlx` 固有型を直接操作しない
- 件数上限は Repository 実装が `info.MaxScoreHistoryEntriesPerChart` を参照し、インターフェースに `limit` 引数は持たせない
- `SELECT *` 禁止（明示カラム指定）
- prune は「今回 INSERT した譜面 ID」のみを対象にバッチ実行し、各譜面について新しい順の先頭 50 件を残して 51 件目以降を削除する
- 履歴 INSERT の主キー重複は無視せず、`ErrScoreHistoryTimestampConflict` へ変換する

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

### 7.3 索引

履歴テーブルの複合主キー `(player_id, chart_id, updated_at)` または `(player_id, worldsend_chart_id, updated_at)` を、履歴取得と prune の両方で利用する。初版では追加索引を設けず、実データのクエリプランと性能を確認してから必要性を判断する。

---

## 8. 実装タスク分解

1. マイグレーション
   - `player_record_histories` / `player_worldsend_record_histories` 作成
2. Domain: 履歴行・難易度判定
3. Repository: ロック付き現行取得 / INSERT / prune / SELECT / 主キー重複エラー変換
4. `applyScores` への書き込み組み込み
5. `ScoreHistoryUsecase` + 読み取り Repository 呼び出し
6. Handler / Router / DTO
7. API エラーコード追加（`score_history_unsupported_difficulty`、`score_history_not_found`）
8. `docs/API.md` 追記
9. テスト
   - 初回のみプレイ → histories 0 件
   - 2 回目更新 → histories 1 件（before が保存される）
   - EXPERT+ 以外は histories に書き込まれない
   - WORLD'S END も同様に動作
   - 51 件目で prune
   - 同一 `updated_at` の履歴 INSERT でエラーとなり、登録全体がロールバック
   - 同時登録時に現行レコードがロックされ、履歴と現行値の整合性が維持される
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
| ランプのみ改善 | before が histories に 1 件 |
| スコア改善後、別の登録でランプのみ改善 | それぞれの before が時系列どおり保存される |
| 履歴が 49 件の状態で改善 | 50 件になる |
| 履歴が 50 件の状態で改善 | 51 件目以降が削除され 50 件を維持 |
| 履歴が上限を超えている状態で改善 | 51 件目以降がすべて削除され 50 件を維持 |
| 同一 `updated_at` の履歴が既に存在 | 識別可能なエラーとなり、履歴・現行レコードを含む登録全体がロールバック |
| histories INSERT 失敗 | 登録全体がロールバック |
| prune 失敗 | 登録全体がロールバック |
| 同一譜面への同時登録 | 現行レコードのロックにより直列化され、同じ before を重複保存しない |
| プレイヤー削除 | 外部キーの `ON DELETE CASCADE` により通常・WORLD'S ENDの履歴も削除 |

### 9.2 読み取り API

| ケース | 期待 |
| --- | --- |
| 初回のみプレイ済み | `entries` 1 件（現行のみ） |
| 複数回更新済み | 先頭が現行、以降 histories |
| 未プレイ | 404 `score_history_not_found` |
| 指定難易度の譜面なし | 404 `chart_not_found` |
| BAS/ADV で要求 | 400 `score_history_unsupported_difficulty` |
| private 本人以外 | 404 |
| private 本人 | 200 |
| 同一秒の更新を含まない正常な複数履歴 | `updated_at DESC` の決定的な順序で返る |

---

## 10. 未決事項

### 10.1 確定済み（レビュー反映）

| 項目 | 決定 |
| --- | --- |
| 未プレイ時のエラーコード | 新規 `score_history_not_found`（HTTP 404）。`resource_not_found`（HTTP 400）は使用しない |
| prune の同時実行競合 | 現行レコードをロックして同一譜面への登録を直列化し、コミット時点で 50 件以内を維持 |
| 現行レコード消失時 | フォールバックなし。`score_history_not_found`（404） |

### 10.2 その他（実装フェーズで対応可）

| 項目 | 備考 |
| --- | --- |
| internal API の要否 | 初版は外部 v1 のみ。必要になれば後追い |
| `docs/er_diagram.puml` 更新 | マイグレーション実装時に合わせて更新 |
