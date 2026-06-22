# プレイヤーデータ登録時の差分仕様

## 1. 対象

本書は、プレイヤーデータ登録時に返すスコア差分と通常譜面の集計差分について、現行コードの挙動を定義する。

対象エンドポイントは次の2つである。

- `POST /internal/me/register-data`
- `POST /internal/player-data/commit`

`/internal/player-data/commit` は一時保存した本文を復元した後、`/internal/me/register-data` と同じ `PlayerDataUsecase.Register` を呼ぶ。そのため、両エンドポイントの差分仕様は同一である。登録処理はプレイヤー情報、称号、通常譜面、WORLD'S END、レーティング再計算を含む1トランザクションで実行され、失敗時は差分を含めて登録全体が確定しない。

リクエスト形式、認証、バリデーション、エラー仕様を含むAPI全体の仕様は [API.md](./API.md) を参照する。

## 2. レスポンス全体

登録成功時は `PlayerDataResult` を返す。差分に関係するフィールドは `statistics`、`counts`、`changes`、`skipped_records` である。

```json
{
  "player_id": 42,
  "app_ver": "0.1.0",
  "imported_at": "2026-06-21T12:00:00Z",
  "profile": {},
  "summary": {},
  "statistics": {},
  "counts": {},
  "changes": [],
  "skipped_records": []
}
```

`changes` と `skipped_records` は該当データがない場合も省略せず、空配列 `[]` を返す。

## 3. レコード単位の差分 (`changes`)

### 3.1 差分の判定対象

入力から解決でき、保存対象になった通常譜面とWORLD'S ENDについて、保存直前のDB状態とupsert予定値を譜面ID単位で比較する。

次の4項目のいずれかが異なる場合だけ、実際に変化したレコードとして扱う。

- スコア
- クリアランプID
- コンボランプID
- フルチェインID

通常譜面の `slot`、`order` と、通常譜面・WORLD'S ENDの `updated_at` は保存されるが、差分判定には含めない。この条件はDBのupsert時に `updated_at` を更新する条件と同一である。

保存前に対象譜面が存在しない場合は、4項目の値にかかわらず `new` とする。保存前に存在し、4項目のいずれかが異なる場合は `updated` とする。全項目が同じ場合は `changes` に含めない。

### 3.2 要素スキーマ

