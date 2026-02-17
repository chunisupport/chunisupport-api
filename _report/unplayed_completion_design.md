# 未プレイ補完 機能設計書（改訂版）

## 1. 背景と目的

現行のユーザーレコード取得API（`GET /internal/users/:username` / `GET /v1/users/:username`）は、プレイ済み譜面のみを返却します。
フロントエンド側では、一覧表示・フィルタ・ソート・集計のために「未プレイ譜面を含む全譜面集合」が必要になるため、APIで未プレイ補完を実施してクライアント実装コストを下げます。

本改訂版は、既存コード・スキーマ確認結果を反映し、実装時の曖昧性を排除した実装計画を定義します。

## 2. 仕様（確定）

### 2.1 互換性・対象範囲
1. **破壊的変更は許容**する。ただし変更規模は最小化し、主変更は `is_played` 追加を中心にする。
2. 対象APIは以下のみ:
   - `GET /internal/users/:username`
   - `GET /v1/users/:username`
3. **互換エンドポイントは対象外**。

### 2.2 クエリ仕様
- 追加クエリ: `include_noplay` (boolean, optional)
  - `true`: `records.all` と `records.worldsend` に未プレイ譜面を補完。
  - `false`（デフォルト）: 従来どおりプレイ済みのみ。
- `view=rating&include_noplay=true` の場合:
  - `include_noplay` は**無視**する（エラー化しない）。
  - API.mdには「無視する」旨を1行追記する。

### 2.3 補完範囲と非補完範囲
- 補完対象: `records.all`, `records.worldsend`
- 非補完対象: `records.best`, `records.best_candidate`, `records.new`, `records.new_candidate`
  - 上記は既存どおり実レコードのみ。
  - ただしDTOに `is_played` が追加されるため、既存レコードには `is_played=true` を付与する。

### 2.4 DTO仕様
#### `PlayerRecordDTO`
- 追加: `is_played: boolean`
- 変更: `clear_lamp: string | null`
- 変更: `updated_at: string | null`

#### `WorldsendRecordDTO`
- 追加: `is_played: boolean`
- 変更: `clear_lamp: string | null`
- 変更: `updated_at: string | null`

### 2.5 値のルール
#### 未プレイ補完レコード
- `is_played`: `false`
- `score`: `0`
- `rating`: `0`
- `overpower`: `0`
- `clear_lamp`: `null`
- `updated_at`: `null`
- `combo_lamp` / `full_chain` / `slot`: 既存ルールどおり `null`
- `id` / `const` / `is_const_unknown` / `title` / `artist` / `difficulty` / `img`: マスタ値を採用

#### 既存プレイレコード
- `is_played`: `true`
- `clear_lamp` / `updated_at`: DB値

### 2.6 ソート・キー・除外ルール
1. 重複判定キー:
   - 通常譜面: `chart_id`
   - WORLD'S END: `worldsend_chart_id`
2. 並び順:
   - 通常譜面: `songs.id ASC, charts.difficulty_id ASC`
   - WORLD'S END: `songs.id ASC, worldsend_charts.id ASC`
3. 補完対象から削除済み楽曲を除外する（曲の削除状態に依存）。
4. 難易度文字列はAPI返却時に常に大文字で扱う。

### 2.7 `records.updated_at`
- 従来仕様を維持:
  - 実レコード群の最大 `updated_at`
  - 実レコードがない場合は `player.updated_at`
- 未プレイ補完レコード（`updated_at=null`）は計算対象外。

### 2.8 パフォーマンス方針
- レスポンスサイズ増加は現時点で許容（gzip前提）。
- 本件で追加のページング導入は行わない。

## 3. 設計方針（Clean Architecture / DDD準拠）

### 3.1 責務分離
未プレイ補完は複数データ集合を突合するため、ユースケース直書きではなくドメインサービスに分離する。

- 新設: `internal/domain/service/record_completion_service.go`
- ドメインサービス責務:
  - 通常譜面 / WORLD'S END の補完判定
  - 補完レコード生成
  - ソート済み配列の返却
- Usecase責務:
  - 認可/公開範囲チェック
  - DB取得（I/O）
  - ドメインサービス呼び出し
  - DTO組み立て

### 3.2 依存方向
- ドメインサービスは純粋ロジックとして実装し、`internal/infra` に依存しない。
- I/OはUsecase + Repositoryで処理。
- 必要なら `internal/domain/repository` に最小限の読取IFを追加。

## 4. 実装計画（TDD）

### 4.1 Red（先にテスト）
1. Usecaseテスト追加
   - `include_noplay=false` で既存挙動維持
   - `include_noplay=true` で `records.all` 補完
   - `include_noplay=true` で `records.worldsend` 補完
   - `is_played` の真偽
   - 補完レコードの `clear_lamp=null` / `updated_at=null`
   - `view=rating&include_noplay=true` で無視される
   - `difficulty` が大文字で返る
2. Domain Service単体テスト追加
   - 重複判定キー（`chart_id` / `worldsend_chart_id`）
   - 補完件数
   - ソート順
   - 削除済み楽曲除外

### 4.2 Green（最小実装）
1. Handlerで `include_noplay` を受け取りUsecaseへ伝播
2. DTO更新（`is_played` 追加、`clear_lamp`/`updated_at` nullable化）
3. `RecordCompletionService` 実装
4. Usecaseで補完呼び出しを追加
5. `view=rating` 経路は `include_noplay` を無視して現行仕様維持

### 4.3 Refactor
- 補完ロジック重複を解消
- 変換責務（Entity→DTO）と補完責務（補完判定）を明確化
- テストデータの重複を抑制

## 5. 影響範囲
- Handler: `internal/app/handler/api_internal/user_handler.go` ほかv1系ハンドラ
- Usecase: `internal/usecase/user_usecase.go`, `internal/usecase/user_usecase_impl.go`
- Domain Service: `internal/domain/service/record_completion_service.go`（新規）
- DTO: `internal/dto/player_record_dto.go`, `internal/dto/worldsend_dto.go`
- Docs: `docs/API.md`

## 6. 実装チェックリスト
- [ ] `include_noplay` が `records.all/worldsend` のみに効いている
- [ ] `view=rating` で `include_noplay` が無視される
- [ ] `is_played` が全対象DTOで設定される
- [ ] 補完レコードの `clear_lamp` / `updated_at` が `null`
- [ ] `difficulty` が大文字で返る
- [ ] 削除済み楽曲は補完対象外
- [ ] API.md 更新済み
- [ ] `go test` 成功
- [ ] `gofmt` 実行済み

## 7. 意思決定メモ
- 採用: `include_noplay` opt-in
- 補完範囲: `records.all`, `records.worldsend`
- 判別子: `is_played`
- `view=rating` 併用時: 無視
- 互換エンドポイント: 今回対象外

