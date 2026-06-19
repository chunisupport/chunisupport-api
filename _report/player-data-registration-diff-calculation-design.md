# プレイヤーデータ登録時差分計算機能 設計書

作成日: 2026-06-04  
最終更新日: 2026-06-08

## 0. 位置づけ

本設計書は、プレイヤーデータ登録時に「今回の登録で実際に変化したスコア」をレスポンスへ含める機能の仕様を定義する。

現在の登録エンドポイントは以下の通り。

- `POST /internal/me/register-data`
- `POST /internal/player-data/temp` + `POST /internal/player-data/commit`

`/internal/player-data/commit` は一時保存済み本文を `PlayerDataPayload` として解釈し、最終的に通常の `PlayerDataUsecase.Register` を通る。そのため、差分計算は `PlayerDataUsecase.Register` 配下に実装すれば、直接登録と一時保存確定の両方を同じ仕様で扱える。

現行実装では、`PlayerDataPayload.scores.full` / `scores.worldsend` に含まれる全譜面の現在ベストを受け取り、`applyScores` で `PlayerRecordForUpsert` / `WorldsendRecordForUpsert` に変換し、`playerDataRepo.SavePlayerData` で bulk upsert している。`PlayerDataCounts` の `*_upserted` は「payload 内で処理対象になった件数」を表す。

また、`player_records.updated_at` / `player_worldsend_records.updated_at` は payload の `updated_at` を格納する。`ON DUPLICATE KEY UPDATE` 内で、score / lamp が変化した場合に `updated_at` を更新している。

## 1. 目的

プレイヤーデータ登録完了時に、サーバ側で前回状態と今回payloadを比較し、実際に変化したレコード数と変更内容を返す。

目的は以下。

- クライアントが「今回の登録でスコアが何件変わったか」「どの譜面が新規登録または更新されたか」を即座に表示できるようにする。
- 複数端末・複数クライアント間の状態差による比較精度を上げる。
- payload から生成した upsert 予定値と保存前 state を比較し、今回登録でDBに保存される値の差分を確定する。
- 既存の登録セマンティクス、トランザクション境界、rating / overpower 再計算を維持する。

本機能で扱う「差分」は、ユーザー向けには「更新差分」と呼ぶが、内部仕様上は「改善」だけではなく「値の変化」を意味する。再取り込みや公式側データ訂正によりスコア・ランプが下がった場合も、DBに保存される値が変わるため差分に含める。

## 2. 採用方針

### 2.1 採用する方式

登録トランザクション内で、保存前の対象 state を読み込み、payload から生成した upsert 予定値と比較して差分を確定する。

差分判定は保存前 state と upsert 予定値の `score` / `clear_lamp_id` / `combo_lamp_id` / `full_chain_id` の比較で行う。payload 直下の `updated_at` は各レコードの保存値として `PlayerRecordState.UpdatedAt` / `WorldsendRecordState.UpdatedAt` に設定し、DB の `TIMESTAMP` 列へ保存する。

```text
tx 開始
  masters ロード
  ensurePlayer
  applyHonors
  counts, skipped, changes, overpower := applyScores(...)
    payloadをupsert対象へ変換
    同一キーのupsert対象を最後の1件へ正規化
    upsert対象キーの保存前stateを軽量ロード
    保存前stateとupsert予定値を比較してchangesを作成
    SavePlayerData
  overpower / rating 再計算
  登録前後の集計スナップショットを比較して aggregate_diff を作成
  result.Counts / result.Changes / result.AggregateDiff を設定
tx commit
```

保存前状態のロードは `SavePlayerData` より前に行う。差分判定はDB側の `updated_at = IF(...)` 条件と同じ比較カラムで行い、`SavePlayerData` 成功後にレスポンスへ反映する。

同じ `updated_at` を持つ payload が再送された場合も、保存前stateとupsert予定値が同じなら `changes` には含めない。今回の保存処理で値が変化するレコードだけを `changes` に含める。

差分計算は登録トランザクション内の保存前 state を基準とする。

## 3. 差分の定義

### 3.1 差分対象カラム

通常譜面、WORLD'S END ともに、以下のいずれかが異なる場合に `updated` とする。

- `score`
- `clear_lamp_id`
- `combo_lamp_id`
- `full_chain_id`

