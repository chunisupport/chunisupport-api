# リファクタリング指摘書 (2026-06-14時点)

本ドキュメントは、現在のコードベースを再確認したうえで、**まだ残っている改善点のみ**を整理したものです。
解消済み、または根拠が現状と一致しなくなった項目は削除しました。

## 優先度定義
- **Critical (緊急)**: セキュリティ上の重大な欠陥、または主要機能の停止に直結する問題。即時対応が必要。
- **High (高)**: アーキテクチャの根幹に関わる、またはセキュリティ・安定性に重大なリスクがある項目。優先して対応が必要。
- **Medium (中)**: 保守性や拡張性を阻害している項目。機能追加の前に解消することが望ましい。
- **Low (低)**: コード品質や一貫性に関わる項目。余裕がある際に対応する。

## 対象範囲
- Goコード: `internal/app`, `internal/domain`, `internal/usecase`, `internal/infra`, `internal/dto`, `internal/info`
- ドキュメント: `docs/API.md`, `docs/domain_model_specification.md`
- レポート: `_report/*.md`

## 作業者へ注意
解決した事項は「解決済み」と追記せず、**必ずこの文書から削除してください**。

---

## 課題一覧

### セキュリティ (SEC)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **SEC-03** | **Medium** | `#nosec` コメントの妥当性レビュー不足 | `internal/app/apierror/codes.go` の `G101` 抑制はコメント根拠がなく、`internal/usecase/player_data_usecase_impl.go` の `G115` 抑制も説明不足です。`internal/dto/worldsend_dto.go` や `internal/infra/models/*record*_model.go` の `G115` は理由付きですが、`Value()` 失敗時の扱いも含めた棚卸しが未完了です。 |
| **SEC-04** | **Medium** | HTTPサーバーのタイムアウト未設定 | `internal/app/server.go` で `echo.Start` を直接使っており、`ReadHeaderTimeout` / `ReadTimeout` / `WriteTimeout` / `IdleTimeout` が明示されていません。Slowloris 系のリソース枯渇対策として、`http.Server` を明示生成してタイムアウトを設定すべきです。 |
| **SEC-05** | **Medium** | DB接続のTLS設定がない | `internal/infra/db/connection.go` のMySQL DSNに `tls` 指定がありません。DBが同一ホストまたは信頼できる閉域網に限定されない場合、通信経路上の盗聴・改ざんリスクがあります。本番設定ではTLS必須化を検討すべきです。 |
| **SEC-06** | **Low** | CORS設定の危険値検証がない | `internal/app/router.go` のCORSは設定値をそのまま反映しています。`allow_credentials=true` と広すぎる `allow_origins` の組み合わせを設定時に拒否するなど、起動時検証を追加すべきです。 |
| **SEC-07** | **Low** | ログ出力のサニタイズ方針が限定的 | `internal/app/middleware/error_handler.go` ではエラー文字列の改行除去がありますが、全ログ出力で統一された機微情報・制御文字サニタイズ方針は見当たりません。`router.go` の health check では `slog.Error("Database health check failed: " + err.Error())` のような文字列連結ログも残っています。ログ注入やトークン・UID等の混入を防ぐため、ログ出力ルールを統一すべきです。 |