```json
{
  "record_type": "standard",
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

| フィールド | 型 | 仕様 |
| --- | --- | --- |
| `record_type` | string | 通常譜面は `standard`、WORLD'S ENDは `worldsend` |
| `change_type` | string | 新規レコードは `new`、既存レコードの更新は `updated` |
| `idx` | string | 楽曲の公式インデックス。マスタから解決できない内部不整合時は楽曲IDまたは譜面IDの文字列 |
| `diff` | string | 通常譜面はマスタ上の大文字難易度名、WORLD'S ENDは常に `WE` |
| `before` | object \| null | 更新前状態。`new` の場合は `null` |
| `after` | object | upsert予定の登録後状態 |

`before` と `after` は `score`、`clear_lamp`、`combo_lamp`、`full_chain` を常に含む。ランプはマスタの `Name` をそのまま返す。マスタ名が `none`（大文字・小文字を区別しない）、未設定、またはマスタから解決できない場合は `null` を返す。

### 3.3 並び順と件数上限

`changes` は次の優先順で昇順に並べる。

1. `idx` の数値
2. `idx` の文字列
3. `record_type`
4. `diff`
5. `change_type`

数値として解釈できる `idx` は、解釈できない `idx` より前に並ぶ。レスポンスに含める詳細は先頭100件までである。実際の変更総数は後述の `counts.*_actually_changed` で確認する。

### 3.4 同一payload内の重複

同一の譜面IDがpayload内に複数回現れた場合は、最後の有効な1件を保存値および差分表示に使用する。正規化は通常譜面とWORLD'S ENDで個別に行う。

## 4. 集計差分 (`statistics`)

`statistics` は保存済みの通常譜面全件を登録前後に集計した結果である。WORLD'S ENDは含めない。`overall` は全難易度の合計、`by_difficulty` は難易度別の集計を返す。

```json
{
  "overall": {
    "total_high_score": { "before": 2000000, "after": 2010000, "delta": 10000 },
    "record_statistics": {
      "aj": { "before": 1, "after": 1, "delta": 0 },
      "fc": { "before": 2, "after": 3, "delta": 1 },
      "clr": { "before": 2, "after": 3, "delta": 1 },
      "fch": { "before": 0, "after": 0, "delta": 0 },
      "max": { "before": 0, "after": 0, "delta": 0 },
      "sss_plus": { "before": 0, "after": 1, "delta": 1 },
      "sss": { "before": 1, "after": 2, "delta": 1 },
      "ss_plus": { "before": 1, "after": 2, "delta": 1 },
      "ss": { "before": 2, "after": 3, "delta": 1 },
      "s_plus": { "before": 2, "after": 3, "delta": 1 },
      "s": { "before": 2, "after": 3, "delta": 1 }
    }
  },
  "by_difficulty": {
    "BASIC": {},
    "ADVANCED": {},
    "EXPERT": {},
    "MASTER": {},
    "ULTIMA": {}
  }
}
```

`by_difficulty` はレコードの有無にかかわらず、`BASIC`、`ADVANCED`、`EXPERT`、`MASTER`、`ULTIMA` の5キーを必ず返す。各集計値は `before`、`after`、`delta` を持ち、`delta = after - before` で計算する。登録によって達成状態が下がった場合、`delta` は負数になる。

### 4.1 集計項目

| キー | 条件 |
| --- | --- |
| `total_high_score` | 通常譜面スコアの合計 |
| `aj` | コンボランプ名が `ALL JUSTICE` |
| `fc` | コンボランプ名が `FULL COMBO` または `ALL JUSTICE` |
| `clr` | クリアランプ名が `FAILED` 以外 |
| `fch` | フルチェイン名が `FULL CHAIN GOLD` または `FULL CHAIN PLATINUM` |
| `max` | スコアが1,010,000と等しい |
| `sss_plus` | スコアが1,009,000以上 |
| `sss` | スコアが1,007,500以上 |
| `ss_plus` | スコアが1,005,000以上 |
| `ss` | スコアが1,000,000以上 |
| `s_plus` | スコアが990,000以上 |
| `s` | スコアが975,000以上 |

ランク件数はそれぞれのボーダー以上を数える累積値である。たとえば1,009,000点の譜面は `sss_plus`、`sss`、`ss_plus`、`ss`、`s_plus`、`s` のすべてに含まれる。

## 5. 件数 (`counts`)

```json
{
  "standard_records_upserted": 1185,
  "worldsend_records_upserted": 120,
  "standard_records_skipped": 2,
  "worldsend_records_skipped": 1,
  "honors_skipped": 0,
  "standard_records_actually_changed": 12,
  "worldsend_records_actually_changed": 3
}
```

| フィールド | 仕様 |
| --- | --- |
| `standard_records_upserted` | `scores.standard` で処理を開始した要素数。名称にかかわらず、後続処理でスキップされた要素も含む |
| `worldsend_records_upserted` | `scores.worldsend` で処理を開始した要素数。名称にかかわらず、後続処理でスキップされた要素も含む |
| `standard_records_skipped` | 通常譜面でマスタ解決、スコア範囲、ランプ解決、スロット解決のいずれかに失敗して保存対象外になった件数 |
| `worldsend_records_skipped` | WORLD'S ENDでマスタ解決、スコア範囲、ランプ解決のいずれかに失敗して保存対象外になった件数 |
| `honors_skipped` | 保存対象外になった称号の件数 |
| `standard_records_actually_changed` | 重複正規化後の通常譜面で `new` または `updated` と判定された全件数 |
| `worldsend_records_actually_changed` | 重複正規化後のWORLD'S ENDで `new` または `updated` と判定された全件数 |

`*_actually_changed` は100件制限前の件数であるため、`changes` の要素数と一致しない場合がある。また、`*_upserted` は入力要素ごとに加算してから検証するため、スキップや同一譜面の重複がある場合は、実際にDBへ渡すupsert行数より大きくなる。

## 6. スキップ情報 (`skipped_records`)

保存対象にできなかった通常譜面、WORLD'S END、称号は次の形式で返す。

```json
{
  "record_type": "standard",
  "reason": "failed to resolve chart",
  "details": "idx=9999, diff=MAS, error=..."
}
```

`record_type` は `standard`、`worldsend`、または `honor` である。`reason` と `details` は失敗箇所が生成する診断用文字列であり、固定のエラーコードではない。スキップされた要素は `changes` と `statistics.after` に含まれない。

## 7. 計算と保存の順序

差分関連処理はトランザクション内で次の順序で行う。

1. プレイヤーを新規作成または更新する。
2. 保存済み通常譜面全件を取得し、`statistics.before` を計算する。
3. payloadのスコアをマスタIDへ変換し、保存できない要素をスキップする。
4. 同一譜面の重複を最後の1件へ正規化する。
5. 保存対象譜面だけの保存前状態を通常譜面・WORLD'S ENDごとに一括取得する。
6. 保存前状態とupsert予定値を比較し、`changes` と `*_actually_changed` を作る。
7. 通常譜面とWORLD'S ENDをbulk upsertする。
8. 保存済み通常譜面全件を再取得し、`statistics.after` とOVER POWERを計算する。
9. 保存済み全スコアからレーティングを再計算し、レスポンスを組み立てる。

保存前状態の取得は譜面ごとの個別問い合わせではなく一括取得する。集計も登録前後に通常譜面全件を1回ずつ取得して行うため、譜面数に比例したN+1問い合わせは発生しない。

## 8. 初回登録と同時登録

初回登録では保存済み譜面がないため、`statistics.before` の全項目は0となり、保存対象の各譜面は `new` になる。

差分は「保存前状態の取得」と「upsert予定値」の比較で作る。同一プレイヤーに対する複数の登録リクエストを同時実行した場合、別リクエストが取得後に状態を変更すると、返却した差分が最終的なDB更新内容と一致しない可能性がある。現行実装は同一プレイヤーの登録が同時実行されない通常利用を前提とし、排他制御による差分の直列化は行わない。

## 9. 実装上の参照先

- レスポンスDTO: `internal/dto/api_internal/player_data_dto.go`
- 登録・差分計算: `internal/usecase/player_data_usecase_impl.go`
- 集計条件: `internal/domain/service/player_record_statistics.go`
- DB upsert条件: `internal/infra/repository/player_data_repository_impl.go`
- HTTPエンドポイント: `internal/app/handler/api_internal/me_handler.go`、`internal/app/handler/api_internal/temporary_player_data_handler.go`