前回レコードが未登録の場合は `new` とする。

この条件は `internal/infra/repository/player_data_repository_impl.go` の `fullRecordChangedCondition` / `worldsendRecordChangedCondition` と一致させる。Go側の差分判定は、DB側の `updated_at = IF(...)` 条件と同期していなければならない。

### 3.2 初版の比較カラム

初版の `new` / `updated` 判定は以下の保存値で行う。

- `score`
- `clear_lamp_id`
- `combo_lamp_id`
- `full_chain_id`

`slot_id` / `slot_order` は通常譜面の保存値として更新される。差分レスポンスの初版では、DB側の `updated_at` 更新条件に合わせて、上記4項目の変化をユーザー向け差分として扱う。

### 3.3 スキップとの関係

差分計算は、既存の `applyFullScores` / `applyWorldsendScores` が upsert 対象として受理したレコードを対象にする。マスタ解決失敗、スコア範囲外、ランプ変換失敗などで `SkippedRecord` になった入力は、`skipped_records` で返す。

### 3.4 payload内重複

通常の入力では、同一譜面は payload 内に1件だけ存在する前提とする。

同一 `chart_id` または `worldsend_chart_id` が同一payload内に複数回現れた場合、初版では upsert 対象生成後にキーごとの最後の1件へ正規化してから保存・差分計算する。

これにより、DBの最終状態、`counts`、`changes` の詳細が同じキー単位で一致する。`*_upserted` は従来通りpayload内で処理対象になった件数を表し、正規化後の保存件数とは区別する。

## 4. レスポンス設計

### 4.1 Counts

`internal/dto/api_internal/player_data_dto.go` の `PlayerDataCounts` に以下を追加する。

```go
type PlayerDataCounts struct {
	FullRecordsUpserted      int `json:"full_records_upserted"`
	WorldsendRecordsUpserted int `json:"worldsend_records_upserted"`
	FullRecordsSkipped       int `json:"full_records_skipped"`
	WorldsendRecordsSkipped  int `json:"worldsend_records_skipped"`
	HonorsSkipped            int `json:"honors_skipped"`

	FullRecordsActuallyChanged      int `json:"full_records_actually_changed"`
	WorldsendRecordsActuallyChanged int `json:"worldsend_records_actually_changed"`
}
```

`*_actually_changed` は `new` と `updated` の合計である。`*_upserted` から `*_skipped` を引いた件数とは別の意味を持つ。同一値の再登録は upsert 対象件数として数え、保存前stateとupsert予定値が同じ場合は `actually_changed` に反映される件数が0になる。

### 4.2 Changes

`PlayerDataResult` に `changes` を追加する。

```go
type PlayerDataResult struct {
	PlayerID       int                      `json:"player_id"`
	AppVersion     string                   `json:"app_ver"`
	ImportedAt     time.Time                `json:"imported_at"`
	Summary        PlayerDataSummary        `json:"summary"`
	Counts         PlayerDataCounts         `json:"counts"`
	AggregateDiff  PlayerDataAggregateDiff  `json:"aggregate_diff"`
	Changes        []PlayerDataRecordChange `json:"changes"`
	SkippedRecords []SkippedRecord          `json:"skipped_records"`
}
```

差分詳細は入力DTOの `PlayerDataScoreEntry` をそのまま返さず、比較対象カラムだけを持つ専用DTOにする。入力DTOには `slot` / `order` が含まれており、初版の差分対象と誤解されやすいためである。

登録成功時は `changes` を常に配列として返す。差分が0件の場合は空配列 `[]` とする。`before` は `new` の場合に `null`、`updated` の場合に変更前状態を返す。`SavePlayerData` 成功後に `Register` が `result.Changes` を設定する。

`changes` の詳細は最大100件まで返す。`*_actually_changed` は実際に変化した全件数を表し、`changes` の件数とは一致しない場合がある。詳細に含める100件は `idx` を数値として昇順に並べて選ぶ。同一 `idx` の場合は `record_type`、`diff` の順で安定させる。通常譜面と WORLD'S END の両方に差分がある場合も、同一レスポンス内の `changes` 全体で最大100件とする。`idx` は公式インデックスの数値文字列のみを想定し、数値として解釈できない値があればソート順の末尾にまとめる。

