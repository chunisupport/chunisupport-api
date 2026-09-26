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

設定ファイルは他のバッチと同じく `config.LoadBatchConfig()` で読み込みます（`APP_ENV`、`.config/<APP_ENV>.settings.json`、DB 接続用の環境変数が必要です）。

## 処理フロー

1. **ロック** – 全起動経路で MySQL アドバイザリロック `chunisupport:song-batch` を取得します。通常実行と `--fill-missing-release-date` は競合時にスキップ（終了コード 0）、`--major-update` はエラー終了します。
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
| `internal/infra/songbatch/registry` | 環境変数からデータソース定義を解決 |
| `internal/infra/songbatch/datasource` | ダウンローダー（HTTP、Google Sheets） |
| `internal/infra/songbatch/importer` | JSON 取り込みと検証 |
| `internal/infra/songbatch/consolidation` | データソースごとの統合処理と MySQL 同期の実行 |
| `internal/infra/songbatch/songchart` | SQLite ワークスペースと MySQL 同期処理 |

## トラブルシューティング

- **データソース解決に失敗する**: 必須データソースの環境変数が未設定の可能性があります。
- **必須データソースのダウンロードが失敗する**: 保存済み JSON へのフォールバックは行いません。429/502/503/504 は数回再試行します。mainframe と additional_songs は同じ API キーを直列に呼びます。
- **別プロセスが実行中**: 通常実行は終了コード 0 でスキップします。`--major-update` はエラー終了します。
