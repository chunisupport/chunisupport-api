# スコアデータ登録差分API改善 設計書

## 1. 目的

スコアデータ登録時の差分レスポンスを、更新差分画像や更新結果画面で扱いやすい形に改善する。

現状の登録レスポンスは、実際に更新されたレコード差分を返せる一方で、次の情報を1レスポンスで扱いにくい。

- 登録前後のプロフィール数値
- 登録前後の難易度別・ランク別・ランプ別集計
- 登録後のトータルハイスコアや平均スコア

本設計では、内部IDや不要なメタ情報を返さず、`before` / `after` / `changes` の3要素で差分表示に必要な情報を提供する。

## 2. 前提と現状整理

### 2.1 現状の差分レスポンス

現状のスコアデータ登録レスポンスは、更新されたレコードを `changes` として返す。

`changes` の各要素は、通常譜面・WORLD'S ENDの種別、変更種別、公式インデックス、難易度、更新前後のスコア・ランプ状態を持つ。

```json
{
  "record_type": "full",
  "change_type": "updated",
  "idx": "2849",
  "diff": "MASTER",
  "before": {
    "score": 990000,
    "clear_lamp": "CLEAR",
    "combo_lamp": null,
    "full_chain": null
  },
  "after": {
    "score": 1002345,
    "clear_lamp": "BRAVE",
    "combo_lamp": "FULL COMBO",
    "full_chain": null
  }
}
```

この構造は「実際に更新されたレコード一覧」には適しているが、トータルハイスコア、平均スコア、ランク別件数などの集計値は含まない。

### 2.2 レーティングの扱い

現在のDBには、公式レーティングと計算レーティングの両方が保存されている。

- `official_player_rating`: 入力payloadの `rating` 由来
- `calculated_player_rating`: 保存済みスコアから再計算したプレイヤーレーティング
- `best_average_rating`: ベスト枠平均
- `new_average_rating`: 新曲枠平均

差分画像や更新結果表示では、入力payload由来の公式値ではなく、保存済みスコアから算出した計算値を利用する。

### 2.3 OVER POWERの扱い

入力payloadには `overpower.value` / `overpower.percentage` が存在するが、これらは互換入力用であり保存値には使用しない。

DBに保存する `overpower_value` は、保存済み通常譜面レコードから再集計した値とする。`overpower_percentage` はDBに保存せず、登録レスポンスまたはプロフィールレスポンスの返却時点で計算する。

### 2.4 ランクキーの表現

既存の譜面統計APIでは、ランクキーは表示文字列ではなく小文字キーで返している。

| 表示 | APIキー |
| --- | --- |
| 既存統計のAAAL | `aaal` |
| S | `s` |
| S+ | `sp` |
| SS | `ss` |
| SS+ | `ssp` |
| SSS | `sss` |
| SSS+ | `sssp` |
| MAX | `max` |

今回追加する集計でも、既存統計APIに合わせて `s`, `sp`, `ss`, `ssp`, `sss`, `sssp`, `max` などのキーを使用する。

## 3. 改善方針

### 3.1 レスポンスは3要素に整理する

レスポンスの基本構造は次の3要素とする。

```json
{
  "before": {},
  "after": {},
  "changes": []
}
```

- `before`: 登録前のプロフィール・集計スナップショット。初回登録時は `null`
- `after`: 登録後のプロフィール・集計スナップショット
- `changes`: 実際に新規追加または更新されたレコード一覧

### 3.2 トップレベルに内部IDやメタ情報を含めない

以下のようなトップレベル項目は返さない。

- `player_id`
- `app_ver`
- `imported_at`
- `source_updated_at`

理由は次の通り。

- `player_id` は内部IDであり、漏洩リスクを下げるためレスポンスに含めない。
- 画像生成・差分表示に不要な情報はレスポンスから除外する。
- 必要な表示情報は `before.profile` / `after.profile` に集約する。

### 3.3 `delta` は返さない

差分値はクライアント側で `after - before` により計算する。

APIは前後の値のみを返し、差分表示の丸め・符号・単位表記は表示側で制御する。

## 4. レスポンス仕様案

### 4.1 全体構造

```json
{
  "before": null,
  "after": {
    "profile": {},
    "records": {}
  },
  "changes": []
}
```

