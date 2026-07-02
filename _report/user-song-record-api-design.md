# 楽曲単位ユーザーレコード API 設計・実装計画書

## 0. 現行実装との差分（2026-07-01 時点）

本書は **未実装の設計案** です。現行のユーザーレコード取得 API は以下のみで、楽曲単位の絞り込みはできません。

| エンドポイント | 粒度 |
| -------------- | ---- |
| `GET /internal/users/:username/record` | ユーザー全レコード（`include_noplay` 任意） |
| `GET /internal/users/:username` (`?view=record`) | 同上 |
| `GET /v1/users/:username` | 同上（外部 API） |

譜面単位で取得できる既存 API は **スコア履歴 API** のみです。`PlayerRecordDTO` 相当の現行レコード（`rating` / `overpower` / `slot` / `is_op_target` 等）は返しません。

フロントエンドの IndexedDB 細分化は本 API 実装後に別途検討します。本計画のスコープは **API サーバー側のエンドポイント増設** に限定します。

---

## 1. 目的

楽曲詳細画面など「特定 1 曲の自己レコード」だけが必要な用途に対し、全件 `GET /users/:username/record` を毎回呼ばずに済むよう、**楽曲単位のユーザーレコード取得 API** を追加する。

### 達成したいこと

1. レスポンスサイズと DB 負荷を全件取得より抑える（`_report/refactor.md` の PERF-003 / PERF-004 の一部緩和）
2. 既存の全件 API は維持し、破壊的変更を避ける
3. 全件 API と **同じ DTO・同じ変換ロジック** を再利用し、フィールド差異を生まない
4. 将来のフロントエンド IndexedDB キャッシュ（曲単位）の前提 API とする

### 非目的

- 既存全件レコード API の削除・非推奨化
- 譜面単位専用エンドポイントの新設（任意の `difficulty` クエリで代替）
- v1 外部 API への同時追加（初期リリースは internal のみ。必要なら後続フェーズで追加）
- ページネーション付きレコード一覧 API

---

## 2. 背景と設計判断

### 2.1 なぜ楽曲単位か

前段の検討により、細分化 API の主軸は **楽曲単位** とする。

| 観点 | 楽曲単位 | 譜面単位 |
| ---- | -------- | -------- |
| 楽曲詳細（複数難易度を一括表示） | 1 リクエストで足りる | 最大 5 リクエストが必要 |
| `include_noplay` | その曲の譜面枠を補完する定義が自然 | 未プレイ 1 譜面の扱いが曖昧 |
| WORLD'S END | 1 曲 = 1 譜面でそのまま対応 | 実質同じ |
| キャッシュキー | `username` + `display_id` | `username` + `display_id` + `difficulty` |

譜面だけ欲しい場合は、楽曲単位 API に **`difficulty` クエリ（任意）** を足してレスポンスを絞る。専用パスは初期実装では設けない。

### 2.2 既存 API との役割分担

| API | 用途 |
| --- | ---- |
| `GET /users/:username/record` | レコード一覧・OVER POWER・目標画面など全集が必要な画面 |
| **新 API（本書）** | 楽曲詳細の自己スコア表示など 1 曲分で足りる画面 |
| `GET /songs/:displayid/score-history/...` | 譜面のスコア履歴（現行ベスト + 過去ベスト）。レコード DTO とは別ドメイン |

---

## 3. エンドポイント定義

通常楽曲と WORLD'S END はマスタ・リポジトリが分かれているため、エンドポイントも分離する。ユーザーデータ系 API の命名規則（`/users/:username/...`）に揃える。

### 3.1 通常楽曲

- **Method**: `GET`
- **Path**: `/internal/users/:username/record/songs/:displayid`
- **認証**: Firebase Bearer（任意）— 既存 `GET /users/:username/record` と同じ
- **レートリミット**: 認証なしで 1 分間 60 回/IP（`publicUsersGroup` の `anonymousRateLimit` を継承）
- **概要**: 指定ユーザーの、指定通常楽曲に属する譜面レコードを返す

### 3.2 WORLD'S END 楽曲

- **Method**: `GET`
- **Path**: `/internal/users/:username/record/worldsend-songs/:displayid`
- **認証・レートリミット**: 3.1 と同じ
- **概要**: 指定ユーザーの、指定 WORLD'S END 楽曲のレコードを返す（0 または 1 件）

