# バージョン追加・削除API 実装指示書

## 0. 文書の位置付けと確定事項

本書は、CHUNITHMバージョン名と稼働日（`versions`）をDBマイグレーション投入から管理画面API投入へ移行するための実装指示書である。フロントエンドとの合意事項を固定し、実装者が迷わず作業できることを目的とする。

確定事項：

* FKではなく日付範囲で解決する。フロント`src/utils/versionConverter.ts:32-42`と同義。
* `released_at`の更新APIは作らない。誤登録の修正は「曲の`release`を正しい範囲へ移してから削除→作り直し」で行う。
* 削除は「最新のみ＋曲なしのみ」可とする。
* 目標参照は削除禁止条件に入れない。削除後は空扱いでクラッシュしない運用とする。
* ホットリロードで再起動なしに反映する。

現行の参照実装：

* `migration/mysql/000001_init_schema.up.sql:319-349`（`versions`定義と初期投入）
* `migration/mysql/000019_add_chunithm_mate_version.up.sql`（従来の追加手段＝マイグレーション）
* `internal/infra/masterdata/cache.go:59-181`（起動時プリロード、未来版除外）
* `internal/app/handler/compat/reiwa/dto.go:113-137`（日付範囲解決の前例）
* `internal/usecase/goal_usecase_impl.go:424-432,522-534`（`VersionRange`解決の前例）
* `internal/usecase/honor_usecase_impl.go`（ADMIN系CRUDの前例）
* `internal/usecase/system_maintenance_usecase.go:28-35,87-125`（ホットリロードの前例）

---

## 1. 用語と帰属定義

本書で「そのバージョンの曲」とは、次を満たす`song`を指す。`version_id`での紐付けは行わない。

```text
対象版を released_at 昇順（同日は id 昇順）に並べたとき、
区間 = [自released_at, 次released_at)
最新版は区間 = [自released_at, ∞)
song.released_at が区間に入る曲
```

補足：

* `songs.released_at IS NULL`はバージョン不定のため対象外とする。
* `songs.is_deleted`の値にかかわらず件数に含める。削除済みでも履歴解釈に影響するためである。
* API側の判定式は`goal_repository_impl.go:267`と同一とする：`(s.released_at >= ? AND s.released_at < ?)`。最新版は`s.released_at >= ?`のみ。

この定義はフロントの日付範囲解決と一致する。API側にFKや`version_id`カラムを追加してはならない。

---

## 2. 対象範囲

### 2.1 対象

* `POST /internal/admin/versions`：バージョン新規追加（未来日の事前登録可）
* `PUT /internal/admin/versions/:id`：バージョン名のみ変更（`released_at`変更不可）
* `DELETE /internal/admin/versions/:id`：最新かつ曲なしの場合のみ物理削除
* `GET /internal/admin/versions`：未来版を含む全件一覧（管理画面用）
* ホットリロード：上記書き込み成功後に`Cache`へ即時反映
* 既存の公開一覧（`GET /internal/master/versions`、`GET /v1/master/versions`、`GET /compat/reiwa/1/chunithm_versions.json`）の仕様維持（リリース済みのみ返却）

### 2.2 対象外

* `released_at`の更新API。作らないことが詰み防止策の一部である。
* 中間バージョンの削除。最新のみ許可する。
* 目標参照を理由とする削除禁止。将来の拡張余地として残すのみ。
* `versions`テーブルのDDL変更。新規マイグレーションは不要。過去マイグレーションは不変とする。
* `INSERT INTO versions`形式のデータ投入マイグレーションの追加。今後は本APIで投入する。

---

## 3. エンドポイント仕様

すべてFirebase Bearer＋`requireAdmin`とする（`router.go:510-518`の`adminGroup`配下）。EDITORは実行不可。

### 3.1 `POST /internal/admin/versions`

発表時の事前登録用。未来日を許可する。

リクエスト：

```json
{
  "name": "CHUNITHM Mate",
  "released_at": "2026-07-02"
}
```

