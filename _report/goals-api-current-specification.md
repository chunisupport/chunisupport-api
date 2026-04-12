# goals API 現行仕様書

この文書は、`chunisupport-api` の現行実装から読み取れる `goals` API の仕様を整理したものです。
設計案や将来仕様ではなく、以下のコードを根拠にしています。

- `internal/app/router.go`
- `internal/app/handler/api_internal/goal_handler.go`
- `internal/dto/api_internal/goal_dto.go`
- `internal/usecase/goal_usecase.go`
- `internal/usecase/goal_usecase_impl.go`
- `internal/infra/repository/goal_repository_impl.go`
- `migration/mysql/000005_add_goals.up.sql`
- `migration/mysql/000010_add_updated_at_to_song_tables.up.sql`

## 1. 概要

`goals` API は、ログイン済みユーザーが自分の目標を CRUD するための内部APIです。

提供エンドポイント:

- `GET /internal/me/goals`
- `POST /internal/me/goals`
- `PUT /internal/me/goals/:id`
- `DELETE /internal/me/goals/:id`

全エンドポイントで Firebase ID トークンによる Bearer 認証が必須です。
ルーティング上は `/internal/me` グループ配下にあり、`firebaseAuth` ミドルウェアが適用されています。

## 2. データモデル

DB上の `goals` テーブルは以下の構造です。

```sql
CREATE TABLE goals (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id INT UNSIGNED NOT NULL,
  title VARCHAR(30) NOT NULL,
  achievement_type_id TINYINT UNSIGNED NOT NULL,
  achievement_params JSON NOT NULL,
  attributes JSON NOT NULL,
  invert BOOLEAN NOT NULL DEFAULT FALSE,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_goals_user_id (user_id),
  CONSTRAINT fk_goals_user_id FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  CONSTRAINT fk_goals_achievement_type_id FOREIGN KEY (achievement_type_id) REFERENCES achievement_types (id) ON DELETE RESTRICT
);
```

補足:

- `updated_at` はありません。更新後も返却される日時は `created_at` のみです。
- 一覧取得順のために `idx_goals_user_created_id(user_id, created_at, id)` が追加されています。
- `achievement_type` はDBに文字列では保存されず、`achievement_type_id` で保存されます。
- `achievement_params` と `attributes` は JSON 型で保存されます。

## 3. 認証と対象範囲

- 対象は常に「自分自身の goal」のみです。
- `GoalHandler` はコンテキストの `userEntity` からユーザーIDを取得します。
- `PUT` と `DELETE` は `:id` を受け取りますが、検索条件は常に `id AND user_id` です。
- 他ユーザーの goal を指定しても `goal_not_found` になります。

## 4. エンドポイント一覧

### 4.1 GET `/internal/me/goals`

自分の目標一覧を取得します。

- 認証: 必須
- ステータス: `200 OK`
- ページング: なし

レスポンス:

```json
{
  "goals": [
    {
      "id": 1,
      "title": "MASTER 14.0以上を1譜面AJ",
      "achievement_type": "combolamp_count",
      "achievement_params": {
        "lamp": "AJ",
        "count": 1
      },
      "attributes": {
        "diff": 4,
        "const": {
          "min": 14.0,
          "max": 15.9
        }
      },
      "invert": false,
      "created_at": "2026-04-12T10:00:00+09:00"
    }
  ]
}
```

取得順:

- `created_at ASC`
- 同一秒内では `id ASC`

### 4.2 POST `/internal/me/goals`

目標を新規作成します。

- 認証: 必須
- ステータス: `201 Created`
- 上限: 1ユーザーあたり100件

リクエスト例:

```json
{
  "title": "MASTER 14.0以上を1譜面AJ",
  "achievement_type": "combolamp_count",
  "achievement_params": {
    "lamp": "AJ",
    "count": 1
  },
  "attributes": {
    "diff": 4,
    "const": {
      "min": 14.0,
      "max": 15.9
    }
  },
  "invert": false
}
```

レスポンスは単一 goal オブジェクトです。形状は一覧内要素と同じです。

### 4.3 PUT `/internal/me/goals/:id`

既存目標を更新します。

- 認証: 必須
- ステータス: `200 OK`
- `:id` は `uint32` として解釈可能な10進数のみ有効
- 更新方法: 部分更新ではなく、`title` / `achievement_type` / `achievement_params` / `attributes` / `invert` の完全上書きです

リクエスト形状は `POST` と同じです。
レスポンスは更新後の単一 goal オブジェクトです。

