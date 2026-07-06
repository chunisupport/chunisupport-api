# プレイヤー Rating・OVER POWER 再計算バッチ 設計・実装指示書

## 1. 文書の目的

本書は、プレイヤーが新しいスコアデータを登録しなくても、最新の楽曲・譜面マスタに基づいて次の保存値を日次更新するバッチの確定仕様と実装指示をまとめたものである。

- 計算レーティング
- ベスト枠平均レーティング
- 新曲枠平均レーティング
- OVER POWER値
- 旧バージョンの公式枠が残っているプレイヤーのRating枠

対象の保存先は次のカラムである。

- `players.calculated_player_rating`
- `players.best_average_rating`
- `players.new_average_rating`
- `players.overpower_value`
- `player_records.slot_id`
- `player_records.slot_order`

`overpower_percent` は保存しない。従来どおり、APIレスポンス時に最新マスタとプレイヤーの未解禁設定から算出する。

本書は実装時の判断基準であり、曖昧な場合は既存実装ではなく本書の確定仕様を優先する。ただし、過去マイグレーションの変更は禁止する。

---

## 2. 背景

RatingとOVER POWERは譜面定数に依存する。次の理由により、保存済みの計算値はプレイヤーがスコアを更新していなくても古くなる。

- ゲームバージョン変更に伴う譜面定数変更
- 追加直後で譜面定数が未判明だった譜面の定数確定
- 楽曲・譜面マスタの修正
- ゲームバージョン切り替えによる新曲枠からベスト枠への移行

現状は `PlayerDataUsecase.Register()` の実行時にしかRatingとOVER POWER値が再計算されない。そのため、ユーザーが能動的にデータを登録しない限り、保存値が不正確なまま残る。

本バッチは処理時間が長くなることを許容し、1日1回、夜間に全プレイヤーを順次処理する。

---

## 3. 重要な用語

### 3.1 運用日

ゲームバージョンの稼働開始時刻は、リリース日の日本時間07:00とする。

バッチ開始時刻を `startedAt` とし、次の方法で運用日を1回だけ決定する。

```go
operationalDate := startedAt.In(jst).Add(-7 * time.Hour)
```

実際には時刻部分を捨てた日付値として扱う。

例:

| 現在時刻 | 運用日 |
|---|---|
| 2026-07-06 06:59 JST | 2026-07-05 |
| 2026-07-06 07:00 JST | 2026-07-06 |

長時間実行中に07:00を跨いでも、同一run内では開始時に固定した運用日を使い続ける。

### 3.2 現行バージョン

次を満たす `versions` のうち、`released_at` が最も新しいものを現行バージョンとする。

```text
versions.released_at <= operationalDate
```

現行バージョンが存在しない場合はマスタ不整合としてバッチ全体を開始前に失敗させる。

現行バージョンの稼働開始日時は次の値である。

```text
versions.released_at 07:00:00 Asia/Tokyo
```

### 3.3 現行版プレイヤー

次を満たすプレイヤーを現行版プレイヤーとする。

```text
players.last_played_at >= 現行バージョン稼働開始日時
```

### 3.4 旧版プレイヤー

次のいずれかを満たすプレイヤーを旧版プレイヤーとする。

- `last_played_at` が現行バージョン稼働開始日時より前
- `last_played_at IS NULL`

`last_played_at IS NULL` を現行版扱いへ補正してはいけない。

---

## 4. Rating対象楽曲の分類

### 4.1 `songs.is_new` の禁止

`songs.is_new` は「最新のアップデートで追加された楽曲か」を示す値であり、ゲームバージョン単位の新曲枠を示す値ではない。

Rating枠の判定で `songs.is_new` を参照してはいけない。

### 4.2 新曲枠対象

次のいずれかを満たす楽曲を新曲枠対象とする。

- `songs.released_at IS NULL`
- 現行バージョンの `released_at <= songs.released_at <= operationalDate`

`released_at IS NULL` は、追加日の入力がまだ完了していない新曲である可能性が高いため、新曲枠へ分類する。

### 4.3 ベスト枠対象

次を満たす楽曲をベスト枠対象とする。

```text
songs.released_at IS NOT NULL
AND songs.released_at < 現行バージョンのreleased_at
```