| フィールド | 型 | 必須 | バリデーション |
| --- | --- | --- | --- |
| `name` | string | 必須 | 前後空白除去後1〜50文字。`CHUNITHM `接頭辞を必須とする。`versions.name`の一意制約に従う |
| `released_at` | string | 必須 | `YYYY-MM-DD`形式の日付のみ。既存版と同日不可 |

レスポンス：`201 Created`で作成行（`id`、`name`、`released_at`）を返す。DTO形状は`dto.VersionDTO`（`internal/dto/master_data_dto.go:16`）と同一とする。

### 3.2 `PUT /internal/admin/versions/:id`

バージョン名のtypo修正専用。`released_at`は受け付けない。曲の有無にかかわらず実行可（範囲が変わらないため）。

リクエスト：

```json
{
  "name": "CHUNITHM VERSE"
}
```

`:id`は正の整数のみ。存在しないIDは`404`とする。

### 3.3 `DELETE /internal/admin/versions/:id`

次をすべて満たす場合のみ`204 No Content`で物理削除する。

1. 対象が最新版である（`ORDER BY released_at DESC, id DESC LIMIT 1`と一致）。
2. 区間内に曲が1件もない（`SELECT 1 FROM songs WHERE released_at >= ? LIMIT 1`が0件）。

いずれかを満たさない場合は削除せず、フロントは選択肢を非表示にする。

### 3.4 `GET /internal/admin/versions`

未来版を含む全件を`released_at`昇順で返す。公開用一覧と異なり日付フィルタを適用しない。レスポンス形状は公開用と同一の配列とする。

既存の公開系は変更しない：

* `GET /internal/master/versions`
* `GET /v1/master/versions`
* `GET /compat/reiwa/1/chunithm_versions.json`

これらは引き続き`released_at <= 当日(JST)`のみ返す。

---

## 4. エラー仕様

| HTTP | エラーコード | 条件 |
| --- | --- | --- |
| 400 | `validation_failed` | JSON不正、`:id`形式不正 |
| 422 | `invalid_version_input` | 名前空・長さ超過・接頭辞なし、日付形式不正・同日重複 |
| 404 | `version_not_found` | 対象IDが存在しない |
| 409 | `version_name_conflict` | 名前重複（`versions.name` UNIQUE） |
| 409 | `version_not_latest` | 最新版ではない削除要求 |
| 409 | `version_in_use` | 区間内に曲が存在する削除要求 |
| 401 | `missing_token` / `invalid_token` | 認証情報がない、または不正 |
| 403 | `forbidden` | ADMIN以外によるアクセス |

新規エラー値は`internal/domain/repository/errors.go`に`ErrVersionNotFound`・`ErrVersionConflict`、`internal/usecase/errors.go`に`ErrInvalidVersionInput`・`ErrVersionNotLatest`・`ErrVersionInUse`として定義し、`internal/app/apierror/mapping.go`で上表へ対応付ける。`honor`の`ErrHonorNotFound`→`ErrNotFound`対応（`mapping.go:97`）を前例とする。

---

## 5. アーキテクチャ設計

依存方向は`handler`→`usecase`→`repository`（I/F）←`repository`（実装）を維持する。`usecase`から`infra`へのimportは厳禁。`entity`に`db`/`json`タグを付けない。`SELECT *`禁止、N+1禁止。

### 5.1 Domain

新規：`internal/domain/entity/version.go`

* 純粋なGo構造体とする。`db`/`json`タグ禁止。
* フィールドは`ID int`、`Name string`、`ReleasedAt time.Time`（日付部のみ有効）とする。
* 豊かなモデルとする：コンストラクタ`NewVersion(name string, releasedAt time.Time)`で名前と日付を検証し、コマンドメソッドは`Rename(newName string)`のみ持つ。`Reschedule`は作らない（`released_at`更新APIを作らないため）。
* 名前検証はUsecaseと重複させず、エンティティ側を正とする。

新規：`internal/domain/repository/version_repository.go`