### 4.4 DELETE `/internal/me/goals/:id`

目標を削除します。

- 認証: 必須
- ステータス: `204 No Content`
- `:id` は `uint32` として解釈可能な10進数のみ有効

レスポンスボディはありません。

## 5. JSON形状の詳細

### 5.1 GoalRequest

リクエストボディのトップレベル形状は以下です。

```json
{
  "title": "string",
  "achievement_type": "string",
  "achievement_params": {},
  "attributes": {},
  "invert": false
}
```

各フィールド:

| フィールド | 型 | 必須 | 説明 |
| --- | --- | --- | --- |
| `title` | string | 必須 | 目標タイトル |
| `achievement_type` | string | 必須 | 達成条件の種別コード |
| `achievement_params` | object | 必須 | `achievement_type` ごとの可変JSON |
| `attributes` | object | 任意 | 対象譜面の絞り込み条件 |
| `invert` | boolean | 任意 | 表示用フラグ。未指定時は `false` |

重要な仕様:

- `Content-Type: application/json` が必須です。
- JSONデコードは strict です。未知のトップレベルキーは `bad_request` になります。
- `attributes` を省略した場合、内部的には `{}` として扱われます。
- `achievement_params` は必須で、省略不可です。

### 5.2 GoalResponse

レスポンスの goal オブジェクト形状は以下です。

```json
{
  "id": 1,
  "title": "MASTER 14.0以上を1譜面AJ",
  "achievement_type": "combolamp_count",
  "achievement_params": {
    "lamp": "AJ",
    "count": 1
  },
  "attributes": {
    "diff": 4,
    "const": {
      "min": 14.0,
      "max": 15.9
    }
  },
  "invert": false,
  "created_at": "2026-04-12T10:00:00+09:00"
}
```

各フィールド:

| フィールド | 型 | 説明 |
| --- | --- | --- |
| `id` | number | goal ID |
| `title` | string | タイトル。保存時に trim 済み |
| `achievement_type` | string | マスタ逆引きしたコード |
| `achievement_params` | object | 保存済みJSONをデコードしたもの |
| `attributes` | object | 保存済みJSONをデコードしたもの |
| `invert` | boolean | 保存値そのまま |
| `created_at` | string | RFC3339形式文字列 |

補足:

- `achievement_params` と `attributes` はレスポンス時に `map[string]any` として返されます。
- 数値はJSON上の通常の number として返ります。
- `created_at` は `time.Time.Format("2006-01-02T15:04:05Z07:00")` で文字列化されます。

## 6. `achievement_type` 仕様

現行実装で有効なのは以下の8種類です。

- `rank_count`
- `score_count`
- `avg_score`
- `hardlamp_count`
- `combolamp_count`
- `total_score`
- `overpower_value`
- `overpower_percent`

この文字列は大文字小文字を含めてマスタ一致が必要です。
例えば `Score_Count` や `SCORE_COUNT` は無効です。

## 7. `achievement_params` 仕様

`achievement_params` は `achievement_type` ごとに厳密な形状を要求されます。
不要キーの混入も許可されません。

### 7.1 `rank_count`

```json
{
  "score": 1009000,
  "count": 10
}
```

条件:

- キーは `score` と `count` の2つだけ
- `score` は整数、`0 <= score <= 1010000`
- `count` は整数、`count >= 1`

注意:

- 名前は `rank_count` ですが、現行実装では `rank` 文字列ではなく `score` 閾値を受け取ります。
- つまり「指定スコア以上の譜面数」を数えるための形状です。

### 7.2 `score_count`

```json
{
  "score": 1000000,
  "count": 25
}
```

条件は `rank_count` と同じです。

### 7.3 `avg_score`

```json
{
  "score": 1005000
}
```

条件:

- キーは `score` のみ
- `score` は整数、`0 <= score <= 1010000`

### 7.4 `hardlamp_count`

```json
{
  "lamp": "HRD",
  "count": 5
}
```

条件:

- キーは `lamp` と `count` の2つだけ
- `count` は整数、`count >= 1`
- `lamp` は以下のいずれか
  - `HRD`
  - `BRV`
  - `ABS`
  - `CTS`

### 7.5 `combolamp_count`

```json
{
  "lamp": "AJ",
  "count": 3
}
```

条件:

- キーは `lamp` と `count` の2つだけ
- `count` は整数、`count >= 1`
- `lamp` は以下のいずれか
  - `FC`
  - `AJ`

