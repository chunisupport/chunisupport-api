# 未解禁曲管理テーブル 実装計画書

このドキュメントは永続化せず、PRマージ前に削除します。

## 1. 目的

本計画書は、OVER POWER（以下OP）計算APIの前提となる、プレイヤーごとの未解禁楽曲管理テーブルを追加するための実装方針を定義する。

OP計算では、プレイヤーが解禁済みの楽曲・譜面のみを計算対象にする必要がある。一方で、CHUNITHMでは未解禁楽曲でも店内マッチングや全国対戦などでスコアが記録されうるため、スコアデータだけでは解禁状態を判定できない。

そのため、本対応では「未解禁として手動登録された楽曲」をDBで管理し、後続のOP計算APIが参照できる状態を作る。

---

## 2. 対象範囲

対象:

- 未解禁曲管理テーブルの追加
- 未解禁曲管理用のドメインモデル追加
- 未解禁曲管理用のリポジトリ追加
- 自分の未解禁曲を登録・解除・一覧取得する内部API追加
- API仕様書と関連ドキュメントの更新
- 後続のOP計算APIから参照しやすい取得メソッドの整備

非対象:

- OP計算ロジック本体
- `players.overpower_value` / `players.overpower_percentage` の更新処理
- フロントエンド実装
- 未解禁状態の履歴管理
- 論理削除
- 解禁済み楽曲の保存

---

## 3. 採用仕様

## 3.1 テーブルの意味

未解禁曲管理テーブルは、プレイヤーごとの「未解禁である通常譜面群またはULTIMA譜面」を保持する。

基本ルール:

- レコードあり: 未解禁として扱う
- レコードなし: 解禁済み、または未解禁管理対象外として扱う
- `is_ultima = false`: 通常譜面群（BASIC / ADVANCED / EXPERT / MASTER）が未解禁
- `is_ultima = true`: ULTIMA譜面のみ未解禁

## 3.2 カラム

```sql
player_id MEDIUMINT UNSIGNED NOT NULL
song_id INT UNSIGNED NOT NULL
is_ultima BOOLEAN NOT NULL
```

日付カラムは持たない。理由は、今回の用途では「現在未解禁かどうか」だけが必要であり、設定日時を追跡するメリットが容量増加に見合わないためである。

## 3.3 制約

```sql
PRIMARY KEY (player_id, song_id, is_ultima)
FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE
```

追加インデックスは当面作成しない。プレイヤー単位の一覧取得は `PRIMARY KEY (player_id, song_id, is_ultima)` の左端一致で処理できるため、実測で必要になるまで `player_id` 単独インデックスは追加しない。

## 3.4 削除方針

論理削除は行わない。

- 未解禁登録: INSERT
- 未解禁解除: DELETE
- player物理削除: `ON DELETE CASCADE` で連動削除
- song物理削除: `ON DELETE CASCADE` で連動削除

楽曲の論理削除は、OP計算側・一覧表示側で `songs.is_deleted = 0` に絞り込んでから扱う。WORLD'S END楽曲は管理対象外のため、登録時に拒否し、一覧表示側でも `songs.is_worldsend = 0` に絞り込む。

---

## 4. DB設計

## 4.1 MySQLマイグレーション

次のマイグレーションを追加する。

- `migration/mysql/000014_create_player_locked_songs.up.sql`
- `migration/mysql/000014_create_player_locked_songs.down.sql`

想定DDL:

```sql
CREATE TABLE player_locked_songs (
  player_id MEDIUMINT UNSIGNED NOT NULL,
  song_id INT UNSIGNED NOT NULL,
  is_ultima BOOLEAN NOT NULL,
  PRIMARY KEY (player_id, song_id, is_ultima),
  CONSTRAINT fk_player_locked_songs_player_id FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE,
  CONSTRAINT fk_player_locked_songs_song_id FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE
);
```

down側:

```sql
DROP TABLE IF EXISTS player_locked_songs;
```

SQLiteマイグレーションは追加しない。本プロジェクトのSQLiteは統計と小規模データ用途であり、アプリケーション本体の未解禁曲管理テーブルはMySQLのみを対象にする。RepositoryテストもMySQLを前提にし、SQLite用のDDLやテストスキーマは追加しない。

---

## 5. ドメイン設計

