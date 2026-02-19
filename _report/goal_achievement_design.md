# 目標（Goal）機能のデータ設計（確定版）

## 目的

CHUNITHM向け目標機能の永続化設計を、実装初期段階で過剰に複雑化させず、
運用しながら安全に拡張できる形で定義する。

本ドキュメントは、現時点で合意済みの仕様のみを記載する。

---

## 1. 基本方針

- 目標はユーザー単位で管理する。
- 1ユーザーあたり目標上限は100件。
- 目標は「属性（attributes）」と「成果（achievement）」を持つ。
- 属性評価は基本AND。`genre` / `version` は配列内OR。
- 比較は原則 `>=`。
- `difficulty` は常に大文字（`BASIC`, `ADVANCED`, `EXPERT`, `MASTER`, `ULTIMA`）で扱う。
- `achievement_type` は厳密固定し、対応する `achievement_params` の構造も厳密固定する。
- DBにはJSONで保存するが、アプリ内部（Usecase/Domain）では型安全な構造体に変換して扱う。
- `invert` は表示用のフラグであり、サーバー側の評価ロジックには影響させない。

---

## 2. テーブル設計（MySQL）

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

- `achievement_type`: 成果種別（`rank_count`, `avg_score`, `hardlamp_count`）
- `achievement_params`: 成果種別ごとの可変パラメータ（JSON）
- `attributes`: 対象譜面の絞り込み条件（JSON）
- `invert`: UI表示反転フラグ（評価計算には不使用）

---

## 3. 100件上限の扱い

- 上限は **Usecaseで件数チェック + トランザクション（A+）** で担保する。
- 101件目作成時は4xx系エラーを返し、専用エラーコードを用意する。
- DBだけで厳密制約化できる場合は将来的に追加検討するが、初期実装はA+で進める。

---

## 4. `achievement_type` と `achievement_params`

## `achievement_type` 一覧

- `rank_count`
- `avg_score`
- `hardlamp_count`

### 型整合ルール

- `achievement_type` と `achievement_params` の不一致は不正入力として4xxで返す。
- 受信時に `achievement_type` で分岐して専用構造体へデコードし、バリデーション後に保存する。

### `achievement_params` 仕様

#### 4.1 `rank_count`

```json
{
  "rank": "AA",
  "count": 100
}
```

- `rank`: 以下の列挙値のみ許可
  - `D`, `C`, `B`, `BB`, `BBB`, `A`, `AA`, `AAA`, `S`, `S+`, `SS`, `SS+`, `SSS`, `SSS+`
- `count`: `integer`, `minimum: 1`
- 判定は「対象譜面のうち、指定rank以上を満たす件数」

#### 4.2 `avg_score`

```json
{
  "threshold": 1000000
}
```

- `threshold`: `integer`, `minimum: 0`, `maximum: 1010000`
- スコアは整数で扱う。
- 平均算出時の端数は小数点以下切り捨て。

#### 4.3 `hardlamp_count`

```json
{
  "lamp": "BRAVE",
  "count": 100
}
```

- `lamp`: `BRAVE`, `ABSOLUTE`, `CATASTROPHY`
- `count`: `integer`, `minimum: 1`

---

## 5. `attributes` 仕様

### 5.1 基本

- `attributes` は「全譜面対象」を許可するため、空オブジェクト `{}` を許可する。
- 条件指定時は以下の各フィールドを任意で指定可能。

### 5.2 例

```json
{
  "difficulty": { "min": "MASTER", "max": "ULTIMA" },
  "level": { "min": 14.0, "max": 14.4 },
  "genre": ["ORIGINAL", "東方Project"],
  "version": ["CHUNITHM SUN PLUS", "CHUNITHM LUMINOUS"]
}
```

### 5.3 各項目

- `difficulty`: 固定序列でレンジ判定、`min <= max` 必須
- `level`: 数値レンジ判定、`min <= max` 必須
- `genre`: 配列内OR（完全一致）
- `version`: 配列内OR（完全一致）

### 5.4 マスタ整合

- `genre` / `version` はマスタ値のみ許可する。
- ユーザー手入力は想定しないため、完全一致のみで判定する。

---

## 6. 序列定義（固定）

- 難易度: `BASIC < ADVANCED < EXPERT < MASTER < ULTIMA`
- ハードランプ: `BRAVE < ABSOLUTE < CATASTROPHY`
- ランク: `D < C < B < BB < BBB < A < AA < AAA < S < S+ < SS < SS+ < SSS < SSS+`

固定序列はアプリケーション層の定数として持ち、評価時に比較可能な値へ変換する。

---

## 7. `invert` の扱い（UI表示専用）

- `invert` は全 `achievement_type` で保持可能。
- ただしサーバー側の達成判定・集計ロジックには影響させない。
- APIは常に生値（非反転値）を返す。
- 反転表示（例: `1010000 - avg_score`）はUI側で実施する。

---

## 8. バリデーション方針

- 方針は **A: Goバリデーション中心 + 必要最小限のSchema併用**。
- 境界（Handler/DTO）で形式チェック。
- Usecaseで業務ルールチェック。
  - `achievement_type` と `params` の一致
  - `min <= max`
  - `genre` / `version` のマスタ一致
  - 100件上限
- 不正入力は4xx系を返す（必要に応じて専用エラーコード追加）。

---

## 9. 更新API方針

- 更新は **PATCH** を採用する。
- 部分更新を受け付けるが、保存前には必ず正規化済みの完全データとして検証する。

---

## 10. JSON保存時の正規化

- DB保存時はコンパクトJSON（インデントなし）で保存する。
- 入力原文をそのまま保持せず、バリデーション済み構造体から再エンコードしたJSONを保存する。

---

## 11. 実装時の注意

- `SELECT *` は使用しない。
- N+1を避けるため、集計対象は事前に必要データをまとめて取得する。
- Usecase層で `internal/infra` をimportしない（依存方向を守る）。
- ドメインモデルにJSONタグやDBタグを直接持ち込まない。
