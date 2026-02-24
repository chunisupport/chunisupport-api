# リファクタリング指摘書 (Current Code Issues)

本ドキュメントは、コードベース評価に基づき修正が必要な項目を整理したものです。
`chunisupport-api` (Go 1.25, Echo, MySQL, ~50k users) の規模と特性、および Clean Architecture/DDD を考慮し、実効性の高い項目に絞り込んでいます。

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

### セキュリティ (SEC)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **SEC-01** | **High** | CSRF対策不足 | Double Submit Cookie または Synchronizer Token を導入。SameSite=Lax/Strict と Origin/Referer 検証を併用。 |
| **SEC-011** | **High** | パスワード複雑性要件の欠如 | 長さチェックのみ。`zxcvbn-go` 等の導入または正規表現による文字種チェックを追加。 |
| **SEC-03** | **Medium** | `#nosec` コメントの妥当性レビュー未実施 | `gosec` などで抑制箇所を洗い出し、根拠を明記。不必要な抑制は削除。 |
| **SEC-008** | **Low** | Cookie Domain属性の要件確認不足 | Domain属性は常時必須ではないため、まず「サブドメイン間でセッション共有が必要か」を要件として明確化し、必要時のみ設定を追加。 |

### パフォーマンス (PERF)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **PERF-003** | **Medium** | 巨大レスポンスの生成 | `GetUserProfileWithRecords` が全件返却。レスポンス簡素化またはページネーション導入。 |
| **PERF-004** | **Medium** | スコア差分計算時の全件スキャン | `player_records` を全件取得。`chart_id` 絞り込みや差分計算のオプション化で負荷削減。 |
| **PERF-006** | **Medium** | IN句クエリのリスト肥大化 | 大量IDをチャンク分割して複数回取得。後続の取得も同様に分割。 |

### 信頼性・運用 (OPS)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **OPS-001** | **Low** | リクエストIDの欠如 | ログにリクエストを一意に識別するID（X-Request-ID等）が付与されておらず、分散環境でのトレーサビリティが低い。 |
| **OPS-002** | **Low** | DBクエリタイムアウトの未設定 | リクエストContextはDB操作に渡されているが、明示的なクエリタイムアウト設定がない。 |

### API設計・入力検証 (API)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **API-001** | **Low** | 入力検証のDTO適用範囲の確認 | `CustomValidator`は実装済み。全DTOでの`validate`タグ適用状況を確認し、未対応のDTOを洗い出す。入力制限を`docs/API.md`に反映。 |

### 実装品質・保守性 (QUAL/GO/DB)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **QUAL-001** | **Low** | TODOコメントの残置 | 解消またはIssue化。現状2件残存（詳細は後述）。 |
| **QUAL-002** | **Medium** | セキュリティヘッダーの欠如 | Echoの `Secure` ミドルウェア導入でHSTS等を設定。 |
| **DB-003** | **Low** | 手動マッピングの冗長性 | `sqlx.StructScan` 等の活用で構造体タグベースに移行。 |
| **QUAL-006** | **Medium** | コンストラクタのエラー無視 | `toChartEntity` 等で値オブジェクトの生成エラーを無視している。不整合なエンティティが生成されるリスク。 |
| **QUAL-009** | **Medium** | Usecase層でのインフラ層エラー直接参照 | `sql.ErrNoRows` をUsecase層で直接参照している。リポジトリ層でドメインエラーに変換すべき。 |
| **QUAL-010** | **Medium** | Domain層のExecutorインターフェースがsqlxに依存 | `internal/domain/repository/executor.go` で `*sqlx.Rows`, `*sqlx.Row` を直接参照。ドメイン層がインフラ実装に依存している。 |
| **QUAL-012** | **Low** | ハンドラーでのValidate呼び出し漏れ | `authRequest`, `changePasswordRequest` 等のリクエスト構造体に `validate` タグがなく、`c.Validate()` も呼ばれていない。 |
| **QUAL-017** | **Low** | ARCHITECTURE.mdのディレクトリ表記不整合 | `domain/rating` と記載されているが、実際は `domain/service`。参照ドキュメントとの不整合。 |

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
  | `internal/app/apierror/codes.go` | 12-15, 39 | G101（ハードコードされたクレデンシャル疑い） | △（エラーコード定数であり実際のクレデンシャルではないが、コメントなし） |
  | `internal/config/config.go` | 77 | G304（ファイルパス挿入） | ○（「LogPaths.Echo comes from trusted configuration」とコメントあり） |
  | `internal/app/router.go` | 381 | G304（ファイルパス挿入） | ○（「comes from trusted configuration」とコメントあり） |
  | `internal/infra/logger/handler.go` | 59 | G304（ファイルパス挿入） | ○（「logDir comes from trusted configuration」とコメントあり） |
  | `internal/dto/worldsend_dto.go` | 36 | G115（整数オーバーフロー） | ○（「Score value is guaranteed to be within uint32 range by domain VO」とコメントあり） |
  | `internal/infra/models/player_record_model.go` | 49 | G115（整数オーバーフロー） | ○（同上） |
  | `internal/infra/models/player_worldsend_record_model.go` | 45 | G115（整数オーバーフロー） | ○（同上） |
  | `internal/usecase/player_data_usecase_impl.go` | 841 | G115（整数オーバーフロー） | △（コメントなし） |
