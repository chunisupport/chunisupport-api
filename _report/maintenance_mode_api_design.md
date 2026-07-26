# APIメンテナンスモード 実装計画・設計書

## 0. 文書の位置付け

本書は、chunisupport API にメンテナンスモードを追加するための実装計画・設計書である。

対象はAPIサーバーのみとし、Cloudflare Pages 上のフロントエンド実装と nginx による代替応答は扱わない。フロントエンドが後から本APIを利用できるよう、状態確認APIとエラー契約までは本書で定義する。

### 目的

- ADMIN だけがメンテナンスモードを開始・終了できるようにする。
- メンテナンス中は、ADMIN / EDITOR だけがAPIを継続利用できるようにする。
- PLAYER / EXTDEV / 未認証利用者には、認証情報の有無にかかわらず原則 `503 Service Unavailable` を返す。
- メンテナンス状態と自由記述コメントをDBへ永続化し、APIプロセス再起動後も状態を維持する。
- 通常時の全APIリクエストでDB参照や追加の認証処理が発生しない構成にする。

## 1. スコープ

### 1.1 対象

- メンテナンス状態を表すドメインモデル
- メンテナンス状態のDB永続化
- ADMIN向けの開始・終了API
- 公開状態確認API
- API全体を遮断するメンテナンスミドルウェア
- Firebase認証とAPIトークン認証の両方における ADMIN / EDITOR の通過
- メンテナンス中のログイン制御
- メンテナンス専用エラーコード、レスポンスヘッダー、ログ制御
- テストおよびAPIドキュメント更新

### 1.2 対象外

- Cloudflare Pages 上のメンテナンス画面、隠しログイン画面、管理画面
- nginx が返すメンテナンス応答
- VPS停止時やネットワーク障害時の検知
- メンテナンス開始・終了日時の予約
- 読み取り専用モード
- メンテナンス開始とデプロイ処理の自動連携
- 複数APIプロセス間でのリアルタイムなキャッシュ同期

## 2. 確定仕様

### 2.1 権限

| 利用者 | 通常時 | メンテナンス中 | 状態変更 |
|---|---:|---:|---:|
| ADMIN | 利用可 | 利用可 | 可 |
| EDITOR | 利用可 | 利用可 | 不可 |
| PLAYER | 利用可 | `503` | 不可 |
| EXTDEV | 利用可 | `503` | 不可 |
| 未認証 | 公開APIのみ利用可 | 原則 `503` | 不可 |

既存の `info.HasRole(accountTypeID, info.AccountTypeEditor)` を利用し、EDITOR以上、すなわち EDITOR / ADMIN をスタッフとして判定する。メンテナンス状態の変更は、既存のADMINロールミドルウェアでADMINだけに制限する。

### 2.2 メンテナンスコメント

- プレーンテキストとする。
- 改行を許可する。
- 最大1,000文字とする。文字数はUTF-8のバイト数ではなくUnicodeコードポイント数で数える。
- `CRLF` と `CR` は `LF` に正規化する。
- 前後の空白を除去する。
- 改行以外の制御文字は拒否する。
- メンテナンス開始時は空文字・空白のみを許可しない。
- メンテナンス終了時はコメントを空文字へ戻す。
- HTMLは保存せず、フロントエンドでもテキストとして描画する前提とする。

### 2.3 状態の永続性

DBを正とし、APIプロセス起動時に現在状態を読み込む。APIプロセス再起動によって自動解除されてはならない。

通常リクエストではインメモリの不変スナップショットだけを参照し、リクエストごとのDBアクセスは行わない。

## 3. 外部API仕様

### 3.1 公開状態確認

```http
GET /internal/system/status
```

認証不要。APIプロセスが動作している限り、通常時・メンテナンス中ともに `200 OK` を返す。

通常時:

```json
{
  "status": "operational",
  "comment": "",
  "updated_at": "2026-07-26T12:00:00+09:00"
}
```

メンテナンス中:

```json
{
  "status": "maintenance",
  "comment": "データ更新のためメンテナンスを実施しています。",
  "updated_at": "2026-07-26T12:30:00+09:00"
}
```

レスポンスヘッダー:

```http
Cache-Control: no-store
```

`updated_by_user_id` は内部監査情報であり、公開レスポンスには含めない。

### 3.2 メンテナンス状態変更