## 5.1 エンティティ

新規ファイル:

- `internal/domain/entity/player_locked_song.go`

想定モデル:

```go
type PlayerLockedSong struct {
    PlayerID  int
    SongID    int
    IsUltima  bool
}
```

このエンティティは永続化タグを持たず、未解禁状態の永続化に必要な純粋な状態だけを保持する。API一覧返却用の `display_id` はプレゼンテーション境界の都合であるため、ドメインエンティティには含めない。DB用構造体が必要な場合は `internal/infra/models` に分離する。

## 5.2 ドメイン上の意味

`PlayerLockedSong` は「未解禁状態の現在値」を表す。履歴や設定日時は責務に含めない。

`is_ultima` の判定規則:

- 対象譜面がULTIMAの場合、`is_ultima = true` の未解禁設定があればOP計算対象から除外する
- 対象譜面がULTIMA以外の場合、`is_ultima = false` の未解禁設定があればOP計算対象から除外する

通常譜面群とULTIMA譜面は解禁単位が異なるため、`is_ultima = false` はULTIMA譜面を包含しない。通常譜面群とULTIMA譜面の両方が未解禁の場合は、同一曲に `is_ultima = false` と `is_ultima = true` の2レコードを登録する。

ULTIMA譜面だけが未解禁の場合のみ `is_ultima = true` を登録する。`is_ultima = true` の登録時は、対象楽曲にULTIMA譜面が存在することを検証する。

---

## 6. リポジトリ設計

## 6.1 インターフェース

新規ファイル:

- `internal/domain/repository/player_locked_song_repository.go`

想定インターフェース:

```go
type PlayerLockedSongRepository interface {
    ListByPlayerID(ctx context.Context, exec Executor, playerID int) ([]*entity.PlayerLockedSong, error)
    Create(ctx context.Context, exec Executor, lockedSong *entity.PlayerLockedSong) error
    Delete(ctx context.Context, exec Executor, playerID int, songID int, isUltima bool) error
}
```

OP計算ではプレイヤー単位で一括取得してメモリ上のセットにするのが効率的なため、`ListByPlayerID` を主要メソッドとする。

`ListByPlayerID` は未解禁集合の永続化状態だけを返し、`songs.display_id` は返さない。論理削除済み楽曲の除外もこのメソッドでは行わない。OP計算側は事前に `songs.is_deleted = 0` の譜面へ絞り込んだうえで、`song_id + is_ultima` のセットだけで判定できるため、ドメインリポジトリに表示用データや楽曲表示条件を混入させない。

`Delete` は `player_id`, `song_id`, `is_ultima` の複合主キー全項目を条件にして、指定された未解禁状態だけを削除する。対象レコードが存在しない場合も、解除APIの冪等性を保つため成功扱いにする。

Repositoryインターフェースには、実装時点で利用予定のない `Exists` は追加しない。必要になった時点で用途を明確にして追加する。

削除APIでは、論理削除済み楽曲の未解禁レコードも後から消せる必要があるため、Usecaseで通常の `SongRepository.FindByDisplayID` による楽曲存在確認とエラー判定は行わない。代わりに、削除専用の楽曲ID解決ポートを用意し、`display_id` から `song_id` を解決できた場合だけ `Delete` を呼び出す。`display_id` に該当する楽曲が存在しない場合は、削除対象なしとして成功扱いにする。

削除専用の楽曲ID解決ポートは、`songs.display_id` を条件にし、`songs.is_deleted` / `songs.is_worldsend` では絞り込まない。これは、論理削除済み楽曲に紐づく未解禁レコードを削除可能にするためである。戻り値は `(*int, error)` 相当とし、`display_id` に該当する楽曲が存在しない場合は `nil, nil` を返す。Usecaseは `nil` を削除対象なしとして扱い、`Delete` を呼び出さず成功扱いにする。

API一覧レスポンスでは `display_id` が必要なため、専用のRead Model取得を別ポートとして定義する。

```go
type PlayerLockedSongReadModel struct {
    SongID    int
    DisplayID string
    IsUltima  bool
}

type PlayerLockedSongQueryService interface {
    ListWithSongDisplayIDByPlayerID(ctx context.Context, exec Executor, playerID int) ([]*PlayerLockedSongReadModel, error)
}
```

