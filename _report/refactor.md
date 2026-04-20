# リファクタリング指摘書 (2026-04-18時点)

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
| **SEC-03** | **Medium** | `#nosec` コメントの妥当性レビュー不足 | `internal/app/apierror/codes.go` の `G101` 抑制はコメント根拠がなく、`internal/usecase/player_data_usecase_impl.go` の `G115` 抑制も説明不足です。他は概ね理由付きですが、全体の棚卸しが未完了です。 |

### パフォーマンス (PERF)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **PERF-003** | **Medium** | ユーザーレコードAPIが全件返却前提 | `GetUserProfileWithRecords` は `records.all` と `records.worldsend` をまとめて返しており、ページネーションがありません。ユーザーの蓄積データ増加に比例してレスポンスが肥大化します。 |
| **PERF-004** | **Medium** | `FindByPlayerID` / `FindByPlayerID`(WORLD'S END) が全件取得 | `user_usecase_impl.go` では通常譜面・WORLD'S ENDともに全件取得してからDTO化や未プレイ補完を行っています。用途別取得への分割余地があります。 |

### 信頼性・運用 (OPS)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **OPS-001** | **Low** | リクエストIDがない | `X-Request-ID` 付与やログへの相関ID埋め込みがなく、障害解析時のトレース性が低い状態です。 |
| **OPS-002** | **Low** | DBクエリの明示的タイムアウトなし | `context.Context` は伝播されていますが、`context.WithTimeout` 等によるDBアクセスの上限時間設定が見当たりません。 |

### 実装品質・保守性 (QUAL)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **QUAL-001** | **Low** | TODOコメントが2件残存 | `internal/app/router.go` に2件（L166: 外部向けhealthエンドポイント、L384: 同関連）残っています。`profile_handler.go` の1件は解消済み。未着手ならIssue化、不要なら削除すべきです。 |
| **QUAL-002** | **Medium** | セキュリティヘッダー未設定 | Echo の `Secure` ミドルウェア相当の設定がなく、HSTS、`X-Content-Type-Options`、`X-Frame-Options` などの標準ヘッダーが不足しています。 |
| **QUAL-010** | **Medium** | Domain層の `Executor` が `sqlx` に依存 | `internal/domain/repository/executor.go` が `*sqlx.Rows`, `*sqlx.Row` を直接公開しており、ドメイン層がインフラ実装詳細に依存しています。 |

### アーキテクチャ・ドメイン (ARCH / DOM / DTO)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **ARCH-002** | **Low** | `OfficialSongWithGenreDTO` に `db:` タグが残存 | `internal/dto/official_dto.go` の DTO がDBタグを持っています。DTO層に永続化都合が漏れており、さらに現状参照箇所も見当たりません。削除または `infra/models` への移動を検討すべきです。 |
| **DOM-006** | **Medium** | `Goal` エンティティが `[]byte` でJSONを保持 | `internal/domain/entity/goal.go` の `AchievementParams` / `Attributes` はインフラ表現に引きずられたままです。型安全な表現への移行余地があります。 |
| **DTO-001** | **Low** | `GoalRequest` / `GoalResponse` が `map[string]any` 依存 | `internal/dto/api_internal/goal_dto.go` で型安全性がなく、スキーマの明確性も低い状態です。現状はUsecase側で厳しめに検証していますが、DTO設計としては弱いままです。 |

### インフラ層 (INFRA)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **INFRA-002** | **Low** | `validation.go` のテーブル名組み立てが文字列連結 | 現状は固定値しか使っていませんが、`fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)` に依存しています。ホワイトリスト化して誤用余地をなくすべきです。 |
| **INFRA-005** | **Low** | `validation.go` がContext非対応 | `ValidateRequiredData` / `GetTableStats` は `context.Context` を受け取らず、`db.Get` を使っています。起動時専用でも、I/O規約の一貫性は崩れています。 |
| **INFRA-007** | **Medium** | `FindAllWithPlayer` と `FindAllWithPlayerForAdmin` の重複 | `internal/infra/repository/user_repository_impl.go` で、クエリ構築・LIKE検索・rows処理がかなり重複しています。 |
| **INFRA-009** | **Medium** | 譜面定数・ノーツ変換時のエラー無視 | `internal/infra/models/song_chart_model.go` の `FromChartEntity` では `ParseFloat` のエラーチェックは追加されましたが、`e.Notes.Value()` / `e.Const.Value()` のエラーは依然として `_` で破棄しています。`internal/infra/repository/song_repository_impl.go` の `toChartEntity` も `chartconstant.NewChartConstant` / `notes.NewNotes` のエラーを無視しています。 |
| **INFRA-010** | **Medium** | `BulkAssignHonors` にチャンク分割がない | `internal/infra/repository/honor_repository_impl.go` は全件を単一INSERTで投げています。他のバルク処理は `info.BulkInsertChunkSize` を使っており不整合です。 |
| **INFRA-011** | **Medium** | `resolveExecutor` の暗黙nilフォールバック | `internal/infra/repository/player_data_repository_impl.go` で `exec == nil` 時に `r.db` へフォールバックします。トランザクション必須箇所で誤って外側DB実行に落ちる危険があります。 |
| **INFRA-012** | **Low** | `ClassEmblem` 系の逆引きが線形探索 | `GetClassEmblemNameByID` / `GetClassEmblemBaseNameByID` / `GetAccountTypeNameByID` が map を持ちながら毎回線形探索しています。 |
| **INFRA-016** | **Medium** | スコアVOからの変換でエラー無視 | `internal/infra/models/player_record_model.go` と `player_worldsend_record_model.go` が `Value()` のエラーを `_` で破棄し、`scoreVal.(int64)` の型アサーション前提で変換しています。`#nosec G115` コメントに範囲保証の根拠は記述されましたが、`scoreVal` が `nil` の場合（`Value()` が失敗した場合）にパニックが発生するリスクは残存しています。 |

### ユースケース層 (UC)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **UC-008** | **Medium** | `applyScores` が巨大で重複も多い | 通常譜面とWORLD'S END譜面の分岐が長大で、解決ロジックも類似しています。分割余地が大きい状態です。 |
| **UC-011** | **Medium** | `Service` / `Usecase` 命名が混在 | `NewAPITokenService`, `NewUserService`, `NewSongService`, `NewPlayerDataService` と `NewAuthUsecase`, `NewGoalUsecase` などが混在しています。 |
| **UC-013** | **Medium** | `goalUsecase.Update` が非トランザクション | `Create` は `tm.Transactional` を使う一方、`Update` は `u.db` へ直接アクセスしています。 |

### ハンドラー / ルーター層 (HDL)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **HDL-001** | **Medium** | `RealIP()` の信頼境界未設定 | `router.go` で `e.IPExtractor` を設定しておらず、リバースプロキシ配下での `RealIP()` 利用が危険です。レートリミットやログに影響します。 |
| **HDL-002** | **Medium** | `displayid` パスパラメータ未検証 | `song_handler.go` などで `displayid` をそのままUsecaseへ渡しています。更新APIのリクエストボディでは長さ検証しており、方針が不統一です。 |
| **HDL-003** | **Medium** | `username` パスパラメータ未検証 | `user_handler.go` / `api_v1/user_handler.go` などで `username` を境界で検証していません。 |
| **HDL-004** | **Medium** | Usecaseエラー変換の不統一 | `DeleteSong`, `RestoreSong`, `AuthHandler.Logout`, `APITokenHandler.Generate/Delete`, `ProfileHandler.UpdatePrivacy` などで `apierror.FromUsecaseError` を使わず、一律 `internal_error` に寄せています。 |
| **HDL-010** | **Low** | `knownFields` が手書きハードコード | `me_handler.go` 内の未知フィールド検出は `PlayerDataPayload` と手動同期になっており、メンテナンス漏れの温床です。 |

---

## 補足

- Firebase 認証への移行で、Cookie セッション前提の CSRF、`password_hash`、`user_recovery_codes`、旧 `auth_usecase_impl.go` に依存した指摘は現状と一致しなくなったため削除しました。
- 逆に、`TODO` 件数や `#nosec` 箇所、`song_repository_impl.go` のVO変換エラー無視のように、**根拠は同じテーマでも現行コード上の実態に合わせて記述を更新**しています。

## まとめ

- 優先度が高いのは、**Goal更新の非トランザクション** と **Domain層の `sqlx` 依存** です。
- 次に、**エラー変換の不統一**, **パスパラメータ未検証**, **巨大レスポンス / 全件取得**, **VO変換時のエラー無視** を詰めると、APIの安定性と保守性が上がります。
- `refactor.md` は現在の未解消課題だけを残したため、今後は項目を消し込んでいけば現状把握に使いやすい状態です。
