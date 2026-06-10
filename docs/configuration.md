# 設定ファイル概要

アプリケーションは `.env` が存在すればそれを読み込み、そのうえで `APP_ENV` に対応する `.config/<APP_ENV>.settings.json` を読み込みます。最低限必要な項目だけを記載しています。

## `.env` (任意、開発時に推奨)

`.env` ファイル自体は必須ではありませんが、以下の環境変数は `.env` または実行環境の環境変数として必ず設定してください。

- `APP_ENV` (例: `develop`)
- `FIREBASE_CREDENTIALS_FILE` (必須。FirebaseサービスアカウントJSONへのパス)
- `TURNSTILE_SECRET_KEY` (必須。Cloudflare Turnstile のシークレットキー)
- `DB_NAME`
- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASS`

`APP_ENV` は読み込む設定ファイル名を決める値で、`.config/<APP_ENV>.settings.json` が参照されます。

## `.config/<APP_ENV>.settings.json` (必須)

最低限、以下のキーを持つJSONを用意してください。

- `app_port`
- `logging.level` (`debug`, `info`, `warn`, `error`)
- `logging.stdout`
- `logging.app_file`（任意。空文字ならアプリログのファイル出力なし）
- `logging.access_file`（任意。空文字ならアクセスログのファイル出力なし）
- `static_db_path` (統計データ用SQLiteデータベースのパス)
- `shutdown_timeout_seconds` (1以上)
- `cors.allow_origins`
- `cors.allow_credentials`
- `cors.max_age`
- `temp_data.max_total_mb`（任意。未設定または0以下の場合は`64`を使用）
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

`logging` は必須です。旧 `log_level` / `log_paths` 形式にはフォールバックしません。
`logging.stdout=false` の場合は `logging.app_file` と `logging.access_file` の両方が必須です。
`logging.app_file` と `logging.access_file` に同じパスは指定できません。
既存ファイルまたは親ディレクトリがシンボリックリンクの場合は実体パスで比較されるため、別名経由でも同じログファイルは指定できません。
ファイル出力を指定した場合、親ディレクトリは起動時に作成され、ログファイルは `0640` で作成されます。
Linux の logrotate 設定は [logrotate設定手順](logrotate.md) を参照してください。

旧設定から移行する場合は、デプロイ前に全環境の `.config/<APP_ENV>.settings.json` を更新してください。リポジトリ外の本番・ステージング設定も対象です。

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
  "logging": {
    "level": "debug",
    "app_file": ".log/app.log",
    "access_file": ".log/access.log",
    "stdout": true
  },
  "static_db_path": "./static.db",
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
ただし `/` の `GET` と `OPTIONS`、および `/internal/player-data/temp` の `POST` と `OPTIONS` に限っては、固定で `https://new.chunithm-net.com` も追加許可されます。

## Firebase 認証

Firebase を使ったログイン・連携エンドポイントは常に有効です。

- `.config/<APP_ENV>.settings.json` に Firebase 用のキーは不要です
- 環境変数 `FIREBASE_CREDENTIALS_FILE` に Firebase サービスアカウント JSON のパスを設定してください

`FIREBASE_CREDENTIALS_FILE` が未設定の場合、アプリケーションは起動時にエラーで終了します。

## Cloudflare Turnstile

ログインと初回登録では Cloudflare Turnstile のサーバー側検証を行います。

- `.config/<APP_ENV>.settings.json` に Turnstile 用のキーは不要です
- 環境変数 `TURNSTILE_SECRET_KEY` に Turnstile のシークレットキーを設定してください

`TURNSTILE_SECRET_KEY` が未設定の場合、アプリケーションは起動時にエラーで終了します。

## 起動失敗時の終了コード

設定読み込み、ログ初期化、DB接続、マスタデータのプリロード、Firebase認証サービスの初期化、サーバ起動、graceful shutdown に失敗した場合、アプリケーションは終了コード `1` で終了します。正常な SIGINT / SIGTERM による停止は終了コード `0` で終了します。

systemd 管理下で運用する場合は、`Restart=always` または `Restart=on-failure` の利用を検討してください。DBが同一ホストにある場合は `After=mysql.service`、外部DBを使う場合は `network-online.target` までの依存も併用できます。logrotate を使う場合は、`ExecReload=/bin/kill -HUP $MAINPID` を設定してください。