このRead Model取得ポートは、ドメインエンティティの永続化リポジトリではなく、Usecaseが一覧表示のために利用するQuery用ポートとして扱う。配置は `internal/usecase` 配下、またはQuery用途が明確な専用パッケージとし、`internal/domain/repository` には置かない。

このRead Model取得は `player_locked_songs` と `songs` をJOINし、`songs.is_deleted = 0` かつ `songs.is_worldsend = 0` のレコードだけを返す。返却順は `songs.display_id ASC, player_locked_songs.is_ultima ASC` とする。JOINを使う理由は、Usecaseで `song_id` ごとに楽曲取得を繰り返すN+1問題を避けるためである。JOINを使わない実装にする場合でも、関連楽曲は `IN` 句などで一括取得し、1件ずつ `SongRepository` を呼び出さない。

このリポジトリは「未解禁集合」への追加・削除を扱うため、エンティティ全体を `Save` する集約リポジトリとは性質が異なる。`player_locked_songs` は独立した属性を持つ集約ではなく、プレイヤーに紐づく現在状態の集合であるため、`Create` / `Delete` による集合操作を例外的に許容する。

## 6.2 Infra実装

新規ファイル:

- `internal/infra/repository/player_locked_song_repository_impl.go`
- `internal/infra/repository/player_locked_song_repository_impl_test.go`

実装方針:

- `SELECT *` は使用しない
- `ListByPlayerID` は `player_locked_songs.player_id, player_locked_songs.song_id, player_locked_songs.is_ultima` を明示取得し、`player_locked_songs.song_id ASC, player_locked_songs.is_ultima ASC` で返す
- `PlayerLockedSongQueryService.ListWithSongDisplayIDByPlayerID` は `player_locked_songs.song_id, player_locked_songs.is_ultima, songs.display_id` を明示取得する
- `Create` は重複キーを成功扱いにする
- `Delete` は `player_id`, `song_id`, `is_ultima` の複合主キー全項目を条件にし、対象なしでも成功扱いにする
- 削除専用の楽曲ID解決ポートは、`display_id` が存在しない場合に `nil, nil` を返し、Usecaseで削除対象なしとして扱う

管理APIの使いやすさを優先し、登録・解除は冪等操作にする。登録時は通常の `INSERT` を行い、複合主キーの重複エラーだけを成功扱いに変換する。実装上 `ON DUPLICATE KEY UPDATE player_id = player_id` 相当を使ってもよいが、`INSERT IGNORE` は重複以外の制約違反も見えにくくするため使用しない。解除時は `DELETE` の影響行数が0件でも成功扱いにする。

---

## 7. Usecase設計

## 7.1 インターフェース

新規ファイル:

- `internal/usecase/player_locked_song_usecase.go`
- `internal/usecase/player_locked_song_usecase_impl.go`
- `internal/usecase/player_locked_song_usecase_impl_test.go`

想定インターフェース:

```go
type PlayerLockedSongUsecase interface {
    List(ctx context.Context, userID int) ([]*PlayerLockedSongOutput, error)
    Lock(ctx context.Context, userID int, input *PlayerLockedSongInput) error
    Unlock(ctx context.Context, userID int, input *PlayerLockedSongInput) error
}
```

入力:

```go
type PlayerLockedSongInput struct {
    DisplayID string
    IsUltima  bool
}
```

出力:

```go
type PlayerLockedSongOutput struct {
    DisplayID string
    IsUltima  bool
}
```

## 7.2 ユーザーとプレイヤーの解決

APIは `/internal/me` 配下に置く想定のため、Handlerからは認証済み `userID` を渡す。

Usecaseでは `PlayerRepository.FindByUserID` を使って対象プレイヤーを解決する。プレイヤー未登録の場合は、既存のエラー方針に合わせて適切なエラーへ変換する。

## 7.3 楽曲存在確認

登録時は `SongRepository` または専用メソッドで対象 `display_id` から楽曲を解決し、Repository境界では `song_id` を使う。

方針:

- 存在しない楽曲は既存APIのエラーコード体系に合わせて `song_not_found` として404を返す
- 論理削除済み楽曲は登録不可とし、`song_not_found` として404を返す
- WORLD'S END楽曲は登録不可とし、`song_not_found` として404を返す
- `is_ultima = true` の登録時にULTIMA譜面が存在しない場合は、既存の譜面未検出エラーである `chart_not_found` として404を返す

