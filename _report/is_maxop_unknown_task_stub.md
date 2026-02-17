# is_maxop_unknown 追加 タスクスタブ

## 背景
- 現在の `maxop` は楽曲内の最大譜面定数 (`MAX(charts.const)`) から算出している。
- 譜面に `is_const_unknown = true` が含まれる場合、真の最大定数を取り逃す可能性があるため、`maxop` の確度を示す情報が必要。

## 目的
- 楽曲レスポンスに `is_maxop_unknown` を追加し、`maxop` が暫定値である可能性を明示する。

## 仕様（合意案）
- 判定ルール:
  - その楽曲に紐づく譜面のうち、1件でも `is_const_unknown = true` が存在すれば `is_maxop_unknown = true`。
  - それ以外は `false`。
- `maxop` 自体は既存どおり `number` で返し、互換性を維持する。
- `is_const_unknown`（譜面単位）と `is_maxop_unknown`（楽曲集約値の確度）は役割を分離する。

## 設計方針
### 1. Domain
- `internal/domain/entity/song.go`
  - `Song` に `IsMaxOPUnknown bool` を追加する。

### 2. Infra (Repository)
- `internal/infra/repository/song_repository_impl.go`
  - `songRow` に `is_maxop_unknown` 受け取りフィールドを追加。
  - `FindAllExcludingWorldsend` / `FindByDisplayIDs` / `FindByDisplayID` の楽曲取得SQLに、`is_const_unknown` 集約を追加。
  - 推奨SQLイメージ（DB方言に合わせて調整）:
    - `EXISTS(SELECT 1 FROM charts c2 WHERE c2.song_id = songs.id AND c2.is_const_unknown = 1) AS is_maxop_unknown`
    - または `MAX(CASE WHEN ... THEN 1 ELSE 0 END)` 方式。
  - `toSongEntity` で `Song.IsMaxOPUnknown` にマッピング。

### 3. DTO
- `internal/dto/api_v1/song_dto.go`
- `internal/dto/api_internal/song_dto.go`
  - `SongDTO` に `is_maxop_unknown` を追加。
  - 変換関数 `ToV1SongDTO` / `ToSongDTO` で `song.IsMaxOPUnknown` を設定。

### 4. APIドキュメント
- `docs/API.md`
  - `songs[].is_maxop_unknown` の説明とサンプルJSONを追加。

## 実装タスク（TDD）
- [ ] Red: Repositoryテストを追加
  - [ ] unknown譜面なしで `is_maxop_unknown=false`
  - [ ] unknown譜面ありで `is_maxop_unknown=true`
  - [ ] 譜面なし楽曲の境界ケースを仕様に沿って確認
- [ ] Red: DTOテストを追加
  - [ ] v1/internalの `SongDTO` に `is_maxop_unknown` が反映されること
  - [ ] JSONシリアライズに `is_maxop_unknown` が出力されること
- [ ] Green: Domain/Repository/DTO実装
- [ ] Refactor: 重複SQL断片の整理（過剰抽象化はしない）
- [ ] ドキュメント更新 (`docs/API.md`)

## 受け入れ条件
- [ ] 既存API互換を維持しつつ `is_maxop_unknown` を追加できている
- [ ] 未判明定数を含む楽曲で `is_maxop_unknown=true` が返る
- [ ] `go test ./...` が通る
- [ ] `gofmt` 実行済み

## リスク・注意点
- SQL集約条件のDB差異（`EXISTS`/`CASE`/真偽値表現）に注意。
- `is_const_unknown` は譜面単位、`is_maxop_unknown` は楽曲単位であり、重複ではなく責務分離である点をドキュメントで明記する。
