# 楽曲データ収集バッチ（song-batch）

外部データソースから楽曲・譜面データを取得し、MySQL の `songs` / `charts` / `worldsend_charts` / `courses` へ統合するバッチです。
以前は `chunisupport-song-batch` リポジトリで管理していましたが、API リポジトリへ統合しました。

## 実行方法

```bash
go run ./cmd/song-batch
```

| フラグ | 説明 |
| --- | --- |
| `--major-update` | 大型アップデート用モード。公式データと追加楽曲だけを取得し、定数更新ルールを適用します |
| `--fill-missing-release-date` | どのデータソースからも日付が得られず、MySQL にも存在しない新規楽曲へ実行日（JST）を `released_at` として補完します |

設定は他のバッチと同じく `config.LoadBatchConfig()` で読み込みます。統合前の song-batch とは次の点が異なります。

- `APP_ENV` が必須です（未設定時に `develop` として扱うことはしません）。`.config/<APP_ENV>.settings.json` と DB 接続用の環境変数も必要なため、cron では API のディレクトリで実行してください。
- ログの出力先は標準出力固定ではなく、設定ファイルの `logging` に従います。
- 実行履歴を `song_batch_jobs` テーブルへ記録するため、マイグレーション `000052` を適用してから新しいバイナリを使ってください。

## 管理画面からの実行

ADMIN は管理画面（`/admin/song-batch`）から、CLI と同じ処理を任意のタイミングで実行できます。API は `POST /internal/admin/song-batch/jobs` で要求を受け付け、API プロセス内でバックグラウンド実行します（仕様は [API.md](API.md) を参照）。

- 画面から選べるのは通常実行・大型アップデートと、リリース日補完の有無です。
- CLI と管理画面は同じアドバイザリロックを使うため、同時に実行される楽曲バッチは常に1つです。管理画面からの要求は、実行中なら `409` で拒否します。
- API の停止時に実行中だったジョブはキャンセルされ、MySQL への同期はロールバックされます。
- 管理画面から実行して成功した場合は、API プロセス内の OVER POWER 分母キャッシュをすぐに無効化します。CLI から実行した場合は、キャッシュの有効期限（10分）で反映されます。
- データソースの環境変数は API プロセスでも必要です。値を変更した場合、管理画面からの実行に反映するには API の再起動が必要です。

## 実行履歴

CLI・管理画面のどちらから実行した場合も、`song_batch_jobs` テーブルへ実行履歴を記録します（ロック競合でスキップした実行は記録しません）。保持するのは最新50件までで、新しいジョブを記録した時点でそれより古いジョブを削除します。管理画面にも最大50件を表示します。

| 状態 | 意味 |
| --- | --- |
| `RUNNING` | 実行中 |
| `SUCCEEDED` | 全データソースを利用して成功 |
| `SUCCEEDED_WITH_WARNINGS` | 補完データソースを除外して成功 |
| `FAILED` | 失敗。MySQL は更新されていません |
| `INTERRUPTED` | プロセス停止などで中断（下記参照） |

実行中にキャンセルされて `INTERRUPTED` になった場合、MySQL への同期はロールバック済みです。
プロセスが異常終了して `RUNNING` のまま残った行は、次に楽曲バッチがロックを取得したときに `INTERRUPTED` へ更新します。この場合は同期のコミット直後に停止した可能性もあるため、同期の成否は不明です。

## 処理フロー

1. **ロック** – 全起動経路（CLI・管理画面）で MySQL アドバイザリロック `chunisupport:song-batch` を取得します。通常実行と `--fill-missing-release-date` は競合時にスキップ（終了コード 0）、`--major-update` はエラー終了します。
2. **データソース解決** – 通常実行は全データソースを解決します。`--major-update` は official と additional_songs だけを対象にします。
3. **データダウンロード** – 実行専用の一時ディレクトリへ毎回取得し、終了後に削除します。過去に取得した JSON は読み込みません。
4. **インポートと検証** – データソースごとのインポーターが JSON を読み取り、構造・必須項目・ソース全体の異常を検証します。
5. **ワークスペース統合** – SQLite ワークスペースを構築し、全ソースのデータを統合します。
6. **MySQL 同期** – 単一トランザクション内で最終テーブルに反映します。必須データソースの解決・取得・解析・検証に失敗した場合は同期しません。

