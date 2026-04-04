# リファクタリング指摘書 (Current Code Issues)

本ドキュメントは、コードベース評価に基づき修正が必要な項目を整理したものです。
`chunisupport-api` (Go 1.26, Echo, MySQL, ~50k users) の規模と特性、および Clean Architecture/DDD を考慮し、実効性の高い項目に絞り込んでいます。

## 優先度定義
- **Critical (緊急)**: セキュリティ上の重大な欠陥、または主要機能の停止に直結する問題。即時対応が必要。
- **High (高)**: アーキテクチャの根幹に関わる、またはセキュリティ・安定性に重大なリスクがある項目。最優先で対応が必要。
- **Medium (中)**: 保守性や拡張性を阻害している項目。機能追加の前に解消することが望ましい。
- **Low (低)**: コード品質や一貫性に関わる項目。余裕がある際に対応する。

## 対象範囲
- Goコード: `main.go`, `internal/app`, `internal/auth`, `internal/config`, `internal/domain`, `internal/usecase`, `internal/infra`, `internal/dto`, `internal/info`, `internal/utils`
- 設定/環境: `internal/config` の設定ローダ
- CI: `.github/workflows/ci.yml`
- 依存関係: `go.mod`, `go.sum`
- DB: `migration/mysql/*.sql`
- ドキュメント: `docs/API.md`
- アーキテクチャ: `ARCHITECTURE.md`

## 解析手順
1. 全体構造の把握（`internal/` + `ARCHITECTURE.md`）
2. 認証・認可フロー追跡（`internal/auth`, `internal/app/middleware`, `internal/app/handler`）
3. 入力点ごとの検証（HTTP境界、DTO、バッチ）
4. データアクセス層（SQL/トランザクション/Context）
5. 並行処理・キャッシュ・外部I/O（timeout/retry）
6. ログ・panic・メトリクスの情報漏えい/観測性
7. 依存関係・設定の危険値や既知脆弱性

## 作業者へ注意
解決した事項は「解決済み」と記載したりすることはなく、**必ず削除してください**。

---

## 課題一覧

### 統合課題一覧（類似項目の再編）

似ている課題、および同時に解決することで効果が高い課題を下記の単位に統合しました。

| 統合ID | 統合テーマ | まとめたID | 統合理由・同時解決方針 |
|---|---|---|---|
| **REF-G01** | 認証・セッション境界の防御強化 | SEC-01, SEC-05, SEC-011 | 認証周辺の攻撃面（CSRF、タイミング攻撃、パスワード複雑性）を同時に見直すことで、脅威モデル・実装を一括で整合できる。 |
| **REF-G02** | 入力検証・エラー変換の境界統一 | HDL-002, HDL-003, HDL-004, UC-005, DTO-001 | HTTP境界での入力検証不足と、層間エラー変換の不整合は同じ「境界責務」の問題。バリデーション方針とエラー変換規約を同時整備する。 |
| **REF-G03** | ドメイン純粋性の回復（インフラ依存排除） | DOM-006, DOM-017, ARCH-002 | ドメイン/DTO側にインフラ都合（dbタグ、JSONバイト生保持）が混入。モデルの責務分離を同時実施して依存方向を正す。 |
| **REF-G04** | 値オブジェクトの整合性・型安全性向上 | DOM-008, INFRA-009, INFRA-016 | VOバリデーション迂回・危険な型変換・エラー無視が連鎖している。VOの生成/変換/永続化パスを一体で修正する。 |
| **REF-G05** | Domain層のインフラ依存排除 | QUAL-010 | Domain層のsqlx依存を整理し、クリーンアーキテクチャ違反を解消する。 |
| **REF-G07** | トランザクション整合性と実行器契約の統一 | UC-004, UC-013, INFRA-011 | トランザクション欠如と暗黙フォールバックは同系統の整合性リスク。境界をまたぐ処理を「必ずTxで完結」に統一する。 |
| **REF-G08** | クエリ負荷・バルク処理最適化 | PERF-003, PERF-004, INFRA-010, INFRA-012 | 全件取得・無分割バルクなど、DB負荷起因の課題群。取得戦略とチャンク戦略を同時に最適化する。 |
| **REF-G09** | 監視性・運用信頼性の標準化 | OPS-001, OPS-002, INFRA-005, UC-014, HDL-009, LIB-004 | リクエスト追跡、タイムアウト、キャンセルログ、ログ運用の課題をまとめて扱い、運用観測性を標準化する。 |
| **REF-G11** | コード重複削減と共通化 | UC-006, UC-008, HDL-005, INFRA-007, DOM-012 | 各層に散在する重複ロジックを、ユースケース/ハンドラ/リポジトリ単位の共通ヘルパーへ抽出して保守性を改善する。 |
| **REF-G12** | コーディング規約・命名・近代化の統一 | DOM-016, UC-009, UC-011, UC-012, DOM-013, DOM-011, QUAL-001 | `slices` への統一、命名規約、メッセージ言語統一、重複定数整理、TODO解消などを同時に進めてコード規律を揃える。 |
| **REF-G14** | セキュリティ運用の継続的検証 | SEC-03, HDL-001, INFRA-002, LIB-005 | 単発修正ではなく、抑制コメント妥当性・IP抽出・SQL安全性・転送効率を含む運用時の継続検証項目として同時管理する。 |


