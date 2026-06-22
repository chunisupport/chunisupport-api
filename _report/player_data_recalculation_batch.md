# プレイヤーデータ再計算バッチ 分析レポート

## 1. 概要

本ドキュメントは、本アプリケーションにおける「プレイヤーデータ再計算バッチ」の実現可能性と負荷見積もりについてまとめたものである。再計算の対象は以下の2つとする。

- **レーティング（Rating）**: 単曲レーティングの上位枠集計によるプレイヤーレーティング
- **オーバーパワー（OVER POWER）**: 楽曲ごとの最高OP合算値と全体理論値に対する割合

## 2. 現状の計算フロー

### 2.1 レーティング計算

現在、レーティング計算は `Register()`（プレイヤーデータ登録API）内で実行されている。

**呼び出しチェーン**:

```
Register()                                    [player_data_usecase_impl.go:195]
  └─ calculateAndUpdateRatings()              [player_data_usecase_impl.go:1346]
       ├─ FindByPlayerIDForRating()           [player_record_repository_impl.go:158]
       │    └─ playerRecordRatingQuery        [player_record_repository_impl.go:103]
       │       （8テーブルJOIN、slot IN ('best','best_candidate','new','new_candidate')）
       ├─ CalcRatingStats()                   [rating_service.go:203]
       │    ├─ CalcSingleRating() × N         [rating_service.go:30]
       │    ├─ ベスト枠 上位30曲 平均
       │    ├─ 新曲枠 上位20曲 平均
       │    └─ (bestSum + newSum) / 50
       └─ UpdateCalculatedRatings()           [player_repository_impl.go:164]
            └─ UPDATE players SET calculated_player_rating, best_average_rating, new_average_rating
```

**計算ロジック**（`internal/domain/service/rating_service.go`）:

- `CalcSingleRating(score, chartConst)`: CHUNITHM Wiki準拠の区分線形計算
  - SSS+（1,009,000～）: 譜面定数 + 2.15
  - SSS～D: スコア帯ごとに線形補間
- `CalcRatingStats(records)`: 全レコードからベスト枠30曲＋新曲枠20曲の平均を算出

### 2.2 OVER POWER 計算

**呼び出しチェーン**:

```
applyScores()                                  [player_data_usecase_impl.go:595]
  ├─ GetOverpowerTargetStats()                [player_data_repository.go:54]
  │    （全楽曲の理論値OP合計を計算、分母として使用）
  ├─ FindByPlayerID()                         [player_record_repository_impl.go:148]
  │    （全レコード取得、8テーブルJOIN）
  ├─ ListByPlayerID() (locked_songs)          [player_locked_song_repository.go:10]
  ├─ calculateOverpowerSummaryFromPlayerRecords()  [player_data_usecase_impl.go:1191]
  │    ├─ locked songs を除外フィルタ
  │    ├─ playerRecordsToOverpowerRecords()    [overpower_record_converter.go:13]
  │    └─ CalcOverpowerSummary()              [overpower_summary_service.go:16]
  │         ├─ CalcSingleOverpower() × N      [rating_service.go:99]
  │         ├─ 楽曲ごとの最大OPを合算
  │         └─ CalcOverpowerPercent()         [overpower_summary_service.go:37]
  └─ Save() (player)                          [player_repository.go:29]
       （overpower_value, overpower_percent を更新）
```

**計算ロジック**（`internal/domain/service/rating_service.go`）:

- `CalcSingleOverpower(score, chartConst, comboLampID)`: スコア区分線形計算 + コンボランプ補正
  - FC: +0.5, AJ: +1.0, AJC/理論値: +1.25
  - S以上は0.005単位、S未満は0.05単位で丸め

### 2.3 既存のバッチ基盤

**存在しない。** 以下はコードベースに一切存在しない。

- 定期実行ジョブ（cron / scheduler）
- 全プレイヤーを列挙するリポジトリメソッド
- CLIコマンド

`OverpowerDenominatorProvider` に TTL 10分のインメモリキャッシュがあるのみで、これはバックグラウンドジョブではない。

## 3. バッチ実装に必要な追加要素

| 要素 | 内容 | 優先度 |
|---|---|---|
| 全プレイヤーID取得 | `PlayerRepository.ListAllPlayerIDs()` の追加 | 必須 |
| バッチユースケース | `internal/usecase/` に再計算バッチロジックを追加 | 必須 |
| 実行トリガー | CLIコマンド（`cmd/`以下）または管理APIエンドポイント | 必須 |
| 進行状況ログ | 構造化ログによる進捗・エラー記録 | 推奨 |
| チャンク分割 | プレイヤーをN人ずつに分割して処理 | 推奨 |
| 定期実行 | スケジューラ（システムcron / タスクスケジューラ） | オプション |

## 4. 負荷見積もり

### 4.1 プレイヤーあたりの処理内容