```http
PUT /internal/admin/maintenance
Authorization: Bearer <Firebase ID token>
Content-Type: application/json
```

ADMINのみ実行可能とする。EDITORが実行した場合は、既存の権限エラー契約に従い `403 Forbidden` を返す。

開始:

```json
{
  "enabled": true,
  "comment": "データ更新のためメンテナンスを実施しています。"
}
```

終了:

```json
{
  "enabled": false,
  "comment": ""
}
```

成功時は `200 OK` で、公開状態確認APIと同じ形式を返す。

リクエストDTOの `enabled` は `*bool` とし、未指定と `false` を区別する。未指定、不正なコメント、開始時の空コメントは `400 Bad Request` とする。終了時にコメントが指定されても、保存値は空文字へ統一する。

### 3.3 メンテナンス遮断応答

```http
HTTP/1.1 503 Service Unavailable
Retry-After: 60
Cache-Control: no-store
Content-Type: application/json
```

```json
{
  "error": {
    "status": 503,
    "code": "maintenance_mode"
  }
}
```

既存の `service_unavailable` は、メンテナンス以外の一時的障害でも利用されている。そのため、フロントエンドが誤判定しないよう専用コード `maintenance_mode` を追加する。

メンテナンスコメントは個々の `503` には含めない。表示が必要なクライアントは `GET /internal/system/status` から取得する。

## 4. メンテナンス中の例外経路

以下だけは、メンテナンスミドルウェアによる即時遮断の対象外とする。

| メソッド・パス | 理由 |
|---|---|
| `OPTIONS *` | CORSプリフライトを成立させるため |
| `GET /healthz` | プロセスの生存確認とメンテナンス状態を分離するため |
| `GET /internal/system/status` | フロントエンドが状態とコメントを確認するため |
| `POST /internal/auth/login` | Firebase認証後にADMIN / EDITORだけログインさせるため |

`POST /internal/auth/signup` は例外に含めず、メンテナンス中は `503` とする。

`PUT /internal/admin/maintenance` も無条件の例外にはしない。メンテナンスミドルウェアでスタッフ認証を通した後、既存のFirebase厳格認証とADMINロール確認を通す。これにより、メンテナンス中でもADMINは終了操作を行え、EDITORは状態を変更できない。

### 4.1 ログイン時の追加制御

ログインAPIは認証処理を行う必要があるためミドルウェアの遮断を通過させるが、ログインユースケース内で現在のメンテナンス状態を確認する。

- 通常時: 既存どおりログインを許可する。
- メンテナンス中かつ ADMIN / EDITOR: ログインを許可する。
- メンテナンス中かつ PLAYER / EXTDEV: `maintenance_mode` の `503` を返す。

隠しログイン画面と通常ログイン画面が同じAPI・コンポーネントを利用しても、API側で権限を保証できる構成とする。

## 5. ドメイン設計

### 5.1 値オブジェクト

追加候補:

```text
internal/domain/vo/maintenancecomment/
  maintenance_comment.go
  maintenance_comment_test.go
```

`MaintenanceComment` は、正規化・最大文字数・制御文字・空文字の規則を保持する不変の値オブジェクトとする。

用途に応じて次の生成方法を用意する。

- メンテナンス開始用: 空文字を拒否する。
- DB復元用: 無効状態の空文字を復元できる。

テスト以外で `Must` 系コンストラクタは使用しない。

### 5.2 エンティティ

追加候補:

```text
internal/domain/entity/system_maintenance.go
internal/domain/entity/system_maintenance_test.go
```

概念上の属性:

```go
type SystemMaintenance struct {
    ID              int
    Enabled         bool
    Comment         maintenancecomment.MaintenanceComment
    UpdatedByUserID *int
    UpdatedAt       time.Time
}
```

ドメインエンティティには `db` / `json` タグを付けない。

振る舞い:

- `Enable(comment, updaterUserID, now)`
- `Disable(updaterUserID, now)`
- `IsEnabled()`

`Enable` はコメント必須の不変条件を保証する。`Disable` はコメントを空にする。ロール判定はメンテナンス状態そのものの規則ではなくアプリケーション上の認可であるため、エンティティへ持ち込まない。

## 6. DB・リポジトリ設計

### 6.1 マイグレーション

実装開始時点の最新番号を再確認したうえで、新しいマイグレーションを追加する。本書作成時点の候補は次のとおり。