`before` は既存プレイヤーの更新時のみオブジェクトになり、初回登録時は必ず `null` とする。初回登録時に `before.profile` や `before.records` を `null` 埋めしたオブジェクトとして返してはいけない。

#### 4.1.1 null契約

レスポンス内で欠落または未計算の値は、原則としてフィールドを省略せず `null` で返す。

例外は次の通り。

- 件数として確定できる値は `0` を返す。
- `changes` は更新レコードが0件の場合も空配列 `[]` を返す。
- 初回登録時の登録前スナップショットは、フィールドごとの `null` 埋めではなくトップレベルの `before: null` で表現する。

クライアントは次の契約だけを見ればよい。

- `before === null`: 初回登録または登録前状態なし
- `before !== null`: `before.profile` と `before.records` を参照可能
- `after`: 登録成功時は常に非null
- `changes`: 常に配列

#### 4.1.2 JSON Schema概略

```json
{
  "type": "object",
  "required": ["before", "after", "changes"],
  "additionalProperties": false,
  "properties": {
    "before": {
      "anyOf": [
        { "$ref": "#/definitions/snapshot" },
        { "type": "null" }
      ]
    },
    "after": { "$ref": "#/definitions/snapshot" },
    "changes": {
      "type": "array",
      "items": { "$ref": "#/definitions/change" }
    }
  },
  "definitions": {
    "snapshot": {
      "type": "object",
      "required": ["profile", "records"],
      "additionalProperties": false,
      "properties": {
        "profile": { "$ref": "#/definitions/profile" },
        "records": { "$ref": "#/definitions/records" }
      }
    },
    "profile": {
      "type": "object",
      "required": [
        "name",
        "level",
        "calculated_rating",
        "best_average_rating",
        "new_average_rating",
        "overpower_value",
        "overpower_percent",
        "last_played_at"
      ],
      "properties": {
        "name": { "type": ["string", "null"] },
        "level": { "type": ["number", "null"] },
        "calculated_rating": { "type": ["number", "null"] },
        "best_average_rating": { "type": ["number", "null"] },
        "new_average_rating": { "type": ["number", "null"] },
        "overpower_value": { "type": ["number", "null"] },
        "overpower_percent": { "type": ["number", "null"] },
        "last_played_at": { "type": ["string", "null"] }
      }
    },
    "records": {
      "type": "object",
      "required": ["by_difficulty"],
      "properties": {
        "by_difficulty": {
          "type": "object",
          "additionalProperties": {
            "anyOf": [
              { "$ref": "#/definitions/difficulty_summary" },
              { "type": "null" }
            ]
          }
        }
      }
    }
  }
}
```

`definitions.change` と `definitions.difficulty_summary` の詳細は後続節のフィールド定義に従う。

### 4.2 `profile`

`profile` は、差分画像や更新結果表示で使うプロフィール数値を保持する。

既存プレイヤーであっても、何らかの理由で個別値を算出できない場合は、フィールドを省略せず `null` を返す。

```json
{
  "name": "プレイヤー名",
  "level": 217,
  "calculated_rating": 17.27,
  "best_average_rating": 17.13,
  "new_average_rating": 17.49,
  "overpower_value": 123469.12,
  "overpower_percent": 98.5608,
  "last_played_at": "2026-06-06T08:11:57+09:00"
}
```

#### フィールド

| フィールド | 型 | 説明 |
| --- | --- | --- |
| `name` | string | プレイヤー名 |
| `level` | number | プレイヤーレベル |
| `calculated_rating` | number \| null | 保存済みスコアから計算したプレイヤーレーティング |
| `best_average_rating` | number \| null | ベスト枠平均レーティング |
| `new_average_rating` | number \| null | 新曲枠平均レーティング |
| `overpower_value` | number \| null | 保存済み通常譜面レコードから再集計したOVER POWER値 |
| `overpower_percent` | number \| null | 登録処理時点の分母で計算したOVER POWER割合 |
| `last_played_at` | string \| null | 最終プレイ日時 |

`official_rating` は原則として含めない。公式値との差分検証が必要な場合は、別用途の管理・診断レスポンスで扱う。

### 4.3 `records`

`records` は、通常譜面の難易度別集計を保持する。

難易度別集計が算出できない場合は、対象難易度の値を `null` にする。集計自体は算出でき、件数が0件である場合は、件数フィールドには `0` を返す。