### 4.4 Rating対象外

次の楽曲・譜面はRating枠再構築とRating集計から除外する。

- `songs.is_deleted = true`
- `songs.released_at > operationalDate`
- WORLD'S END

通常譜面の `player_records` にWORLD'S ENDが入らないDB構造であっても、バッチ用マスタ投影では `is_worldsend` を確認する。

新曲枠とベスト枠のプールは排他的である。同一譜面を両方へ入れて二重加算してはいけない。

### 4.5 未判明譜面定数

`charts.is_const_unknown = true` の譜面も除外しない。`charts.const` に保存されている暫定定数を使って計算する。

---

## 5. 実行開始時に固定するスナップショット

バッチrun開始時に次を固定し、全プレイヤーで同じ値を使用する。

- 開始時刻
- 運用日
- 現行バージョン
- 現行バージョン稼働開始日時
- 対象プレイヤーID上限
- Rating・OVER POWER計算に必要な楽曲マスタ
- Rating・OVER POWER計算に必要な譜面マスタ

楽曲・譜面マスタは、必要なカラムだけを明示したクエリで一括取得し、不変のmapへ変換する。全プレイヤーのスコアを一括でメモリへ読み込んではいけない。

最低限必要なマスタ項目:

### 楽曲

- ID
- `released_at`
- `is_deleted`
- `is_worldsend`
- `official_idx`

### 譜面

- ID
- 楽曲ID
- 難易度ID
- 譜面定数
- `is_const_unknown`

run途中でマスタが更新されても、当該runは開始時のスナップショットで完了させる。変更は翌日のrunで反映する。これにより、処理順によってプレイヤーごとの計算基準が変わることを防ぐ。

---

## 6. マスタ事前検証

プレイヤー処理を開始する前に、マスタ全体を検証する。

### 6.1 `official_idx`

Rating対象になり得る通常楽曲の `official_idx` は、次の条件を満たさなければならない。

- `strconv.ParseUint(value, 10, 64)` が成功する
- 数値化後の値が楽曲間で重複しない

先頭ゼロは数値比較に影響させない。たとえば `"00123"` と `"123"` は同じ値であり、同時に存在する場合は重複エラーとする。

変換失敗または数値重複がある場合、決定的な順序を作れないため、バッチ全体を開始前に終了コード1で失敗させる。

### 6.2 その他

最低限、次も検証する。

- 現行バージョンが一意に決定できる
- スロットマスタに `none`、`best`、`best_candidate`、`new`、`new_candidate` が存在する
- プレイヤーレコードが参照する譜面をスナップショットから解決できる
- Rating計算に必要な譜面定数が値オブジェクトとして有効

プレイヤー固有の参照不整合は当該プレイヤーだけを失敗させる。全プレイヤーに共通するマスタ不整合はrun全体を開始前に失敗させる。

---

## 7. Ratingの並び順

旧版プレイヤーの本枠選出と候補枠の表示順には、次の比較キーを順番に使用する。

1. 単曲レートの降順
2. 譜面定数の降順
3. スコアの昇順
4. `official_idx` を10進数として解釈した値の昇順

単曲レートは既存の `CalcSingleRating()` と同じ規則で、小数点以下2桁に切り捨てた値を比較する。

浮動小数点数を直接比較キーにしてはいけない。既存の整数スケール計算を共有し、0.01単位の整数値で比較する。

`official_idx` の数値重複は事前検証で禁止するため、上記4キーで必ず決定的な順序になる。

---

## 8. 現行版プレイヤーの処理

現行版プレイヤーについては、公式ソースから保存された本枠を正とする。譜面定数が未判明または変更され、ローカル計算上の単曲レート順と公式順が一致しなくても、枠をローカルで再構築してはいけない。

### 8.1 Rating集計対象

- `slot = best` の最大30件
- `slot = new` の最大20件

次は集計対象外である。

- `best_candidate`
- `new_candidate`
- `none`

候補枠を本枠候補として再ソートし、本枠へ昇格させてはいけない。

### 8.2 公式本枠の検証

保存済みの公式本枠は次の条件を満たさなければならない。

#### ベスト枠

