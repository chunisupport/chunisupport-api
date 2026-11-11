# 設定ファイル概要

アプリケーションは `.env` が存在すればそれを読み込み、そのうえで `APP_ENV` に対応する `.config/<APP_ENV>.settings.json` を読み込みます。最低限必要な項目だけを記載しています。

加えて、新規登録ユーザー名の禁止語を `.config/username_forbidden_words.json` から読み込みます。このファイルは必須で、変更を反映するにはアプリケーションの再起動が必要です。

## `.env` (任意、開発時に推奨)

`.env` ファイル自体は必須ではありませんが、以下の環境変数は `.env` または実行環境の環境変数として必ず設定してください。

- `APP_ENV` (例: `develop`)
- `FIREBASE_CREDENTIALS_FILE` (必須。FirebaseサービスアカウントJSONへのパス)
- `TURNSTILE_SECRET_KEY` (必須。Cloudflare Turnstile のシークレットキー)
- `X_API_KEY` (必須。X アプリの OAuth 1.0a コンシューマーキー（API Key）)
- `X_API_KEY_SECRET` (必須。X アプリの OAuth 1.0a コンシューマーシークレット（API Key Secret）)
- `OPENAI_API_KEY` (フィーチャーフラグ `profile_comment` が有効な環境では必須。プロフィールコメントの内容判定に使う OpenAI の API キー。無効な環境では未設定でも起動できます)
- `DATA_TRANSFER_HMAC_SECRET` (必須。Base64で表現した32バイト以上のデータ移行用共有秘密鍵)
- `DB_NAME`
- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASS`

`APP_ENV` は読み込む設定ファイル名を決める値で、`.config/<APP_ENV>.settings.json` が参照されます。

## `.config/<APP_ENV>.settings.json` (必須)

最低限、以下のキーを持つJSONを用意してください。

- `app_port`
- `timezone` (APIレスポンスに使用するIANAタイムゾーン。例: `Asia/Tokyo`)
- `logging.level` (`debug`, `info`, `warn`, `error`)
- `logging.stdout`
- `logging.app_file`（任意。空文字ならアプリログのファイル出力なし）
- `logging.access_file`（任意。空文字ならアクセスログのファイル出力なし）
- `shutdown_timeout_seconds` (1以上)
- `client_ip.trusted_proxy_cidrs`（任意。設定時は、指定CIDRのプロキシから受信した`X-Forwarded-For`だけを`RealIP()`へ反映）
- `cors.allow_origins`
- `cors.allow_credentials`
- `cors.max_age`
- `temp_data.max_total_mb`（任意。一時保存するgzip本文の合計サイズの上限。未設定または0以下の場合は`64`を使用）
- `database.startup.max_wait_sec`（任意。未設定の場合は`120`、`0`の場合は再試行せず即時失敗）
- `database.startup.interval_sec`（任意。未設定の場合は`5`、1以上）
- `database.pool.max_open_conns`（必須）
- `database.pool.max_idle_conns`（必須）
- `database.pool.conn_max_lifetime_sec`（必須）
- `database.pool.conn_max_idle_time_sec`（必須）

`database.pool.*` はすべて必須です。いずれかが欠けている場合、アプリケーションは起動時にエラーで終了します。
`0` を明示した場合は `sql.DB` と同様に「無制限/無効」として扱います。
また、`max_open_conns` が 1 以上で `max_idle_conns` がそれを上回る場合は、`max_idle_conns` は `max_open_conns` へ丸められます。

`database.startup.*` は起動時のMySQL接続待機設定です。
MySQLがまだ起動していない場合、`max_wait_sec` の範囲内で `interval_sec` ごとに接続を再試行します。

MySQL接続はGo側の日時解釈とMySQLセッションの両方をUTCに固定します。DBサーバー全体のタイムゾーン設定には依存しません。

### 手作業SQLのタイムゾーン

アプリケーションの接続は DSN の `loc=UTC` と `time_zone='+00:00'` で、Go の日時解釈と接続プール内の MySQL セッションを UTC に固定します。この固定はアプリケーションの接続にだけ適用され、サーバーの `@@global.time_zone` には依存しません。DATETIME 列はタイムゾーンを持たず、アプリケーションは UTC の壁時計を保存します。TIMESTAMP 列は UTC の瞬間として保存され、読み書きのたびにセッションのタイムゾーンで変換されます。

mysql CLI や管理ツールなど、アプリケーションを経由しない手作業の SQL では、セッションのタイムゾーンはサーバー設定のままです。UTC 以外のセッションで `NOW()` や `CURRENT_TIMESTAMP` を使うと、DATETIME 列へ UTC 以外の壁時計が書き込まれ、アプリケーションが UTC として読むと時刻がずれます。DATETIME 列のオフセットなしの時刻リテラルは変換されず、書いた壁時計がそのまま保存されるため、UTC の壁時計を書いてください。オフセット付きの時刻リテラルは、MySQL 8.4 では保存前にセッションのタイムゾーンへ変換されます。UTC のセッションでは `'2020-01-01 10:10:10+05:30'` は `2020-01-01 04:40:10` として保存されます。TIMESTAMP 列の時刻リテラルはセッションのタイムゾーンから UTC へ変換して保存され、参照時はセッションのタイムゾーンへ戻して表示されます。セッションを UTC にすると、アプリケーションが保存した UTC の値と見た目が一致します。

手作業の SQL を実行するときは、他の文より先に次を実行して、そのセッションだけを UTC にしてください。

```sql
SET time_zone = '+00:00';
```

mysql CLI は接続が切れると一度だけ自動再接続し、送ろうとした文を再送します。再接続したセッションでは `time_zone` を含むセッション設定が初期化されるため、先に実行した `SET` は失われます。再接続後も UTC にするには、接続時に `--init-command` を指定してください。この文は接続時と再接続時に実行されます。

```bash
mysql --init-command="SET time_zone = '+00:00'" -u <DB_USER> -p <DB_NAME>
```

自動再接続を止め、切断をエラーにする場合は `--skip-reconnect` を指定します。接続し直したあとは、再び他の文より先に `SET time_zone = '+00:00';` を実行してください。

```bash
mysql --skip-reconnect -u <DB_USER> -p <DB_NAME>
```

パラメータ付きクエリは、DSNの `interpolateParams=true` によりクライアント側で値を展開します。サーバーサイドプリペアドステートメントの Prepare / Execute / Close をクエリごとに行わないためです。接続の文字セットと照合順序はDSNで上書きせず、go-sql-driver/mysql の既定である utf8mb4（`utf8mb4_general_ci`）を使います。スキーマも `utf8mb4` です。ドライバが展開を拒否する big5 / cp932 / gb2312 / gbk / sjis / gb18030 は指定しません。`NO_BACKSLASH_ESCAPES` はアプリケーションからもマイグレーションからも有効化せず、MySQL 8.4 の既定 `sql_mode` にも含まれません。このSQLモードを有効にした接続では使わないでください。`[]byte` の値は `_binary'...'` に展開され、JSON列へ書き込むとエラー3144になるため、JSON列へ渡す値は `json.RawMessage` にします。

`client_ip.trusted_proxy_cidrs` が未設定の場合、クライアントIPにはTCP接続元を使用します。リバースプロキシ配下では、アプリケーションへ接続するプロキシのCIDRをすべて指定してください。指定されたCIDR以外のアドレスや、プロキシが付加したと確認できないIPは信頼しません。外側のプロキシでは、クライアントから持ち込まれた`X-Forwarded-For`を上書きまたは破棄してから付加する設定が必要です。

`logging` は必須です。旧 `log_level` / `log_paths` 形式にはフォールバックしません。
`logging.stdout=false` の場合は `logging.app_file` と `logging.access_file` の両方が必須です。
`logging.app_file` と `logging.access_file` に同じパスは指定できません。
既存ファイルまたは親ディレクトリがシンボリックリンクの場合は実体パスで比較されるため、別名経由でも同じログファイルは指定できません。
ファイル出力を指定した場合、親ディレクトリは起動時に作成され、ログファイルは `0640` で作成されます。
Linux の logrotate 設定は [logrotate設定手順](logrotate.md) を参照してください。

旧設定から移行する場合は、デプロイ前に全環境の `.config/<APP_ENV>.settings.json` を更新してください。リポジトリ外の本番・ステージング設定も対象です。

## `.config/username_forbidden_words.json` (必須)

新規登録時だけ適用するユーザー名の禁止語を指定します。既存ユーザーの参照やDBからの復元には適用されません。

```json
{
  "exact": [
    "root",
    "official"
  ],
  "contains": [
    "admin",
    "system"
  ]
}
```

- `exact`: ユーザー名全体が一致した場合に拒否します。
- `contains`: ユーザー名の一部に含まれた場合に拒否します。
- どちらもユーザー名からアンダーバー（`_`）を除いた文字列で判定します（例: `ad_min` は `admin` に一致）。そのため、アンダーバーを含む禁止語は一致しません。
- 禁止語には空文字以外の任意の文字列を指定できます。
- UTF-8 BOMの有無にかかわらず読み込めます。
- 空文字、不明なJSONフィールド、壊れたJSONがある場合は起動に失敗します。
- 重複した禁止語は起動時に除去されます。

禁止語に該当した場合、APIは一致した単語を公開せず、`registration_failed`を返します。

旧形式:

```json
{
  "log_level": "info",
  "log_paths": {
    "app": ".log/app-20060102-150405.log",
    "access": ".log/access-20060102-150405.log"
  }
}
```

新形式:

```json
{
  "logging": {
    "level": "info",
    "app_file": ".log/app.log",
    "access_file": ".log/access.log",
    "stdout": false
  }
}
```

開発環境の例:

```json
{
  "app_port": 3002,
  "timezone": "Asia/Tokyo",
  "logging": {
    "level": "debug",
    "app_file": ".log/app.log",
    "access_file": ".log/access.log",
    "stdout": true
  },
  "shutdown_timeout_seconds": 20,
  "cors": {
    "allow_origins": [
      "http://localhost:3000"
    ],
    "allow_credentials": true,
    "max_age": 3600
  },
  "database": {
    "startup": {
      "max_wait_sec": 600,
      "interval_sec": 1
    },
    "pool": {
      "max_open_conns": 25,
      "max_idle_conns": 25,
      "conn_max_lifetime_sec": 300,
      "conn_max_idle_time_sec": 60
    }
  }
}
```

`cors.allow_origins` は全体の基本設定です。
オリジンには `https://*.example.pages.dev` のような `*` / `?` を含むパターンを指定できます。
この場合、許可された実際のリクエスト元オリジンが `Access-Control-Allow-Origin` に返されます。
ただし `/healthz` の `GET` と `OPTIONS`、および `/internal/player-data/temp` の `POST` と `OPTIONS` に限っては、固定で `https://new.chunithm-net.com` も追加許可されます。

