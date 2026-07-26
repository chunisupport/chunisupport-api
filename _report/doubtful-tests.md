# 仕様判断が必要なテストの調査結果

## 目的

テストが現在の実装を正しく固定していても、その期待値自体がプロダクト仕様として妥当とは限らない。
本書では、実装者だけでは決定できない期待値と、仕様書・実装・テスト間で契約が分かれている項目を整理する。

以下の項目は別途修正済みのため、本書の判断対象には含めない。

- ADMIN向けレート制限を「無制限」と表現していたテスト
- `username.UserName` / `playername.PlayerName` のScannerがDBのNULL・空文字列を許容していた挙動
- `X-Forwarded-For` を検証しているように見えて、実際には同一の`RemoteAddr`で成功していたテスト

## 判断が必要な項目

### 1. `app_ver`を検証するか

#### 現状

- `TestValidatePlayerDataPayload_AppVersionを検証しない`は、空文字列と任意の文字列を正常としている。
- `validatePlayerDataPayload`は`AppVersion`を検証していない。
- `docs/API.md`には、`app_ver`が必須で対応バージョンは`0.1.0`、非対応バージョンは`400 Bad Request`になるという記述が残っている。
- コミット`4d71164`では、アプリバージョン非対応に関するエラーハンドリングが意図的に削除されている。

#### 疑義

現在の実装では、将来ペイロードの構造や意味が変わっても、サーバー側で非対応バージョンを拒否できない。
一方、バージョンを厳密に制限すると、古いインポートアプリからの登録を直ちに拒否することになる。

#### 選択肢

1. `app_ver`を互換性確認用の情報としてのみ保存し、未指定・任意値を許容する。
2. 対応バージョン一覧を定義し、一覧外と未指定を拒否する。
3. メジャー・スキーマバージョンだけを検証し、パッチバージョンの違いは許容する。

#### 関連箇所

- `internal/usecase/player_data_usecase_impl_test.go`
- `internal/usecase/player_data_usecase_impl.go`
- `docs/API.md`

### 2. Nginx配下で利用者IPをどこから取得するか

#### 現状

- ルーターは`echo.New()`のままで、信頼するプロキシとIP抽出方法を設定していない。
- サーバーは`127.0.0.1`で待ち受け、Nginxからの接続を想定している。
- IPレート制限、ログイン、サインアップ、一時プレイヤーデータ保存で`c.RealIP()`を使用している。
- 修正後のテストは、IP抽出設定がない場合に`RemoteAddr`が識別子になることを明示している。

#### 疑義

本番で`RemoteAddr`がNginxのループバックアドレスになる場合、全利用者が同じIPとして扱われる可能性がある。
逆に、外部から受け取った`X-Forwarded-For`を無条件に信頼すると、任意のIPを詐称してレート制限を回避できる。

#### 選択肢

1. 接続元が信頼済みNginxの場合だけ、Nginxが設定した転送ヘッダーから利用者IPを抽出する。
2. アプリでは`RemoteAddr`だけを使用し、IPレート制限をNginxなどのエッジ側へ移す。
3. アプリとエッジの両方で制限し、それぞれの責務と識別方法を明文化する。

#### 関連箇所

- `internal/app/router.go`
- `internal/app/server.go`
- `internal/app/middleware/rate_limit_middleware.go`
- `internal/app/middleware/rate_limit_middleware_test.go`
- `internal/app/handler/api_internal/login_handler.go`
- `internal/app/handler/api_internal/signup_handler.go`
- `internal/app/handler/api_internal/temporary_player_data_handler.go`

### 3. 譜面定数`0`を実在する値として扱うか

#### 現状

- `chartconstant.ChartConstant`は`0.0`を有効値として受け付ける。
- API仕様も譜面定数の範囲を`0.0`以上としている。
- `TestCalcSongMaxOP`は最大譜面定数が`0`の場合に最大OPを`0`としている。
- `TestCalcSingleOverpowerPercent`は譜面定数が`0`の場合に達成率を`0%`としている。
- OVER POWERの数式上、定数`0`の理論値は`(0 + 3) × 5 = 15`になる。

#### 疑義

同じ`0`が「実在する譜面定数」と「対象譜面が存在しないことを表す番兵値」の両方に使われている。
このままでは、値だけからどちらの状態か判別できない。

#### 選択肢

1. `0`を有効な譜面定数として計算し、譜面なしはポインタや存在フラグで表す。
2. `0`を譜面なし・未設定の番兵値とし、実在する通常譜面では`0`を禁止する。
3. 未確定譜面には必ず正の暫定定数を設定し、`0`の利用箇所を集約処理だけに限定する。

#### 関連箇所

- `internal/domain/vo/chartconstant/chartconstant.go`
- `internal/domain/vo/chartconstant/chartconstant_test.go`
- `internal/domain/service/rating_service.go`
- `internal/domain/service/rating_service_test.go`
- `docs/overpower_calculation.md`

