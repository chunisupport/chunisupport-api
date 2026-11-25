# 譜面統計バッチの集計仕様

## 概要

譜面統計バッチ（`cmd/chart-stats-batch`）は、プレイヤーの譜面別・コース別記録をベスト枠平均レーティング帯ごとに集計し、APIが参照する統計テーブルを再構築する。通常譜面と WORLD'S END 譜面には同じ集計規則を適用する。コースには専用の集計規則を適用する。実行方法と運用は [譜面統計バッチ](chart_stats_batch.md) を参照。

## 集計単位

通常譜面は `chart_id × rating_band_id`、WORLD'S END 譜面は `worldsend_chart_id × rating_band_id`、コースは `course_id × rating_band_id` を集計単位とする。

レーティング帯の判定には `best_average_rating` を使用する。下限は包含、上限は除外とし、`ALL` にも同じ記録を加える。`best_average_rating` が `NULL` のプレイヤーと、不正利用疑い（`users.is_suspicious = 1`）のユーザーのプレイヤーは対象外とする。削除済み楽曲（`songs.is_deleted = 1`）の譜面も対象外とする。これらの条件は譜面統計・WORLD'S END 譜面統計・ベスト枠採用率のすべてに共通である。コース統計もプレイヤーの条件は同じで、削除済みコース（`courses.is_deleted = 1`）を対象外とする。

不審フラグの変更は即時には統計へ反映されず、次回の再集計で反映される。

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

通常譜面だけを対象とし、削除済み楽曲と WORLD'S END 譜面を除外する。

不正利用疑いのないプレイヤーのうち、`best_average_rating` が `NULL` でないプレイヤーを集計対象人数とする。各レーティング帯について、現在の `best` 枠にその譜面を持つ人数を数える。採用率は `best_player_count / eligible_player_count × 100` を小数点以下4桁に丸め、集計対象人数が0人の場合は `NULL` とする。

## コース統計

コース記録（`player_course_records`）を `course_id × rating_band_id` で集計する。コースには公式のランク区分がないため、スコアランク分布は集計しない。

### 未クリアの判定

`is_clear = 0` を未クリアとする。未クリア記録は記録自体を除外せず、スコアに関する統計からだけ除外する。未クリアのコースは途中で終了した分のスコアしか持たないため、クリア済み記録と混ぜると平均・中央値が完走時の実力を表さなくなる。譜面統計が FAILED をスコア統計から除外するのと同じ扱いである。

### フィールドごとの母集団

| 統計 | 母集団 | 未クリアの扱い |
|---|---|---|
| `player_count` | 全記録 | 含む |
| `clear.*` | 全記録 | `clear.failed` に含む |
| `combo.*` | 全記録 | 含む |
| `average_score` | クリア済み記録 | 除外 |
| `median_score` | クリア済み記録 | 除外 |

`player_count` はプレイ済み人数であり、`clear.failed + clear.clear` と一致する。

### 平均・中央値

譜面統計と同じ方法で求める。`average_score` はクリア済み記録のスコア合計をクリア済み記録数で除算し、`median_score` は昇順に並べて奇数件なら中央の値、偶数件なら中央2件の算術平均とする。クリア済み記録が0件の場合は、どちらも `NULL` とする。

### クリア状態の分布

| 条件 | フィールド |
|---|---|
| `is_clear = 0` | `clear.failed` |
| `is_clear = 1` | `clear.clear` |

### コンボランプ分布

全記録を `combo_lamp_id` により排他的に分類する。コースの理論値（3,030,000点）は `combo.aj` に含め、譜面統計の `combo.ajc` に相当する区分は持たない。

| 条件 | フィールド |
|---|---|
| `combo_lamp_id == 1` | `combo.none` |
| `combo_lamp_id == 2` | `combo.fc` |
| `combo_lamp_id == 3` | `combo.aj` |

## 上位%（スコア分布境界）

通常譜面・WORLD'S END譜面・コースのそれぞれについて、全体（`ALL`）帯における上位%（小数点以下2桁、`0.01`〜`100.00`）を逆引きするための境界データを集計する。

- **母集団**: 他の統計と同じプレイヤー・削除除外条件を満たす全プレイ記録（通常譜面・WORLD'S ENDの `FAILED`、コースの未クリア `is_clear = 0` を含む）を対象とし、クリア状況によらず純粋なスコア順で集計する。
- **算出式**: 対象スコア $x$ より厳密に高いスコアの記録数を $H(x)$、母集団の総記録数を $N$ としたとき、競技順位ベースの割合 `(H(x) + 1) / N × 100` を小数第3位で四捨五入し、最小値 `0.01`、最大値 `100.00` の範囲にクランプする。
- **格納形式**: 同じ `0.01%` 値となる連続スコア区間を1つに統合した境界列（先頭1バイトのバージョン + スコア昇順の `min_score (uint32)`・`top_percent_bp (uint16)` ペア列）をバイナリで保存する。

## 読み取りと保存

譜面統計・WORLD'S END 譜面統計・ベスト枠採用率・コース統計・上位%境界の集計元は、単一の REPEATABLE READ スナップショット内で読み取る。

集計結果は次のテーブルへ保存する。全統計テーブルの削除・挿入と `chart_stats_revision.revision` の更新は単一トランザクション内で行うため、再集計中や失敗時も直前の統計が参照される。

- `chart_stats_by_rating_band`
- `worldsend_chart_stats_by_rating_band`
- `chart_best_slot_stats_by_rating_band`
- `course_stats_by_rating_band`
- `chart_score_percentiles`
- `worldsend_chart_score_percentiles`
- `course_score_percentiles`
- `chart_stats_revision`

APIレスポンスのフィールド構造はDBカラムに対応する。`average_score` と `median_score` が `NULL` になり得る一方、`player_count` と各分布のカウントは整数で返す。上位%（`top_percent`）は内部API（`/internal`）のレコードレスポンスのみに付与し、未プレイまたは統計未集計時は `null` を返す。