```go
type VersionRepository interface {
    FindAll(ctx context.Context, exec Executor) ([]*entity.Version, error)
    FindByID(ctx context.Context, exec Executor, id int) (*entity.Version, error)
    FindByName(ctx context.Context, exec Executor, name string) (*entity.Version, error)
    FindLatest(ctx context.Context, exec Executor) (*entity.Version, error)
    ExistsSongInRange(ctx context.Context, exec Executor, from time.Time, to *time.Time) (bool, error)
    Create(ctx context.Context, exec Executor, version *entity.Version) (*entity.Version, error)
    Save(ctx context.Context, exec Executor, version *entity.Version) error
    Delete(ctx context.Context, exec Executor, id int) error
}
```

備考：

* 第一引数は必ず`context.Context`とする。
* `ExistsSongInRange`は`songs`を参照するが、バージョン集約の削除可否判定に必要な最小境界として本I/Fに置く。`to == nil`のときは`s.released_at >= ?`のみで判定する。
* 部分更新メソッドは作らず、エンティティ変更＋`Save`パターンに統一する。
* ホットリロード用に次も同ファイルへ定義する。`usecase`が`infra/masterdata`へ依存しないための境界である。

```go
type VersionCacheReloader interface {
    ReloadVersions(ctx context.Context) error
}
```

### 5.2 Infrastructure

新規：`internal/infra/models/version_model.go`

* `ToEntity()`／`FromEntity()`で`entity.Version`と相互変換する。`master_model.go`と同型。
* `released_at`は`DATE`として日付部のみ扱い、時刻・タイムゾーンを持ち込まない。

新規：`internal/infra/repository/version_repository_impl.go`

* 全SQLでカラムを明示する。例：`SELECT id, name, released_at FROM versions WHERE id = ?`。
* `FindLatest`は`SELECT id, name, released_at FROM versions ORDER BY released_at DESC, id DESC LIMIT 1`とする。同日並びの決定的順序のため`id DESC`を付ける。
* `ExistsSongInRange`は`SELECT 1 FROM songs WHERE released_at >= ? AND released_at < ? LIMIT 1`（`to == nil`時は後半なし）とする。`COUNT(*)`ではなく存在確認に留める。
* 名前重複はMySQL重複エラーを`ErrVersionConflict`へ変換する。`honor_repository_impl.go:72-79`の`wrapHonorDuplicateError`を前例とする。
* 削除時は`DELETE FROM versions WHERE id = ?`とし、対象なしは`ErrVersionNotFound`とする。

既存変更：`internal/infra/masterdata/cache.go`

* `Cache`に`sync.RWMutex`を追加し、`Versions`／`VersionsByID`の直接読み書きをやめる。
* `ReloadVersions(ctx, db)`を追加：全件`SELECT id, name, released_at FROM versions`→新map構築→Writeロック内でスワップ。DB保存成功後のみ公開する点は`systemMaintenanceUsecase.Update:120-125`と同一である。
* 公開用と管理用の2ビューを用意する：
  * 管理用：全件（未来版含む）
  * 公開用：`released_at <= 当日(JST)`のみ。判定日の取得は起動時1回ではなく読取時に行い、テスト容易のため時計を注入できる形にする。
* 既存の直接参照（例：`reiwa/dto.go:119`の`cache.VersionsByID` range）をアクセサ経由へ置き換える。`GoalMasters()`／`MasterDataMasters()`の`maps.Clone`方針は維持する。

### 5.3 Usecase

新規：`internal/usecase/version_usecase.go`、`internal/usecase/version_usecase_impl.go`

```go
type VersionUsecase interface {
    ListAll(ctx context.Context) ([]*entity.Version, error)
    Create(ctx context.Context, name string, releasedAt time.Time) (*entity.Version, error)
    Rename(ctx context.Context, id int, newName string) (*entity.Version, error)
    Delete(ctx context.Context, id int) error
}
```

責務：

