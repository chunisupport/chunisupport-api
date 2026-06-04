# プレイヤーデータ登録時差分計算機能 設計書

作成日: 2026-06-04  
最終更新日: 2026-06-04

## 0. 位置づけ

本設計書は、プレイヤーデータ登録時に「今回の登録で実際に変化したスコア」をレスポンスへ含めるための未実装機能の設計である。

現在の登録エンドポイントは以下の通り。

- `POST /internal/me/register-data`
- `POST /internal/player-data/temp` + `POST /internal/player-data/commit`

`/internal/player-data/commit` は一時保存済み本文を `PlayerDataPayload` として解釈し、最終的に通常の `PlayerDataUsecase.Register` を通る。そのため、差分計算は `PlayerDataUsecase.Register` 配下に実装すれば、直接登録と一時保存確定の両方を同じ仕様で扱える。

現行実装では、`PlayerDataPayload.scores.full` / `scores.worldsend` に含まれる全譜面の現在ベストを受け取り、`applyScores` で `PlayerRecordForUpsert` / `WorldsendRecordForUpsert` に変換し、`playerDataRepo.SavePlayerData` で bulk upsert している。`PlayerDataCounts` の `*_upserted` は「payload 内で処理対象になった件数」であり、DB上で実際に値が変わった件数ではない。

また、`player_records.updated_at` / `player_worldsend_records.updated_at` はサーバの登録時刻ではなく、payload の `updated_at` を格納する。現在も `ON DUPLICATE KEY UPDATE` 内で、score / lamp が変化した場合にのみ `updated_at` を更新しているが、その変化内容はレスポンスに含まれていない。

## 1. 目的

プレイヤーデータ登録完了時に、サーバ側で前回状態と今回payloadを比較し、実際に変化したレコード数と変更内容を返す。

目的は以下。

- クライアントが「今回の登録でスコアが何件変わったか」「どの譜面が新規登録または更新されたか」を即座に表示できるようにする。
- 複数端末・複数クライアント間の状態差による比較ミスを避ける。
- `updated_at` の時刻フィルタに頼らず、DBの現在状態とpayloadから生成したupsert対象を直接比較する。
- 既存の登録セマンティクス、トランザクション境界、rating / overpower 再計算を維持する。

本機能で扱う「差分」は、ユーザー向けには「更新差分」と呼ぶが、内部仕様上は「改善」だけではなく「値の変化」を意味する。再取り込みや公式側データ訂正によりスコア・ランプが下がった場合も、DBに保存される値が変わるため差分に含める。

## 2. 採用方針

### 2.1 採用する方式

登録トランザクション内で、保存前の軽量な現在状態を読み込み、payloadから作った upsert 対象と比較する。

```text
tx 開始
  masters ロード
  ensurePlayer
  applyHonors
  counts, skipped, changes, overpower := applyScores(...)
    applyScores内で保存前stateを軽量ロード
    payloadをupsert対象へ変換
    upsert対象と保存前stateを比較
    SavePlayerData
  overpower / rating 再計算
  result.Counts / result.Changes を設定
tx commit
```

前回状態のロードは `SavePlayerData` より前に行う。保存後に読むと before 値が失われるためである。

### 2.2 不採用: `updated_at` の最近値フィルタ

「登録後に `updated_at` が直近10分以内のレコードを読む」方式は採用しない。

- `updated_at` は payload 由来であり、サーバの登録時刻ではない。
- 10分という閾値はクライアント時計、遅延、再送、同時登録で破綻しやすい。
- before 値を返すには結局保存前状態が必要になる。
- exact timestamp と chart_id の IN 条件を組み合わせても、時刻依存の脆さが残る。

保存前状態を読むなら、その状態を使ってアプリケーション層で直接比較するのが最も単純で正確である。

## 3. 差分の定義

### 3.1 差分対象カラム

通常譜面、WORLD'S END ともに、以下のいずれかが異なる場合に `updated` とする。

- `score`
- `clear_lamp_id`
- `combo_lamp_id`
- `full_chain_id`

前回レコードが存在しない場合は `new` とする。

この条件は `internal/infra/repository/player_data_repository_impl.go` の `fullRecordChangedCondition` / `worldsendRecordChangedCondition` と一致させる。Go側の差分判定は、DB側の `updated_at = IF(...)` 条件と同期していなければならない。

### 3.2 差分対象外

以下は初版では差分対象外とする。

- `slot_id`
- `slot_order`
- `updated_at`
- 楽曲名、譜面定数、スロット分類などの表示用派生情報
- 称号、プレイヤー名、レベル、rating、overpower

`slot_id` / `slot_order` はDBでは常に上書きされるが、現行の `updated_at` 更新条件には含まれていない。初版ではDBの意味的更新条件に合わせ、ユーザー向け差分にも含めない。

### 3.3 スキップとの関係

マスタ解決失敗、スコア範囲外、ランプ変換失敗などで `SkippedRecord` になった入力は差分に含めない。差分計算は、既存の `applyFullScores` / `applyWorldsendScores` が upsert 対象として受理したレコードだけに対して行う。