```json
{
  "by_difficulty": {
    "MASTER": {
      "total_score": 1636397944,
      "average_score": 1009499.0401,
      "played_count": 1621,
      "rank": {
        "aaal": 1621,
        "s": 1621,
        "sp": 1621,
        "ss": 1621,
        "ssp": 1599,
        "sss": 1548,
        "sssp": 1350,
        "max": 89
      },
      "combo": {
        "none": 254,
        "fc": 1367,
        "aj": 1235
      },
      "clear": {
        "failed": 0,
        "clear": 120,
        "hard": 1000,
        "brave": 400,
        "absolute": 90,
        "catastrophy": 11
      },
      "full_chain": {
        "none": 1488,
        "gold": 100,
        "platinum": 33
      }
    }
  }
}
```

#### `records.by_difficulty.*`

| フィールド | 型 | 説明 |
| --- | --- | --- |
| `total_score` | number | 対象難易度の合計スコア |
| `average_score` | number \| null | 対象難易度の平均スコア。対象レコードが0件の場合は `null` |
| `played_count` | number | プレイ済み譜面数 |
| `rank` | object | ランク閾値以上の件数 |
| `combo` | object | コンボランプ別件数 |
| `clear` | object | クリアランプ別件数 |
| `full_chain` | object | フルチェイン別件数 |

#### 集計対象難易度

通常譜面の難易度キーは大文字名称を使用する。

- `BASIC`
- `ADVANCED`
- `EXPERT`
- `MASTER`
- `ULTIMA`

画像生成では `MASTER` / `ULTIMA` のみを利用する場合でも、APIは全難易度を返せる設計にする。

#### ランク集計

`rank` は既存統計APIのキーに合わせる。

```json
{
  "aaal": 0,
  "s": 0,
  "sp": 0,
  "ss": 0,
  "ssp": 0,
  "sss": 0,
  "sssp": 0,
  "max": 0
}
```

各値は「そのランク以上を達成している件数」とする。たとえば `sssp` はSSS+以上、`sss` はSSS以上、`max` は理論値達成数を表す。

#### コンボ集計

`combo` は既存統計APIに合わせる。

```json
{
  "none": 0,
  "fc": 0,
  "aj": 0
}
```

#### クリア集計

`clear` は既存統計APIに合わせる。

```json
{
  "failed": 0,
  "clear": 0,
  "hard": 0,
  "brave": 0,
  "absolute": 0,
  "catastrophy": 0
}
```

#### フルチェイン集計

`full_chain` は差分画像用途で追加する。

```json
{
  "none": 0,
  "gold": 0,
  "platinum": 0
}
```

入力互換上、`fch_lv` は `1=NONE`, `2=PLATINUM`, `3=GOLD` として解釈しているため、集計キーはマスタ名を小文字正規化した値に合わせる。

### 4.4 `changes`

`changes` は実際に新規追加または更新されたレコードのみを返す。

```json
[
  {
    "record_type": "full",
    "change_type": "updated",
    "idx": "2849",
    "diff": "MASTER",
    "before": {
      "score": 1007285,
      "clear_lamp": "HARD",
      "combo_lamp": null,
      "full_chain": null
    },
    "after": {
      "score": 1007800,
      "clear_lamp": "HARD",
      "combo_lamp": null,
      "full_chain": null
    }
  }
]
```

現状の `changes` と同等の情報を維持する。

#### フィールド

| フィールド | 型 | 説明 |
| --- | --- | --- |
| `record_type` | string | `full` または `worldsend` |
| `change_type` | string | `new` または `updated` |
| `idx` | string | 楽曲の公式インデックス |
| `diff` | string | 通常譜面は大文字難易度名、WORLD'S ENDは `WE` |
| `before` | object \| null | 更新前状態。新規追加時は `null` |
| `after` | object | 更新後状態 |

楽曲名、ジャケット、表記レベルは別APIで解決する。表記レベルは譜面定数からクライアント側で算出できるため、このレスポンスには含めない。

## 5. サンプルレスポンス