- 件数が30件以下
- `slot_order` がNULLではない
- `slot_order` が1以上30以下
- `slot_order` が重複しない

#### 新曲枠

- 件数が20件以下
- `slot_order` がNULLではない
- `slot_order` が1以上20以下
- `slot_order` が重複しない

異常な公式順をローカルソートで推測して補正してはいけない。当該プレイヤーのトランザクションをロールバックし、失敗としてログへ記録する。

候補枠の順序異常はRating集計に影響しないため、スロットを変更せず警告ログだけを出す。

### 8.3 保存内容

公式スロットと `slot_order` は一切変更しない。

開始時マスタスナップショットの最新譜面定数を使い、公式本枠だけから次を再計算する。

- `calculated_player_rating`
- `best_average_rating`
- `new_average_rating`

プレイヤーレーティングの除数は、既存仕様どおり本枠数にかかわらず50固定とする。ベスト平均と新曲平均は、各枠に実在する件数を除数とし、0件の場合は0とする。

---

## 9. 旧版プレイヤーの枠再構築

旧版プレイヤーは、保存済みの全通常譜面スコアから、現在のゲームバージョン基準で枠を再構築する。

### 9.1 本枠

- ベスト枠対象プールを規定の順序で並べ、先頭30件を `best` とする
- 新曲枠対象プールを規定の順序で並べ、先頭20件を `new` とする

プールの件数が本枠定員以下の場合は、対象レコードをすべて本枠へ入れる。

選出順に1から `slot_order` を設定する。

### 9.2 候補枠の意味

候補枠には、現在は本枠外だが、その譜面のスコアが伸びれば本枠へ入れるレコードだけを載せる。

ベスト候補と新曲候補は、それぞれ対応する排他的なプール内で判定する。

### 9.3 候補判定

本枠外レコードごとに次を実行する。

1. 現在スコアが `1_009_000` 以上なら候補外とする
2. 当該レコードだけの仮定スコアをSSS+下限の `1_009_000` にする
3. 仮定単曲レートを本枠下限の単曲レートと比較する
4. 仮定単曲レートが本枠下限を厳密に上回る場合だけ候補とする。同値の場合は候補外とする

Ratingはスコアと譜面定数だけで決まるため、仮定時にランプを変更する必要はない。

現在すでにSSS+以上の譜面は、これ以上スコアを伸ばしても単曲レートが増えないため候補外とする。

本枠の件数が定員未満なら全対象レコードが本枠へ入るため、候補枠は空になる。

### 9.4 候補枠の件数と表示順

候補条件を満たしたレコードを、仮定値ではなく現在値の4キーで並べる。

- `best_candidate`: 先頭10件
- `new_candidate`: 先頭10件

選出順に1から `slot_order` を設定する。

### 9.5 完全置換

旧版プレイヤーの枠更新は差分更新ではなく、全スロットの完全置換として行う。

同一トランザクション内で次の順に実行する。

1. 対象プレイヤーの `player_records` 全行を `chart_id` 昇順で `FOR UPDATE` する
2. 削除曲、未来リリース曲などの計算対象外行を含む全行を `slot = none`、`slot_order = NULL` へ更新する
3. 選出した本枠・候補枠だけを一括割り当てする

選出されたchart IDだけをCASE更新し、対象外行に過去スロットを残してはいけない。

---

## 10. 登録APIのRating計算修正

現行の登録処理には、`best`、`best_candidate`、`new`、`new_candidate` をまとめて取得し、ローカルの単曲レートで本枠を再選出する挙動がある。

この挙動は確定仕様と一致しない。バッチだけを修正すると、登録直後と翌日のバッチ後で計算値が往復するため、登録処理も同時に修正する。

### 修正後

- `best` 本枠だけをベスト集計へ使う
- `new` 本枠だけを新曲集計へ使う
- 候補枠をRating集計へ含めない
- 公式 `slot_order` を正とし、本枠をローカル再選出しない
- バッチの現行版プレイヤー集計と同じDomain Serviceを使う

既存の `CalcRatingStats()` は旧挙動を前提としているため、そのままバッチへ流用しない。低水準の単曲Rating整数計算と平均計算は共有し、公式本枠集計と旧版枠再構築を別の明確なドメイン操作として追加する。

