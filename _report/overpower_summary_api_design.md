# OVER POWER集計API 設計書

## 概要

`_report/overpower_summary_api_plan.md` に基づき、本人向け OVER POWER 集計API の詳細設計を定義する。

初版では認証済みユーザー自身の集計結果のみを返し、未解禁曲管理や公開プロフィール向け提供は含めない。

## エンドポイント

- `GET /internal/me/overpower-summary`

計画書では `POST` 案が示されていたが、本APIは副作用を持たない集計結果取得であり、リクエストボディも不要なため `GET` を採用する。

## 認証

- Cookie 認証必須
- 未認証時は `401 Unauthorized`

## レスポンス責務

以下の25項目を返す。

- 全曲: 1
- ジャンル別: 7
- 難易度別: 5
- レベル別: 12

各項目は以下の情報を持つ。

- `current_op`
- `max_op`
- `percent`
- `target_count`
- `played_count`

レスポンス全体には `updated_at` を含める。

## レスポンス構造

```json
{
  "updated_at": "2026-03-25T12:34:56Z",
  "overall": {
    "current_op": 12345.67,
    "max_op": 23456.78,
    "percent": 52.63,
    "target_count": 1234,
    "played_count": 1200
  },
  "genres": {
    "POPS & ANIME": {
      "current_op": 0,
      "max_op": 0,
      "percent": 0,
      "target_count": 0,
      "played_count": 0
    }
  },
  "difficulties": {
    "MASTER": {
      "current_op": 0,
      "max_op": 0,
      "percent": 0,
      "target_count": 0,
      "played_count": 0
    }
  },
  "levels": {
    "14+": {
      "current_op": 0,
      "max_op": 0,
      "percent": 0,
      "target_count": 0,
      "played_count": 0
    }
  }
}
```

## レスポンスキー

### `genres`

以下の固定キーを返す。

- `POPS & ANIME`
- `niconico`
- `東方Project`
- `VARIETY`
- `イロドリミドリ`
- `ゲキマイ`
- `ORIGINAL`

### `difficulties`

以下の固定キーを返す。

- `BASIC`
- `ADVANCED`
- `EXPERT`
- `MASTER`
- `ULTIMA`

### `levels`

以下の固定キーを返す。

- `10`
- `10+`
- `11`
- `11+`
- `12`
- `12+`
- `13`
- `13+`
- `14`
- `14+`
- `15`
- `15+`

クライアントが空カテゴリ判定のために存在確認をしなくて済むよう、対象件数0でもキーは省略しない。

## Usecase 設計

`OverpowerSummaryUsecase` を新設する。

責務:

- 認証済みユーザーから `player_id` を解決する
- 通常譜面レコードを取得する
- 通常楽曲マスタを取得する
- 25項目の集計を行う
- レスポンスDTOへ変換する

既存の `UserUsecase` や `PlayerDataUsecase` へ責務を混在させない。

### 想定インターフェース

```go
type OverpowerSummaryUsecase interface {
    Get(ctx context.Context, user *entity.User) (*dtoapiinternal.OverpowerSummaryResponse, error)
}
```

## 利用する Repository

- `PlayerRecordRepository.FindByPlayerID`
- `SongRepository.FindAllExcludingWorldsend`

初版では追加クエリを作らず、Usecase 内で集計する。

## 集計の前提データ

### プレイヤーレコード

- 通常譜面レコードのみ対象
- WORLD'S END は対象外
- 各レコードのスコア、コンボランプ、譜面情報を用いて単譜面 OP を計算する

### 楽曲マスタ

- 通常楽曲のみ対象
- 削除済み楽曲は対象外
- 楽曲ごとのジャンル、譜面ごとの難易度・定数を利用する

## 集計アルゴリズム

1. `user.PlayerID` が未設定なら `ErrPlayerNotLinked` を返す
2. プレイヤーの通常譜面レコードを取得する
3. 通常楽曲マスタを取得する
4. 楽曲IDと難易度IDでレコードを索引化する
5. 全通常楽曲を走査し、譜面ごとに以下を求める
   - 理論最大単譜面 OP
   - 現在の単譜面 OP
   - プレイ済みかどうか
6. 各楽曲について楽曲単位集計用の値を求める
   - `song_current_op`: その楽曲内の現在単譜面 OP 最大値
   - `song_max_op`: その楽曲内の理論最大単譜面 OP 最大値
   - `song_played`: 1譜面以上プレイ済みなら true
