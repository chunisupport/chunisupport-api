# is_maxop_unknown 追加 タスクスタブ

## 背景
- 現在の `maxop` は楽曲内の最大譜面定数 (`MAX(charts.const)`) から算出している。
- 楽曲内で最も定数が高い譜面が `is_const_unknown = true` の場合、真の最大定数を特定できないため、`maxop` の確度を示す情報が必要。
- `FindAllExcludingWorldsend` / `FindByDisplayIDs` では既に譜面を一括取得しており、過去はアプリケーション側で集約していた。
- その後、集約処理をDB（相関サブクエリ）へ移譲したが、楽曲件数に比例してサブクエリ評価コストが増えるため、再びアプリケーション側での集約へ戻す方針を検討する。

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
  - `songs` 取得SQLから `max_chart_const` / `is_maxop_unknown` 用の相関サブクエリを除去し、楽曲基本情報の取得に責務を限定する。
  - 既存どおり `charts` を楽曲単位で一括取得し、リポジトリ実装内で `songID` ごとに `max_chart_const` と `is_maxop_unknown` を同時に集約する。
  - 同一constが複数ある場合のtie-break（difficulty_id優先など）をリポジトリ側ロジックで明示し、DB方言差異に依存しない挙動に統一する。
  - 集約結果マップを使って `toSongEntity` で `Song.MaxChartConst` / `Song.IsMaxOPUnknown` を設定する。
  - `FindByDisplayID` は単曲取得のため、既存の譜面読み出し結果を使って同じ集約関数を再利用する。

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
  - [ ] `songs` 取得SQLに相関サブクエリ（`SELECT MAX(...)` / `EXISTS(...)`）を含めないことを検証
  - [ ] 取得済み譜面から `max_chart_const` を正しく集約できることを検証
  - [ ] unknown譜面なしで `is_maxop_unknown=false`
  - [ ] 最大定数譜面がunknownで `is_maxop_unknown=true`
  - [ ] 下位難易度のみunknownでも最大定数譜面がknownなら `is_maxop_unknown=false`
  - [ ] 同一constのtie-break条件を満たすことを検証
  - [ ] 譜面なし楽曲の境界ケースを仕様に沿って確認
- [ ] Red: DTOテストを追加
  - [ ] v1/internalの `SongDTO` に `is_maxop_unknown` が反映されること
  - [ ] JSONシリアライズに `is_maxop_unknown` が出力されること
- [ ] Green: Domain/Repository/DTO実装
- [ ] Refactor: 譜面集約ロジックを関数化し、`FindAllExcludingWorldsend` / `FindByDisplayIDs` / `FindByDisplayID` で共有
- [ ] ドキュメント更新 (`docs/API.md`)

## 受け入れ条件
- [ ] 既存API互換を維持しつつ `is_maxop_unknown` を追加できている
- [ ] 最大定数譜面が未判明の楽曲で `is_maxop_unknown=true` が返る
- [ ] `go test ./...` が通る
- [ ] `gofmt` 実行済み

## リスク・注意点
- アプリケーション側集約で、譜面数が多いケースのループコストとメモリ使用量を計測する。
- SQL集約条件のDB差異は減る一方、集約ロジックの重複実装による仕様ズレに注意する。
- `is_const_unknown` は譜面単位、`is_maxop_unknown` は「最大定数譜面に対する確度」なので責務が異なる点をドキュメントで明記する。