- **影響範囲**:
  - 実際の脆弱性（パス・トラバーサル、整数オーバーフローなど）を見逃す可能性。
- **修正案**:
  - `internal/app/apierror/codes.go`：エラーコード定数であり実際のクレデンシャルではないことを明記（例: `// #nosec G101 -- これはエラーコード定数であり、実際のクレデンシャルではない`）
  - `internal/usecase/player_data_usecase_impl.go:841`：根拠コメントを追加
  - その他の箇所は適切な根拠コメントあり
- **追加で確認したい点**:
  - 利用中の静的解析ツールとCI連携の有無。

---

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
  | `internal/app/handler/compat/chunirec/chunirec_handler.go` | 126 | `// TODO: UserProfileWithRecordsDTOにUserIDフィールドを追加してリファクタリング` |
  | `internal/usecase/user_usecase_impl.go` | 45 | `// TODO: 最適化の余地あり - 現在はユーザー→プレイヤー→称号→レコードで4回クエリを発行している。` |
- **修正案**:
  - 解消またはIssue化。

---

### API-001: 入力検証のDTO適用範囲の確認
- **根拠**:
  - `CustomValidator` は `internal/app/router.go` で実装済み。一部DTOには `validate` タグが使用されているが、全DTOへの統一的な適用状況が未確認（`internal/usecase/auth_usecase.go`, `internal/usecase/player_data_usecase_impl.go`）。
- **影響範囲**:
  - `validate` タグが未適用のDTOでは、手動バリデーションに依存しており、バリデーションロジックが分散する可能性。
  - API仕様書との整合性が取れていない箇所がある可能性。
- **修正案**:
  - 全DTOで `validate` タグの適用状況を確認し、未対応の箇所を洗い出す。
  - 入力上限（最大長・最大件数）を仕様（`docs/API.md`）に反映し、実装と仕様の整合性を確保する。
- **追加で確認したい点**:
  - 現状のAPI仕様書に入力制約が明記されているか。

---

### QUAL-009: Usecase層でのインフラ層エラー直接参照
- **根拠**:
  - `internal/usecase/worldsend_usecase.go`, `internal/usecase/auth_usecase.go` 等で `database/sql` パッケージの `sql.ErrNoRows` を直接 `errors.Is()` で判定している。
  - 例: `if errors.Is(err, sql.ErrNoRows) { return repository.ErrSongNotFound }`
- **現状の該当箇所**:
  | ファイル | 該当箇所数 |
  |---|---|
  | `internal/usecase/auth_usecase.go` | 10箇所以上 |
  | `internal/usecase/user_usecase_impl.go` | 4箇所 |
  | `internal/usecase/worldsend_usecase.go` | 3箇所 |
  | `internal/usecase/api_token_usecase_impl.go` | 2箇所 |
  | `internal/infra/repository/worldsend_chart_repository_impl.go` | 3箇所（リポジトリ層で変換せず直接返している） |
- **影響範囲**:
  - Usecase層がインフラ層の実装詳細（SQLドライバーのエラー型）に依存しており、クリーンアーキテクチャの依存方向に違反。
  - リポジトリ実装を別のストレージ（NoSQL等）に変更した場合、Usecase層も修正が必要になる。
- **修正案**:
  - リポジトリ層で `sql.ErrNoRows` をドメイン層で定義されたエラー（例: `repository.ErrNotFound`）に変換して返す。
  - Usecase層では `sql.ErrNoRows` をインポートせず、ドメインエラーのみを扱う。

---

### QUAL-010: Domain層のExecutorインターフェースがsqlxに依存
- **根拠**:
  - `internal/domain/repository/executor.go` で `*sqlx.Rows`, `*sqlx.Row` を戻り値の型として使用している。
  - ドメイン層が `github.com/jmoiron/sqlx` をインポートしている。
- **影響範囲**:
  - Clean Architectureの原則に反し、ドメイン層がインフラ層の実装詳細に依存。
  - リポジトリのモックを作成する際にsqlxの型に依存することになり、テスタビリティが低下。
- **修正案**:
  - `Executor` インターフェースをインフラ層に移動するか、戻り値を抽象化してドメイン層からsqlx依存を排除。
  - ただし影響範囲が大きいため、現状の妥協として許容し、将来的なリファクタリング対象とする選択肢もある。

---

### QUAL-012: ハンドラーでのValidate呼び出し漏れ
- **根拠**:
  - `internal/app/handler/api_internal/auth_handler.go` の `authRequest`, `changePasswordRequest`, `recoveryCodeRecoverRequest` 等のリクエスト構造体に `validate` タグがなく、`c.Validate()` も呼ばれていない。
  - 他のハンドラー（`song_handler.go`, `player_handler.go`）では `c.Validate()` が呼ばれている。