### セキュリティ (SEC)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **SEC-01** | **High** | CSRF対策不足 | Double Submit Cookie または Synchronizer Token を導入。SameSite=Lax/Strict と Origin/Referer 検証を併用。 |
| **SEC-011** | **High** | パスワード複雑性要件の欠如 | 長さチェックのみ。`zxcvbn-go` 等の導入または正規表現による文字種チェックを追加。 |
| **SEC-03** | **Medium** | `#nosec` コメントの妥当性レビュー未実施 | `gosec` などで抑制箇所を洗い出し、根拠を明記。不必要な抑制は削除。 |

### パフォーマンス (PERF)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **PERF-003** | **Medium** | 巨大レスポンスの生成 | `GetUserProfileWithRecords` が全件返却。レスポンス簡素化またはページネーション導入。 |
| **PERF-004** | **Medium** | プレイヤーレコードの全件取得 | `FindByPlayerID` が全レコードを取得し、複数のユースケースで使用。必要なデータのみ取得する最適化を検討。 |

### 信頼性・運用 (OPS)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **OPS-001** | **Low** | リクエストIDの欠如 | ログにリクエストを一意に識別するID（X-Request-ID等）が付与されておらず、分散環境でのトレーサビリティが低い。 |
| **OPS-002** | **Low** | DBクエリタイムアウトの未設定 | リクエストContextはDB操作に渡されているが、明示的なクエリタイムアウト設定がない。 |

### 実装品質・保守性 (QUAL)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **QUAL-001** | **Low** | TODOコメントの残置 | 解消またはIssue化。現状1件残存（詳細は後述）。 |
| **QUAL-002** | **Medium** | セキュリティヘッダーの欠如 | Echoの `Secure` ミドルウェア導入でHSTS等を設定。 |
| **QUAL-006** | **Medium** | コンストラクタのエラー無視 | `toChartEntity` 等で値オブジェクトの生成エラーを無視している。不整合なエンティティが生成されるリスク。 |
| **QUAL-010** | **Medium** | Domain層のExecutorインターフェースがsqlxに依存 | `internal/domain/repository/executor.go` で `*sqlx.Rows`, `*sqlx.Row` を直接参照。ドメイン層がインフラ実装に依存している。 |

---

## 詳細

### SEC-01: Cookieベース認証に対するCSRF対策不足
- **根拠**:
  - 認証情報はCookie（`token`）で保持され、JWTはCookieから取得されます。`AuthHandler` がCookieを発行し `JWTMiddleware` がCookieを検証しています（`internal/app/handler/api_internal/auth_handler.go`, `internal/app/middleware/auth_middleware.go`）。
