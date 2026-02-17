# is_maxop_unknown 追加 タスクスタブ

## 背景
- 現在の `maxop` は楽曲内の最大譜面定数 (`MAX(charts.const)`) から算出している。
- 楽曲内で最も定数が高い譜面が `is_const_unknown = true` の場合、真の最大定数を特定できないため、`maxop` の確度を示す情報が必要。

## 目的
- 楽曲レスポンスに `is_maxop_unknown` を追加し、`maxop` が暫定値である可能性を明示する。

## 仕様（合意案）
- 判定ルール:
  - その楽曲に紐づく譜面のうち、「現在の最大定数として採用される譜面」が `is_const_unknown = true` の場合に `is_maxop_unknown = true`。
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
  - `FindAllExcludingWorldsend` / `FindByDisplayIDs` / `FindByDisplayID` の楽曲取得SQLに、"最大定数譜面がunknownか" を判定する列を追加。
  - 推奨SQLイメージ（DB方言に合わせて調整）:
    - `COALESCE((SELECT c2.is_const_unknown FROM charts c2 WHERE c2.song_id = songs.id ORDER BY c2.const DESC, c2.difficulty_id DESC LIMIT 1), 0) AS is_maxop_unknown`
    - ※ 同一constが複数ある場合のtie-break（difficulty_id優先など）を仕様で固定する。
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
  - [ ] 最大定数譜面がunknownで `is_maxop_unknown=true`
  - [ ] 下位難易度のみunknownでも最大定数譜面がknownなら `is_maxop_unknown=false`
  - [ ] 譜面なし楽曲の境界ケースを仕様に沿って確認
- [ ] Red: DTOテストを追加
  - [ ] v1/internalの `SongDTO` に `is_maxop_unknown` が反映されること
  - [ ] JSONシリアライズに `is_maxop_unknown` が出力されること
- [ ] Green: Domain/Repository/DTO実装
- [ ] Refactor: 重複SQL断片の整理（過剰抽象化はしない）
- [ ] ドキュメント更新 (`docs/API.md`)

## 受け入れ条件
- [ ] 既存API互換を維持しつつ `is_maxop_unknown` を追加できている
- [ ] 最大定数譜面が未判明の楽曲で `is_maxop_unknown=true` が返る
- [ ] `go test ./...` が通る
- [ ] `gofmt` 実行済み

## リスク・注意点
- SQL集約条件のDB差異（`EXISTS`/`CASE`/真偽値表現）に注意。
- `is_const_unknown` は譜面単位、`is_maxop_unknown` は「最大定数譜面に対する確度」なので責務が異なる点をドキュメントで明記する。