```text
migration/mysql/000039_create_system_maintenance.up.sql
migration/mysql/000039_create_system_maintenance.down.sql
```

既存マイグレーションは変更しない。`migration/schema_mysql.sql` にも完成後のスキーマを反映する。

テーブル案:

```sql
CREATE TABLE system_maintenance (
    id TINYINT UNSIGNED NOT NULL,
    enabled BOOLEAN NOT NULL,
    comment VARCHAR(1000) NOT NULL DEFAULT '',
    updated_by_user_id INT UNSIGNED NULL,
    updated_at DATETIME(6) NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT fk_system_maintenance_updated_by_user
        FOREIGN KEY (updated_by_user_id)
        REFERENCES users (id)
        ON DELETE SET NULL,
    CONSTRAINT chk_system_maintenance_singleton
        CHECK (id = 1)
);
```

マイグレーション時に `id = 1`、`enabled = false` の初期行を投入する。

MySQLのバージョンと既存スキーマ規約を実装時に確認し、`CHECK` 制約がプロジェクトの対応範囲と合わない場合は、主キーとリポジトリ実装で `id = 1` を保証する。ここは実装前に実環境のMySQLバージョンを確認して確定する。

### 6.2 Infraモデル

```text
internal/infra/models/system_maintenance_model.go
```

DBタグはInfraモデルにだけ定義し、`ToEntity()` / `FromEntity()` でドメインエンティティと変換する。

### 6.3 リポジトリ

インターフェース:

```text
internal/domain/repository/system_maintenance_repository.go
```

実装:

```text
internal/infra/repository/system_maintenance_repository_impl.go
```

責務:

- `Find(ctx)` で `id = 1` の集約を復元する。
- `Save(ctx, entity)` で集約全体を保存する。
- SQLは `SELECT *` を使わず、全カラムを明示する。
- 部分更新専用メソッドは作らない。
- 初期行が存在しない場合は設定不備として扱い、自動生成で隠蔽しない。

この更新は単一行の `UPDATE` で完結するため、リポジトリインターフェースへDBやトランザクションの実装詳細を露出させない。

同時に複数のADMINが更新した場合は最終更新を採用する。ただし、DB更新とメモリ反映の順序が逆転しないよう、ユースケース側で更新処理を直列化する。

## 7. ユースケースとキャッシュ

### 7.1 責務分割

ミドルウェアが参照する読み取り専用インターフェースと、管理APIが利用する更新ユースケースを分ける。

概念例:

```go
type MaintenanceStateProvider interface {
    Current() MaintenanceState
}

type SystemMaintenanceUsecase interface {
    Current() MaintenanceOutput
    Update(
        ctx context.Context,
        actorUserID int,
        enabled bool,
        comment string,
    ) (MaintenanceOutput, error)
}
```

単一の具象ユースケースが両方を実装してよい。ミドルウェアは更新操作へ依存しない。

### 7.2 起動時

1. DB接続を確立する。
2. `system_maintenance` の単一行を取得する。
3. 不変な状態スナップショットを作成する。
4. `atomic.Pointer` などで公開する。
5. 取得・変換に失敗した場合は、誤って通常運用を開始せずAPI起動を失敗させる。

依存生成はComposition Rootで行う。起動エラーを適切に返せるよう、`cmd/api/main.go` からサーバー生成までのシグネチャを必要に応じて変更し、初期化失敗を `panic` で処理しない。

### 7.3 参照時

通常リクエスト・状態確認API・ログインユースケースは、メモリ上の不変スナップショットを読み取る。読み取りごとのロックやDBアクセスは行わない。

### 7.4 更新時

1. 更新用 `sync.Mutex` を取得する。
2. 現在のエンティティに `Enable` または `Disable` を適用する。
3. リポジトリの `Save` でDBへ保存する。
4. DB保存成功後にだけ、メモリ上のスナップショットを原子的に差し替える。
5. ロックを解放する。

DB保存に失敗した場合、メモリ上の状態は変更しない。この順序により、「レスポンス上は失敗したがAPIだけメンテナンス状態になった」という不整合を防ぐ。

本設計は単一APIプロセスを前提とする。将来複数プロセス化する場合は、DB通知、Redis、短周期ポーリングなどによる同期方式を別途設計する。

## 8. ミドルウェア設計

### 8.1 基本フロー