```go
type PlayerDataRecordChange struct {
	RecordType string                  `json:"record_type"` // "full" | "worldsend"
	ChangeType string                  `json:"change_type"` // "new" | "updated"
	Idx        string                  `json:"idx"`
	Diff       string                  `json:"diff"`
	Before     *PlayerDataRecordState  `json:"before"`
	After      PlayerDataRecordState   `json:"after"`
}

type PlayerDataRecordState struct {
	Score     int     `json:"score"`
	ClearLamp *string `json:"clear_lamp"`
	ComboLamp *string `json:"combo_lamp"`
	FullChain *string `json:"full_chain"`
}
```

`before` / `after` は `score` と3種類のランプ名を返す。ランプ名は登録処理でロード済みの `masters` から逆引きし、マスタの `Name` をそのまま用いる。`none` 相当および未設定は `null` とする。

WORLD'S END の `diff` は入力値に依存せず、レスポンスでは `"WE"` 固定にする。

### 4.3 集計差分

譜面単位の `changes` とは別に、クライアントが登録完了画面で表示する集計値の差分を `aggregate_diff` として返す。

対象は以下。

- RATING
- OVER POWER
- TOTAL HIGH SCORE
- 難易度別 TOTAL HIGH SCORE / 平均スコア
- 難易度別 RECORD STATISTICS
- ランプ・スコアランク別達成数

`aggregate_diff` は登録成功時に常にオブジェクトとして返す。各フィールドは `before` / `after` / `delta` を持つ。初回登録などで登録前値が存在しない場合、数値の `before` は `0`、nullable なプロフィール由来値は `null` とする。

RATING は `players.official_player_rating`、つまり payload の `rating` から保存される公式レーティングを対象にする。計算レーティング（`calculated_player_rating`、`best_average_rating`、`new_average_rating`）は別概念であり、初版の `aggregate_diff.rating` には含めない。

OVER POWER は以下を返す。

- `value`: 登録処理と同じ条件で通常譜面から再集計した OVER POWER 値
- `percentage`: 登録処理時点の分母に対する OVER POWER 達成率

OVER POWER の `before` は、保存前レコードと同じ未解禁曲設定・同じ分母を用いて再計算する。DB に保存済みの `players.overpower_value` をそのまま使うと、分母や未解禁設定の変更によって `after` と比較条件がずれる可能性があるためである。

TOTAL HIGH SCORE は、通常譜面レコードの `score` 合計とする。WORLD'S END は初版の TOTAL HIGH SCORE 集計から除外する。難易度別集計は `BASIC` / `ADVANCED` / `EXPERT` / `MASTER` / `ULTIMA` をキーにし、レスポンスのキー・値はすべて大文字難易度名で扱う。

平均スコアは `total_score / played_count` とし、未プレイ補完レコードは含めない。`played_count` が0の場合、平均スコアは `null` とする。

RECORD STATISTICS は難易度別に以下を返す。各項目は `before` / `after` / `delta` を持つ。

| 項目 | 判定 |
| ---- | ---- |
| `aj` | `combo_lamp` が `ALL JUSTICE` |
| `fc` | `combo_lamp` が `FULL COMBO` または `ALL JUSTICE` |
| `clr` | `clear_lamp` が `FAILED` / none 相当ではない |
| `fch` | `full_chain` が none 相当ではない |
| `max` | `score == 1010000` |
| `sss_plus` | `score >= 1009000` |
| `sss` | `score >= 1007500` |
| `ss_plus` | `score >= 1005000` |
| `ss` | `score >= 1000000` |

スコアランク系の `sss_plus` / `sss` / `ss_plus` / `ss` は累積カウントとする。例えば `sss` は SSS 以上、`ss` は SS 以上の件数を表す。