### 3.3 クエリパラメータ（共通）

| パラメータ | 必須 | 説明 |
| ---------- | ---- | ---- |
| `include_noplay` | 任意 | `true` のとき、全件 API と同様に未プレイ譜面を補完して返す。補完エントリは `is_played=false`、`updated_at` / ランプ類は `null` |
| `difficulty` | 任意 | 通常楽曲のみ有効。大文字小文字不問。`BASIC` / `ADVANCED` / `EXPERT` / `MASTER` / `ULTIMA` のいずれか。指定時は該当難易度の譜面だけ返す。WORLD'S END では無視する（将来 400 にしてもよいが、初期は黙って無視） |

---

## 4. レスポンス設計

### 4.1 通常楽曲

既存 `PlayerRecordDTO` をそのまま使う。全件 API の `standard` 配列の **部分集合**（最大 5 件）とする。

```json
{
  "standard": [
    {
      "is_played": true,
      "is_op_target": true,
      "updated_at": "2026-06-20T10:00:00Z",
      "difficulty": "MASTER",
      "id": "0000000000000001",
      "title": "楽曲名",
      "artist": "アーティスト名",
      "const": 14.5,
      "is_const_unknown": false,
      "score": 1009500,
      "rating": 17.14,
      "overpower": 5.67,
      "justice_count": null,
      "overpower_percent": 98.2857,
      "img": "https://example.com/jacket.png",
      "clear_lamp": "CLEAR",
      "combo_lamp": "FULL COMBO",
      "full_chain": null,
      "slot": null
    }
  ],
  "meta": {
    "updated_at": "2026-06-20T10:00:00Z"
  }
}
```

#### 新規 DTO（案）

```go
// UserSongRecordDTO は楽曲単位の通常譜面レコードレスポンスです。
type UserSongRecordDTO struct {
    Standard []*dto.PlayerRecordDTO `json:"standard"`
    Meta     *UserSongRecordMetaDTO `json:"meta"`
}

type UserSongRecordMetaDTO struct {
    UpdatedAt *time.Time `json:"updated_at"`
}
```

`meta.updated_at` は **返却した `standard` 内のプレイ済みレコード** の `updated_at` 最大値。プレイ済みが 1 件もなければ `null`。

#### WORLD'S END

```go
type UserWorldsendSongRecordDTO struct {
    Worldsend *dto.WorldsendRecordDTO   `json:"worldsend"` // 未プレイ補完時もエントリあり。レコード自体が無ければ null
    Meta      *UserSongRecordMetaDTO    `json:"meta"`
}
```

`worldsend` は 1 曲 1 譜面のためオブジェクト単体とする（全件 API の配列とは形が異なるが、件数は常に 0〜1 であり楽曲詳細用途に合う）。フロントが配列前提の場合は `[worldsend]` に包む薄いアダプタをクライアント側で置く。

---

## 5. エラー・境界条件

既存ユーザーレコード API と同じポリシーを踏襲する。

| 条件 | HTTP | エラーコード（案） |
| ---- | ---- | ------------------ |
| `username` が不正 | 400 | `validation_failed` |
| `displayid` が不正フォーマット | 400 | `validation_failed` |
| `difficulty` が無効（通常楽曲 API） | 400 | `invalid_difficulty` |
| ユーザー不存在 / 非公開で閲覧不可 | 404 | `user_not_found` |
| 楽曲不存在（削除済み含む） | 404 | `song_not_found` |
| 通常楽曲 API に WORLD'S END の `displayid` を指定 | 404 | `song_not_found` |
| WORLD'S END API に通常楽曲の `displayid` を指定 | 404 | `song_not_found` |
| プレイヤー未連携 | 200 | `standard: []` または `worldsend: null`、`meta.updated_at: null` |

**404 にしないケース**: 楽曲は存在するがプレイデータが無い（`include_noplay=false`）→ `200` で空配列 / `null`。

---

## 6. ドメイン・実装方針

### 6.1 レイヤー構成