## feature-flag（機能のリリース切り替え）

`internal/config/feature-flag.yml` を `go:embed` でバイナリに同梱します。秘密値は置きません。変更の反映には再ビルドと再配布が必要で、緊急停止スイッチとしては使えません。フラグ自体を切り替える環境変数はありません。`APP_ENV` が `develop` / `staging` / `production` のどれかに対応する値を選びます。

現在の登録フラグは `profile_comment`（プロフィールコメント）だけです。

```yaml
profile_comment:
  develop: true
  staging: true
  production: false
```

`develop` と `staging` では `PUT /internal/me/comment` と `GET /internal/users/:username/comment` を登録し、保存時に OpenAI Moderation API を呼びます。`production` は明示的に `false` です。本番の設定ファイルや環境変数へ何も足さなくても無効になり、省略した場合と同じです。無効時は両エンドポイントを登録しないため、未登録パスと同じ 404 になります。Moderation クライアントは作らないので API は呼ばれず、`OPENAI_API_KEY` も不要です。コメントは専用エンドポイント以外のレスポンスには含まれません。フロントへフラグ一覧を返す API はありません。

### ファイル形式と検証

以下は書式例です。

```yaml
new_ranking_page:
  develop: true
  staging: true
  production: false
```

運用環境は `develop` / `staging` / `production` の3種類です。省略した環境は `false`、`new_ranking_page: {}` は全環境で `false` になります。

