# 残対応メモ

## (1) static.db の配置先（go run とバイナリ実行の差異）

- `os.Executable()` は `go run` 実行時に一時ビルドディレクトリを指すため、`filepath.Dir(executablePath)` を使うと `static.db` が意図しない場所に作成される可能性がある。
- README / PR説明では `go run` 時にカレントディレクトリに `static.db` が作成されることを期待しているように見えるため、`go run` を検知してカレントディレクトリを使う実装が推奨される。

## (2) buildChartEntries の冗長なDBアクセス

- `buildChartEntries` 内で `worldsendChartRepo.FindByDisplayID` を呼び出しているが、`GetSongStatsByDisplayID` 冒頭で `songRepo.FindByDisplayID` により `songs` テーブル情報は既に取得済みのため、重複問い合わせになっている。
- `song_id` から `WorldsendChart` のみを取得するメソッド（例: `FindChartBySongID`）を `worldsendChartRepo` に追加し、`songWithCharts.Song.ID` を使って譜面情報だけ取得するようにリファクタリングすると、不要なDBアクセスを削減できる。