管理対象はOP計算対象曲であるため、論理削除済み楽曲とWORLD'S END楽曲の登録は拒否する。既存の `SongRepository.FindByDisplayID` は通常楽曲（WORLD'S END除く）だけを取得し、削除済み楽曲も取得する契約である。そのため、WORLD'S END楽曲は `SongRepository.FindByDisplayID` の時点で未検出となり、Usecaseでは `song_not_found` に変換する。取得できた通常楽曲については `song.IsDeleted` を確認し、論理削除済みであれば `song_not_found` に変換する。既に登録済みの曲が後から論理削除された場合は、一覧取得・OP計算のどちらでも `songs.is_deleted = 0` により除外する。ただし削除APIでは、論理削除済み楽曲に紐づく未解禁レコードを消せるようにするため、楽曲の論理削除状態では絞り込まない。

`song_not_found` は `docs/API.md` と `internal/app/apierror/codes.go` で既に楽曲未検出として使われているため、計画書側もこれに合わせる。未解禁設定自体が存在しないケースは、登録・解除を冪等操作にするためAPIエラーにしない。
`chart_not_found` は `docs/API.md`、`docs/error_code_reason_codes.md`、`internal/app/apierror/codes.go`、`internal/usecase/errors.go` で既に譜面未検出として定義済みであり、指定楽曲にULTIMA譜面が存在しないケースに適用できる。そのため、未解禁曲管理専用の新規エラーコードは追加しない。

## 7.4 登録・解除の扱い

Lockは指定された `is_ultima` のレコードだけを作成する。

- `is_ultima = false`: 通常譜面群（BASIC / ADVANCED / EXPERT / MASTER）の未解禁として作成する
- `is_ultima = true`: ULTIMA譜面のみ未解禁として作成する

通常譜面群とULTIMA譜面は別の解禁単位として扱うため、同一曲に `is_ultima = false` と `is_ultima = true` の両方が存在することを許容する。両方が存在する場合は、通常譜面群とULTIMA譜面の両方をOP計算対象から除外する。

Unlockは指定された `is_ultima` のレコードだけを削除する。削除APIでは通常の楽曲存在確認によるエラー判定を行わず、対象レコードが存在しない場合も成功扱いにする。Usecaseは削除専用の楽曲ID解決ポートで `display_id` から `song_id` を解決する。この解決では `songs.is_deleted` / `songs.is_worldsend` で絞り込まない。楽曲が見つからない場合は削除対象なしとして204相当にする。`song_id` が解決できた場合だけ、`PlayerLockedSongRepository.Delete` を呼び出す。

---

## 8. API設計

## 8.1 エンドポイント案

`/internal/me` 配下に追加する。

```text
GET    /internal/me/locked-songs
POST   /internal/me/locked-songs
DELETE /internal/me/locked-songs/:displayid[?is_ultima={true|false}]
```

登録は `POST /internal/me/locked-songs`、解除は `DELETE /internal/me/locked-songs/:displayid` + query とする。DELETE bodyに依存しないため、クライアント・プロキシ差異の影響を受けにくい。

DELETEの `is_ultima` queryは任意とし、未指定時は `false` として扱う。空文字や `true` / `false` 以外の値は `bad_request` とする。

## 8.2 リクエスト案

登録:

```json
{
  "display_id": "0000000000000123",
  "is_ultima": false
}
```

DBは `song_id` を保持するが、API境界では既存の楽曲APIに合わせて `display_id` を受ける。これにより、内部IDをAPI契約へ直接出さずに済む。

POSTのJSONリクエストは `BindStrictJSON` で厳格にデコードし、未知のトップレベルキーは `bad_request` として拒否する。

解除:

```text
DELETE /internal/me/locked-songs/0000000000000123?is_ultima=false
```

## 8.3 レスポンス案

一覧:

```json
{
  "items": [
    {
      "display_id": "0000000000000123",
      "is_ultima": false
    },
    {
      "display_id": "0000000000000456",
      "is_ultima": true
    }
  ]
}
```

