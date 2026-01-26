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
- ドキュメント: `docs/API.md`, `docs/public_api.md`
- アーキテクチャ: `ARCHITECTURE.md`

## 解析手順
1. 全体構造の把握（`internal/` + `ARCHITECTURE.md`）
2. 認証・認可フロー追跡（`internal/auth`, `internal/app/middleware`, `internal/app/handler`）
3. 入力点ごとの検証（HTTP境界、DTO、バッチ）
4. データアクセス層（SQL/トランザクション/Context）
5. 並行処理・キャッシュ・外部I/O（timeout/retry）
6. ログ・panic・メトリクスの情報漏えい/観測性
7. 依存関係・設定の危険値や既知脆弱性

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
| **OPS-002** | **Medium** | セッション期限切れの定期クリーンアップがない | `expires_at < NOW()` の定期削除ジョブ、またはメンテナンスAPIを導入。 |

### API設計・入力検証 (API)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **API-001** | **Low** | 入力検証のDTO適用範囲の確認 | `CustomValidator`は実装済み。全DTOでの`validate`タグ適用状況を確認し、未対応のDTOを洗い出す。入力制限を`docs/API.md`に反映。 |

### 実装品質・保守性 (QUAL/GO/DB)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **QUAL-002** | **Medium** | セキュリティヘッダーの欠如 | Echoの `Secure` ミドルウェア導入でHSTS等を設定。 |
| **DB-003** | **Low** | 手動マッピングの冗長性 | `sqlx.StructScan` 等の活用で構造体タグベースに移行。 |
| **QUAL-001** | **Low** | TODOコメントの残置 | 解消またはIssue化。 |
| **QUAL-004** | **Medium** | レイヤー間の依存性違反 (UC -> Infra) | `AuthUsecase` がインフラ層の `masterdata` パッケージを直接インポートしている。DIPを適用。 |
| **QUAL-005** | **Medium** | ドメイン集約の配置不備 | `SongWithCharts` が `repository` パッケージに定義されている。本来は `entity` パッケージに配置すべき集約の概念。 |
| **QUAL-006** | **Medium** | コンストラクタのエラー無視 | `toChartEntity` 等で値オブジェクトの生成エラーを無視している。不整合なエンティティが生成されるリスク。 |
| **QUAL-007** | **Medium** | 論理削除の連動ロジック不足 | ユーザーの論理削除時に、DB制約（CASCADE）が効かない関連データ（将来的なフレンド等）を無効化する仕組みが未整備。 |
| **QUAL-008** | **Low** | 自明なコードに対する冗長なコメント | `toSongEntity` 等、コードから明らかな変換処理を日本語で説明しているコメントが散見される。 |

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
  - 複数の `#nosec` コメントが存在し（例: `internal/config/config.go:77`, `internal/usecase/player_data_usecase_impl.go:1016`）、抑制の妥当性が未レビュー。
- **影響範囲**:
  - 実際の脆弱性（パス・トラバーサル、整数オーバーフローなど）を見逃す可能性。
- **再現手順**:
  1. `gosec` などで静的解析を実行し、`#nosec` が付いた箇所を確認。
  2. 抑制が不要な場合でも警告が隠れる。
- **修正案**:
  - 静的解析で抑制箇所を洗い出し、正当性の根拠を明記。
  - 不要な抑制は削除し、許容される場合のみ抑制を残す。
- **追加で確認したい点**:
  - 利用中の静的解析ツールとCI連携の有無。

---

### OPS-002: セッション期限切れの定期クリーンアップがない
- **根拠**:
  - セッション削除は `Authenticate` での期限切れ検知やユーザー操作時のみ（`internal/usecase/auth_usecase.go`, `internal/infra/repository/session_repository_impl.go`）。
- **影響範囲**:
  - セッションテーブルの肥大化、インデックスサイズ増大による性能低下。
- **再現手順**:
  1. 大量のログインを行い、期限切れ後も一切アクセスしない。
  2. セッション行が削除されないまま残存。
- **修正案**:
  - バッチ（cron）で `expires_at < NOW()` を定期削除。
  - もしくは `DELETE ... WHERE expires_at < NOW()` のメンテナンスAPI。
- **追加で確認したい点**:
  - 運用環境でのジョブ運用可否（バッチ基盤の有無）。

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

### QUAL-004: レイヤー間の依存性違反 (UC -> Infra)
- **根拠**:
  - `AuthUsecase` (internal/usecase/auth_usecase.go) が `internal/infra/masterdata` をインポートしている。