1. 入力検証（空・長さ・接頭辞・日付形式・同日重複・ID正値）をエンティティと協調して行う。DTO型に依存しない。
2. `tm.Transactional`内でDB操作を行う。`honorUsecaseImpl.Create:50-60`を前例とする。
3. 同時書き込みの直列化に`updateMu sync.Mutex`を持つ。`systemMaintenanceUsecase:34,93`と同一である。
4. `Delete`は同一トランザクション内で次の順序とする：
   1. `FindByID`（`FOR UPDATE`で行ロック）
   2. `FindLatest`とID一致確認（不一致は`ErrVersionNotLatest`）
   3. `ExistsSongInRange(From: 対象released_at, To: nil)`確認（存在は`ErrVersionInUse`）
   4. `Delete`
5. DBコミット成功後のみ`VersionCacheReloader.ReloadVersions`を呼ぶ。リロード失敗はDB巻き戻し不可のためログ記録＋エラー返却とし、次回読取で再試行できる状態に留める。
6. `Rename`は曲有無を問わない。`released_at`変更要求はI/F自体に存在させないことで誤用を防ぐ。

`router.go`では`Cache`（`VersionCacheReloader`実装）を本Usecaseへ注入する。

### 5.4 Presentation

新規：

* `internal/dto/api_internal/version_dto.go`（要求・応答DTO）
* `internal/app/handler/api_internal/version_handler.go`（ADMIN専用ハンドラ）

ハンドラ責務はパス検証・Usecase呼び出し・DTO変換に限定し、ドメイン判定を流出させない。`honor_handler.go`を前例とする。日付入出力は`YYYY-MM-DD`文字列とし、`time.DateOnly`で変換する。

ルーティング（`internal/app/router.go`の`adminGroup`へ追加）：

```text
GET    /internal/admin/versions
POST   /internal/admin/versions
PUT    /internal/admin/versions/:id
DELETE /internal/admin/versions/:id
```

---

## 6. 詰み防止の運用手順

誤った`released_at`で追加した場合の復旧手順を固定する。本APIに`released_at`修正機能はないため、次の手順が唯一の正規手順である。

1. 誤って作ったバージョンの区間に入っている曲を特定する（管理一覧＋楽曲一覧で`released_at`確認）。
2. 既存の楽曲更新API（`PUT /internal/songs`、EDITOR以上）で該当曲の`released_at`を正しい範囲へ移す。
3. 誤バージョンが最新かつ曲なしになったことを確認する。
4. `DELETE /internal/admin/versions/:id`で削除する（ホットリロードで即時反映）。
5. 必要であれば正しい日付で`POST /internal/admin/versions`し直す。

この手順を残すことで「曲が紐付いて削除禁止→直せない」の詰みを回避する。中間版の削除は手順のいかんにかかわらず禁止とする。

---

## 7. DB・性能設計

* `SELECT *`禁止。必要カラムのみ明示する。
* 削除可否は`LIMIT 1`の存在確認とし、`COUNT(*)`の全走査にしない。
* 譜面単位の取得や目標単位のループでバージョンを1件ずつ引かない。N+1禁止。
* 新規インデックスは追加しない。`versions`は件数が少なく、`songs.released_at`の範囲確認は既存運用で許容できる想定である。実測で問題が出た場合のみ検討する。
* 日付比較は`DATE`同士で行い、時刻・タイムゾーン変換を持ち込まない。公開可否の当日判定のみJST（`Asia/Tokyo`）を用いる。`cache.go:163-173`の現行方針を踏襲する。

---

## 8. テスト設計

TDD（Red→Green→Refactor）で実装する。アサーションは`assert`、前提条件・`nil`直後の参照は`require`とする。既存テストの改変・削除は行わない。

### 8.1 Usecaseテスト（テーブルテスト＋Given-When-Then）

* 正常な作成が`released_at`昇順で保存される
* 名前空・長さ超過・接頭辞なし・日付形式不正・同日重複を拒否する
* 名前重複を`ErrVersionConflict`として返す
* 改名が曲ありでも成功し、`released_at`が不変である
* 最新かつ曲なしの削除が成功する
* 中間版の削除が`ErrVersionNotLatest`になる
* 曲あり最新版の削除が`ErrVersionInUse`になる
* `released_at IS NULL`の曲が削除可否に影響しない
* DB成功後に`ReloadVersions`が1回呼ばれる

### 8.2 Repositoryテスト

