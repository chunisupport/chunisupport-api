# 目標評価API 設計書

## 1. 目的

本設計書は、既存の目標（Goal）定義CRUD APIとは分離して、
「目標が達成済みかどうか」をバックエンドで判定し返却するAPIを定義する。

- 対象: 認証済みユーザー本人の目標
- 非対象: 他ユーザーの目標判定
- 対象譜面: **通常譜面のみ（WORLD'S ENDは除外）**

---

## 2. 背景と方針

既存の `/internal/me/goals` は目標定義のCRUDを担う。
達成判定はクライアントで再計算せず、バックエンドで一元管理することで、
クライアント間の判定差異を防ぎ、仕様変更時の追従コストを低減する。

### 方針

1. 目標CRUD APIの責務は維持（破壊的変更を避ける）
2. 判定APIを新設し、判定結果は都度計算で返す
3. 判定不能（プレイヤーデータ未連携など）はエラーで返す
4. レスポンスには「達成可否」だけでなく「不足値」も含める

---

## 3. エンドポイント定義

## 3.1 一覧判定

- **Method**: `GET`
- **Path**: `/internal/me/goals/evaluations`
- **Auth**: Cookie/JWT 必須
- **説明**: 自分の全Goal（最大100件）を評価し、判定結果を返却する

## 3.2 単体判定

- **Method**: `GET`
- **Path**: `/internal/me/goals/:id/evaluation`
- **Auth**: Cookie/JWT 必須
- **説明**: 指定Goal 1件を評価し、判定結果を返却する

---

## 4. レスポンス設計

## 4.1 共通レスポンス要素

- `goal`: 既存Goalレスポンス互換の目標定義
- `evaluation.is_achieved`: 達成可否
- `evaluation.actual`: 実績値（achievement_typeごとの可変構造）
- `evaluation.target`: 目標値（achievement_paramsの正規化表現）
- `evaluation.remaining`: 不足値（達成時は0）
- `evaluation.progress_rate`: 進捗率（0.0〜1.0）
- `evaluation.evaluated_at`: 判定時刻（RFC3339）

## 4.2 一覧レスポンス例

```json
{
  "evaluations": [
    {
      "goal": {
        "id": 1,
        "title": "MASTER 14+ OP合計",
        "achievement_type": "overpower_value",
        "achievement_params": { "total": 9500.0 },
        "attributes": { "diff": 4, "const": { "min": 14.0, "max": 14.9 } },
        "invert": false,
        "created_at": "2026-04-01T12:00:00Z"
      },
      "evaluation": {
        "is_achieved": false,
        "actual": { "total": 9123.456 },
        "target": { "total": 9500.0 },
        "remaining": { "total": 376.544 },
        "progress_rate": 0.9604,
        "evaluated_at": "2026-04-01T12:34:56Z"
      }
    }
  ]
}
```

## 4.3 単体レスポンス例

```json
{
  "goal": {
    "id": 5,
    "title": "AJ 100譜面",
    "achievement_type": "combolamp_count",
    "achievement_params": { "lamp": "AJ", "count": 100 },
    "attributes": { "diff": [3,4] },
    "invert": false,
    "created_at": "2026-03-10T00:00:00Z"
  },
  "evaluation": {
    "is_achieved": true,
    "actual": { "count": 117 },
    "target": { "count": 100 },
    "remaining": { "count": 0 },
    "progress_rate": 1.0,
    "evaluated_at": "2026-04-01T12:34:56Z"
  }
}
```

---

## 5. achievement_type別 判定仕様

既存のachievement_typeに準拠し、`remaining` と `progress_rate` は以下の共通式で計算する。

- `invert: false`（目標値以上を目指す）
  - 判定: `actual >= target`
  - `remaining`: `max(target - actual, 0)`
  - `progress_rate`: `min(actual / target, 1.0)`（`target=0` の場合は `1.0` 扱い）
- `invert: true`（目標値以下を目指す）
  - 判定: `actual <= target`
  - `remaining`: `max(actual - target, 0)`（= 目標値をどれだけ上回っているか）
  - `progress_rate`: `min(target / max(actual, 1), 1.0)`（`actual=0` かつ判定達成時は `1.0`）

`avg_score` の `invert: true` では、UI表示用の反転値を理論最大値との差分ではなく、必ず `threshold - current_avg_score`（= `target.score - actual.score`）基準で扱う。

