# chunisupport-api

`chunisupport-api`は、音楽ゲーム「CHUNITHM」のスコア管理などをサポートするアプリケーション `chunisupport` のためのバックエンドAPIサーバーです。

## 主な機能

- **内部API認証**: `/internal` エンドポイントでは Firebase ID トークンによる Bearer 認証を提供します。
- **APIトークン認証**: 外部クライアント向けに、1ユーザー1トークンの永続APIキーで保護された `/v1` エンドポイントを提供します。
- **プレイヤー情報**: ユーザーに紐づくプレイヤー情報を管理します。
- **楽曲データ**: CHUNITHMの公式楽曲データを元にしたデータベースを提供します。データの構築は別リポジトリのバッチ処理で行われます。

## ドキュメント

- [API仕様書（内部/公開）](docs/API.md)
- [アーキテクチャ概要](ARCHITECTURE.md)
- [logrotate設定手順](docs/logrotate.md)

## 技術スタック

- **言語**: [Go](https://golang.org/) (1.26.4)
- **Webフレームワーク**: [Echo](https://echo.labstack.com/)
- **データベース**: [MySQL](https://www.mysql.com/)
- **O/Rマッパー**: [sqlx](https://github.com/jmoiron/sqlx)
- **設定管理**: `encoding/json` と 環境変数 (`.env`)
- **マイグレーション**: [golang-migrate](https://github.com/golang-migrate/migrate)

## 開発環境のセットアップ

### 手順

1. リポジトリをクローンする。
   ```bash
   git clone https://github.com/chunisupport/chunisupport-api.git
   cd chunisupport-api
   ```
2. 依存関係を取得する。
   ```bash
   go mod tidy
   ```
3. 設定ファイルを用意する（詳細は `docs/configuration.md` を参照）。
   ```bash
   mkdir -p .config
   ```
   ```bash
   # .env
   APP_ENV=develop
   FIREBASE_CREDENTIALS_FILE=path/to/service-account.json
   DB_NAME=chunisupport
   DB_HOST=localhost
   DB_PORT=3306
   DB_USER=your_user
   DB_PASS=your_password
   ```
   ```json
   {
      "app_port": 3000,
      "logging": {
         "level": "debug",
         "app_file": ".log/app.log",
         "access_file": ".log/access.log",
         "stdout": true
      },
      "shutdown_timeout_seconds": 20,
      "cors": {
         "allow_origins": [
               "http://localhost:3000",
               "http://localhost:5173"
         ],
         "allow_credentials": true,
         "max_age": 3600
      },
      "static_db_path": "./static.db",
      "smalldata_db_path": "./smalldata.db"
   }
   ```
4. データベースを作成してマイグレーションする。
   - `static.db` の配置先は `.config/<環境>.settings.json` の `static_db_path` で指定します。`smalldata.db` の配置先は `smalldata_db_path` で指定します。マイグレーションを実行する際は、コマンド内のパスをこれらの設定値と一致させてください。
   ```bash
   mysql -u <DB_USER> -p -e "CREATE DATABASE IF NOT EXISTS <DB_NAME>;"
   ```
   ```bash
   go install -tags 'mysql sqlite' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
   migrate -database "mysql://<DB_USER>:<DB_PASS>@tcp(<DB_HOST>:<DB_PORT>)/<DB_NAME>" -path migration/mysql up
   migrate -database "sqlite3://./static.db" -path migration/sqlite up
   migrate -database "sqlite3://./smalldata.db" -path migration/sqlite_smalldata up
   ```

5. 起動する。
   ```bash
   APP_ENV=develop go run main.go
   ```