### 必須データソースと補完データソース

| モード | 必須 | 補完 |
| --- | --- | --- |
| 通常 | official、additional_songs、mainframe | st1027、otoge_db |
| 大型更新 | official、additional_songs | なし |

補完データソースが利用できない場合は、ソース名・失敗段階・理由を警告ログへ記録して除外し、他の有効なソースだけで同期します（警告付き成功）。除外されたソースが補完する既存値は維持されます。

## 環境変数

| 変数名 | 用途 |
| --- | --- |
| `CHUNISUPPORT_BATCH_OFFICIAL_URL` | 公式データソースのダウンロード URL |
| `CHUNISUPPORT_BATCH_ST1027_URL` | st1027 データソースのダウンロード URL |
| `CHUNISUPPORT_BATCH_OTOGE_DB_URL` | otoge-db データソースのダウンロード URL（リリース日、WORLD'S END の BPM・ノーツ数・譜面製作者、Wiki ページタイトル補完用） |
| `CHUNISUPPORT_BATCH_WIKI_BASE_URL` | otoge-db の `wikiwiki_url` から除去する Wiki のベース URL（例: `https://wikiwiki.jp/chunithmwiki/`）。`songs.wiki_page_title` が未設定（NULL）の楽曲にのみ保存します。未設定の場合は補完をスキップします |
| `CHUNISUPPORT_BATCH_GOOGLE_CLOUD_API_KEY` | Google Sheets API のキー（mainframe / additional_songs） |
| `CHUNISUPPORT_BATCH_GOOGLE_SHEET_ID` | mainframe のスプレッドシート ID |
| `CHUNISUPPORT_BATCH_ADDITIONAL_SONGS_SHEET_ID` | additional_songs のスプレッドシート ID |
| `CHUNISUPPORT_BATCH_GOOGLE_SPREADSHEET_BASE_URL` | Google Sheets API のベース URL |

データソースの URL とシート ID は実行ごとに環境変数から解決します。`CHUNISUPPORT_BATCH_WIKI_BASE_URL` だけは起動時に一度読み込みます。

## `display_id` の生成

楽曲の `display_id` は、`crypto/rand` で生成した 8 バイトの乱数を 16 進文字列に変換した 16 文字の ID です。既存楽曲を同期する際は、既存の `display_id` を維持します。

## コード構成

| パス | 内容 |
| --- | --- |
| `cmd/song-batch` | CLI のエントリーポイント（フラグ解析、DB 接続、ロック） |
| `internal/domain/songbatch` | 実行モード、データソース種別、必須判定などの業務ルールと、取り込み用のエンティティ・値オブジェクト |
| `internal/usecase/song_batch_usecase.go` | 取得・必須判定・インポート・統合の実行 |
| `internal/usecase/song_batch_job_usecase.go` | ロック取得、実行履歴の記録、管理画面からのバックグラウンド実行 |
| `internal/infra/songbatch/registry` | 環境変数からデータソース定義を解決 |
| `internal/infra/songbatch/datasource` | ダウンローダー（HTTP、Google Sheets） |
| `internal/infra/songbatch/importer` | JSON 取り込みと検証 |
| `internal/infra/songbatch/consolidation` | データソースごとの統合処理と MySQL 同期の実行 |
| `internal/infra/songbatch/songchart` | SQLite ワークスペースと MySQL 同期処理 |

## トラブルシューティング

- **データソース解決に失敗する**: 必須データソースの環境変数が未設定の可能性があります。
- **必須データソースのダウンロードが失敗する**: 保存済み JSON へのフォールバックは行いません。429/502/503/504 は数回再試行します。mainframe と additional_songs は同じ API キーを直列に呼びます。
- **別プロセスが実行中**: 通常実行は終了コード 0 でスキップします。`--major-update` はエラー終了します。