- **影響範囲**:
  - 入力検証のアプローチが統一されておらず、バリデーション漏れのリスクがある。
  - 例えば `authRequest` ではユーザー名やパスワードの長さチェックをUsecase層で個別に行っているが、これはHandler/DTO層で統一的に行うべき。
- **修正案**:
  - 全リクエスト構造体に適切な `validate` タグを追加。
  - Bindの直後に `c.Validate()` を呼ぶパターンを全ハンドラーで統一。

---

### QUAL-017: ARCHITECTURE.mdのディレクトリ表記不整合
- **根拠**:
  - `ARCHITECTURE.md:21` に `domain/rating` と記載されているが、実際のディレクトリ構造は `internal/domain/service`。
  - レーティング計算サービスは `internal/domain/service/rating_service.go` に配置されている。
- **影響範囲**:
  - AGENTS.mdから参照される関連ドキュメントの不整合は、AIエージェントが正しいディレクトリを判断できない原因となる。
  - 新規参加者がドキュメントを読んだ際に混乱する。
- **修正案**:
  - `ARCHITECTURE.md` の該当箇所を `domain/service` に修正。
  - プロジェクト全体のドキュメントで他にも古い記載がないか確認。

---

### INFRA-001: `playerRepository.Create` のカラム名誤りによる実行時エラー
- **根拠**:
  - `internal/infra/repository/player_repository_impl.go` の `Create` メソッドが `INSERT INTO players (name) VALUES (?)` を実行するが、DBスキーマ上のカラム名は `player_name`。さらに `user_id`（NOT NULL）と `player_level`（NOT NULL, CHECK >= 1）が欠落。
  - 同リポジトリの `insert` メソッドは正しく `player_name` を使用しており、`Create` と `insert` で不整合。
- **影響範囲**:
  - `CreatePlayer` ハンドラ → usecase → `playerRepo.Create` の呼び出しチェーンで、プレイヤー作成APIが実行時に必ずSQLエラーで失敗する。
- **修正案**:
  - `Create` メソッドを廃止し、`Save` メソッドに統一する。

---

### INFRA-003: `FindByUserID` の文字列比較によるエラー判定
- **根拠**:
  - `internal/infra/repository/player_repository_impl.go` で `err.Error() == "sql: no rows in result set"` という文字列比較で `sql.ErrNoRows` を判定している。
- **影響範囲**:
  - Goのバージョンアップやエラーラッピングでメッセージが変更された場合、エラー判定が壊れる。
- **修正案**:
  - `errors.Is(err, sql.ErrNoRows)` に変更する。

---

### UC-001: AuthUsecaseの責務過多（God Interface）
- **根拠**:
  - `internal/usecase/auth_usecase.go` の `AuthUsecase` インターフェースが10個のメソッドを持ち、認証（Register/Login/Logout/Authenticate）、ユーザー情報取得（GetUser）、プロファイル管理（UpdatePrivacy/ChangePassword/DeleteUser）、リカバリー（IssueRecoveryCodes/RecoverWithRecoveryCode）が混在。
- **影響範囲**:
  - テスト時に巨大なモックが必要になり、テスタビリティが低下。単一責任原則に違反。
- **修正案**:
  - 認証関連のみ `AuthUsecase` に残し、プロファイル管理は `UserUsecase` に移管、リカバリーは `RecoveryUsecase` として分離。
  - コンストラクタ名も `NewAuthService` → `NewAuthUsecase` に統一。

---

### UC-004: Register でのトランザクション未使用
- **根拠**:
  - `internal/usecase/auth_usecase.go` の `Register` でユーザー作成（`Create`）とセッション作成（`createSessionAndToken`）が非トランザクションで連続実行。
- **影響範囲**:
  - ユーザー作成成功後にセッション作成が失敗すると、ログインできないユーザーが作成される。
- **修正案**:
  - `Register` 全体を `s.tm.Transactional` で囲む。

---

### UC-005: `convertUsernameError` の文字列比較によるエラー変換
- **根拠**:
  - `internal/usecase/auth_usecase.go` の `convertUsernameError` がVOのエラーメッセージ文字列と直接比較してusecase層のエラーに変換。
  - 例: `errMsg == "username cannot be empty"` → `return ErrUsernameEmpty`
- **影響範囲**:
  - VO側のメッセージが変更されると変換が壊れ、適切なusecase層エラーが返されなくなる。
- **修正案**:
  - `username` パッケージにセンチネルエラー（`var ErrEmpty = errors.New(...)` 等）を定義し、`errors.Is` で判定する。

---

### DOM-007/008: 値オブジェクトの `Scan`/`UnmarshalJSON` がバリデーションをバイパス
- **根拠**:
  - `internal/domain/vo/chartconstant/chartconstant.go` の `Scan` と `UnmarshalJSON` が `*c = ChartConstant(f)` で直接設定し、`NewChartConstant` の「0以上」バリデーションをスキップ。
  - `internal/domain/vo/notes/notes.go` の `Scan` が `Notes(v)` で直接キャストし、`NewNotes` のバリデーションをスキップ。
  - 一方、`internal/domain/vo/score/score.go` の `Scan` は負値チェック・最大値チェックを適切に実施しており、模範的な実装。
