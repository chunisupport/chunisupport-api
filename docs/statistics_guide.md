# 統計機能 - 実装ガイド

## 概要

譜面ごとのランク・ランプ統計をレーティング帯別に集計する機能です。
APIを通じて統計データを提供することで、プレイヤーが自分の実力帯での達成状況を把握できます。

## 設計思想

### なぜ事前集計するのか

リアルタイム集計はデータベースへの負荷が高いため、バッチ処理による事前集計（キャッシュ）方式を採用しています。

- **N+1問題の回避**: 楽曲一覧取得時に譜面ごとに集計クエリを実行すると性能が劣化します
- **複雑な集計の事前計算**: レーティング帯別・ランク別・ランプ別の多次元集計を事前に行います
- **API応答速度の最適化**: プリ集計されたデータを返すだけなので高速です

### レーティング帯の基準

**ベスト枠平均レーティング（`best_average_rating`）** を基準とします。

- **なぜ公式レーティングではないのか**: 新曲枠の影響で短期的に変動するため、実力帯の指標として不安定
- **なぜベスト枠平均なのか**: ベスト30の平均は安定しており、プレイヤーの実力帯を正確に表します
- **範囲**: 15.0～17.6（0.1刻み）+ 17.7以上（一括）= 計28区分
- **15.0未満は除外**: 統計として意味のある母数を確保するため、下限を15.0に設定

### 対象譜面の制限

**譜面定数10.0以上の譜面のみ** を統計対象とします。

- **理由**: 低難度譜面は実力帯別の統計を取る意義が薄いため
- **実装**: 譜面定数10.0未満の譜面では `statistics` フィールドが `null` になります

## データモデル

### テーブル構造

```sql
CREATE TABLE chart_statistics (
    chart_id MEDIUMINT UNSIGNED NOT NULL,      -- 譜面ID
    rating_tier SMALLINT NOT NULL,             -- レーティング帯（150-177）
    
    -- ランク別人数
    rank_s_count INT UNSIGNED NOT NULL,        -- S (975,000-989,999)
    rank_s_plus_count INT UNSIGNED NOT NULL,   -- S+ (990,000-999,999)
    rank_ss_count INT UNSIGNED NOT NULL,       -- SS (1,000,000-1,004,999)
    rank_ss_plus_count INT UNSIGNED NOT NULL,  -- SS+ (1,005,000-1,007,499)
    rank_sss_count INT UNSIGNED NOT NULL,      -- SSS (1,007,500-1,008,999)
    rank_sss_plus_count INT UNSIGNED NOT NULL, -- SSS+ (1,009,000+)
    
    -- ランプ別人数
    lamp_aj_count INT UNSIGNED NOT NULL,       -- ALL JUSTICE
    lamp_fc_count INT UNSIGNED NOT NULL,       -- FULL COMBO
    lamp_other_count INT UNSIGNED NOT NULL,    -- その他
    
    total_count INT UNSIGNED NOT NULL,         -- 合計人数（検算用）
    updated_at TIMESTAMP NOT NULL,
    
    PRIMARY KEY (chart_id, rating_tier),
    INDEX idx_chart_statistics_chart_id (chart_id)
);
```

### レーティング帯の表現

整数で10倍した値を保存します：

| レーティング帯 | 保存値 | 説明 |
|--------------|--------|------|
| 15.0 | 150 | 15.00-15.09 |
| 15.1 | 151 | 15.10-15.19 |
| ... | ... | ... |
| 17.6 | 176 | 17.60-17.69 |
| 17.7+ | 177 | 17.70以上すべて |

## API仕様

### レスポンス例

```json
{
  "charts": {
    "MASTER": {
      "const": 14.5,
      "statistics": {
        "15.0": {
          "rank": {"s": 10, "s_plus": 25, "ss": 40, "ss_plus": 30, "sss": 20, "sss_plus": 5},
          "lamp": {"aj": 15, "fc": 45, "other": 70}
        },
        "17.7+": {
          "rank": {"s": 1, "s_plus": 3, "ss": 5, "ss_plus": 8, "sss": 12, "sss_plus": 20},
          "lamp": {"aj": 25, "fc": 15, "other": 9}
        }
      }
    },
    "BASIC": {
      "const": 8.5,
      "statistics": null
    }
  }
}
```

### フィールド説明

- `statistics`: 譜面定数10.0以上の譜面にのみ存在（10.0未満は `null`）
- 統計データが存在する場合、全レーティング帯（15.0~17.7+）のデータを含みます
- キー: "15.0", "15.1", ..., "17.6", "17.7+"
- `rank`: ランク別人数（S, S+, SS, SS+, SSS, SSS+）
- `lamp`: ランプ別人数（AJ, FC, その他）

## バッチ処理の実装（別リポジトリ）

### 集計ロジック