### 4. 集計レーティングを小数点以下2桁と4桁のどちらで保持するか

#### 現状

- `CalcRatingStats`とそのテストは、小数点以下4桁で切り捨てる。
- `docs/API.md`も、計算レーティングと各平均を小数点以下4桁としている。
- `docs/rating_calculation.md`は、プレイヤーレーティングと各平均を小数点以下2桁で切り捨てる数式になっている。

#### 疑義

内部保存値、API値、公式画面に表示する値の精度が区別されていない。
同じ「プレイヤーレーティング」という名称で2種類の精度が存在すると、比較・ランキング・キャッシュ・クライアント表示で不一致が発生し得る。

#### 選択肢

1. 内部保存値とAPI値は4桁、公式表示互換値は2桁とし、各フィールドの責務を明記する。
2. 保存・API・表示をすべて2桁に統一する。
3. 内部計算は高精度のまま保持し、APIごとに明示的な丸め規則を定義する。

#### 関連箇所

- `internal/domain/service/rating_service.go`
- `internal/domain/service/rating_service_test.go`
- `docs/rating_calculation.md`
- `docs/API.md`

### 5. 同一レーティング・同一譜面定数で低いスコアを優先するか

#### 現状

`BuildRatingSlots`は次の順番でレコードを並べる。

1. 単曲レーティング降順
2. 譜面定数降順
3. スコア昇順
4. 公式ID昇順

`TestBuildRatingSlots_レートと定数が同じ時はスコア昇順で並ぶ`は、`1,009,000`点を`1,010,000`点より先にする。

#### 疑義

本枠の境界で同一レーティングの譜面が並んだ場合、より高いスコアの譜面が本枠外になる。
集計値は同じでも、APIに現れる本枠譜面と候補枠判定が変わる。

#### 選択肢

1. スコア降順にし、より高い達成結果を優先する。
2. スコアを比較せず、公式IDなどの安定した識別子で決定する。
3. 現在のスコア昇順を維持し、改善余地のある譜面を優先する仕様として明記する。

#### 関連箇所

- `internal/domain/service/rating_slot_service.go`
- `internal/domain/service/rating_slot_service_test.go`

### 6. DB保存失敗時にも一時データを消費するか

#### 現状

- `TemporaryPlayerDataUsecase.Commit`は、JSON解釈とプレイヤーデータ登録より先に一時データを消費する。
- DB保存など後続処理が失敗しても同じトークンでは再試行できない。
- テストと`docs/API.md`は、再アップロード必須としてこの挙動を明示している。
- コミット`c7f21c8`で意図的に現在の仕様へ変更されている。

#### 疑義

同一トークンの並行実行を防ぎやすい一方、一時的なDB障害やサーバー内部エラーでも利用者に再アップロードを要求する。

#### 選択肢

1. 現在のat-most-once方式を維持する。
2. 一時データを「未処理・処理中・完了」の状態で管理し、再試行可能な失敗では未処理へ戻す。
3. 一時データ保存先をDBへ移行した上で、登録処理とトークン消費を同一DBトランザクションに含め、登録成功時だけ消費する。

#### 関連箇所

- `internal/usecase/temporary_player_data_usecase_impl.go`
- `internal/usecase/temporary_player_data_usecase_impl_test.go`
- `docs/API.md`

### 7. `is_maxop_unknown`でEXPERT以下の未確定定数を無視するか

#### 現状

- MASTERまたはULTIMAに未確定定数が1件でもあれば`true`になる。
- BASIC、ADVANCED、EXPERTの未確定定数は判定対象外である。
- テストとドメイン仕様書は現在の判定を明示している。

#### 疑義

MASTER・ULTIMAが存在しない楽曲でEXPERTの定数が未確定でも、`is_maxop_unknown=false`になる。
フィールド説明の「MAX OPが暫定値である可能性」と、難易度固定の判定規則が一致しない場合がある。

#### 選択肢

1. 難易度に関係なく、未確定定数を持つ譜面があれば`true`にする。
2. 未確定譜面が現在の最大定数になり得る場合だけ`true`にする。
3. 現在のMASTER・ULTIMA限定判定を維持し、フィールド名・説明を限定的な意味に変更する。

#### 関連箇所

- `internal/domain/service/song_aggregation_service.go`
- `internal/domain/service/song_aggregation_service_test.go`
- `docs/domain_model_specification.md`
- `docs/API.md`

### 8. 認証済みユーザーを公開参照APIのレート制限対象外にするか

#### 現状

- `AnonymousIPRateLimitMiddleware`は、認証済みユーザーを検出するとレート制限を適用せずに後続処理へ進む。
- テストは、上限1回の設定でも認証済みユーザーの3回のリクエストがすべて成功することを要求している。
- `docs/API.md`も、公開参照系の制限を「未認証時のみ」としている。

#### 疑義

アプリケーション内では、無料アカウントを作成すれば公開参照APIのIP制限を完全に回避できる。
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