- **影響範囲**:
  - DBやJSON入力に不正な値（負値等）が存在した場合、バリデーションなしで不正なVOが生成される。
- **修正案**:
  - `Scan`/`UnmarshalJSON` 内で `NewChartConstant`/`NewNotes` を経由してバリデーションを実行する。`score.Score` のパターンに統一。

---

### DOM-015: `username.validateUserName` で毎回正規表現をコンパイル
- **根拠**:
  - `internal/domain/vo/username/username.go` で `regexp.MatchString("^[a-z0-9]+$", value)` を使用。これは呼び出しのたびに正規表現をコンパイルする。
  - 同パッケージの `internal/domain/vo/displayid/displayid.go` では `regexp.MustCompile` でプリコンパイル済みの正規表現を使用しており、パターンが不一致。
- **影響範囲**:
  - ユーザー登録・ログイン時に毎回コンパイルコストが発生。高負荷時にパフォーマンス影響。
- **修正案**:
  - パッケージレベルで `var usernamePattern = regexp.MustCompile(...)` として定義し、`usernamePattern.MatchString(value)` を使用する。

---

### HDL-004: エラーハンドリングが粗い箇所の複数存在
- **根拠**:
  - 以下のハンドラでユースケースエラーを一律 `ErrInternalError` にしているが、`FromUsecaseError()` を使って適切なHTTPステータスに変換すべき。
  - 該当箇所:
    - `api_internal/song_handler.go`: `DeleteSong`, `RestoreSong`
    - `api_internal/api_token_handler.go`: `Generate`, `Delete`
    - `api_internal/auth_handler.go`: `DeleteAccount`, `Logout`, `UpdatePrivacy`
  - 他のハンドラ（WorldsendHandler等）は正しく `FromUsecaseError()` を使用しており不整合。
- **影響範囲**:
  - 404系エラー（楽曲/トークン/ユーザー未発見）が500で返されてしまい、クライアント側の適切なエラーハンドリングが困難。
- **修正案**:
  - 全箇所で `apierror.FromUsecaseError(err)` を使うように統一。

---

### SEC-05: ログイン失敗時のタイミング攻撃リスク
- **根拠**:
  - `internal/usecase/auth_usecase.go` の `Login` メソッドで、ユーザーが存在しない場合はハッシュ比較を行わずに即座にエラーを返す。パスワード不一致の場合はArgon2idハッシュ比較が実行されるため、レスポンス時間に有意な差が生じる。
- **影響範囲**:
  - 攻撃者がレスポンス時間の差からユーザー名の存在有無を推定できる（ユーザー列挙攻撃）。
- **修正案**:
  - ユーザーが存在しない場合にもダミーのArgon2idハッシュ比較を実行し、計算時間を一定にする。

---

## 実装簡素化・ライブラリ活用提案 (LIB)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **LIB-002** | **Low** | 設定読み込みの自動化 | 手動での環境変数読み込み・型変換を廃止し、`kelseyhightower/envconfig` による構造体タグベースの宣言的な設定読み込みに移行する。 |
| **LIB-003** | **Low** | コレクション操作の効率化 | 冗長なループ処理（Map, Filter, Uniq 等）を `samber/lo` で置き換え、コードを簡潔にする。 |
| **LIB-004** | **Low** | ログファイルローテーション | 現状は起動毎に新ファイルを作成する形式だが、日付ベースのローテーションやサイズ制限がない。運用期間が長くなるとログファイルが肥大化する可能性がある。`lumberjack` などのライブラリ導入、または日付ベースのファイル切り替えロジックを実装する。 |
| **LIB-005** | **Low** | レスポンス圧縮ミドルウェアの導入検討 | 現状はリクエストのgzip解凍は実装されているが、レスポンスの圧縮は行われていない。大量のレコードを返すエンドポイントでは、gzip圧縮ミドルウェアの導入で帯域削減が期待できる。 |

---

