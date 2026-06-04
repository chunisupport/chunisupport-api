# プレイヤーデータ登録時差分計算機能 設計書

作成日: 2026-06-04

## 0. 現行実装との差分（2026-06-04時点）

本設計書は **未実装の機能** に対する設計・計画です。

現在のプレイヤーデータ登録（スコアデータ登録）処理は以下の通りです。

- エンドポイント
  - `POST /internal/me/player-data`（直接登録）
  - `POST /internal/player-data/temp` + `POST /internal/player-data/commit`（一時保存→確定フロー）
- `PlayerDataPayload.scores`（full + worldsend）でクライアントから全譜面の現在ベストを送信（理論値現在約7000件、将来1万件程度想定）。
- `usecase/player_data_usecase_impl.go` の `Register` → `applyScores` → `applyFullScores` / `applyWorldsendScores` で payload を検証・変換し、`PlayerRecordForUpsert` / `WorldsendRecordForUpsert` リストを作成。
- `playerDataRepo.SavePlayerData` で `player_records` / `player_worldsend_records` に対して `INSERT ... ON DUPLICATE KEY UPDATE` を実行。
  - `updated_at` の更新は `IF(score <> VALUES(score) OR clear_lamp_id <> ... OR ...)` の条件でのみ行われる（`fullRecordChangedCondition` / `worldsendRecordChangedCondition` 参照）。
  - それ以外のカラム（score, lamps, slot など）は常に上書き。
- レスポンス `PlayerDataResult`（`api_internal`）は以下のみ:
  - `counts`: `full_records_upserted`（payload 内の件数）、`full_records_skipped` など（「処理しようとした件数」であって「実際に値が更新された件数」ではない）。
  - `skipped_records`: 検証失敗などでスキップされたもの。
  - `summary`: 再計算された overpower など。
- 差分（この登録で何が「改善・新規」になったか）は一切計算・返却されていない。
- 登録後、overpower/rating 再計算のために `FindByPlayerID` で全レコードを再取得している（JOIN 重め）。

クライアントはこれまで「登録した」という事実と upsert 件数しか得られず、「今回のプレイ/データ更新で何が変わったか」をサーバに聞くことができなかった。

---

## 1. 目的

スコアデータ登録完了時に「更新差分」をサーバ側で計算し、レスポンスに含めてクライアントに返す機能を提供する。

- クライアントが「この登録で X 件のスコアが更新され、Y 件が新規クリアになった」といったフィードバックを即座にユーザーに示せるようにする。
- 差分計算の責務をサーバ（信頼できる唯一の情報源）に置き、クライアント側の状態管理負担と不整合リスクを排除する。
- 既存の登録処理（トランザクション、overpower 再計算、rating 計算、スキップ処理など）のセマンティクスを維持したまま、最小限の変更で実現する。

対象エンドポイント:
- 直接登録と temp+commit 両方（内部的には同一の `PlayerDataUsecase.Register` を通るため自動的にカバー）。

---

## 2. 背景と方針

### 2.1 サーバ側計算 vs クライアント側比較

- クライアント側で前回状態を保持して比較する場合、複数クライアント・デバイス間での不整合、一時登録フローでの認証前状態取得困難、fetch-then-send によるラウンドトリップ増・レース条件などが懸念される。
- サーバ側が前回状態を DB から読み、payload と比較するのが信頼性・一貫性・将来拡張性の観点で優位。
- 規模（1 万件程度）であれば in-memory 比較のコストは無視できる。

### 2.2 「登録後 updated_at 10分以内フィルタ」案の評価

ユーザ提案の「前回全件取得 → 登録 → updated_at が最近（10分以内）のものを取得して差分とする」アプローチは以下の理由で不採用とする。

