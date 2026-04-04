# マスタデータ返却形式の調査

作成日: 2026-04-04

## 結論

現状は完全に統一されていません。大きく見ると、以下の3系統が混在しています。

1. 読み取りAPIでサーバー側がマスタIDを名称に展開して返すもの
2. フロント向けマスタAPIで `id + name` の両方を返すもの
3. 一部APIでマスタIDやコードをそのまま返すもの

特にフロントエンド観点で重要なのは、通常の楽曲・ユーザー閲覧系APIはかなり名称展開寄りなのに対し、目標APIの `attributes` とプレイヤーの `class_emblem_id` / `class_emblem_base_id` はID寄りのまま残っていることです。

## 調査対象

- フロント向けマスタAPI: `GET /internal/master`
- 内部API DTO/Handler
- 公開API v1 DTO/Handler
- 目標機能の Usecase
- chunirec 互換APIの変換処理
- API仕様書

## 1. フロント向けマスタAPIは `id + name` を返している

`/internal/master` は、フロントで辞書的に使えるマスタ一覧を返しています。実装上は以下を返却しています。

- `genres`: `id`, `name`
- `difficulties`: `id`, `name`
- `account_types`: `id`, `name`
- `versions`: `id`, `name`, `released_at`
- `rating_bands`: `id`, `label`, `min_inclusive`, `max_exclusive`, `sort_order`
- `achievement_types`: `id`, `name`（実体は表示名というより achievement type のコード文字列）

根拠:

- `internal/app/handler/api_internal/master_data_handler.go`
- `internal/dto/master_data_dto.go`

つまり「フロントにもマスタデータを渡している」は事実です。ただし、このAPIで返しているのは一部のマスタだけです。

## 2. 読み取り系APIは、かなり名称展開して返している

### 2-1. 楽曲API

通常楽曲・WORLD'S END ともに、楽曲レスポンスはジャンルIDではなくジャンル名を返しています。

- `SongDTO.Genre` は `*string`
- `WorldsendSongDTO.Genre` は `*string`
- 生成時に `GenreNamesByID` で `GenreID -> Name` 変換している

さらに通常楽曲の `charts` は難易度IDではなく、`BASIC` / `ADVANCED` / `EXPERT` / `MASTER` / `ULTIMA` をキーにしたマップです。

根拠:

- `internal/dto/api_internal/song_dto.go`
- `internal/dto/api_v1/song_dto.go`
- `internal/dto/api_internal/worldsend_song_dto.go`
- `internal/dto/api_v1/worldsend_song_dto.go`
- `internal/app/handler/api_internal/song_handler.go`
- `internal/app/handler/api_internal/worldsend_handler.go`
- `internal/app/handler/api_v1/song_handler.go`
- `internal/app/handler/api_v1/v1_worldsend_handler.go`

### 2-2. ユーザー/レコードAPI

ユーザー閲覧系も、マスタ名称返却が多いです。

- `UserDTO.AccountType` は `string`
- `PlayerRecordDTO.Difficulty` は `string`
- `PlayerRecordDTO.ClearLamp` / `ComboLamp` / `FullChain` / `Slot` はいずれも名称文字列または `null`
- `WorldsendRecordDTO.ClearLamp` / `ComboLamp` / `FullChain` も名称文字列または `null`
- `HonorDTO.TypeName` も名称系の値

特に `PlayerRecordDTO` は `toMasterNamePtr` でマスタ値オブジェクトから `Name` を取り出して返しており、IDを外へ出していません。

根拠:

- `internal/dto/api_internal/user_dto.go`
- `internal/dto/api_internal/user_list_dto.go`
- `internal/dto/player_record_dto.go`
- `internal/dto/worldsend_dto.go`
- `internal/dto/player_dto.go`
- `internal/dto/api_v1/user_dto.go`

### 2-3. 譜面統計API

譜面統計もIDをそのまま返すのではなく、表示用ラベルに寄せています。

- 難易度は Usecase 側で `DifficultyNamesByID` を使ってキー化
- レーティング帯は `rating_band` にラベル文字列を返却

根拠:

- `internal/usecase/chart_stats_usecase.go`
- `internal/dto/chart_stats_dto.go`

## 3. それでも ID/コード のまま返している箇所がある

### 3-1. 目標APIは意図的に混在している

目標APIは最も分かりやすく混在しています。

- `achievement_type` は ID ではなくコード文字列を返す
- しかし `attributes.diff` / `attributes.genre` / `attributes.ver` はマスタIDのまま返す

つまり、同じレスポンスオブジェクト内で:

- 成果種別はコード文字列
- 難易度/ジャンル/バージョン条件はID

という構造です。

これは偶然ではなく、Usecase 実装上そうなっています。