### 3.4 payload内重複

通常の入力では、同一譜面は payload 内に1件だけ存在する前提とする。

同一 `chart_id` または `worldsend_chart_id` が同一payload内に複数回現れた場合、初版では既存の保存処理と同じく入力順に upsert 対象へ積む。ただし、差分レスポンスに同一譜面が複数回出るとクライアント表示が不安定になるため、実装時には以下のどちらかを明確に選ぶ。

- 推奨: upsert 対象生成後、`chart_id` / `worldsend_chart_id` ごとに最後の1件へ正規化してから保存・差分計算する。
- 最小変更: 既存保存挙動を維持し、重複入力は未定義として扱う。テストでは重複を正常系に含めない。

将来の保守性を考えると、前者を別タスクで実施する余地がある。

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

`*_actually_changed` は `new` と `updated` の合計である。`*_upserted` から `*_skipped` を引いた件数とは一致しない。同一値の再登録は upsert 対象になっても `actually_changed` には含めない。

### 4.2 Changes

`PlayerDataResult` に `changes` を追加する。

```go
type PlayerDataResult struct {
	PlayerID       int                      `json:"player_id"`
	AppVersion     string                   `json:"app_ver"`
	ImportedAt     time.Time                `json:"imported_at"`
	Summary        PlayerDataSummary        `json:"summary"`
	Counts         PlayerDataCounts         `json:"counts"`
	Changes        []PlayerDataRecordChange `json:"changes,omitempty"`
	SkippedRecords []SkippedRecord          `json:"skipped_records,omitempty"`
}
```

差分詳細は入力DTOの `PlayerDataScoreEntry` をそのまま返さず、比較対象カラムだけを持つ専用DTOにする。理由は、入力DTOには `slot` / `order` が含まれており、初版の差分対象と誤解されやすいためである。

```go
type PlayerDataRecordChange struct {
	RecordType string                 `json:"record_type"` // "full" | "worldsend"
	ChangeType string                 `json:"change_type"` // "new" | "updated"
	Idx        string                 `json:"idx"`
	Diff       string                 `json:"diff,omitempty"`
	Before     *PlayerDataRecordState `json:"before,omitempty"`
	After      PlayerDataRecordState  `json:"after"`
}

type PlayerDataRecordState struct {
	Score       int     `json:"score"`
	ClearLampID int     `json:"clear_lamp_id"`
	ComboLampID int     `json:"combo_lamp_id"`
	FullChainID int     `json:"full_chain_id"`
	ClearLamp   *string `json:"clear_lamp,omitempty"`
	ComboLamp   *string `json:"combo_lamp,omitempty"`
	FullChain   *string `json:"full_chain,omitempty"`
}
```

IDフィールドは必須、名前フィールドは任意とする。初版では、UIがすぐ表示しやすいように `ClearLamp` / `ComboLamp` / `FullChain` も返す方針を推奨する。名前の復元は `masters` から逆引きする。逆引き実装が過剰になる場合はIDのみで初版を出し、API.mdに明記する。

WORLD'S END の `diff` は入力値に依存せず、レスポンスでは省略または `"WE"` に統一する。クライアントの扱いやすさを優先するなら `"WE"` 固定を推奨する。

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
        "combo_lamp_id": 1,
        "full_chain_id": 1,
        "clear_lamp": "CLEAR",
        "combo_lamp": "NONE",
        "full_chain": "NONE"
      },
      "after": {
        "score": 1001000,
        "clear_lamp_id": 2,
        "combo_lamp_id": 3,
        "full_chain_id": 1,
        "clear_lamp": "CLEAR",
        "combo_lamp": "ALL JUSTICE",
        "full_chain": "NONE"
      }
    },
    {
      "record_type": "worldsend",
      "change_type": "new",
      "idx": "5678",
      "diff": "WE",
      "after": {
        "score": 950000,
        "clear_lamp_id": 2,
        "combo_lamp_id": 1,
        "full_chain_id": 1,
        "clear_lamp": "CLEAR",
        "combo_lamp": "NONE",
        "full_chain": "NONE"
      }
    }
  ]
}
```

`changes` は0件の場合、省略する。`skipped_records` も既存実装と同様に0件時は省略する。

## 5. アーキテクチャ設計

### 5.1 Domain / Repository

既存の `repository.PlayerRecordState` と `repository.WorldsendRecordState` を前回状態・今回状態の比較に流用する。

`internal/domain/repository/player_data_repository.go` に軽量ロード用メソッドを追加する。

```go
type PlayerDataRepository interface {
	LoadMasterData(ctx context.Context, officialIdxList []string) (*PlayerDataMaster, error)
	SavePlayerData(ctx context.Context, exec Executor, input PlayerDataSaveInput) error

	FindCurrentPlayerRecordStates(ctx context.Context, exec Executor, playerID int) (map[int]PlayerRecordState, error)
	FindCurrentWorldsendRecordStates(ctx context.Context, exec Executor, playerID int) (map[int]WorldsendRecordState, error)

	GetOverpowerTargetStats(ctx context.Context, filter OverpowerTargetFilter) (*OverpowerTargetStats, error)
	GetOverpowerTargetStatsWithExecutor(ctx context.Context, exec Executor, filter OverpowerTargetFilter) (*OverpowerTargetStats, error)
}
```

キーは以下。

- 通常譜面: `chart_id`
- WORLD'S END: `worldsend_chart_id`

`exec` は nil を許容しない方針を推奨する。保存前状態は登録トランザクション内で読む必要があり、呼び出し側の誤用を早期に検出できるためである。`SavePlayerData` と同じく nil の場合は repository error を返す。

### 5.2 Infra

`internal/infra/repository/player_data_repository_impl.go` に実装する。

通常譜面:

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
```

