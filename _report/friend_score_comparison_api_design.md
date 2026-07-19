# VSフレンド スコア比較API 設計書

## 0. 文書の位置付け

この文書は、フロントエンドに「VSフレンド」画面を追加するための **未実装API設計** です。

現行APIには以下が存在します。

- `GET /internal/users/:username/record`: 指定ユーザーの全レコード取得
- `GET /internal/friend-rankings/songs/:displayid/charts/:difficulty`: 通常譜面1件のフレンドランキング取得
- `GET /internal/friend-rankings/worldsend-songs/:displayid`: WORLD'S END譜面1件のフレンドランキング取得

既存の全レコードAPIを自分とフレンドの2人分取得してクライアント側で比較することは可能ですが、比較に不要な楽曲情報・レーティング・OVER POWERなども重複して転送・展開します。また、譜面単位ランキングAPIを全譜面分呼び出す方式はNリクエストになるため採用しません。

本設計では、1人の承認済みフレンドと、指定した1難易度の全通常譜面を比較する専用APIを新設します。

---

## 1. 目的

- 自分と選択したフレンドのスコアを、通常譜面単位で比較できるようにする
- 未プレイ譜面を含む勝敗集計をAPI側で一貫して行う
- 1回のリクエストでは1難易度だけを対象とし、レスポンス件数とメモリ使用量を制限する
- 比較に必要なデータだけを返し、既存の全レコードDTOを2人分取得する方式より通信量を削減する
- フレンド関係の認可と比較仕様をバックエンドに集約する

---

## 2. 対象範囲

### 2.1 対象

- Firebase認証済みユーザー本人と、指定した承認済みフレンドの比較
- 通常譜面
- 難易度 `BASIC` / `ADVANCED` / `EXPERT` / `MASTER` / `ULTIMA`
- 選択された難易度に存在する全有効譜面
- プレイ済み、片方のみ未プレイ、両者未プレイのすべて
- 現在のベストスコアと現在のランプ

### 2.2 対象外

- WORLD'S END
- コースモード
- スコア履歴同士の比較
- 3人以上の同時比較
- 全難易度の一括取得
- 難易度以外のサーバー側絞り込み
- レーティング、OVER POWER、順位の比較

---

## 3. 比較ルール

## 3.1 未プレイの扱い

未プレイも比較対象および集計対象に含めます。

API内部では未プレイを次の値へ正規化します。

| フィールド | 未プレイ時の値 |
| --- | --- |
| `is_played` | `false` |
| `score` | `0` |
| `clear_lamp` | `null` |
| `combo_lamp` | `null` |
| `full_chain` | `null` |
| `updated_at` | `null` |

未プレイのためだけに疑似レコードをDBへ保存してはいけません。LEFT JOIN結果が存在しない場合にUsecaseまたはDTO変換で上記の値へ正規化します。

## 3.2 勝敗判定

`score_difference` は常に次の式で算出します。

```text
score_difference = self.score - friend.score
```

| 条件 | `result` | 集計先 |
| --- | --- | --- |
| `self.score > friend.score` | `SELF_WIN` | `summary.self_wins` |
| `self.score = friend.score` | `DRAW` | `summary.draws` |
| `self.score < friend.score` | `FRIEND_WIN` | `summary.friend_wins` |

したがって、未プレイを含む判定例は次の通りです。

| 自分 | フレンド | 結果 |
| --- | --- | --- |
| プレイ済み | 未プレイ | `SELF_WIN` |
| 未プレイ | プレイ済み | `FRIEND_WIN` |
| 未プレイ | 未プレイ | `DRAW` |

スコアが同じ場合は、ランプや更新日時をタイブレークに使用せず、必ず引き分けとします。

## 3.3 対象譜面

対象は次の条件をすべて満たす譜面です。

- `songs.is_deleted = false`
- `songs.is_worldsend = false`
- `difficulties.name = パスで指定された難易度`
- 対応する `charts` が存在する

両者が未プレイでも対象譜面から除外しません。譜面マスタを起点に両者のレコードをLEFT JOINすることで、選択難易度の全譜面を取得します。

---

## 4. エンドポイント