| 入力 | 扱い |
| --- | --- |
| 空ファイル・コメントのみ・`{}` | フラグ0件 |
| ルートがマッピングでない | エラー |
| YAMLドキュメントが複数 | エラー |
| 機能名の重複 | エラー |
| 機能の値がマッピングでない（空値、`true` など） | エラー |
| `develop` / `staging` / `production` 以外の環境キー | エラー |
| 環境キーの重複 | エラー |
| 環境の値が `!!bool` タグのスカラー以外（`"true"`、`yes`、`1`、`null`、空値など） | エラー |

機能名と環境キーには文字列スカラーを使います。別名参照をbool値として指定することはできません。`yaml.Node` の種別とタグを確認してからbool値へ変換し、暗黙の型変換には頼りません。

APIとバッチは共通の `loadConfig` を通り、全環境のYAML定義を検証した後、機能名を `featureflag.RegisteredNames()` と両方向に突き合わせます。片方にしかない名前はエラーです。最後に `APP_ENV` に対応する値を選び、`Config.FeatureFlags` に格納します。このフィールドはJSONに出力しません。

未知の `APP_ENV` はフラグを全て無効にしますが、YAMLの検証と登録一覧の突き合わせは省略しません。`APP_ENV=test` のテストでも定義の破損を検出します。なお、`validateEnv` が検査するのは使用可能な文字だけなので、未知の `APP_ENV` でも対応する `.config/<APP_ENV>.settings.json` と必要な設定があれば起動します。

