# 設定ファイル概要

アプリケーションは `.env` が存在すればそれを読み込み、そのうえで `APP_ENV` に対応する `.config/<APP_ENV>.settings.json` を読み込みます。最低限必要な項目だけを記載しています。

## `.env` (任意、開発時に推奨)

`.env` ファイル自体は必須ではありませんが、以下の環境変数は `.env` または実行環境の環境変数として必ず設定してください。

- `APP_ENV` (例: `develop`)
- `JWT_SECRET` (32文字以上)
- `PW_PEPPER` (32文字以上)
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
- `auth.jwt_expiration_hour`
- `auth.session_expiration_hour`
- `auth.cookie_secure`
- `auth.cookie_same_site`
- `cors.allow_origins`
- `cors.allow_credentials`
- `cors.max_age`
- `database.pool.max_open_conns`（必須）
- `database.pool.max_idle_conns`（必須）
- `database.pool.conn_max_lifetime_sec`（必須）
- `database.pool.conn_max_idle_time_sec`（必須）

`database.pool.*` はすべて必須です。いずれかが欠けている場合、アプリケーションは起動時にエラーで終了します。
`0` を明示した場合は `sql.DB` と同様に「無制限/無効」として扱います。
また、`max_open_conns` が 1 以上で `max_idle_conns` がそれを上回る場合は、`max_idle_conns` は `max_open_conns` へ丸められます。

## Firebase 認証

Firebase を使ったログイン・連携エンドポイントは常に有効です。

- `.config/<APP_ENV>.settings.json` に Firebase 用のキーは不要です
- 環境変数 `FIREBASE_CREDENTIALS_FILE` に Firebase サービスアカウント JSON のパスを設定してください

`FIREBASE_CREDENTIALS_FILE` が未設定の場合、アプリケーションは起動時にエラーで終了します。