### GET `/internal/friend-comparisons/:friend_user_id/charts/:difficulty`

- **認証**: Firebase Bearer必須
- **概要**: 自分と指定した承認済みフレンドについて、指定難易度の全通常譜面のスコア比較と集計を返す
- **ページング**: なし

### 4.1 パスパラメータ

| パラメータ | 型 | 必須 | 説明 |
| --- | --- | --- | --- |
| `friend_user_id` | positive integer | 必須 | フレンド一覧APIで返される内部ユーザーID |
| `difficulty` | string | 必須 | `BASIC` / `ADVANCED` / `EXPERT` / `MASTER` / `ULTIMA` |

難易度は既存ドメインルールに従って大文字へ正規化してから検証します。レスポンスでは必ず大文字を返します。

### 4.2 ページングを設けない理由

1回の対象を1難易度に限定するため、返却件数の上限はその難易度の有効譜面マスタ数になります。レスポンスも比較専用のコンパクトDTOとし、既存 `PlayerRecordDTO` にあるレーティング、OVER POWER、スロット等は返しません。

全件を返すことで、フロントエンドは追加リクエストなしで集計値と一覧を一致させられます。将来、1難易度の件数または実測レスポンスサイズが許容値を超えた場合にのみ、カーソルページングとサーバー側集計の分離を再検討します。

---

## 5. レスポンス設計

## 5.1 レスポンス例

```json
{
  "difficulty": "MASTER",
  "self": {
    "user_id": 1,
    "username": "myuser",
    "player_name": "MY PLAYER"
  },
  "friend": {
    "user_id": 2,
    "username": "frienduser",
    "player_name": "FRIEND"
  },
  "summary": {
    "total_charts": 3,
    "self_wins": 1,
    "draws": 1,
    "friend_wins": 1,
    "self_played": 2,
    "friend_played": 1,
    "both_played": 1,
    "self_only_played": 1,
    "friend_only_played": 0,
    "both_unplayed": 1
  },
  "items": [
    {
      "song": {
        "id": "0000000000000001",
        "title": "楽曲名",
        "artist": "アーティスト名"
      },
      "chart": {
        "const": 14.5,
        "is_const_unknown": false
      },
      "self": {
        "is_played": true,
        "score": 1009000,
        "clear_lamp": "CLEAR",
        "combo_lamp": "FULL COMBO",
        "full_chain": null,
        "updated_at": "2026-07-20T10:00:00Z"
      },
      "friend": {
        "is_played": true,
        "score": 1007500,
        "clear_lamp": "CLEAR",
        "combo_lamp": null,
        "full_chain": null,
        "updated_at": "2026-07-19T10:00:00Z"
      },
      "score_difference": 1500,
      "result": "SELF_WIN"
    },
    {
      "song": {
        "id": "0000000000000002",
        "title": "未プレイ楽曲",
        "artist": "アーティスト名"
      },
      "chart": {
        "const": 13.0,
        "is_const_unknown": false
      },
      "self": {
        "is_played": false,
        "score": 0,
        "clear_lamp": null,
        "combo_lamp": null,
        "full_chain": null,
        "updated_at": null
      },
      "friend": {
        "is_played": false,
        "score": 0,
        "clear_lamp": null,
        "combo_lamp": null,
        "full_chain": null,
        "updated_at": null
      },
      "score_difference": 0,
      "result": "DRAW"
    }
  ]
}
```

## 5.2 DTO定義

### `FriendScoreComparisonUserDTO`

| フィールド | 型 | 説明 |
| --- | --- | --- |
| `user_id` | integer | 内部ユーザーID |
| `username` | string | ユーザー名 |
| `player_name` | string | プレイヤー名 |

### `FriendScoreComparisonSummaryDTO`

| フィールド | 型 | 説明 |
| --- | --- | --- |
| `total_charts` | integer | 対象難易度の全有効譜面数 |
| `self_wins` | integer | 自分の勝利数 |
| `draws` | integer | 引き分け数。両者未プレイを含む |
| `friend_wins` | integer | フレンドの勝利数 |
| `self_played` | integer | 自分のプレイ済み数 |
| `friend_played` | integer | フレンドのプレイ済み数 |
| `both_played` | integer | 両者プレイ済み数 |
| `self_only_played` | integer | 自分だけプレイ済みの数 |
| `friend_only_played` | integer | フレンドだけプレイ済みの数 |
| `both_unplayed` | integer | 両者未プレイ数 |