```go
type PlayerDataAggregateDiff struct {
	Rating         NullableFloatDiff                    `json:"rating"`
	Overpower      PlayerDataOverpowerDiff              `json:"overpower"`
	TotalHighScore IntDiff                              `json:"total_high_score"`
	ByDifficulty   map[string]PlayerDataDifficultyDiff  `json:"by_difficulty"`
}

type PlayerDataOverpowerDiff struct {
	Value      NullableFloatDiff `json:"value"`
	Percentage NullableFloatDiff `json:"percentage"`
}

type PlayerDataDifficultyDiff struct {
	TotalHighScore IntDiff                   `json:"total_high_score"`
	AverageScore   NullableFloatDiff         `json:"average_score"`
	PlayedCount    IntDiff                   `json:"played_count"`
	Statistics     PlayerDataStatisticsDiff  `json:"statistics"`
}

type PlayerDataStatisticsDiff struct {
	AJ      IntDiff `json:"aj"`
	FC      IntDiff `json:"fc"`
	CLR     IntDiff `json:"clr"`
	FCH     IntDiff `json:"fch"`
	MAX     IntDiff `json:"max"`
	SSSPlus IntDiff `json:"sss_plus"`
	SSS     IntDiff `json:"sss"`
	SSPlus  IntDiff `json:"ss_plus"`
	SS      IntDiff `json:"ss"`
}

type IntDiff struct {
	Before int `json:"before"`
	After  int `json:"after"`
	Delta  int `json:"delta"`
}

type NullableFloatDiff struct {
	Before *float64 `json:"before"`
	After  *float64 `json:"after"`
	Delta  *float64 `json:"delta"`
}
```

`delta` は `after - before` とする。数値が低下した場合は負数を返す。`before` または `after` が `null` の nullable diff では、`delta` も `null` とする。

レスポンス例:

```json
{
  "aggregate_diff": {
    "rating": { "before": 17.34, "after": 17.35, "delta": 0.01 },
    "overpower": {
      "value": { "before": 96110.42, "after": 96123.91, "delta": 13.49 },
      "percentage": { "before": 76.26, "after": 76.27, "delta": 0.01 }
    },
    "total_high_score": { "before": 16363921404, "after": 16363979444, "delta": 58040 },
    "by_difficulty": {
      "MASTER": {
        "total_high_score": { "before": 16363921404, "after": 16363979444, "delta": 58040 },
        "average_score": { "before": 1009493.52, "after": 1009499.04, "delta": 5.52 },
        "played_count": { "before": 1621, "after": 1621, "delta": 0 },
        "statistics": {
          "aj": { "before": 1234, "after": 1235, "delta": 1 },
          "fc": { "before": 1366, "after": 1367, "delta": 1 },
          "clr": { "before": 1613, "after": 1621, "delta": 8 },
          "fch": { "before": 133, "after": 133, "delta": 0 },
          "max": { "before": 89, "after": 89, "delta": 0 },
          "sss_plus": { "before": 1347, "after": 1350, "delta": 3 },
          "sss": { "before": 1546, "after": 1548, "delta": 2 },
          "ss_plus": { "before": 1599, "after": 1599, "delta": 0 },
          "ss": { "before": 1621, "after": 1621, "delta": 0 }
        }
      }
    }
  }
}
```

### 4.4 レスポンス例

```json
{
  "player_id": 42,
  "app_ver": "0.1.0",
  "imported_at": "2026-06-04T10:45:00Z",
  "summary": {
    "name": "プレイヤー名",
    "level": 217,
    "rating": 17.29,
    "last_played_at": "2026-06-04T16:42:00+09:00",
    "overpower_value": 96123.91,
    "overpower_percentage": 76.27
  },
  "counts": {
    "full_records_upserted": 1185,
    "worldsend_records_upserted": 120,
    "full_records_skipped": 0,
    "worldsend_records_skipped": 0,
    "honors_skipped": 0,
    "full_records_actually_changed": 12,
    "worldsend_records_actually_changed": 3
  },
  "aggregate_diff": {
    "rating": { "before": 17.28, "after": 17.29, "delta": 0.01 },
    "overpower": {
      "value": { "before": 96110.42, "after": 96123.91, "delta": 13.49 },
      "percentage": { "before": 76.26, "after": 76.27, "delta": 0.01 }
    },
    "total_high_score": { "before": 16363921404, "after": 16363979444, "delta": 58040 },
    "by_difficulty": {
      "MASTER": {
        "total_high_score": { "before": 16363921404, "after": 16363979444, "delta": 58040 },
        "average_score": { "before": 1009493.52, "after": 1009499.04, "delta": 5.52 },
        "played_count": { "before": 1621, "after": 1621, "delta": 0 },
        "statistics": {
          "aj": { "before": 1234, "after": 1235, "delta": 1 },
          "fc": { "before": 1366, "after": 1367, "delta": 1 },
          "clr": { "before": 1613, "after": 1621, "delta": 8 },
          "fch": { "before": 133, "after": 133, "delta": 0 },
          "max": { "before": 89, "after": 89, "delta": 0 },
          "sss_plus": { "before": 1347, "after": 1350, "delta": 3 },
          "sss": { "before": 1546, "after": 1548, "delta": 2 },
          "ss_plus": { "before": 1599, "after": 1599, "delta": 0 },
          "ss": { "before": 1621, "after": 1621, "delta": 0 }
        }
      }
    }
  },
  "changes": [
    {
      "record_type": "full",
      "change_type": "updated",
      "idx": "1234",
      "diff": "MASTER",
      "before": {
        "score": 990000,
        "clear_lamp": "CLEAR",
        "combo_lamp": null,
        "full_chain": null
      },
      "after": {
        "score": 1001000,
        "clear_lamp": "CLEAR",
        "combo_lamp": "FULL COMBO",
        "full_chain": null
      }
    },
    {
      "record_type": "worldsend",
      "change_type": "new",
      "idx": "5678",
      "diff": "WE",
      "before": null,
      "after": {
        "score": 950000,
        "clear_lamp": "CLEAR",
        "combo_lamp": null,
        "full_chain": null
      }
    }
  ],
  "skipped_records": []
}
```