### パフォーマンス (PERF)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **PERF-003** | **Medium** | ユーザーレコードAPIが全件返却前提 | `GetUserProfileWithRecords` と `GetUserProfileRecordView` は `records.all` と `records.worldsend` をまとめて返しており、ページネーションがありません。`view=rating` で軽量化できる経路はありますが、レコード一覧系はユーザーの蓄積データ増加に比例してレスポンスが肥大化します。 |
| **PERF-004** | **Medium** | レコード表示系の `FindByPlayerID` / `FindByPlayerID`(WORLD'S END) が全件取得 | `user_usecase_impl.go` の profile / record 系では通常譜面・WORLD'S ENDともに `FindByPlayerID` で全件取得してからDTO化や未プレイ補完を行っています。`view=rating` では `FindByPlayerIDForRating` に分離済みですが、レコード表示用途は用途別取得への分割余地があります。 |
| **PERF-005** | **Low** | 楽曲一覧APIが全件返却前提 | `song_handler.go` / `v1/song_handler.go` / `worldsend_handler.go` / `v1_worldsend_handler.go` の楽曲一覧はページネーションなしで全件返却します。マスターデータ的用途として許容する判断もあり得ますが、データ増加時のレスポンス肥大化リスクは `PERF-003` と同様に棚卸しすべきです。 |

### 信頼性・運用 (OPS)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **OPS-001** | **Low** | リクエストIDがない | `X-Request-ID` 付与やログへの相関ID埋め込みがなく、障害解析時のトレース性が低い状態です。 |
| **OPS-002** | **Low** | DBクエリの明示的タイムアウトなし | `context.Context` は伝播されていますが、`context.WithTimeout` 等による通常リクエスト中のDBアクセス上限時間設定が見当たりません。`router.go` の health check も `db.Ping()` を使っており、リクエストContextを使っていません。 |

### 実装品質・保守性 (QUAL)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **QUAL-002** | **Medium** | セキュリティヘッダー未設定 | Echo の `Secure` ミドルウェア相当の設定がなく、HSTS、`X-Content-Type-Options`、`X-Frame-Options` などの標準ヘッダーが不足しています。 |
| **QUAL-010** | **Medium** | Domain層の `Executor` が `sqlx` に依存 | `internal/domain/repository/executor.go` が `*sqlx.Rows`, `*sqlx.Row` を直接公開しており、ドメイン層がインフラ実装詳細に依存しています。 |
| **QUAL-011** | **Low** | `interface{}` が残存 | AGENTS.md では `interface{}` を禁止し `any` を使用する方針ですが、`internal/infra/repository/player_locked_song_repository_impl.go` に `[]interface{}` が残っています。 |
| **QUAL-012** | **Low** | テストコードに `t.Errorf` / `t.Fatalf` の直接使用が多数残存 | AGENTS.md では新規テストのアサーションに `assert` / `require` を使う方針ですが、既存テストには `t.Errorf` / `t.Fatalf` / `t.Fatal` の直接使用が多数残っています。既存テストの全面置換は優先度を下げつつ、新規・更新テストから統一すべきです。 |

### アーキテクチャ・ドメイン (ARCH / DOM / DTO)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **ARCH-002** | **Low** | `OfficialSongWithGenreDTO` に `db:` タグが残存 | `internal/dto/official_dto.go` の DTO がDBタグを持っています。DTO層に永続化都合が漏れており、さらに現状参照箇所も見当たりません。削除または `infra/models` への移動を検討すべきです。 |
| **DOM-006** | **Medium** | `Goal` エンティティが `[]byte` でJSONを保持 | `internal/domain/entity/goal.go` の `AchievementParams` / `Attributes` はインフラ表現に引きずられたままです。型安全な表現への移行余地があります。 |
| **DOM-007** | **Low** | 本番コードに `Must` 系Value Objectコンストラクタが残存 | AGENTS.md ではテストコードを除き `panic` を起こす `Must` 系関数を避ける方針ですが、`internal/domain/vo/username/username.go` / `reauthtoken/reauth_token.go` / `playername/playername.go` に `MustNew...` が定義されています。利用箇所は主にテストですが、本番パッケージの公開APIとして残っています。 |
| **DTO-001** | **Low** | `GoalRequest` / `GoalResponse` が `map[string]any` 依存 | `internal/dto/api_internal/goal_dto.go` で型安全性がなく、スキーマの明確性も低い状態です。現状はUsecase側で厳しめに検証していますが、DTO設計としては弱いままです。 |

### インフラ層 (INFRA)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **INFRA-002** | **Low** | `validation.go` のテーブル名組み立てが文字列連結 | 現状は固定値しか使っていませんが、`fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)` に依存しています。ホワイトリスト化して誤用余地をなくすべきです。 |
| **INFRA-005** | **Low** | `validation.go` がContext非対応 | `ValidateRequiredData` / `GetTableStats` は `context.Context` を受け取らず、`db.Get` を使っています。起動時専用でも、I/O規約の一貫性は崩れています。 |
| **INFRA-007** | **Medium** | `FindAllWithPlayer` と `FindAllWithPlayerForAdmin` の重複 | `internal/infra/repository/user_repository_impl.go` で、クエリ構築・LIKE検索・rows処理がかなり重複しています。 |
| **INFRA-009** | **Medium** | 譜面定数・ノーツ変換時のエラー無視 | `internal/infra/models/song_chart_model.go` の `FromChartEntity` では `ParseFloat` のエラーチェックは追加されましたが、`e.Notes.Value()` / `e.Const.Value()` のエラーは依然として `_` で破棄しています。`internal/infra/repository/song_repository_impl.go` の `toChartEntity` も `chartconstant.NewChartConstant` / `notes.NewNotes` のエラーを無視しています。 |
| **INFRA-010** | **Low** | 一時プレイヤーデータリポジトリが `context.Context` を無視 | `internal/infra/repository/temporary_player_data_repository_impl.go` の `Create` / `FindByToken` / `ConsumeByToken` / `Delete` は `context.Context` を受け取りながら `_ context.Context` として無視しています。インメモリ実装でも、ロック取得前後や重いペイロード処理前にキャンセルを確認するなど、リポジトリ契約の一貫性を保つべきです。 |
| **INFRA-016** | **Medium** | スコアVOからの変換でエラー無視 | `internal/infra/models/player_record_model.go` と `player_worldsend_record_model.go` が `Value()` のエラーを `_` で破棄し、`scoreVal.(int64)` の型アサーション前提で変換しています。`#nosec G115` コメントに範囲保証の根拠は記述されましたが、`scoreVal` が `nil` の場合（`Value()` が失敗した場合）にパニックが発生するリスクは残存しています。 |

### ユースケース層 (UC)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **UC-013** | **Medium** | `goalUsecase.Update` が非トランザクション | `Create` は `tm.Transactional` を使う一方、`Update` は `u.db` へ直接アクセスしています。 |
| **UC-014** | **Medium** | WORLD'S ENDレコード取得エラーを握りつぶしている | `user_usecase_impl.go` の `getUserProfileWorldsendRecords` は `worldsendRecordRepo.FindByPlayerID` のエラーをログ出力後に空スライス・nil errorとして返します。通常譜面側はエラーを返すため挙動が不一致で、障害時に部分的な欠損レスポンスを正常扱いするリスクがあります。 |

### ハンドラー / ルーター層 (HDL)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **HDL-001** | **Medium** | `RealIP()` の信頼境界未設定 | `router.go` で `e.IPExtractor` を設定しておらず、リバースプロキシ配下での `RealIP()` 利用が危険です。レートリミットやログに影響します。 |
| **HDL-002** | **Medium** | `displayid` パスパラメータ未検証 | `song_handler.go` などで `displayid` をそのままUsecaseへ渡しています。更新APIのリクエストボディでは長さ検証しており、方針が不統一です。compat chunirec の `id` クエリも空文字以外の形式検証がありません。 |
| **HDL-003** | **Medium** | `username` パスパラメータ未検証 | `user_handler.go` / `api_v1/user_handler.go` などで `username` を境界で検証していません。compat chunirec の `user_name` クエリも、未指定時のフォールバック以外に形式検証がありません。 |
| **HDL-010** | **Low** | `knownFields` が手書きハードコード | `me_handler.go` 内の未知フィールド検出は `PlayerDataPayload` と手動同期になっており、メンテナンス漏れの温床です。 |
| **HDL-011** | **Low** | `include_noplay` の不正値を黙って `false` 扱い | `user_handler.go` / `api_v1/user_handler.go` で `strconv.ParseBool(c.QueryParam("include_noplay"))` のエラーを `_` で破棄しており、不正なクエリ値がバリデーションエラーになりません。 |
| **HDL-012** | **Low** | 厳格JSONデコードの適用が不統一 | `BindStrictJSON` が導入されていますが、`login_handler.go` / `song_handler.go` / `worldsend_handler.go` / `honor_handler.go` などに `c.Bind` が残っています。未知フィールドの扱いがエンドポイントごとに不統一です。 |

---

## まとめ

- 優先度が高いのは、**Goal更新の非トランザクション**、**Domain層の `sqlx` 依存**、**WORLD'S ENDレコード取得エラーの握りつぶし** です。
- 次に、**エラー変換の不統一**, **パスパラメータ未検証**, **巨大レスポンス / 全件取得**, **VO変換時のエラー無視**, **Context伝播の不徹底** を詰めると、APIの安定性と保守性が上がります。
- AGENTS.md 準拠の観点では、**`interface{}` 残存**, **本番パッケージの `Must` 系関数**, **テストでの `t.Errorf` / `t.Fatalf` 直接使用**, **厳格JSONデコードの不統一** が追加の棚卸し対象です。
- `refactor.md` は現在の未解消課題だけを残したため、今後は項目を消し込んでいけば現状把握に使いやすい状態です。