```json
{
  "before": {
    "profile": {
      "name": "プレイヤー名",
      "level": 217,
      "calculated_rating": 17.26,
      "best_average_rating": 17.12,
      "new_average_rating": 17.47,
      "overpower_value": 123456.78,
      "overpower_percent": 98.5503,
      "last_played_at": "2026-06-06T08:01:57+09:00"
    },
    "records": {
      "by_difficulty": {
        "MASTER": {
          "total_score": 1636387364,
          "average_score": 1009492.5133,
          "played_count": 1621,
          "rank": {
            "aaal": 1621,
            "s": 1621,
            "sp": 1621,
            "ss": 1621,
            "ssp": 1599,
            "sss": 1546,
            "sssp": 1347,
            "max": 89
          },
          "combo": {
            "none": 255,
            "fc": 1366,
            "aj": 1234
          },
          "clear": {
            "failed": 0,
            "clear": 121,
            "hard": 999,
            "brave": 400,
            "absolute": 90,
            "catastrophy": 11
          },
          "full_chain": {
            "none": 1488,
            "gold": 100,
            "platinum": 33
          }
        }
      }
    }
  },
  "after": {
    "profile": {
      "name": "プレイヤー名",
      "level": 217,
      "calculated_rating": 17.27,
      "best_average_rating": 17.13,
      "new_average_rating": 17.49,
      "overpower_value": 123469.12,
      "overpower_percent": 98.5608,
      "last_played_at": "2026-06-06T08:11:57+09:00"
    },
    "records": {
      "by_difficulty": {
        "MASTER": {
          "total_score": 1636397944,
          "average_score": 1009499.0401,
          "played_count": 1621,
          "rank": {
            "aaal": 1621,
            "s": 1621,
            "sp": 1621,
            "ss": 1621,
            "ssp": 1599,
            "sss": 1548,
            "sssp": 1350,
            "max": 89
          },
          "combo": {
            "none": 254,
            "fc": 1367,
            "aj": 1235
          },
          "clear": {
            "failed": 0,
            "clear": 120,
            "hard": 1000,
            "brave": 400,
            "absolute": 90,
            "catastrophy": 11
          },
          "full_chain": {
            "none": 1488,
            "gold": 100,
            "platinum": 33
          }
        }
      }
    }
  },
  "changes": [
    {
      "record_type": "full",
      "change_type": "updated",
      "idx": "2849",
      "diff": "MASTER",
      "before": {
        "score": 1007285,
        "clear_lamp": "HARD",
        "combo_lamp": null,
        "full_chain": null
      },
      "after": {
        "score": 1007800,
        "clear_lamp": "HARD",
        "combo_lamp": null,
        "full_chain": null
      }
    }
  ]
}
```

## 6. 集計仕様

### 6.1 スコア集計

- `total_score`: 対象難易度のプレイ済み通常譜面スコア合計
- `average_score`: `total_score / played_count`
- `played_count`: プレイ済み通常譜面数

未プレイ補完データは登録処理の保存対象ではないため、集計対象に含めない。

### 6.2 ランク集計

各ランクは「以上」の件数として集計する。

| キー | 条件 |
| --- | --- |
| `aaal` | 既存譜面統計APIの `aaal` と同じ条件 |
| `s` | 975000以上 |
| `sp` | 990000以上 |
| `ss` | 1000000以上 |
| `ssp` | 1005000以上 |
| `sss` | 1007500以上 |
| `sssp` | 1009000以上 |
| `max` | 1010000 |

`aaal` は既存統計APIとの整合を優先する。実装時は既存の統計生成処理の閾値を確認し、同じ条件を共通関数化して利用する。

### 6.3 ランプ集計

ランプ集計はマスタIDではなく、レスポンス用の固定キーに正規化する。

- コンボ: `none`, `fc`, `aj`
- クリア: `failed`, `clear`, `hard`, `brave`, `absolute`, `catastrophy`
- フルチェイン: `none`, `gold`, `platinum`

## 7. 実装方針

### 7.1 ユースケース層に集計ロジックを置く

ビジネスロジック・集計ロジックはハンドラ層に置かず、ユースケース層またはドメインサービスに配置する。

ハンドラはリクエスト形式の判定、DTOバインド、レスポンス返却に限定する。

### 7.2 保存後レコード取得の再利用

スコア登録処理では、OVER POWER再計算のために保存後の通常譜面レコードを取得している。この取得済みデータを、次の計算にも利用する。

- `after.profile.calculated_rating`
- `after.profile.best_average_rating`
- `after.profile.new_average_rating`
- `after.profile.overpower_value`
- `after.profile.overpower_percent`
- `after.records.by_difficulty`

これにより、追加のDBアクセスを増やしすぎずに集計値を作る。

