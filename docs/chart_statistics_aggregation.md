# 譜面統計バッチの集計仕様

## 概要

`chunisupport-stat-batch` は、プレイヤーの譜面別記録をベスト枠平均レーティング帯ごとに集計し、`chunisupport-api` が参照する統計テーブルを再構築する。通常譜面と WORLD'S END 譜面には同じ集計規則を適用する。

## 集計単位

通常譜面は `chart_id × rating_band_id`、WORLD'S END 譜面は `worldsend_chart_id × rating_band_id` を集計単位とする。

レーティング帯の判定には `best_average_rating` を使用する。下限は包含、上限は除外とし、`ALL` にも同じ記録を加える。`best_average_rating` が `NULL` のプレイヤーは対象外とする。

## 譜面統計

### 未クリアの判定

`clear_lamp_id == 1` を未クリア（FAILED）とする。それ以外をスコア統計上のクリア済み記録として扱う。未クリア記録は記録自体を除外せず、スコアに関する統計からだけ除外する。

### フィールドごとの母集団

| 統計 | 母集団 | FAILEDの扱い |
|---|---|---|
| `player_count` | 全記録 | 含む |
| `clear.*` | 全記録 | `clear.failed` に含む |
| `combo.*` | 全記録 | 含む |
| `average_score` | クリア済み記録 | 除外 |
| `median_score` | クリア済み記録 | 除外 |
| `rank.*` | クリア済み記録 | 除外 |

`rank.*` の合計はクリア済み人数となり、`player_count` と一致しない場合がある。クリア済み人数は `clear.clear`、`clear.hard`、`clear.brave`、`clear.absolute`、`clear.catastrophy` の合計から求められる。

### 平均・中央値

`average_score` はクリア済み記録のスコア合計をクリア済み記録数で除算する。`median_score` はクリア済みスコアを昇順に並べ、奇数件なら中央の値、偶数件なら中央2件の算術平均とする。クリア済み記録が0件の場合は、どちらも `NULL` とする。

### ランク分布

クリア済み記録だけを、次のスコア範囲へ排他的に分類する。

| フィールド | スコア範囲 |
|---|---:|
| `rank.max` | 1,010,000以上 |
| `rank.sssp` | 1,009,000～1,009,999 |
| `rank.sss` | 1,007,500～1,008,999 |
| `rank.ssp` | 1,005,000～1,007,499 |
| `rank.ss` | 1,000,000～1,004,999 |
| `rank.sp` | 990,000～999,999 |
| `rank.s` | 975,000～989,999 |
| `rank.aaal` | 974,999以下 |

### クリアランプ分布

全記録を `clear_lamp_id` により排他的に分類する。

| ID | フィールド |
|---:|---|
| 1 | `clear.failed` |
| 2 | `clear.clear` |
| 3 | `clear.hard` |
| 4 | `clear.brave` |
| 5 | `clear.absolute` |
| 6 | `clear.catastrophy` |

### コンボランプ分布

全記録を `combo_lamp_id` により排他的に分類する。

| 条件 | フィールド |
|---|---|
| `combo_lamp_id == 1` | `combo.none` |
| `combo_lamp_id == 2` | `combo.fc` |
| `combo_lamp_id == 3` かつスコアが1,010,000未満 | `combo.aj` |
| `combo_lamp_id == 3` かつスコアが1,010,000 | `combo.ajc` |

`combo.aj` と `combo.ajc` は重複しない。

## ベスト枠採用率統計

通常譜面だけを対象とし、削除済み楽曲と WORLD'S END 譜面を除外する。読み取りは単一の REPEATABLE READ スナップショット内で行う。

不正利用疑いのないプレイヤーのうち、`best_average_rating` が `NULL` でないプレイヤーを集計対象人数とする。各レーティング帯について、現在の `best` 枠にその譜面を持つ人数を数える。採用率は `best_player_count / eligible_player_count × 100` を小数点以下4桁に丸め、集計対象人数が0人の場合は `NULL` とする。

## 保存先とAPI

集計結果は次のテーブルへ保存する。

- `chart_stats_by_rating_band`
- `worldsend_chart_stats_by_rating_band`
- `chart_best_slot_stats_by_rating_band`

APIレスポンスのフィールド構造はDBカラムに対応する。`average_score` と `median_score` が `NULL` になり得る一方、`player_count` と各分布のカウントは整数で返す。