### 7.6 `total_score`

```json
{
  "total": 123456789
}
```

条件:

- キーは `total` のみ
- `total` は整数
- `total >= 0`

実装上、Go側では `int64` として受けています。

### 7.7 `overpower_value`

```json
{
  "total": 1234.567
}
```

条件:

- キーは `total` のみ
- `total >= 0`
- 小数第3位まで許可

### 7.8 `overpower_percent`

```json
{
  "total": 98.765
}
```

条件:

- キーは `total` のみ
- `0 <= total <= 100`
- 小数第3位まで許可

## 8. `attributes` 仕様

`attributes` は対象譜面の絞り込み条件です。
許可キーは以下のみです。

- `diff`
- `const`
- `genre`
- `ver`

未知キーが1つでもあると `goal_invalid_attributes` です。

### 8.1 全体形状

```json
{
  "diff": 4,
  "const": {
    "min": 14.0,
    "max": 15.9
  },
  "genre": [1, 2],
  "ver": 20
}
```

空オブジェクト `{}` は有効です。

### 8.2 `diff`

難易度IDです。

許可される入力形状:

- 単一整数
- 整数配列

例:

```json
{ "diff": 4 }
```

```json
{ "diff": [3, 4] }
```

条件:

- 空配列は不可
- `null` は不可
- 小数は不可
- マスタに存在する difficulty ID のみ可

正規化仕様:

- 昇順ソートされます
- 重複は除去されます
- 1件だけならスカラーに正規化されます

例:

入力:

```json
{ "diff": [4, 4] }
```

保存・返却:

```json
{ "diff": 4 }
```

### 8.3 `genre`

ジャンルIDです。

許可される入力形状:

- 単一整数
- 整数配列

正規化仕様は `diff` と同じです。
存在する genre ID のみ許可されます。

### 8.4 `ver`

バージョンIDです。

許可される入力形状:

- 単一整数
- 整数配列

正規化仕様は `diff` と同じです。
存在する version ID のみ許可されます。

重要な内部仕様:

- 動的上限計算では、`ver` は単純なID比較ではなく、各バージョンの `released_at` に対応する期間条件へ変換されます。
- 複数 `ver` 指定時は、それぞれの期間を `OR` で連結して対象曲を絞ります。

### 8.5 `const`

譜面定数範囲です。

入力形状:

```json
{
  "const": {
    "min": 14.0,
    "max": 15.9
  }
}
```

条件:

- `min` と `max` はそれぞれ任意
- 未指定時はデフォルト値に補完されます
- 小数第1位までのみ許可
- 範囲は `1.0 <= min <= max <= 15.9`

補完仕様:

- `min` 省略時は `1.0`
- `max` 省略時は `15.9`

例:

入力:

```json
{
  "const": {
    "max": 15.9
  }
}
```

保存・返却:

```json
{
  "const": {
    "min": 1.0,
    "max": 15.9
  }
}
```

## 9. 正規化ルール

このAPIは入力JSONをそのまま保存しません。
バリデーション後の正規化済みJSONを再エンコードして保存します。

そのため、レスポンスJSONは入力と完全一致しないことがあります。

主な正規化:

- `title` は前後空白を trim
- `attributes.diff` / `genre` / `ver` はソート・重複除去
- 単要素配列はスカラーへ変換
- `attributes.const.min` / `max` は省略時に既定値補完
- `attributes` 未指定時は `{}` として保存

## 10. 動的上限バリデーション

`achievement_params` の一部は、固定値だけではなく「対象譜面数」に応じた動的上限で検証されます。
対象譜面は `charts` と `songs` を参照して算出され、`songs.is_deleted = 0` の譜面だけが対象です。

対象条件:

- `diff`
- `genre`
- `ver`
- `const`

で絞り込まれます。

### 10.1 `count` 系

対象:

- `rank_count`
- `score_count`
- `hardlamp_count`
- `combolamp_count`

条件:

- `count <= 対象譜面数`

### 10.2 `total_score`

条件:

- `total <= 対象譜面数 * 1010000`

### 10.3 `overpower_value`

条件:

- `total <= ((対象譜面定数合計 + 対象譜面数 * 2.0) * 5.0) + 対象譜面数 * 5.0`

### 10.4 `overpower_percent`

- 固定範囲 `0..100` のみで判定され、動的上限はありません。

## 11. 作成・更新・削除の挙動

### 11.1 作成

作成処理はトランザクション内で行われます。

