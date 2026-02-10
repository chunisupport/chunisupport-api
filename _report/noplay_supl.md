# 未プレイ補完機能 実装計画書（確定版）

## 目的
`GET /internal/users/:username` で `include_noplay=true` を指定した際に、通常譜面 (`records.all`) と WORLD'S END (`records.worldsend`) の両方で未プレイ譜面を補完して返却する。

API互換性は不要とし、`is_played` 追加や一部フィールドの `null` 許容を含めて仕様変更する。

---

## 仕様確定事項（合意済み）
1. `view=rating` と `include_noplay=true` 併用時は **include_noplay を無視**する（エラーにはしない）。
2. `records.updated_at` は **従来どおり**（レコード最大更新日時。レコードがない場合は `player.updated_at`）。
3. 未プレイ補完データの `score` / `rating` / `overpower` は **固定値 0** を返す。
4. 重複判定キーは以下で統一する。
   - 通常譜面: `chart_id`
   - WORLD'S END: `worldsend_chart_id`
5. 並び順は以下で固定する。
   - 通常譜面: `songs.id ASC, charts.difficulty_id ASC`
   - WORLD'S END: `songs.id ASC, worldsend_charts.id ASC`
6. `is_played` は `all` / `worldsend` だけでなく、同DTOを使う配列（best/new系）にも付与する。
7. `clear_lamp` / `updated_at` は **未プレイ補完データのみ `null`**。DB 側の NOT NULL 制約は維持する。
8. 補完対象から削除済み楽曲は除外する。
9. レスポンスサイズ増加は現時点では許容する（gzip 圧縮前提）。

---

## 設計方針（レビュー反映）
未プレイ補完ロジックは複数リポジトリを横断するため、`UserUsecase` に直接ロジックを寄せず、専用のドメインサービスへ責務を分離する。

- 新設候補: `internal/domain/service/record_completion_service.go`
- 役割:
  - 通常譜面 / WORLD'S END の未プレイ補完判定
  - 補完レコードの組み立て
  - ソート済み結果の返却
- `UserUsecase` は以下に専念:
  - 認可/公開範囲チェック
  - 必要データの取得と引数整形
  - ドメインサービス呼び出し
  - DTO組み立てのオーケストレーション

### 依存関係の整理
クリーンアーキテクチャの依存方向を守るため、ドメインサービス自体は具体リポジトリ実装に依存させない。

- ドメインサービスは「入力として渡されたデータ集合」を処理する純粋ロジックにする。
- I/O（DB取得）は Usecase で行う。
- 必要なら `internal/domain/repository` に最小限の読み取りインターフェースを追加し、Usecase経由で注入する。

> 注: ドメインサービスが `internal/infra` に直接依存する構造は採用しない。

---

## 対象API
- `GET /internal/users/:username`
  - 追加クエリ: `include_noplay` (boolean, 任意)
  - `include_noplay=true` のとき:
    - `records.all` に通常譜面の未プレイ補完を含める
    - `records.worldsend` に WORLD'S END の未プレイ補完を含める

---

## データ設計
### DTO変更
- `PlayerRecordDTO`
  - 追加: `is_played: boolean`
  - 変更: `clear_lamp: string | null`
  - 変更: `updated_at: string | null`

- `WorldsendRecordDTO`
  - 追加: `is_played: boolean`
  - 変更: `clear_lamp: string | null`
  - 変更: `updated_at: string | null`

### 未プレイ補完時のフィールド値
- `is_played`: `false`
- `score`: `0`
- `rating`: `0`
- `overpower`: `0`
- `clear_lamp`: `null`
- `updated_at`: `null`
- `combo_lamp` / `full_chain` / `slot`: 既存ルールに従って `null`

### 既存プレイデータ
- `is_played`: `true`
- `clear_lamp` / `updated_at`: 既存DB値を返却

---

## 実装ステップ
### 1. Handler / Usecase のパラメータ伝播
- `include_noplay` をハンドラで受け取り、Usecase に伝播する。
- `view=rating` 時は `include_noplay` を無視する（レスポンスは既存rating表示仕様）。

### 2. DTO型と変換関数の更新
- `internal/dto/player_record_dto.go` を更新。
- `internal/dto/worldsend_dto.go` を更新。
- 既存レコード変換時は `is_played=true` を設定。

### 3. ドメインサービス導入
- `RecordCompletionService`（仮）を新設。
- 入力:
  - 既存プレイヤーレコード集合
  - 補完対象マスタ集合（通常譜面 / WORLD'S END）
- 出力:
  - 補完済み通常譜面DTO列
  - 補完済みWORLD'S ENDDTO列
- キー判定・補完値設定・ソートをサービス内部に集約。

### 4. Usecase からの呼び出し
1. 既存 `player_records` / `player_worldsend_records` を取得。
2. 削除済み除外で通常譜面・WORLD'S END母集団を一括取得。
3. `RecordCompletionService` を呼び出して補完済み配列を得る。
4. `records.all` / `records.worldsend` に反映。

### 5. APIドキュメント更新（`docs/API.md`）
- `include_noplay` クエリを追加。
- `view=rating` 併用時は `include_noplay` 無視と明記。
- `PlayerRecordDTO` / `WorldsendRecordDTO` に `is_played` を追記。
- 未プレイ補完時に `clear_lamp` / `updated_at` が `null` となることを明記。

---

## テスト計画
1. `include_noplay=false` で既存挙動が維持される。
2. `include_noplay=true` で通常譜面の未プレイ補完が入る。
3. `include_noplay=true` で WORLD'S END の未プレイ補完が入る。
4. 既存レコード `is_played=true` / 補完レコード `is_played=false` を検証。
5. 補完レコードで `clear_lamp=null` / `updated_at=null` を検証。
6. `view=rating&include_noplay=true` で include_noplay が無視されることを検証。
7. ドメインサービス単体テストで以下を検証:
   - キー重複判定（`chart_id` / `worldsend_chart_id`）
   - 補完件数
   - ソート順

---

## 影響範囲
- Handler:
  - `internal/app/handler/api_internal/user_handler.go`
  - 必要に応じて `api_v1` / `compat` の呼び出し箇所
- Usecase:
  - `internal/usecase/user_usecase.go`
  - `internal/usecase/user_usecase_impl.go`
- Domain Service:
  - `internal/domain/service/record_completion_service.go`（新規）
- DTO:
  - `internal/dto/player_record_dto.go`
  - `internal/dto/worldsend_dto.go`
- Document:
  - `docs/API.md`

---

## 実装上の注意
- N+1を避けるため、母集団取得は必ず一括取得で行う。
- 既存DBスキーマの NOT NULL は変更せず、未プレイ補完時のみアプリ側で `null` を組み立てる。
- 依存関係は「外側 → 内側」を維持し、Usecase から Infra 実装への直接依存を増やさない。
