# プレイヤーデータ登録時差分計算機能 設計書

作成日: 2026-06-04  
最終更新日: 2026-06-08

## 0. 位置づけ

本設計書は、プレイヤーデータ登録時に「今回の登録で実際に変化したスコア」をレスポンスへ含めるための未実装機能の設計である。

現在の登録エンドポイントは以下の通り。

- `POST /internal/me/register-data`
- `POST /internal/player-data/temp` + `POST /internal/player-data/commit`

`/internal/player-data/commit` は一時保存済み本文を `PlayerDataPayload` として解釈し、最終的に通常の `PlayerDataUsecase.Register` を通る。そのため、差分計算は `PlayerDataUsecase.Register` 配下に実装すれば、直接登録と一時保存確定の両方を同じ仕様で扱える。

現行実装では、`PlayerDataPayload.scores.full` / `scores.worldsend` に含まれる全譜面の現在ベストを受け取り、`applyScores` で `PlayerRecordForUpsert` / `WorldsendRecordForUpsert` に変換し、`playerDataRepo.SavePlayerData` で bulk upsert している。`PlayerDataCounts` の `*_upserted` は「payload 内で処理対象になった件数」を表す。

また、`player_records.updated_at` / `player_worldsend_records.updated_at` は payload の `updated_at` を格納する。現在も `ON DUPLICATE KEY UPDATE` 内で、score / lamp が変化した場合に `updated_at` を更新している。

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

payload 直下の `updated_at` は保存値として利用するが、差分候補抽出キーには使わない。秒精度の衝突や再送による誤検出を避けるため、差分は保存前 state と upsert 予定値の比較だけで判定する。

`updated_at` はDB保存値とGo側の値がずれないよう、`time.Parse` 後に `updatedAt = updatedAt.Truncate(time.Second)` で秒精度へ正規化し、その値を `PlayerRecordState.UpdatedAt` / `WorldsendRecordState.UpdatedAt` に設定する。

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
  result.Counts / result.Changes を設定
tx commit
```

保存前状態のロードは `SavePlayerData` より前に行う。差分判定はDB側の `updated_at = IF(...)` 条件と同じ比較カラムで行い、`SavePlayerData` が成功した場合だけレスポンスへ反映される。

同じ `updated_at` を持つ payload が再送された場合も、保存前stateとupsert予定値が同じなら `changes` には含めない。今回の保存処理で値が変化するレコードだけを `changes` に含める。

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
	Changes        []PlayerDataRecordChange `json:"changes"`
	SkippedRecords []SkippedRecord          `json:"skipped_records"`
}
```

差分詳細は入力DTOの `PlayerDataScoreEntry` をそのまま返さず、比較対象カラムだけを持つ専用DTOにする。理由は、入力DTOには `slot` / `order` が含まれており、初版の差分対象と誤解されやすいためである。

レスポンスでは基本的に `omitempty` を使わず、値がない場合は `null` または空配列として明示する。`changes` は差分が0件の場合も空配列 `[]` を返す。`before` は `new` の場合に `null`、`updated` の場合に変更前状態を返す。

`changes` の詳細は最大100件まで返す。`*_actually_changed` は実際に変化した全件数を表し、`changes` の件数とは一致しない場合がある。詳細に含める100件は `idx` を数値として昇順に並べて選ぶ。同一 `idx` の場合は `record_type`、`diff` の順で安定させる。通常譜面と WORLD'S END の両方に差分がある場合も、同一レスポンス内の `changes` 全体で最大100件とする。

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
	Score         int    `json:"score"`
	ClearLampID   int    `json:"clear_lamp_id"`
	ClearLampName string `json:"clear_lamp_name"`
	ComboLampID   int    `json:"combo_lamp_id"`
	ComboLampName string `json:"combo_lamp_name"`
	FullChainID   int    `json:"full_chain_id"`
	FullChainName string `json:"full_chain_name"`
}
```

フロントエンド側でランプIDから名前を解決しなくてよいよう、初版から `clear_lamp_name` / `combo_lamp_name` / `full_chain_name` を返す。名前は登録処理でロード済みの `masters` から逆引きして設定する。IDはDB保存値との対応確認や将来の互換性のため残す。

WORLD'S END の `diff` は入力値に依存せず、レスポンスでは `"WE"` 固定にする。

### 4.3 レスポンス例

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
  "changes": [
    {
      "record_type": "full",
      "change_type": "updated",
      "idx": "1234",
      "diff": "MASTER",
      "before": {
        "score": 990000,
        "clear_lamp_id": 2,
        "clear_lamp_name": "CLEAR",
        "combo_lamp_id": 1,
        "combo_lamp_name": "NONE",
        "full_chain_id": 1,
        "full_chain_name": "NONE"
      },
      "after": {
        "score": 1001000,
        "clear_lamp_id": 2,
        "clear_lamp_name": "CLEAR",
        "combo_lamp_id": 3,
        "combo_lamp_name": "FULL COMBO",
        "full_chain_id": 1,
        "full_chain_name": "NONE"
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
        "clear_lamp_id": 2,
        "clear_lamp_name": "CLEAR",
        "combo_lamp_id": 1,
        "combo_lamp_name": "NONE",
        "full_chain_id": 1,
        "full_chain_name": "NONE"
      }
    }
  ]
}
```