```text
リクエスト
  ├─ メンテナンスOFF → 追加認証なしで次へ
  └─ メンテナンスON
       ├─ 例外経路 → 次へ
       ├─ 有効な認証 + ADMIN / EDITOR → 認証結果をContextへ保存して次へ
       └─ その他 → maintenance_mode / 503
```

通常時のホットパスは、原子的な状態参照と条件分岐だけにする。

### 8.2 認証方式ごとのゲート

APIごとに既存の認証方式が異なるため、認証トークンの形式を推測する巨大なグローバルミドルウェアは作らない。

| ルート群 | メンテナンス中のスタッフ確認 |
|---|---|
| `/internal` | Firebase ID token |
| `/v1` | API token |
| `/compat` | API token |
| `/version` | API token |
| `/` | Firebase ID token |

各ゲートは共通の `MaintenanceStateProvider` とスタッフ判定を利用し、資格情報の解決部分だけをFirebase用・APIトークン用に分ける。

### 8.3 認証結果の再利用

メンテナンス中にゲートが認証済みの利用者をEcho Contextへ保存した場合、後続の既存認証ミドルウェアはその値を再利用する。

- Firebase: `userEntity` が既に存在すればトークン再検証を省略する。
- API token: 利用者とAPIトークン情報が既に存在すればDB再取得を省略する。

これにより、スタッフの1リクエストで同じ資格情報を二重検証しない。

Context値のキーは文字列を各所へ直書きせず、既存規約に合わせて定数化する。

### 8.4 認証エラーのマスク

メンテナンス中の一般利用者には、次の違いを公開しない。

- 認証情報なし
- 形式不正
- 期限切れ
- PLAYER / EXTDEV
- 存在しないAPIトークン

すべて `maintenance_mode` の `503` とする。これにより、メンテナンス中の外部契約を統一し、認証情報の有効性も余計に開示しない。

ただし、FirebaseやDBへの接続障害など内部の認証基盤エラーは、クライアントには `503 maintenance_mode` を返しつつ、原因をサーバーログへ別途記録する。

### 8.5 ミドルウェア順序

- メンテナンスゲートは、一般利用者向けの認証・レート制限より前に評価する。
- グローバルまたはルート固有のCORS処理はメンテナンスゲートの応答にも適用し、許可済みフロントエンドが `503` とエラーコードを読み取れるようにする。
- フロントエンドが待機時間を参照できるよう、CORSの `ExposeHeaders` に `Retry-After` を追加する。
- ログインAPIの既存レート制限とTurnstile検証は維持する。
- 互換APIに固有のエラーレスポンス変換がある場合、その契約を維持できる順序にする。
- Echo v5のミドルウェア適用順はルーターテストで固定する。

## 9. ヘルスチェックの扱い

`GET /healthz` はメンテナンス中も既存どおり `204 No Content` とする。

メンテナンスは「プロセスが異常」という意味ではないため、ヘルスチェックを `503` にすると、監視や将来のロードバランサーがプロセスを再起動・切り離す可能性がある。

利用者向けの運用状態は `GET /internal/system/status`、プロセスの生存は `/healthz` で分離する。

## 10. エラー・ログ設計

### 10.1 APIエラー

追加対象:

- `CodeMaintenanceMode = "maintenance_mode"`
- HTTPステータス `503 Service Unavailable`
- フロントエンド表示用の理由コードが必要な既存設計であれば、その対応表も追加する。

既存のカスタムHTTPエラーハンドラーを経由させ、レスポンス形式を統一する。`Retry-After` と `Cache-Control` はメンテナンスエラー生成時または専用エラーハンドリング時に必ず設定する。

### 10.2 ログ

メンテナンス中は大量の `503` が正常に発生し得る。`maintenance_mode` を通常の予期しない5xxと同じERRORレベルで毎回記録すると、障害ログが埋もれる。

- アクセスログには通常どおり記録する。
- `maintenance_mode` 自体はINFO相当または専用の抑制対象とする。
- DB保存失敗、状態初期化失敗、認証基盤障害などはERRORで記録する。
- メンテナンス開始・終了は、実行者ID、変更後状態、時刻を構造化ログへ記録する。
- コメント全文は、改行や任意入力によるログ汚染を避けるため原則ログへ出さない。

## 11. Clean Architecture上の配置

依存方向は次のとおりとする。