関連する既存テストを削除・変更して期待値を合わせるのではなく、既存テストが誤仕様を固定している場合は、仕様変更の根拠を確認したうえで必要最小限の期待値更新を行う。ただし、重複ケースや不要ケースを増やしてはいけない。

---

## 11. OVER POWER再計算

全プレイヤーについて、保存済み通常譜面レコードと開始時マスタスナップショットからOVER POWER値を再計算する。

既存の次の仕様を維持する。

- 削除済み楽曲を除外
- WORLD'S ENDを除外
- プレイヤーが未解禁設定した楽曲・ULTIMA譜面を除外
- 同一楽曲の複数難易度では単曲OPが最大の譜面だけを採用
- `charts.is_const_unknown = true` でも暫定定数で計算
- `players.overpower_value` だけを保存
- `overpower_percent` は保存しない

Ratingと異なり、OPでは `songs.released_at` が未来日の楽曲を特別に除外しない。これは既存登録処理および動的分母の対象集合と一致させるためである。通常運用では稼働前楽曲のスコアは存在しない。

バッチだけに未来日除外を導入して、登録直後のOP値やレスポンス時の動的分母と異なる規則を作ってはいけない。

既存のlocked-awareな変換・集計処理とDomain Serviceを共有し、同じ計算式を複製しない。

---

## 12. トランザクションと競合制御

### 12.1 単位

1プレイヤーにつき1トランザクションとする。全プレイヤーを1つのトランザクションへ入れてはいけない。

### 12.2 API登録との競合

ページ列挙時にプレイヤーIDと `data_collected_at` を取得し、スナップショットとして保持する。

プレイヤートランザクション内では次の順序を厳守する。

1. `players` 行を `SELECT ... FOR UPDATE`
2. 行が存在しなければ削除済みとしてスキップ
3. 現在の `data_collected_at` とページ列挙時スナップショットをNULL-safe比較
4. 不一致なら競合スキップとしてロールバック
5. `player_records` 全行を `chart_id` 昇順で `FOR UPDATE`
6. プレイヤー未解禁設定を取得
7. Rating枠処理
8. Rating・OVER POWER計算
9. `players` 集計値を条件付き更新
10. commit

最後の更新条件にも次を含める。

```sql
WHERE id = ?
  AND data_collected_at <=> ?
```

`RowsAffected() == 0` の場合は、スロット更新を含むトランザクション全体をロールバックし、競合スキップとする。

登録処理は既に `players` 更新後に `player_records` を更新する。バッチも `players`、`player_records` の順にロックすることで、ロック順を統一する。

### 12.3 競合時の結果

- 登録処理が先行: バッチはplayerロック待機後、`data_collected_at` 不一致を検出してスキップ
- バッチが先行: 登録処理はバッチcommit後に進み、より新しい公式データと計算値を保存

どちらの場合も古いバッチ計算値が最新登録結果を上書きしない。

---

## 13. 全体多重起動防止

MySQLのアドバイザリロックを使う。

### 13.1 接続

通常のプレイヤートランザクションとは別に、バッチrun専用の `*sql.Conn` をコネクションプールから取得する。

同じ専用接続で次を行う。

1. `GET_LOCK(lockName, 0)`
2. バッチrun中は接続を保持
3. `defer` で `RELEASE_LOCK(lockName)`
4. 接続をclose

`GET_LOCK` と `RELEASE_LOCK` を異なるプール接続で実行してはいけない。

### 13.2 結果

- 戻り値1: ロック取得成功、処理開始
- 戻り値0: 既存runが稼働中。正常スキップとして終了コード0
- NULLまたはSQLエラー: インフラ障害として終了コード1

接続切断時はMySQLがロックを解放するが、通常終了時は必ず明示的に解放する。

ロック名は `internal/info/info.go` に定数として定義する。

---

## 14. プレイヤー列挙

### 14.1 対象上限

run開始時に `MAX(players.id)` を取得し、`upperBoundPlayerID` として固定する。

### 14.2 キーセットページング

次の形式で列挙する。

```sql
SELECT id, data_collected_at
FROM players
WHERE id > ?
  AND id <= ?
ORDER BY id
LIMIT ?
```

`OFFSET` ページングは禁止する。