`skipped_records` も0件の場合は空配列 `[]` を返す。

## 5. アーキテクチャ設計

### 5.1 Domain / Repository

既存の `repository.PlayerRecordState` と `repository.WorldsendRecordState` を前回状態・今回状態の比較に流用する。

`internal/domain/repository/player_data_repository.go` に軽量ロード用メソッドを追加する。

```go
type PlayerDataRepository interface {
	LoadMasterData(ctx context.Context, officialIdxList []string) (*PlayerDataMaster, error)
	SavePlayerData(ctx context.Context, exec Executor, input PlayerDataSaveInput) error

	FindPlayerRecordStatesByChartIDs(ctx context.Context, exec Executor, playerID int, chartIDs []int) (map[int]PlayerRecordState, error)
	FindWorldsendRecordStatesByChartIDs(ctx context.Context, exec Executor, playerID int, worldsendChartIDs []int) (map[int]WorldsendRecordState, error)

	GetOverpowerTargetStats(ctx context.Context, filter OverpowerTargetFilter) (*OverpowerTargetStats, error)
	GetOverpowerTargetStatsWithExecutor(ctx context.Context, exec Executor, filter OverpowerTargetFilter) (*OverpowerTargetStats, error)
}
```

キーは以下。

- 通常譜面: `chart_id`
- WORLD'S END: `worldsend_chart_id`

`exec` は `SavePlayerData` と同じく必須にする。保存前状態は登録トランザクション内で読む。

### 5.2 Infra

`internal/infra/repository/player_data_repository_impl.go` に実装する。

通常譜面の保存前 state:

```sql
SELECT
    chart_id,
    score,
    clear_lamp_id,
    combo_lamp_id,
    full_chain_id,
    slot_id,
    slot_order,
    updated_at
FROM player_records
WHERE player_id = ?
  AND chart_id IN (?)
```

WORLD'S END の保存前 state:

```sql
SELECT
    worldsend_chart_id,
    score,
    clear_lamp_id,
    combo_lamp_id,
    full_chain_id,
    updated_at
FROM player_worldsend_records
WHERE player_id = ?
  AND worldsend_chart_id IN (?)
```

明示カラムSELECTを使う。JOINを伴わない単純SELECTで、`IN (?)` は既存の `selectModelsInChunks` と同じ考え方で `info.BulkInsertChunkSize`（3000）単位でチャンク分割する。

### 5.3 Usecase

`PlayerDataUsecase` が差分計算のオーナーである。handlerやinfraへ比較ロジックを置かない。

推奨する内部構成は以下。