- **影響範囲**:
  - 認証済みユーザーが悪意あるサイトを閲覧しただけで、`/internal/me/privacy` や `/internal/me/password` などの状態変更系APIが第三者の意図で実行される可能性。
  - 特に `SameSite=None` を設定した場合はCSRF耐性がほぼ失われます。
- **再現手順**:
  1. 被害者がログイン済み（Cookieに`token`がある状態）。
  2. 攻撃者が自サイトから `POST /internal/me/password` などのフォーム送信を行う。
  3. サーバー側にCSRF検証がないためリクエストが通る。
- **修正案**:
  - **Double Submit Cookie** または **CSRFトークン（Synchronizer Token）** を導入。
  - 併せて `SameSite=Lax/Strict` の強制と `Origin`/`Referer` 検証を追加。
  - 状態変更系ルート（`/internal/me/*`, `/internal/auth/*`）で必須。
- **追加で確認したい点**:
  - `CookieSameSite` がどの環境で `None` になっているか。実運用値の確認が必要（`internal/config/config.go`, `internal/app/router.go`）。

---

### SEC-03: `#nosec` コメントの妥当性レビュー未実施
- **根拠**:
  - 複数の `#nosec` コメントが存在し、抑制の妥当性が未レビュー。
- **現状の`#nosec`箇所一覧**:
  | ファイル | 行 | 抑制内容 | 根拠の有無 |
  |---|---|---|---|
  | `internal/app/apierror/codes.go` | 12-15, 41 | G101（ハードコードされたクレデンシャル疑い） | △（エラーコード定数であり実際のクレデンシャルではないが、コメントなし） |
  | `internal/config/config.go` | 48 | G117（ハードコードされたパスワード疑い） | ○（JWTシークレットを環境変数から読み込むフィールドであり、値はコードに埋め込まれていない） |
  | `internal/config/config.go` | 99 | G703, G304（ファイルパス挿入） | ○（`APP_ENV` は `validateEnv` で許可値に制限済み） |
  | `internal/app/router.go` | 442 | G304（ファイルパス挿入） | ○（「LogPaths.Echo comes from trusted configuration」とコメントあり） |
  | `internal/app/handler/api_internal/auth_handler.go` | 32 | G117（API入力仕様として必要） | ○（コメントあり） |
  | `internal/app/handler/api_internal/auth_handler_shared.go` | 15 | G124（Cookie Secure属性） | ○（本番環境ではSecure=trueを設定する旨のコメントあり） |
  | `internal/infra/logger/handler.go` | 59 | G304（ファイルパス挿入） | ○（「logDir comes from trusted configuration」とコメントあり） |
  | `internal/dto/worldsend_dto.go` | 38 | G115（整数オーバーフロー） | ○（「Score value is guaranteed to be within uint32 range by domain VO」とコメントあり） |
  | `internal/infra/models/player_record_model.go` | 48 | G115（整数オーバーフロー） | ○（同上） |
  | `internal/infra/models/player_worldsend_record_model.go` | 43 | G115（整数オーバーフロー） | ○（同上） |
  | `internal/usecase/player_data_usecase_impl.go` | 941 | G115（整数オーバーフロー） | △（コメントなし） |
- **影響範囲**:
  - 実際の脆弱性（パス・トラバーサル、整数オーバーフローなど）を見逃す可能性。
- **修正案**:
  - `internal/app/apierror/codes.go`：エラーコード定数であり実際のクレデンシャルではないことを明記（例: `// #nosec G101 -- これはエラーコード定数であり、実際のクレデンシャルではない`）
  - `internal/usecase/player_data_usecase_impl.go:941`：根拠コメントを追加
  - その他の箇所は適切な根拠コメントあり

---

### QUAL-006: コンストラクタのエラー無視
- **根拠**:
  - `songRepository.toChartEntity` 内で `chartconstant.NewChartConstant(row.Const)` などのエラー戻り値を捨てている。
