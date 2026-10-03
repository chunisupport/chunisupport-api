# 仕様判断が必要なテストの調査結果

再検証日: 2026-10-03
調査対象: `develop`

## 目的

テストが現在の実装を正しく固定していても、その期待値自体がプロダクト仕様として妥当とは限らない。
本書では、現在の実装・テスト・仕様書を再検証し、現時点で仕様判断が必要な項目と、仕様・実装・検証結果が一致している項目を記録する。

## 判断が必要な項目

### 1. 同一レーティング・同一譜面定数で低いスコアを優先するか

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

- `internal/domain/service/rating_slot_service.go`
- `internal/domain/service/rating_slot_service_test.go`

### 2. 認証済みユーザーを公開参照APIのレート制限対象外にするか

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

- `internal/app/middleware/rate_limit_middleware.go`
- `internal/app/middleware/rate_limit_middleware_test.go`
- `internal/app/router.go`
- `docs/API.md`

## 仕様・実装・検証結果が一致している項目

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