- `rank_count` / `score_count`
  - actual: `{ "count": int }`
  - 判定: `invert=false` の場合は `actual.count >= target.count`、`invert=true` の場合は `actual.count <= target.count`
- `avg_score`
  - actual: `{ "score": int }`（平均の小数以下は既存仕様に合わせて切り捨て）
  - 判定: `invert=false` の場合は `actual.score >= target.score`、`invert=true` の場合は `actual.score <= target.score`
  - `remaining`: `invert=false` は `max(target.score - actual.score, 0)`、`invert=true` は `max(actual.score - target.score, 0)`
- `hardlamp_count` / `combolamp_count`
  - actual: `{ "count": int }`
  - 判定: `invert=false` の場合は `actual.count >= target.count`、`invert=true` の場合は `actual.count <= target.count`
- `total_score`
  - actual: `{ "total": int }`
  - 判定: `invert=false` の場合は `actual.total >= target.total`、`invert=true` の場合は `actual.total <= target.total`
- `overpower_value`
  - actual: `{ "total": float64 }`
  - 判定: `invert=false` の場合は `actual.total >= target.total`、`invert=true` の場合は `actual.total <= target.total`
- `overpower_percent`
  - actual: `{ "percent": float64 }`
  - 判定: `invert=false` の場合は `actual.percent >= target.percent`、`invert=true` の場合は `actual.percent <= target.percent`

---

## 6. エラー仕様

既存Goal APIのエラーコードと整合させる。

- `goal_not_found` (404)
- `goal_evaluation_unavailable` (409)
  - 例: プレイヤーデータ未連携、必要な判定元データが存在しない
- `internal_error` (500)

`goal_evaluation_unavailable` は「入力不正」ではなく「評価前提の不足」を示す。

---

## 7. アーキテクチャ設計（Clean Architecture準拠）

## 7.1 Usecase

新規: `GoalEvaluationUsecase`

- `ListEvaluations(ctx, userID)`
- `GetEvaluation(ctx, userID, goalID)`

責務:

- Goal取得
- 判定対象データ取得
- achievement_type別評価器への委譲
- DTO整形

## 7.2 Domain

新規: `goal_evaluator`（strategy）

- achievement_typeごとの評価関数を定義
- 判定結果を共通構造に正規化

## 7.3 Repository

新規/拡張:

- `FindGoalEvaluationDatasetByUserID(ctx, userID)`
  - 通常譜面のみのレコードを一括取得
  - GoalごとのN+1クエリを回避

---

## 8. パフォーマンス設計

## 8.1 目標

- 100 goals/user を想定
- 1リクエストあたりDBクエリ回数: **3回以内**を目標
  - goals一覧
  - 判定用データ一括取得（records + charts + songs必要列）
  - 必要に応じた補助マスタ

## 8.2 N+1回避

禁止:

- Goal 1件ごとにDB集計クエリ実行（最大100回）

推奨:

- 一括取得してメモリ上で100件評価

## 8.3 将来拡張

- 高負荷時は短TTLキャッシュ（ユーザー単位）を導入可能な構造にする
- ただし判定の正は常に「再計算可能」であることを維持

---

## 9. APIバージョニングと互換性

- 既存 `/internal/me/goals` のレスポンス形式は変更しない
- 新規APIとして追加し、既存クライアントへの破壊的影響を回避

---

## 10. 実装タスク分解（提案）

1. DTO追加（evaluation response）
2. Usecase interface/impl追加
3. Repositoryの一括取得クエリ実装（通常譜面限定）
4. evaluator strategy実装（achievement_type網羅）
5. Handler追加 + Router組み込み
6. APIエラーコード追加（`goal_evaluation_unavailable`）
7. API.md追記
8. テスト（Usecase中心）

---

## 11. テスト観点

- 正常系
  - 8種achievement_typeそれぞれ達成/未達
  - remaining計算、progress_rate丸め
- 準正常系
  - goal 0件
  - 100件
- 異常系
  - `goal_not_found`
  - `goal_evaluation_unavailable`
- 性能系
  - goals=100でN+1が発生していないこと（クエリ回数アサート）

---

## 12. 未決事項（将来）

- 判定結果永続化（`achieved_at`, `last_evaluated_at`）の要否
- 一覧APIにページングを設けるか
- progress_rateの丸め桁規約