- **影響範囲**:
  - DBに不正な値が入っていた場合、異常な状態でドメイン層にデータが渡り、予期せぬ挙動や計算ミスを引き起こす。
- **修正案**:
  - エラーが発生した場合は上位に返し、データの整合性エラーとして適切にログ出力・処理する。

### QUAL-001: TODOコメントの残置
- **根拠**:
  - 以下のTODOコメントが残存している。
- **現状のTODO箇所一覧**:
  | ファイル | 行 | 内容 |
  |---|---|---|
  | `internal/usecase/auth_usecase_impl.go` | 148 | `// TODO: internal/domain/vo/username パッケージでエラー変数を公開し、errors.Is() を使った判定に切り替える。` |
- **修正案**:
  - 解消またはIssue化。UC-005と同時に対応可能。

---

### UC-004: Register でのトランザクション未使用
- **根拠**:
  - `internal/usecase/auth_usecase_impl.go` の `Register` でユーザー保存（`Save`）とセッション作成（`issueSession`）が非トランザクションで連続実行。
- **影響範囲**:
  - ユーザー保存成功後にセッション作成が失敗すると、ログインできないユーザーが作成される。
- **修正案**:
  - `Register` 全体を `s.tm.Transactional` で囲む。

---

### UC-005: `convertUsernameError` の文字列比較によるエラー変換
- **根拠**:
  - `internal/usecase/auth_usecase_impl.go` の `convertUsernameError` がVOのエラーメッセージ文字列と直接比較してusecase層のエラーに変換。
  - 例: `errMsg == "username cannot be empty"` → `return ErrUsernameEmpty`
- **影響範囲**:
  - VO側のメッセージが変更されると変換が壊れ、適切なusecase層エラーが返されなくなる。
- **修正案**:
  - `username` パッケージにセンチネルエラー（`var ErrEmpty = errors.New(...)` 等）を定義し、`errors.Is` で判定する。

---

### DOM-008: `Notes.Scan` がバリデーションをバイパス
- **根拠**:
  - `internal/domain/vo/notes/notes.go` の `Scan` が `Notes(v)` で直接キャストし、`NewNotes` のバリデーション（0以上）を経由しない。
  - 一方、`internal/domain/vo/score/score.go` の `Scan` は負値チェック・最大値チェックを適切に実施しており、模範的な実装。
  - `internal/domain/vo/chartconstant/chartconstant.go` の `Scan` と `UnmarshalJSON` は `NewChartConstant` を経由するよう修正済み。
- **影響範囲**:
  - DBに不正な値（負値等）が存在した場合、バリデーションなしで不正なVOが生成される。
- **修正案**:
  - `Scan` 内で `NewNotes` を経由してバリデーションを実行する。`score.Score` のパターンに統一。

---

### HDL-004: エラーハンドリングが粗い箇所の複数存在
- **根拠**:
  - 以下のハンドラでユースケースエラーを一律 `ErrInternalError` にしているが、`FromUsecaseError()` を使って適切なHTTPステータスに変換すべき。
  - 該当箇所:
    - `api_internal/song_handler.go`: `DeleteSong`, `RestoreSong`
    - `api_internal/api_token_handler.go`: `Generate`, `Delete`
    - `api_internal/auth_handler.go`: `Logout`
    - `api_internal/profile_handler.go`: `UpdatePrivacy`
  - 他のハンドラ（WorldsendHandler等）は正しく `FromUsecaseError()` を使用しており不整合。
- **影響範囲**:
  - 404系エラー（楽曲/トークン/ユーザー未発見）が500で返されてしまい、クライアント側の適切なエラーハンドリングが困難。
- **修正案**:
  - 全箇所で `apierror.FromUsecaseError(err)` を使うように統一。

---