```sql
INSERT INTO chart_statistics (
    chart_id, rating_tier,
    rank_s_count, rank_s_plus_count, rank_ss_count,
    rank_ss_plus_count, rank_sss_count, rank_sss_plus_count,
    lamp_aj_count, lamp_fc_count, lamp_other_count,
    total_count, updated_at
)
SELECT
    pr.chart_id,
    -- レーティング帯の計算（ベスト枠平均を使用）
    CASE
        WHEN p.best_average_rating >= 17.7 THEN 177
        ELSE FLOOR(p.best_average_rating * 10)
    END as rating_tier,
    
    -- ランク別集計
    SUM(CASE WHEN pr.score >= 975000 AND pr.score < 990000 THEN 1 ELSE 0 END),
    SUM(CASE WHEN pr.score >= 990000 AND pr.score < 1000000 THEN 1 ELSE 0 END),
    SUM(CASE WHEN pr.score >= 1000000 AND pr.score < 1005000 THEN 1 ELSE 0 END),
    SUM(CASE WHEN pr.score >= 1005000 AND pr.score < 1007500 THEN 1 ELSE 0 END),
    SUM(CASE WHEN pr.score >= 1007500 AND pr.score < 1009000 THEN 1 ELSE 0 END),
    SUM(CASE WHEN pr.score >= 1009000 THEN 1 ELSE 0 END),
    
    -- ランプ別集計
    SUM(CASE WHEN pr.combo_lamp_id = 3 THEN 1 ELSE 0 END), -- AJ
    SUM(CASE WHEN pr.combo_lamp_id = 2 THEN 1 ELSE 0 END), -- FC
    SUM(CASE WHEN pr.combo_lamp_id = 1 THEN 1 ELSE 0 END), -- その他
    
    COUNT(*), NOW()
FROM
    player_records pr
    INNER JOIN players p ON pr.player_id = p.id
    INNER JOIN charts c ON pr.chart_id = c.id
WHERE
    c.const >= 10.0                      -- 譜面定数10.0以上
    AND p.best_average_rating >= 15.0    -- ベスト枠平均15.0以上
    AND p.best_average_rating IS NOT NULL
GROUP BY
    pr.chart_id, rating_tier
ON DUPLICATE KEY UPDATE
    rank_s_count = VALUES(rank_s_count),
    rank_s_plus_count = VALUES(rank_s_plus_count),
    rank_ss_count = VALUES(rank_ss_count),
    rank_ss_plus_count = VALUES(rank_ss_plus_count),
    rank_sss_count = VALUES(rank_sss_count),
    rank_sss_plus_count = VALUES(rank_sss_plus_count),
    lamp_aj_count = VALUES(lamp_aj_count),
    lamp_fc_count = VALUES(lamp_fc_count),
    lamp_other_count = VALUES(lamp_other_count),
    total_count = VALUES(total_count),
    updated_at = VALUES(updated_at);
```

### 実行頻度

- **推奨**: 日次（深夜AM4:00など）
- **初回実行**: 全譜面の統計を作成
- **2回目以降**: UPSERT により既存データを更新

## 実装状況

### 完了済み

- ✅ データベーステーブル設計 (`migration/mysql/000002_chart_statistics.up.sql`)
- ✅ ドメインエンティティ (`internal/domain/entity/chart_statistics.go`)
- ✅ リポジトリインターフェース (`internal/domain/repository/chart_statistics_repository.go`)
- ✅ DTO定義 (`internal/dto/api_v1/chart_statistics_dto.go`)
- ✅ API仕様書更新 (`docs/API.md`)
- ✅ ER図更新 (`docs/er_diagram.puml`)
- ✅ ドメインモデル仕様書更新 (`docs/domain_model_specification.md`)

### 未実装（次フェーズ）

1. **インフラ層実装**
   - `internal/infra/models/chart_statistics.go`: データベースモデル
   - `internal/infra/repository/chart_statistics_repository_impl.go`: リポジトリ実装

2. **ハンドラー層改修**
   - 楽曲取得API（`GET /v1/songs/:songId?content=full`）で統計データを結合
   - 譜面定数10.0未満は `statistics: null` を保証

3. **バッチ処理**（別リポジトリ）
   - 統計集計バッチの実装
   - cron設定とデプロイ

## パフォーマンス考慮事項

### N+1問題の回避

```go
// ✅ 良い例: 一括取得
chartIDs := extractChartIDs(songs)
statsList, _ := statsRepo.FindByChartIDs(ctx, exec, chartIDs)

// ❌ 悪い例: ループ内で個別取得
for _, chart := range charts {
    stats, _ := statsRepo.FindByChartID(ctx, exec, chart.ID)
}
```

### インデックスの活用

- `PRIMARY KEY (chart_id, rating_tier)`: 複合主キーで一意性を保証
- `INDEX idx_chart_statistics_chart_id`: 譜面IDでの検索を高速化

### データ整合性

- `total_count` カラムで検算可能（ランク合計 = ランプ合計 = total_count）
- 外部キー制約により譜面削除時に統計も自動削除

## 将来の拡張可能性

### 追加可能な統計指標

- 平均スコア（`average_score DECIMAL(8,2)`）
- スコア中央値（`median_score MEDIUMINT UNSIGNED`）
- 標準偏差（`score_stddev DECIMAL(8,2)`）

### レーティング帯の範囲変更

レーティング帯の範囲を変更する場合は以下を更新：

1. バッチ処理のSQL（CASE文）
2. `ChartStatistics.IsValidRatingTier()` メソッド
3. API仕様書の記載
4. 本ドキュメント

### 実装時の注意事項

- テーブルのカラム追加は互換性を保つ（`NULL` 可能または `DEFAULT` 値を設定）
- 既存の統計データを維持したまま新指標を追加する設計にする
- バッチ処理は冪等性を保つ（何度実行しても同じ結果になる）