### 14.3 実行中の追加・削除

- run開始後に追加されたプレイヤーは次回runで処理する
- 列挙後、処理前に削除されたプレイヤーは `SELECT ... FOR UPDATE` が0行となるため、`deleted-skipped` として継続する
- ID欠番は問題にしない

初期実装では並列数を1に固定する。計測なしに並列化してはいけない。

---

## 15. Clean Architecture上の配置

### 15.1 Domain Layer

`internal/domain/service` に、少なくとも次の責務を置く。

- 公式本枠からのRating集計
- 旧版プレイヤーの本枠選出
- 候補入り可能性判定
- 候補枠選出
- Rating比較キーの生成

Domain ServiceはDB、SQL、cron、loggerへ依存してはいけない。

Domain層へ渡す入力は、必要な値だけを持つ純粋な構造体とする。

### 15.2 Repository Ports

`internal/domain/repository` に、次のような目的別portを定義する。

- プレイヤーバッチ対象のキーセット取得
- プレイヤー行のロック付き取得
- バッチ用プレイヤーレコード投影の取得
- 全スロットのnone化
- 選出スロットの一括割り当て
- Rating三値とOVER POWER値の競合条件付き更新
- バッチ用マスタスナップショット取得
- アドバイザリロック

SQLや `*sql.Tx`、`*sql.Conn` をDomainまたはUsecaseインターフェースへ露出させてはいけない。既存の `repository.Executor` 方針には従う。

### 15.3 Usecase Layer

`PlayerStatsRecalculationBatchUsecase` は次を担当する。

- runスナップショット作成
- マスタ事前検証
- プレイヤーページング
- 現行版・旧版分類
- プレイヤートランザクションのオーケストレーション
- エラー分類
- 結果件数集計

ソート規則や候補判定をUsecaseへ実装してはいけない。

### 15.4 Infra Layer

Infraは次を担当する。

- 明示列SQL
- MySQL行ロック
- キーセットページング
- スロット一括更新
- `GET_LOCK`用接続ライフサイクル
- Repository portの実装

バッチ用クエリでは、既存の8テーブルJOINを無条件に流用せず、必要なプレイヤー可変データだけを取得する専用projectionを使う。

---

## 16. SQL実装上の指示

- `SELECT *` 禁止
- プレイヤー全件の一括メモリロード禁止
- ループ内で楽曲・譜面マスタを毎回取得するN+1禁止
- player単位の未解禁設定取得はplayer固有データなので許容
- 全スロットのnone化と再割り当ては同一トランザクション
- 選出スロットの更新はループで1件ずつUPDATEせず、一括UPDATEまたはバルクUPSERTを使う
- 行ロック対象は先行する `SELECT chart_id ... ORDER BY chart_id FOR UPDATE` で決定する
- 既存マイグレーションを変更しない
- 新しい永続カラムが不要ならマイグレーションを追加しない

`migration/schema_mysql.sql` は現行マイグレーション適用後の状態と同期する。現在のsnapshotに `data_collected_at` が反映されていない場合は、既存の `dump-schema` 手順で更新する。これは過去マイグレーションの変更ではない。

---

## 17. バイナリとディレクトリ

汎用プレースホルダーの次を削除する。

```text
cmd/batch/
```

専用エントリーポイントを追加する。

```text
cmd/recalculate-player-stats/main.go
```

今後別のバッチを追加するときも、汎用バッチへサブコマンドを詰め込まず、責務ごとに `cmd/<batch-name>` とバイナリを分ける。

配布成果物名には既存APIバイナリと同様にOSとアーキテクチャを含める。

```text
chunisupport-recalculate-player-stats-linux-amd64
chunisupport-recalculate-player-stats-linux-arm64
```

ローカルTaskfileで先頭にアンダースコアを付ける既存規約を維持する場合も、少なくとも次の形式とする。

```text
_chunisupport-recalculate-player-stats-linux-amd64
_chunisupport-recalculate-player-stats-linux-arm64
```

OS・アーキテクチャを省略した単一名をビルド成果物として使ってはいけない。

---

## 18. CI・Taskfile・ドキュメント

### 18.1 GitHub Actions

`.github/workflows/build.yml` を更新し、APIと同じく次をビルドする。

