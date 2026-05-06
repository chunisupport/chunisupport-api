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

未解禁曲管理テーブルは、プレイヤーごとの「未解禁である楽曲またはULTIMA譜面」を保持する。

基本ルール:

- レコードあり: 未解禁として扱う
- レコードなし: 解禁済み、または未解禁管理対象外として扱う
- `is_ultima = false`: 楽曲全体が未解禁。対象楽曲にULTIMA譜面がある場合、そのULTIMA譜面も未解禁として扱う
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

楽曲の論理削除は、OP計算側・一覧表示側で `songs.is_deleted = 0` に絞り込んでから扱う。

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
    DisplayID string
    IsUltima  bool
}
```

このエンティティは永続化タグを持たない。`DisplayID` はAPI一覧返却用にJOINで取得する読み取り値であり、DB主キーとしては扱わない。既存方針どおり、DB用構造体が必要な場合は `internal/infra/models` に分離する。

## 5.2 ドメイン上の意味

`PlayerLockedSong` は「未解禁状態の現在値」を表す。履歴や設定日時は責務に含めない。

`is_ultima` の判定規則:

- 対象譜面がULTIMAの場合、`is_ultima = false` または `is_ultima = true` の未解禁設定があればOP計算対象から除外する
- 対象譜面がULTIMA以外の場合、`is_ultima = false` の未解禁設定があればOP計算対象から除外する

楽曲全体が未解禁の場合は `is_ultima = false` の1レコードだけを登録し、ULTIMA用の追加レコードは作らない。これにより、通常解禁単位の登録・解除は常に1レコード操作で済む。

ULTIMA譜面だけが未解禁の場合のみ `is_ultima = true` を登録する。`is_ultima = true` の登録時は、対象楽曲にULTIMA譜面が存在することを検証する。

通常運用では同一曲に `is_ultima = false` と `is_ultima = true` の両方を登録する操作導線は想定しない。ただし、既存レコードを正規化するために別の未解禁レコードを自動削除・書き換えすることもしない。万一両方のレコードが存在した場合は、`is_ultima = false` が楽曲全体未解禁として機能するため、OP計算上はULTIMAを含む全譜面を除外する。

---

## 6. リポジトリ設計

## 6.1 インターフェース

新規ファイル:

- `internal/domain/repository/player_locked_song_repository.go`

想定インターフェース:

```go
type PlayerLockedSongRepository interface {
    ListActiveByPlayerID(ctx context.Context, exec Executor, playerID int) ([]*entity.PlayerLockedSong, error)
    Create(ctx context.Context, exec Executor, lockedSong *entity.PlayerLockedSong) error
    Delete(ctx context.Context, exec Executor, playerID int, songID int, isUltima bool) error
}
```

OP計算ではプレイヤー単位で一括取得してメモリ上のセットにするのが効率的なため、`ListActiveByPlayerID` を主要メソッドとする。

`ListActiveByPlayerID` は `songs` とJOINし、`songs.is_deleted = 0` のレコードだけを返す。APIレスポンスでも `display_id` が必要になるため、未解禁管理リポジトリ側で `songs.display_id` も同時に取得する。Usecaseで `song_id` ごとに `SongRepository` を呼ぶとN+1になりやすく、`SongRepository.FindByIDs` は今回の用途以外で明確な必要性がないため追加しない。

このリポジトリは「未解禁集合」への追加・削除を扱うため、エンティティ全体を `Save` する集約リポジトリとは性質が異なる。`player_locked_songs` は独立した属性を持つ集約ではなく、プレイヤーに紐づく現在状態の集合であるため、`Create` / `Delete` による集合操作を例外的に許容する。

## 6.2 Infra実装

新規ファイル:

- `internal/infra/repository/player_locked_song_repository_impl.go`
- `internal/infra/repository/player_locked_song_repository_impl_test.go`

実装方針:

- `SELECT *` は使用しない
- `ListActiveByPlayerID` は `player_locked_songs.player_id, player_locked_songs.song_id, player_locked_songs.is_ultima, songs.display_id` を明示取得する
- `Create` は重複キーを成功扱いにする
- `Delete` は対象なしでも成功扱いにする

管理APIの使いやすさを優先し、登録・解除は冪等操作にする。登録時は `INSERT IGNORE` 相当、または重複キーエラーを成功扱いに変換する。解除時は `DELETE` の影響行数が0件でも成功扱いにする。

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

- 存在しない楽曲は `player_locked_song_not_found` として404を返す
- 論理削除済み楽曲は登録不可とし、`player_locked_song_not_found` として404を返す
- `is_ultima = true` の登録時にULTIMA譜面が存在しない場合は `player_locked_song_ultima_not_found` として400を返す

管理対象はOP計算対象曲であるため、論理削除済み楽曲の登録は拒否する。既に登録済みの曲が後から論理削除された場合は、一覧取得・OP計算のどちらでも `songs.is_deleted = 0` により除外する。

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

## 8.2 リクエスト案

登録:

```json
{
  "display_id": "0000000000000123",
  "is_ultima": false
}
```

DBは `song_id` を保持するが、API境界では既存の楽曲APIに合わせて `display_id` を受ける。これにより、内部IDをAPI契約へ直接出さずに済む。

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
| `player_locked_song_not_found` | 404 | `display_id` に対応する通常楽曲が存在しない、または論理削除済み |
| `player_locked_song_ultima_not_found` | 400 | `is_ultima = true` だが対象楽曲にULTIMA譜面が存在しない |
| `internal_error` | 500 | DB異常、マスタ不整合など |

`display_id` は既存の楽曲APIと同じくパスパラメータまたはJSON文字列として受ける。16文字の小文字16進数として不正な場合は `validation_failed`、形式は正しいが存在しない場合は `player_locked_song_not_found` にする。削除済み楽曲を外部から区別できないようにするため、論理削除済みも同じ404とする。

---

## 9. OP計算APIとの接続方針

OP計算API本体では、次の順序で対象譜面を絞り込む。

1. 通常楽曲のみ対象にする
2. 論理削除済み楽曲を除外する
3. WORLD'S END楽曲を除外する
4. プレイヤーの未解禁設定を一括取得する
5. 譜面難易度に応じて未解禁設定と照合する
6. 残った譜面から楽曲ごとの最大OPを計算する

N+1回避のため、未解禁設定は `ListActiveByPlayerID` で一括取得し、`song_id + is_ultima` のセットとして扱う。

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
- `is_ultima = false` があるULTIMA譜面は未解禁扱いになる
- 重複登録が成功扱いになる
- 未登録解除が成功扱いになる
- 解除後に一覧へ出ない
- 論理削除済み楽曲が一覧へ出ない
- `SELECT *` を使っていない

同一曲に `is_ultima = false` と `is_ultima = true` が共存する状態は通常運用では作らないが、データ上共存しても一覧取得・OP計算用の判定が破綻しないことを確認する。

## 10.3 Usecase

- 認証ユーザーに紐づくプレイヤーの未解禁曲だけを操作できる
- 他ユーザーのプレイヤーIDを直接指定できない
- 存在しない楽曲を登録できない
- 論理削除済み楽曲を登録できない
- ULTIMA譜面が存在しない楽曲を `is_ultima = true` で登録できない
- 登録・解除が冪等である

## 10.4 Handler

- `GET /internal/me/locked-songs` が認証必須で一覧を返す
- `POST /internal/me/locked-songs` が妥当な入力で登録する
- `DELETE /internal/me/locked-songs/:displayid?is_ultima=false` が解除する
- DELETEの `is_ultima` 未指定時は `false` として扱う
- 不正な `display_id` / `is_ultima` でバリデーションエラーになる

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
10. APIエラーコードとUsecaseエラー変換を追加する
11. `docs/API.md` を更新する
12. `docs/overpower_calculation.md` を更新する
13. `go test ./...` を通す
14. `gofmt` を実行する
15. AGENTS.mdに基づくセルフレビューを行い、改善と再テストを繰り返す

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

ユーザーが実際に管理する必要のない曲を未解禁リストに入れられない方が分かりやすいため、登録時に拒否する。一覧取得時も `songs.is_deleted = 0` で除外する。

## 13.4 ULTIMA判定の厳密性

採用:

- `is_ultima = false` は楽曲全体未解禁として扱い、ULTIMA譜面も除外対象に含める
- `is_ultima = true` はULTIMA譜面だけ未解禁として扱う
- `is_ultima = true` の登録時は、対象楽曲にULTIMA譜面が存在することを検証する

楽曲全体未解禁の場合は `is_ultima = false` の1レコードのみ登録する。ULTIMA用の2レコード目は作らない。

通常運用では同一曲に `is_ultima = false` と `is_ultima = true` の両方を登録する導線は想定しない。ただし、既存レコードの自動削除や書き換えによる正規化は行わない。万一両方が存在する場合は `is_ultima = false` を楽曲全体未解禁として扱う。

## 13.5 APIで使う楽曲識別子

採用:

- `display_id`

DBとRepository内部では `song_id` を使う。APIでは、既存の楽曲APIが `display_id` をパスパラメータとして使っているため、フロントエンドから操作する管理APIも `display_id` を受け、UsecaseまたはRepository境界で `song_id` に変換する。

DB内部IDへの依存をAPI契約に出さず、既存の `/internal/songs/:displayid` と揃えられるため `display_id` を採用する。

---

## 14. 結論

今回の未解禁曲管理は、日付・履歴・論理削除を持たない最小構成で進めるのが妥当である。

`player_locked_songs` は `player_id`, `song_id`, `is_ultima` の複合主キーだけを持つことで、容量を抑えつつ、楽曲全体未解禁とULTIMA単独未解禁の両方を表現できる。

後続のOP計算APIでは、プレイヤー単位で未解禁設定を一括取得してセット化することで、N+1を避けながら計算対象の除外に利用する。
