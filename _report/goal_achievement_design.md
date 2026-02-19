# 目標（Goal）機能のデータ設計

## 目的

CHUNITHM向け目標機能の永続化設計を、実装初期段階で過剰に複雑化させず、
運用しながら安全に拡張できる形で定義する。

本ドキュメントは、現時点で合意済みの仕様のみを記載する。

---

## 1. 基本方針

- 目標はユーザー単位で管理する。
- 1ユーザーあたり目標上限は100件。
- 目標は「属性（attributes）」と「成果（achievement）」を持つ。
- 属性評価は基本AND。
- 比較は原則 `>=`。
- `diff` は固定序数（整数 0〜4）の単値で扱う。DBのJSONに文字列ではなく整数を保存するためである。序数とその対応は §6 の序列定義に従い、アプリケーション定数として管理する。
- `genre` / `ver` は文字列ではなくマスタIDの単値で保存する。DBのJSONサイズを削減するためであり、複数ジャンル・バージョンを対象にしたい場合は目標を分けて作成することで対応できるため、配列にする必要はない。**IDの数値は順序・序列を表さないため、大小比較・レンジ判定に使用してはならない。**
- `achievement_type` は厳密固定し、対応する `achievement_params` の構造も厳密固定する。
- DBにはJSONで保存するが、アプリ内部（Usecase/Domain）では型安全な構造体に変換して扱う。
- `invert` は表示用のフラグであり、サーバー側の評価ロジックには影響させない。

---

## 2. テーブル設計（MySQL）

マイグレーションは1ファイル、一括で行う。

### 2.1 `achievement_types` マスタテーブル

成果種別のマスタをアプリコードに埋め込まず、DBのテーブルとして管理する。
値はマイグレーションで固定的に投入し、ユーザーによる追加・変更・削除は行わない（読み取り専用マスタ）。

```sql
CREATE TABLE achievement_types (
  id   TINYINT UNSIGNED NOT NULL AUTO_INCREMENT,
  code VARCHAR(30)  NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uq_achievement_types_code (code)
);

-- マイグレーションで固定値を事前投入
INSERT INTO achievement_types (code) VALUES
  ('rank_count'),
  ('score_count'),
  ('avg_score'),
  ('hardlamp_count'),
  ('combolamp_count'),
  ('total_score'),
  ('overpower_value'),
  ('overpower_percent');
```

#### カラム方針

| カラム | 型 | 説明 |
|---|---|---|
| `id` | `TINYINT UNSIGNED` AUTO_INCREMENT PK | `goals` テーブルからの FK 参照に使用する数値キー |
| `code` | `VARCHAR(30)` UNIQUE | アプリ内部・API で使用する識別キー（英小文字スネークケース） |

### 2.2 `goals` テーブル

`achievement_type_id` は `achievement_types.id` への外部キーとし、DBレベルで不正な種別の登録を防ぐ。

```sql
CREATE TABLE goals (
  id                   INT UNSIGNED     NOT NULL AUTO_INCREMENT,
  user_id              INT UNSIGNED     NOT NULL,
  title                VARCHAR(30)      NOT NULL,
  achievement_type_id  TINYINT UNSIGNED NOT NULL,
  achievement_params   JSON             NOT NULL,
  attributes           JSON             NOT NULL,
  invert               BOOLEAN          NOT NULL DEFAULT FALSE,
  PRIMARY KEY (id),
  KEY idx_goals_user_id (user_id),
  CONSTRAINT fk_goals_user_id             FOREIGN KEY (user_id)             REFERENCES users             (id) ON DELETE CASCADE,
  CONSTRAINT fk_goals_achievement_type_id FOREIGN KEY (achievement_type_id) REFERENCES achievement_types (id)
);
```

時間系カラムは設けない。楽曲追加によって達成した・していないが揺らぐ可能性があるため。

#### カラム方針

