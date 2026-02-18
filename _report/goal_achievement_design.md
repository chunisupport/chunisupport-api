# 目標（Goal）機能のデータ設計メモ

## 目的

CHUNITHM向け目標機能の永続化設計を、実装初期段階で過剰に複雑化させず、
運用しながら安全に拡張できる形で定義する。

前提:

- 目標はユーザー単位で管理する。
- 1ユーザーあたり目標上限は100件。
- 目標は「1つ以上の属性」と「1つの成果（achievement）」を持つ。
- 属性の評価は基本AND、ただし `genre` / `version` は配列内ORで扱う。
- 比較は原則 `>=` のみ。
- 難易度・ハードランプの序列は固定。
- `invert`（未達成数を表示する概念）はJSONではなく `goals` テーブルのカラムで持つ。

## テーブル設計（MySQL）

```sql
CREATE TABLE goals (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id INT UNSIGNED NOT NULL,
  achievement_type VARCHAR(64) NOT NULL,
  achievement_params JSON NOT NULL,
  attributes JSON NOT NULL,
  invert BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_goals_user_id (user_id),
  CONSTRAINT fk_goals_user_id FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
```

### カラム方針

- `achievement_type`: 成果種別（例: `rank_count`, `avg_score`, `hardlamp_count`）
- `achievement_params`: 成果種別ごとの可変パラメータ
- `attributes`: 対象譜面の絞り込み条件
- `invert`: 表示/評価を未達成ベースで反転するかどうか

## JSON仕様

### `attributes` の例

```json
{
  "difficulty": { "min": "MASTER", "max": "ULTIMA" },
  "level": { "min": 14.0, "max": 14.4 },
  "genre": ["ORIGINAL", "東方Project"],
  "version": ["CHUNITHM SUN PLUS", "CHUNITHM LUMINOUS"]
}
```

- `difficulty`: 固定序列でレンジ判定
- `level`: 数値レンジ判定
- `genre` / `version`: 配列内OR判定

### `achievement_params` の例

#### ランク達成数

```json
{
  "rank": "AA",
  "count": 100
}
```

#### 平均スコア

```json
{
  "threshold": 9800000
}
```

#### ハードランプ達成数

```json
{
  "lamp": "BRAVE",
  "count": 100
}
```

## スキーマ定義例（JSON Schema）

### `attributes` のスキーマ

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "difficulty": {
      "type": "object",
      "additionalProperties": false,
      "required": ["min", "max"],
      "properties": {
        "min": { "type": "string", "enum": ["BASIC", "ADVANCED", "EXPERT", "MASTER", "ULTIMA"] },
        "max": { "type": "string", "enum": ["BASIC", "ADVANCED", "EXPERT", "MASTER", "ULTIMA"] }
      }
    },
    "level": {
      "type": "object",
      "additionalProperties": false,
      "required": ["min", "max"],
      "properties": {
        "min": { "type": "number", "minimum": 1.0, "maximum": 15.7 },
        "max": { "type": "number", "minimum": 1.0, "maximum": 15.7 }
      }
    },
    "genre": {
      "type": "array",
      "minItems": 1,
      "items": { "type": "string" }
    },
    "version": {
      "type": "array",
      "minItems": 1,
      "items": { "type": "string" }
    }
  },
  "anyOf": [
    { "required": ["difficulty"] },
    { "required": ["level"] },
    { "required": ["genre"] },
    { "required": ["version"] }
  ]
}
```

### `achievement_params` のスキーマ（`achievement_type` 別）

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "oneOf": [
    {
      "title": "rank_count",
      "type": "object",
      "additionalProperties": false,
      "required": ["rank", "count"],
      "properties": {
        "rank": { "type": "string" },
        "count": { "type": "integer", "minimum": 1 }
      }
    },
    {
      "title": "avg_score",
      "type": "object",
      "additionalProperties": false,
      "required": ["threshold"],
      "properties": {
        "threshold": { "type": "integer", "minimum": 0 }
      }
    },
    {
      "title": "hardlamp_count",
      "type": "object",
      "additionalProperties": false,
      "required": ["lamp", "count"],
      "properties": {
        "lamp": { "type": "string", "enum": ["BRAVE", "ABSOLUTE", "CATASTROPHY"] },
        "count": { "type": "integer", "minimum": 1 }
      }
    }
  ]
}
```

## コード定義例（Go）

```go
package goal

import "encoding/json"

type Goal struct {
	ID                int
	UserID            int
	AchievementType   AchievementType
	AchievementParams json.RawMessage
	Attributes        json.RawMessage
	Invert            bool
}

type AchievementType string

const (
	AchievementTypeRankCount    AchievementType = "rank_count"
	AchievementTypeAvgScore     AchievementType = "avg_score"
	AchievementTypeHardlampCount AchievementType = "hardlamp_count"
)

type RangeString struct {
	Min string `json:"min"`
	Max string `json:"max"`
}

type RangeFloat struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type Attributes struct {
	Difficulty *RangeString `json:"difficulty,omitempty"`
	Level      *RangeFloat  `json:"level,omitempty"`
	Genre      []string     `json:"genre,omitempty"`
	Version    []string     `json:"version,omitempty"`
}

type RankCountParams struct {
	Rank  string `json:"rank"`
	Count int    `json:"count"`
}

type AvgScoreParams struct {
	Threshold int `json:"threshold"`
}

type HardlampCountParams struct {
	Lamp  string `json:"lamp"`
	Count int    `json:"count"`
}
```

## 序列定義（固定）

- 難易度: `BASIC < ADVANCED < EXPERT < MASTER < ULTIMA`
- ハードランプ: `BRAVE < ABSOLUTE < CATASTROPHY`

固定序列はアプリケーション層の定数として持ち、
評価時に数値へマッピングして比較する。

## `invert` の解釈

`invert=false`:

- 達成済み件数を集計して表示する。

`invert=true`:

- 未達成件数を集計して表示する。
- 実装は `対象件数 - 達成件数` で算出する。

`invert` は表示や進捗評価の基本モードに近く、
成果パラメータそのものではないため、JSONではなくテーブルカラムに保持する。

## 実装時の注意

- `difficulty` は常に大文字で扱う（`BASIC` など）。
- `SELECT *` は使用しない。
- 入力JSONは境界でバリデーションする。
- 集計対象は事前に必要データをまとめて取得し、N+1を避ける。