## 実装簡素化・ライブラリ活用提案 (LIB)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **LIB-003** | **Low** | コレクション操作の効率化 | 冗長なループ処理（Map, Filter, Uniq 等）を `samber/lo` で置き換え、コードを簡潔にする。 |
| **LIB-004** | **Low** | ログファイルローテーション | 現状は起動毎に新ファイルを作成する形式だが、日付ベースのローテーションやサイズ制限がない。運用期間が長くなるとログファイルが肥大化する可能性がある。`lumberjack` などのライブラリ導入、または日付ベースのファイル切り替えロジックを実装する。 |
| **LIB-005** | **Low** | レスポンス圧縮ミドルウェアの導入検討 | 現状はリクエストのgzip解凍は実装されているが、レスポンスの圧縮は行われていない。大量のレコードを返すエンドポイントでは、gzip圧縮ミドルウェアの導入で帯域削減が期待できる。 |

---

### ドメイン層設計 (DOM)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **DOM-006** | **Medium** | `Goal` エンティティが貧血症モデル＋`[]byte`フィールド | `AchievementParams []byte` と `Attributes []byte` はJSONバイト列の生保持であり、インフラ層の都合がドメイン層に漏洩している。適切な構造体やマップに変換すべき。 |
| **DOM-008** | **Medium** | `Notes.Scan` がバリデーションをバイパス | `Notes(v)` で直接キャストしており、`NewNotes` のバリデーション（0以上）を経由しない。`score.Score` の `Scan` 実装を模範にすべき。 |
| **DOM-011** | **Medium** | 理論値スコア定数の二重定義 | `internal/domain/service/info.go` の `theoreticalScore uint32 = 1010000` と `internal/info/info.go` の `TheoreticalScore = 1010000` が重複。1箇所に集約すべき。 |
| **DOM-012** | **Low** | `WorldsendSongWithChart` と `WorldsendSongChartPair` の重複 | `repository` 層と `service` 層にフィールド同一の重複構造体。entity層に統一構造体を定義すべき。 |
| **DOM-013** | **Low** | エラーメッセージの日英混在 | 値オブジェクトは英語、エンティティバリデーションは日本語。同一パッケージ内でも混在あり（例: `notes.go` の `Scan` 内で日本語メッセージ）。方針を統一すべき。 |
| **DOM-016** | **Low** | `record_completion_service.go` が `sort.Slice` 使用 | `rating_service.go` は `slices.SortFunc` 使用。Go 1.26で推奨される `slices` パッケージに統一すべき。 |
| **DOM-017** | **Low** | `PlayerHonor` がrepository層に定義 | ドメイン概念だが `repository` パッケージ内に定義。`entity` パッケージに移動すべき。 |
| **DOM-021** | **Low** | Deprecated関数が残存 | `rating_service.go` の `CalcBestAverageRating`, `CalcNewAverageRating`, `CalcPlayerRating`。本体コードからは未使用だがテストで参照中。移行完了後に削除すべき。 |

---

### インフラ層 (INFRA)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **INFRA-002** | **Low** | `validation.go` のテーブル名組み立ての安全性改善余地 | 現状は内部固定値のみで直ちに脆弱性とは言えないが、将来の誤用防止のためテーブル名をホワイトリスト化し、任意入力を受け付けないAPIに制限すべき。 |
| **INFRA-005** | **Low** | `validation.go` の全関数でContext未伝播 | `internal/infra/db/validation.go` が `context.Context` を引数に取らず、`db.Get` を使用（`GetContext` ではない）。起動時専用のため影響は限定的だが、コード一貫性の観点で改善が望ましい。 |
| **INFRA-007** | **Medium** | `FindAllWithPlayer` と `FindAllWithPlayerForAdmin` のコード重複 | クエリ構築・LIKE検索・rows反復がほぼ同一。共通ヘルパーに抽出すべき。 |
| **INFRA-009** | **Medium** | `FromChartEntity` の脆弱な定数変換処理 | `internal/infra/models/song_chart_model.go` の `FromChartEntity` で `Value()` → 型アサーション(string) → `ParseFloat` の多段変換を行い、失敗時に0.0フォールバック。`ChartConstant` に `Float64()` アクセサを追加すべき。 |
| **INFRA-010** | **Medium** | `BulkAssignHonors` にチャンクサイズ制限なし | 全件を1つのINSERTで発行。他のバルク処理は `info.BulkInsertChunkSize` で分割済み。 |
| **INFRA-011** | **Medium** | `resolveExecutor` の暗黙nil フォールバック | `internal/infra/repository/player_data_repository_impl.go` で exec が nil の場合に `r.db` へフォールバック。トランザクション保証が暗黙に破壊されるリスク。他リポジトリはexec必須。 |
| **INFRA-012** | **Low** | `Cache.GetClassEmblemNameByID` 等のO(n)線形探索 | 他のマスタは `*NamesByID` マップでO(1)ルックアップ済み。パターン統一すべき。 |
| **INFRA-016** | **Medium** | `FromPlayerRecordEntity`/`FromPlayerWorldsendRecordEntity` のエラー無視 | `internal/infra/models/player_record_model.go` および `player_worldsend_record_model.go` で `e.Score.Value()` のエラーを `_` で無視し、戻り値に対する型アサーション→uint32キャストで panic 発生の可能性がある。 |