### 利用ルール

- フラグで制御するのは機能を使えるかどうかだけです。必須設定や外部サービスの初期化は変更しません。例外として `profile_comment` が無効なときは、その機能専用の OpenAI Moderation クライアントを作らず、`OPENAI_API_KEY` も必須にしません。有効なときだけキーを必須にし、クライアントを初期化します。
- 新しいエンドポイントはrouterで登録を切り替え、無効時は404にします。
- 既存処理の切り替えや、APIとCLIの両方で使う処理はusecaseで判定します。handlerとusecaseでの二重判定は必須ではありません。
- 無効時に404以外を返す必要があれば、機能を追加する際に決めて `docs/API.md` に記載します。
- `featureflag.Flags` は値として必要なコンストラクタだけに渡します。routerで組み立てる場合も、CLIのmainで組み立てる場合も同じです。
- テストは `featureflag.New(featureflag.X)` で有効な機能を明示し、`featureflag.New()` またはゼロ値で全機能を無効にします。テスト専用名も渡せます。`New` は本番の登録一覧を参照せず、集合に含まれない名前に対する `Enabled` は `false` です。

### 追加・リリース・削除の手順

1. 開発開始時に `internal/domain/featureflag/info.go` の機能名定数、非公開の `registeredNames`、YAMLの3か所へ追加します。最初は `develop: true` だけを指定します。
2. staging、productionの順に `true` にします。コードと一緒にレビューし、再ビルド・再配布します。
3. 全環境で安定したら定数、登録一覧の要素、YAMLの定義、判定の分岐を削除します。

定数だけを追加して登録一覧への追加を忘れた場合、その定数とYAMLの突き合わせは行われません。コード生成や静的解析による検出は行わないため、追加時のレビューで3か所を確認してください。`RegisteredNames()` はコピーを返し、`Flags` は生成後に変更しない読み取り専用の集合として並行参照できます。

APIとバッチを別々に配布すると、一時的にフラグの状態がずれることがあります。DBスキーマやデータ形式の互換性は、フラグとは別に管理してください。

フロントへの公開API、有効なフラグ一覧の起動時ログ、実行中の切り替え（ホットリロード）は対象外です。

## Firebase 認証

Firebase を使ったログイン・連携エンドポイントは常に有効です。

- `.config/<APP_ENV>.settings.json` に Firebase 用のキーは不要です
- 環境変数 `FIREBASE_CREDENTIALS_FILE` に Firebase サービスアカウント JSON のパスを設定してください

`FIREBASE_CREDENTIALS_FILE` が未設定の場合、アプリケーションは起動時にエラーで終了します。

## Cloudflare Turnstile

ログインと初回登録では Cloudflare Turnstile のサーバー側検証を行います。

- `.config/<APP_ENV>.settings.json` に Turnstile 用のキーは不要です
- 環境変数 `TURNSTILE_SECRET_KEY` に Turnstile のシークレットキーを設定してください
- siteverify の `success` に加え、`hostname` が `cors.allow_origins` のホストと一致すること、`action` がエンドポイントの期待値と一致することを確認します
- ログインの `action` は `login`、新規登録の `action` は `signup` です。フロントの Turnstile ウィジェットにも同じ値を設定してください。未設定や不一致のトークンは無効として拒否します
- `hostname` はスキームとポートを含まないため、`http://localhost:3000` は `localhost` として照合します。`https://*.example.pages.dev` のようなワイルドカードもホスト名に対して同様に判定します
- `*` や `https://*`、`https://**` のような全許可のオリジン（ホストが `*` だけで構成されるもの）は、Turnstile の許可ホストにはしません。別ホストで取得したトークンの流用を防げなくなるためです
- 許可ホストが1件も残らない場合、ログインと新規登録はすべて無効トークンとして拒否されます。この場合は起動時に警告ログを出力します
- `/healthz` などに追加する `https://new.chunithm-net.com` は Turnstile の許可ホストに含めません
- ローカル開発では Cloudflare のテスト用キー（シークレット `1x0000000000000000000000000000000AA` とフロントのサイトキー `1x00000000000000000000AA`）を使えます。テスト用シークレットの応答は `hostname` が `example.com` 固定で `action` も付かないため、siteverify が `metadata.result_with_testing_key: true` を返した場合は `hostname` と `action` を照合しません。実キーではこのフラグが返らないため、本番の照合には影響しません