```
router.go
  └─ UserHandler.GetUserSongRecord / GetUserWorldsendSongRecord
       └─ UserUsecase.GetUserSongRecord / GetUserWorldsendSongRecord
            ├─ getAccessibleUser（既存）
            ├─ getOptionalPlayer（既存）
            ├─ songRepo.FindByDisplayID / worldsendChartRepo.FindByDisplayID
            ├─ playerRecordRepo.FindByPlayerIDAndSongDisplayID（新規）
            ├─ worldsendRecordRepo.FindByPlayerIDAndSongDisplayID（新規）
            ├─ recordCompletionSvc.CompletePlayerRecords（既存・曲 1 件だけ渡す）
            ├─ recordCompletionSvc.CompleteWorldsendRecords（既存・曲 1 件だけ渡す）
            ├─ markOPTargetPlayerRecords（既存・当該曲分のみで可）
            └─ dto.ToPlayerRecordDTO / dto.ToWorldsendRecordDTO（既存）
```

### 6.2 リポジトリ追加

#### `PlayerRecordRepository`

```go
// FindByPlayerIDAndSongDisplayID はプレイヤーと楽曲 display_id で通常譜面レコードを取得します。
FindByPlayerIDAndSongDisplayID(ctx context.Context, exec Executor, playerID int, displayID string) ([]*entity.PlayerRecord, error)
```

- 既存 `playerRecordQuery` に `AND s.display_id = ?` を追加した専用クエリ
- `s.is_deleted = 0` は既存クエリを踏襲
- slot による絞り込みはしない（全件 API と同じ）

#### `WorldsendRecordRepository`

```go
// FindByPlayerIDAndSongDisplayID はプレイヤーと WORLD'S END 楽曲 display_id でレコードを取得します。
FindByPlayerIDAndSongDisplayID(ctx context.Context, exec Executor, playerID int, displayID string) ([]*entity.PlayerWorldsendRecord, error)
```

- 0 件または 1 件を返す想定

#### インデックス

既存の `idx_player_records_chart_id`、`charts.song_id` 結合で十分と想定。実装後に `EXPLAIN` で確認し、必要なら `(player_id, chart_id)` 経由ではなく `s.display_id` 条件付きの実行計画を計測する。

### 6.3 未プレイ補完（`include_noplay`）

全件 API は `completePlayerRecords` で **全曲マスタ** を走査しているが、楽曲単位 API では対象曲だけ渡す。

```go
// 疑似コード
song, err := songRepo.FindByDisplayID(ctx, exec, displayID)
records, err := playerRecordRepo.FindByPlayerIDAndSongDisplayID(ctx, exec, playerID, displayID)
if includeNoPlay {
    records = recordCompletionSvc.CompletePlayerRecords(records, []*entity.Song{song}, ...)
}
```

`RecordCompletionService` のシグネチャ変更は不要。WORLD'S END も同様に `[]*entity.WorldsendSongWithChart{songChart}` を渡す。

### 6.4 `is_op_target` の算出

`markOPTargetPlayerRecords` は曲ごとに最高 OVERPOWER の譜面を選ぶ。対象が 1 曲に限定されていても、**その曲の譜面集合内で同じロジックが成立する**ため、全件レコードの再取得は不要。

### 6.5 `difficulty` クエリの適用タイミング

1. リポジトリは曲単位で取得（DB では絞らない）
2. DTO 化前に `difficulty` でフィルタ
3. `include_noplay=true` かつ `difficulty` 指定時は、**指定難易度の譜面が曲に存在する場合のみ**未プレイ補完する（他難易度は返さない）

曲に存在しない難易度を指定した場合は `400 invalid_difficulty` とする（マスタ上その難易度の chart が無い）。

### 6.6 Usecase インターフェース追加

`internal/usecase/user_usecase.go` に以下を追加する。

```go
GetUserSongRecord(ctx context.Context, username string, requester *entity.User, displayID string, includeNoPlay bool, difficulty string) (*api_internal.UserSongRecordDTO, error)
GetUserWorldsendSongRecord(ctx context.Context, username string, requester *entity.User, displayID string, includeNoPlay bool) (*api_internal.UserWorldsendSongRecordDTO, error)
```

### 6.7 ルーティング

`internal/app/router.go` の `publicUsersGroup` に追加する。既存 `/:username/record` より **具体パスを先に登録** する（Echo のルート順序に注意）。

```go
publicUsersGroup.GET("/:username/record/songs/:displayid", handlers.User.GetUserSongRecord)
publicUsersGroup.GET("/:username/record/worldsend-songs/:displayid", handlers.User.GetUserWorldsendSongRecord)
```