- `title`: 目標のタイトル。30文字以内の任意の文字列。ユーザーが自由に設定できる。
- `achievement_type_id`: `achievement_types.id` を参照する外部キー。アプリ層では対応する `code` に変換して扱う
- `achievement_params`: 成果種別ごとの可変パラメータ（JSON）
- `attributes`: 対象譜面の絞り込み条件（JSON）
- `invert`: UI表示反転フラグ（評価計算には不使用）

---

## 3. 100件上限の扱い

- 上限は **Usecaseで件数チェック + トランザクション** で担保する。
- 作成トランザクション内の先頭で `SELECT COUNT(*) FROM goals WHERE user_id = ? FOR UPDATE` を実行し、ユーザー行にロックを取得してからチェックする。これにより、同一ユーザーの並列リクエストがシリアライズされ、レースコンディションを防止できる。MySQLのデフォルト分離レベル（REPEATABLE READ）のままで対応可能。
- 101件目作成時は4xx系エラーを返し、専用エラーコードを用意する。
- DBだけで厳密制約化できる場合は将来的に追加検討するが、初期実装はこの方針で進める。

---

## 4. `achievement_type` と `achievement_params`

### `achievement_type` 一覧

正式な一覧は `achievement_types` テーブルの `code` カラムが唯一の真実の情報源（Source of Truth）となる。
アプリ起動時にテーブルをメモリへ読み込み（後述 §12 参照）、Go 定数と照合して型安全なマッピングを行う。

| code | 意味 |
|---|---|
| `rank_count` | 指定rank以上の譜面数 |
| `score_count` | 指定スコア以上の譜面数 |
| `avg_score` | 全譜面の平均スコア |
| `hardlamp_count` | 指定したハードランプの達成数 |
| `combolamp_count` | 指定したコンボランプの達成数 |
| `total_score` | 全譜面のスコア合計 |
| `overpower_value` | 全譜面のOverPower値合計 |
| `overpower_percent` | 全譜面のOverPower値割合 |

### 型整合ルール

- `achievement_type` と `achievement_params` の不一致は不正入力として4xxで返す。
- 受信時に `achievement_type` で分岐して専用構造体へデコードし、バリデーション後に保存する。
- DBの外部キー制約が最終防衛として機能するが、Usecase層でキャッシュを用いた事前検証を行い、ユーザーフレンドリーな4xxエラーを先に返す（DB制約エラーの5xx化を防ぐ）。

### `achievement_params` 仕様

#### 4.1 `rank_count`

`score_count`と同じもので扱う。typeのみが違うようにしたい。

```json
{
  "score": 1000000,
  "count": 100
}
```

- `score`: `integer`
  - 最小値: 0
  - 最大値: 1010000
- `count`: `integer`
  - 最小値: 1
  - 最大値: 対象譜面数
- 判定は「対象譜面のうち、指定score以上を獲得している件数」

ランクはスコアと完全に対応する（SSS=1007500、S=975000など）ため、`rank_count` と `score_count` は同じ構造で扱い、判定ロジックも同様に「指定スコア以上を獲得している件数」とする。UI側でランクとスコアの対応を持ち、`rank_count` の場合は指定されたランクに対応するスコアを内部的に参照して判定するイメージ。

#### 4.2 `score_count`

```json
{
  "score": 1000000,
  "count": 100
}
```

`rank_count`と判定は同じ

#### 4.3 `avg_score`

```json
{
  "score": 1000000
}
```

- `score`: `integer`
  - 最小値: 0
  - 最大値: 1010000
  - スコアは整数で扱う。
  - 平均算出時の端数は小数点以下切り捨て。

#### 4.4 `hardlamp_count`

```json
{
  "lamp": "BRV",
  "count": 100
}
```

- `lamp`: 下表の略称を使用する

  | 略称 | マスタ `clear_lamp_types.name` |
  |---|---|
  | `HRD` | `HARD` |
  | `BRV` | `BRAVE` |
  | `ABS` | `ABSOLUTE` |
  | `CTS` | `CATASTROPHY` |

- `count`: `integer`
  - 最小値: 1
  - 最大値: 対象譜面数

#### 4.5 `combolamp_count`