### ドメイン層設計 (DOM)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **DOM-001** | **Medium** | VOが `database/sql/driver` に依存 | 全値オブジェクト（`chartconstant`, `notes`, `passwordhash`, `playername`, `score`, `username`）が `driver.Valuer`/`sql.Scanner` を実装しており、DB永続化というインフラ関心事がドメイン層に混入。`infra/models` 層でアダプタを用意し、VOからDB依存を排除する。 |
| **DOM-003** | **High** | `Player` エンティティが貧血症モデル | メソッドが一切なく、`PlayerRepository.UpdateCalculatedRatings` という部分更新メソッドが存在する。AGENTS.mdで禁止されているパターン。`player.UpdateRatings(...)` + `repo.Save()` パターンに移行すべき。 |
| **DOM-004** | **Medium** | `Song` エンティティが貧血症モデル | メソッドがなく、`SongRepository.DeleteSong`/`RestoreSong` がDisplayID指定の直接操作。`song.Delete()`, `song.Restore()`, `song.IsActive()` メソッドを追加すべき。また「`Charts` はnil禁止」という不変条件を強制するコンストラクタが存在しない。 |
| **DOM-005** | **Low** | `Session` エンティティにメソッドなし | `IsExpired()` メソッドがあるべき。セッション有効期限判定ロジックがドメイン外に流出している。 |
| **DOM-006** | **Medium** | `Goal` エンティティが貧血症モデル＋`[]byte`フィールド | `AchievementParams []byte` と `Attributes []byte` はJSONバイト列の生保持であり、インフラ層の都合がドメイン層に漏洩している。適切な構造体やマップに変換すべき。 |
| **DOM-007** | **Medium** | `ChartConstant.Scan`/`UnmarshalJSON` がバリデーションをバイパス | コンストラクタは「0以上」の検証を行うが、`Scan` と `UnmarshalJSON` は直接値を設定。負値がDB/JSONから入力された場合に不正なVOが生成される。`score.Score` の `Scan` 実装を模範にすべき。 |
| **DOM-008** | **Medium** | `Notes.Scan` がバリデーションをバイパス | `Notes(v)` で直接キャストしており、`NewNotes` のバリデーション（0以上）を経由しない。 |
| **DOM-011** | **Medium** | 理論値スコア定数の二重定義 | `internal/domain/service/info.go` の `theoreticalScore uint32 = 1010000` と `internal/info/info.go` の `TheoreticalScore = 1010000` が重複。1箇所に集約すべき。 |
| **DOM-012** | **Low** | `WorldsendSongWithChart` と `WorldsendSongChartPair` の重複 | `repository` 層と `service` 層にフィールド同一の重複構造体。entity層に統一構造体を定義すべき。 |
| **DOM-013** | **Low** | エラーメッセージの日英混在 | 値オブジェクトは英語、エンティティバリデーションは日本語。同一パッケージ内でも混在あり。方針を統一すべき。 |
| **DOM-014** | **Medium** | `PlayerDataChart.Const` がVO未使用 | `float64` で定義されており、通常の `Chart` が使う `chartconstant.ChartConstant` と不整合。VOによるバリデーションが適用されない。 |
| **DOM-015** | **Medium** | `username.validateUserName` で毎回正規表現をコンパイル | `regexp.MatchString` は呼び出しのたびにコンパイル。`displayid.go` では `regexp.MustCompile` でプリコンパイル済みであり一貫性がない。 |
| **DOM-016** | **Low** | `record_completion_service.go` が `sort.Slice` 使用 | `rating_service.go` は `slices.SortFunc` 使用。Go 1.25で推奨される `slices` パッケージに統一すべき。 |
| **DOM-017** | **Low** | `PlayerHonor` がrepository層に定義 | ドメイン概念だが `repository` パッケージ内に定義。`entity` パッケージに移動すべき。 |
| **DOM-018** | **Medium** | `repository.errors.go` のエラー定義不足 | `ErrSongNotFound` のみで `ErrUserNotFound` 等はusecase層に定義。リポジトリが適切なドメインエラーを返せず、QUAL-009の根本原因となっている。 |
| **DOM-019** | **Low** | `Player.Users` フィールドがDDD集約境界を侵害 | `User` への直接参照。DDDでは集約間はID参照が原則。また命名が `Users`（複数形）だが実態は単一。 |
| **DOM-021** | **Low** | Deprecated関数が残存 | `rating_service.go` の `CalcBestAverageRating`, `CalcNewAverageRating`, `CalcPlayerRating`。テストでまだ使用中。移行完了後に削除すべき。 |
| **DOM-010** | **Low** | `WorldsendChart.Validate` が構築済みVOを冗長に再検証 | `notes.NewNotes(int(*w.Notes))` で既に構築済みの `Notes` VOを再バリデーション。VOの不変性を信頼する設計にすべき。 |

---

