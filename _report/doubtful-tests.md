# 仕様判断が必要なテストの調査結果

再検証日: 2026-09-09
調査対象: `develop` (`fa5ff0e`)

## 目的

テストが現在の実装を正しく固定していても、その期待値自体がプロダクト仕様として妥当とは限らない。
本書では、現在の実装・テスト・仕様書を再検証し、現時点でも仕様判断が必要な項目だけを本文に残す。解消済みの項目は末尾の「確認済み・解消済み」に記録する。

## 判断が必要な項目

### 1. 譜面定数`0`を実在する値として扱うか

#### 現状

- `chartconstant.NewChartConstant`は`0.0`を有効値として受け付ける。
- `ChartConstant.Scan(nil)`もDBの`NULL`を`0.0`へ変換する。
- `CalcSongMaxOP`は最大譜面定数が`0`以下なら最大OVER POWERを`0`として返す。
- `CalcSingleOverpowerPercent`も譜面定数が`0`以下なら達成率を`0%`として返す。
- OVER POWERの数式上、定数`0`を実在値として計算した場合の理論値は`(0 + 3) × 5 = 15`になる。

#### 疑義

同じ`0`が「実在する譜面定数」と「未設定・対象譜面なしを表す値」の両方に使われている。
値だけでは状態を判別できず、計算関数も`0`を実在値ではなく番兵値として扱っている。

#### 選択肢

1. `0`を有効な譜面定数として計算し、未設定・譜面なしはポインタや存在フラグで表す。
2. `0`を未設定・譜面なしの番兵値とし、実在する通常譜面では`0`を禁止する。
3. 未確定譜面には必ず正の暫定定数を設定し、`0`を集約処理など限定された用途だけに使用する。

#### 関連箇所

- `internal/domain/vo/chartconstant/chartconstant.go`
- `internal/domain/vo/chartconstant/chartconstant_test.go`
- `internal/domain/service/rating_service.go`
- `internal/domain/service/rating_service_test.go`
- `docs/overpower_calculation.md`

### 2. 同一レーティング・同一譜面定数で低いスコアを優先するか

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

### 3. DB保存失敗時にも一時データを消費するか

#### 現状

- `TemporaryPlayerDataUsecase.Commit`は、プレイヤーデータ登録より先に一時データを消費する。
- DB保存など後続処理が失敗しても同じトークンでは再試行できない。
- `TestTemporaryPlayerDataUsecase_Commit_DB失敗時は再試行不可になる`が、この挙動を明示的に固定している。
- API仕様も再アップロードが必要な方式として扱っている。

#### 疑義

同一トークンの並行実行を防げる一方、一時的なDB障害やサーバー内部エラーでも利用者に再アップロードを要求する。

#### 選択肢

1. 現在のat-most-once方式を維持する。
2. 一時データを「未処理・処理中・完了」の状態で管理し、再試行可能な失敗では未処理へ戻す。
3. 一時データ保存先をDBへ移行し、登録処理とトークン消費を同一DBトランザクションに含め、登録成功時だけ消費する。

#### 関連箇所

- `internal/usecase/temporary_player_data_usecase_impl.go`
- `internal/usecase/temporary_player_data_usecase_impl_test.go`
- `docs/API.md`

### 4. `is_maxop_unknown`でEXPERT以下の未確定定数を無視するか

#### 現状

- MASTERまたはULTIMAに未確定定数が1件でもあれば`is_maxop_unknown=true`になる。
- BASIC、ADVANCED、EXPERTの未確定定数は判定対象外である。
- ドメイン実装、テスト、仕様書はいずれも現在の判定規則を明示している。

#### 疑義

MASTER・ULTIMAが存在しない楽曲でEXPERTの定数が未確定でも、`is_maxop_unknown=false`になる。
フィールド名が示す「最大OVER POWERが未確定か」という意味と、難易度固定の判定規則が一致しないケースがある。

#### 選択肢

1. 難易度に関係なく、未確定定数を持つ譜面があれば`true`にする。
2. 未確定譜面が現在の最大定数になり得る場合だけ`true`にする。
3. 現在のMASTER・ULTIMA限定判定を維持し、フィールド名・説明を限定的な意味に変更する。

#### 関連箇所

- `internal/domain/entity/song.go`
- `internal/domain/service/song_aggregation_service.go`
- `internal/domain/service/song_aggregation_service_test.go`
- `docs/domain_model_specification.md`
- `docs/API.md`

### 5. 認証済みユーザーを公開参照APIのレート制限対象外にするか

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

## 確認済み・解消済み

### `app_ver`の扱い

- `app_ver`は互換性確認用の情報として保存するだけで、未指定・任意文字列を許容する仕様へ統一済み。
- `docs/API.md`も現在の実装・テストと一致している。

### Nginx配下での利用者IP取得

- `TrustedProxyCIDRs`未設定時は接続元IPを直接使用する。
- 信頼済みプロキシを設定した場合だけ、信頼範囲を指定して`X-Forwarded-For`から利用者IPを抽出する。
- 専用テストで直接接続、信頼済みプロキシ、無効なCIDRを検証している。

### 集計レーティングの精度

- 単曲レーティングは小数点以下2桁で切り捨て、その値を合算する。
- プレイヤーレーティング、ベスト枠平均、新曲枠平均は小数点以下4桁で切り捨てる。
- `CalcRatingStats`、テスト、`docs/rating_calculation.md`の記述は現在この仕様で一致している。

### その他の修正済み項目

- ADMIN向けレート制限を「無制限」と表現していたテスト。
- `username.UserName` / `playername.PlayerName` のScannerがDBのNULL・空文字列を許容していた挙動。
- `X-Forwarded-For`を検証しているように見えて、実際には同一の`RemoteAddr`で成功していたテスト。