集計値は次の不変条件を満たす必要があります。

```text
total_charts = self_wins + draws + friend_wins
total_charts = both_played + self_only_played + friend_only_played + both_unplayed
self_played = both_played + self_only_played
friend_played = both_played + friend_only_played
```

### `FriendScoreComparisonRecordDTO`

| フィールド | 型 | 説明 |
| --- | --- | --- |
| `is_played` | boolean | レコードが存在するか |
| `score` | integer | 現在のベストスコア。未プレイは0 |
| `clear_lamp` | string \| null | クリアランプ。未プレイまたはマスタ値 `NONE` はnull |
| `combo_lamp` | string \| null | コンボランプ。未プレイまたはマスタ値 `NONE` はnull |
| `full_chain` | string \| null | フルチェイン。未プレイまたはマスタ値 `NONE` はnull |
| `updated_at` | string \| null | レコード更新日時。未プレイはnull |

### `FriendScoreComparisonItemDTO`

| フィールド | 型 | 説明 |
| --- | --- | --- |
| `song` | object | `id`、`title`、`artist` を持つ楽曲概要 |
| `chart` | object | `const`、`is_const_unknown` を持つ譜面概要 |
| `self` | `FriendScoreComparisonRecordDTO` | 自分のレコード |
| `friend` | `FriendScoreComparisonRecordDTO` | フレンドのレコード |
| `score_difference` | integer | 自分のスコアからフレンドのスコアを引いた値 |
| `result` | string | `SELF_WIN` / `DRAW` / `FRIEND_WIN` |

`difficulty` はレスポンス全体で共通のため、各 `item.chart` には重複して持たせません。

## 5.3 並び順

`items` は `songs.id ASC` のマスタ順で返します。同じリクエスト条件に対して常に決定的な順序になることを保証します。

勝敗順やスコア差順は本APIでは提供しません。必要になった場合は、取得済みの1難易度分をフロントエンドで並べ替えます。

---

## 6. 認証・認可

## 6.1 必須条件

次の条件をすべて満たす場合だけ比較結果を返します。

1. リクエスト元ユーザーがFirebase認証済み
2. `friend_user_id` が自分自身ではない
3. 自分から相手への `accepted` が存在する
4. 相手から自分への `accepted` が存在する
5. 自分と相手の両方がプレイヤーデータ連携済み

公開アカウントであっても、承認済みフレンドでなければ本APIでは比較できません。「VSフレンド」という機能境界を維持し、任意ユーザー比較APIとして利用されることを防ぎます。

## 6.2 情報秘匿

ユーザー不存在、片方向だけの不整合、申請中、解除済みはクライアントから区別できないよう、すべて同じ `friend_not_found` として扱います。

プレイヤーデータ未連携は、承認済みフレンド関係を確認した後に `friend_score_comparison_unavailable` を返します。これにより、フレンドではないユーザーの連携状態を推測できないようにします。

---

## 7. エラー仕様

| HTTP | エラーコード | 条件 |
| --- | --- | --- |
| 400 | `validation_failed` | `friend_user_id` が正の整数でない |
| 400 | `invalid_difficulty` | 難易度が未指定または許可値でない |
| 401 | `missing_token` / `invalid_token` | 認証情報がない、または不正 |
| 404 | `friend_not_found` | 対象が承認済み双方向フレンドではない、自分自身を指定した、または対象ユーザーが存在しない |
| 409 | `friend_score_comparison_unavailable` | 自分または承認済みフレンドがプレイヤーデータ未連携 |
| 500 | `internal_error` | DBアクセスなどの予期しない内部エラー |

`friend_not_found` と `friend_score_comparison_unavailable` は新規エラーコードとして `internal/app/apierror`、フロントエンドの `ErrorCode`、`docs/API.md` に追加します。

対象難易度に有効譜面が0件の場合はエラーにせず、すべての集計値が0で `items: []` の正常レスポンスを返します。

---