- `player_records.updated_at` はクライアント由来の `payload.UpdatedAt`（データ取得時点）を格納するものであり、「このインポートで最後に触られた」というサーバ側タイムスタンプではない。
- 10分という閾値が恣意的。クライアント時計ずれ、同一時刻での複数登録、遅延したインポートなどで誤検知・取りこぼしが発生しやすい。
- before 値を得るには結局前回状態の保持が必要になり、時刻フィルタのメリットが薄れる。
- より正確にするには `updated_at = <この登録で使った正確な値> AND chart_id IN (この payload で送った全 chart_id)` のようなクエリが必要になるが、それでも時刻依存の脆さと追加クエリコストが残る。

結論: 前回状態をロードするなら、そのデータを使って**アプリケーション層で直接比較**するのが最もシンプルで堅牢。

### 2.3 全体方針

1. Clean Architecture / DDD を厳守（Usecase 層が差分計算のオーナー、Domain は純粋、Infra への依存禁止）。
2. 既存の `changedCondition`（score またはいずれかの lamp の変化）と**厳密に同一の条件**で差分を判定（ロジック乖離防止）。
3. slot / slot_order の変化は現時点では差分対象外（DB の timestamp 更新条件に合わせる。将来的に別途要否検討）。
4. 過剰実装を避け、シンプルに。初版は「件数サマリ + 変更のあった譜面の最小識別子＋主要値の before/after」を返す。
5. 既存登録処理のトランザクション境界・エラーセマンティクス・パフォーマンス特性を崩さない。
6. TDD（Red → Green → Refactor）を基本とし、テストは `assert` を用いたテーブルテスト中心に記述。
7. 実装完了後 `go test ./...` と `gofmt -s -w .` を必須とする。

---

## 3. 機能要件

### 3.1 差分の定義（判定条件）

`player_records` / `player_worldsend_records` の以下のいずれかが異なる場合に「更新」とみなす（DB の `fullRecordChangedCondition` / `worldsend...` と完全一致）。

- `score`
- `clear_lamp_id`
- `combo_lamp_id`
- `full_chain_id`

前回レコードが存在しない場合（初回登録、またはその譜面が未プレイ）は「新規」として差分に含める。

スキップされたレコード（検証失敗など）は差分に含めない。

### 3.2 出力内容

- **必須**: `counts` への追加フィールドで「実際に値が変わった件数」を返す。
  - `full_records_actually_changed`
  - `worldsend_records_actually_changed`
- **推奨（初版で実装）**: 変更のあった譜面のリストを `changes` として返す（`skipped_records` と同様に 0 件時は省略可）。
  - 各要素はクライアントが簡単に扱える `idx` + `diff` をキーとする。
  - before/after を含め、UI で「スコア 990000 → 1001000」「新たに AJ 達成」などを表示可能にする。
- 詳細リストが巨大になる初回登録時などの配慮: リストは常に返す（1万件 JSON は profile 系の "all" ですでに類似規模を返しており、クライアント側で扱える前提）。必要に応じて将来「詳細省略モード」や件数上限を検討。

### 3.3 非機能

- 登録処理全体のレイテンシ増加を最小限に（軽量クエリ 1 回 + in-memory 比較）。
- 差分計算に失敗した場合の扱い: 登録トランザクション自体を失敗させる（現状のエラー伝播に合わせる）。差分は「付加価値」ではなく「登録結果の一部」と位置づける。
- 後方互換: 既存 JSON フィールドは変更せず、フィールド追加のみ。

---

## 4. レスポンス設計（提案）

### 4.1 追加/変更 DTO（`internal/dto/api_internal/player_data_dto.go`）

```go
// PlayerDataCounts に以下を追加
type PlayerDataCounts struct {
	FullRecordsUpserted        int `json:"full_records_upserted"`
	WorldsendRecordsUpserted   int `json:"worldsend_records_upserted"`
	FullRecordsSkipped         int `json:"full_records_skipped"`
	WorldsendRecordsSkipped    int `json:"worldsend_records_skipped"`
	HonorsSkipped              int `json:"honors_skipped"`

	// 新規: 実際に値が変化した件数（差分計算結果）
	FullRecordsActuallyChanged      int `json:"full_records_actually_changed"`
	WorldsendRecordsActuallyChanged int `json:"worldsend_records_actually_changed"`
}

// 新規構造体
type PlayerDataRecordChange struct {
	RecordType string `json:"record_type"` // "full" | "worldsend"
	Idx        string `json:"idx"`
	Diff       string `json:"diff"`

	// "new" = 前回レコードなし / "updated" = 値が改善
	ChangeType string `json:"change_type"`

	// 新規の場合は before を省略（または null）
	Before *PlayerDataScoreEntry `json:"before,omitempty"`
	After  PlayerDataScoreEntry  `json:"after"`
}
```