7. 全曲・ジャンル別へ楽曲単位で加算する
8. 難易度別・レベル別へ譜面単位で加算する
9. `percent = current_op / max_op * 100` を計算する
10. DTO に変換して返却する

## 楽曲単位集計ルール

対象:

- `overall`
- `genres[*]`

各楽曲につき採用するのは単譜面 OP 最大の1譜面のみ。

- `current_op`: その楽曲で現在獲得している単譜面 OP の最大値
- `max_op`: その楽曲の理論最大単譜面 OP の最大値
- `target_count`: 対象楽曲数
- `played_count`: 1譜面以上プレイ済みの楽曲数

## 譜面単位集計ルール

対象:

- `difficulties[*]`
- `levels[*]`

各譜面は一意にカテゴリへ属するため、そのまま加算する。

- `current_op`: 対象譜面の現在 OP 合計
- `max_op`: 対象譜面の理論最大 OP 合計
- `target_count`: 対象譜面数
- `played_count`: プレイ済み譜面数

## レベル別分類

譜面定数 `const` から以下で求める。

- `bucket = floor(const * 2) / 2`

対応:

- `10.0` 以上 `10.5` 未満 -> `10`
- `10.5` 以上 `11.0` 未満 -> `10+`
- 中略
- `15.0` 以上 `15.5` 未満 -> `15`
- `15.5` 以上 -> `15+`

初版の集計対象は `10` から `15+` までとし、それ未満の譜面はレベル別集計に含めない。

## `updated_at` の定義

`updated_at` はユーザーの通常譜面レコード群の最終更新日時を返す。

候補:

- プレイヤーデータ登録時の `players.updated_at`
- 対象レコードの最大 `updated_at`

初版は追加クエリを避けるため、既存のプレイヤーサマリーと整合しやすい `players.updated_at` を返す設計とする。

## 小数の扱い

- OP 計算は既存の `service.CalcSingleOverpower` を利用する
- `current_op` / `max_op` は `float64`
- JSON では丸め固定をせず、Go の `encoding/json` 標準出力に従う
- `percent` は `max_op == 0` の場合のみ `0`

## エラー方針

- `ErrPlayerNotLinked` -> `player_not_linked`
- プレイヤー取得不可 -> `player_not_found`
- 楽曲・レコード取得失敗 -> `internal_error`

入力パラメータが存在しないため、専用のバリデーションエラーは設けない。

## DTO 設計

### 集計1項目

```go
type OverpowerSummaryItem struct {
    CurrentOP   float64 `json:"current_op"`
    MaxOP       float64 `json:"max_op"`
    Percent     float64 `json:"percent"`
    TargetCount int     `json:"target_count"`
    PlayedCount int     `json:"played_count"`
}
```

### レスポンス

```go
type OverpowerSummaryResponse struct {
    UpdatedAt    time.Time                       `json:"updated_at"`
    Overall      OverpowerSummaryItem            `json:"overall"`
    Genres       map[string]OverpowerSummaryItem `json:"genres"`
    Difficulties map[string]OverpowerSummaryItem `json:"difficulties"`
    Levels       map[string]OverpowerSummaryItem `json:"levels"`
}
```

## Handler 設計

`MeHandler` へ追加するより、責務分離のため専用 handler を追加する。

候補:

- `internal/app/handler/api_internal/overpower_summary_handler.go`

責務:

- 認証済みユーザーの取得
- Usecase 呼び出し
- Usecase エラーの API エラー変換
- JSON レスポンス返却

## Router 設計

`/internal/me` 配下に以下を追加する。

```go
me.GET("/overpower-summary", handlers.OverpowerSummary.Get)
```

## テスト方針

### Usecase テスト

以下を最低限確認する。

- 楽曲単位集計で同一楽曲の最大単譜面 OP のみ採用される
- ジャンル別も楽曲単位で集計される
- 難易度別は譜面単位で集計される
- レベル別は `floor(const * 2) / 2` で分類される
- `10` 未満はレベル別に含まれない
- 未プレイ譜面・未プレイ楽曲の `played_count` が正しい
- `user.PlayerID == nil` で `ErrPlayerNotLinked`

### Handler テスト

以下を確認する。

- 認証済みで 200 を返す
- 未認証で 401 を返す
- `ErrPlayerNotLinked` を 404 に変換する

## 将来拡張

後続では以下を追加しやすい構成にしておく。

- 公開プロフィール向け `GET /v1/users/:username/overpower-summary`
- 未解禁曲考慮版の別集計
- マスタキャッシュまたは集計結果キャッシュ
