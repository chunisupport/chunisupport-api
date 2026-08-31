# chunisupport-api

`chunisupport-api`は、音楽ゲーム「CHUNITHM」のスコア管理などをサポートするアプリケーション `chunisupport` のためのバックエンドAPIサーバーです。

## 主な機能

- **内部API認証**: `/internal` エンドポイントでは Firebase ID トークンによる Bearer 認証を提供します。
- **APIトークン認証**: 外部クライアント向けに、1ユーザーあたり最大10個の名前付き永続APIキーで保護された `/v1` エンドポイントを提供します。
- **プレイヤー情報**: ユーザーに紐づくプレイヤー情報を管理します。
- **楽曲データ**: CHUNITHMの公式楽曲データを元にしたデータベースを提供します。データの構築は別リポジトリのバッチ処理で行われます。

## ドキュメント

- [API仕様書（内部/公開）](docs/API.md)
- [アーキテクチャ概要](ARCHITECTURE.md)
- [logrotate設定手順](docs/logrotate.md)

## 技術スタック

- **言語**: [Go](https://golang.org/) (1.27.0)
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
    TURNSTILE_SECRET_KEY=<Cloudflare Turnstileのシークレットキー>
    DATA_TRANSFER_HMAC_SECRET=<Base64で表現した32バイト以上のランダム値>
    DB_NAME=chunisupport
   DB_HOST=localhost
   DB_PORT=3306
   DB_USER=your_user
   DB_PASS=your_password
   ```
   ```json
    {
       "app_port": 3000,
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
               "http://localhost:3000",
               "http://localhost:5173"
         ],
          "allow_credentials": true,
          "max_age": 3600
       },
       "database": {
          "pool": {
             "max_open_conns": 25,
             "max_idle_conns": 25,
             "conn_max_lifetime_sec": 300,
             "conn_max_idle_time_sec": 60
          }
       }
    }
   ```
4. データベースを作成してマイグレーションする。
   ```bash
   mysql -u <DB_USER> -p -e "CREATE DATABASE IF NOT EXISTS <DB_NAME>;"
   ```
   ```bash
   go install -tags mysql github.com/golang-migrate/migrate/v4/cmd/migrate@latest
   migrate -database "mysql://<DB_USER>:<DB_PASS>@tcp(<DB_HOST>:<DB_PORT>)/<DB_NAME>" -path migration/mysql up
   ```

5. 起動する。
   ```bash
   APP_ENV=develop go run ./cmd/api
   ```

## プロジェクト構成

```
cmd/
├── api/          # APIサーバー用エントリポイント
│   └── main.go
├── recalculate-player-data/ # プレイヤーデータ再計算バッチ
│   └── main.go
└── export-static-data/ # 静的データ出力バッチ
    └── main.go
internal/         # 共通のドメインロジック・ユースケース・インフラ
└── ...
```

APIサーバーとバッチジョブは `internal/` 配下のドメイン層・ユースケース層を共有するマルチバイナリ構成です。
各バイナリは独立してビルド・実行できます。

| バイナリ | ビルドコマンド | 実行コマンド |
|---|---|---|
| APIサーバー | `go build -o _chunisupport-api ./cmd/api` | `go run ./cmd/api` |
| プレイヤーデータ再計算バッチ | `GOOS=linux GOARCH=amd64 go build -o _chunisupport-recalculate-player-data-linux-amd64 ./cmd/recalculate-player-data` | `go run ./cmd/recalculate-player-data` |
| 静的データ出力バッチ | `go build -o _chunisupport-export-static-data ./cmd/export-static-data` | `go run ./cmd/export-static-data` |

## プレイヤーデータ再計算バッチ

`go run ./cmd/recalculate-player-data` は最新マスタに基づいて全プレイヤーのRatingとOVER POWERを再計算します。MySQLアドバイザリロックで多重起動を防ぎ、プレイヤー単位のトランザクションで処理します。運用では07:00前後を避け、cronまたはsystemd timerから1日1回起動してください。