`TURNSTILE_SECRET_KEY` が未設定の場合、アプリケーションは起動時にエラーで終了します。

## X API

設定画面に X アカウントのユーザー名（`@` ID）を表示するため、`PUT /internal/me/x-account` で X API の `GET /2/users/me` を呼び出します。

- 環境変数 `X_API_KEY` と `X_API_KEY_SECRET` に、X アプリの OAuth 1.0a コンシューマーキーを設定してください
- Firebase の X ログインが発行するアクセストークンは、Firebase コンソールの X プロバイダに設定した X アプリのものです。署名に別アプリのキーを使うと X に拒否されるため、同じ X アプリのキーを設定してください
- アプリ専用の Bearer トークンは使いません。`GET /2/users/me` は利用者本人のトークンでしか呼べないためです
- `GET /2/users/me` は従量課金でも課金対象外とされています（X Developers Community での X スタッフの回答、2026年5月）。念のため Developer Console で支払い上限を設定し、使用量に計上されていないことを確認してください

`X_API_KEY` または `X_API_KEY_SECRET` が未設定の場合、アプリケーションは起動時にエラーで終了します。

## OpenAI Moderation API

プロフィールコメントの保存（`PUT /internal/me/comment`）では、文字数と文字種の検証後に OpenAI Moderation API（`POST /v1/moderations`、モデル `omni-moderation-latest`）で内容を判定します。この処理とコメントAPIの登録は、フィーチャーフラグ `profile_comment` が有効な環境だけです。現在の定義では `develop` と `staging` が有効、`production` が無効です。

- 有効な環境では、環境変数 `OPENAI_API_KEY` に OpenAI の API キーを設定してください。未設定または空白だけの場合、アプリケーションは起動時にエラーで終了します
- 無効な環境ではキーが未設定でも起動します。Moderation クライアントは作らず、APIは呼びません
- 送信するのは空白除去後のコメント本文だけです。ユーザー名やIDなど、利用者を特定できる情報は送信しません
- 応答の `flagged` が `true` のコメントは保存しません
- 通信の失敗、タイムアウト（5秒）、2xx 以外の応答など、判定結果を得られなかった場合も保存しません
- コメントの削除（空文字列の保存）は判定せずに実行します
- 判定の対象は保存時のコメントだけです。導入前に保存されたコメントは判定しません

## オブジェクトストレージ静的データ

`cmd/export-static-data` を実行する場合に限り、以下の環境変数が追加で必要です。

- `OBJECT_STORAGE_ENDPOINT_URL`
- `OBJECT_STORAGE_ACCESS_KEY_ID`
- `OBJECT_STORAGE_SECRET_ACCESS_KEY`
- `OBJECT_STORAGE_BUCKET_NAME`
- `CLOUDFLARE_API_TOKEN`
- `CLOUDFLARE_ZONE_ID`
- `STATIC_DATA_PUBLIC_BASE_URL`

APIサーバーの起動にはこれらの環境変数は不要です。開発、ステージング、本番では、エンドポイント、バケット、Zone ID、公開URLをその環境のSecretへ置き、環境間で共有しません。コードはこれらを固定しません。`Cache-Control` の値は3環境で同じです。秘密情報はAPIサーバーと共有せず、バッチ専用のsystemd EnvironmentFileやコンテナ基盤のSecretへ保存してください。EnvironmentFileはバッチ実行ユーザーだけが読める`0600`とし、共有`.env`の使用はローカル開発に限定します。

出力するオブジェクトキー、`Cache-Control`、実行方法は [静的データのオブジェクトストレージエクスポート](static_data_export.md) を参照してください。