WORLD'S END:

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
```

`SELECT *` は使用しない。JOINもしない。N+1を避けるため、それぞれ1クエリで全件をmapへ詰める。

### 5.3 Usecase

`PlayerDataUsecase` が差分計算のオーナーである。handlerやinfraへ比較ロジックを置かない。

推奨する内部構成は以下。

- `applyFullScores` / `applyWorldsendScores` は、従来通り「入力検証・マスタ解決・upsert用state生成」を担当する。
- `computeFullRecordChanges` / `computeWorldsendRecordChanges` を新設し、受理済み upsert list と前回state mapを比較する。
- `applyScores` は前回stateをロードし、upsert list生成後、保存前に差分を計算し、保存後に counts / changes を返す。

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

`Register` 側では `result.Changes = changes` を設定し、0件なら空のままにする。

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

- 追加クエリは通常譜面1回、WORLD'S END1回の合計2回。
- どちらも `player_id` 条件の単純SELECTで、JOINしない。
- 比較は payload 件数に対する O(N)。
- 1万件規模でも map と slice のメモリ使用量は許容範囲。
- 初回登録では `changes` が全件分になる可能性がある。これは既存のプロフィール系APIで全件レコードを返している規模と同程度だが、実運用で重ければ将来 `changes` 省略フラグを追加する。

差分詳細の上限は初版では設けない。上限を設けると counts と details の不一致説明が必要になり、APIが複雑になるためである。

## 7. テスト計画

### 7.1 Usecaseテスト

`internal/usecase/player_data_usecase_apply_scores_test.go` を中心に追加する。

優先ケース:

- 前回なしの通常譜面は `new` になり、`full_records_actually_changed` が増える。
- 前回なしの WORLD'S END は `new` になり、`worldsend_records_actually_changed` が増える。
- score のみ変化したら `updated`。
- `clear_lamp_id` のみ変化したら `updated`。
- `combo_lamp_id` のみ変化したら `updated`。
- `full_chain_id` のみ変化したら `updated`。
- score / lamp が同一で slot / order だけ違う場合は差分なし。
- スキップされた入力は `changes` に入らない。
- full と worldsend の件数が独立して集計される。

テストはテーブルテスト + Given / When / Then コメントで書き、結果検証は `assert`、前提確認は `require` を使う。

### 7.2 Repositoryテスト

`internal/infra/repository/player_data_repository_impl_test.go` に軽量ロードのテストを追加する。

- `player_records` から必要カラムだけを読み、`chart_id` keyed map を返す。
- `player_worldsend_records` から必要カラムだけを読み、`worldsend_chart_id` keyed map を返す。
- 対象プレイヤーにレコードがない場合は空mapを返す。
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
2. Repository interface追加: `FindCurrentPlayerRecordStates` / `FindCurrentWorldsendRecordStates`。
3. Infra実装: 明示カラムSELECTで現在state mapを返す。
4. Usecaseテスト追加: 差分判定・counts・changesの期待値を先に書く。
5. Usecase実装: 保存前stateロード、upsert listとの比較、resultへの反映。
6. API.md更新: レスポンス例、スキーマ、TypeScript interface、差分定義のNoteを更新。
7. `gofmt -s -w` を対象Goファイルに実行。
8. `go test ./...` を実行。
9. セルフレビュー: AGENTS.md のチェックリストに沿って、文字化け・N+1・不要実装・API.md反映を確認。

ドキュメントのみを変更する場合は `go test ./...` は必須ではない。ただしGoコードを変更した実装PRでは必須とする。

## 9. リスクと未決事項

### 9.1 初版で決めるべき事項

- ランプ名を `changes` に含めるか、IDだけにするか。
- WORLD'S END の `diff` を省略するか、`"WE"` 固定にするか。
- payload内重複を正規化するか、未定義として扱うか。

推奨は「ランプ名を含める」「WORLD'S ENDは `diff: "WE"` 固定」「payload内重複は別タスクで正規化検討」である。

### 9.2 リスク

- DB側条件とGo側条件が将来ずれる。
- 初回登録で `changes` が数千件になり、クライアント処理が重くなる。
- before/afterにランプ名を含める場合、マスタ逆引きの実装が少し増える。
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
- 「差分情報は返却されません」という現行Noteの削除・置換

`/internal/player-data/commit` は `/internal/me/register-data` と同じ `PlayerDataResult` を返すため、個別の詳細更新は不要。ただし、差分も含まれることを一文補足する。