`PlayerDataResult` に追加:

```go
type PlayerDataResult struct {
	...
	Counts  PlayerDataCounts         `json:"counts"`
	Changes []PlayerDataRecordChange `json:"changes,omitempty"`
	SkippedRecords []SkippedRecord   `json:"skipped_records,omitempty"`
}
```

### 4.2 レスポンス例（通常ケース）

```json
{
  "player_id": 42,
  "app_ver": "0.0.1a",
  "imported_at": "2025-11-27T10:45:00+09:00",
  "summary": { ... },
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
      "idx": "1234",
      "diff": "MAS",
      "change_type": "updated",
      "before": { "score": 990000, "clear_lamp": "clear", "cmb_lv": 2, "fch_lv": 1 },
      "after":  { "score": 1001000, "clear_lamp": "clear", "cmb_lv": 3, "fch_lv": 1 }
    },
    {
      "record_type": "full",
      "idx": "5678",
      "diff": "EXP",
      "change_type": "new",
      "after": { "score": 950000, "clear_lamp": "hard", "cmb_lv": 2, "fch_lv": null }
    }
  ]
}
```

### 4.3 初回登録時の例

`before` がすべて省略され、`change_type: "new"` が多数並ぶ。クライアントは「全件新規登録」として扱う想定。

### 4.4 docs/API.md への反映

- レスポンス説明・例・テーブルを更新。
- 最下部の TypeScript interface `PlayerDataCounts` / `PlayerDataResult` に新フィールドを追加。
- 「差分計算はサーバ側で行い、changed_condition と同一の条件で判定する」旨を補足。

---

## 5. アーキテクチャ設計（Clean Architecture 準拠）

### 5.1 責務分担

- **Usecase 層**（オーナー）:
  - トランザクション内で前回状態の軽量ロード。
  - payload 処理中に差分計算（または準備した upsert リストに対して比較）。
  - `api_internal.PlayerDataResult` への差分情報のセット。
  - 比較ロジックはここ（または呼び出す domain ヘルパー）。
- **Domain 層**:
  - 可能なら `PlayerRecordState`（repository 定義）に `Equals(other PlayerRecordState) bool` や `HasMeaningfulChange(...)` を追加して Rich Model 化（必須ではないが望ましい）。
  - 値オブジェクト的な不変性は維持。
- **Repository 層（Domain interface）**:
  - 新規メソッド: 軽量な現在状態取得（重い JOIN なし）。
  - `PlayerDataRepository` に追加（player data import 専用の責務として自然）。
- **Infra 層**:
  - インターフェースの実装のみ。比較ロジックは一切持たない。
- **DTO / Handler**:
  - 既存の result パススルー。handler 変更不要。
- **禁止**:
  - usecase が infra の具体実装を直接 import。
  - 差分ロジックの controller への流出。
  - `SELECT *` や無駄な全カラムロード。

### 5.2 新規/変更ファイル一覧（最小限）

- `internal/domain/repository/player_data_repository.go`
  - `GetCurrentRecordStates(ctx, exec, playerID) (map[int]PlayerRecordState, error)`
  - `GetCurrentWorldsendRecordStates(...)` （worldsend_chart_id → state）
- `internal/infra/repository/player_data_repository_impl.go`
  - 上記 2 メソッドの実装（明示的カラム SELECT + map 構築、バッチ不要）。
  - 既存の `playerDataRecordRow` 構造体を流用または最小サブセット定義。
