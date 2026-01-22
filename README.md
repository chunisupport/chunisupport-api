# chunisupport-api

`chunisupport-api`は、音楽ゲーム「CHUNITHM」のスコア管理などをサポートするアプリケーション `chunisupport` のためのバックエンドAPIサーバーです。

## 主な機能

- **ユーザー認証**: JWTとサーバーサイドセッションを組み合わせたハイブリッド認証方式による、安全なユーザー登録・ログイン・ログアウト機能を提供します。
- **APIトークン認証**: 外部クライアント向けに、1ユーザー1トークンの永続APIキーで保護された `/v1` エンドポイントを提供します。
- **プレイヤー情報**: ユーザーに紐づくプレイヤー情報を管理します。
- **楽曲データ**: CHUNITHMの公式楽曲データを元にしたデータベースを提供します。データの構築は別リポジトリのバッチ処理で行われます。

## ドキュメント

- [内部API仕様書](docs/API.md)
- [公開API仕様書（暫定）](docs/public_api.md)
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

### 1. リポジトリのクローン

```bash
git clone https://github.com/Qman110101/chunisupport-api.git
cd chunisupport-api
```

### 2. 設定ファイルの準備

このアプリケーションは、**環境変数 (.env)** と **JSON設定ファイル** の2種類を組み合わせて設定を読み込みます。

#### a. `.env` ファイルの作成 (機密情報・DB接続情報)

プロジェクトのルートディレクトリに `.env` ファイルを作成し、以下の情報を設定します。

```bash
# .env

# アプリケーション環境 (必須)
APP_ENV=develop

# セキュリティ (必須: 32文字以上の強力なランダム文字列)
JWT_SECRET=your_super_secret_jwt_key_at_least_32_characters_long_here
PW_PEPPER=your_super_secret_pepper_value_at_least_32_characters_long

# データベース接続情報 (必須)
DB_NAME=chunisupport
DB_HOST=localhost
DB_PORT=3306
DB_USER=your_user
DB_PASS=your_password
```

> **注意**: `.env` ファイルは `.gitignore` で除外されています。**このファイルは絶対にリポジトリにコミットしないでください。**

#### b. JSON設定ファイルの作成 (アプリケーション動作設定)

次に、プロジェクトルートに `.config` ディレクトリを作成し、その中に環境に応じた設定ファイルを作成します。ファイル名は `(develop|staging|production).settings.json` の形式です。

まずは `.config/develop.settings.json` を作成します。

```bash
mkdir .config
```

**`.config/develop.settings.json` の内容例:**

```json
{
  "app_port": 3002,
  "log_level": "debug",
  "log_paths": {
    "app": "log",
    "echo": "log"
  },
  "shutdown_timeout_seconds": 20,
  "auth": {
    "jwt_expiration_hour": 24,
    "session_expiration_hour": 24,
    "cookie_secure": false,
    "cookie_same_site": "lax"
  },
  "cors": {
    "allow_origins": [
      "http://localhost:3000",
      "http://localhost:5173"
    ],
    "allow_credentials": true,
    "max_age": 3600
  }
}
```

**設定項目の補足:**
- `log_paths`: ログを出力する**ディレクトリ**を指定します（例: `"log"` とすると `log/YYYYMMDD-HHMMSS.log` が生成されます）。
- `shutdown_timeout_seconds`: シャットダウン時に待機する秒数を指定します。
- `cookie_secure`: HTTPSを使用する場合は`true`に設定（開発環境では`false`）。
- `cors.allow_origins`: フロントエンドアプリケーションのオリジンを指定。

### 3. データベースのセットアップ

ローカル環境にMySQLサーバーを準備し、`.env` で指定したデータベース名（例: `chunisupport`）でデータベースを作成してください。

### 4. 依存関係のインストール

```bash
go mod tidy
```

### 5. データベースマイグレーション

**ツールのインストール（未導入の場合）:**
```bash
go install -tags 'mysql sqlite' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

**マイグレーションの実行:**

以下のコマンドで、`migration/mysql` ディレクトリ内のマイグレーションファイルをデータベースに適用します。
DB接続情報は `.env` の設定値に合わせて変更してください。

```bash
migrate -database "mysql://<DB_USER>:<DB_PASS>@tcp(<DB_HOST>:<DB_PORT>)/<DB_NAME>" -path migration/mysql up
```

SQLiteはまだ利用していないため、コマンドの記載はありません。

### 6. アプリケーションの起動

実行時には `APP_ENV` 環境変数の指定が必須です（`.env` に記載していても、コマンド実行時に明示するか、`godotenv` による読み込みに依存します）。

```bash
APP_ENV=develop go run main.go
```

サーバーの起動ポートは JSON設定ファイルの `app_port` に従います（例: `3002`）。
