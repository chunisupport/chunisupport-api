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
| `CLOUDFLARE_API_TOKEN` | 対象Zoneの`Cache Purge`権限だけを持つAPIトークン | `...` |
| `CLOUDFLARE_ZONE_ID` | 公開カスタムドメインが属するCloudflare Zone ID | 下表参照 |
| `SONG_SNAPSHOT_PUBLIC_BASE_URL` | オブジェクトを配信するHTTPSカスタムドメイン。パスは指定しない | 下表参照 |

環境ごとの公開先は次のとおりです。開発とステージングは同じ公開先を使用します。

| 環境 | `CLOUDFLARE_ZONE_ID` | `SONG_SNAPSHOT_PUBLIC_BASE_URL` |
| --- | --- | --- |
| 開発・ステージング | `575f883bc4eb7c2d89c56ee987c73873` | `https://static.chunisup-dev.f5.si` |
| beta | `6ef634111241a2dc524992ed7cfcf20f` | `https://static.beta-chunisup.f5.si` |
| 本番 | `c7e970656a686c79cce6fad84c888d2c` | `https://static.chunisupport.net` |

バッチは既存のログ、タイムゾーン、DB接続設定も使用するため、`APP_ENV`、`DB_NAME`、`DB_HOST`、`DB_PORT`、`DB_USER`、`DB_PASS`も必要です。
HTTPサーバー固有の`FIREBASE_CREDENTIALS_FILE`、`TURNSTILE_SECRET_KEY`、`.config/username_forbidden_words.json`は使用しません。

CORS、CDNキャッシュ、公開ポリシーはCloudflare側で設定します。

## 保存するオブジェクト

バケット直下の次の固定キーを、実行のたびに上書きします。

| オブジェクトキー | JSON形式 |
| --- | --- |
| `v1/songs.json` | `GET /v1/songs` と同じ `V1SongsResponse` |
| `v1/worldsend-songs.json` | `GET /v1/worldsend-songs` と同じ `V1WorldsendSongsResponse` |
| `compat/chunirec/2.0/music/showall.json` | `GET /compat/chunirec/2.0/music/showall` と同じchunirec互換楽曲配列 |
| `compat/reiwa/1/chunithm_record/original.json` | `GET /compat/reiwa/1/chunithm_record/original` と同じreiwa互換譜面配列 |

アップロード時の `Content-Type` は `application/json; charset=utf-8` です。
通常楽曲、WORLD'S END楽曲、またはreiwa互換譜面が0件の場合は、異常な空JSONで既存オブジェクトを上書きしないようにバッチを失敗させます。

4種類のJSONはオブジェクトストレージへの書き込み開始前に生成します。ただし、各オブジェクトのPUTは単一トランザクションではありません。
途中のPUTが失敗した場合、それより前のオブジェクトだけが新しい内容になります。次回の正常実行ですべて揃います。

4オブジェクトのPUTがすべて成功した後、対応する公開URLを1回のCloudflare APIリクエストでパージします。
途中のPUTが失敗した場合はパージしません。パージAPIの通信失敗、`429`、`5xx`は最大3回まで再試行し、それでも失敗した場合はバッチを失敗させます。

## スケジュールと監視

- 実行間隔: 6時間
- 正常終了: 終了コード `0`
- 設定、DB取得、JSON生成、オブジェクトストレージへのアップロード、Cloudflareキャッシュパージの失敗: 終了コード `1`
- 既に別プロセスが実行中: 何も変更せず終了コード `1`

スケジューラー側で終了コード `1` を検知し、最後の正常終了から12時間を超えた場合に通知してください。