| # | 処理 | 種別 | 負荷 |
|---|---|---|---|
| 1 | `FindByPlayerID()` | DB READ | **重い**（8テーブルJOIN、全レコード） |
| 2 | `ListByPlayerID()` (locked_songs) | DB READ | 軽い（単一テーブル、数件） |
| 3 | `GetOverpowerTargetStats()` | DB READ | 中の上（全楽曲集計、**全プレイヤー共通で1回**） |
| 4 | `CalcRatingStats()` | メモリ計算 | 軽い（O(n log n)、n≦50） |
| 5 | `CalcOverpowerSummary()` | メモリ計算 | 軽い（O(n)、n=全レコード数） |
| 6 | `UpdateCalculatedRatings()` | DB WRITE | 軽い（単一行UPDATE） |
| 7 | `Save()` (player, OP値) | DB WRITE | 軽い（単一行UPDATE） |

### 4.2 理論上の処理時間

仮定:
- プレイヤー数: **N 人**
- 1人あたり平均レコード数: **R 件**（100～500件想定）
- 8テーブルJOIN 1クエリあたり: **T ミリ秒**（インデックス次第、目安5～30ms）

| N（プレイヤー数） | R=300, T=10ms | R=300, T=30ms | R=1000, T=30ms |
|---|---|---|---|
| 1,000 | 約35秒 | 約100秒 | 約3分 |
| 5,000 | 約3分 | 約9分 | 約15分 |
| 10,000 | 約6分 | 約17分 | 約30分 |
| 50,000 | 約30分 | 約1.5時間 | 約2.5時間 |

> 注: 上記はシーケンシャル処理の理論値。DB負荷やネットワーク遅延により変動する。

### 4.3 支配的なボトルネック

1. **`FindByPlayerID()` の8テーブルJOIN**（`player_record_repository_impl.go:60-101`）
   - `player_records` + `charts` + `songs` + `clear_lamp_types` + `combo_lamp_types` + `full_chain_types` + `slots` + `difficulties`
   - レコード数に比例して結果セットが肥大化
   - N+1 問題: N人に対して N 回のJOINクエリが発生

2. **DB接続数とロック**
   - 大量のSELECT + UPDATEが連続するとDBの同時接続を使い切る可能性
   - 他のAPIリクエストと競合する可能性

### 4.4 軽減策

1. **OP分母の共通化**: `GetOverpowerTargetStats()` は全プレイヤーで同一値のため、バッチ開始時に1回だけ計算
2. **チャンク分割**: 100～500人ずつ処理し、DB負荷を平準化
3. **インターバル挿入**: チャンク間にスリープを入れてDB負荷を抑制
4. **Rating専用クエリの活用**: レーティング再計算には `FindByPlayerIDForRating()`（rating対象スロットのみ）で十分であり、OPと同時に再計算しない場合は全レコード取得不要
5. **夜間実行**: ユーザーアクティビティの少ない時間帯にスケジュール
6. **専用バルククエリの検討**: 全プレイヤーのrating対象レコードを1クエリで取得する最適化（実装コスト中）

## 5. 推奨アプローチ

### 5.1 最小構成（推奨）

以下の順で実装し、段階的に改善する。

```
Phase 1: CLIコマンドとして実装
  ├─ PlayerRepository.ListAllPlayerIDs() 追加
  ├─ usecase に RecalculateAllPlayers() 追加
  ├─ cmd/recalculate として CLI エントリポイント作成
  └─ シーケンシャル処理＋構造化ログ

Phase 2: 安定化
  ├─ チャンク分割（100人/チャンク）
  ├─ インターバル挿入（チャンク間1秒）
  ├─ エラーハンドリング強化（1プレイヤー失敗でも継続）
  └─ 結果サマリーログ

Phase 3: 定期実行（必要に応じて）
  └─ システムcron / タスクスケジューラで夜間定期実行
```

### 5.2 疑似コード

```go
func (us *playerDataUsecase) RecalculateAllPlayers(ctx context.Context) error {
    // 1. OP分母を事前計算（全プレイヤー共通）
    targetStats, err := us.playerDataRepo.GetOverpowerTargetStats(ctx, filter)
    if err != nil {
        return err
    }

    // 2. 全プレイヤーIDを取得
    playerIDs, err := us.playerRepo.ListAllPlayerIDs(ctx)
    if err != nil {
        return err
    }

    // 3. チャンク単位で処理
    for chunk := range slices.Chunk(playerIDs, 100) {
        for _, playerID := range chunk {
            if err := us.recalculateSinglePlayer(ctx, playerID, targetStats); err != nil {
                slog.Error("recalculation failed", "player_id", playerID, "error", err)
                continue
            }
        }
        time.Sleep(1 * time.Second) // DB負荷抑制
    }

    return nil
}
```

## 6. 判断サマリー

| 観点 | 評価 |
|---|---|
| 実現可能性 | **高い**（コアロジックは既存、追加実装は少数） |
| 実装工数 | **小**（概算 2～4人日、CLIコマンド＋バッチユースケース） |
| 実行負荷 | **中～大**（プレイヤー数に比例、数千人なら数分、数万人なら十数分～数十分） |
| リスク | DB負荷によるAPI性能低下（夜間実行で回避可能） |
| 運用負荷 | **低**（CLIコマンドとして手動実行 or cron設定のみ） |