- `GOOS=linux GOARCH=amd64`
- `GOOS=linux GOARCH=arm64`

API成果物とバッチ成果物が名前衝突しないようにする。

必要な `BuildDate`、`Revision` のldflagsはAPIと同じにする。

### 18.2 Taskfile

- `run-batch` の `./cmd/batch` 参照を削除
- 責務が分かるタスク名へ変更
- build対象を新しい専用バイナリへ変更
- OS・アーキテクチャ付きファイル名を維持

例:

```text
run-recalculate-player-stats
```

### 18.3 README・仕様書

少なくとも次を更新する。

- `README.md`
- `docs/rating_calculation.md`
- `docs/overpower_calculation.md`
- `docs/API.md`

特に、次の古い説明を残してはいけない。

- 新曲がベスト枠にも二重に入るという説明
- 候補枠をRating集計に含める説明
- `is_new` がゲームバージョン新曲枠を表すという説明
- 全保存譜面から登録時に常に本枠をローカル再選出する説明

---

## 19. コマンド終了とログ

### 19.1 シグナル

`signal.NotifyContext` でSIGINT・SIGTERMを受ける。

シグナル受信時:

- 新しいプレイヤー処理を開始しない
- 実行中トランザクションはcontext cancelによりロールバック
- サマリーログを出す
- 外部からの正常停止として終了コード0

### 19.2 終了コード

| 状態 | 終了コード |
|---|---:|
| 全対象成功 | 0 |
| 他runがロック保持中 | 0 |
| 競合スキップだけ発生 | 0 |
| 削除済みスキップだけ発生 | 0 |
| SIGINT/SIGTERMによる正常停止 | 0 |
| 1件以上のプレイヤー失敗 | 1 |
| マスタ事前検証失敗 | 1 |
| アドバイザリロックSQLエラー | 1 |
| DB接続・初期化失敗 | 1 |

### 19.3 構造化ログ

最低限、次のキーを出す。

- `started_at`
- `operational_date`
- `current_version`
- `upper_bound_player_id`
- `processed`
- `success`
- `current_preserved`
- `legacy_rebuilt`
- `conflict_skipped`
- `deleted_skipped`
- `failed`
- `last_player_id`
- `duration`

プレイヤー失敗ログには次を含める。

- `player_id`
- `rebuild_reason`
- `error`

`last_played_at IS NULL` による再構築は `rebuild_reason=legacy_null_last_played` とする。

---

## 20. cron運用

スケジュール機能をアプリケーション内へ組み込まず、cronまたはsystemd timerから専用バイナリを起動する。

JSTのcron例:

```cron
30 3 * * * /opt/chunisupport/bin/chunisupport-recalculate-player-stats-linux-amd64
```

サーバーのcronタイムゾーンがUTCの場合はJSTへ換算するか、crontabでタイムゾーンを明示する。

07:00前後を避ける。実行が07:00を跨いでも、run開始時の運用日が固定されるため同一run内の結果は一貫する。

---

## 21. TDD実装順序

次の順序でRed、Green、Refactorを進める。

### 21.1 Domain単体テスト

1. 単曲Rating比較キー
2. 4キーの決定的ソート
3. ベスト本枠30件
4. 新曲本枠20件
5. 枠数不足時の全件本枠入り
6. candidate判定
7. candidate各10件制限
8. 公式本枠だけのRating集計

### 21.2 バージョン・分類テスト

- リリース日06:59 JST
- リリース日07:00 JST
- `last_played_at` 境界直前・一致・直後
- `last_played_at IS NULL`
- `released_at IS NULL`
- 現行バージョン開始日前
- 現行バージョン開始日
- 運用日当日
- 運用日より未来
- `is_new` の値が分類に影響しない

### 21.3 candidateテスト

- 1,008,999点は候補判定対象
- 1,009,000点は候補外
- SSS+仮定で本枠へ入る
- SSS+仮定でも本枠下限を下回る
- 仮定Rating同率で譜面定数により入る
- Rating・定数同率で数値公式IDにより入る
- 候補表示順は仮定値ではなく現在値
- 本枠不足時は候補なし

### 21.4 Repository統合テスト

