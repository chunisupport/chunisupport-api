# 仕様判断が必要なテストの調査結果

再検証日: 2026-10-28
調査対象: `origin/develop` (`d6748035bebab9b499440168c256a19d0a9bc441`)

前回の基準日以降の比較元: `99bb540defd52f9b5f7ceaf6884a4f5638b1d920`（2026-10-03）。比較期間中に追加・変更・削除されたテスト関連ファイル82件を差分で確認し、既存2項目と追加機能の期待値を実装・正式ドキュメントに照合した。全テストの全経路を網羅した証明ではない。

今回の結果は、仕様判断の継続2件、追加テストの検証不足2件。追加機能で明確に誤った期待値と断定できる新規項目は今回の範囲では確認していない。

## 目的

テストが現在の実装を正しく固定していても、その期待値自体がプロダクト仕様として妥当とは限らない。
本書では、現在の実装・テスト・仕様書を再検証し、現時点で仕様判断が必要な項目と、仕様・実装・検証結果が一致している項目を記録する。

## 判断が必要な項目

### 1. 【継続・中】同一レーティング・同一譜面定数で低いスコアを優先するか

#### 現状

`BuildRatingSlots`は次の順番でレコードを並べる。

1. 単曲レーティング降順
2. 譜面定数降順
3. スコア昇順
4. 公式ID昇順

そのため、単曲レーティングと譜面定数が同じ場合は、より低いスコアのレコードが先になる。

#### 疑義

本枠の境界で同一レーティングの譜面が並んだ場合、より高いスコアの譜面が本枠外になる。
集計値は同じでも、APIに現れる本枠譜面と候補枠判定が変わる。

#### 選択肢

1. スコア降順にし、より高い達成結果を優先する。
2. スコアを比較せず、公式IDなどの安定した識別子だけで決定する。
3. 現在のスコア昇順を維持し、改善余地のある譜面を優先する仕様として明記する。

#### 関連箇所

- `internal/domain/service/rating_slot_service.go:50`
- `internal/domain/service/rating_slot_service_test.go:118`

### 2. 【継続・中】認証済みユーザーを公開参照APIのレート制限対象外にするか

#### 現状

- `AnonymousIPRateLimitMiddleware`は、認証済みユーザーを検出するとIPレート制限を適用せず後続処理へ進む。
- テストは認証済みユーザーが未認証向けIP上限の対象外になることを固定している。
- API仕様も公開参照系のIP制限を未認証時の制限として扱っている。

#### 疑義

アプリケーション内では、アカウントを作成すれば公開参照APIのIP制限を回避できる。
エッジ側に別の制限がなければ、スクレイピングや高負荷リクエストへの保護が弱くなる。

#### 選択肢

1. 未認証時はIP単位、認証済み時はユーザーID単位で制限する。
2. 認証状態に関係なくIP単位で制限する。
3. 現在の認証済みユーザー除外を維持し、エッジ側のレート制限を必須要件として明文化する。

#### 関連箇所

- `internal/app/middleware/rate_limit_middleware.go:218`
- `internal/app/middleware/rate_limit_middleware_test.go:573`
- `internal/app/router.go:392`
- `docs/API.md`

## 追加テストで残る検証不足

以下は期待値が誤っているという指摘ではなく、テスト成功から保証できる範囲の限界である。

### 3. 【新規・中】コースの実MySQL並行更新テストが標準CIではスキップされる

- `internal/infra/repository/course_concurrency_mysql_test.go:26` は `COURSE_CONCURRENCY_MYSQL_DSN` がないとスキップする。
- `.github/workflows/ci.yml:18` および `build.yml` は `go test ./...` を呼ぶが、リポジトリ内のworkflowにはこのDSNの設定もMySQLサービスの用意もない。外部のCI変数が別途設定されているかは未確認。
- 実MySQLテスト自体は `performance_schema.data_lock_waits` で後続処理の待機まで確認する。一方、`internal/usecase/course_usecase_concurrency_test.go:84` の通常テストはメモリ上のロック付き代替リポジトリを使う。こちらの成功だけではInnoDBの実際のロック待機・コミット後の読み取りは保証できない。
- 例: `FOR UPDATE` の実DB上の排他が失われても、代替リポジトリ側のロックが働けば通常テストが通る可能性がある。
- 対応判断: MySQLを用意する統合テストjobを追加するか、別の実行場所・頻度と直近の実行結果を記録する。

### 4. 【新規・中】SP称号の並行登録リカバリーはSQLを模倣したテストに留まる

- `internal/infra/repository/honor_repository_impl_test.go:247` は、事前検索で未検出→INSERTの一意制約エラー→`FOR UPDATE`で既存ID取得、という応答を `honorEnsureExec` に設定している。
- `:267` 以降でSQL文字列・引数を検証しているが、別トランザクションのコミットを待つ挙動、REPEATABLE READのスナップショットとカレントリードの違いは実行していない。
- `internal/infra/repository/honor_repository_impl.go:172` 付近の実装意図は「先行コミット済みの行をREPEATABLE READでも取得する」こと。代替ExecutorにIDを返させるテスト成功だけでは、その意図がMySQLで成立する証拠にならない。
- 対応判断: 同一ファイル名・URL違いの2接続による並行登録をMySQLで実行し、両方が同じIDで成功し、保存済みURLが変わらないことを検証する。現実装の誤りと断定はしない。