* `FindLatest`が`released_at DESC, id DESC`で決定的に返る
* `ExistsSongInRange`が`[from, to)`境界どおりに判定する（`to == nil`含む）
* 存在しないIDの取得・削除が`ErrVersionNotFound`になる
* 明示カラムで取得でき`SELECT *`を使わない

### 8.3 Handlerテスト

* 正常系のDTO形状とステータス（作成`201`、改名`200`、削除`204`、管理一覧`200`）
* `:id`形式不正、未認証、`forbidden`の対応付け
* `version_not_latest`・`version_in_use`のHTTP `409`対応付け

### 8.4 性能・回帰確認

* `go test ./...`が100%成功すること
* `GET /internal/master/versions`が未来版を含まないこと
* `GET /internal/admin/versions`が未来版を含むこと
* 書き込み後に再起動なしで両一覧へ反映されること
* 既存の目標・reiwa・バッチ関連テストが破綻しないこと

---

## 9. ドキュメント更新

実装時に次を更新する。

* `docs/API.md`：エンドポイント一覧、管理バージョンAPI群、リクエスト・レスポンス、エラー仕様、誤登録時の復旧手順
* `docs/master_data_preload_policy.md`：`versions`のホットリロード対応と2ビュー（公開・管理）の説明、「再起動が必要」の条件緩和
* `internal/app/apierror/codes.go`：新規エラーコード追加
* フロント連携：管理一覧・作成・改名・削除のDTOとエラーコード共有、削除ボタン表示条件（最新かつ曲なし）の実装

---

## 10. 実装タスク分解

1. `domain/entity/version.go`をTDDで追加（`NewVersion`＋`Rename`）
2. `domain/repository/version_repository.go`と`VersionCacheReloader`を追加
3. 既存・新規エラー値を`domain/repository/errors.go`と`usecase/errors.go`へ追加
4. `infra/models/version_model.go`を追加（`ToEntity`／`FromEntity`）
5. `infra/repository/version_repository_impl.go`をTDDで追加（明示カラム・重複変換・`FindLatest`・`ExistsSongInRange`・`Delete`）
6. `infra/masterdata/cache.go`へ`RWMutex`・`ReloadVersions`・2ビューを追加し、直接参照をアクセサへ置換
7. `usecase/version_usecase*.go`をTDDで追加（`tm.Transactional`＋`updateMu`＋リロード連携）
8. `dto/api_internal/version_dto.go`と`handler/api_internal/version_handler.go`を追加
9. `router.go`の`adminGroup`へ4ルートを追加し、`VersionCacheReloader`を注入
10. `apierror`の対応付けを追加
11. Usecase・Repository・Handlerテストを追加
12. `docs/API.md`と`docs/master_data_preload_policy.md`を更新
13. `go test ./...`を実行し100%成功を確認
14. `gofmt -s -w .`を実行（`.git/logs`・`vendor/`除外）

---

## 11. 受け入れ条件

* 発表時に未来日でバージョン追加でき、管理一覧に即時表示される
* 公開一覧・v1・reiwa互換に未来版が漏れない
* バージョン名のtypoを曲ありでも修正できる
* `released_at`を変更するAPIが存在しない
* 中間版の削除が拒否される
* 曲あり最新版の削除が拒否される
* 最新かつ曲なしの削除が再起動なしで反映される
* 誤登録時に「曲移動→削除→作り直し」で復旧できる
* 削除済み版を参照する既存目標がクラッシュしない
* N+1と`SELECT *`がない
* `usecase`が`infra`へ依存していない
* `go test ./...`と`gofmt`がエラーなく通る

---

## 12. 将来検討

本実装には含めない。

* 目標参照を削除禁止条件へ追加（`goals.attributes`のJSON参照カウント）
* `versions.released_at`への一意制約DDL（MySQLのDDL暗黙コミットに注意し、別途移行計画が必要）
* 管理画面での削除可否プレビュー（最新判定＋曲件数の事前表示）
* バージョン結合・マージ機能（中間版の安全な削除手段）
* 削除・改名の監査ログと操作者記録