- キーセットページング
- upper boundより後のプレイヤーを除外
- `players FOR UPDATE`
- `player_records` 全行ロック
- 全行none化
- 選出枠の一括割り当て
- 削除曲・未来曲にも旧slotが残らない
- 条件付き集計UPDATE
- `RowsAffected() == 0`
- トランザクションロールバックでスロットも戻る
- `official_idx` 変換失敗・数値重複

### 21.5 Usecaseテスト

- 現行版プレイヤーはslot不変
- 現行版のcandidateをRating集計へ含めない
- 現行版公式本枠のNULL順位
- 現行版公式本枠の重複順位
- 現行版公式本枠の範囲外順位
- 旧版プレイヤーの完全再構築
- `last_played_at IS NULL` の再構築
- locked songをOPから除外
- `is_const_unknown` を暫定定数で計算
- 1プレイヤー失敗後も次へ進む
- 競合スキップ
- 削除済みプレイヤースキップ
- context cancel
- 同一run条件での冪等性

### 21.6 コマンド・インフラテスト

- アドバイザリロック取得
- 二重起動時の正常スキップ
- NULL・SQLエラー時の失敗
- 同じ接続でのロック解放
- SIGINT・SIGTERM
- 終了コード
- サマリーログ

新規テストは `testify/assert` を基本とし、後続アクセスの前提確認には `testify/require` を使う。

---

## 22. 受入条件

次をすべて満たしたとき実装完了とする。

- 譜面定数変更後、登録操作なしで翌runにRatingとOP値が更新される
- 現行版プレイヤーの公式本枠・候補枠が変更されない
- 現行版プレイヤーのRatingに候補枠が混入しない
- 登録直後とバッチ後でRating集計規則が一致する
- バージョン切り替え後、旧新曲がベスト枠対象へ移動する
- 現行版の新曲が新曲枠へ入る
- `released_at IS NULL` が新曲枠へ入る
- `is_new` の値がRating枠を変えない
- 本枠がベスト30件、新曲20件を超えない
- 候補枠が各10件を超えない
- SSS+済み譜面が候補へ入らない
- SSS+でも本枠入り不能な譜面が候補へ入らない
- スロット再構築後に古いslotが残らない
- API登録との競合で古い計算値を上書きしない
- バッチ多重起動が発生しない
- 途中停止後に再実行できる
- 同じ入力で再実行しても結果が変わらない
- 全SQLで必要カラムを明示している
- 過去マイグレーションを変更していない
- `go test ./...` が成功する
- `gofmt -s -w` 適用後もテストが成功する
- コメント・文字列・文書に文字化けがない

---

## 23. 実装時の変更候補

実際の型名は既存命名に合わせるが、変更範囲は概ね次を想定する。

```text
cmd/
  batch/                                  削除
  recalculate-player-stats/               追加

internal/domain/service/
  rating_service.go                       低水準計算共有
  rating_slot_service.go                  追加候補
  rating_slot_service_test.go             追加候補

internal/domain/repository/
  player_repository.go                    port拡張
  player_record_repository.go             port拡張
  player_stats_batch_repository.go        追加候補
  batch_lock.go                           追加候補

internal/usecase/
  player_stats_recalculation_batch.go      追加候補
  player_stats_recalculation_batch_test.go 追加候補
  player_data_usecase_impl.go              登録時Rating集計修正

internal/infra/repository/
  player_stats_batch_repository_impl.go    追加候補
  player_stats_batch_repository_impl_test.go

internal/infra/db/
  advisory_lock.go                         追加候補
  advisory_lock_test.go

.github/workflows/build.yml                更新
Taskfile.yml                               更新
README.md                                  更新
docs/rating_calculation.md                 更新
docs/overpower_calculation.md              更新
docs/API.md                                更新
migration/schema_mysql.sql                 現行状態へ同期
```

小規模アプリケーションであるため、Repositoryを不必要に細分化しない。既存Repositoryへ自然に追加できる責務は既存インターフェースへ追加し、バッチ固有の複数集約投影だけを専用Query ServiceまたはRepositoryとして分離する。

---

## 24. Grokレビュー結果と対応判断

本設計は `grok -p` による外部レビューを3回実施し、妥当な指摘がなくなるまで修正した。

