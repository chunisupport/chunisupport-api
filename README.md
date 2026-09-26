# chunisupport-api

`chunisupport-api`は、音楽ゲーム「CHUNITHM」のスコア管理などをサポートするアプリケーション `chunisupport` のためのバックエンドAPIサーバーです。

## 主な機能

- **内部API認証**: `/internal` エンドポイントでは Firebase ID トークンによる Bearer 認証を提供します。
- **APIトークン認証**: 外部クライアント向けに、1ユーザーあたり最大10個の名前付き永続APIキーで保護された `/v1` エンドポイントを提供します。
- **プレイヤー情報**: ユーザーに紐づくプレイヤー情報を管理します。
- **楽曲データ**: CHUNITHMの公式楽曲データを元にしたデータベースを提供します。データは同梱の楽曲データ収集バッチ（`cmd/song-batch`）で構築し、ADMINは管理画面からも実行できます。

## ドキュメント

- [API仕様書（内部/公開）](docs/API.md)
- [設定ファイル・環境変数](docs/configuration.md)
- [楽曲データ収集バッチ](docs/song_batch.md)
- [譜面統計バッチの集計仕様](docs/chart_statistics_aggregation.md)
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
├── export-static-data/ # 静的データ出力バッチ
│   └── main.go
└── song-batch/   # 楽曲データ収集バッチ
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
| 楽曲データ収集バッチ | `GOOS=linux GOARCH=amd64 go build -o _chunisupport-song-batch-linux-amd64 ./cmd/song-batch` | `go run ./cmd/song-batch` |

## 楽曲データ収集バッチ

`go run ./cmd/song-batch` は公式データ、追加楽曲シート、mainframe などの外部データソースを取得し、`songs` / `charts` / `worldsend_charts` / `courses` を更新します。以前は `chunisupport-song-batch` リポジトリで管理していたものを統合しました。

| 実行方法 | 内容 |
|---|---|
| `go run ./cmd/song-batch` | 通常実行（cron から定期実行） |
| `go run ./cmd/song-batch --major-update` | 大型アップデート用。公式データと追加楽曲だけで更新し、譜面定数の更新ルールを適用 |
| `go run ./cmd/song-batch --fill-missing-release-date` | 日付が得られない新規楽曲に実行日（JST）をリリース日として補完 |
| 管理画面 `/admin/song-batch` | ADMINが上記と同じ処理を任意のタイミングで実行（API プロセス内でバックグラウンド実行） |

- CLI と管理画面は同じ MySQL アドバイザリロックを使うため、同時に実行される楽曲バッチは常に1つです。
- 実行結果は `song_batch_jobs` テーブルに記録され、管理画面で確認できます。保持するのは最新50件までです。
- データソースの URL や Google スプレッドシートの ID は `CHUNISUPPORT_BATCH_*` 環境変数で指定します。API サーバーと CLI の両方の実行環境に設定してください（[設定ファイル・環境変数](docs/configuration.md#楽曲データ収集バッチ)）。

処理の流れ、必須・補完データソース、実行履歴の状態は [楽曲データ収集バッチ](docs/song_batch.md) を参照してください。

## プレイヤーデータ再計算バッチ

`go run ./cmd/recalculate-player-data` は最新マスタに基づいて全プレイヤーのRatingとOVER POWERを再計算します。MySQLアドバイザリロックで多重起動を防ぎ、プレイヤー単位のトランザクションで処理します。運用では07:00前後を避け、cronまたはsystemd timerから1日1回起動してください。

現行版プレイヤーの正常な公式本枠は保持します。`best` / `new` 本枠の件数超過、`slot_order` の未設定・範囲外・重複を検出した場合は、旧版プレイヤーと同様に対象となる通常譜面のスコアから枠を再構築し、RatingとOVER POWERを更新します。この修復はバッチ失敗として扱わず、再構築した推定枠は次回のプレイヤーデータ登録時に公式枠へ置き換わります。候補枠だけの不正では再構築しません。

Playerの既存データは、同一トランザクション内で更新用検索により集約全体をロックし、集約メソッドで変更して `PlayerRepository.Save` で保存します。関連する成績・未解禁曲の読み書きもPlayerロック取得後に行い、コミットまで保持します。ユーザー行も変更する通常登録は「ユーザー → Player → 関連レコード」の順にロックし、バッチと未解禁曲更新はPlayerから開始してユーザー行をロックしません。

`Save` はプロフィール、公式指標、計算レーティング3項目、OVER POWER値、取得日時、更新日時を保存します。ID・所有ユーザー・作成日時は作成後に変更せず、OVER POWER割合は永続化しない派生値です。公式指標と取得日時は通常登録が変更し、公式指標の履歴も同じトランザクションで保存します。再計算は計算レーティングとOVER POWER値、未解禁曲更新はOVER POWER値を変更し、それ以外はロック後の最新値を維持します。再計算と未解禁曲更新では取得日時・更新日時を変更しません。

バッチは一覧取得時とPlayerロック取得後の `data_collected_at` を比較し、異なる場合は競合、行がない場合は削除済みとしてスキップします。取得日時を変えない未解禁曲更新は競合扱いにせず、Playerロック取得後の最新状態で再計算します。再計算値が保存済みの値と同じ場合も成功とし、スロットの変更とPlayer保存は一緒にコミットまたはロールバックします。

MySQLでの並行更新テストは `PLAYER_PERSISTENCE_MYSQL_DSN` に検証用接続先を設定し、`go test ./internal/infra/repository -run TestPlayerPersistenceMySQL -count=1` で実行します。一時データベースの作成・削除と `performance_schema.data_lock_waits` / `data_locks` の参照権限が必要です。テストは作成した専用データベースだけを削除します。