---

### ユースケース層設計 (UC)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **UC-004** | **Medium** | `Register` でトランザクション未使用 | ユーザー保存とセッション作成が非トランザクション。ユーザー保存成功・セッション作成失敗でログイン不能ユーザーが生成されるリスク。 |
| **UC-005** | **Medium** | `convertUsernameError` の文字列比較によるエラー変換 | VOのエラーメッセージ文字列と直接比較。メッセージ変更時に検知不能。VOにセンチネルエラーを定義し `errors.Is` で判定すべき。 |
| **UC-006** | **Low** | パスワードバリデーションロジックの3箇所重複 | `auth_usecase_impl.go`（Register）、`recovery_usecase.go`（RecoverWithRecoveryCode）、`user_credential_usecase.go`（ChangePassword）で長さチェックが重複。`validatePassword` ヘルパーに抽出すべき。 |
| **UC-008** | **Medium** | `applyScores` のGod Function（約200行） | 通常譜面ループとWE譜面ループで解決ロジックがほぼ同一のまま2回繰り返し。共通関数に抽出すべき。 |
| **UC-009** | **Low** | `sort` パッケージ使用（Go 1.26では `slices` 推奨） | `player_data_usecase_impl.go` と `chart_stats_usecase.go` で `sort.Strings`/`sort.Slice` 使用。`slices.Sort`/`slices.SortFunc` に統一すべき。 |
| **UC-011** | **Medium** | コンストラクタ名のService/Usecase混在 | `NewAPITokenService`, `NewPlayerDataService`, `NewPlayerService`, `NewSongService`, `NewUserService` の"Service"接尾辞と `NewAuthUsecase`, `NewChartStatsUsecase`, `NewGoalUsecase`, `NewWorldsendUsecase`, `NewRecoveryUsecase`, `NewSessionUsecase` 等の"Usecase"接尾辞が混在。AGENTS.mdでは `Usecase` を推奨。統一すべき。 |
| **UC-012** | **Low** | テストモック手法の不一致 | testify/mockベースと手動スタブが混在。プロジェクト全体で統一すべき。 |
| **UC-013** | **Medium** | `goalUsecase.Update` にトランザクション欠如 | `Create` はトランザクション内で実行しているが `Update` にはない。並行アクセス時のrace condition リスク。 |
| **UC-014** | **Low** | `context.Canceled` ログの一貫性欠如 | 一部のメソッドのみ `context.Canceled` でWarn/Error分岐。共通ヘルパーに抽出し全ユースケースで統一適用すべき。 |

---

