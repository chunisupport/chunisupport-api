# 残対応メモ

## (1) `GetSongStatsByDisplayID` の WORLD'S END フロー不整合

- `GetSongStatsByDisplayID` は先頭で `songRepo.FindByDisplayID` を呼び出していますが、現在の `SongRepository` 実装は WORLD'S END 楽曲を取得対象から除外しています。
- 一方で `buildChartEntries` には `song.IsWorldsend` を前提とした分岐が残っており、実運用の依存関係では到達しないコードになっています。
- 対応方針としては、`GetChartStatsByDisplayIDAndDifficulty` と同様に WORLD'S END 専用フローを `GetSongStatsByDisplayID` にも用意するか、逆に `buildChartEntries` から到達不能な WORLD'S END 分岐を除去して責務を分離するのが妥当です。
- 現状の論点は「重複DBアクセス」ではなく、「通常楽曲用フローと WORLD'S END 用フローの責務が噛み合っていないこと」にあります。