必要に応じて、フロント表示用に楽曲名を含める案もある。ただし容量・責務を優先するなら、まずは最小レスポンスにし、表示に必要な情報は既存の楽曲APIまたはフロント側マスタと突き合わせる。

## 8.4 DTO

新規ファイル:

- `internal/dto/api_internal/player_locked_song_dto.go`

DTOはAPI境界の責務として、Usecase入出力と分離する。

## 8.5 ステータスコード

| 操作 | 成功時ステータス | レスポンス |
| --- | --- | --- |
| `GET /internal/me/locked-songs` | 200 | 一覧JSON |
| `POST /internal/me/locked-songs` | 204 | なし |
| `DELETE /internal/me/locked-songs/:displayid?is_ultima=false` | 204 | なし |

登録・解除は冪等操作のため、既に登録済みの曲を登録しても204、未登録の曲を解除しても204を返す。

## 8.6 エラー仕様

| エラーコード | HTTPステータス | 条件 |
| --- | --- | --- |
| `unauthorized` | 401 | 認証情報がない、またはコンテキストにユーザーがいない |
| `bad_request` | 400 | JSON不正、Content-Type不正、未知トップレベルキー、`is_ultima` queryがboolとして解釈できない |
| `validation_failed` | 422 | DTOレベル必須チェック失敗、`display_id` の形式不正 |
| `player_not_linked` | 404 | 認証ユーザーにプレイヤーが紐づいていない |
| `song_not_found` | 404 | 登録時に、`display_id` に対応する通常楽曲が存在しない、論理削除済み、またはWORLD'S END楽曲 |
| `chart_not_found` | 404 | 登録時に、`is_ultima = true` だが対象楽曲にULTIMA譜面が存在しない |
| `internal_error` | 500 | DB異常、マスタ不整合など |

`display_id` は既存の楽曲APIと同じくパスパラメータまたはJSON文字列として受ける。16文字の小文字16進数として不正な場合は `validation_failed`、形式は正しいが存在しない場合は、登録時には `song_not_found` にする。削除APIは冪等な状態削除として扱うため、形式が正しい `display_id` であれば通常の楽曲存在確認によるエラー判定を行わず、対象レコードが存在しなくても204を返す。削除済み楽曲やWORLD'S END楽曲を外部から区別できないようにするため、登録時の管理対象外楽曲も同じ404とする。一覧取得時はRead Model取得で通常楽曲かつ未削除の楽曲に絞り込む。

ULTIMA譜面未存在は、既存の `chart_not_found` を使う。既存コードでは `chart_not_found` が譜面未検出の意味で定義済みであり、`docs/API.md` でも指定難易度の譜面が存在しない場合のエラーとして使われているため、この用途に新規エラーコードは追加しない。

---

## 9. OP計算APIとの接続方針

OP計算API本体では、次の順序で対象譜面を絞り込む。

1. 通常楽曲のみ対象にする
2. 論理削除済み楽曲を除外する
3. WORLD'S END楽曲を除外する
4. プレイヤーの未解禁設定を一括取得する
5. 譜面難易度に応じて未解禁設定と照合する
6. 残った譜面から楽曲ごとの最大OPを計算する

N+1回避のため、未解禁設定は `ListByPlayerID` で一括取得し、`song_id + is_ultima` のセットとして扱う。

API一覧取得では `display_id` が必要なため、Read Model取得でJOINするか、`song_id` 群に対して `IN` 句によるバルクフェッチを行う。いずれの場合も、未解禁レコード1件ごとに楽曲取得を行う実装は禁止する。

---

## 10. テスト方針

## 10.1 マイグレーション

- MySQL用DDLが既存の `players.id` / `songs.id` 型と一致していること
- 複合主キーで重複登録できないこと
- `players` 削除時に連動削除されること
- `songs` 物理削除時に連動削除されること

## 10.2 Repository

- プレイヤー単位で未解禁一覧を取得できる
- `is_ultima = false` と `is_ultima = true` を同一曲で別レコードとして扱える
- 重複登録が成功扱いになる
- 未登録解除が成功扱いになる
- 削除専用の楽曲ID解決ポートとRepository連携で、存在しない楽曲・論理削除済み楽曲・未登録レコードのいずれも成功扱いになる
- 解除後に一覧へ出ない
- Read Model取得では `display_id` を返せる
- Read Model取得では論理削除済み楽曲が一覧へ出ない
- Read Model取得ではWORLD'S END楽曲が一覧へ出ない
- Read Model取得では `display_id ASC, is_ultima ASC` の順序で返る
- `SELECT *` を使っていない

