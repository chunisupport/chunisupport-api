# 目標機能 仕様書兼実装指示書（暫定）

本ドキュメントは、目標機能（ゴール機能）の仕様と実装方針をまとめた暫定設計書である。正式ドキュメントではなく、実装作業の指示と合意内容の記録を目的とする。

## 背景・目的

- CHUNITHMプレイヤーの目標（例: 15全AJ、14全理論値など）を定義し、達成状況を可視化する。
- 目標は保存するが、達成状況は保存せず、フロント側でスコアデータと突き合わせて都度判定する。
- サーバーは単一VPS・単一プロセスを前提とする。

## 方針まとめ

- 目標データは SQLite に保存する。
- 目標数はユーザーあたり最大100件。
- 目標データ（goalbody）は API 側で可能な限り厳密に検証し、不正データは拒否する。
- 目標の達成判定は API で行わず、フロントで計算する。
- グループ名は goalbody から分離し、テーブルカラムとして保持する。
- invert は goalbody 内に残す（カラム化しない）。
- id指定を基本とし、diff/genre/ver/lamp 系はマスタの ID で指定する。
- schemaVersion は 1 固定とする。
- count は必須。値がない場合は null を許容し、対象譜面数を目標値として扱う。
- skill は今回実装しない（将来対応）。

## データ保存（SQLite）

### テーブル: goals

| カラム | 型 | 必須 | 説明 |
| --- | --- | --- | --- |
| id | TEXT | ✓ | UUID |
| user_id | INTEGER | ✓ | ユーザーID |
| group_name | TEXT | | グループ名（NULL可） |
| goalbody | TEXT | ✓ | goalbody JSON |
| created_at | DATETIME | ✓ | 作成日時 |
| updated_at | DATETIME | ✓ | 更新日時 |

- group_name は UI の分類用途。goalbody に含めない。
- group リネームはフロントで全目標を更新する。

## API設計（/internal/me/goals）

- GET /internal/me/goals
  - 目標一覧を返す（goalbody + group_name）。
  - 達成状況は返さない。
- POST /internal/me/goals
  - 目標新規作成。
  - 100件超過はエラー。
- PUT /internal/me/goals/:id
  - 目標更新。
- DELETE /internal/me/goals/:id
  - 目標削除。

## goalbody 仕様

### 基本構造

```json
{
  "schemaVersion": 1,
  "name": "15.5全SSS",
  "attributes": [
    {
      "type": "const",
      "lower": 15.5,
      "upper": 15.5
    }
  ],
  "achievement": {
    "type": "score",
    "value": 1007500,
    "count": 11
  },
  "invert": false
}
```

### 共通ルール

- `schemaVersion` は必須で `1` 固定。
- `name` は必須。最大30 rune。禁止語チェックを実施。
- `attributes` は AND 条件で評価。
- 同一 type の attributes は禁止（type重複不可）。
- `invert` は表示の反転のみ（内部計算値は変更しない）。
- 未知の type は API で拒否。

### attributes

| type | 形式 | 説明 |
| --- | --- | --- |
| const | number lower/upper | 譜面定数（小数1桁の数値） |
| diff | int lower/upper | 難易度マスタ ID の範囲 |
| genre | int lower/upper | ジャンルマスタ ID の範囲 |
| ver | int lower/upper | バージョンマスタ ID の範囲 |

- const は小数1桁の数値のみ許容する。
- API 側で丸めはしない。
- lower > upper はエラー。
- WE は対象外（diff に含めない）。

### achievement

| type | 形式 | 説明 |
| --- | --- | --- |
| score | value + count | スコア下限 + 達成譜面数 |
| score_sum | value + count | 合計スコア下限 + 対象譜面数（countはnull許可） |
| lamp_combo | value + count | コンボランプ ID + 達成譜面数 |
| lamp_chain | value + count | フルチェインランプ ID + 達成譜面数 |

- skill は今回未実装（将来追加予定）。
- count は必須キー。`null` の場合は対象譜面数を目標値として扱う。
- score_sum の invert 表示は「総失点」表示とし、理論値は 1,010,000 固定。

## バリデーション方針

API 側で可能な限り厳密に検証し、不正は拒否する。

- JSONとしてパースできるか
- 必須キーの存在
- typeごとの構造・型チェック
- const のフォーマットチェック
- マスタ ID の存在確認（diff/genre/ver/lamp）
- name/group の文字数・禁止語
- attributes の type 重複禁止

## フロント側判定

- 目標達成判定はフロント側で実施。
- 進捗表示は `現在値 / 目標値` で統一。
- count が null の場合は対象譜面数を目標値として扱う。
- invert は表示を「未達成数」「失点数」に反転する。

## 禁止語リスト

- 静的ファイルで保持する。
- 起動時に読み込む想定。
- 文字列バリデーション用の共通関数を用意する（目標機能専用ではない）。

## 将来拡張予定

- skill の追加（順序や比較ルールが固まった段階で実装）。
- API での目標判定ロジックの追加（現時点では不要）。
- group の一括リネームAPI（必要になった場合のみ）。
