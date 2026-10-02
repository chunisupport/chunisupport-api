# 仕様判断が必要なテストの調査結果

再検証日: 2026-10-03
調査対象: `develop`

## 目的

テストが現在の実装を正しく固定していても、その期待値自体がプロダクト仕様として妥当とは限らない。
本書では、現在の実装・テスト・仕様書を再検証し、現時点で仕様判断が必要な項目と、仕様は確定したが実装が追従していない項目を記録する。

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

## 仕様は確定したが実装が追従していない項目

### 譜面定数は必ず`1.0`以上とする

#### 確定した仕様

実在する譜面の譜面定数は必ず`1.0`以上である。`0`は実在する譜面定数ではない。

#### 実装との差分

- `chartconstant.NewChartConstant`の下限は`constants.ChartConstValueMin = 0.0`であり、`0.0`以上`1.0`未満を受け付ける。
- `ChartConstant.Scan(nil)`はDBの`NULL`を`0.0`へ変換する。
- `charts.const`のCHECK制約は`const >= 0`である。
- WORLD'S END譜面を含むUNIONクエリ（管理者向け譜面ランキング・フレンド譜面ランキング・フレンドスコア比較）は、譜面定数を持たない行を`0 AS chart_const`として値オブジェクトへ読み込んでいる。
- `CalcSongMaxOP`・`CalcSingleOverpowerPercent`は`0`以下を「譜面定数なし」の番兵値として扱っている。

#### 追従に必要な作業

1. 譜面定数を持たない行を`0`ではなくNULL許容の型で表し、値オブジェクトの下限を`1.0`へ引き上げる。
2. 既存データに`1.0`未満の譜面定数がないことを確認したうえで、CHECK制約を`const >= 1.0`へ変更するマイグレーションを追加する。

#### 関連箇所

- `internal/domain/constants/chart.go`
- `internal/domain/vo/chartconstant/chartconstant.go`
- `internal/domain/vo/chartconstant/chartconstant_test.go`
- `internal/domain/service/rating_service.go`
- `internal/infra/repository/admin_chart_ranking_query_service_impl.go`
- `internal/infra/repository/friend_chart_ranking_query_service_impl.go`
- `internal/infra/repository/friend_score_comparison_query_service_impl.go`