同一曲に `is_ultima = false` と `is_ultima = true` が共存しても、一覧取得で両方返せることを確認する。OP計算用の判定は後続のOP計算API側でテストする。

## 10.3 Usecase

- 認証ユーザーに紐づくプレイヤーの未解禁曲だけを操作できる
- 他ユーザーのプレイヤーIDを直接指定できない
- 存在しない楽曲を登録できない
- 論理削除済み楽曲を登録できない
- WORLD'S END楽曲を登録できない
- ULTIMA譜面が存在しない楽曲を `is_ultima = true` で登録すると `chart_not_found` になる
- 登録・解除が冪等である
- 存在しない楽曲・論理削除済み楽曲の解除は成功扱いになる
- 通常譜面群未解禁とULTIMAのみ未解禁を同一曲に共存させられる
- 通常譜面群未解禁を解除しても、ULTIMAのみ未解禁レコードは削除されない
- ULTIMAのみ未解禁を解除しても、通常譜面群未解禁レコードは削除されない

## 10.4 Handler

- `GET /internal/me/locked-songs` が認証必須で一覧を返す
- `POST /internal/me/locked-songs` が妥当な入力で登録する
- `POST /internal/me/locked-songs` は厳格JSONデコードを行い、未知トップレベルキーを `bad_request` にする
- `DELETE /internal/me/locked-songs/:displayid?is_ultima=false` が解除する
- DELETEの `is_ultima` 未指定時は `false` として扱う
- DELETEの `is_ultima` が空文字またはboolとして解釈できない値の場合は `bad_request` にする
- DELETEは形式が正しい `display_id` であれば通常の楽曲存在確認によるエラー判定を行わず、未登録でも204を返す
- 不正な `display_id` は `validation_failed`、不正な `is_ultima` queryは `bad_request` になる

---

## 11. ドキュメント更新

実装時に更新するドキュメント:

- `docs/API.md`
  - 未解禁曲管理APIを追加
- `docs/overpower_calculation.md`
  - OP計算対象から未解禁曲を除外する旨を追加
- `docs/er_diagram.puml`
  - `player_locked_songs` を追加
- 必要に応じて `docs/domain_model_specification.md`
  - 未解禁曲管理モデルの責務を追加

---

## 12. 実装タスク分解

1. MySQLマイグレーションを追加する
2. `migration/schema_mysql.sql` とER図を更新する
3. `PlayerLockedSong` エンティティを追加する
4. `PlayerLockedSongRepository` インターフェースを追加する
5. Infraリポジトリ実装とテストを追加する
6. Usecaseインターフェース・実装・テストを追加する
7. API DTOを追加する
8. Handlerを追加する
9. RouterとDIに追加する
10. API一覧用のRead Model取得ポートとInfra実装を追加する
11. 削除専用の楽曲ID解決ポートとInfra実装を追加する
12. `song_not_found` / `chart_not_found` / `player_not_linked` へのUsecaseエラー変換を追加する
13. `docs/API.md` を更新する
14. `docs/overpower_calculation.md` を更新する
15. `go test ./...` を通す
16. `gofmt` を実行する
17. AGENTS.mdに基づくセルフレビューを行い、改善と再テストを繰り返す

---

## 13. 確定事項

## 13.1 解除APIの形式

採用:

- `DELETE /internal/me/locked-songs/:displayid?is_ultima=false`

DELETE bodyに依存しないため、クライアント・プロキシ差異の影響を受けにくい。

## 13.2 登録・解除の冪等性

採用:

- 登録済みを再登録したら成功扱い
- 未登録を解除しても成功扱い

手動管理UIからの再送や二重クリックに強く、状態管理APIとして扱いやすいため、冪等操作にする。

## 13.3 論理削除済み楽曲の登録可否

採用:

- 登録時に拒否する

ユーザーが実際に管理する必要のない曲を未解禁リストに入れられない方が分かりやすいため、登録時に拒否する。API一覧取得用のRead Modelでも `songs.is_deleted = 0` で除外する。

## 13.4 ULTIMA判定の厳密性