---

## 7. テスト計画

TDD で進める。既存 `user_usecase_impl_test.go` / `user_handler_test.go` のパターンを踏襲する。

### 7.1 リポジトリテスト

- 指定 `display_id` のレコードのみ返ること
- 他曲のレコードが混ざらないこと
- 0 件時にエラーではなく空配列になること

### 7.2 Usecase テスト

| ケース | 期待 |
| ------ | ---- |
| 公開ユーザー・プレイ済みあり | 該当曲の譜面のみ DTO 化 |
| `include_noplay=true` | 未プレイ譜面が補完される |
| `difficulty=MASTER` | MASTER のみ返る |
| プレイヤー未連携 | 空レスポンス |
| 非公開ユーザー・他人 | `ErrUserNotFound` |
| 存在しない `displayid` | `ErrSongNotFound`（新規エラー定義） |
| 通常 API に WE の ID | `ErrSongNotFound` |

### 7.3 Handler テスト

- クエリパラメータのパース（`include_noplay`, `difficulty`）
- ステータスコードと JSON 形状
- 既存 `handleUserProfileError` との整合

---

## 8. ドキュメント更新

実装完了時に以下を更新する。

| ファイル | 内容 |
| -------- | ---- |
| `docs/API.md` | エンドポイント定義・リクエスト/レスポンス例・エラー一覧 |
| `_report/refactor.md` | PERF-003 / PERF-004 に「楽曲単位 API 追加済み」の旨を追記するか、残課題を更新 |

---

## 9. 実装タスク（PR 分割案）

依存関係順に 3 PR に分割することを推奨する。

### PR-1: リポジトリ層

- [ ] `PlayerRecordRepository.FindByPlayerIDAndSongDisplayID` 追加
- [ ] `WorldsendRecordRepository.FindByPlayerIDAndSongDisplayID` 追加
- [ ] infra 実装 + リポジトリテスト

### PR-2: Usecase + DTO

- [ ] `UserSongRecordDTO` / `UserWorldsendSongRecordDTO` 定義
- [ ] `GetUserSongRecord` / `GetUserWorldsendSongRecord` 実装
- [ ] `ErrSongNotFound`（または既存エラー再利用）の整理
- [ ] usecase テスト

### PR-3: Handler + ルーティング + API ドキュメント

- [ ] `UserHandler` 2 メソッド追加
- [ ] `router.go` 登録
- [ ] handler テスト
- [ ] `docs/API.md` 更新

---

## 10. 将来拡張（本計画のスコープ外）

- **v1 外部 API**: internal 実装・運用確認後に `GET /v1/users/:username/record/songs/:displayid` 等を検討
- **フロント IndexedDB**: 曲単位キャッシュ store、`updated-at` 連携はフロント側別計画
- **バッチ取得**: `display_id` の複数指定（`?display_ids=a,b,c`）は楽曲詳細以外の需要が出るまで見送り
- **全件 API の内部最適化**: 本 API とは独立に、全件側の `include_noplay` 時の全曲マスタ走査（`completePlayerRecords`）の効率化は別課題

---

## 11. 未決事項

実装着手前に以下を確定する。

| # | 論点 | 推奨（デフォルト案） |
| - | ---- | -------------------- |
| 1 | WORLD'S END レスポンスをオブジェクト単体にするか配列にするか | オブジェクト単体（0〜1 件のため） |
| 2 | 通常 API で WE の `displayid` を渡したときのエラー | `404 song_not_found`（パスで種別を分けているため） |
| 3 | `difficulty` 未対応難易度（曲に chart 無し） | `400 invalid_difficulty` |
| 4 | 初期リリースで v1 も同時追加するか | internal のみ（フロントは internal 利用） |

---

## 12. 参考：現行フロントの呼び出し（実装動機）

楽曲詳細は現状、全件キャッシュ API から曲を `find` している。

```ts
// chunisupport-front/src/pages/songs/SongDetail/SongDetail.tsx
fetchUserRecordWithCache(username) // 全レコード
records.find((c) => c.id === song.id && c.difficulty === difficulty && c.is_played)
```

本 API 実装後は、楽曲詳細のみ `GET /users/:username/record/songs/:displayid` へ切り替える想定。切り替え自体はフロント別タスクとする。