### インフラ層 (INFRA)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **INFRA-001** | **Critical** | `playerRepository.Create` のカラム名誤り | `INSERT INTO players (name)` だがDBスキーマでは `player_name`。さらに `user_id`/`player_level` 等の必須カラムが欠落。プレイヤー作成APIが実行時に必ず失敗する。`Create` を廃止し `Save` に統一すべき。 |
| **INFRA-002** | **Low** | `validation.go` のテーブル名組み立ての安全性改善余地 | 現状は内部固定値のみで直ちに脆弱性とは言えないが、将来の誤用防止のためテーブル名をホワイトリスト化し、任意入力を受け付けないAPIに制限すべき。 |
| **INFRA-003** | **High** | `FindByUserID` の文字列比較によるエラー判定 | `err.Error() == "sql: no rows in result set"` は脆弱。`errors.Is(err, sql.ErrNoRows)` に変更すべき。 |
| **INFRA-004** | **Medium** | WORLD'S END楽曲 `UpdateSongs` のN+1問題 | 楽曲と譜面を個別ループでUPDATE。通常楽曲はCASE式一括更新を実装済みであり不整合。バルク更新パターンに統一すべき。 |
| **INFRA-005** | **Medium** | `validation.go` の全関数でContext未伝播 | `context.Context` を引数に取らず、`db.Get` を使用（`GetContext` ではない）。起動時のキャンセル不能の原因。 |
| **INFRA-007** | **Medium** | `FindAllWithPlayer` と `FindAllWithPlayerForAdmin` のコード重複 | クエリ構築・LIKE検索・rows反復がほぼ同一。共通ヘルパーに抽出すべき。 |
| **INFRA-008** | **Medium** | `song_repository_impl.go` の `DeleteSong`/`RestoreSong` で `RowsAffected` 未確認 | 対象不存在でもエラーにならない。WE版では `RowsAffected` チェック済みであり不整合。 |
| **INFRA-009** | **Medium** | `FromChartEntity` の脆弱な定数変換処理 | `Value()` → 型アサーション(string) → `ParseFloat` の多段変換で失敗時に0.0フォールバック。`ChartConstant` に `Float64()` アクセサを追加すべき。 |
| **INFRA-010** | **Medium** | `BulkAssignHonors` にチャンクサイズ制限なし | 全件を1つのINSERTで発行。他のバルク処理は `info.BulkInsertChunkSize` で分割済み。 |
| **INFRA-011** | **Medium** | `resolveExecutor` の暗黙nil フォールバック | exec が nil の場合に `r.db` へフォールバック。トランザクション保証が暗黙に破壊されるリスク。他リポジトリはexec必須。 |
| **INFRA-012** | **Low** | `Cache.GetClassEmblemNameByID` 等のO(n)線形探索 | 他のマスタは `*NamesByID` マップでO(1)ルックアップ済み。パターン統一すべき。 |
| **INFRA-014** | **Low** | LIKE句で `ESCAPE` 節が未指定 | MySQLデフォルトで動作するが、`sql_mode` 変更で壊れる可能性。`ESCAPE '\\'` を明示すべき。 |
| **INFRA-016** | **Low** | `FromPlayerRecordEntity`/`FromPlayerWorldsendRecordEntity` のエラー無視 | `e.Score.Value()` のエラーを `_` で無視、`nil` に対する型アサーションでpanic発生の可能性。 |

---

### ユースケース層設計 (UC)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **UC-001** | **High** | `AuthUsecase` の責務過多（God Interface） | 10個のメソッドに認証・ユーザー管理・リカバリーが混在しSRP違反。認証（Register/Login/Logout/Authenticate）のみ残し、プロファイル管理は `UserUsecase` に、リカバリーは `RecoveryUsecase` に分離すべき。 |
| **UC-002** | **Medium** | `DeleteUser` の重複定義 | `AuthUsecase.DeleteUser(userID)` と `UserUsecase.DeleteUser(requester, username)` が異なるセマンティクスで同名。auth版には削除済みチェックも欠如。命名統一と重複排除が必要。 |
| **UC-003** | **Medium** | Setter Injection アンチパターン | `userUsecase.SetWorldsendRecordRepository` が構築後にリポジトリ注入。temporal couplingを導入。コンストラクタで全依存を渡すべき。 |
| **UC-004** | **Medium** | `Register` でトランザクション未使用 | ユーザー作成とセッション作成が非トランザクション。ユーザー作成成功・セッション作成失敗でログイン不能ユーザーが生成されるリスク。 |
| **UC-005** | **Medium** | `convertUsernameError` の文字列比較によるエラー変換 | VOのエラーメッセージ文字列と直接比較。メッセージ変更時に検知不能。VOにセンチネルエラーを定義し `errors.Is` で判定すべき。 |
| **UC-006** | **Low** | パスワードバリデーションロジックの3箇所重複 | `Register`, `ChangePassword`, `RecoverWithRecoveryCode` で長さチェックが重複。`validatePassword` ヘルパーに抽出すべき。 |
| **UC-007** | **Low** | 削除済み楽曲の権限チェックパターンの4箇所重複 | `song_usecase_impl`, `worldsend_usecase`, `chart_stats_usecase` で同一パターン。共通ヘルパーに抽出すべき。 |
| **UC-008** | **Medium** | `applyScores` のGod Function（約200行） | 通常譜面ループとWE譜面ループで解決ロジックがほぼ同一のまま2回繰り返し。共通関数に抽出すべき。 |
| **UC-009** | **Low** | `sort` パッケージ使用（Go 1.25では `slices` 推奨） | `player_data_usecase_impl.go` と `chart_stats_usecase.go` で `sort.Strings`/`sort.Slice` 使用。`slices.Sort`/`slices.SortFunc` に統一すべき。 |
| **UC-010** | **Low** | `time.LoadLocation` の繰り返し呼び出し | `player_data_usecase_impl.go` で都度 `time.LoadLocation("Asia/Tokyo")` を実行。ホットパス化する場合はパッケージレベルでキャッシュを検討。 |
| **UC-011** | **Medium** | コンストラクタ名のService/Usecase混在 | `NewAuthService`, `NewUserService` 等の"Service"接尾辞と `NewWorldsendUsecase`, `NewGoalUsecase` 等の"Usecase"接尾辞が混在。AGENTS.mdでは `Usecase` を推奨。統一すべき。 |
| **UC-012** | **Low** | テストモック手法の不一致 | testify/mockベースと手動スタブが混在。プロジェクト全体で統一すべき。 |
| **UC-013** | **Medium** | `goalUsecase.Update` にトランザクション欠如 | `Create` はトランザクション内で実行しているが `Update` にはない。並行アクセス時のrace condition リスク。 |
| **UC-014** | **Low** | `context.Canceled` ログの一貫性欠如 | 一部のメソッドのみ `context.Canceled` でWarn/Error分岐。共通ヘルパーに抽出し全ユースケースで統一適用すべき。 |
| **UC-015** | **Medium** | `NewAuthService` のパラメータ数過多（11個） | 構成情報（jwtSecret, pepper等）とリポジトリが混在。`AuthConfig` 構造体にまとめて削減すべき。 |
| **UC-016** | **Medium** | テストコードからinfra層への直接依存 | `auth_usecase_test.go` が `internal/infra/masterdata` をimport、`worldsend_usecase_test.go` が `sqlx` をimport。テスト用スタブを作成すべき。 |
| **UC-017** | **Low** | `Register` 内のバリデーション順序が非効率 | ユーザー名VOバリデーションがDB問合せより後に実行される。不正ユーザー名でもDBアクセスが発生。 |

