# 楽曲スナップショットのオブジェクトストレージエクスポート

外部利用者向けの楽曲・譜面一覧JSONをS3互換オブジェクトストレージへ保存するワンショットバッチです。
スケジュール機能はAPIプロセスに持たせず、systemd timerやcronなどから6時間ごとに起動します。

## 実行コマンド

```shell
go run ./cmd/export-song-snapshots
```

本番環境では通常、ビルド済みの `export-song-snapshots` バイナリを実行します。
同時に複数起動された場合はMySQLのAdvisory Lockにより、先にロックを取得した1プロセスだけがエクスポートします。

## オブジェクトストレージ接続設定

次の値をバッチ専用のEnvironmentFile、またはバッチ実行環境のSecretへ設定します。
秘密情報を `.config/<APP_ENV>.settings.json` やGit管理対象ファイルへ記載してはいけません。
本番ではAPIサーバープロセスと共有せず、バッチ実行ユーザーだけが読める権限（EnvironmentFileの場合は`0600`）にしてください。共有`.env`の使用はローカル開発に限定します。

| 環境変数 | 内容 | 例 |
| --- | --- | --- |
| `OBJECT_STORAGE_ENDPOINT_URL` | S3互換APIエンドポイント。HTTPSのオリジンのみ指定 | `https://<ACCOUNT_ID>.r2.cloudflarestorage.com` |
| `OBJECT_STORAGE_ACCESS_KEY_ID` | 対象バケットへ書き込み可能な認証情報のAccess Key ID | `...` |
| `OBJECT_STORAGE_SECRET_ACCESS_KEY` | 同じ認証情報のSecret Access Key | `...` |
| `OBJECT_STORAGE_BUCKET_NAME` | 公開用オブジェクトを保存する既存バケット名 | `chunisupport-public` |

バッチは既存のログ、タイムゾーン、DB接続設定も使用するため、`APP_ENV`、`DB_NAME`、`DB_HOST`、`DB_PORT`、`DB_USER`、`DB_PASS`も必要です。
HTTPサーバー固有の`FIREBASE_CREDENTIALS_FILE`、`TURNSTILE_SECRET_KEY`、`.config/username_forbidden_words.json`は使用しません。

カスタムドメインのURL、CORS、CDNキャッシュ、公開ポリシーはCloudflare側で設定します。バッチは公開URLを参照しません。

## 保存するオブジェクト

バケット直下の次の固定キーを、実行のたびに上書きします。

| オブジェクトキー | JSON形式 |
| --- | --- |
| `v1/songs.json` | `GET /v1/songs` と同じ `V1SongsResponse` |
| `v1/worldsend-songs.json` | `GET /v1/worldsend-songs` と同じ `V1WorldsendSongsResponse` |

アップロード時の `Content-Type` は `application/json; charset=utf-8` です。
通常楽曲またはWORLD'S END楽曲が0件の場合は、異常な空JSONで既存オブジェクトを上書きしないようにバッチを失敗させます。

両方のJSONはオブジェクトストレージへの書き込み開始前に生成します。ただし、2オブジェクトのPUTは単一トランザクションではありません。
後から書き込む `v1/worldsend-songs.json` だけが失敗した場合、`v1/songs.json` は新しい内容、`v1/worldsend-songs.json` は前回の内容になります。次回の正常実行で揃います。

## スケジュールと監視

- 実行間隔: 6時間
- 正常終了: 終了コード `0`
- 設定、DB取得、JSON生成、オブジェクトストレージへのアップロードの失敗: 終了コード `1`
- 既に別プロセスが実行中: 何も変更せず終了コード `1`

スケジューラー側で終了コード `1` を検知し、最後の正常終了から12時間を超えた場合に通知してください。