## 楽曲データ収集バッチ

`cmd/song-batch` と、管理画面から楽曲バッチを実行する API サーバーの両方で、以下の環境変数が必要です。

| 変数名 | 通常実行 | 大型アップデート | 用途 |
| --- | --- | --- | --- |
| `CHUNISUPPORT_BATCH_OFFICIAL_URL` | 必須 | 必須 | 公式データソースのURL |
| `CHUNISUPPORT_BATCH_ADDITIONAL_SONGS_SHEET_ID` | 必須 | 必須 | 追加楽曲のスプレッドシートID |
| `CHUNISUPPORT_BATCH_GOOGLE_CLOUD_API_KEY` | 必須 | 必須 | Google Sheets APIのキー |
| `CHUNISUPPORT_BATCH_GOOGLE_SPREADSHEET_BASE_URL` | 必須 | 必須 | Google Sheets APIのベースURL |
| `CHUNISUPPORT_BATCH_GOOGLE_SHEET_ID` | 必須 | 不要 | mainframe（譜面定数）のスプレッドシートID |
| `CHUNISUPPORT_BATCH_ST1027_URL` | 任意 | 不要 | st1027データソースのURL |
| `CHUNISUPPORT_BATCH_OTOGE_DB_URL` | 任意 | 不要 | otoge-dbデータソースのURL |
| `CHUNISUPPORT_BATCH_WIKI_BASE_URL` | 任意 | 不要 | otoge-dbの `wikiwiki_url` から除去するWikiのベースURL。未設定の場合はWikiページタイトルを補完しません |

- 必須の変数が欠けている場合、その実行は失敗し、MySQLは更新されません。任意の変数が欠けている場合は、そのデータソースを除外して警告付き成功になります。
- API サーバーは起動時に環境変数を読み込みます。大型アップデートで `CHUNISUPPORT_BATCH_GOOGLE_SHEET_ID` などを変更した場合は、管理画面から実行する前に API サーバーを再起動してください。CLI は実行のたびに読み込みます。
- API キーは秘密情報です。設定ファイル（`.config/<APP_ENV>.settings.json`）には書かず、環境変数で渡してください。

`.env` の例:

```bash
CHUNISUPPORT_BATCH_OFFICIAL_URL=https://chunithm.sega.jp/storage/json/music.json
CHUNISUPPORT_BATCH_ST1027_URL=https://ntools.st1027.org/json/songdata_chunithm.json
CHUNISUPPORT_BATCH_OTOGE_DB_URL=https://otoge-db.net/chunithm/data/music-ex.json
CHUNISUPPORT_BATCH_WIKI_BASE_URL=https://wikiwiki.jp/chunithmwiki/
CHUNISUPPORT_BATCH_GOOGLE_SPREADSHEET_BASE_URL=https://sheets.googleapis.com/v4/spreadsheets
CHUNISUPPORT_BATCH_GOOGLE_CLOUD_API_KEY=<Google Cloud APIキー>
CHUNISUPPORT_BATCH_GOOGLE_SHEET_ID=<mainframeのスプレッドシートID>
CHUNISUPPORT_BATCH_ADDITIONAL_SONGS_SHEET_ID=<追加楽曲のスプレッドシートID>
```

## 起動失敗時の終了コード

設定読み込み、ログ初期化、DB接続、マスタデータのプリロード、Firebase認証サービスの初期化、サーバ起動、graceful shutdown に失敗した場合、アプリケーションは終了コード `1` で終了します。正常な SIGINT / SIGTERM による停止は終了コード `0` で終了します。

systemd 管理下で運用する場合は、`Restart=always` または `Restart=on-failure` の利用を検討してください。DBが同一ホストにある場合は `After=mysql.service`、外部DBを使う場合は `network-online.target` までの依存も併用できます。logrotate を使う場合は、`ExecReload=/bin/kill -HUP $MAINPID` を設定してください。

## ユーザーデータ移行

settings JSONにデータ移行用の設定節はありません。エクスポート、検証、インポートの各APIは常時登録されます。

APIサーバーでは、環境変数`DATA_TRANSFER_HMAC_SECRET`へBase64で表現した32バイト以上の共有秘密鍵を必ず設定します。移行ファイルを相互利用するChuniSupportサーバーには同じ秘密鍵を設定してください。秘密鍵が異なるサーバーのファイルは署名検証で拒否されます。
