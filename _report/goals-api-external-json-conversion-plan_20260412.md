# 目標API 外向きJSON変換計画書

## 0. 現行状況（2026-05-18時点）

本計画はまだ実装されていません。現行の `/internal/me/goals` 系APIは、以下の通り内部IDを含むJSON契約のままです。

- `achievement_type`: コード文字列
- `attributes.diff`: difficulty ID（1〜5）
- `attributes.genre`: genre ID
- `attributes.ver`: version ID

一方で、`achievement_params` は当初調査時より許容形が広がっており、以下は省略または `null` を受け付けます。

- `rank_count.count`
- `score_count.count`
- `hardlamp_count.count`
- `combolamp_count.count`
- `total_score.total`
- `overpower_value.total`

上記は「対象譜面数」や「対象譜面の理論値合計」などの動的上限値を目標値として扱うための入力です。外向きJSON変換を実装する場合も、この省略/null許容仕様は維持する必要があります。

## 1. 目的

本計画書は、`/internal/me/goals` 系APIが現在フロントエンドに露出している
マスタデータ内部値依存を解消し、**DB保存形式を変更せずに、APIの入出力JSONだけを外向き表現へ変換する**ための計画を定義する。

対象:

- `GET /internal/me/goals`
- `POST /internal/me/goals`
- `PUT /internal/me/goals/:id`

非対象:

- `DELETE /internal/me/goals/:id`
- goalsテーブルのカラム構成変更
- 既存の動的上限計算ロジックそのものの変更

---

## 2. 背景

現状の goal API は、同じレスポンス内で以下が混在している。

- `achievement_type`: コード文字列で返している
- `attributes.diff` / `attributes.genre` / `attributes.ver`: マスタIDのまま返している

このため、フロントエンドは以下の問題を抱える。

1. APIレスポンスだけでは表示に必要な情報が完結しない
2. `/internal/master` の辞書を前提にしないと意味解釈できない
3. API契約が「表示用の外向き仕様」ではなく「内部正規化JSON」に引きずられている
4. 将来マスタIDや保存形式を見直しにくい

調査結果としても、楽曲・レコード系APIは名称展開寄りである一方、
goal API だけが内部値露出の色が強いことが確認されている。

---

## 3. 現状整理

## 3.1 現状の責務配置

- Handler:
  - `GoalRequest` の `map[string]any` をそのまま `json.Marshal` して Usecase に渡す
- Usecase:
  - `achievement_type` をコードから `achievement_type_id` に変換する
  - `attributes` はJSONを正規化するが、`diff` / `genre` / `ver` は内部IDのまま保存する
  - 出力時は保存済みJSONを `map[string]any` に戻してそのまま返す

結果として、**永続化表現とAPI表現が分離されていない**状態になっている。

## 3.2 問題の本質

問題は「DBにIDを保存していること」ではない。
問題は「APIが内部保存表現を外部契約としてそのまま公開していること」である。

したがって、解決方針はDB変更ではなく、
**API境界で外向き表現と内部表現を明確に変換すること**になる。

---

## 4. 方針

本対応では以下の3層を明確に分離する。

1. 外向きAPI表現
2. 内部正規化表現
3. DB保存表現

### 4.1 外向きAPI表現

フロントエンドが直接利用するためのJSON契約。
マスタIDではなく、意味の分かる安定した値を使う。

### 4.2 内部正規化表現

現行Usecaseがバリデーション・動的上限計算・永続化に利用するJSON。
`diff` / `genre` / `ver` は内部IDで扱ってよい。

### 4.3 DB保存表現

goalsテーブルの `achievement_params` / `attributes` に保存するJSON。
今回は変更しない。

---

## 5. 採用案

## 5.1 基本戦略

**DB保存形式は維持し、API入出力の境界で相互変換する。**

具体的には:

- リクエスト受信時:
  - 外向きJSONを受ける
  - API用Mapperで内部正規化JSONへ変換する
  - その後、既存Usecaseのバリデーション・永続化へ流す

- レスポンス返却時:
  - Usecaseが返した内部値ベースの出力を受ける
  - API用Mapperで外向きJSONへ変換して返す

これにより、UsecaseとRepositoryの責務を大きく壊さずに改善できる。