## 8. アーキテクチャ設計

既存のClean Architectureの依存方向を維持します。

## 8.1 Domain / Repository interface

新規ファイル案:

- `internal/domain/repository/friend_score_comparison_query_service.go`

読み取り専用モデル:

- `FriendScoreComparisonUser`
- `FriendScoreComparisonChartRecord`

インターフェース案:

```go
type FriendScoreComparisonQueryService interface {
    FindAcceptedFriendPair(
        ctx context.Context,
        exec Executor,
        selfUserID int,
        friendUserID int,
    ) (*FriendScoreComparisonUsers, error)

    ListChartRecords(
        ctx context.Context,
        exec Executor,
        selfPlayerID int,
        friendPlayerID int,
        difficulty string,
    ) ([]*FriendScoreComparisonChartRecord, error)
}
```

`FindAcceptedFriendPair` は双方向 `accepted` と両ユーザー情報を1回で検証・取得します。プレイヤーIDは未連携を表現できるようポインタで保持します。

## 8.2 Usecase

新規ファイル案:

- `internal/usecase/friend_score_comparison_usecase.go`

インターフェース案:

```go
type FriendScoreComparisonUsecase interface {
    Get(
        ctx context.Context,
        selfUserID int,
        friendUserID int,
        difficulty string,
    ) (*FriendScoreComparisonResult, error)
}
```

責務:

1. 難易度を大文字へ正規化して検証する
2. 双方向の承認済みフレンド関係を検証する
3. 両ユーザーのプレイヤーデータ連携を検証する
4. 指定難易度の全譜面比較行を一括取得する
5. DBレコードがない側を未プレイ値へ正規化する
6. `score_difference` と `result` を算出する
7. `summary` を1回のループで集計する

勝敗判定と集計はUsecaseで行い、Handlerへ流出させません。

## 8.3 Infrastructure

新規ファイル案:

- `internal/infra/repository/friend_score_comparison_query_service_impl.go`

比較行取得は譜面マスタを起点にし、両者の `player_records` をそれぞれLEFT JOINします。

SQL概略:

```sql
SELECT
    s.id AS song_sort_id,
    s.display_id AS song_display_id,
    s.title AS song_title,
    s.artist AS song_artist,
    c.const AS chart_const,
    c.is_const_unknown,
    self_pr.score AS self_score,
    self_cl.name AS self_clear_lamp,
    self_co.name AS self_combo_lamp,
    self_fc.name AS self_full_chain,
    self_pr.updated_at AS self_updated_at,
    friend_pr.score AS friend_score,
    friend_cl.name AS friend_clear_lamp,
    friend_co.name AS friend_combo_lamp,
    friend_fc.name AS friend_full_chain,
    friend_pr.updated_at AS friend_updated_at
FROM charts c
INNER JOIN songs s
    ON s.id = c.song_id
INNER JOIN difficulties d
    ON d.id = c.difficulty_id
LEFT JOIN player_records self_pr
    ON self_pr.chart_id = c.id
   AND self_pr.player_id = ?
LEFT JOIN player_records friend_pr
    ON friend_pr.chart_id = c.id
   AND friend_pr.player_id = ?
LEFT JOIN clear_lamp_types self_cl
    ON self_cl.id = self_pr.clear_lamp_id
LEFT JOIN combo_lamp_types self_co
    ON self_co.id = self_pr.combo_lamp_id
LEFT JOIN full_chain_types self_fc
    ON self_fc.id = self_pr.full_chain_id
LEFT JOIN clear_lamp_types friend_cl
    ON friend_cl.id = friend_pr.clear_lamp_id
LEFT JOIN combo_lamp_types friend_co
    ON friend_co.id = friend_pr.combo_lamp_id
LEFT JOIN full_chain_types friend_fc
    ON friend_fc.id = friend_pr.full_chain_id
WHERE s.is_deleted = FALSE
  AND s.is_worldsend = FALSE
  AND d.name = ?
ORDER BY s.id ASC;
```

実装時は `SELECT *` を使用せず、実際に返却する列だけを明示します。

## 8.4 Presentation

新規ファイル案:

- `internal/app/handler/api_internal/friend_score_comparison_handler.go`
- `internal/dto/api_internal/friend_score_comparison_dto.go`