- `ensurePlayer` 実行前または直後に保存前プレイヤーサマリーを退避する。RATING は既存 `players.official_player_rating`、OVER POWER は保存前レコードから同一条件で再計算する。
- `applyFullScores` / `applyWorldsendScores` は、従来通り「入力検証・マスタ解決・upsert用state生成」を担当する。
- `normalizeFullRecordsForUpsert` / `normalizeWorldsendRecordsForUpsert` を新設し、同一キーは最後の1件へ正規化する。
- `collectFullChartIDs` / `collectWorldsendChartIDs` を新設し、受理済み upsert list から保存前ロード対象キーを作る。
- `computeFullRecordChanges` / `computeWorldsendRecordChanges` を新設し、保存前 state map と upsert 予定値を比較する。
- `sortAndLimitRecordChanges` を新設し、`idx` を数値として昇順に並べたうえでレスポンス詳細を最大100件に制限する。同一 `idx` の場合は `record_type`、`diff` の順で安定させる。
- `applyScores` は upsert list生成後に正規化し、保存前stateをロードして changes を計算し、`SavePlayerData` 成功後に counts / changes を返す。
- `buildPlayerDataAggregateSnapshot` を新設し、通常譜面レコード一覧から TOTAL HIGH SCORE、難易度別平均、ランプ・スコアランク別統計を作る。
- `computePlayerDataAggregateDiff` を新設し、保存前 snapshot と保存後 snapshot を比較して `aggregate_diff` を作る。

`applyScores` の戻り値は以下のように変更する。

```go
func (us *playerDataUsecase) applyScores(
	ctx context.Context,
	tx repository.Executor,
	playerID int,
	scores PlayerDataScorePayload,
	masters *playerDataMaster,
	updatedAt time.Time,
) (api_internal.PlayerDataCounts, []api_internal.SkippedRecord, []api_internal.PlayerDataRecordChange, calculatedOverpowerSummary, error)
```

`Register` 側では `SavePlayerData` 成功後に `result.Changes = changes` を設定する。差分0件の場合も空sliceを設定し、JSONでは `changes: []` として返す。

正規化後の upsert 予定値だけを差分比較する。DB側で `updated_at` が更新される条件と、Go側の `new` / `updated` 判定対象を一致させるためである。

集計差分は、保存前レコード一覧と保存後レコード一覧の両方から snapshot を作って比較する。保存前 snapshot は `SavePlayerData` より前に取得した通常譜面レコードから作り、保存後 snapshot は `SavePlayerData` 後、OVER POWER / rating 再計算後に取得した通常譜面レコードとプレイヤー情報から作る。

OVER POWER の before/after は同じ `OverpowerTargetStats` と同じ未解禁曲設定を使って計算する。これにより、登録中にマスタや未解禁曲設定が変化しない限り、`delta` はスコア登録による差分として解釈できる。

TOTAL HIGH SCORE と RECORD STATISTICS は、通常譜面のみを対象にする。WORLD'S END を含めるかどうかは将来拡張で別フィールドとして検討する。

### 5.4 差分判定関数

DBの `changedCondition` と同期するため、比較関数は小さく明示的に書く。

```go
func playerRecordMeaningfullyChanged(before, after repository.PlayerRecordState) bool {
	return before.Score != after.Score ||
		before.ClearLampID != after.ClearLampID ||
		before.ComboLampID != after.ComboLampID ||
		before.FullChainID != after.FullChainID
}

func worldsendRecordMeaningfullyChanged(before, after repository.WorldsendRecordState) bool {
	return before.Score != after.Score ||
		before.ClearLampID != after.ClearLampID ||
		before.ComboLampID != after.ComboLampID ||
		before.FullChainID != after.FullChainID
}
```

この関数には、DB側の `fullRecordChangedCondition` / `worldsendRecordChangedCondition` と同じ条件であることを日本語コメントで明記する。

## 6. パフォーマンス設計

- 追加ロード処理は通常譜面の保存前、WORLD'S ENDの保存前の合計2種類。
- どちらも `player_id` 条件と対象キー条件の単純SELECTで、JOINを伴わない。チャンク分割するため、実際のクエリ数は `ceil(keys / BulkInsertChunkSize)` に応じて増える。譜面7000件規模では `BulkInsertChunkSize`（3000）により種別あたり最大3クエリとなる。
- 比較は payload 件数に対する O(N)。
- 楽曲1700件・譜面7000件規模でも、リクエスト単位の map と slice のメモリ使用量は許容範囲である。マスタは payload 参照 idx のみ、保存前 state は upsert 対象キーのみをロードする。
- 初回登録では実際の差分件数が数千件になる可能性があるため、`*_actually_changed` は全件数を返し、`changes` の詳細は最大100件に制限する。