## 5.2 採用理由

1. goalsテーブルや既存データの移行が不要
2. 動的上限計算や既存バリデーションの再利用が容易
3. 後方互換を段階的に制御しやすい
4. 将来、goalの型安全DTOへ進める際の土台になる

---

## 6. 外向きJSON設計方針

## 6.1 原則

- マスタIDをそのまま外へ出さない
- 表示文言そのものではなく、可能なら安定したコード値を使う
- 難易度文字列は既存方針どおり大文字で扱う
- API契約はフロントが `/internal/master` を参照しなくても基本的な表示・編集が成立する形を目指す

## 6.2 `achievement_type`

現状どおりコード文字列を維持する。

例:

- `rank_count`
- `score_count`
- `avg_score`

## 6.3 `attributes.diff`

内部IDではなく難易度コードを使う。

例:

```json
{
  "diff": "MASTER"
}
```

または

```json
{
  "diff": ["EXPERT", "MASTER"]
}
```

補足:

- 難易度コードは `BASIC` / `ADVANCED` / `EXPERT` / `MASTER` / `ULTIMA`
- 正規化ルールは現状どおり、重複除去・表示順ソートを行う
- 単要素はスカラーに正規化してよい

## 6.4 `attributes.genre`

理想は表示名ではなく公開用コードだが、現時点で専用コードがなければ段階的に決める必要がある。

候補:

1. `genre` に公開用コードを新設する
2. 当面は `genre_name` 相当の値をAPI契約として使う

本計画では、**表示名そのものをAPI契約に固定するのは弱いため、公開用コードの導入を推奨**する。

ただし短期対応としては、既存マスタに公開用コードが存在しない場合のみ、
一時的に名称ベースの変換を許容する。

## 6.5 `attributes.ver`

`genre` と同様に、内部IDではなく公開用コードまたは安定した識別子を使う。

候補:

1. 公開用コードを設ける
2. 既存 `versions.name` を使う

バージョン名称も将来変更余地があるため、
長期的には表示名ではなく公開用識別子を持つ形が望ましい。

## 6.6 `achievement_params`

現時点では大きな変更は行わず、既存構造を維持する。

理由:

- 現在の問題の中心は `attributes` 側の内部値露出である
- `achievement_type` ごとの構造変更まで同時に行うと影響範囲が広がりすぎる

ただし、現行仕様では `count` 系の `count`、`total_score.total`、`overpower_value.total` が省略または `null` を許容する。この互換性は外向きJSON化後も維持する。

## 6.7 旧形式と新形式のイメージ

