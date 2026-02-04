# 設定ファイル概要

アプリケーションは `.env` と `.config/<環境>.settings.json` を読み込みます。最低限必要な項目だけを記載しています。

## `.env` (必須)

- `APP_ENV` (例: `develop`)
- `JWT_SECRET` (32文字以上)
- `PW_PEPPER` (32文字以上)
- `DB_NAME`
- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASS`

## `.config/<環境>.settings.json` (必須)

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