差分詳細の上限は初版から100件とする。レスポンスサイズとクライアント処理負荷を抑えるためである。詳細は `idx` を数値として昇順に並べ、同一 `idx` の場合は `record_type`、`diff` の順で安定させる。

## 7. テスト計画

### 7.1 Usecaseテスト

`internal/usecase/player_data_usecase_apply_scores_test.go` を中心に追加する。

優先ケース:

- 前回未登録の通常譜面は `new` になり、`full_records_actually_changed` が増える。
- 前回未登録の WORLD'S END は `new` になり、`worldsend_records_actually_changed` が増える。
- score のみ変化したら `updated`。
- `clear_lamp_id` のみ変化したら `updated`。
- `combo_lamp_id` のみ変化したら `updated`。
- `full_chain_id` のみ変化したら `updated`。
- score / lamp が同一で slot / order だけ違う場合は `changes` が0件になる。
- スキップされた入力は `skipped_records` に反映される。
- full と worldsend の件数が独立して集計される。
- payload内で同一キーが重複した場合、最後の1件だけが保存・差分詳細の対象になる。
- `new` の場合、`before` は `null` になる。
- `changes` は `idx` を数値として昇順に並べ、最大100件まで返る。
- `changes` が100件を超える場合も、`*_actually_changed` は全件数を返す。
- ランプ名が `clear_lamp` / `combo_lamp` / `full_chain` に設定され、`none` 相当は `null` になる。
- `SavePlayerData` 成功時のみ `changes` が返る。
- `aggregate_diff.rating` は保存前 `official_player_rating` と payload の `rating` の差分になる。
- `aggregate_diff.overpower.value` / `percentage` は保存前後レコードを同じ分母で再計算した差分になる。
- `aggregate_diff.total_high_score` は通常譜面スコア合計の before / after / delta を返す。
- `aggregate_diff.by_difficulty.*.total_high_score` は難易度別スコア合計の差分を返す。
- `aggregate_diff.by_difficulty.*.average_score` は難易度別平均スコアの差分を返し、played_count が0の場合は `null` になる。
- `aggregate_diff.by_difficulty.*.statistics.aj/fc/clr/fch/max` はランプ・MAX達成数の差分を返す。
- `aggregate_diff.by_difficulty.*.statistics.sss_plus/sss/ss_plus/ss` は累積スコアランク件数の差分を返す。
- スコアやランプが下がった場合、各 `delta` は負数になる。
- WORLD'S END は初版の TOTAL HIGH SCORE / RECORD STATISTICS に含まれない。

テストはテーブルテスト + Given / When / Then コメントで書き、結果検証は `assert`、前提確認は `require` を使う。

### 7.2 Repositoryテスト

`internal/infra/repository/player_data_repository_impl_test.go` に軽量ロードのテストを追加する。

- `player_records` から保存前stateの必要カラムだけを読み、`chart_id` keyed map を返す。
- `player_worldsend_records` から保存前stateの必要カラムだけを読み、`worldsend_chart_id` keyed map を返す。
- 対象プレイヤーが未登録状態の場合は空mapを返す。
- `exec == nil` の場合はエラーを返す。

### 7.3 DB条件との同期テスト

既存の `fullRecordChangedCondition` / `worldsendRecordChangedCondition` の文字列テストに加え、Go側の差分判定関数のテストを追加する。

完全な機械同期は難しいが、以下の観点を固定テストにする。

- score差分あり
- clear lamp差分あり
- combo lamp差分あり
- full chain差分あり
- slot差分のみ
- updated_at差分のみ
- 全値同一

## 8. 実装タスク