---

### ハンドラー/ルーター層 (HDL)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **HDL-001** | **Medium** | IPスプーフィングリスク | `c.RealIP()` のIP取得方法が未設定。リバースプロキシ構成で `X-Forwarded-For` 偽装によりIPベースレートリミット回避可能。`router.go` で `e.IPExtractor` を適切に設定すべき。 |
| **HDL-002** | **Medium** | `displayid` パスパラメータの未検証 | GET/DELETE/Restore等のハンドラで `displayid` を検証せずユースケースに渡している。`UpdateSongRequest` では `validate:"required,len=16"` があるのに不整合。ヘルパー関数で統一検証すべき。 |
| **HDL-003** | **Medium** | `username` パスパラメータの未検証 | 全ハンドラで `username` パスパラメータが無検証。極端に長い文字列や特殊文字がそのままDB検索に到達。 |
| **HDL-004** | **Medium** | エラーハンドリングが粗い箇所の複数存在 | `DeleteSong`/`RestoreSong`, `APITokenHandler`, `DeleteAccount`, `Logout`, `UpdatePrivacy` でエラーを一律 `ErrInternalError` にしている。`FromUsecaseError()` を使って適切な HTTP ステータスに変換すべき。 |
| **HDL-005** | **Low** | ユーザープロファイルエラーハンドリングの3箇所重複 | `api_internal/user_handler`, `api_v1/user_handler`, `chunirec_handler` で同一パターン。共通ヘルパーに抽出すべき。 |
| **HDL-006** | **Low** | `OrderedChartsMap.MarshalJSON` のロジック重複＋非効率 | `api_internal/song_dto.go` と `api_v1/song_dto.go` でほぼ同一。文字列連結でJSON構築も非効率。共通ヘルパーに抽出し `strings.Builder` を使用すべき。 |
| **HDL-007** | **Low** | `handler/player_handler.go` がデッドコード | `api_internal/player_handler.go` と完全に同一で、ルーターからの参照なし。削除すべき。 |
| **HDL-008** | **Low** | `sanitizeLogValue` 関数の重複 | `middleware/error_handler.go` と `chunirec/error_handler.go` に同一定義。共通ユーティリティに抽出すべき。 |
| **HDL-009** | **Low** | `me_handler.go` で `c.Logger()` と `slog` が混在 | プロジェクト全体では `slog` だが `me_handler.go` だけ `c.Logger().Warnf` を使用。`user.ID` を `%s` フォーマットで出力するバグもあり。 |
| **HDL-010** | **Low** | `me_handler.go` の `knownFields` マップがハードコード | `PlayerDataPayload` 構造体のフィールド変更時に同期忘れリスク。`reflect` で自動生成し初期化時にキャッシュすべき。 |

---

### セキュリティ追加 (SEC)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **SEC-04** | **Medium** | JWT Secret の最小長チェックなし | 空文字チェックのみ。短い秘密鍵（例: `abc`）でもHS256で受理され、総当たり攻撃に脆弱。最小32バイト以上を要求すべき。 |
| **SEC-05** | **Medium** | ログイン失敗時のタイミング攻撃軽減の不在 | ユーザー非存在時はハッシュ比較を行わないためレスポンス時間に差が生じ、ユーザー列挙の手がかりになり得る。ダミーハッシュ比較を実行して計算時間を一定にすべき。 |
| **SEC-06** | **Low** | `PW_PEPPER` の最小長チェックなし | 空文字チェックのみ。短すぎるペッパーはハッシュ強化に寄与しない。 |