```json
{
  "lamp": "AJ",
  "count": 100
}
```

- `lamp`: 下表の略称を使用する

  | 略称 | マスタ `combo_lamp_types.name` |
  |---|---|
  | `FC` | `FULL COMBO` |
  | `AJ` | `ALL JUSTICE` |

- `count`: `integer`
  - 最小値: 1
  - 最大値: 対象譜面数

#### 4.6 `total_score`
```json
{
  "total": 100000000
}
```

- `total`: `integer`
  - 最小値: 0
  - 最大値: 対象譜面数 × 1010000

#### 4.7 `overpower_value`

```json
{
  "total": 1000000.000
}
```

- `total`: `number`（小数点以下3桁まで）
  - 最小値: 0
  - 最大値: 対象譜面のOverPower値（理論値）の合計。
  - 各譜面のOP理論値の算出方針:
    - `diff` が指定されている場合: その難易度の譜面のOP値を使う。
    - `diff` が省略され `const` が指定されている場合: `const` の範囲にマッチした各譜面自身のOP値を使う（特定難易度に絞り込まれているため）。
    - `diff` も `const` も省略されている場合: その曲で最も譜面定数が高い難易度のOP値（song APIで取れる `maxop` 値）を使う。

#### 4.8 `overpower_percent`

```json
{
  "total": 100.00
}
```

- `total`: `number`（小数点以下2桁まで）
  - 最小値: 0
  - 最大値: 100
  - 計算方法: `overpower_value` の合計 ÷ 対象譜面のOverPower値（理論値）の合計 × 100。
  - 各譜面のOP理論値の算出方針は `overpower_value`（§4.7）と同一。

---

## 5. `attributes` 仕様

### 5.1 基本

- 値を指定しないフィールドは条件なし（全譜面対象）とみなす。nullにしないのはJSONサイズ削減のためである。
- `attributes` は「全譜面対象」を許可するため、空オブジェクト `{}` を許可する。
- 条件指定時は以下の各フィールドを任意で指定可能。

### 5.2 例

```json
{
  "diff": 3,
  "const": { "min": 14.0, "max": 14.4 },
  "genre": 1,
  "ver": 20
}
```

- `diff` の `3` は `MASTER` に対応する（序数の詳細は §6 参照）。省略時は全難易度対象。
- `const` は数値レンジで指定する。`min <= max` が必須。省略時は定数条件なし。
- `genre` はマスタの `id`（整数・単値）を格納する。複数ジャンルを対象にしたい場合は目標を分けて作成する。省略時は全ジャンル対象。
- `ver` はマスタの `id`（整数・単値）を格納する。複数バージョンを対象にしたい場合は目標を分けて作成する。省略時は全バージョン対象。

### 5.3 各項目

- `diff`: 固定序数（整数）の単値で指定する。文字列ではなく整数を使用する理由はDBのJSONサイズ削減のためである。序数はアプリケーション定数として固定管理する（§6 参照）。有効値は `0〜4`、範囲外は4xxで返す。省略時は全難易度対象。
- `const`: 譜面定数（Chart Constant）のレンジ判定。`min <= max` 必須。いずれも `float64`（小数点以下1桁）で扱う。有効範囲は `1.0 ≤ min, max ≤ 15.9`。省略時は定数条件なし。
- `genre`: マスタの `id`（整数・単値）を格納する。文字列名ではなくIDを使用する理由はDBのJSONサイズ削減のためである。複数ジャンルを対象にしたい場合は目標を分けて作成する。
- `ver`: マスタの `id`（整数・単値）を格納する。理由は `genre` と同様。

### 5.4 マスタ整合

- `genre` / `ver` は起動時プリロード済みのマスタIDのみ許可する。存在しないIDは4xxで返す。
- `genre` / `ver` のIDは存在確認（一致判定）のみに使用する。IDの数値による順序比較・レンジ判定は行ってはならない。
- `diff` は `0〜4` の範囲のみ許可する。範囲外は4xxで返す。

---

## 6. 序列定義（固定）