### ハンドラー/ルーター層 (HDL)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **HDL-001** | **Medium** | IPスプーフィングリスク | `c.RealIP()` のIP取得方法が未設定。リバースプロキシ構成で `X-Forwarded-For` 偽装によりIPベースレートリミット回避可能。`router.go` で `e.IPExtractor` を適切に設定すべき。 |
| **HDL-002** | **Medium** | `displayid` パスパラメータの未検証 | GET/DELETE/Restore等のハンドラで `displayid` を検証せずユースケースに渡している。`UpdateSongRequest` では `validate:"required,len=16"` があるのに不整合。ヘルパー関数で統一検証すべき。 |
| **HDL-003** | **Medium** | `username` パスパラメータの未検証 | 全ハンドラで `username` パスパラメータが無検証。極端に長い文字列や特殊文字がそのままDB検索に到達。 |
| **HDL-004** | **Medium** | エラーハンドリングが粗い箇所の複数存在 | `DeleteSong`/`RestoreSong`, `APITokenHandler`, `Logout`, `UpdatePrivacy` でエラーを一律 `ErrInternalError` にしている。`FromUsecaseError()` を使って適切な HTTP ステータスに変換すべき。 |
| **HDL-005** | **Low** | ユーザープロファイルエラーハンドリングの重複 | `api_internal/user_handler`, `api_v1/user_handler` で類似のエラーハンドリングパターン。共通ヘルパーに抽出すべき。 |
| **HDL-009** | **Low** | `me_handler.go` で `c.Logger()` と `slog` が混在 | プロジェクト全体では `slog` だが `me_handler.go` だけ `c.Logger().Warnf` を使用。`user.ID` を `%s` フォーマットで出力するバグもあり。 |
| **HDL-010** | **Low** | `me_handler.go` の `knownFields` マップがハードコード | `PlayerDataPayload` 構造体のフィールド変更時に同期忘れリスク。`reflect` で自動生成し初期化時にキャッシュすべき。 |

---

### セキュリティ追加 (SEC)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **SEC-05** | **Medium** | ログイン失敗時のタイミング攻撃軽減の不在 | ユーザー非存在時はハッシュ比較を行わないためレスポンス時間に差が生じ、ユーザー列挙の手がかりになり得る。ダミーハッシュ比較を実行して計算時間を一定にすべき。 |

---

### アーキテクチャ追加 (ARCH)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **ARCH-002** | **Low** | `OfficialSongWithGenreDTO` にインフラ層の `db:` タグ | DTO層にDBタグ付き構造体があるのは不適切。`infra/models` に移動すべき。未使用の可能性もあり、その場合は削除。 |

---

### DTO設計 (DTO)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **DTO-001** | **Low** | `GoalRequest` の `AchievementParams`/`Attributes` が `map[string]any` | 型安全でなく、スキーマ検証なし。任意データをDBに保存可能。サイズ上限チェックも不在。 |

## 追加で重点的に確認したい事項
- **JWTの運用ポリシー**: `issuer`/`audience` の運用があるか（必要なら `ValidateToken` に追加）。
- **DB接続のTLS**: MySQL接続にTLSが必要な環境か（必要ならDSNで設定）。
- **APIドキュメント反映**: 入力制約やセキュリティ要件を `docs/API.md` に追記する必要性。

---

## まとめ
- 主要なリスクは **CSRF対策不足（SEC-01）** と **タイミング攻撃（SEC-05）** です。
- アーキテクチャ面では **Domain層のsqlx依存（QUAL-010）** がクリーンアーキテクチャ違反として要対応。
- ドメイン層では **`Notes.Scan` のバリデーションバイパス（DOM-008）** が不正データ流入のリスク。
- インフラ層では **`FromChartEntity`/`FromPlayerRecordEntity` のエラー無視（INFRA-009, INFRA-016）** がpanic発生のリスク。
- **コンストラクタのエラー無視（QUAL-006）** と **トランザクション未使用（UC-004, UC-013）** は整合性に関わる重要項目。
- 入力検証の統一方針（パスパラメータの検証）とエラーハンドリング（`FromUsecaseError()` への統一）は、バグ防止・運用事故防止に直結する。
- **コード重複**がユースケース層・ハンドラー層・インフラ層に散在しており、共通ヘルパーへの抽出で保守性を改善すべき。
- **命名規約**（Service/Usecase混在）と **ソートAPI** （sort vs slices）の統一が未完了。