採用:

- `is_ultima = false` は通常譜面群（BASIC / ADVANCED / EXPERT / MASTER）の未解禁として扱い、ULTIMA譜面は除外対象に含めない
- `is_ultima = true` はULTIMA譜面だけ未解禁として扱う
- `is_ultima = true` の登録時は、対象楽曲にULTIMA譜面が存在することを検証する
- 通常譜面群とULTIMA譜面の両方が未解禁の場合は、同一曲に `is_ultima = false` と `is_ultima = true` の2レコードを登録する

`is_ultima = false` が `is_ultima = true` を包含する設計にはしない。CHUNITHM上の通常解禁とULTIMA解禁を別の状態として扱い、両方の状態が必要な場合は2レコードで表現する。

## 13.5 ドメインモデルと一覧表示用データの分離

採用:

- `entity.PlayerLockedSong` は `PlayerID`, `SongID`, `IsUltima` のみ保持する
- API一覧用の `display_id` はUsecase出力またはRead Modelで扱う
- N+1回避が必要な一覧取得はJOINまたは `IN` 句によるバルクフェッチで行う

`display_id` はAPI契約上必要だが、未解禁状態そのもののドメイン状態ではないため、ドメインエンティティには含めない。

## 13.6 APIで使う楽曲識別子

採用:

- `display_id`

DB内部では `song_id` を使う。APIでは、既存の楽曲APIが `display_id` をパスパラメータとして使っているため、フロントエンドから操作する管理APIも `display_id` を受ける。登録時はUsecaseで通常楽曲であること、論理削除されていないこと、WORLD'S END楽曲ではないことを検証したうえで `song_id` に変換する。解除時は論理削除済み楽曲の未解禁レコードも消せるように、通常の楽曲取得ではなく削除専用の楽曲ID解決ポートで `song_id` を解決する。未解禁リポジトリ自体は `song_id` と `is_ultima` を引数に取る `Delete` を提供し、Repository境界にAPI用の `display_id` を持ち込まない。

DB内部IDへの依存をAPI契約に出さず、既存の `/internal/songs/:displayid` と揃えられるため `display_id` を採用する。

## 13.7 楽曲未検出エラーコード

採用:

- `song_not_found`

既存 `docs/API.md` では通常楽曲API・WORLD'S END楽曲APIともに楽曲未検出を `song_not_found` としている。未解禁設定の登録時に対象楽曲が存在しない場合も同じ意味のため、`player_locked_song_not_found` は採用しない。未解禁設定レコード自体が存在しない場合は、解除操作を冪等にするためエラーにしない。

## 13.8 ULTIMA譜面未検出エラーコード

採用:

- `chart_not_found`

既存 `docs/API.md` では、指定された難易度の譜面が存在しない場合に `chart_not_found` を使っている。`is_ultima = true` の登録時に対象楽曲へULTIMA譜面が存在しないケースも「指定譜面が存在しない」状態であるため、未解禁曲管理専用の新規エラーコードは追加しない。

## 13.9 解除時の楽曲存在確認

採用:

- 削除APIでは通常の楽曲存在確認によるエラー判定を行わない
- `display_id` に該当する楽曲が存在しない場合も204を返す
- 論理削除済み楽曲に紐づく未解禁レコードも削除できる

存在しない楽曲の削除操作をエラーにすると、楽曲が論理削除された後に未解禁レコードをユーザー操作で消せなくなる。そのため、解除APIは状態削除の冪等操作として扱い、形式が正しい `display_id` であれば削除対象なしでも成功扱いにする。実装では通常の `SongRepository` で楽曲を事前取得せず、削除専用の楽曲ID解決ポートで `song_id` を解決できた場合だけ `PlayerLockedSongRepository.Delete` で削除する。

---

## 14. 結論

今回の未解禁曲管理は、日付・履歴・論理削除を持たない最小構成で進めるのが妥当である。

`player_locked_songs` は `player_id`, `song_id`, `is_ultima` の複合主キーだけを持つことで、容量を抑えつつ、通常譜面群未解禁とULTIMA単独未解禁の両方を表現できる。

後続のOP計算APIでは、プレイヤー単位で未解禁設定を一括取得してセット化することで、N+1を避けながら計算対象の除外に利用する。
