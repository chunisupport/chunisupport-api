# is_maxop_unknown 追加 タスクスタブ

## 背景
- 現在の `maxop` は楽曲内の最大譜面定数 (`MAX(charts.const)`) から算出している。
- MASTER/ULTIMA は定数未判明時に暫定値（例: 14.5）で保持されるため、片方だけ先に判明すると、もう片方の未判明譜面が真の最大定数候補であっても見落とす可能性がある。
- そのため、最大定数候補となる難易度（MASTER/ULTIMA）に未判明譜面が残っている間は、`maxop` の確度を示す情報が必要。
- `FindAllExcludingWorldsend` / `FindByDisplayIDs` では既に譜面を一括取得しており、過去はアプリケーション側で集約していた。
- その後、集約処理をDB（相関サブクエリ）へ移譲したが、楽曲件数に比例してサブクエリ評価コストが増えるため、再びアプリケーション側での集約へ戻す方針を検討する。

## 目的
- 楽曲レスポンスに `is_maxop_unknown` を追加し、`maxop` が暫定値である可能性を明示する。

## 仕様（合意案）
- 判定ルール:
  - その楽曲に紐づく譜面のうち、MASTERまたはULTIMAの譜面に `is_const_unknown = true` が1件でも含まれる場合、`is_maxop_unknown = true`。
  - この判定は「現時点で `const` が最大に見える譜面」のみではなく、**最大定数候補（MASTER/ULTIMA）全体**を対象とする。
  - EXPERT以下のunknownは `is_maxop_unknown` 判定対象に含めない（最高定数候補はMASTER/ULTIMAに限定する運用前提）。
  - それ以外は `false`。
- WORLD'S END楽曲はOVER POWERの概念がないため、`is_maxop_unknown` は常に `false` として扱う（必要に応じてAPIドキュメントにも明記する）。
- `maxop` 自体は既存どおり `number` で返し、互換性を維持する。
- `is_const_unknown`（譜面単位）と `is_maxop_unknown`（楽曲集約値の確度）は役割を分離する。

## 設計方針
### 1. Domain
- `internal/domain/entity/song.go`
  - `Song` に `IsMaxOPUnknown bool` を追加する。
- `internal/domain/service/song_aggregation_service.go`（新規）
  - `Song` と譜面リストを受け取り、`max_chart_const` と `is_maxop_unknown` を計算するドメインサービスを追加する。
  - 判定ルール（MASTER/ULTIMAのunknown判定、最大定数算出）を単一点に集約し、リポジトリから分離する。

### 2. Infra (Repository)
- `internal/infra/repository/song_repository_impl.go`
  - `songs` 取得SQLから `max_chart_const` / `is_maxop_unknown` 用の相関サブクエリを除去し、楽曲基本情報の取得に責務を限定する。
  - 既存どおり `charts` を楽曲単位で一括取得し、取得後にドメインサービスへ譜面情報を渡して集約結果を取得する。
  - リポジトリは「永続化データの再構築」に責務を限定し、判定ロジックやtie-breakの詳細は保持しない。
  - `toSongEntity` は `songRow` から `Song` への変換専用に保ち、集約結果の適用は `FindAllExcludingWorldsend` / `FindByDisplayIDs` / `FindByDisplayID` 各メソッド内で行う。
  - `FindAllExcludingWorldsend` / `FindByDisplayIDs` / `FindByDisplayID` のすべてで同一ドメインサービスを利用し、経路差による仕様ズレを防ぐ。

### 3. DTO
- `internal/dto/api_v1/song_dto.go`
- `internal/dto/api_internal/song_dto.go`
  - `SongDTO` に `is_maxop_unknown` を追加。
  - 変換関数 `ToV1SongDTO` / `ToSongDTO` で `song.IsMaxOPUnknown` を設定。

### 4. APIドキュメント
- `docs/API.md`
  - `songs[].is_maxop_unknown` の説明とサンプルJSONを追加。
  - WORLD'S END楽曲では `is_maxop_unknown=false` とする扱いを追記。

## 実装タスク（TDD）
- [ ] Red: Repositoryテストを追加
  - [ ] `songs` 取得SQLに相関サブクエリ（`SELECT MAX(...)` / `EXISTS(...)`）を含めないことを検証
  - [ ] リポジトリがドメインサービスを利用して `Song.MaxChartConst` / `Song.IsMaxOPUnknown` を設定することを検証
- [ ] Red: Domain Serviceテストを追加
  - [ ] 取得済み譜面から `max_chart_const` を正しく集約できることを検証
  - [ ] unknown譜面なしで `is_maxop_unknown=false`
  - [ ] MASTER known / ULTIMA unknown（暫定値が低く見えるケース）でも `is_maxop_unknown=true`（例: MASTER 14.6 known, ULTIMA 14.5 unknown）
  - [ ] MASTER unknown / ULTIMA known でも `is_maxop_unknown=true`
  - [ ] EXPERT以下のみunknown、かつMASTER/ULTIMAがknownなら `is_maxop_unknown=false`
  - [ ] 同一constのtie-break条件を満たすことを検証
  - [ ] 譜面なし楽曲の境界ケースを仕様に沿って確認
- [ ] Red: DTOテストを追加
  - [ ] v1/internalの `SongDTO` に `is_maxop_unknown` が反映されること
  - [ ] JSONシリアライズに `is_maxop_unknown` が出力されること
- [ ] Green: Domain/Repository/DTO実装
- [ ] Refactor: リポジトリ内の譜面集約ロジックを除去し、ドメインサービス呼び出しへ統一
- [ ] ドキュメント更新 (`docs/API.md`)

## 受け入れ条件
- [ ] 既存API互換を維持しつつ `is_maxop_unknown` を追加できている
- [ ] 最大定数譜面が未判明の楽曲で `is_maxop_unknown=true` が返る
- [ ] `go test ./...` が通る
- [ ] `gofmt` 実行済み

## リスク・注意点
- アプリケーション側集約で、譜面数が多いケースのループコストとメモリ使用量を計測する。
- SQL集約条件のDB差異は減る一方、ドメインサービスの利用漏れ（取得経路追加時の呼び忘れ）に注意する。
- `is_const_unknown` は譜面単位、`is_maxop_unknown` は「最大定数譜面に対する確度」なので責務が異なる点をドキュメントで明記する。