- **影響範囲**:
  - クリーンアーキテクチャの依存方向（外から内）に反しており、ユニットテストが困難になる。
- **修正案**:
  - `MasterCache` をドメインサービスまたはリポジトリとして抽象化し、インターフェースを介して依存させる。

### QUAL-005: ドメイン集約の配置不備
- **根拠**:
  - `SongWithCharts` 構造体が `internal/domain/repository/song_repository.go` に定義されている。
- **影響範囲**:
  - 楽曲とその譜面リストを扱うビジネスルール（集約）がリポジトリ層に漏れ出しており、ドメイン知識が分散する原因となる。
- **修正案**:
  - `SongWithCharts` 構造体を廃止し、既存の `entity.Song` に `Charts []*entity.Chart` フィールドを追加して集約として完成させます。リポジトリは `*entity.Song` を直接返すように変更します。

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

### QUAL-008: 自明なコードに対する冗長なコメント
- **根拠**:
  - `toSongEntity` や `toChartEntity` などのメソッドにおいて、「〜を entity.Song に変換します」といった、関数名やシグネチャから明らかな内容を日本語で説明するだけのコメントが複数存在する。
- **影響範囲**:
  - コードの可読性が低下し、ロジック変更時にコメントのメンテナンス漏れが発生するリスク（嘘のコメント化）を高める。
- **修正案**:
  - 公開メソッドであっても、その振る舞いが自明な場合はコメントを削除、または「なぜその処理が必要か」という背景を説明するドキュメントコメントに置き換える。

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

## 実装簡素化・ライブラリ活用提案 (LIB)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **LIB-002** | **Low** | 設定読み込みの自動化 | 手動での環境変数読み込み・型変換を廃止し、`kelseyhightower/envconfig` による構造体タグベースの宣言的な設定読み込みに移行する。 |
| **LIB-003** | **Low** | コレクション操作の効率化 | 冗長なループ処理（Map, Filter, Uniq 等）を `samber/lo` で置き換え、コードを簡潔にする。 |
| **LIB-004** | **Low** | ログファイルローテーション | 現状は起動毎に新ファイルを作成する形式だが、日付ベースのローテーションやサイズ制限がない。運用期間が長くなるとログファイルが肥大化する可能性がある。`lumberjack` などのライブラリ導入、または日付ベースのファイル切り替えロジックを実装する。 |

---

## 将来のリファクタリング計画 (FUTURE)

| ID | 優先度 | 概要 | 詳細・対応方針 |
|---|---|---|---|
| **FUTURE-001** | **Low** | Primitive Obsession 対応 | レビューで指摘された「プリミティブ型への執着」を将来的に解消する。現時点では別テーマとして扱い、段階的に進める。<br><br>**対象と狙い**: `PlayerDataMasters` 内の `map[string]Item` など、ドメイン概念のキーを `string` で扱っている箇所。クリアランプ名や難易度などはドメイン固有の概念であり、Value Object 化で型安全性を高める。<br><br>**方針**:<br>1. `ClearLampName` などの Value Object を Domain 層に定義し、コンストラクタで正規化・検証を行う。<br>2. `map[string]Item` を `map[ClearLampName]Item` のように置換し、呼び出し側も Value Object を使うように統一する。<br>3. 正規化処理（大文字化など）を Value Object に集約し、重複する `strings.ToUpper()`/`strings.ToLower()` を排除する。<br>4. 影響範囲が大きいため、マスターデータの一部（例: クリアランプ/難易度）から段階的に移行する。<br>5. 既存テストの修正で整合を取ることを基本とし、新規テスト追加は重複が出ない範囲に限定する。 |

## 追加で重点的に確認したい事項
- **JWTの運用ポリシー**: `issuer`/`audience` の運用があるか（必要なら `ValidateToken` に追加）。
- **CORS設定値**: `AllowOrigins` と `AllowCredentials` の組み合わせが安全か（`*` の禁止など）。
- **DB接続のTLS**: MySQL接続にTLSが必要な環境か（必要ならDSNで設定）。
- **APIドキュメント反映**: 入力制約やセキュリティ要件を `docs/API.md` / `docs/public_api.md` に追記する必要性。

---

## まとめ
- 主要なリスクは **CSRF対策不足** と **DBコネクションプール設定欠如**。
- パフォーマンス面では **セッション肥大化** が運用事故につながる可能性がある。
- 入力検証の統一方針とAPI仕様の整合性は、バグ防止・運用事故防止に直結する。
