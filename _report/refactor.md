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
| **SEC-008** | **Medium** | Cookie Domain属性の未設定 | サブドメイン間のセッション共有が必要な場合に備え、Domain属性の設定可否を追加。 |

### パフォーマンス (PERF)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **PERF-001** | **Medium** | DBコネクションプール設定が未設定 | `SetMaxOpenConns` / `SetMaxIdleConns` / `SetConnMaxLifetime` を設定値として管理。 |
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
| **QUAL-007** | **Medium** | 論理削除の連動ロジック不足 | ユーザーの論理削除時に、DB制約（CASCADE）が効かない関連データ（将来的なフレンド等）を無効化する仕組みが未整備。 |
| **QUAL-009** | **Medium** | Usecase層でのインフラ層エラー直接参照 | `sql.ErrNoRows` をUsecase層で直接参照している。リポジトリ層でドメインエラーに変換すべき。 |
| **QUAL-010** | **Medium** | Domain層のExecutorインターフェースがsqlxに依存 | `internal/domain/repository/executor.go` で `*sqlx.Rows`, `*sqlx.Row` を直接参照。ドメイン層がインフラ実装に依存している。 |
| **QUAL-012** | **Low** | ハンドラーでのValidate呼び出し漏れ | `authRequest`, `changePasswordRequest` 等のリクエスト構造体に `validate` タグがなく、`c.Validate()` も呼ばれていない。 |
| **QUAL-014** | **High** | Usecase層がInfra層をimport | `chart_stats_usecase.go` が `internal/infra/masterdata` をimport。AGENTS.mdで厳禁とされているクリーンアーキテクチャ違反。 |
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

### PERF-001: DBコネクションプール設定が未設定
- **根拠**:
  - `sqlx.Open` 後に `SetMaxOpenConns` / `SetMaxIdleConns` / `SetConnMaxLifetime` が未設定（`internal/infra/db/connection.go`）。
- **影響範囲**:
  - 高負荷時にコネクション数が増えすぎ、DB側の上限到達で接続エラー。
  - 低負荷時でも接続が閉じず、無駄なリソース消費。
- **再現手順**:
  1. 多数の並行リクエストを送信。
  2. DBの接続数上限に到達し失敗が発生。
- **修正案**:
  - 設定値を `internal/config` に追加し、接続数/ライフタイムを制御。
  - 例: `db.SetMaxOpenConns(25)`, `db.SetMaxIdleConns(25)`, `db.SetConnMaxLifetime(5 * time.Minute)`。
- **追加で確認したい点**:
  - 本番DBの接続上限値と想定ピークトラフィック。

---

### QUAL-006: コンストラクタのエラー無視
- **根拠**:
  - `songRepository.toChartEntity` 内で `chartconstant.NewChartConstant(row.Const)` などのエラー戻り値を捨てている。
- **影響範囲**:
  - DBに不正な値が入っていた場合、異常な状態でドメイン層にデータが渡り、予期せぬ挙動や計算ミスを引き起こす。
- **修正案**:
  - エラーが発生した場合は上位に返し、データの整合性エラーとして適切にログ出力・処理する。

### QUAL-007: 論理削除の連動ロジック不足
- **根拠**:
  - `userRepository.SoftDelete` 等で `is_deleted` フラグを更新しているが、アプリケーションレベルでの関連データ（将来的なフレンド関係、一時的なバッチ処理用データなど）を無効化するフックや連動ロジックが欠如している。
- **影響範囲**:
  - ユーザーが論理削除されても、他ユーザーからフレンドとして見え続ける、あるいは集計処理に意図せず含まれるといった不整合が発生する。
- **修正案**:
  - ドメイン層に「ユーザー削除」のドメインイベントを導入するか、ユースケース層で連動して削除すべきエンティティを一括で処理するロジックを実装する。

---

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

### QUAL-014: Usecase層がInfra層をimport
- **根拠**:
  - `internal/usecase/chart_stats_usecase.go:13` で `internal/infra/masterdata` パッケージをimport。
  - プロダクションコードで `masterdata.StaticCache` を直接参照している。
  - AGENTS.mdでは「`internal/usecase` パッケージが `internal/infra` パッケージを import すること」は**厳禁**と明記されている。
- **影響範囲**:
  - Clean Architectureの依存方向に重大な違反。Usecase層がインフラ層の実装詳細に依存しており、テスタビリティが低下。
  - マスターデータの取得方法を変更する場合、Usecase層の修正が必要になる。
  - 実装とAGENTS.mdのルールが乖離しており、AIエージェントが矛盾した判断をする原因となる。
- **修正案**:
  - マスターデータへのアクセスをDomain層のインターフェース経由に変更。
  - `internal/domain/repository` に `MasterDataRepository` インターフェースを定義し、Usecase層はこれに依存する。
  - Infra層の実装（`masterdata.StaticCache`）はDI経由でUsecaseに注入する。
- **追加で確認したい点**:
  - `internal/domain/masterdata` パッケージが既に存在するため、このパッケージとの関係を整理する必要がある。

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

## 実装簡素化・ライブラリ活用提案 (LIB)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **LIB-002** | **Low** | 設定読み込みの自動化 | 手動での環境変数読み込み・型変換を廃止し、`kelseyhightower/envconfig` による構造体タグベースの宣言的な設定読み込みに移行する。 |
| **LIB-003** | **Low** | コレクション操作の効率化 | 冗長なループ処理（Map, Filter, Uniq 等）を `samber/lo` で置き換え、コードを簡潔にする。 |
| **LIB-004** | **Low** | ログファイルローテーション | 現状は起動毎に新ファイルを作成する形式だが、日付ベースのローテーションやサイズ制限がない。運用期間が長くなるとログファイルが肥大化する可能性がある。`lumberjack` などのライブラリ導入、または日付ベースのファイル切り替えロジックを実装する。 |
| **LIB-005** | **Low** | レスポンス圧縮ミドルウェアの導入検討 | 現状はリクエストのgzip解凍は実装されているが、レスポンスの圧縮は行われていない。大量のレコードを返すエンドポイントでは、gzip圧縮ミドルウェアの導入で帯域削減が期待できる。 |

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
- 主要なリスクは **CSRF対策不足** と **DBコネクションプール設定欠如**。
- アーキテクチャ面では **Usecase層からのsql.ErrNoRows参照** と **Domain層のsqlx依存** がクリーンアーキテクチャ違反として要対応。
- 入力検証の統一方針（`c.Validate()` の呼び出し漏れ解消）とAPI仕様の整合性は、バグ防止・運用事故防止に直結する。
- goroutineの終了処理は現状軽微だが、より堅牢なgraceful shutdownのために対応が望ましい。
