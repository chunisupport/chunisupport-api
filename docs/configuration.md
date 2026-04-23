# 設定ファイル概要

アプリケーションは `.env` が存在すればそれを読み込み、そのうえで `APP_ENV` に対応する `.config/<APP_ENV>.settings.json` を読み込みます。最低限必要な項目だけを記載しています。

## `.env` (任意、開発時に推奨)

`.env` ファイル自体は必須ではありませんが、以下の環境変数は `.env` または実行環境の環境変数として必ず設定してください。

- `APP_ENV` (例: `develop`)
- `FIREBASE_CREDENTIALS_FILE` (必須。FirebaseサービスアカウントJSONへのパス)
- `DB_NAME`
- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASS`

`APP_ENV` は読み込む設定ファイル名を決める値で、`.config/<APP_ENV>.settings.json` が参照されます。

## `.config/<APP_ENV>.settings.json` (必須)

最低限、以下のキーを持つJSONを用意してください。

- `app_port`
- `log_level`
- `log_paths.app`
- `log_paths.echo`
- `static_db_path` (統計データ用SQLiteデータベースのパス)
- `shutdown_timeout_seconds` (1以上)
- `cors.allow_origins`
- `cors.allow_credentials`
- `cors.max_age`
- `temp_data.max_total_mb`（任意。未設定または0以下の場合は`64`を使用）
- `database.pool.max_open_conns`（必須）
- `database.pool.max_idle_conns`（必須）
- `database.pool.conn_max_lifetime_sec`（必須）
- `database.pool.conn_max_idle_time_sec`（必須）

`database.pool.*` はすべて必須です。いずれかが欠けている場合、アプリケーションは起動時にエラーで終了します。
`0` を明示した場合は `sql.DB` と同様に「無制限/無効」として扱います。
また、`max_open_conns` が 1 以上で `max_idle_conns` がそれを上回る場合は、`max_idle_conns` は `max_open_conns` へ丸められます。

`cors.allow_origins` は全体の基本設定です。
ただし `/` の `GET` と `OPTIONS`、および `/internal/player-data/temp` の `POST` と `OPTIONS` に限っては、固定で `https://new.chunithm-net.com` も追加許可されます。

## Firebase 認証

Firebase を使ったログイン・連携エンドポイントは常に有効です。

- `.config/<APP_ENV>.settings.json` に Firebase 用のキーは不要です
- 環境変数 `FIREBASE_CREDENTIALS_FILE` に Firebase サービスアカウント JSON のパスを設定してください

`FIREBASE_CREDENTIALS_FILE` が未設定の場合、アプリケーションは起動時にエラーで終了します。

## 起動失敗時の終了コード

設定読み込み、DB接続、マスタデータのプリロード、Firebase認証サービスの初期化、サーバ起動、graceful shutdown に失敗した場合、アプリケーションは終了コード `1` で終了します。正常な SIGINT / SIGTERM による停止は終了コード `0` で終了します。

systemd 管理下で運用する場合は、`Restart=always` または `Restart=on-failure` の利用を検討してください。DBが同一ホストにある場合は `After=mysql.service`、外部DBを使う場合は `network-online.target` までの依存を検討し、DB起動待ちリトライは必要に応じて別途実装してください。
