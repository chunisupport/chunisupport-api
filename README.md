# chunisupport-api

`chunisupport-api`は、音楽ゲーム「CHUNITHM」のスコア管理などをサポートするアプリケーション `chunisupport` のためのバックエンドAPIサーバーです。

## 主な機能

- **ユーザー認証**: JWTとサーバーサイドセッションを組み合わせたハイブリッド認証方式による、安全なユーザー登録・ログイン・ログアウト機能を提供します。
- **APIトークン認証**: 外部クライアント向けに、1ユーザー1トークンの永続APIキーで保護された `/v1` エンドポイントを提供します。
- **プレイヤー情報**: ユーザーに紐づくプレイヤー情報を管理します。
- **楽曲データ**: CHUNITHMの公式楽曲データを元にしたデータベースを提供します。データの構築は別リポジトリのバッチ処理で行われます。

## ドキュメント

- [API仕様書（内部/公開）](docs/API.md)
- [リカバリーコード仕様](docs/recovery_code_spec.md)
- [アーキテクチャ概要](ARCHITECTURE.md)

## 技術スタック

- **言語**: [Go](https://golang.org/) (1.25.5)
- **Webフレームワーク**: [Echo](https://echo.labstack.com/)
- **データベース**: [MySQL](https://www.mysql.com/)
- **O/Rマッパー**: [sqlx](https://github.com/jmoiron/sqlx)
- **設定管理**: `encoding/json` と 環境変数 (`.env`)
- **マイグレーション**: [golang-migrate](https://github.com/golang-migrate/migrate)

## 開発環境のセットアップ

### 手順

1. リポジトリをクローンする。
   ```bash
   git clone https://github.com/Qman110101/chunisupport-api.git
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
   JWT_SECRET=your_secret_32_chars_min
   PW_PEPPER=your_pepper_32_chars_min
   DB_NAME=chunisupport
   DB_HOST=localhost
   DB_PORT=3306
   DB_USER=your_user
   DB_PASS=your_password
   ```
   ```json
   // .config/develop.settings.json
   {
     "app_port": 3002,
     "log_level": "debug",
     "log_paths": { "app": "log", "echo": "log" },
     "shutdown_timeout_seconds": 20,
    "auth": {
      "jwt_expiration_hour": 24,
      "session_expiration_hour": 24,
      "cookie_secure": false,
      "cookie_same_site": "lax"
    },
    "cors": {
      "allow_origins": ["http://localhost:3000"],
      "allow_credentials": true,
      "max_age": 3600
    }
  }
  ```
4. データベースを作成してマイグレーションする。
   - `static.db` は実行バイナリと同じディレクトリに配置する運用です。
     - `go run` の場合はカレントディレクトリが実行バイナリ相当になるため、`static.db` も同じ場所に作成してください。
   ```bash
   mysql -u <DB_USER> -p -e "CREATE DATABASE IF NOT EXISTS <DB_NAME>;"
   ```
   ```bash
   go install -tags 'mysql sqlite' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
   migrate -database "mysql://<DB_USER>:<DB_PASS>@tcp(<DB_HOST>:<DB_PORT>)/<DB_NAME>" -path migration/mysql up
   migrate -database "sqlite3://./static.db" -path migration/sqlite up
   
   **重要**: マイグレーション実行後、MySQLのイベントスケジューラを有効化してください（セッション自動クリーンアップに必要）。
   ```bash
   mysql -u <DB_USER> -p -e "SET GLOBAL event_scheduler = ON;"
   ```
   または、MySQL設定ファイル（`my.cnf`/`my.ini`）に以下を追加して永続化してください：
   ```ini
   [mysqld]
   event_scheduler = ON
   ```
   
5. 起動する。
   ```bash
   APP_ENV=develop go run main.go
   ```