固定序列はアプリケーション層の定数として持ち、評価時に比較可能な値へ変換する。ランクはフロントエンドで持つため、サーバサイドでは管理しない。

### 難易度序数

`attributes.diff` の値として格納する整数値。DBのIDとは独立した固定定数であり、アプリケーション側で定義する。

| 序数 | 難易度 |
|---|---|
| 0 | `BASIC` |
| 1 | `ADVANCED` |
| 2 | `EXPERT` |
| 3 | `MASTER` |
| 4 | `ULTIMA` |

### ハードランプ序列

`HRD < BRV < ABS < CTS`（略称については §4.4 参照）

### ランク序列

`D < C < B < BB < BBB < A < AA < AAA < S < S+ < SS < SS+ < SSS < SSS+`

---

## 7. `invert` の扱い（UI表示専用）

- `invert` は全 `achievement_type` で保持可能。
- ただしサーバー側の達成判定・集計ロジックには影響させない。
- APIは常に生値（非反転値）を返す。
- 反転表示（例: `threshold - current_avg_score`）はUI側で実施する。

---

## 8. バリデーション方針

- 方針は **A: Goバリデーション中心 + 必要最小限のSchema併用**。
- 境界（Handler/DTO）で形式チェック。
- Usecaseで業務ルールチェック。
  - `achievement_type` の有効性確認（起動時プリロード済みのキャッシュ `AchievementTypesByCode` で検索。存在しなければ4xx）
  - `achievement_type` と `params` の一致
  - `const` の有効範囲チェック（`1.0 ≤ min, max ≤ 15.9`・小数点以下1桁に丸め・`min <= max`）
  - `diff` の範囲チェック（0〜4）
  - `genre` / `ver` のマスタID存在確認（起動時プリロード済みのキャッシュで検索）
  - 100件上限（`SELECT COUNT(*) ... FOR UPDATE` でロックを取ってからチェック。§3 参照）
- 不正入力は4xx系を返す（必要に応じて専用エラーコード追加）。
- DBの `fk_goals_achievement_type_id` 制約が最終防衛として機能し、Usecase検証をすり抜けた場合でもDB整合性は保たれる。

---

## 9. 更新API方針

- 更新は **PATCH** を採用する。
- 部分更新を受け付けるが、保存前には必ず正規化済みの完全データとして検証する。
  - 部分更新はあくまで全ての目標をやりとりしなくて良いというだけで、目標のうちachievement_typeだけが送られるというようなことはない。目標1つ1つは完全な構造体で送られる前提。

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

---

## 12. `achievement_types` マスタのプリロード方針

`achievement_types` は件数が少なく（初期8件）、かつ全リクエストで参照される固定マスタであるため、
`master_data_preload_policy.md` の方針に基づき**起動時にメモリへ読み込む**。

### キャッシュ構造（Goイメージ）

```go
// AchievementType はマスタの1件を表すドメイン型。
type AchievementType struct {
    ID   uint8
    Code string
}

// GoalMasters は目標機能で必要になるマスタ集合です。
// 命名は既存の PlayerDataMasters / SongMasters と同パターン。
// internal/domain/masterdata/masterdata.go に追加する。
type GoalMasters struct {
    AchievementTypes       []*AchievementType
    AchievementTypesByID   map[uint8]*AchievementType  // DB FK 解決用
    AchievementTypesByCode map[string]*AchievementType // バリデーション用
}
```

### ランプ略称の変換テーブル

`hardlamp_count` / `combolamp_count` で使う略称（`HRD`/`BRV` 等）とマスタ名の変換テーブルは  
`internal/info/info.go` に `var` として定義する（AGENTS.md「定数は `info.go` 内に切り出す」方針に従う）。

```go
// HardLampAbbrevToName はAPI略称→マスタ名（clear_lamp_types.name）への変換テーブル
var HardLampAbbrevToName = map[string]string{
    "HRD": "HARD",
    "BRV": "BRAVE",
    "ABS": "ABSOLUTE",
    "CTS": "CATASTROPHY",
}

// ComboLampAbbrevToName はAPI略称→マスタ名（combo_lamp_types.name）への変換テーブル
var ComboLampAbbrevToName = map[string]string{
    "FC": "FULL COMBO",
    "AJ": "ALL JUSTICE",
}
```