`changes` は0件の場合も省略せず、空配列 `[]` を返す。`skipped_records` も省略せず、0件の場合は空配列 `[]` を返す。

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

明示カラムSELECTを使う。JOINを伴わない単純SELECTで、`IN (?)` は既存の `selectModelsInChunks` と同じ考え方でチャンク分割する。

### 5.3 Usecase

`PlayerDataUsecase` が差分計算のオーナーである。handlerやinfraへ比較ロジックを置かない。

推奨する内部構成は以下。

- `applyFullScores` / `applyWorldsendScores` は、従来通り「入力検証・マスタ解決・upsert用state生成」を担当する。
- `normalizeFullRecordsForUpsert` / `normalizeWorldsendRecordsForUpsert` を新設し、同一キーは最後の1件へ正規化する。
- `collectFullChartIDs` / `collectWorldsendChartIDs` を新設し、受理済み upsert list から保存前ロード対象キーを作る。
- `computeFullRecordChanges` / `computeWorldsendRecordChanges` を新設し、保存前 state map と upsert 予定値を比較する。
- `sortAndLimitRecordChanges` を新設し、`idx` を数値として昇順に並べたうえでレスポンス詳細を最大100件に制限する。同一 `idx` の場合は `record_type`、`diff` の順で安定させる。
- `applyScores` は upsert list生成後に正規化し、保存前stateをロードして changes を計算し、保存成功後に counts / changes を返す。

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

`Register` 側では `result.Changes = changes` を設定する。0件の場合も `nil` ではなく空sliceを設定し、JSONでは `changes: []` として返す。

正規化後の upsert 予定値だけを差分比較する。DB側で `updated_at` が更新される条件と、Go側の `new` / `updated` 判定対象を一致させるためである。

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
- どちらも `player_id` 条件と対象キー条件の単純SELECTで、JOINを伴わない。チャンク分割するため、実際のクエリ数は `ceil(keys / BulkInsertChunkSize)` に応じて増える。
- 比較は payload 件数に対する O(N)。
- 1万件規模でも map と slice のメモリ使用量は許容範囲。
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
- ランプIDに対応するランプ名が `clear_lamp_name` / `combo_lamp_name` / `full_chain_name` に設定される。

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
5. Usecase実装: `updatedAt` 秒精度正規化、upsert対象の重複正規化、保存前stateロード、upsert予定値との比較、resultへの反映。
6. API.md更新: レスポンス例、スキーマ、TypeScript interface、差分定義のNoteを更新。
7. `gofmt -s -w` を対象Goファイルに実行。
8. `go test ./...` を実行。
9. セルフレビュー: AGENTS.md のチェックリストに沿って、文字化け・N+1・実装範囲・API.md反映を確認。

Goコードを変更した実装PRでは `go test ./...` を実行する。

## 9. リスクと未決事項

### 9.1 初版で決めるべき事項

初版方針は「後方互換性は重視しない」「基本的に `omitempty` は使わない」「`changes` は最大100件」「`changes` は `idx` を数値として昇順」「`new` の `before` は `null`」「ランプ名はIDとあわせて返す」「WORLD'S ENDは `diff: "WE"` 固定」「payload内重複は最後の1件へ正規化」で固定する。

### 9.2 リスク

- DB側条件とGo側条件が将来ずれる。
- 初回登録で実際の差分件数が数千件になった場合、`changes` 詳細は100件に制限されるため、クライアントは `*_actually_changed` と `changes.length` が一致しない前提で扱う必要がある。
- before/afterにランプ名を含めるため、マスタ逆引きの実装が少し増える。
- 「改善」という表示文言にすると、スコア下降やランプ下降を誤表現する可能性がある。

### 9.3 将来拡張

- `change_fields: ["score", "combo_lamp"]` のような変更カラム一覧。
- `score_delta`、rating delta、overpower delta。
- `?include_changes=false` のような軽量モード。
- 変更履歴テーブルへの保存。

## 10. 関連ドキュメント更新

実装時に必ず `docs/API.md` を更新する。

更新対象:

- `/internal/me/register-data` のレスポンス例
- `PlayerDataResult` レスポンススキーマ
- `PlayerDataCounts` 説明
- `changes` のスキーマ
- TypeScript interface
- `changes` が最大100件で、`*_actually_changed` は全件数であることの説明
- `before` が `null` になり得ることの説明
- ランプ名フィールドの説明
- 差分情報を返す仕様Noteへの置換

`/internal/player-data/commit` は `/internal/me/register-data` と同じ `PlayerDataResult` を返す。差分も含まれることを一文補足する。