```text
App / Middleware / Handler
          ↓
       Usecase
          ↓
Domain Entity / Value Object / Repository Interface
          ↑
Infra Model / Repository Implementation
```

主な追加・変更候補:

```text
internal/domain/vo/maintenancecomment/maintenance_comment.go
internal/domain/entity/system_maintenance.go
internal/domain/repository/system_maintenance_repository.go

internal/infra/models/system_maintenance_model.go
internal/infra/repository/system_maintenance_repository_impl.go

internal/usecase/system_maintenance_usecase.go

internal/app/handler/api_internal/system_maintenance_handler.go
internal/app/middleware/maintenance_middleware.go
internal/app/middleware/firebase_auth_middleware.go
internal/app/middleware/api_token_middleware.go
internal/app/apierror/codes.go
internal/app/apierror/api_error.go
internal/app/router.go
internal/app/server.go

cmd/api/main.go

migration/mysql/<next>_create_system_maintenance.up.sql
migration/mysql/<next>_create_system_maintenance.down.sql
migration/schema_mysql.sql

docs/API.md
docs/error_code_reason_codes.md
```

実際のファイル名は既存の命名規則を優先する。UsecaseからInfra実装やDBモデルをimportしてはならない。

## 12. TDDによる実装計画

### 12.1 値オブジェクト・エンティティ

Red:

- 通常コメントを生成できる。
- 複数行コメントを保持できる。
- `CRLF` / `CR` が `LF` へ正規化される。
- 1,000文字の日本語を許可し、1,001文字を拒否する。
- 開始時の空文字・空白のみを拒否する。
- 改行以外の制御文字を拒否する。
- `Enable` が状態、コメント、更新者、時刻を更新する。
- `Disable` がコメントを空にする。

Green / Refactor:

- 最小限の値オブジェクトとエンティティを実装する。
- バリデーションをHandlerやUsecaseへ重複させない。

### 12.2 リポジトリ・Infraモデル

Red:

- DBモデルとエンティティを相互変換できる。
- `id = 1` を明示カラムで取得できる。
- `Save` が集約全体を保存する。
- 行がない場合とDBエラーを区別できる。

Green / Refactor:

- 新規マイグレーションとリポジトリを実装する。
- 既存のDBテスト方法に合わせてup/downを検証する。

### 12.3 ユースケース

Red:

- 起動時の状態をキャッシュへ読み込む。
- 開始・終了後に公開出力が更新される。
- DB保存失敗時にキャッシュが変化しない。
- 終了時にコメントが消える。
- 同時更新でもDBとキャッシュの順序が逆転しない。
- 時計とリポジトリをモックして結果を固定できる。

Green / Refactor:

- 読み取りは不変スナップショットとする。
- 更新はミューテックスで直列化する。

### 12.4 ミドルウェア

Red:

- 通常時は資格情報リゾルバーを呼ばずに通過する。
- メンテナンス中の未認証・不正トークン・PLAYER・EXTDEVが `503` になる。
- Firebase認証のADMIN / EDITORが通過する。
- APIトークン認証のADMIN / EDITORが通過する。
- `OPTIONS` と定義済み例外経路が通過する。
- 認証済みContextが後続ミドルウェアで再利用される。
- `503` に `Retry-After` と `Cache-Control` が付く。
- 許可済みOriginへのメンテナンス応答にCORSヘッダーが付く。
- 内部の認証基盤エラーは記録しつつクライアントにはメンテナンス応答を返す。

### 12.5 Handler・Router・ログイン

Red:

- 状態確認APIが通常時・メンテナンス中とも `200` を返す。
- ADMINが開始・終了できる。
- EDITORは状態変更できない。
- コメント不正時に `400` を返す。
- メンテナンス中もADMIN / EDITORがログインできる。
- メンテナンス中のPLAYER / EXTDEVログインが `503` になる。
- 代表的な `/internal`、`/v1`、`/compat` ルートで遮断と通過が成立する。
- 互換APIの既存エラー形式が壊れない。
- `/healthz` がメンテナンス中も `204` を維持する。

### 12.6 最終検証

コード実装時には、プロジェクト規約に従い次を実行する。

```text
go test ./...
gofmt -s -w .
go test ./...
```

並行読み取り・更新部分は、実行環境が許せば `go test -race ./...` も追加で確認する。

## 13. 実装順序

