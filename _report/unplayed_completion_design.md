# 未プレイ補完 機能設計書

## 1. 背景と目的

現行のユーザーレコード取得API（`GET /internal/users/:username` / `GET /v1/users/:username`）は、プレイ済み譜面のみを返却します。
一方でフロントエンドでは、一覧表示・フィルタ・ソート・集計のために「未プレイ譜面を含む全譜面集合」を必要とするケースがあり、クライアント側でマスタデータとの突合実装が必要になっています。

本設計書では、APIが未プレイ譜面を補完して返却する仕様を、後方互換性を維持しながら導入する方針を定義します。

## 2. 仕様確定事項

API互換性は不要（一部仕様変更あり）とし、`is_played` 追加や一部フィールドの `null` 許容を含めて仕様変更する。

### 2.1 基本ルール
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

### 2.2 対象API
- `GET /internal/users/:username`
- `GET /v1/users/:username` (および互換エンドポイント)

**追加クエリパラメータ**
- `include_noplay` (boolean, optional)
  - `true` の場合、通常譜面 (`records.all`) と WORLD'S END (`records.worldsend`) に未プレイ譜面を補完して返却する。
  - デフォルトは `false` (未プレイを含めない)。

### 2.3 データ設計
#### DTO変更
- `PlayerRecordDTO`
  - 追加: `is_played: boolean`
  - 変更: `clear_lamp: string | null`
  - 変更: `updated_at: string | null`

- `WorldsendRecordDTO`
  - 追加: `is_played: boolean`
  - 変更: `clear_lamp: string | null`
  - 変更: `updated_at: string | null`

#### 未プレイ補完時のフィールド値
- `is_played`: `false`
- `score`: `0`
- `rating`: `0`
- `overpower`: `0`
- `clear_lamp`: `null`
- `updated_at`: `null`
- `combo_lamp` / `full_chain` / `slot`: 既存ルールに従って `null`
- `is_const_unknown`: 譜面マスタ値を採用
- `const` / `title` / `artist` / `difficulty` / `img`: 譜面・楽曲マスタ値を採用

#### 既存プレイデータ
- `is_played`: `true`
- `clear_lamp` / `updated_at`: 既存DB値を返却

## 3. アーキテクチャ・設計

未プレイ補完ロジックは複数リポジトリを横断するため、`UserUsecase` に直接ロジックを寄せず、専用のドメインサービスへ責務を分離する。

- **新設**: `internal/domain/service/record_completion_service.go`
- **役割**:
  - 通常譜面 / WORLD'S END の未プレイ補完判定
  - 補完レコードの組み立て
  - ソート済み結果の返却
- **UserUsecase の役割**:
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

## 4. 実装計画

### 4.1 実装ステップ
1. **Handler / Usecase のパラメータ伝播**
   - `include_noplay` をハンドラで受け取り、Usecase に伝播する。
   - `view=rating` 時は `include_noplay` を無視する（レスポンスは既存rating表示仕様）。

2. **DTO型と変換関数の更新**
   - `internal/dto/player_record_dto.go` を更新。
   - `internal/dto/worldsend_dto.go` を更新。
   - 既存レコード変換時は `is_played=true` を設定。

3. **ドメインサービス導入**
   - `RecordCompletionService` を新設。
   - 入力:
     - 既存プレイヤーレコード集合
     - 補完対象マスタ集合（通常譜面 / WORLD'S END）
   - 出力:
     - 補完済み通常譜面エンティティ列（`[]*entity.PlayerRecord`）
     - 補完済みWORLD'S ENDエンティティ列（`[]*entity.PlayerWorldsendRecord`）
   - キー判定・補完値設定・ソートをサービス内部に集約。

4. **Usecase からの呼び出し**
   - 既存 `player_records` / `player_worldsend_records` を取得。
   - 削除済み除外で通常譜面・WORLD'S END母集団を一括取得。
   - `RecordCompletionService` を呼び出して補完済み配列を得る。
   - `records.all` / `records.worldsend` に反映。

5. **APIドキュメント更新（`docs/API.md`）**
   - `include_noplay` クエリを追加。
   - `view=rating` 併用時は `include_noplay` 無視と明記。
   - `PlayerRecordDTO` / `WorldsendRecordDTO` に `is_played` を追記。
   - 未プレイ補完時に `clear_lamp` / `updated_at` が `null` となることを明記。

### 4.2 テスト計画
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

### 4.3 影響範囲
- Handler: `internal/app/handler/api_internal/user_handler.go` 等
- Usecase: `internal/usecase/user_usecase.go`
- Domain Service: `internal/domain/service/record_completion_service.go`（新規）
- DTO: `internal/dto/player_record_dto.go`, `internal/dto/worldsend_dto.go`
- Document: `docs/API.md`

## 5. 意思決定メモ

- **採用**: `include_noplay` の opt-in 方式
- **補完範囲**: `records.all` と `records.worldsend`
- **判別子**: `is_played` を追加
- **`view=rating` との併用**: `include_noplay` 無視

上記により、既存互換性を維持しつつ、フロントエンド実装コストを削減できます。