### 24.1 第1回で採用した主な指摘

- 登録APIが候補枠を含めて再選出しており、バッチ仕様と不一致
  - 登録APIも公式本枠だけを集計する共通Domain Serviceへ統一する
- 既存 `CalcRatingStats()` だけでは新仕様を実装できない
  - 公式本枠集計と旧版枠再構築のDomain Serviceを追加する
- `data_collected_at` 条件付き更新が必要
  - Rating三値とOP値を同一の条件付きUPDATEへまとめる
- プレイヤー列挙、アドバイザリロック、CIが未整備
  - 専用port・adapter・workflowを追加する
- 新曲とベストのプール関係が古い文書で矛盾
  - 本仕様ではバージョン基準の排他的プールと明記し、関連文書を修正する
- candidateの仮定スコアが不明確
  - SSS+下限の1,009,000点と明記する
- `official_idx` の数値比較規則が不明確
  - 10進Parseと数値重複禁止を事前検証へ追加する

### 24.2 第2回で採用した主な指摘

- 選出行だけのCASE更新では古いslotが残る
  - 全行none化後に選出枠を割り当てる完全置換へ変更する
- `GET_LOCK` は同じ接続を保持する必要がある
  - 専用 `*sql.Conn` のライフサイクルを明記する
- 長時間run中に07:00を跨ぐと基準が変わり得る
  - 開始時刻、運用日、現行バージョンをrun単位で固定する
- 現行版公式枠の順位異常時の動作が不明
  - 補正せず当該プレイヤーを失敗させる
- マスタ更新によって処理順依存が生じ得る
  - 楽曲・譜面マスタをrun開始時スナップショットとして固定する
- keyset処理中の追加で対象が伸びる
  - 開始時の最大プレイヤーIDを上限として固定する

### 24.3 採用しなかった指摘

#### OPから未来リリース曲を除外する提案

採用しない。

既存のOP登録処理とレスポンス時の動的分母は、削除済み、WORLD'S END、未解禁設定を除外するが、未来リリース日を特別に除外していない。バッチだけで未来日を除外すると、登録直後の保存値や動的分母と計算対象が一致しなくなる。

通常は稼働前楽曲のプレイヤースコアが存在しないため、既存OP仕様を維持する。Rating枠の未来日除外とは別規則として文書化する。

#### `last_played_at IS NULL` を現行版扱いへ補正する提案

採用しない。

これはレビュー前に利用者と確認済みで、NULLは旧版扱いとする確定仕様である。代わりに、再構築理由を構造化ログへ明示する。

#### 登録トランザクション中の未commit値が見えるという説明

説明自体は採用しない。通常のトランザクション分離では未commit更新は他接続から見えない。

ただし、登録処理とバッチ処理の競合リスクは妥当であるため、`players FOR UPDATE`、`data_collected_at` スナップショット比較、終端条件付きUPDATEの三重対策を採用した。

### 24.4 最終レビュー

第3回レビューでは、正確性、race/deadlock、決定性、既存仕様との整合、実装可能性の観点で「妥当な指摘なし」と判定された。

---

## 25. 実装後セルフレビュー

提出前に必ず次を確認する。

- [ ] `go test ./...` が成功した
- [ ] `gofmt -s -w` をプロジェクトコードへ適用した
- [ ] 整形後に再度 `go test ./...` が成功した
- [ ] 候補枠をRating集計へ含めていない
- [ ] `songs.is_new` をRating枠判定へ使っていない
- [ ] 新曲とベストのプールが排他的である
- [ ] 全スロットnone化が削除曲・未来曲を含む
- [ ] playerとrecordのロック順が統一されている
- [ ] API登録との競合時にslot更新もロールバックされる
- [ ] `GET_LOCK` と `RELEASE_LOCK` が同じ接続である
- [ ] run途中で時刻・バージョン・マスタ基準が変わらない
- [ ] `SELECT *` がない
- [ ] プレイヤーループ内のマスタN+1がない
- [ ] 過去マイグレーションを変更していない
- [ ] README、API、Rating、OP文書を更新した
- [ ] OS・アーキテクチャ付きの両バイナリをビルドできる
- [ ] 日本語コメント・文字列・文書に文字化けがない