RouterではFirebase認証必須グループへ次を追加します。

```text
GET /internal/friend-comparisons/:friend_user_id/charts/:difficulty
```

Handlerの責務はパスパラメータ検証、Usecase呼び出し、DTO変換に限定します。

---

## 9. DB・パフォーマンス設計

## 9.1 クエリ回数

1リクエストあたり原則2クエリとします。

1. 双方向フレンド関係と両ユーザー・プレイヤーの取得
2. 指定難易度の全譜面と両者レコードの一括取得

譜面ごとのレコード取得やランプ取得は行わず、N+1を禁止します。

## 9.2 既存インデックス

`player_records` の主キーは `(player_id, chart_id)` です。両プレイヤーのLEFT JOINはこの主キーを使用できます。

既存の主な関連インデックス:

- `player_records PRIMARY KEY (player_id, chart_id)`
- `charts` の主キー
- `songs PRIMARY KEY (id)`
- `songs INDEX (is_worldsend, is_deleted)`
- `friendships PRIMARY KEY (user_id, friend_user_id)`

初期実装では新規インデックスを追加しません。実データに対する `EXPLAIN ANALYZE` でフルスキャンや一時テーブルが問題になる場合のみ、`charts(difficulty_id, song_id)` などの追加を検討します。

## 9.3 メモリとレスポンスサイズ

- 全難易度や2人分の全 `PlayerRecordDTO` は保持しない
- 1難易度分の比較行だけをDB・Usecase・JSONレスポンスで保持する
- 各行にレーティング、OVER POWER、画像URL、スロットを含めない
- `difficulty` はトップレベルに1回だけ返す
- 集計用の別配列を作らず、比較行の生成と同じループで集計する

実装後、実データに近い最大件数で以下を計測します。

- SQL実行時間
- JSON生成前後のレスポンスサイズ
- HTTP圧縮後の転送サイズ
- APIプロセスの1リクエストあたり追加メモリ
- フロントエンドでのJSONパース時間

## 9.4 性能目標

具体的な数値は実環境計測後に確定しますが、受け入れ基準として少なくとも次を満たすものとします。

- DBクエリ回数が譜面数に依存せず2回である
- 返却件数が指定難易度の有効譜面数と一致する
- レスポンスに他難易度の行が混入しない
- 既存の全レコードAPIを2人分取得するより非圧縮レスポンスが小さい

---

## 10. キャッシュ方針

初期実装ではAPIサーバー内キャッシュを追加しません。

理由:

- 比較結果は2ユーザーと難易度の組み合わせでキャッシュキー数が増える
- どちらかのスコア更新時に無効化が必要になる
- 1難易度限定かつ2クエリで取得でき、まず実測すべきである

フロントエンドでは画面表示中の取得結果だけを保持し、別フレンドまたは別難易度を選択した際に不要な比較配列を解放できる構造とします。永続キャッシュは初期実装の対象外です。

将来キャッシュを追加する場合は、少なくとも次をキーまたは検証値に含めます。

- 自分のユーザーID
- フレンドのユーザーID
- 難易度
- 自分のレコード最終更新日時
- フレンドのレコード最終更新日時
- 楽曲・譜面マスタ最終更新日時

---

## 11. テスト設計

## 11.1 Usecaseテスト

Given-When-Then形式で次を確認します。

- 自分のスコアが高い場合は `SELF_WIN`
- フレンドのスコアが高い場合は `FRIEND_WIN`
- 同点は `DRAW`
- 自分だけプレイ済みの場合は `SELF_WIN`
- フレンドだけプレイ済みの場合は `FRIEND_WIN`
- 両者未プレイの場合は `DRAW`
- ランプ差があっても同スコアなら `DRAW`
- `score_difference` が自分基準の符号になる
- 全summary項目と不変条件が一致する
- 難易度が大文字へ正規化される
- 不正難易度をRepositoryへ渡さない
- フレンドでない場合は比較行を取得しない
- 自分またはフレンドがプレイヤーデータ未連携の場合は比較行を取得しない
- 比較対象譜面0件を正常な空レスポンスとして返す