### 7.3 `before` の作り方

`before` は初回登録時と既存プレイヤー更新時で契約を分ける。

- 初回登録時: `before` は必ず `null` とする。
- 既存プレイヤー更新時: `before` は `profile` と `records` を持つスナップショットオブジェクトとする。

初回登録時に `before.profile` や `before.records` を `null` 埋めしたオブジェクトとして返すと、クライアント側が「更新前状態は存在するが各値が欠落している」と誤解するため禁止する。

既存プレイヤー更新時の `before` は次のどちらかで構築する。

#### 方針A: 保存前の全レコードを取得する

保存前に通常譜面レコードを全件取得し、`before.records` と `before.profile` を素直に集計する。

メリット:

- 実装が単純
- 集計ロジックが `before` / `after` で共通化しやすい

デメリット:

- 保存前全件取得が増える

#### 方針B: 保存後集計と `changes` から復元する

`after.records` を保存後全件から作り、`before.records` は `changes` を逆適用して作る。

メリット:

- 保存前全件取得を避けられる

デメリット:

- ランク・ランプ・平均スコアの逆算ロジックが複雑
- 実装ミスが起きやすい
- 将来の集計項目追加に弱い

#### 推奨

初期実装では方針Aを推奨する。

理由は、レスポンス設計の主目的が正確で扱いやすい差分表示であり、逆算ロジックで複雑化するよりも、保存前後のスナップショットを同じ集計器に通すほうが保守しやすいため。

負荷が問題になった場合は、方針Bまたは集計クエリ最適化を検討する。

### 7.4 レーティング計算の共通化

現在のレーティング再計算処理はDB更新のみを行う。差分レスポンスで計算値を返すため、計算処理は次のように分離する。

- レコード一覧から `RatingStats` を計算する関数
- `RatingStats` をDBへ保存する処理
- `RatingStats` をレスポンススナップショットへ変換する処理

これにより、同じ計算結果をDB保存とレスポンス生成で共有できる。

### 7.5 OVER POWER計算の共通化

OVER POWERも同様に、通常譜面レコード一覧から計算した結果をDB保存とレスポンス生成で共有する。

`overpower_percent` は保存しないが、登録処理時点の分母で算出した値を `after.profile.overpower_percent` として返す。

既存プレイヤー更新時の `before.profile.overpower_percent` は、保存前レコードと同じ分母条件で計算する。これにより、`before` / `after` の差分は今回登録による変化として扱いやすくなる。初回登録時は `before` 自体が `null` のため、`before.profile.overpower_percent` は存在しない。

## 8. DB・マイグレーション方針

本改善だけでは新規テーブル追加は不要。

既存の `official_player_rating` については、差分レスポンスでは使用しない。

将来的に公式値を不要と判断する場合は、別タスクとして次を検討する。

- 入力payloadの `rating` を受け取るだけにするか、任意項目化する
- `official_player_rating` の保存停止
- 公開DTOの `rating` を計算値へ統一
- 管理画面や互換APIで公式値を使っている箇所の整理
- `official_player_rating` カラム削除マイグレーション

この設計書の範囲では、公式値カラムの削除は行わない。

## 9. API公開方針

### 9.1 既存登録APIのレスポンスを差し替える案

後方互換性を考慮しない場合、既存の `PlayerDataResult` を本設計のレスポンスへ置き換える。

メリット:

- API利用側が単純
- 登録後すぐ画像生成できる

デメリット:

- 登録処理のレスポンス生成コストが常に増える

### 9.2 クエリパラメータで切り替える案

通常登録では軽量レスポンスを返し、差分画像用途では明示的に詳細レスポンスを返す。

例:

```http
POST /internal/me/register-data?format=json&response=diff
```

メリット:

- 通常登録の負荷を抑えられる
- 画像生成用途だけ詳細集計を返せる

デメリット:

- レスポンス型が複数になり、実装とドキュメントが複雑になる

### 9.3 推奨

画像生成・更新結果表示を主用途とするなら、既存登録APIのレスポンスを本設計へ寄せる案を推奨する。

ただし、負荷を厳密に分離したい場合は `response=diff` のような明示指定を採用する。

## 10. テスト方針

### 10.1 集計器の単体テスト

通常譜面レコード一覧から、難易度別集計が正しく作られることをテーブルテストで確認する。

確認項目:

- `total_score`
- `average_score`
- `played_count`
- `rank.sssp` などのランク件数
- `combo.fc` / `combo.aj`
- `clear.hard` などのクリア件数
- `full_chain.gold` / `full_chain.platinum`

### 10.2 before / after レスポンス生成テスト

保存前後のレコードを入力し、`before` / `after` のプロフィール・集計値が正しく生成されることを確認する。初回登録時は `before` が `null` で返ることも確認する。

### 10.3 レーティング計算値のテスト

レスポンスの `calculated_rating` が入力payloadの公式値ではなく、保存済みレコードから算出した値であることを確認する。

### 10.4 OVER POWER計算値のテスト

レスポンスの `overpower_value` が入力payloadの公式値ではなく、保存済み通常譜面レコードから再集計された値であることを確認する。

### 10.5 ID非露出テスト

レスポンスJSONに `player_id` などの内部IDが含まれないことを確認する。

### 10.6 null契約テスト

欠落または未計算の値がフィールド省略ではなく `null` で返ることを確認する。件数として確定できる値は `0`、更新レコードがない場合の `changes` は `[]` で返ることも確認する。

## 11. 懸念点と対応

### 11.1 保存前全件取得による負荷

保存前後のスナップショットを正確に作るには、保存前全件取得が最も単純である。

負荷が問題になる場合は、次を検討する。

- 差分レスポンスを明示指定時のみ生成する
- SQL集計で一部集計値だけ取得する
- `changes` から `before` 集計を逆算する

### 11.2 公式値との混同

`rating` という曖昧な名前は避け、レスポンスでは `calculated_rating` を使用する。

公式値を返す必要が出た場合のみ `official_rating` として明示的に追加する。

### 11.3 WORLD'S ENDの扱い

本設計の `records.by_difficulty` は通常譜面集計を主対象とする。

WORLD'S ENDは `changes` には含めるが、トータルハイスコアやレーティング・OVER POWER集計とは別扱いにする。

将来的にWORLD'S END集計が必要な場合は、`records.worldsend` を追加する。

## 12. 実装タスク案

1. 差分レスポンス用DTOを定義する。
2. 通常譜面レコード集計用のドメインサービスまたはユースケース内ヘルパーを追加する。
3. レーティング計算処理を、計算結果の取得とDB保存に分離する。
4. OVER POWER計算結果をレスポンス生成でも使えるように整理する。
5. 登録処理で保存前スナップショットを取得する。
6. 登録処理で保存後スナップショットを生成する。
7. レスポンスから内部ID・不要メタ情報を除外する。
8. APIドキュメントを更新する。
9. 単体テスト・ユースケーステストを追加する。
10. `go test ./...` と `gofmt -s -w .` を実行する。

## 13. 最終レスポンス案

### 13.1 初回登録時

初回登録時は登録前状態が存在しないため、`before` は必ず `null` とする。

```json
{
  "before": null,
  "after": {
    "profile": {
      "name": "プレイヤー名",
      "level": 217,
      "calculated_rating": 17.27,
      "best_average_rating": 17.13,
      "new_average_rating": 17.49,
      "overpower_value": 123469.12,
      "overpower_percent": 98.5608,
      "last_played_at": "2026-06-06T08:11:57+09:00"
    },
    "records": {
      "by_difficulty": {}
    }
  },
  "changes": []
}
```

### 13.2 既存プレイヤー更新時

既存プレイヤー更新時は `before` と `after` の両方にスナップショットを返す。

```json
{
  "before": {
    "profile": {
      "name": "プレイヤー名",
      "level": 217,
      "calculated_rating": 17.26,
      "best_average_rating": 17.12,
      "new_average_rating": 17.47,
      "overpower_value": 123456.78,
      "overpower_percent": 98.5503,
      "last_played_at": "2026-06-06T08:01:57+09:00"
    },
    "records": {
      "by_difficulty": {}
    }
  },
  "after": {
    "profile": {
      "name": "プレイヤー名",
      "level": 217,
      "calculated_rating": 17.27,
      "best_average_rating": 17.13,
      "new_average_rating": 17.49,
      "overpower_value": 123469.12,
      "overpower_percent": 98.5608,
      "last_played_at": "2026-06-06T08:11:57+09:00"
    },
    "records": {
      "by_difficulty": {}
    }
  },
  "changes": []
}
```