- `internal/dto/api_internal/player_data_dto.go`
  - `PlayerDataRecordChange` 追加、`PlayerDataCounts` フィールド追加。
- `internal/usecase/player_data_usecase_impl.go`
  - `applyScores` 内で前回状態ロード（save より前）。
  - `applyFullScores` / `applyWorldsendScores` を拡張（prev map を受け取り、差分スライスも返す形にリファクタ推奨）。
  - または別ヘルパー `computeScoreDiffs(...)` を新設して呼び出し。
  - 差分を `result` に設定。
- `internal/usecase/player_data_usecase_apply_scores_test.go`
  - 差分計算ケースをテーブルテストで多数追加（before 状態を stub 的に与える）。
- `docs/API.md`
  - 説明・例・TS interface 更新。

テストファイルの追加は最小限に留め、既存の apply テストを拡張する。

### 5.3 処理フロー（トランザクション内）

```text
tx 開始
  masters ロード
  ensurePlayer
  applyHonors
  prevFullStates     := playerDataRepo.GetCurrentRecordStates(...)        // NEW: 軽量
  prevWorldsendStates:= ...
  counts, skipped, changes, overpower, err := applyScoresWithDiff(
      ctx, tx, playerID, payload.Scores, masters, updatedAt,
      prevFullStates, prevWorldsendStates)
  save
  overpower 再計算用 FindByPlayerID（既存）
  ensurePlayer 2回目
  rating 計算
  result.Changes = changes
  result.Counts に actually_changed をセット
tx コミット
return result
```

`applyScores` のシグネチャ変更は内部的なので、公開インターフェース（`PlayerDataUsecase`）には影響なし。

### 5.4 比較ロジックの実装場所

- 条件を `internal/usecase/player_data_usecase_impl.go` 内に `const` または `func recordStateChanged(old, new repository.PlayerRecordState) bool` として定義（DB の文字列条件と並べてコメントで同期を明記）。
- または `PlayerRecordState` にメソッドを追加して `state.ChangedFrom(prev)` を呼ぶ。
- テストで「この条件で変わった/変わらない」ケースを厳密に検証し、DB 側条件との一致を担保。

---

## 6. パフォーマンス・規模設計

- **ロード**: chart_id をキーにした map[int]PlayerRecordState。1 万件でもメモリ数十〜数百 KB。
- **クエリ追加**: player_records / player_worldsend_records それぞれ 1 回のシンプル SELECT（player_id 条件のみ、JOIN なし）。既存の重い FindByPlayerID とは別。
- **比較**: O(N) in-memory ループ。N=10k でも 1ms 未満。
- **レスポンス**: 変更件数が通常数十件程度。初回のみ大規模になるが、profile record view で既に全件返却している実績があるため許容。
- **全体**: 登録処理の DB ラウンドトリップは +1〜2 回程度に留める。既存の bulk upsert チャンク処理に影響なし。
- N+1 厳禁: 差分用ロードは 1 クエリで全件取得。

将来的に「差分詳細は別途取得する軽量 counts のみ返すモード」が必要になった場合、フラグで制御可能にする拡張性を持たせる。

---

## 7. 実装タスク分解（提案、TDD 推奨順）

1. DTO 定義（`player_data_dto.go` に `PlayerDataRecordChange` と counts フィールド追加）。
2. Repository interface 拡張（`player_data_repository.go`）。
3. Infra 実装 + 単体テスト（`player_data_repository_impl_test.go` に軽量ロードのテスト追加）。
4. Usecase 差分計算ロジック実装（ヘルパー関数 + apply 系への統合）。
5. Register フローへの組み込み（前回ロード、結果セット）。
6. 既存 apply テストの大幅拡張（before 状態を与えたテーブルテストで new/updated/no-change/skip 網羅）。
7. `docs/API.md` のレスポンス説明・例・TS interface 更新。
8. `go test ./...` 全パス確認 + `gofmt -s -w .`。
9. セルフレビュー（AGENTS.md チェックリスト準拠）。
10. （任意）手動で実データ規模の登録を実行し、レスポンスと DB updated_at の一致を確認。