## 11.2 Repositoryテスト

SQLiteテストまたは既存のRepositoryテスト方式に合わせて次を確認します。

- 双方向 `accepted` のみフレンドとして取得できる
- 片方向 `accepted`、`pending`、不存在を取得できない
- 指定難易度だけを返す
- 削除済み楽曲を除外する
- WORLD'S ENDを除外する
- 両者未プレイの譜面も返す
- 片方のみレコードがある場合に他方がNULLになる
- 両者レコードがある場合に同じ譜面行へ結合される
- 譜面数にかかわらず単一クエリで比較行を取得する
- `songs.id ASC` の決定的な順序になる

## 11.3 Handlerテスト

- 正常レスポンスのDTO形状
- `friend_user_id` の形式不正
- 難易度の不足・不正
- 未認証
- `friend_not_found` のHTTP 404マッピング
- `friend_score_comparison_unavailable` のHTTP 409マッピング

## 11.4 性能確認

- 最大想定譜面数でレスポンス件数とサイズを計測する
- `EXPLAIN ANALYZE` で両 `player_records` JOINが主キーを使用することを確認する
- 譜面数に比例した追加クエリがないことを確認する

---

## 12. 既存機能との関係

### 12.1 フレンドランキングAPI

既存の譜面単位フレンドランキングAPIは変更しません。

- フレンドランキング: 1譜面について自分と全フレンドを比較
- VSフレンド: 1フレンドについて指定難易度の全譜面を比較

取得軸が異なるため、同じエンドポイントやレスポンスへ統合しません。ただし、双方向 `accepted` の判定条件、ランプの `NONE` から `null` への変換など、共通仕様は一致させます。

### 12.2 ユーザー全レコードAPI

`GET /internal/users/:username/record` は変更しません。VSフレンド画面からは本専用APIだけを呼び、比較のために2人分の全レコードAPIを追加取得しません。

---

## 13. APIドキュメント更新

実装時に次を更新します。

- `docs/API.md`
  - エンドポイント一覧
  - `/internal/friend-comparisons` グループ
  - リクエスト、レスポンス、比較ルール、エラー仕様
- `docs/friendship.md`
  - 承認済みフレンドだけが比較対象であること
- `internal/app/apierror/codes.go`
  - `friend_not_found`
  - `friend_score_comparison_unavailable`
- フロントエンド `src/types/api.ts`
  - 新規レスポンスDTO
  - 新規エラーコード

---

## 14. 実装タスク分解

1. Domainの読み取りモデルとQueryService interfaceを追加
2. Repositoryのフレンドペア取得クエリを実装
3. Repositoryの指定難易度比較クエリを実装
4. Usecaseの未プレイ正規化、勝敗判定、summary集計をTDDで実装
5. API DTOとHandlerを追加
6. Routerの認証必須グループへルートを追加
7. 新規APIエラーを追加
8. `docs/API.md` と `docs/friendship.md` を更新
9. Repository・Usecase・Handlerテストを追加
10. `go test ./...` を実行
11. `gofmt -s -w .` を実行
12. 実データ相当件数でクエリ計画、レスポンスサイズ、メモリを確認

---

## 15. 受け入れ条件

- 承認済み双方向フレンドだけを比較できる
- 1リクエストで指定できる難易度は1つだけである
- 指定難易度の全有効通常譜面が1件ずつ返る
- 両者未プレイを含むすべての譜面が勝敗集計に含まれる
- 未プレイはスコア0、ランプと更新日時nullで返る
- 同スコアはプレイ状態やランプにかかわらず引き分けになる
- `summary` と `items` から再計算した値が一致する
- WORLD'S ENDと削除済み楽曲が混入しない
- DBアクセスにN+1がない
- 既存のフレンドランキングAPIとユーザーレコードAPIに破壊的変更がない

---

## 16. 将来検討

本設計の初期実装には含めません。

- WORLD'S END比較
- コース比較
- 複数難易度の同時比較
- 勝敗、未プレイ状態、譜面定数によるサーバー側絞り込み
- サーバー側並び替え
- カーソルページング
- 比較結果キャッシュ
- スコア履歴の時系列比較
- 勝敗推移や期間指定集計