---

### 設定・運用 (CFG)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **CFG-001** | **Medium** | CORS `AllowOrigins: ["*"]` と `AllowCredentials: true` の組み合わせチェックなし | 仕様的に無効な組み合わせだが設定バリデーションで検出されない。`config.go` でチェックを追加すべき。 |
| **CFG-002** | **Low** | `AppPort` の範囲チェックなし | 0や負値、65535超でもエラーにならない。 |
| **CFG-003** | **Low** | `JWTExpirationHour`/`SessionExpirationHour` の検証なし | 0や負値でもエラーにならず、JWTが即時無効になる。正の値であることを検証すべき。 |
| **CFG-004** | **Low** | `main.go` の `ValidateRequiredData` がコメントアウト | 必須データの存在チェックが無効化されており、空DBでもアプリ起動。意図的なら理由をコメントに残すべき。 |
| **CFG-005** | **Low** | `Taskfile.yml` にDBパスワードがハードコード | 環境変数を使用すべき。 |

---

### アーキテクチャ追加 (ARCH)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **ARCH-001** | **Medium** | `ChunirecHandler` が `repository.UserRepository` と `*sqlx.DB` を直接保持 | Handler層がUsecase層を飛ばしてリポジトリ・DB接続を直接参照。Clean Architecture違反。DTO/UsecaseにUserID返却機能を追加しHandler依存を排除すべき。 |
| **ARCH-002** | **Low** | `OfficialSongWithGenreDTO` にインフラ層の `db:` タグ | DTO層にDBタグ付き構造体があるのは不適切。`infra/models` に移動すべき。未使用の可能性もあり、その場合は削除。 |

---

### DTO設計 (DTO)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **DTO-001** | **Low** | `GoalRequest` の `AchievementParams`/`Attributes` が `map[string]any` | 型安全でなく、スキーマ検証なし。任意データをDBに保存可能。サイズ上限チェックも不在。 |

---

## 将来のリファクタリング計画 (FUTURE)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **FUTURE-001** | **Low** | Primitive Obsession 対応 | レビューで指摘された「プリミティブ型への執着」を将来的に解消する。現時点では別テーマとして扱い、段階的に進める。<br><br>**対象と狙い**: `PlayerDataMasters` 内の `map[string]Item` など、ドメイン概念のキーを `string` で扱っている箇所。クリアランプ名や難易度などはドメイン固有の概念であり、Value Object 化で型安全性を高める。<br><br>**方針**:<br>1. `ClearLampName` などの Value Object を Domain 層に定義し、コンストラクタで正規化・検証を行う。<br>2. `map[string]Item` を `map[ClearLampName]Item` のように置換し、呼び出し側も Value Object を使うように統一する。<br>3. 正規化処理（大文字化など）を Value Object に集約し、重複する `strings.ToUpper()`/`strings.ToLower()` を排除する。<br>4. 影響範囲が大きいため、マスターデータの一部（例: クリアランプ/難易度）から段階的に移行する。<br>5. 既存テストの修正で整合を取ることを基本とし、新規テスト追加は重複が出ない範囲に限定する。 |

## 追加で重点的に確認したい事項
- **JWTの運用ポリシー**: `issuer`/`audience` の運用があるか（必要なら `ValidateToken` に追加）。
- **CORS設定値**: `AllowOrigins` と `AllowCredentials` の組み合わせが安全か（`*` の禁止など）。
- **DB接続のTLS**: MySQL接続にTLSが必要な環境か（必要ならDSNで設定）。
- **APIドキュメント反映**: 入力制約やセキュリティ要件を `docs/API.md` に追記する必要性。

---

## まとめ
- 主要なリスクは **CSRF対策不足** と **INFRA-001（`playerRepository.Create`の実行時エラー）**。
- **INFRA-001（`playerRepository.Create`のカラム名誤り）** は即時修正が必要な実行時エラー。
- アーキテクチャ面では **Usecase層からのsql.ErrNoRows参照** と **Domain層のsqlx依存** がクリーンアーキテクチャ違反として要対応。
- ドメイン層では **貧血症モデル（Player, Song, Goal等）** がDDD原則に反しており、Rich Model化が必要。
- **AuthUsecaseの責務過多（10メソッド）** はSRP違反であり、分割が望ましい。
- **値オブジェクトのScan/UnmarshalJSONがバリデーションをバイパス** する問題は、不正データがドメイン層に流入するリスク。
- 入力検証の統一方針（`c.Validate()` の呼び出し漏れ解消・パスパラメータの検証）とAPI仕様の整合性は、バグ防止・運用事故防止に直結する。
- **コード重複**がハンドラー層・ユースケース層・インフラ層の各所に散在しており、共通ヘルパーへの抽出で保守性を改善すべき。
- goroutineの終了処理は現状軽微だが、より堅牢なgraceful shutdownのために対応が望ましい。