タスクは小さい単位で commit（`feat: ...` プレフィックス）。

---

## 8. テスト観点（優先順）

- 正常系
  - スコアのみ改善
  - ランプのみ改善（clear / combo / full_chain 個別・複合）
  - スコア + ランプ同時改善
  - 新規レコード（前回なし）
  - 同一値（no change）多数混在 → actually_changed = 0
  - full と worldsend の混在
- 初回登録（player_records が空）
- スキップされたエントリが差分に混入しないこと
- 条件の厳密一致: Go 側の `changed` 判定結果と、実際に DB が updated_at を更新した行が一致すること（統合テストまたはリポジトリテストで担保）
- 異常系: ロード失敗時、比較中の不正データ時 → エラーが登録全体を失敗させる（現状セマンティクス維持）
- 性能・規模: 5000 件規模の before/after でクエリ回数・メモリが想定内であること（テスト内で簡易計測可）
- テーブルテスト + Given-When-Then コメントを徹底（AGENTS.md テンプレート準拠）

`require` は前条件で、`assert` は結果検証で使用。

---

## 9. リスク・未決事項

### 9.1 未決（初版で判断が必要なもの）

- `changes` リストを**常に**返すか（0 件時も空配列）、`skipped_records` のように「存在時のみ」にするか。
- before/after に `PlayerDataScoreEntry` をそのまま使うか、より軽量な専用構造体（`{Score, ClearLamp, ComboLv, FullChain}`）にするか（後者の方がレスポンスが少し軽くなる）。
- change_type の値: `"new"` / `"updated"` で十分か、`"score"` / `"lamp_clear"` など細かく分類するか。
- slot/order 変化を将来的に「ユーザー向け更新差分」として含めるか（現在 DB timestamp 条件外）。
- 差分リストに楽曲タイトルなどを enrich するか（masters から引けるが、初版では最小に留める）。

### 9.2 リスク

- 初回登録で `changes` が 7000 件になる場合の JSON サイズ・クライアント処理負荷（実運用で観測し、必要なら上限や省略オプションを追加）。
- 既存の `updated_at` セマンティクス（クライアント時刻）を変更しないことの確認。
- 差分計算ロジックと DB の `IF` 条件の同期漏れ（テストで機械的にガード）。

### 9.3 将来拡張の余地

- 差分に基づく「この登録で rating がどれだけ上がったか」のサマリ追加。
- 変更履歴テーブルの別途記録（今回はスコープ外）。
- クライアントが「差分詳細は不要、件数だけ欲しい」場合の軽量モード。

---

## 10. 関連ドキュメント更新

- **必須**: `docs/API.md`（レスポンス仕様、例、TS interface、補足説明）。
- 任意: `docs/domain_model_specification.md`（PlayerRecordState などの言及があれば）。
- 本設計書自体を `_report/` に残し、実装 PR で「この設計に基づく」と参照。
- 実装完了後、AGENTS.md の「変更時の必須ステップ」を厳守。

---

## 11. 参考: 代替アプローチ（記録用）

- 後方取得 + exact timestamp + IN (chart list): 検討したが、時刻依存と before 値取得の二重コストで却下。
- 登録前に全件ロード → save → 登録後全件再ロード → 2 つの map を diff: シンプルだが重い JOIN を 2 回呼ぶことになり非効率。
- 純粋に DB の affected 情報を使う: MySQL の ON DUPLICATE + IF では ROW_COUNT() が「実際に timestamp を更新したか」を正確に教えてくれないため不採用。

サーバ側 in-memory 比較（前回軽量状態 + payload 処理時）が、現時点で最もバランスが良い。

---

本設計書に基づき実装を進める場合、まずリポジトリの軽量 state 取得から着手し、Usecase のテストを厚く書くことを推奨する。追加質問や設計の微調整は随時受け付ける。