1. エラーコードと期待する外部API契約のテストを追加する。
2. コメント値オブジェクトとメンテナンスエンティティをTDDで実装する。
3. 新規マイグレーション、Infraモデル、リポジトリをTDDで実装する。
4. 起動時ロード、原子的な参照、直列化された更新を持つユースケースを実装する。
5. 公開状態確認HandlerとADMIN更新Handlerを実装する。
6. Firebase用・APIトークン用のメンテナンスゲートを実装する。
7. 既存認証ミドルウェアへContext再利用を追加する。
8. ログインユースケースへメンテナンス中のロール制限を追加する。
9. Routerへ各ゲートと例外経路を明示的に組み込む。
10. カスタムエラーハンドラーのメンテナンスログ抑制を実装する。
11. `docs/API.md` とエラーコード関連文書を更新する。
12. 全テスト、フォーマット、文字化け、依存方向を確認する。

## 14. デプロイ・ロールバック計画

### 14.1 デプロイ

1. 新規マイグレーションを適用し、初期行が無効状態で存在することを確認する。
2. APIをデプロイする。
3. `GET /healthz` が `204`、`GET /internal/system/status` が `operational` を返すことを確認する。
4. ADMINでメンテナンスを開始する。
5. 未認証・PLAYERが `503`、EDITOR / ADMINが通過することを確認する。
6. ADMINでメンテナンスを終了する。
7. 再度通常利用できることを確認する。
8. 検証環境では、メンテナンスONのままAPIを再起動し、状態が維持されることも確認する。

APIバイナリは起動時に新規テーブルを必要とするため、マイグレーションを先に適用する。

### 14.2 ロールバック

旧APIバイナリは追加テーブルを参照しないため、原則としてテーブルを残したままバイナリだけ戻す。障害時にdownマイグレーションまで即時適用すると、保存したメンテナンス状態と監査情報を失うため避ける。

旧バイナリへ戻した時点でAPIレベルのメンテナンスゲートは存在しなくなる。そのため、ロールバック手順では「旧バイナリでサービス公開してよいか」を運用判断として明示する。

## 15. 受け入れ条件

- ADMINだけがメンテナンス状態を変更できる。
- 状態変更はDBへ保存され、API再起動後も維持される。
- 通常時のリクエストごとに追加のDB参照・認証処理が発生しない。
- メンテナンス中、ADMIN / EDITORはFirebase認証とAPIトークン認証の対象APIを利用できる。
- メンテナンス中、PLAYER / EXTDEV / 未認証利用者は原則 `503 maintenance_mode` を受け取る。
- メンテナンス中、ADMIN / EDITORだけがログインに成功する。
- 公開状態確認APIから状態、コメント、更新時刻を取得できる。
- `/healthz` はメンテナンス中も `204` を返す。
- `503` には `Retry-After: 60` と `Cache-Control: no-store` が付く。
- メンテナンスによる想定内の `503` がERRORログを大量発生させない。
- 既存APIの認証、互換レスポンス、エラー契約を通常時に変更しない。
- `go test ./...` が成功し、`gofmt -s -w .` 適用後も差分が整形済みである。

## 16. 実装前に再確認する事項

以下は設計方針を変えるものではないが、実装着手時にリポジトリと実環境から確定する。

- 次に使用できるマイグレーション番号
- 本番MySQLのバージョンと `CHECK` 制約の扱い
- `/compat` のエラーレスポンス変換とEcho v5ミドルウェアの適用順
- APIが単一プロセスで稼働していること
- 既存のContextキーと認証済みAPIトークン情報の保持形式

複数APIプロセスで稼働していることが判明した場合だけは、本設計のインメモリキャッシュをそのまま採用せず、プロセス間同期方式を決めてから実装する。

## 17. 結論

APIのメンテナンスモードは、DBに永続化した単一の状態を起動時に読み込み、リクエスト処理ではメモリ上のスナップショットを参照する。

遮断処理は認証方式ごとのメンテナンスゲートとして配置し、メンテナンス中だけADMIN / EDITORの資格情報を確認する。状態変更は既存のADMIN認可で保護し、ログインAPIでは認証完了後に同じロール制限を適用する。

この構成により、通常時の負荷をほぼ増やさず、API再起動にも耐え、フロントエンドが明確な `maintenance_mode` と自由記述コメントを利用できる。