1. DTO追加: `PlayerDataCounts` に `*_actually_changed`、`PlayerDataResult` に `changes`、差分用DTOを追加。
2. Repository interface追加: 保存前キー指定ロードの2メソッド。
3. Infra実装: 明示カラムSELECTで保存前state mapを返す。
4. Usecaseテスト追加: 差分判定・counts・changesの期待値を先に書く。
5. Usecase実装: upsert対象の重複正規化、保存前stateロード、upsert予定値との比較、`sortAndLimitRecordChanges`、保存成功後の result 反映。
6. DTO追加: `PlayerDataResult` に `aggregate_diff`、集計差分用DTO（`IntDiff` / `NullableFloatDiff` / 難易度別統計）を追加。
7. Usecaseテスト追加: RATING、OVER POWER、TOTAL HIGH SCORE、難易度別平均、ランプ・スコアランク別統計の before / after / delta を先に書く。
8. Usecase実装: 保存前後の通常譜面レコードから集計 snapshot を作り、`aggregate_diff` を計算する。
9. API.md更新: レスポンス例、スキーマ、TypeScript interface、差分定義のNoteを更新。
10. `gofmt -s -w` を対象Goファイルに実行。
11. `go test ./...` を実行。
12. セルフレビュー: AGENTS.md のチェックリストに沿って、文字化け・N+1・実装範囲・API.md反映を確認。

Goコードを変更した実装PRでは `go test ./...` を実行する。

## 9. 初版仕様とリスク

### 9.1 初版仕様

- `changes` は登録成功時に常に配列として返し、最大100件とする。
- `aggregate_diff` は登録成功時に常にオブジェクトとして返す。
- `changes` は `idx` を数値として昇順に並べる。
- `new` の `before` は `null` とする。
- `before` / `after` のランプは `clear_lamp` / `combo_lamp` / `full_chain` とし、マスタ `Name` を返す。`none` 相当は `null` とする。
- WORLD'S END の `diff` は `"WE"` 固定とする。
- payload内重複は最後の1件へ正規化する。
- `SavePlayerData` 成功後に `result.Changes` を設定する。
- 集計差分の RATING は公式レーティング、OVER POWER は同一分母で再計算した値、TOTAL HIGH SCORE / RECORD STATISTICS は通常譜面のみを対象にする。

### 9.2 リスク

- DB側条件とGo側条件が将来ずれる。
- 初回登録で実際の差分件数が数千件になった場合、`changes` 詳細は100件に制限されるため、クライアントは `*_actually_changed` と `changes.length` が一致しない前提で扱う必要がある。
- before/afterにランプ名を含めるため、マスタ逆引きの実装が少し増える。
- 「改善」という表示文言にすると、スコア下降やランプ下降を誤表現する可能性がある。
- OVER POWER の `before` を再計算するため、保存前レコード一覧の取得・変換コストが増える。
- RATING は公式レーティングと計算レーティングが別物であるため、APIドキュメントで `aggregate_diff.rating` の対象を明確にしないと誤解される。
- TOTAL HIGH SCORE / RECORD STATISTICS は WORLD'S END を初版では含めないため、クライアント表示で対象範囲を明記する必要がある。
- ランプやスコアランクの集計は累積件数であり、排他的な分布ではない。UI側のラベルと説明に注意する。

### 9.3 将来拡張

- `change_fields: ["score", "combo_lamp"]` のような変更カラム一覧。
- 譜面単位の `score_delta`、rating delta、overpower delta。
- 計算レーティング（`calculated_player_rating` / `best_average_rating` / `new_average_rating`）の差分。
- WORLD'S END を含めた TOTAL HIGH SCORE / RECORD STATISTICS。
- `?include_changes=false` のような軽量モード。
- 変更履歴テーブルへの保存。

## 10. 関連ドキュメント更新

実装時に必ず `docs/API.md` を更新する。

更新対象:

- `/internal/me/register-data` のレスポンス例
- `PlayerDataResult` レスポンススキーマ
- `PlayerDataCounts` 説明
- `aggregate_diff` のスキーマ
- `changes` のスキーマ
- TypeScript interface
- `changes` が最大100件で、`*_actually_changed` は全件数であることの説明
- `aggregate_diff.rating` は公式レーティングであることの説明
- `aggregate_diff.overpower` は同一分母で再計算した差分であることの説明
- TOTAL HIGH SCORE / RECORD STATISTICS は通常譜面のみ対象であることの説明
- `before` が `null` になり得ることの説明
- ランプ名フィールドの説明
- 差分情報を返す仕様Noteへの置換

`/internal/player-data/commit` は `/internal/me/register-data` と同じ `PlayerDataResult` を返す。差分も含まれることを一文補足する。