- 入力時: `AchievementTypesByCode` でコードを ID に変換
- 出力時: `AchievementTypesByID` で ID をコードに戻す
- ただし `Attributes` はJSONをそのまま decode して返すだけなので、内部保存されたIDがそのまま出る

根拠:

- `internal/dto/api_internal/goal_dto.go`
- `internal/app/handler/api_internal/goal_handler.go`
- `internal/usecase/goal_usecase_impl.go`
- `docs/API.md` の目標API仕様

フロントエンドが目標の条件を表示する場合、`/internal/master` の以下を使って解決する構成が想定しやすい状態です。

- `difficulties`
- `genres`
- `versions`
- `achievement_types`（`name` に achievement type のコード文字列が入る）

### 3-2. プレイヤーの class emblem 系は ID のまま返している

プレイヤー情報では以下がIDのままです。

- `class_emblem_id`
- `class_emblem_base_id`

これは内部APIでも公開API v1 でも同じです。

根拠:

- `internal/dto/player_dto.go`
- `internal/dto/api_v1/user_dto.go`

一方で、サーバー内部にはクラスエンブレム系マスタのキャッシュ自体は存在します。

- `ClassEmblems`
- `ClassEmblemBases`
- `GetClassEmblemNameByID`
- `GetClassEmblemBaseNameByID`

根拠:

- `internal/infra/masterdata/cache.go`

にもかかわらず、`/internal/master` ではこれらを返していません。名称化して返しているのは chunirec 互換APIだけです。

根拠:

- `internal/app/handler/compat/chunirec/dto.go`

したがって、class emblem 系は「内部にはマスタがあるが、フロント向けマスタAPIには露出しておらず、通常APIではIDのまま返している」状態です。

## 4. 実態の整理

### 名称/ラベルに展開して返すもの

- 楽曲の `genre`
- 楽曲 `charts` の難易度キー
- ユーザーの `account_type`
- レコードの `difficulty`
- レコードの `clear_lamp` / `combo_lamp` / `full_chain` / `slot`
- 譜面統計の `rating_band`
- 目標の `achievement_type`（IDではなくコード）

### `id + name` の両方をマスタAPIで返すもの

- `genres`
- `difficulties`
- `account_types`
- `versions`
- `rating_bands`
- `achievement_types`

### IDのまま返しているもの

- 目標 `attributes.diff`
- 目標 `attributes.genre`
- 目標 `attributes.ver`
- プレイヤー `class_emblem_id`
- プレイヤー `class_emblem_base_id`

## 5. フロントエンド観点での意味

### フロントでマスタ解決が不要な箇所

以下はレスポンスだけでほぼ表示可能です。

- 楽曲一覧/詳細
- WORLD'S END 楽曲一覧/詳細
- ユーザープロフィールの大部分
- レコード一覧
- 譜面統計

### フロントでマスタ解決が必要な箇所

以下は `GET /internal/master` を参照しないと表示名に戻せません。

- 目標 `attributes.diff`
- 目標 `attributes.genre`
- 目標 `attributes.ver`

### フロントで現状は解決しづらい箇所

以下は通常APIでIDが返る一方、`/internal/master` には対応マスタがありません。

- `class_emblem_id`
- `class_emblem_base_id`

このため、フロントで人間向け名称を表示したい場合は、現状では次のいずれかが必要です。

1. `/internal/master` に `class_emblems` / `class_emblem_bases` を追加する
2. ユーザー系APIで class emblem 名も同時に返す
3. フロントが別の静的辞書を持つ

## 6. 補足: 実装と仕様書の差分

`/internal/master` は実装上 `achievement_types` を返していますが、`docs/API.md` の当該レスポンス例とフィールド説明は完全には追随していません。

- 見出し直下に「`achievement_types` が追加されます」とはある
- ただしレスポンス例に `achievement_types` が載っていない
- フィールド一覧にも `achievement_types` 行がない

根拠:

- `internal/app/handler/api_internal/master_data_handler.go`
- `internal/dto/master_data_dto.go`
- `docs/API.md`

## 最終まとめ

現状は「マスタデータをフロントにも渡しているが、全レスポンスをID参照型に統一しているわけではない」です。

実態としては以下です。

- 閲覧系APIは名称展開を多用している
- 目標APIはコード文字列とIDが混在している
- class emblem 系はID返却のまま残っている
- `/internal/master` は存在するが、フロントが必要としそうな全マスタを網羅しているわけではない

「混ざっている気がする」は正しく、特に問題として認識しやすいのは以下の2点です。

1. 目標APIの `achievement_type` は文字列なのに `attributes` はID
2. `class_emblem_id` / `class_emblem_base_id` はIDなのに、通常のフロント向けマスタAPIでは解決材料が返らない