旧形式:

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
      "max": 16.0
    },
    "genre": [1, 2],
    "ver": 20
  },
  "invert": false
}
```

新形式の目標イメージ:

```json
{
  "title": "MASTER 14.0以上を1譜面AJ",
  "achievement_type": "combolamp_count",
  "achievement_params": {
    "lamp": "AJ",
    "count": 1
  },
  "attributes": {
    "diff": "MASTER",
    "const": {
      "min": 14.0,
      "max": 16.0
    },
    "genre": ["POPS_ANIME", "NICONICO"],
    "ver": "LUMINOUS"
  },
  "invert": false
}
```

上記の `genre` / `ver` の値はあくまで公開用コードを採用した場合のイメージであり、
実際の識別子は別途確定する。

---

## 7. 具体的な変換責務

## 7.1 入力変換

新しい API input mapper が以下を担当する。

- `diff`: 難易度コード -> difficulty ID
- `genre`: 公開用値 -> genre ID
- `ver`: 公開用値 -> version ID
- 重複排除
- 並び順の正規化
- 空配列や未知値の検証

変換後は現行Usecaseに渡す内部正規化JSONを生成する。

## 7.2 出力変換

新しい API output mapper が以下を担当する。

- `diff`: difficulty ID -> 難易度コード
- `genre`: genre ID -> 公開用値
- `ver`: version ID -> 公開用値

変換失敗時は、マスタ不整合として `internal_error` 扱いにする。

---

## 8. 責務分離の設計

## 8.1 推奨構成

- `internal/dto/api_internal/goal_dto.go`
  - goal APIの外向きDTOを定義
- `internal/app/handler/api_internal/goal_handler.go`
  - HTTP入出力とバリデーション開始点のみ担当
- 新規 mapper
  - API DTO <-> Usecase入力/出力の変換を担当
- `internal/usecase/goal_usecase_impl.go`
  - 内部正規化表現に対する検証・保存・取得を担当

## 8.2 避けたい構成

- Handler内にマスタ参照ロジックを直接大量に書く
- Usecaseが「HTTP API専用の表示都合」を抱え込む
- RepositoryでAPI用変換を行う

API表現の変換は、Clean Architecture上は外側の責務として扱うのが自然である。

---

## 9. 移行方式

## 9.1 推奨方式

段階移行を採用する。

### Step 1

計画書とAPI仕様案を確定する。

### Step 2

サーバーを新旧入力両対応にする。

- 旧: IDベース入力
- 新: 外向き値ベース入力

ただしレスポンスはまだ旧形式のままでもよい。

### Step 3

レスポンスを新形式へ切り替える。

フロントが新形式を前提に動作する状態へ移行する。

### Step 4

旧入力の受け付けを廃止する。

## 9.2 理由

いきなり完全切替すると、既存フロントと既存保存データの両方を同時に気にする必要があり、切り戻しも難しくなる。
新旧入力両対応を短期間だけ挟む方が安全である。

---

## 10. 互換性ポリシー

## 10.1 破壊的変更の扱い

レスポンスの `attributes.diff` / `genre` / `ver` をIDから外向き値へ変更するのは破壊的変更である。

そのため、以下のいずれかを選ぶ必要がある。

1. 同一エンドポイントで段階移行し、フロントを先に追従させる
2. 新しいレスポンスフィールドを追加して移行期間を設ける
3. `/internal/me/goals` をv2相当に分ける

小規模プロジェクトであることを考えると、
**短期間の新旧入力両対応 + レスポンス切替** が最も現実的である。

## 10.2 文書更新

本対応時には、少なくとも以下を更新する。

- `docs/API.md`
- `_report/goals-api-current-specification.md` または後継仕様書
- 必要に応じて `docs/domain_model_specification.md`

---

## 11. テスト方針

## 11.1 追加すべき観点

- 新形式入力を内部IDへ正しく変換できる
- 旧形式入力も移行期間中は受け付ける
- レスポンスで内部IDが露出しない
- `diff` の単値/配列/重複/順序が正しく正規化される
- `genre` / `ver` の未知値で適切にバリデーションエラーになる
- 既存の動的上限計算が壊れていない

## 11.2 特に重要な回帰観点

- `attributes` 未指定時に `{}` が返る
- `achievement_type` ごとの `achievement_params` バリデーションは変わらない
- `invert` の保存・返却は変わらない
- 一覧順は `created_at ASC, id ASC` のまま

---

## 12. 実装タスク分解

1. 外向きgoal JSON仕様を確定する
2. `genre` / `ver` の公開用識別子ポリシーを決める
3. goal API用 mapper を追加する
4. request DTO を外向き表現へ合わせる
5. response DTO を外向き表現へ合わせる
6. handler から mapper を呼ぶように変更する
7. 旧形式との互換対応を入れる
8. handler / usecase テストを追加・更新する
9. `docs/API.md` を更新する
10. 旧形式廃止時に互換コードを削除する

---

## 13. 未決事項

## 13.1 `genre` / `ver` の公開値

最重要の未決事項。

以下をどれにするか決める必要がある。

- 表示名をそのまま使う
- 新しい公開用コードを定義する
- `/internal/master` のレスポンス構造も合わせて見直す

本計画としては、**表示名固定より公開用コード導入を推奨**する。

## 13.2 旧形式入力の許容期間

- フロントの改修順序
- リリース頻度
- 切り戻し手順

を見て決める必要がある。

---

## 14. 結論

goal API の問題は、DB内部値を保存していることではなく、
**内部保存表現がそのままAPI契約になっていること**にある。

したがって、最も筋の良い対応は以下である。

- DBは変えない
- 内部正規化JSONも原則維持する
- API境界に変換レイヤーを設ける
- 外向きJSONはマスタIDではなく意味の分かる安定値で表現する

この方針であれば、現在の実装資産を活かしつつ、
フロントエンド依存の改善と将来の保守性向上を両立できる。