流れ:

1. 入力検証
2. `users` の対象行を `FOR UPDATE` でロック
3. そのユーザーの goal 件数を数える
4. 100件未満なら INSERT
5. 直後に `FindByIDAndUserID` で再読込
6. レスポンスへ変換

### 11.2 更新

流れ:

1. 入力検証
2. `id + user_id` で既存 goal を取得
3. 値を上書き
4. UPDATE 実行
5. メモリ上の goal をレスポンス化

補足:

- 更新処理は `created_at` を変更しません。
- 更新後の再取得はしていません。

### 11.3 削除

`DELETE FROM goals WHERE id = ? AND user_id = ?` を実行します。
対象0件なら `goal_not_found` です。

## 12. エラー仕様

### 12.1 goals API 固有エラー

| エラーコード | HTTPステータス | 条件 |
| --- | --- | --- |
| `goal_not_found` | 404 | 指定IDの goal が存在しない、または他ユーザー所有 |
| `goal_limit_exceeded` | 400 | 100件上限超過 |
| `goal_invalid_title` | 400 | `title` が trim 後空、31文字以上、または制御文字含む |
| `goal_invalid_achievement_type` | 400 | `achievement_type` がマスタに存在しない |
| `goal_invalid_achievement_params` | 400 | `achievement_params` の形状不正、範囲不正、動的上限超過 |
| `goal_invalid_attributes` | 400 | `attributes` の形状不正、未知キー、ID不正、範囲不正 |
| `invalid_goal_input` | 400 | ユースケース入力全般不正 |

### 12.2 共通エラー

| エラーコード | HTTPステータス | 条件 |
| --- | --- | --- |
| `unauthorized` | 401 | 認証情報がない、またはコンテキストにユーザーがいない |
| `bad_request` | 400 | JSON不正、`Content-Type` 不正、未知トップレベルキー、`id` が数値でない |
| `validation_failed` | 422 | DTOレベル必須チェック失敗 |
| `internal_error` | 500 | マスタ不整合、DB異常など |

注意点:

- `title` / `achievement_type` の未指定または空文字、`achievement_params` の未指定は、ハンドラ層ではまず `validation_failed` になる可能性があります。
- JSONの構造自体が壊れている場合や strict decode に反する場合は `bad_request` です。

## 13. JSON形状の注意点

フロントエンド実装で注意すべき点は以下です。

- `attributes` は常に object で返ります。未指定入力でもレスポンスでは `{}` です。
- `diff` / `genre` / `ver` は、入力時に配列でも返却時はスカラーになる場合があります。
- `created_at` は常に文字列で返り、UNIX時刻ではありません。
- `achievement_params` は型安全DTOではなく object 扱いなので、`achievement_type` を見て解釈を切り替える必要があります。
- `invert` はサーバー側で評価条件に使われていません。保存・返却される表示用フラグです。

## 14. 実装から見える補足事項

- 一覧APIは評価結果を返しません。返すのは目標定義のみです。
- 絞り込み対象は `charts` ベースで、`songs.is_deleted = 0` のみを対象にします。
- `WORLD'S END` 用の `worldsend_charts` は goals の対象計算に含まれていません。
- `achievement_type_id` から `achievement_type` 文字列への逆引きに失敗すると、一覧・作成・更新レスポンスはいずれも `internal_error` になります。

## 15. フロントエンド向け TypeScript 表現例

```ts
type GoalAchievementType =
  | 'rank_count'
  | 'score_count'
  | 'avg_score'
  | 'hardlamp_count'
  | 'combolamp_count'
  | 'total_score'
  | 'overpower_value'
  | 'overpower_percent';

type GoalAttributes = {
  diff?: number | number[];
  const?: {
    min: number;
    max: number;
  };
  genre?: number | number[];
  ver?: number | number[];
};

type GoalRequest = {
  title: string;
  achievement_type: GoalAchievementType;
  achievement_params: Record<string, unknown>;
  attributes?: GoalAttributes;
  invert?: boolean;
};

type GoalResponse = {
  id: number;
  title: string;
  achievement_type: GoalAchievementType;
  achievement_params: Record<string, unknown>;
  attributes: GoalAttributes;
  invert: boolean;
  created_at: string;
};

type GoalsResponse = {
  goals: GoalResponse[];
};
```

実際には返却時の `diff` / `genre` / `ver` は、正規化の結果として `number` になることがあります。
そのため受信側は `number | number[]` として扱うのが安全です。
