# 概要

未解禁曲の判定をドメインサービス `PlayerLockedSongSet` に集約し、OVER POWER の全計算経路で同じ規則を使うようにしました。closes #228

issue の指摘どおり、`player_locked_songs` の `is_ultima = false`（通常譜面群が未解禁）の扱いが経路ごとに食い違っていました。スコア登録時と未解禁設定変更時だけが `(song_id, 難易度がULTIMA)` の組で照合していたため、通常譜面を未解禁にした楽曲のULTIMA記録が分子にだけ残り、その楽曲が分母からは外れる状態でした。結果として、同じ記録・同じ未解禁設定でも最後に計算した経路でトータルOPが変わり、OP%が実際より高く出ていました。

# 採用した意味

分母・バッチ・引き継ぎ・フロントが揃っている「`is_ultima = false` は楽曲ごと除外」を正としました。通常譜面群が未解禁の楽曲は、譜面単位ではなく楽曲ごと計算対象から外します。

| 未解禁設定 | 通常譜面の記録 | ULTIMA譜面の記録 |
| --- | --- | --- |
| `is_ultima = false` | 除外 | 除外 |
| `is_ultima = true` | 対象 | 除外 |

# 変更内容

- `internal/domain/service/player_locked_song.go` を追加し、判定を `PlayerLockedSongSet` に集約
  - `IsLocked(songID, isUltima)` は譜面単位の判定、`IsSongLocked` / `IsUltimaLocked` は楽曲単位の判定
- 次の5経路をすべてこの型に置き換え
  - スコア登録時 `calculateOverpowerSummaryFromPlayerRecords`
  - 未解禁設定変更時 `recalculatePlayerOverpowerWithTx`
  - 日次再計算バッチ `buildUpdate`
  - データ引き継ぎ `calculateTransferDerivedMetrics`
  - プロフィールOP%の分母 `applyDynamicOverpowerPercent`
- プロフィールOP%の分母は、通常譜面群未解禁なら楽曲最大OPを丸ごと、ULTIMAだけの未解禁なら最大OPと通常譜面最大OPの差だけを減算するように修正
- データ引き継ぎは `official_idx` から楽曲IDを解決してから判定に渡すよう変更（譜面マスタ1件からしかIDを引けないため）
- 各経路に回帰テストを追加し、`PlayerLockedSongSet` 自体のテーブルテストも追加
- `docs/player_locked_songs.md` と `docs/overpower_calculation.md` に規則と集約方針を明記

# 動作確認

- `go test ./...` がすべて成功
- `go vet ./...` が指摘なし
- 新規・変更した Go ファイルのみ `gofmt -s -w` を実行
- `git diff --check` で問題なし

# レビュー観点

- 分子（OP値）と分母（OP%）で規則がずれていないか
- 5経路すべてが同じ規則になっているか
- データ引き継ぎで N+1 を作っていないか（楽曲IDの解決はメモリ上で1回だけ）