Usecase でマスタIDを引く際にこのテーブルを経由する。`info` パッケージは `domain`/`infra` に依存しないため依存方向が守られる。

### プリロードのタイミング

- アプリ起動時（`app.New()` 内）に既存マスタと同様に一括ロードする。
- `achievement_types` テーブルの内容が変わった場合はアプリの再起動が必要（固定マスタのため許容）。

### API一覧エンドポイント

- `GET /internal/master` のレスポンスに `achievement_types` フィールドを追加し、既存マスタと一括返却する。
- レスポンスはキャッシュから直接返却するため、DBアクセスは発生しない。
- 表示名・説明はフロントエンドで i18n 対応するため、API はコードのみを返す。
- **外部API（`/v1`）には公開しない。** 目標機能は `/internal/me/goals` 配下のみで完結するため、APIトークンユーザーが achievement_types を参照する必要はない。

| フィールド | 内容 |
|---|---|
| `code` | 識別キー |

---

## 13. APIエンドポイント定義

目標機能のエンドポイントはすべて `/internal/me/goals` 配下に置く。
認証は Cookie 必須（既存の `/internal/me` グループと同じ方針）。

| メソッド | パス | 概要 |
|---|---|---|
| `GET` | `/internal/me/goals` | 自分の目標一覧を取得 |
| `POST` | `/internal/me/goals` | 目標を新規作成（100件上限チェックあり） |
| `PATCH` | `/internal/me/goals/{id}` | 指定した目標を更新 |
| `DELETE` | `/internal/me/goals/{id}` | 指定した目標を削除 |

### リクエスト／レスポンスの共通構造

#### Goal オブジェクト

```json
{
  "id": 1,
  "title": "マスター14+ 100枚",
  "achievement_type": "score_count",
  "achievement_params": { "score": 1007500, "count": 100 },
  "attributes": { "diff": [3], "const": { "min": 14.0, "max": 14.9 } },
  "invert": false
}
```

- `id` はレスポンスのみ（POST/PATCHのリクエストボディには含まない）。
- `achievement_type` はコード文字列（`achievement_types.code`）で送受信する。DB保存時にIDへ変換するのはInfra層の責務。

### GET `/internal/me/goals`

- 自分が作成した目標を全件返す。
- ソート順: `id` 昇順（作成順）。

### POST `/internal/me/goals`

- Goal オブジェクト（`id` 除く）をリクエストボディとして受け取る。
- 100件上限を超える場合は専用エラーコードで4xxを返す。

### PATCH `/internal/me/goals/{id}`

- 更新対象の goal を Goal オブジェクト（`id` 除く）で完全上書きする（§9 参照）。
- 他ユーザーの goal を指定した場合は404を返す（存在しないとみなす）。

### DELETE `/internal/me/goals/{id}`

- 指定した goal を削除する。
- 他ユーザーの goal を指定した場合は404を返す（存在しないとみなす）。

---

## 13. APIパス設計

目標はユーザー個人のデータであり、認証済みユーザーの個人データ操作が集約されている `/internal/me` 配下に追加する。
他ユーザーへの公開は現時点では行わない。

### 目標CRUD

| メソッド | パス | 説明 |
|---|---|---|
| `GET` | `/internal/me/goals` | 目標一覧取得 |
| `POST` | `/internal/me/goals` | 目標作成 |
| `PATCH` | `/internal/me/goals/:id` | 目標更新（部分更新） |
| `DELETE` | `/internal/me/goals/:id` | 目標削除 |

### 認証

- `/internal/me` 配下の既存エンドポイントと同様、JWT認証（`jwtAuth` ミドルウェア）を適用する。

### レート制限

- 既存の `/internal/me` グループのミドルウェア設定をそのまま引き継ぐ（個別指定なし）。