## 今回追加で確認し、疑義として採用しなかった項目

- 名前順フォルダの濁音・小文字などが `NUMBER` になるケース: `ResolveNameFolderCode` は正規化済み読みを前提とし、正式API仕様（`docs/API.md:4825`）も選択文字列をNFKC正規化して先頭文字で判定する。テストの非正規化入力だけから通常データの誤分類とは断定しない。
- Firebaseアカウント変更後の旧ユーザー削除失敗でも成功するケース: DB差し替え後の外部処理失敗として正式API仕様と実装に明記済み。コミット失敗時に旧アカウントを削除しない経路の専用テストはないが、コードはトランザクション失敗時に即returnしている。
- 互換APIで未認証を503へ変換するケース、`rating_max`へ現在値を返すケース: `docs/API.md` の互換契約に明記済み。
- 固定件数目標を対象数減少時に自動補正しないケース: `docs/API.md:2505` に維持する仕様が明記済み。
- 権限変更時のレート制限: 今回追加のテストは枠の開始時刻・消費回数を維持して上限のみ変える挙動を検証する。前述の未認証IP制限の免除問題とは別であり、この追加テストを疑義とはしない。

## 今回の検証と限界

- 実装・テスト差分・正式API仕様・CI設定を静的に確認した。変更はこの資料のみで、本番コードとテストの期待値は変更していない。
- `go test ./...` は本環境の `/usr/bin/go` がGo言語のCLIとして動かず、`Go: Unknown option: test` となり実行できなかった。今回のテスト成功を主張しない。
- `go vet`、実MySQL並行テスト、実DB照会、マイグレーション適用は今回未実行。
- 下記の譜面定数に関する実行結果・データ件数は2026-10-03の過去記録であり、今回のdevelopで再実行した結果ではない。

## 前回確認済みの項目（実行結果は2026-10-03）

### 譜面定数は必ず`1.0`以上とする

#### 現在の仕様と実装

- 実在する通常譜面の定数は`1.0`～`16.0`の0.1刻みとし、`0.0`～`0.9`、NaN、無限大を受け付けない。
- `chartconstant.NewChartConstant`は`constants.ChartConstMin = 1.0`を下限に使う。`ChartConstant.Scan(nil)`はエラーを返し、DB保存時の`Value()`も値を検証する。
- WORLD'S ENDの管理者向け譜面ランキング・フレンド譜面ランキング・フレンドスコア比較は、SQLの`NULL AS chart_const`を`*ChartConstant`の`nil`として読み込む。APIでは従来どおり`const`を省略する。
- ランキングのレーティング・OVER POWER計算は定数がある場合だけ行う。プレイヤーレコードの関連譜面がない場合も計算を行わず、APIの`const`・レーティング・OVER POWER・達成率は`0`を返す。
- `CalcSongMaxOP`・`CalcSingleOverpowerPercent`の数値入力に対する`0`以下の判定は維持する。値オブジェクトの生成やWORLD'S ENDの読取には番兵値を使用しない。
- MySQLのスキーマ定義と楽曲バッチ用SQLiteの`charts.const`は`1.0`～`16.0`を制約とする。既存MySQL DB用の制約追加は`000056_restrict_chart_constant_range`で行う。

#### 検証結果

- `0.0`～`0.9`を拒否する生成・DB読取・DB保存テスト、NULL・JSONの不正入力を拒否するテスト、下限`1.0`の正常系テストが成功している。
- WORLD'S ENDのSQL読取で定数が`nil`になること、APIで`const`が省略されること、ランキングのOVER POWERが`0`になることを確認している。
- 関連譜面がないプレイヤーレコードでは、理論値スコアとALL JUSTICEでもレーティング・OVER POWER・達成率を計算しないことを確認している。
- 楽曲バッチ用SQLiteで範囲外の定数を拒否するテストが成功している。
- `go test ./...`、`go vet ./...`、`git diff --check`が成功している。
- 2026-10-03にローカルDB（`localhost:3306 / chunisupport`）へ直接照会し、全6,868譜面のうち範囲外・NULLは0件だった。MySQLのマイグレーション適用は未検証。

#### 既存データの確認SQL

```sql
SELECT id, song_id, difficulty_id, const, is_const_unknown
FROM charts
WHERE const IS NULL
   OR const < 1.0
   OR const > 16.0
ORDER BY id;
```

範囲外の値がある場合は、正しい譜面定数を確認して修正してから制約を追加する。

#### 関連箇所

- `internal/domain/constants/chart.go`
- `internal/domain/vo/chartconstant/chartconstant.go`
- `internal/domain/vo/chartconstant/chartconstant_test.go`
- `internal/domain/service/rating_service.go`
- `internal/dto/player_record_dto.go`
- `internal/dto/player_record_dto_test.go`
- `internal/infra/repository/admin_chart_ranking_query_service_impl.go`
- `internal/infra/repository/friend_chart_ranking_query_service_impl.go`
- `internal/infra/repository/friend_score_comparison_query_service_impl.go`
- `internal/infra/songbatch/songchart/schema.sql`
- `internal/infra/songbatch/songchart/constant_range_test.go`
- `migration/mysql/000056_restrict_chart_constant_range.up.sql`
