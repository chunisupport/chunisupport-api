# chunisupport-api API仕様書

このドキュメントは `chunisupport-api` が提供する内部API(`/internal` プレフィックス)、公開API(`/v1` プレフィックス)、chunirec互換API(`/compat/chunirec/2.0` プレフィックス)の仕様をまとめたものです。

**最終更新日**: 2026年06月15日

## ベースURLと環境

アプリケーションは `.config/<APP_ENV>.settings.json` の `app_port` で待ち受けポートを決定します。`APP_ENV=<name> go run main.go` で環境を切り替えます。

ローカル開発の例: `.config/<APP_ENV>.settings.json` で `app_port: 3002` を指定している場合、`http://localhost:3002`

主要なパス構成:

- 監視用API: `http://localhost:<app_port>/`
- 内部向けAPI: `http://localhost:<app_port>/internal`
- 公開API (APIトークン認証): `http://localhost:<app_port>/v1`
- chunirec互換API (APIトークン認証): `http://localhost:<app_port>/compat/chunirec/2.0`

## CORS

すべてのエンドポイントでCORSが有効です。基本設定は `cors.*` を参照してください（設定方法は `docs/configuration.md` を参照）。
ただし `GET /`、`OPTIONS /`、`POST /internal/player-data/temp`、`OPTIONS /internal/player-data/temp` は、設定された許可オリジンに加えて `https://new.chunithm-net.com` も常に許可します。

## 認証

### 内部API (`/internal`)

- 認証必須エンドポイントでは `Authorization: Bearer <Firebase ID Token>` を送信します。
- 認証必須エンドポイントでは Firebase ID トークンを検証し、ユーザー情報をリクエストコンテキストに格納します。
- Bearer 任意のエンドポイントでは、未認証時にレートリミットが適用されます。
- `token` Cookie や独自セッションは使用しません。

### 公開API (`/v1`, `/compat/chunirec/2.0`)

- `Authorization: Bearer <token>` ヘッダーで API トークンを送信します。
- `/v1` と `/compat/chunirec/2.0` はどちらも API トークン認証です。
- トークンは `/internal/auth/api-tokens` で発行します。

## レートリミット（現行実装値）

ルーター実装（`internal/app/router.go`）および定数定義（`internal/info/info.go`）に基づく主要なレートリミットは以下です。

- `/internal/auth/signup`: **1分あたり5回/IP**
- `/internal/me/register-data`: **30秒あたり1回/ユーザー**
- `/internal/player-data/temp`: **1分あたり30回/IP**
- `/internal/player-data/commit`: **30秒あたり1回/ユーザー**
- `/internal/users/*` および `/internal/songs/*` の公開参照系（Firebase Bearer任意）: **未認証時のみ1分あたり10回/IP**
- `/v1/*`: **15分あたり150回（一般ユーザー） / 150,000回（ADMIN）**
- `/compat/chunirec/2.0/*`: **`/v1` と同一**

実際の制限値を変更した場合は、`internal/info/info.go` と本ドキュメントの両方を更新してください。

## 共通レスポンス仕様

- コンテンツタイプは `application/json`。
- カスタムエラーハンドラーは以下形式を返します。

```json
{
  "error": {
    "status": 401,
    "code": "invalid_token",
    "message": "...",
    "details": [
      {
        "field": "username",
        "message": "5〜50文字の小文字英数字で入力してください。"
      }
    ]
  }
}
```

`error` オブジェクト内の `code` フィールドには機械処理しやすいスネークケースのエラーコードが入ります。`status` フィールドにはHTTPステータスコードが入ります。`validation_failed` の場合のみ、入力フォーマット修正のための安全な `message` と `details` を返すことがあります（認証成否や内部状態などの機微情報は含みません）。

## エラーコード一覧（主要）

主要なエラーコードは以下の通りです。全一覧は `internal/app/apierror/codes.go` を参照してください。

| エラーコード | 説明 |
| --- | --- |
| `bad_request` | リクエスト形式不正（JSONパースエラーなど） |
| `validation_failed` | 入力バリデーション失敗 |
| `unauthorized` | 認証が必要 |
| `invalid_token` | トークンが不正 |
| `invalid_turnstile_token` | Turnstile トークンが不正 |
| `token_expired` | トークン期限切れ |
| `missing_token` | トークン未指定 |
| `forbidden` | 権限不足 |
| `invalid_credentials` | 認証情報不正 |
| `firebase_uid_already_linked` | Firebase UID が他ユーザーまたは削除済みユーザーに連携済み |
| `username_empty` | ユーザー名が空 |
| `username_too_short` | ユーザー名が短すぎる |
| `username_too_long` | ユーザー名が長すぎる |
| `username_invalid_char` | ユーザー名に使用できない文字が含まれる |
| `not_found` | エンドポイントが見つからない |
| `too_many_requests` | レートリミット超過 |
| `service_unavailable` | サービス利用不可（DB接続失敗など） |
| `internal_error` | 予期しないサーバーエラー |

## マスターデータ概要

主なマスタ定義は `migration/mysql/000001_init_schema.up.sql` に記載されています。

## エンドポイント一覧

| パス | メソッド | 認証 | 概要 |
| ---- | -------- | ---- | ---- |
| `/` | GET | 不要 | アプリケーション名とビルド日を返します |
| `/healthz` | GET | 不要 | 外部監視向けの軽量な死活チェック |
| `/version` | GET | APIトークン(ADMIN) | APIのバージョン識別子取得 |
| `/internal/auth/login` | POST | Firebase Bearer + Turnstile | Firebase IDトークンとTurnstileでログイン検証 |
| `/internal/auth/signup` | POST | Firebase Bearer | Firebase IDトークンで初回ユーザー登録 |
| `/internal/auth/api-tokens` | GET | Firebase Bearer | APIトークン発行状態取得 |
| `/internal/auth/api-tokens` | POST | Firebase Bearer | APIトークン発行 |
| `/internal/auth/api-tokens` | DELETE | Firebase Bearer | APIトークン削除 |
| `/internal/admin/build-info` | GET | Firebase Bearer (ADMIN+) | 管理者画面向けAPIビルド情報取得 |
| `/internal/me` | GET | Firebase Bearer | 自身のユーザー情報 |
| `/internal/me/privacy` | PUT | Firebase Bearer | 非公開設定更新 |
| `/internal/me` | DELETE | Firebase Bearer + X-Reauth-Token | アカウント物理削除 |
| `/internal/me/register-data` | POST | Firebase Bearer | CHUNITHMプレイヤーデータ登録 |
| `/internal/me/player-data` | DELETE | Firebase Bearer | プレイヤー連携を解除し、プレイヤー関連レコードを削除 |
| `/internal/me/locked-songs` | POST | Firebase Bearer | 自分の未解禁曲を登録 |
| `/internal/me/locked-songs/batch` | POST | Firebase Bearer | 自分の未解禁曲をまとめて登録・解除 |
| `/internal/me/locked-songs/:displayid` | DELETE | Firebase Bearer | 自分の未解禁曲を解除 |
| `/internal/player-data/temp` | POST | なし | 未ログインでプレイヤーデータを一時受付（gzip JSON） |
| `/internal/player-data/commit` | POST | Firebase Bearer | 一時受付したプレイヤーデータを確定保存 |
| `/internal/me/goals` | GET | Firebase Bearer | 目標一覧を取得 |
| `/internal/me/goals` | POST | Firebase Bearer | 目標を作成 |
| `/internal/me/goals/:id` | PUT | Firebase Bearer | 目標を更新 |
| `/internal/me/goals/:id` | DELETE | Firebase Bearer | 目標を削除 |
| `/internal/me/record-filters` | GET | Firebase Bearer | 保存済みレコードフィルタ一覧を取得 |
| `/internal/me/record-filters` | POST | Firebase Bearer | レコードフィルタを保存 |
| `/internal/me/record-filters/:id` | PUT | Firebase Bearer | 保存済みレコードフィルタを更新 |
| `/internal/me/record-filters/:id` | DELETE | Firebase Bearer | 保存済みレコードフィルタを削除 |
| `/internal/users/` | GET | Firebase Bearer (ADMIN+) | 全ユーザー一覧取得（プライベート・プレイヤー未紐付けを含む） |
| `/internal/users/:username/profile` | GET | Firebase Bearer (任意) | ユーザー名とプレイヤー情報のみ取得 |
| `/internal/users/:username/updated-at` | GET | Firebase Bearer (任意) | ユーザー関連データの最終更新日時のみ取得 |
| `/internal/users/:username/rating` | GET | Firebase Bearer (任意) | レーティング枠のみ取得 |
| `/internal/users/:username/record` | GET | Firebase Bearer (任意) | レコード枠のみ取得 |
| `/internal/users/:username/locked-songs` | GET | Firebase Bearer (任意) | ユーザーの未解禁曲一覧を取得 |
| `/internal/users/:username` | GET | Firebase Bearer (任意) | プロファイルとレコードを一括取得 |
| `/internal/users/:username` | DELETE | Firebase Bearer (ADMIN+) | ユーザーの物理削除 |
| `/internal/songs/updated-at` | GET | Firebase Bearer (任意) | 楽曲情報キャッシュ用の最終更新日時のみ取得 |
| `/internal/songs` | GET | Firebase Bearer (任意) | WORLD'S END以外の楽曲一覧取得 |
| `/internal/songs/:displayid` | GET | Firebase Bearer (任意) | 楽曲詳細取得 |
| `/internal/songs/:displayid/stats/:difficulty` | GET | Firebase Bearer (任意) | 難易度別楽曲統計取得 |
| `/internal/songs` | POST | Firebase Bearer (ADMIN+) | 楽曲の新規追加 |
| `/internal/songs` | PUT | Firebase Bearer (EDITOR+) | 楽曲情報と譜面情報の一括更新 |
| `/internal/songs/:displayid` | DELETE | Firebase Bearer (ADMIN+) | 楽曲の論理削除 |
| `/internal/songs/:displayid/restore` | POST | Firebase Bearer (EDITOR+) | 楽曲の復活 |
| `/internal/worldsend-songs` | GET | Firebase Bearer (任意) | WORLD'S END楽曲一覧取得 |
| `/internal/worldsend-songs/:displayid` | GET | Firebase Bearer (任意) | WORLD'S END楽曲詳細取得 |
| `/internal/worldsend-songs` | POST | Firebase Bearer (ADMIN+) | WORLD'S END楽曲の新規追加 |
| `/internal/worldsend-songs` | PUT | Firebase Bearer (EDITOR+) | WORLD'S END楽曲情報と譜面情報の一括更新 |
| `/internal/worldsend-songs/:displayid` | DELETE | Firebase Bearer (ADMIN+) | WORLD'S END楽曲の論理削除 |
| `/internal/worldsend-songs/:displayid/restore` | POST | Firebase Bearer (EDITOR+) | WORLD'S END楽曲の復活 |
| `/internal/honors` | GET | Firebase Bearer (ADMIN+) | 称号一覧取得 |
| `/internal/honors/:id` | GET | Firebase Bearer (ADMIN+) | 称号詳細取得 |
| `/internal/honors` | POST | Firebase Bearer (ADMIN+) | 称号の新規追加 |
| `/internal/honors/:id` | PUT | Firebase Bearer (ADMIN+) | 称号の更新 |
| `/internal/honors/:id` | DELETE | Firebase Bearer (ADMIN+) | 称号の物理削除 |
| `/internal/editor/songs` | GET | Firebase Bearer (EDITOR+) | 編集者向け通常楽曲一覧取得（`is_deleted`, `updated_at`, 譜面の `updated_at` を含む） |
| `/internal/editor/songs/:displayid` | GET | Firebase Bearer (EDITOR+) | 編集者向け通常楽曲詳細取得（`is_deleted`, `updated_at`, 譜面の `updated_at` を含む） |
| `/internal/editor/worldsend-songs` | GET | Firebase Bearer (EDITOR+) | 編集者向けWORLD'S END楽曲一覧取得（`is_deleted`, `updated_at`, 譜面の `updated_at` を含む） |
| `/internal/editor/worldsend-songs/:displayid` | GET | Firebase Bearer (EDITOR+) | 編集者向けWORLD'S END楽曲詳細取得（`is_deleted`, `updated_at`, 譜面の `updated_at` を含む） |
| `/internal/master` | GET | 不要 | フロントエンド向けマスターデータ取得 |
| `/internal/master/versions` | GET | 不要 | バージョン一覧取得 |
| `/internal/master/honor-types` | GET | 不要 | 称号タイプ一覧取得 |
| `/v1/songs` | GET | APIトークン | 全楽曲一覧取得（WORLD'S END除く） |
| `/v1/songs` | PUT | APIトークン (EDITOR+) | 楽曲情報と譜面情報の一括更新 |
| `/v1/songs/:displayid` | GET | APIトークン | 楽曲詳細取得 |
| `/v1/songs/:displayid/stats/:difficulty` | GET | APIトークン | 難易度別楽曲統計取得 |
| `/v1/worldsend-songs` | GET | APIトークン | WORLD'S END楽曲一覧取得 |
| `/v1/worldsend-songs/:displayid` | GET | APIトークン | WORLD'S END楽曲詳細取得 |
| `/v1/users/:username` | GET | APIトークン | ユーザープロファイルとレコード取得 |
| `/v1/master/versions` | GET | APIトークン | バージョン一覧取得 |
| `/compat/chunirec/2.0/music/showall` | GET | APIトークン | chunirec互換：全楽曲一覧取得 |
| `/compat/chunirec/2.0/music/show` | GET | APIトークン | chunirec互換：1楽曲情報取得 |
| `/compat/chunirec/2.0/users/show` | GET | APIトークン | chunirec互換：ユーザープロフィール取得 |

---

## 監視用エンドポイント

> **警告**: これらのエンドポイントはアプリケーションの稼働状況を確認するために使用されます。本番環境では、不正な情報漏洩を防ぐため、ネットワーク設定（例: ファイアウォール、ロードバランサ）によってアクセスを内部ネットワークや特定のIPアドレスに制限することが強く推奨されます。

### GET `/`
- **認証**: 不要
- **レスポンス**: 常に 200 OK で、アプリケーション名とビルド日を返します。リビジョン（Git短縮ハッシュ）は公開しません。

```json
{
  "app_name": "chunisupport-api",
  "build_date": "20240528"
}
```

### GET `/healthz`
- **認証**: 不要
- **CORS**:
  - `https://new.chunithm-net.com` からの `GET` / `OPTIONS` を許可します。
  - それ以外の許可オリジンは通常どおり `cors.allow_origins` に従います。
- **チェック内容**: APIプロセスがHTTP応答できることのみを確認します。DBなどの依存サービスは確認しません。
- **レスポンス**:
  - 204 No Content: 空レスポンス

### GET `/version`
- **認証**: APIトークン (ADMIN)
- **レスポンス**:
  - 200 OK: APIのビルド識別子とGoバージョンを返します。

```json
{
  "app_name": "chunisupport-api",
  "build_date": "20240528",
  "commit_hash": "a1b2c3d",
  "go_version": "go1.26.4"
}
```

---

## 管理者向け情報エンドポイント

### GET `/internal/admin/build-info`
- **認証**: Firebase Bearer (ADMIN)
- **概要**: 管理者画面で表示するAPIのビルド情報を取得します。
- **レスポンス**: 200 OK

```json
{
  "app_name": "chunisupport-api",
  "build_date": "20240528",
  "commit_hash": "a1b2c3d",
  "go_version": "go1.26.4"
}
```

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `app_name` | string | APIアプリケーション名 |
| `build_date` | string | ビルド日 |
| `commit_hash` | string | APIのGit短縮コミットハッシュ。開発起動時は `none` |
| `go_version` | string | APIバイナリのGoバージョン |

---

## 認証エンドポイント

### POST `/internal/auth/login`
- **認証**: Firebase Bearer 必須 + Turnstile 必須
- **リクエストヘッダー**: `Authorization: Bearer <Firebase ID Token>`
- **リクエストボディ**:

```json
{
  "turnstile_token": "0.xxxxx"
}
```

| フィールド | 型 | 必須 | バリデーション |
| ---------- | -- | ---- | -------------- |
| `turnstile_token` | string | ✓ | Cloudflare Turnstile の応答トークン |

- **レスポンス**: 200 OK。`UserDTO` を返します。

```json
{
  "username": "sampleuser",
  "account_type": "PLAYER",
  "is_private": false,
  "last_score_update": null
}
```

- **主なエラー**:
  - 400 Bad Request (`bad_request`): リクエスト形式不正（JSONパースエラー）
  - 401 Unauthorized (`missing_token`): Bearerトークン未指定
  - 401 Unauthorized (`invalid_token`): Firebase IDトークンが不正または失効済み、または未登録ユーザー
  - 401 Unauthorized (`invalid_turnstile_token`): Turnstileトークンが不正または検証済み
  - 422 Unprocessable Entity (`validation_failed`): `turnstile_token` 未指定
  - 500 Internal Server Error (`internal_error`): 予期しないサーバーエラー

### POST `/internal/auth/signup`
- **認証**: Firebase Bearer 必須 + Turnstile 必須
- **リクエストヘッダー**: `Authorization: Bearer <Firebase ID Token>`
- **リクエストボディ**:

```json
{
  "username": "sampleuser",
  "turnstile_token": "0.xxxxx"
}
```

| フィールド | 型 | 必須 | バリデーション |
| ---------- | -- | ---- | -------------- |
| `username` | string | ✓ | 5〜50文字、小文字英数字のみ |
| `turnstile_token` | string | ✓ | Cloudflare Turnstile の応答トークン |

- **レスポンス**: 201 Created。`UserDTO` を返します。

```json
{
  "username": "sampleuser",
  "account_type": "PLAYER",
  "is_private": false,
  "last_score_update": null
}
```

- **主なエラー**:
  - 400 Bad Request (`bad_request`): リクエスト形式不正（JSONパースエラー）
  - 400 Bad Request (`username_empty`): ユーザー名が空
  - 400 Bad Request (`username_too_short`): ユーザー名が5文字未満
  - 400 Bad Request (`username_too_long`): ユーザー名が50文字超過
  - 400 Bad Request (`username_invalid_char`): ユーザー名に使用できない文字が含まれている（小文字英数字のみ可）
  - 400 Bad Request (`registration_failed`): ユーザー登録失敗（詳細隠蔽）
  - 401 Unauthorized (`missing_token`): Bearerトークン未指定
  - 401 Unauthorized (`invalid_token`): Firebase IDトークンが不正または失効済み
  - 401 Unauthorized (`invalid_turnstile_token`): Turnstileトークンが不正または検証済み
  - 409 Conflict (`firebase_uid_already_linked`): Firebase UID が既存ユーザーに連携済み
  - 422 Unprocessable Entity (`validation_failed`): `turnstile_token` 未指定
  - 500 Internal Server Error (`internal_error`): 予期しないサーバーエラー

### POST `/internal/auth/api-tokens`
- **認証**: Firebase Bearer 必須
- **レスポンス**: 200 OK

```json
{"token":"plain-text-api-token"}
```

トークンはレスポンスでのみ平文が取得できます。

### GET `/internal/auth/api-tokens`
- **認証**: Firebase Bearer 必須
- **レスポンス**: 200 OK

```json
{
  "has_token": true,
  "created_at": "2026-04-16T12:34:56Z"
}
```

- APIトークンが未発行の場合は `has_token=false`、`created_at=null` を返します。
- `created_at` は現在有効なAPIトークンの発行日時です。再発行した場合はその時刻に更新されます。
- **主なエラー**:
  - 401 Unauthorized (`missing_token` / `invalid_token`): 認証が必要
  - 500 Internal Server Error (`internal_error`): 予期しないサーバーエラー

### DELETE `/internal/auth/api-tokens`
- **認証**: Firebase Bearer 必須
- **レスポンス**: 204 No Content
- 自分のAPIトークンを削除します。トークンが存在しない場合でも204を返します。
- **主なエラー**:
  - 401 Unauthorized (`missing_token` / `invalid_token`): 認証が必要

---

## `/internal/me` グループ

### GET `/internal/me`
- **認証**: Firebase Bearer 必須
- **レスポンス**: `UserDTO`

```json
{
  "username": "sample_user",
  "account_type": "PLAYER",
  "is_private": false,
  "last_score_update": "2025-11-27T12:00:00+09:00"
}
```

**UserDTO スキーマ**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `username` | string | ユーザー名 |
| `account_type` | string | アカウントタイプ (PLAYER, EDITOR, ADMIN) |
| `is_private` | bool | 非公開設定 (true: 非公開, false: 公開) |
| `last_score_update` | string \| null | プレイヤースコアの最終更新日時 (ISO8601)。プレイヤーが紐付いていない場合やレコードが存在しない場合は null |

- 最終スコア更新日時の取得に失敗した場合、このエンドポイントは成功レスポンスを返さずエラーを返します。

### PUT `/internal/me/privacy`
- **認証**: Firebase Bearer 必須
- **リクエストボディ**:

```json
{"is_private": true}
```

- **レスポンス**:

```json
{
  "is_private": true
}
```

- **主なエラー**:
  - 400 Bad Request (`bad_request`): リクエスト形式不正
  - 401 Unauthorized (`missing_token` / `invalid_token`): 認証が必要
  - 404 Not Found (`user_not_found`): ユーザーが見つからない

### DELETE `/internal/me`
- **認証**: Firebase Bearer 必須
- **必須ヘッダ**: `X-Reauth-Token: <再認証直後の Firebase ID トークン>`
- **レスポンス**: 204 No Content。ボディは空です。

ユーザーを物理削除します。ユーザーに紐づく `players` / `player_records` / `player_worldsend_records` / `player_honors` / `api_tokens` も外部キー制約により削除されます。Firebase UID が連携されている場合は Firebase ユーザー削除も試行します（失敗時はサーバーログに記録し、APIレスポンスは成功を維持します）。

このエンドポイントでは通常の Bearer 認証に加えて、退会直前に取得した recent sign-in 済み Firebase ID トークンを `X-Reauth-Token` ヘッダで送る必要があります。バックエンドは `X-Reauth-Token` の `auth_time` が 5 分以内であること、およびトークンの UID が削除対象ユーザーに連携された Firebase UID と一致することを検証します。

- **主なエラー**:
  - 401 Unauthorized (`missing_token` / `invalid_token`): 通常認証が必要
  - 401 Unauthorized (`recent_sign_in_required`): 再認証トークン未指定・不正・期限切れ
  - 401 Unauthorized (`invalid_credentials`): 削除対象アカウントと再認証情報の整合性が取れない認証失敗。詳細理由はレスポンスに含めず、サーバーログで監視します
  - 404 Not Found (`user_not_found`): ユーザーが見つからない
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー（DB削除失敗など）

### DELETE `/internal/me/player-data`
- **認証**: Firebase Bearer 必須
- **レスポンス**: 204 No Content（ボディなし）

ユーザーアカウントは残したまま、`users.player_id` を `NULL` にし、紐づく `players` および `player_records`/`player_worldsend_records`/`player_honors` を物理削除します。削除はトランザクション内で実行され、連携済みでない状態でも冪等に成功します。

- **主なエラー**:
  - 401 Unauthorized (`missing_token` / `invalid_token`): 認証が必要

### GET `/internal/users/:username/locked-songs`
- **認証**: Firebase Bearer 任意
- **概要**: 指定ユーザーのプレイヤーに紐づく未解禁曲一覧を取得します。通常未解禁とULTIMA未解禁は `is_ultima` で区別されます。対象ユーザーが非公開設定の場合、本人以外にはユーザー未発見として扱われます。
- **パスパラメータ**:

| パラメータ | 型 | 説明 |
| ---------- | -- | ---- |
| `username` | string | 対象ユーザー名 |

- **レスポンス**: 200 OK

```json
{
  "items": [
    {
      "display_id": "0000000000000001",
      "title": "楽曲名",
      "is_ultima": false
    },
    {
      "display_id": "0000000000000002",
      "title": "ULTIMA未解禁の楽曲名",
      "is_ultima": true
    }
  ]
}
```

**PlayerLockedSongsResponse フィールド**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `items` | PlayerLockedSongResponseItem[] | 未解禁曲の一覧。未解禁曲がない場合は空配列 |
| `items[].display_id` | string | 楽曲の表示用ID |
| `items[].title` | string | 楽曲名 |
| `items[].is_ultima` | bool | trueの場合はULTIMA譜面のみ未解禁、falseの場合は通常の未解禁 |

- **主なエラー**:
  - 404 Not Found (`user_not_found`): ユーザーが見つからない、または非公開設定で閲覧できない
  - 404 Not Found (`player_not_linked`): プレイヤーデータが連携されていない
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### POST `/internal/me/locked-songs`
- **認証**: Firebase Bearer 必須
- **概要**: 自分のプレイヤーに未解禁曲を登録します。同じ曲・同じ `is_ultima` の登録は冪等に成功します。
- **リクエストボディ**:

```json
{
  "display_id": "0000000000000001",
  "is_ultima": false
}
```

| フィールド | 型 | 必須 | バリデーション |
| ---------- | -- | ---- | -------------- |
| `display_id` | string | ✓ | 楽曲の表示用ID |
| `is_ultima` | bool | - | trueの場合はULTIMA譜面のみ未解禁として登録。省略時はfalse |

- **レスポンス**: 204 No Content（ボディなし）
- WORLD'S END楽曲、削除済み楽曲、存在しない楽曲は登録できません。
- `is_ultima=true` の場合、対象楽曲にULTIMA譜面が存在しないと `chart_not_found` を返します。

- **主なエラー**:
  - 400 Bad Request (`bad_request`): リクエスト形式不正
  - 400 Bad Request (`validation_failed`): `display_id` が未指定または形式不正
  - 401 Unauthorized (`missing_token` / `invalid_token`): 認証が必要
  - 404 Not Found (`player_not_linked`): プレイヤーデータが連携されていない
  - 404 Not Found (`song_not_found`): 楽曲が見つからない、または登録対象外
  - 404 Not Found (`chart_not_found`): ULTIMA譜面が存在しない
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### DELETE `/internal/me/locked-songs/:displayid`
- **認証**: Firebase Bearer 必須
- **概要**: 自分のプレイヤーから指定した未解禁曲を解除します。対象の未解禁曲が存在しない場合でも204を返します。
- **パスパラメータ**:

| パラメータ | 型 | 説明 |
| ---------- | -- | ---- |
| `displayid` | string | 楽曲の表示用ID |

- **クエリパラメータ**:

| パラメータ | 型 | 必須 | 説明 |
| ---------- | -- | ---- | ---- |
| `is_ultima` | bool | - | trueの場合はULTIMA未解禁を解除。省略時はfalse |

- **レスポンス**: 204 No Content（ボディなし）

- **主なエラー**:
  - 400 Bad Request (`bad_request`): `is_ultima` がboolとして解釈できない
  - 400 Bad Request (`validation_failed`): `displayid` が未指定または形式不正
  - 401 Unauthorized (`missing_token` / `invalid_token`): 認証が必要
  - 404 Not Found (`player_not_linked`): プレイヤーデータが連携されていない
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### POST `/internal/me/locked-songs/batch`
- **認証**: Firebase Bearer 必須
- **概要**: 自分のプレイヤーに対して、未解禁曲の登録（`add`）と解除（`delete`）を1リクエストで実行します。
- **リクエストボディ**:

```json
{
  "add": [
    { "display_id": "0000000000000001", "is_ultima": false }
  ],
  "delete": [
    { "display_id": "0000000000000002", "is_ultima": true }
  ]
}
```

| フィールド | 型 | 必須 | 説明 |
| ---------- | -- | ---- | ---- |
| `add` | object[] | - | 追加する未解禁曲の配列 |
| `delete` | object[] | - | 解除する未解禁曲の配列 |
| `add[].display_id` / `delete[].display_id` | string | ✓ | 楽曲の表示用ID |
| `add[].is_ultima` / `delete[].is_ultima` | bool | - | true の場合はULTIMA未解禁を対象 |

- **レスポンス**: 204 No Content（ボディなし）
- **実行順**: `add` を先に実行し、その後 `delete` を実行します。

- **主なエラー**:
  - 400 Bad Request (`bad_request`): リクエスト形式不正
  - 400 Bad Request (`validation_failed`): `display_id` が未指定または形式不正
  - 401 Unauthorized (`missing_token` / `invalid_token`): 認証が必要
  - 404 Not Found (`player_not_linked`): プレイヤーデータが連携されていない
  - 404 Not Found (`song_not_found`): 追加対象の楽曲が見つからない、または登録対象外
  - 404 Not Found (`chart_not_found`): 追加対象で `is_ultima=true` かつ ULTIMA譜面が存在しない
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### POST `/internal/me/register-data`
- **認証**: Firebase Bearer 必須
- **コンテンツタイプ**: 
  - デフォルト（クエリパラメータなし）: `application/octet-stream` または `text/plain`（base64+gzip形式）
  - `?format=json`: `application/json`（デバッグ用、通常は使用しない）
- **レートリミット**: ユーザーIDベースで30秒に1回
- **制限**: リクエストボディ最大5MB（圧縮前のJSONデータに対して適用）。空ボディや余分なデータは 400。ファイルサイズ超過で 413。
- **リクエストボディ**: 
  - **デフォルト形式（推奨）**: JSONデータをgzip圧縮後、base64エンコードした文字列
  - **デバッグ形式（`?format=json`）**: `PlayerDataPayload` 構造に準拠した生JSON。公式アプリのエクスポートJSONをそのまま送信する想定。
  - **未知のフィールドの扱い**: 構造体に定義されていないフィールドは無視されます。将来の互換性のため、クライアント側で追加情報を含めることができます。未知のフィールドが含まれていた場合、サーバーログに警告が記録されますが、エラーにはなりません。

#### リクエスト形式

##### デフォルト形式（base64+gzip）

1. JSONデータをUTF-8でエンコード
2. gzip圧縮（CompressionStream等）
3. base64エンコード
4. POSTリクエストのボディとして送信

フロントエンド実装例（JavaScript）:
```javascript
// 1. JSONをUTF-8エンコード
const encoder = new TextEncoder();
const uint8Array = encoder.encode(JSON.stringify(data));

// 2. gzip圧縮
const compressionStream = new CompressionStream("gzip");
const writer = compressionStream.writable.getWriter();
writer.write(uint8Array);
writer.close();

// 3. 圧縮データを取得
const reader = compressionStream.readable.getReader();
const chunks = [];
while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    chunks.push(value);
}
const totalLength = chunks.reduce((acc, chunk) => acc + chunk.length, 0);
const compressedData = new Uint8Array(totalLength);
let offset = 0;
for (const chunk of chunks) {
    compressedData.set(chunk, offset);
    offset += chunk.length;
}

// 4. base64エンコード
let binary = "";
for (const byte of compressedData) {
    binary += String.fromCharCode(byte);
}
const base64Data = btoa(binary);

// 5. POST
fetch('/internal/me/register-data', {
    method: 'POST',
    headers: {
        Authorization: `Bearer ${firebaseIdToken}`
    },
    body: base64Data,
});
```

##### デバッグ形式（?format=json）

クエリパラメータ `?format=json` を付与し、JSONを直接送信します。
この形式は開発・デバッグ目的でのみ使用してください。

```bash
curl -X POST \
  'http://localhost:8080/internal/me/register-data?format=json' \
  -H 'Content-Type: application/json' \
  -d '{ "app_ver": "0.0.1a", ... }'
```

#### プレイヤーレーティング再計算の仕様

プレイヤーデータ登録時に、保存済みの全スコアから以下の3つのレーティング値を自動計算して `players` テーブルに保存します:

| カラム名 | 型 | 説明 |
| -------- | -- | ---- |
| `calculated_player_rating` | DECIMAL(6,4) | プレイヤーレーティング（ベスト枠30曲 + 新曲枠20曲の加重平均） |
| `best_average_rating` | DECIMAL(6,4) | ベスト枠の平均レーティング（全譜面から上位30曲） |
| `new_average_rating` | DECIMAL(6,4) | 新曲枠の平均レーティング（新曲から上位20曲） |

**計算の詳細**:

1. **新曲の判定**: 
   - スロット名が `new` または `new_candidate` のレコードを新曲として扱います
   - 入力JSONの `slot` フィールドをそのまま使用します（公式アプリの判定結果を信頼）

2. **単曲レーティングの計算**: 
   - CHUNITHMのWiki記載の公式計算式に準拠（実装: [rating_service.go](../internal/domain/service/rating_service.go)）
   - 譜面定数が不明（`is_const_unknown=true`）な譜面も計算に含めます（除外するとより不正確になるため）

3. **プレイヤーレーティングの計算式**:
   ```
   プレイヤーレーティング = (ベスト枠30曲の合計 + 新曲枠20曲の合計) / 50
   ```

4. **ベスト枠平均の計算**:
   - 全譜面から単曲レーティング上位30曲を選択
   - 30曲の平均を算出

5. **新曲枠平均の計算**:
   - 新曲（`slot` が `new` または `new_candidate`）から単曲レーティング上位20曲を選択
   - 20曲の平均を算出

**注意事項**:
- レーティング計算は毎回全レコードを対象に行うため、10万ユーザー規模でも問題なくスケール可能です
- `official_player_rating` は入力データの `rating` フィールドから設定され、`calculated_player_rating` とは独立して保存されます

- **コンテンツタイプ**: `application/json`

#### リクエストボディ例

```json
{
  "app_ver": "0.0.1a",
  "name": "プレイヤー名",
  "level": 217,
  "rating": 17.29,
  "last_played": "2025/11/02 16:42",
  "overpower": {
    "value": 96123.91,
    "percentage": 76.27
  },
  "class_emblem": {
    "medal_class": "06",
    "base_class": "04"
  },
  "team": {
    "name": "チーム名",
    "color": "green"
  },
  "honors": {
    "1": { "title": "称号1", "class": "platina", "img_url": "https://..." },
    "2": { "title": "称号2", "class": "silver", "img_url": "https://..." },
    "3": { "title": "称号3", "class": "normal", "img_url": "https://..." }
  },
  "scores": {
    "standard": [
      {
        "diff": "MAS",
        "idx": "2849",
        "score": 1002345,
        "clear_lamp": "brave",
        "cmb_lv": 2,
        "fch_lv": 1,
        "slot": "best",
        "order": 1
      }
    ],
    "worldsend": [
      {
        "diff": "WE",
        "idx": "8001",
        "score": 990000,
        "clear_lamp": "clear",
        "cmb_lv": 1,
        "fch_lv": 1
      }
    ]
  },
  "updated_at": "2025-11-27T10:30:03+09:00"
}
```

#### リクエストボディスキーマ

| フィールド | 型 | 必須 | 説明 |
| ---------- | -- | ---- | ---- |
| `app_ver` | string | ✓ | インポートアプリのバージョン。対応バージョン: `0.1.0` |
| `name` | string | ✓ | プレイヤー名（全角8文字以内、半角英数字・半角カタカナ不可） |
| `level` | number | ✓ | プレイヤーレベル |
| `rating` | number | ✓ | レーティング |
| `last_played` | string | ✓ | 最終プレイ日時 (`YYYY/MM/DD HH:mm` 形式) |
| `overpower.value` | number | ✓ | オーバーパワー値（互換入力用。登録時は受け取るが保存値には使わず、通常譜面スコアから楽曲OP合計を再計算） |
| `overpower.percentage` | number | ✓ | オーバーパワー割合（互換入力用。登録時は受け取るが保存値には使わず、未解禁設定を除外した通常楽曲の最大OP合計を分母として再計算） |
| `class_emblem.medal_class` | string | ✓ | クラスエンブレム（0埋め2桁） |
| `class_emblem.base_class` | string | ✓ | クラスエンブレムベース（0埋め2桁） |
| `team.name` | string | | チーム名 |
| `team.color` | string | | チームカラー |
| `honors` | object | | 称号情報（キー: スロット番号 "1"〜"3"） |
| `scores.standard` | array | ✓ | 通常譜面スコア配列 |
| `scores.worldsend` | array | ✓ | WORLD'S END スコア配列 |
| `updated_at` | string | ✓ | 更新日時 (ISO8601) |

**スコアエントリスキーマ (`scores.standard` / `scores.worldsend` の各要素)**:

| フィールド | 型 | 必須 | 説明 |
| ---------- | -- | ---- | ---- |
| `diff` | string | ✓ | 難易度 (`BAS`, `ADV`, `EXP`, `MAS`, `ULT`, `WE`) |
| `idx` | string | ✓ | 楽曲の公式インデックス |
| `score` | number | ✓ | スコア (0〜1,010,000) |
| `clear_lamp` | string \| null | | クリアランプ (`clear`, `hard`, `brave`, `absolute`, `catastrophy`, `null`=FAILED) |
| `cmb_lv` | number \| null | | コンボランプ (1=NONE, 2=FULL COMBO, 3=ALL JUSTICE) |
| `fch_lv` | number \| null | | フルチェイン（後方互換のため **1=NONE, 2=PLATINUM, 3=GOLD** として解釈） |
| `slot` | string \| null | | スロット (`best`, `best_candidate`, `new`, `new_candidate`, `null`=none) |
| `order` | number \| null | | スロット内順序 |

- **レスポンス**: 200 OK。登録結果 `PlayerDataResult` を返します。
  - `summary.overpower_value` は通常楽曲レコードから再集計して保存されるOVER POWER値です。
  - `summary.overpower_percentage` は登録処理時点の計算結果です。`players` テーブルには保存されず、プロフィール系レスポンスでは最新マスタデータとプレイヤーの未解禁設定（未解放/解放済みの譜面）を組み合わせて分母を再計算し、その分母を使って随時計算された `overpower_percent` が返ります。

#### レスポンス例

```json
{
  "player_id": 42,
  "app_ver": "0.0.1a",
  "imported_at": "2025-11-27T10:45:00+09:00",
  "profile": {
    "player_id": 42,
    "name": "プレイヤー名",
    "level": 217,
    "rating": 17.29,
    "class_emblem_id": 6,
    "class_emblem_base_id": 4,
    "last_played_at": "2025-11-02T16:42:00+09:00",
    "overpower_value": 96123.91,
    "overpower_percent": 76.27
  },
  "summary": {
    "name": "プレイヤー名",
    "level": 217,
    "rating": 17.29,
    "last_played_at": "2025-11-02T16:42:00+09:00",
    "overpower_value": 96123.91,
    "overpower_percentage": 76.27
  },
  "statistics": {
    "total_high_score": 1183287650,
    "lamp_counts": {
      "clear": {
        "FAILED": 12,
        "CLEAR": 450,
        "HARD": 300,
        "BRAVE": 250,
        "ABSOLUTE": 170,
        "CATASTROPHY": 3
      },
      "combo": {
        "none": 900,
        "full combo": 220,
        "all justice": 65
      },
      "full_chain": {
        "none": 1160,
        "full chain gold": 20,
        "full chain platinum": 5
      }
    }
  },
  "counts": {
    "standard_records_upserted": 1185,
    "worldsend_records_upserted": 120,
    "standard_records_skipped": 0,
    "worldsend_records_skipped": 0,
    "honors_skipped": 0,
    "standard_records_actually_changed": 12,
    "worldsend_records_actually_changed": 3
  },
  "changes": [
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
        "combo_lamp": "full combo",
        "full_chain": null
      }
    },
    {
      "record_type": "worldsend",
      "change_type": "new",
      "idx": "8001",
      "diff": "WE",
      "before": null,
      "after": {
        "score": 990000,
        "clear_lamp": "CLEAR",
        "combo_lamp": null,
        "full_chain": null
      }
    }
  ],
  "skipped_records": [
    {
      "record_type": "standard",
      "reason": "unknown_song",
      "details": "idx=9999"
    }
  ]
}
```

#### レスポンススキーマ

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `player_id` | number | 登録されたプレイヤーID |
| `app_ver` | string | リクエストのアプリバージョン |
| `imported_at` | string | インポート実行日時 (ISO8601) |
| `profile` | object | 登録後のプレイヤープロフィール情報。`class_emblem_id` / `class_emblem_base_id` を含みます |
| `summary` | object | プレイヤーサマリー情報 |
| `statistics` | object | 登録後の通常譜面集計。`total_high_score` とランプごとの件数を含みます |
| `counts` | object | 各種レコードの処理件数。`*_actually_changed` は保存前状態と比較して `new` または `updated` になった件数 |
| `changes` | array | 実際に新規追加または更新されたスコア差分。0件の場合は空配列。詳細は最大100件 |
| `skipped_records` | array | スキップされたレコード情報。0件の場合は空配列 |

`statistics.total_high_score` は削除済み楽曲を除く保存後の通常譜面スコア合計です。WORLD'S ENDは含みません。`statistics.lamp_counts.clear` / `combo` / `full_chain` はランプマスタの `Name` をキーにした件数です。`none` 相当のコンボランプ・フルチェインも集計では `none` キーとして返します。

**`changes` の要素スキーマ**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `record_type` | string | `standard` または `worldsend` |
| `change_type` | string | 未登録レコードは `new`、保存済みレコードの比較対象カラムが変化した場合は `updated` |
| `idx` | string | 楽曲の公式インデックス |
| `diff` | string | 通常譜面は大文字難易度名、WORLD'S END は入力値にかかわらず `WE` |
| `before` | object \| null | 更新前状態。`change_type=new` では `null` |
| `after` | object | 登録後状態 |

`before` / `after` は常に `score`, `clear_lamp`, `combo_lamp`, `full_chain` を含みます。ランプ名はマスタの `Name` を返し、`none` 相当・未設定は `null` です。`slot` / `order` は保存されますが、差分判定および `changes` には含まれません。同一payload内で同じ譜面キーが複数回現れた場合は、最後の1件を保存・差分表示の対象にします。`changes` は `idx` を数値として昇順に並べ、同一 `idx` の場合は `record_type`、`diff` の順で並びます。`idx` を数値として解釈できない値は末尾に並びます。`counts.*_actually_changed` は実際に変化した全件数で、`changes` はレスポンスサイズ抑制のため最大100件です。

- **主なエラー**:
  - 400 Bad Request (`bad_request` / `resource_not_found` / `app_version_unsupported`): JSON構文不備・楽曲マスタ未登録・非対応バージョンなど
  - 401 Unauthorized (`missing_token` / `invalid_token`): Bearerトークン欠如または無効
  - 409 Conflict (`conflict`): 別ユーザーのプレイヤーデータと競合
  - 413 Request Entity Too Large (`payload_too_large`): ボディサイズ5MB超過
  - 422 Unprocessable Entity (`validation_failed`): バリデーションエラー（スコア範囲外など）

---

## `/internal/player-data` グループ

### POST `/internal/player-data/temp`

プレイヤーデータ（gzip圧縮JSON）を一時保存します。保存データは5分で失効します。

このエンドポイントの一時保存先はDBではなく、APIプロセス内のインメモリ領域です。したがって、APIプロセスの再起動後や複数インスタンス構成では、発行済み `uploadToken` が引き継がれない場合があります。
また、有効期限切れデータの判定とメモリ回収は、この一時保存機能へのアクセス時にまとめて行う遅延クリーンアップ方式です。TTL経過直後に即座にメモリから削除されるわけではありませんが、次回アクセス時には期限切れとして扱われます。

- **認証**: 不要
- **レート制限**: 30 req/IP/min
- **CORS**:
  - `https://new.chunithm-net.com` からの `POST` / `OPTIONS` を許可します。
  - それ以外の許可オリジンは通常どおり `cors.allow_origins` に従います。
- **ヘッダー**:
  - `Content-Encoding: gzip`
  - `Content-Type: application/json`
- **制限**:
  - gzip後サイズ: 500KB以下
  - 解凍後JSONサイズ: 500KB以下
  - 同時保持件数: 1IPあたり最大3件
- **検証内容**:
  - この時点では `Content-Encoding: gzip`、`Content-Type: application/json`、gzip展開の可否、およびサイズ制限のみを検証します。
  - 展開後の本文は生のバイト列のまま保持し、`PlayerDataPayload` へのデコードや妥当性検証は行いません。
  - そのため、JSON構文が壊れている本文や、`PlayerDataPayload` として解釈できない本文でも一時保存される場合があります。
  - 厳密な検証および実際の登録処理は `/internal/player-data/commit` 実行時に初めて行われます。
  - 認証状態は判定に使いません。認証済みブラウザから呼び出した場合でも、未認証と同じ扱いで受け付けます。

#### レスポンス（201 Created）

```json
{
  "uploadToken": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "expiresAt": "2026-04-08T12:34:56Z"
}
```

#### 主なエラー

- `400 Bad Request`: gzip不正 / `Content-Encoding` 不正 / `Content-Type` 不正
- `413 Payload Too Large`: サイズ上限超過
- `409 Conflict`: 1IPあたり保持件数上限超過
- `429 Too Many Requests`: レート超過
- `503 Service Unavailable`: 一時データ総量上限超過

### POST `/internal/player-data/commit`

一時保存済みデータを、認証済みユーザーに紐づけて確定保存します。

このエンドポイントでは、保存済み本文を `PlayerDataPayload` として解釈し、通常の `/internal/me/register-data` と同じ登録処理を実行します。ただし、一時データは登録処理の開始前に `uploadToken` 単位で消費されます。したがって、登録処理中にエラーになった場合でも同じ `uploadToken` では再試行できず、再アップロードが必要です。

- **認証**: 必須（Firebase Bearer）
- **リクエスト**:

```json
{
  "uploadToken": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
```

#### レスポンス（200 OK）

`/internal/me/register-data` と同じ `PlayerDataResult` を返します。保存前状態との差分がある場合は `changes` も含まれます。

#### 主なエラー

- `401 Unauthorized`: 未認証
- `400 Bad Request`: 保存済み本文がJSONとして解釈できない、または対応していない `app_ver`
- `404 Not Found`: token期限切れ / 未存在
- `422 Unprocessable Entity`: `uploadToken` の形式不正、またはスコア整合性など `PlayerDataPayload` のバリデーション不正
- `500 Internal Server Error`: DB保存失敗、またはプレイヤー名・日時形式など一部の入力不正を含む想定外エラー（tokenは消費済みのため再アップロードが必要）

#### バリデーションの補足

- `uploadToken` は UUID v4 を要求します。
- 一時保存時点では厳密な妥当性検証を行わないため、`commit` 時に初めて不正データとして弾かれることがあります。
- 一時保存時点では JSON デコードすら行わないため、構文破損や型不一致も `commit` 時まで遅延します。
- すべての入力不正が `422` になるわけではありません。実装上、`app_ver` は `400`、一部のプレイヤー名・日時形式の不正は `500` として扱われます。

---

## 目標（Goal）API

目標はユーザー個人のデータであり、認証済みユーザーの個人データ操作が集約されている `/internal/me` 配下に配置されます。他ユーザーへの公開は現時点では行いません。

- 1ユーザーあたり目標上限は **100件** です。
- 目標は「属性（`attributes`）」と「成果（`achievement`）」を持ちます。
- 外部API（`/v1`）には公開しません。

### Goal オブジェクト

```json
{
  "id": 1,
  "title": "マスター14+ 100枚",
  "achievement_type": "score_count",
  "achievement_params": { "score": 1007500, "count": 100 },
  "attributes": { "diff": 4, "const": { "min": 14.0, "max": 14.9 } },
  "invert": false,
  "created_at": "2026-01-01T09:00:00+09:00"
}
```

| フィールド | 型 | 方向 | 説明 |
|---|---|---|---|
| `id` | `integer` | レスポンスのみ | 目標ID（自動採番） |
| `title` | `string` | 双方向 | 目標タイトル。trim後30文字以内、空文字不可、制御文字不可 |
| `achievement_type` | `string` | 双方向 | 成果種別コード（`achievement_types.code` と完全一致。大文字小文字の混在不可） |
| `achievement_params` | `object` | 双方向 | 成果種別ごとの可変パラメータ（詳細は後述） |
| `attributes` | `object` | 双方向 | 対象譜面の絞り込み条件（詳細は後述）。空オブジェクト `{}` は全譜面対象 |
| `invert` | `boolean` | 双方向 | UI表示反転フラグ。サーバー側の達成判定には影響しない |
| `created_at` | `string` | レスポンスのみ | 作成日時（RFC3339、タイムゾーンオフセット付き） |

**作成・更新リクエストでの省略可否**:

- `title` / `achievement_type` / `achievement_params` は必須です。
- `attributes` は省略可能です。省略時は絞り込み条件なしとして扱います。明示する場合は空オブジェクト `{}` を推奨します。
- `invert` は省略可能です。省略時は `false` として扱います。
- `id` / `created_at` はレスポンス専用です。作成・更新リクエストには含めません。

### `achievement_type` 一覧

| code | 意味 |
|---|---|
| `rank_count` | 指定ランク（スコア）以上の譜面数 |
| `score_count` | 指定スコア以上の譜面数 |
| `avg_score` | 全譜面の平均スコア |
| `hardlamp_count` | 指定ハードランプの達成数 |
| `combolamp_count` | 指定コンボランプの達成数 |
| `total_score` | 全譜面のスコア合計 |
| `overpower_value` | 全譜面のOverPower値合計 |
| `overpower_percent` | 全譜面に対するOverPower達成割合（%） |

### `achievement_params` 仕様

`achievement_params` オブジェクト自体は必須です。ただし、成果種別によってはオブジェクト内の一部パラメータを省略または `null` にできます。省略可能なパラメータは以下の通りです。

| `achievement_type` | 省略可能なパラメータ | 省略/null時の扱い |
|---|---|---|
| `rank_count` / `score_count` | `count` | 対象譜面数（動的上限） |
| `hardlamp_count` / `combolamp_count` | `count` | 対象譜面数（動的上限） |
| `total_score` | `total` | 対象譜面数 × 1,010,000（動的上限） |
| `overpower_value` | `total` | 対象譜面の理論値OP合計（動的上限） |

上記以外のパラメータは必須です。例えば `score_count` の `score`、`avg_score` の `score`、`overpower_percent` の `total` は省略できません。

#### `rank_count` / `score_count`

`rank_count` と `score_count` は同じ構造・同じ判定ロジックです。`rank_count` はUIが「ランク由来の目標」として判別するために分けています。ランク境界はフロントエンドが保持し、バックエンドはスコア閾値のみを扱います。

```json
{ "score": 1000000, "count": 100 }
```

| パラメータ | 型 | 範囲 | 説明 |
|---|---|---|---|
| `score` | `integer` | 0〜1,010,000 | スコア閾値 |
| `count` | `integer \| null` | null または 1〜対象譜面数 | 目標件数。省略/null時は「対象譜面数（動的上限）」として扱います |

#### `avg_score`

```json
{ "score": 1000000 }
```

| パラメータ | 型 | 範囲 | 説明 |
|---|---|---|---|
| `score` | `integer` | 0〜1,010,000 | 平均スコア目標値。平均算出時の端数は小数点以下切り捨て |

#### `hardlamp_count`

```json
{ "lamp": "BRV", "count": 100 }
```

| パラメータ | 型 | 範囲 | 説明 |
|---|---|---|---|
| `lamp` | `string` | 下表の略称（完全一致） | ハードランプ種別 |
| `count` | `integer \| null` | null または 1〜対象譜面数 | 目標件数。省略/null時は「対象譜面数（動的上限）」として扱います |

**ハードランプ略称**:

| 略称 | マスタ名（`clear_lamp_types.name`） |
|---|---|
| `HRD` | `HARD` |
| `BRV` | `BRAVE` |
| `ABS` | `ABSOLUTE` |
| `CTS` | `CATASTROPHY` |

序列: `HRD < BRV < ABS < CTS`

#### `combolamp_count`

```json
{ "lamp": "AJ", "count": 100 }
```

| パラメータ | 型 | 範囲 | 説明 |
|---|---|---|---|
| `lamp` | `string` | 下表の略称（完全一致） | コンボランプ種別 |
| `count` | `integer \| null` | null または 1〜対象譜面数 | 目標件数。省略/null時は「対象譜面数（動的上限）」として扱います |

**コンボランプ略称**:

| 略称 | マスタ名（`combo_lamp_types.name`） |
|---|---|
| `FC` | `FULL COMBO` |
| `AJ` | `ALL JUSTICE` |

#### `total_score`

```json
{ "total": 100000000 }
```

| パラメータ | 型 | 範囲 | 説明 |
|---|---|---|---|
| `total` | `integer \| null` | null または 0〜対象譜面数 × 1,010,000 | スコア合計目標値。省略/null時は「対象譜面数 × 1,010,000（動的上限）」として扱います |

#### `overpower_value`

```json
{ "total": 1000000.000 }
```

| パラメータ | 型 | 範囲 | 説明 |
|---|---|---|---|
| `total` | `number \| null` | null または 0〜対象譜面の理論値OP合計（小数点以下3桁まで） | OverPower合計目標値。省略/null時は「対象譜面の理論値OP合計（動的上限）」として扱います |

理論値OP合計はリクエスト時にマスタデータから算出されます。

#### `overpower_percent`

```json
{ "total": 76.500 }
```

| パラメータ | 型 | 範囲 | 説明 |
|---|---|---|---|
| `total` | `number` | 0〜100（小数点以下3桁まで） | OverPower達成割合の目標値（%） |

### `attributes` 仕様

対象譜面の絞り込み条件です。省略したフィールドは条件なし（全譜面対象）とみなします。空オブジェクト `{}` は全譜面が対象です。

**許可キーは `diff` / `const` / `genre` / `ver` のみ**です。未知キーは `goal_invalid_attributes` エラーになります。

```json
{
  "diff": [3, 4],
  "const": { "min": 14.0, "max": 14.4 },
  "genre": [1, 2],
  "ver": [20, 21]
}
```

| フィールド | 型 | 必須 | 説明 |
|---|---|---|---|
| `diff` | `integer \| integer[]` | 任意 | 難易度ID（`difficulties.id` と同値、1〜5）。単一値または配列で指定可能。省略時は全難易度対象 |
| `const` | `object` | 任意 | 譜面定数レンジ。`min`/`max` を `float64`（小数1桁）で指定。`min <= max` 必須。範囲: `1.0 ≤ min, max ≤ 16.0`。省略時は定数条件なし |
| `genre` | `integer \| integer[]` | 任意 | ジャンルマスタID。単一値または配列で指定可能。省略時は全ジャンル対象 |
| `ver` | `integer \| integer[]` | 任意 | バージョンマスタID。単一値または配列で指定可能。省略時は全バージョン対象 |

**難易度IDの対応**:

| 値 | 難易度 |
|---|---|
| 1 | `BASIC` |
| 2 | `ADVANCED` |
| 3 | `EXPERT` |
| 4 | `MASTER` |
| 5 | `ULTIMA` |

**マスタ整合**:
- `genre` / `ver` は起動時プリロード済みのマスタIDのみ許可。存在しないIDは `goal_invalid_attributes` エラー。
- `genre` / `ver` のIDは存在確認（一致判定）のみに使用し、IDの数値による順序比較・レンジ判定は行いません。
- `diff` は 1〜5 の範囲のみ許可。範囲外は `goal_invalid_attributes` エラー。

**配列入力の正規化**:
- `diff` / `genre` / `ver` は単一値（例: `"diff": 4`）と配列（例: `"diff": [3, 4]`）の両方を受け付けます。
- 配列は重複除去 + 昇順ソートで正規化されます。
- 要素数1の配列は単一値に正規化されます（例: `"diff": [4]` → `"diff": 4`）。
- 配列の実質上限は、対応するマスタデータの全件数です。
- レスポンスの `attributes` は正規化後の形式で返却されます（要素1ならスカラー、複数なら配列）。

### バリデーション方針

#### 境界（Handler/DTO）での検査

- リクエストボディは厳格デコード（`BindStrictJSON`）されるため、`title` / `achievement_type` / `achievement_params` / `attributes` / `invert` 以外の未知キーを含むと `bad_request` になります。

#### Usecase層での業務ルール検査

1. **`title`**: trim後に空文字・30ルーン超・制御文字を含む場合はエラー
2. **`achievement_type`**: マスタキャッシュで検証。完全一致のみ許可（例: `score_count` は可、`Score_Count` は不可）
3. **`attributes`**: 許可キーのみ。各値をマスタ検証。`diff` / `genre` / `ver` は `integer | integer[]` を受け付け、配列は重複除去+昇順ソートで正規化（要素1はスカラー化）。`const` は小数1桁に丸め、`min <= max`、有効範囲 `[1.0, 16.0]`
4. **`achievement_params`**: `achievement_type` に対応する構造体へデコードし、パラメータ値を検証
5. **動的上限チェック**: `attributes` で絞り込まれた対象譜面数をもとに以下を検証
   - `rank_count` / `score_count` / `hardlamp_count` / `combolamp_count` の `count` ≤ 対象譜面数
   - `total_score.total` ≤ 対象譜面数 × 1,010,000
   - `overpower_value.total` ≤ 対象譜面の理論値OverPower合計
   - `overpower_percent.total` は 0〜100 の固定上限

#### 100件上限の担保

作成トランザクション内で `SELECT id FROM users WHERE id = ? FOR UPDATE` によりユーザー行をロックした後、`SELECT COUNT(*)` で件数を確認します。これにより同一ユーザーの並列リクエストがシリアライズされ、レースコンディションを防止します。

### GET `/internal/me/goals`

自分が作成した目標を全件返します。ソート順は `created_at` 昇順（作成順）です。

**レスポンス**: 200 OK

```json
{
  "goals": [
    {
      "id": 1,
      "title": "マスター14+ 100枚",
      "achievement_type": "score_count",
      "achievement_params": { "score": 1007500, "count": 100 },
      "attributes": { "diff": 4, "const": { "min": 14.0, "max": 14.9 } },
      "invert": false,
      "created_at": "2026-01-01T09:00:00+09:00"
    }
  ]
}
```

### POST `/internal/me/goals`

目標を新規作成します。100件上限を超える場合は `goal_limit_exceeded` エラーを返します。

**リクエストボディ**: Goal オブジェクト（`id` / `created_at` 除く）

```json
{
  "title": "マスター14+ 100枚",
  "achievement_type": "score_count",
  "achievement_params": { "score": 1007500, "count": 100 },
  "attributes": { "diff": 4, "const": { "min": 14.0, "max": 14.9 } },
  "invert": false
}
```

**レスポンス**: 201 Created（作成された Goal オブジェクト）

### PUT `/internal/me/goals/:id`

指定IDの目標を完全上書き更新します。他ユーザーの目標を指定した場合は `goal_not_found` を返します。

**リクエストボディ**: Goal オブジェクト（`id` / `created_at` 除く）

**レスポンス**: 200 OK（更新後の Goal オブジェクト）

### DELETE `/internal/me/goals/:id`

指定IDの目標を削除します。他ユーザーの目標を指定した場合は `goal_not_found` を返します。

**レスポンス**: 204 No Content

### Goal API エラーコード

| エラーコード | HTTP | 説明 |
|---|---|---|
| `goal_not_found` | 404 | 指定した goal が存在しない（他ユーザーの goal も含む） |
| `goal_limit_exceeded` | 400 | 100件上限を超えて作成しようとした |
| `goal_invalid_title` | 400 | `title` が trim 後に空文字、30文字超、または制御文字を含む |
| `goal_invalid_achievement_type` | 400 | `achievement_type` が不正（マスタに存在しない・大文字小文字不一致） |
| `goal_invalid_achievement_params` | 400 | `achievement_params` の形式不正・範囲不正・動的上限超過・`achievement_type` との組み合わせ不一致 |
| `goal_invalid_attributes` | 400 | `attributes` の形式不正・マスタ不整合・未許可キー・`const` 範囲外・`diff` 範囲外 |
| `invalid_goal_input` | 400 | goal 入力全般の不正（JSONデコード失敗など） |

---

## `/internal/me/record-filters` グループ

認証済みユーザーが保存したレコードフィルタをサーバーに保存します。通常レコードと WORLD'S END は `filter_type` で区別します。

サーバーは `filter` の内部フィールドを解釈しません。`filter` が JSON オブジェクトであること、`schema_version` が正の整数であること、圧縮前の保存ペイロードが 8KB 以下であることのみ検証し、gzip 圧縮して保存します。

### RecordFilter オブジェクト

```json
{
  "id": "11111111-1111-1111-1111-111111111111",
  "name": "高難度FC狙い",
  "filter_type": "standard",
  "schema_version": 3,
  "filter": {
    "title": "",
    "difficulties": ["MASTER", "ULTIMA"]
  },
  "created_at": "2026-06-15T12:00:00Z",
  "updated_at": "2026-06-15T12:00:00Z"
}
```

| フィールド | 型 | 説明 |
|---|---|---|
| `id` | string | サーバー生成 UUID |
| `name` | string | 保存名。trim 後 1〜30文字、制御文字不可 |
| `filter_type` | `"standard"` \| `"worldsend"` | 通常レコードまたは WORLD'S END の区別 |
| `schema_version` | number | フロント側フィルタスキーマのバージョン。正の整数 |
| `filter` | object | フロント側のフィルタ状態 JSON。サーバーでは中身を解釈しない |
| `created_at` | string | 作成日時（ISO 8601） |
| `updated_at` | string | 更新日時（ISO 8601） |

### GET `/internal/me/record-filters`

保存済みレコードフィルタ一覧を返します。`filter_type` クエリで種別を絞り込めます。省略時は全件を返します。ソート順は `updated_at` 降順です。

**クエリパラメータ**

| パラメータ | 型 | 必須 | 説明 |
|---|---|---|---|
| `filter_type` | `"standard"` \| `"worldsend"` | いいえ | 取得対象のフィルタ種別 |

**レスポンス**: 200 OK

```json
{
  "filters": [
    {
      "id": "11111111-1111-1111-1111-111111111111",
      "name": "高難度FC狙い",
      "filter_type": "standard",
      "schema_version": 3,
      "filter": {
        "title": "",
        "difficulties": ["MASTER", "ULTIMA"]
      },
      "created_at": "2026-06-15T12:00:00Z",
      "updated_at": "2026-06-15T12:00:00Z"
    }
  ]
}
```

### POST `/internal/me/record-filters`

レコードフィルタを新規保存します。1ユーザーあたり最大100件です。

**リクエストボディ**

```json
{
  "name": "高難度FC狙い",
  "filter_type": "standard",
  "schema_version": 3,
  "filter": {
    "title": "",
    "difficulties": ["MASTER", "ULTIMA"]
  }
}
```

**レスポンス**: 201 Created（作成された RecordFilter オブジェクト）

### PUT `/internal/me/record-filters/:id`

指定IDの保存済みレコードフィルタを完全上書き更新します。他ユーザーのフィルタを指定した場合は `record_filter_not_found` を返します。

**リクエストボディ**: POST と同じ

**レスポンス**: 200 OK（更新後の RecordFilter オブジェクト）

### DELETE `/internal/me/record-filters/:id`

指定IDの保存済みレコードフィルタを削除します。他ユーザーのフィルタを指定した場合は `record_filter_not_found` を返します。

**レスポンス**: 204 No Content

### RecordFilter API エラーコード

| エラーコード | HTTP | 説明 |
|---|---|---|
| `record_filter_not_found` | 404 | 指定したフィルタが存在しない（他ユーザーのフィルタも含む） |
| `record_filter_limit_exceeded` | 400 | 100件上限を超えて作成しようとした |
| `invalid_record_filter_input` | 400 | `name` / `filter_type` / `schema_version` / `filter` / サイズ制限のいずれかが不正 |
| `invalid_record_filter_id` | 400 | `:id` が UUID 形式ではない |

---

## `/internal/users` グループ

### GET `/internal/users/`
- **認証**: Firebase Bearer 必須（ADMIN権限必須）
- **説明**: ADMIN専用のエンドポイントです。プライベートアカウント、プレイヤー未紐付けアカウントを含む全ユーザーの一覧を取得します。
- **クエリパラメータ**:
    - `page` (任意): ページ番号 (デフォルト: 1)
    - `name` (任意): ユーザー名またはプレイヤー名の前方一致検索
- **レスポンス**: `AdminUserListResponse` の配列を返します。

#### レスポンス例

```json
[
  {
    "username": "user1",
    "account_type": "ADMIN",
    "created_at": "2025-11-27T12:00:00+09:00",
    "updated_at": "2025-11-28T22:23:32+09:00",
    "player_name": "player1",
    "rating": 17.25,
    "overpower_value": 9500.00,
    "is_suspicious": false,
    "is_private": false,
    "firebase_uid": "firebase-uid-1"
  },
  {
    "username": "user2",
    "account_type": "PLAYER",
    "created_at": "2025-11-20T09:30:00+09:00",
    "updated_at": "2025-11-21T08:15:00+09:00",
    "player_name": null,
    "rating": null,
    "overpower_value": null,
    "is_suspicious": true,
    "is_private": true,
    "firebase_uid": null
  }
]
```

#### AdminUserListResponse スキーマ

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `username` | string | ユーザー名 |
| `account_type` | string | アカウント種別（`PLAYER` / `EDITOR` / `ADMIN`） |
| `created_at` | string | ユーザー作成日時 (ISO8601) |
| `updated_at` | string | ユーザー更新日時 (ISO8601) |
| `player_name` | string \| null | プレイヤー名（未連携の場合は `null`） |
| `rating` | number \| null | レーティング（未連携の場合は null） |
| `overpower_value` | number \| null | オーバーパワー値（未連携の場合は null） |
| `is_suspicious` | boolean | 不審アカウントフラグ |
| `is_private` | boolean | プライベートアカウントかどうか |
| `firebase_uid` | string \| null | 連携済み Firebase UID（未連携の場合は `null`） |

---

### GET `/internal/users/:username`
- **認証**: Firebase Bearer (任意)
- **レートリミット**: 認証なしは1分10回/IP
- **パスパラメータ**: `username` - 対象ユーザーのユーザー名
- **クエリパラメータ**:
    - `view` (任意): `rating` を指定すると、`records` は `updated_at`/`best`/`best_candidate`/`new`/`new_candidate` のみを返します（`standard`/`worldsend` は返しません）。`record` を指定すると、`records` は `updated_at`/`standard`/`worldsend` のみを返します。
    - `include_noplay` (任意): `true` を指定すると、`records.standard` と `records.worldsend` に未プレイ譜面を補完して返します。未プレイ補完データは `is_played=false` となり、`updated_at` / `clear_lamp` は `null` になります。`view=rating` と併用した場合は `include_noplay` は無視されます。`view=record` と併用した場合も補完されます。
- **レスポンス**: ユーザープロファイルとプレイヤーレコードを一括で返します。非公開設定のユーザーは本人以外 404 を返します。プレイヤー未連携の場合は `200 OK` で `player` と `records` が `null` になります。
  - `player.overpower_value` は保存済みの楽曲OP合計です。
  - `player.overpower_percent` はレスポンス時点の通常楽曲マスタとプレイヤーの未解禁設定から随時計算されます。曲追加、削除状態変更、譜面定数変更により、プレイヤーデータ再登録なしで割合のみ変動する場合があります。

#### レスポンス例

```json
{
  "username": "sample_user",
  "player": {
    "name": "プレイヤー名",
    "level": 217,
    "rating": 17.29,
    "class_emblem_id": 6,
    "class_emblem_base_id": 4,
    "last_played_at": "2025-11-02T16:42:00+09:00",
    "overpower_value": 96123.91,
    "overpower_percent": 76.27,
    "honors": [
      { "slot": 1, "name": "称号名（上段）", "type_name": "gold", "image_url": "https://..." },
      { "slot": 2, "name": "称号名（中段）", "type_name": "platina", "image_url": "https://..." },
      { "slot": 3, "name": "称号名（下段）", "type_name": "rainbow", "image_url": "" }
    ],
    "created_at": "2025-11-27T12:00:00+09:00",
    "updated_at": "2025-11-27T12:00:00+09:00"
  },
  "records": {
    "updated_at": "2025-11-28T22:23:32+09:00",
    "best": [...],
    "best_candidate": [...],
    "new": [...],
    "new_candidate": [...],
    "standard": [
      {
        "is_played": true,
        "updated_at": "2025-11-28T22:23:32+09:00",
        "difficulty": "MASTER",
        "id": "d3b6f3dd66b06bf4",
        "title": "New York Back Raise",
        "artist": "saaa + kei_iwata + stuv + わかどり",
        "const": 14.3,
        "is_const_unknown": false,
        "score": 1009975,
        "rating": 16.45,
        "overpower": 86.21,
        "overpower_percent": 99.6647,
        "img": "9f060e856cb7ad10",
        "clear_lamp": "ABSOLUTE",
        "combo_lamp": "ALL JUSTICE",
        "full_chain": null,
        "slot": null
      }
    ]
  },
  "updated_at": "2025-11-28T22:23:32+09:00"
}
```

#### UserProfileWithRecordsDTO スキーマ

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `username` | string | ユーザー名 |
| `player` | PlayerDTO \| null | プレイヤー情報。未連携の場合は `null` |
| `records` | UserRecordResponseDTO \| null | スロット別レコード。未連携の場合は `null` |
| `updated_at` | string \| null | プレイヤーデータの最終更新日時 (ISO8601)。未連携の場合は `null` |

#### プレイヤー未連携時のレスポンス例

```json
{
  "username": "sample_user",
  "player": null,
  "records": null,
  "updated_at": null
}
```

---

### GET `/internal/users/:username/profile`
- **認証**: Firebase Bearer (任意)
- **レートリミット**: 認証なしで1分間10回/IP
- **パスパラメータ**: `username` - 対象ユーザーのユーザー名
- **レスポンス**: ユーザー名とプレイヤー情報のみを返します。非公開設定のユーザーは本人以外 404 を返します。プレイヤー未連携の場合は `200 OK` で `player` が `null` になります。

#### レスポンス例

```json
{
  "username": "sample_user",
  "player": {
    "name": "プレイヤー名",
    "level": 50,
    "rating": 16.5,
    "class_emblem_id": 3,
    "class_emblem_base_id": 1,
    "last_played_at": "2024-12-01T15:30:00Z",
    "overpower_value": 1234.56,
    "overpower_percent": 98.76,
    "honors": [
      {
        "slot": 1,
        "name": "称号名",
        "type_name": "gold",
        "image_url": "https://example.com/honor.png"
      }
    ],
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-12-20T10:00:00Z"
  }
}
```

#### UserProfileDTO スキーマ

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `username` | string | ユーザー名 |
| `player` | object \| null | プレイヤー情報。スキーマは `PlayerDTO` と同一。未連携の場合は `null` |

#### プレイヤー未連携時のレスポンス例

```json
{
  "username": "sample_user",
  "player": null
}
```

### GET `/internal/users/:username/updated-at`
- **認証**: Firebase Bearer (任意)
- **レートリミット**: 認証なしで1分間10回/IP
- **パスパラメータ**: `username` - 対象ユーザーのユーザー名
- **レスポンス**: `profile.updated_at` と `rating/record` 系の元になるレコード最終更新日時のうち、新しい方のみを返します。非公開設定のユーザーは本人以外 404 を返します。プレイヤー未連携の場合は `200 OK` で `updated_at` が `null` になります。

#### レスポンス例

```json
{
  "updated_at": "2026-04-18T12:34:56Z"
}
```

#### プレイヤー未連携時のレスポンス例

```json
{
  "updated_at": null
}
```

#### UserUpdatedAtDTO スキーマ

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `updated_at` | string \| null | `players.updated_at` と `player_records` / `player_worldsend_records` の `updated_at` の最大値 (ISO8601)。プレイヤー未連携の場合は `null` |

### GET `/internal/users/:username/rating`
- **認証**: Firebase Bearer (任意)
- **レートリミット**: 認証なしで1分間10回/IP
- **パスパラメータ**: `username` - 対象ユーザーのユーザー名
- **レスポンス**: レーティング枠のみを返します。非公開設定のユーザーは本人以外 404 を返します。プレイヤー未連携の場合は各配列が空、`meta.updated_at` が `null` になります。

#### レスポンス例

```json
{
  "best": [
    {
      "updated_at": "2024-12-20T10:00:00Z",
      "difficulty": "MASTER",
      "id": "0000000000000001",
      "title": "楽曲名",
      "artist": "アーティスト名",
      "const": 14.5,
      "is_const_unknown": false,
      "score": 1009500,
      "justice_count": null,
      "rating": 17.14,
      "overpower": 5.67,
      "overpower_percent": 98.2857,
      "img": "https://example.com/jacket.png",
      "clear_lamp": "CLEAR",
      "combo_lamp": "FULL COMBO",
      "full_chain": null,
      "slot": "best"
    }
  ],
  "best_candidate": [],
  "new": [],
  "new_candidate": [],
  "meta": {
    "updated_at": "2024-12-20T10:00:00Z"
  }
}
```

#### UserRatingDTO スキーマ

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `best` | PlayerRecordDTO[] | ベスト枠レコード |
| `best_candidate` | PlayerRecordDTO[] | ベスト候補枠レコード |
| `new` | PlayerRecordDTO[] | 新曲枠レコード |
| `new_candidate` | PlayerRecordDTO[] | 新曲候補枠レコード |
| `meta` | UserRatingMetaDTO | メタ情報 |

#### UserRatingMetaDTO スキーマ

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `updated_at` | string \| null | レーティング枠レコードの最終更新日時 (ISO8601)。対象レコードが存在しない場合は `player.updated_at`、プレイヤー未連携の場合は `null` |

#### プレイヤー未連携時のレスポンス例

```json
{
  "best": [],
  "best_candidate": [],
  "new": [],
  "new_candidate": [],
  "meta": {
    "updated_at": null
  }
}
```

### GET `/internal/users/:username/record`
- **認証**: Firebase Bearer (任意)
- **レートリミット**: 認証なしで1分間10回/IP
- **概要**: 指定されたユーザーのレコード枠のみを取得します。非公開設定のユーザーは本人以外 404 を返します。プレイヤー未連携の場合は `200 OK` で `standard` / `worldsend` が空配列、`meta.updated_at` が `null` になります。
- **パスパラメータ**:

| パラメータ | 型 | 説明 |
| ---------- | -- | ---- |
| `username` | string | ユーザー名 |

- **クエリパラメータ**:
    - `include_noplay` (任意): `true` を指定すると、`standard` と `worldsend` に未プレイ譜面を補完して返します。未プレイ補完データは `is_played=false` となり、`updated_at` / `clear_lamp` は `null` になります。

- **レスポンス**: `UserRecordDTO`

```json
{
  "standard": [
    {
      "updated_at": "2024-12-20T10:00:00Z",
      "difficulty": "MASTER",
      "id": "0000000000000001",
      "title": "楽曲名",
      "artist": "アーティスト名",
      "const": 14.5,
      "is_const_unknown": false,
      "score": 1009500,
      "rating": 17.14,
      "overpower": 5.67,
      "overpower_percent": 98.2857,
      "img": "https://example.com/jacket.png",
      "clear_lamp": "CLEAR",
      "combo_lamp": "FULL COMBO",
      "full_chain": null,
      "slot": "best"
    }
  ],
  "worldsend": [
    {
      "updated_at": "2024-12-20T10:00:00Z",
      "id": "0000000000000002",
      "title": "楽曲名",
      "artist": "アーティスト名",
      "level_star": 5,
      "attribute": "狂",
      "notes": 2000,
      "score": 1000000,
      "justice_count": null,
      "img": "https://example.com/jacket.png",
      "clear_lamp": "CLEAR",
      "combo_lamp": null,
      "full_chain": null
    }
  ],
  "meta": {
    "updated_at": "2024-12-20T10:00:00Z"
  }
}
```

#### UserRecordDTO スキーマ

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `standard` | PlayerRecordDTO[] | 通常譜面の全レコード |
| `worldsend` | WorldsendRecordDTO[] | WORLD'S END の全レコード |
| `meta` | UserRecordMetaDTO | メタ情報 |

#### UserRecordMetaDTO スキーマ

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `updated_at` | string \| null | レコードの最終更新日時 (ISO8601)。通常譜面・WORLD'S END の両方にレコードが存在しない場合は `player.updated_at`、プレイヤー未連携の場合は `null` |

#### プレイヤー未連携時のレスポンス例

```json
{
  "standard": [],
  "worldsend": [],
  "meta": {
    "updated_at": null
  }
}
```

### GET `/internal/songs/updated-at`
- **認証**: Firebase Bearer (任意)
- **レートリミット**: 認証なしで1分間10回/IP
- **レスポンス**: `songs`, `charts`, `worldsend_charts` の `updated_at` の最大値のみを返します。楽曲情報キャッシュの更新判定に使用できます。

#### レスポンス例

```json
{
  "updated_at": "2026-04-09T12:34:56Z"
}
```

#### SongUpdatedAtDTO スキーマ

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `updated_at` | string \| null | `songs`, `charts`, `worldsend_charts` の `updated_at` の最大値 (ISO8601)。対象データが存在しない場合は `null` |

#### UserRecordResponseDTO スキーマ

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `updated_at` | string | `player_records` と `player_worldsend_records` の `updated_at` の最大値（ISO8601）。両方にレコードが存在しない場合は `player.updated_at` |
| `best` | PlayerRecordDTO[] | ベスト枠レコード |
| `best_candidate` | PlayerRecordDTO[] | ベスト候補枠レコード |
| `new` | PlayerRecordDTO[] | 新曲枠レコード |
| `new_candidate` | PlayerRecordDTO[] | 新曲候補枠レコード |
| `standard` | PlayerRecordDTO[] | 通常譜面の全レコード |

#### PlayerRecordDTO スキーマ

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `is_played` | boolean | プレイ済みかどうか（未プレイ補完データは `false`） |
| `updated_at` | string \\| null | 更新日時 (ISO8601)。未プレイ補完データは `null` |
| `difficulty` | string | 難易度名称 |
| `id` | string | 楽曲表示用ID |
| `title` | string | 楽曲タイトル |
| `artist` | string | アーティスト名 |
| `const` | number | 譜面定数 |
| `is_const_unknown` | boolean | 譜面定数が不明か |
| `score` | number | スコア |
| `justice_count` | number \| null | JUSTICE数。スコアが1,010,000の場合はノーツ数不明でも `0`。それ以外はALL JUSTICEかつノーツ数がある場合のみ `round(notes * (1010000 - score) / 10000)` で算出し、条件を満たさない場合は `null` |
| `rating` | number | 単曲レーティング（譜面定数とスコアから計算） |
| `overpower` | number | 単曲OVER POWER（譜面定数・スコア・コンボランプから計算） |
| `overpower_percent` | number | 譜面別理論値OVER POWERに対する単曲OVER POWER達成割合（%） |
| `img` | string | 楽曲画像ID |
| `clear_lamp` | string \\| null | クリアランプ名称。未プレイ補完データは `null` |
| `combo_lamp` | string \| null | コンボランプ名称（マスタ値が「NONE」の場合は `null`） |
| `full_chain` | string \| null | フルチェイン名称（マスタ値が「NONE」の場合は `null`） |
| `slot` | string \| null | スロット名称（マスタ値が「none」の場合は `null`） |

#### WorldsendRecordDTO スキーマ

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `is_played` | boolean | プレイ済みかどうか（未プレイ補完データは `false`） |
| `updated_at` | string \| null | 更新日時 (ISO8601)。未プレイ補完データは `null` |
| `id` | string | 楽曲表示用ID |
| `title` | string | 楽曲タイトル |
| `artist` | string | アーティスト名 |
| `level_star` | number \| null | WORLD'S END レベル |
| `attribute` | string \| null | WORLD'S END 属性 |
| `notes` | number \| null | ノーツ数 |
| `score` | number | スコア |
| `justice_count` | number \| null | JUSTICE数。スコアが1,010,000の場合はノーツ数不明でも `0`。それ以外はALL JUSTICEかつノーツ数がある場合のみ `round(notes * (1010000 - score) / 10000)` で算出し、条件を満たさない場合は `null` |
| `img` | string | 楽曲画像ID |
| `clear_lamp` | string \| null | クリアランプ名称。未プレイ補完データは `null` |
| `combo_lamp` | string \| null | コンボランプ名称（マスタ値が「NONE」の場合は `null`） |
| `full_chain` | string \| null | フルチェイン名称（マスタ値が「NONE」の場合は `null`） |

- **主なエラー**:
  - 401 Unauthorized (`missing_token` / `invalid_token`): 認証が必要
  - 404 Not Found (`user_not_found`): ユーザーが見つからない（非公開/プレイヤー未紐付含む）

### DELETE `/internal/users/:username`
- **認証**: Firebase Bearer 必須
- **権限**: ADMIN 権限が必要
- **パスパラメータ**: `username` - 削除対象ユーザーのユーザー名
- **レスポンス**: 204 No Content

**説明**: 指定されたユーザー名のユーザーを物理削除します。関連データ（プレイヤー・レコード・APIトークン）も外部キー制約により削除されます。Firebase UID が連携されている場合は Firebase ユーザー削除も試行します（失敗時はサーバーログに記録し、APIレスポンスは成功を維持します）。

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): Bearerトークン欠如または無効
  - 403 Forbidden (`forbidden`): ADMIN権限が不足
  - 404 Not Found (`user_not_found`): ユーザーが存在しない
  - 400 Bad Request (`operation_failed`): 操作失敗（詳細隠蔽）

---

## `/internal/songs` グループ

### GET `/internal/songs`
- **認証**: Firebase Bearer (任意)
- **レートリミット**: 認証なしは1分10回/IP
- **概要**: WORLD'S END以外の全楽曲を譜面情報付きで取得します。デフォルトでは削除済み楽曲は除外されます。
- **クエリパラメータ**:
  - `include_deleted` (bool, optional): `true` で削除済み楽曲も含めます。ただし、EDITOR 権限が必要です。権限がない場合は自動的に `false` として処理されます。デフォルト: `false`
- **レスポンス**: 200 OK

**レスポンス例**:
```json
{
  "songs": [
    {
      "id": "0123456789abcdef",
      "title": "楽曲名",
      "reading": "ガッキョクメイ",
      "artist": "アーティスト名",
      "genre": "ジャンル名",
      "bpm": 180,
      "release": "2024-01-15T00:00:00Z",
      "jacket": "img_filename",
      "official_idx": "123",
      "maxop": 82.5,
      "is_maxop_unknown": false,
      "op_target_difficulty": "MASTER",
      "charts": {
        "BASIC": {
          "const": 3.0,
          "is_const_unknown": false,
          "notes": 500,
          "notes_designer": "譜面作者A"
        },
        "MASTER": {
          "const": 13.5,
          "is_const_unknown": false,
          "notes": 1800,
          "notes_designer": "譜面作者B"
        }
      }
    }
  ]
}
```

**レスポンスフィールド（トップレベル）**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `songs` | SongDTO[] | 楽曲情報の配列 |

**SongDTO**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `id` | string | 楽曲の表示用ID（16進数16文字） |
| `title` | string | 楽曲名 |
| `reading` | string \| null | 楽曲名の読み |
| `artist` | string | アーティスト名 |
| `genre` | string | ジャンル名（IDではなく名称） |
| `bpm` | int \| null | BPM（未設定の場合null） |
| `release` | string \| null | リリース日（ISO8601形式、未設定の場合null） |
| `jacket` | string \| null | ジャケット画像ファイル名（未設定の場合null） |
| `official_idx` | string | 公式インデックス |
| `maxop` | number | その曲の全譜面のうち最も定数が高い譜面で理論値(AJC)を取ったときのOP値 |
| `is_maxop_unknown` | bool | `maxop` が暫定値である可能性があるかどうか。MASTERまたはULTIMAの譜面定数が未判明（`is_const_unknown=true`）の場合に`true` |
| `op_target_difficulty` | string \| null | `maxop` の算出対象となった譜面の難易度。譜面が存在しない場合は `null` |
| `charts` | Map<string, ChartDTO> | 譜面情報のマップ。キーはBASIC, ADVANCED, EXPERT, MASTER, ULTIMA（大文字）の順序で固定されます。譜面が存在しない難易度はnullとなります |

**ChartDTO**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `const` | float | 譜面定数（小数点以下1桁表記） |
| `is_const_unknown` | bool | 譜面定数が未確定かどうか |
| `notes` | int \| null | ノーツ数（未設定の場合null/省略） |
| `notes_designer` | string \| null | 譜面製作者名（未設定の場合null/省略） |

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/internal/songs/:displayid`
- **認証**: Firebase Bearer (任意)
- **レートリミット**: 認証なしは1分10回/IP
- **パスパラメータ**: `displayid` - 楽曲の表示用ID
- **概要**: 指定されたDisplayIDの楽曲を譜面情報付きで取得します。削除済み楽曲も取得可能です。
- **レスポンス**: 200 OK

```json
{
  "id": "0123456789abcdef",
  "title": "楽曲名",
  "artist": "アーティスト名",
  "genre": "ジャンル名",
  "bpm": 180,
  "release": "2024-01-15T00:00:00Z",
  "jacket": "img_filename",
  "official_idx": "123",
  "maxop": 82.5,
  "is_maxop_unknown": false,
  "op_target_difficulty": "MASTER",
  "charts": {
    "BASIC": {
      "const": 3.0,
      "is_const_unknown": false,
      "notes": 500
    },
    "MASTER": {
      "const": 13.5,
      "is_const_unknown": false,
      "notes": 1800
    }
  }
}
```

レスポンスフィールドの詳細は GET `/internal/songs` と同様です。

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 500 Internal Server Error (`internal_error`): 楽曲が存在しない、またはサーバー内部エラー

### GET `/internal/songs/:displayid/stats/:difficulty`
- **認証**: Firebase Bearer (任意)
- **レートリミット**: 認証なしは1分10回/IP
- **パスパラメータ**: 
  - `displayid` - 楽曲の表示用ID
  - `difficulty` - 難易度名（小文字）: `basic`, `advanced`, `expert`, `master`, `ultima`, `worldsend`
- **概要**: 指定楽曲の特定難易度のレーティング帯別統計を取得します。削除済みの譜面は集計対象外です。
- **レスポンス**: 200 OK

```json
{
  "song_id": "0000000000000001",
  "stats": [
    {
      "rating_band": "ALL",
      "rank": {
        "aaal": 45,
        "s": 28,
        "sp": 15,
        "ss": 8,
        "ssp": 3,
        "sss": 1,
        "sssp": 0,
        "max": 0
      },
      "combo": {
        "none": 20,
        "fc": 52,
        "aj": 28
      },
      "clear": {
        "failed": 5,
        "clear": 60,
        "hard": 18,
        "brave": 10,
        "absolute": 5,
        "catastrophy": 2
      },
      "average_score": 1006234.8,
      "player_count": 100
    },
    {
      "rating_band": "15.0",
      "rank": {
        "aaal": 12,
        "s": 5,
        "sp": 2,
        "ss": 1,
        "ssp": 0,
        "sss": 0,
        "sssp": 0,
        "max": 0
      },
      "combo": {
        "none": 3,
        "fc": 10,
        "aj": 5
      },
      "clear": {
        "failed": 1,
        "clear": 10,
        "hard": 3,
        "brave": 1,
        "absolute": 0,
        "catastrophy": 0
      },
      "average_score": 1007500.5,
      "player_count": 18
    }
  ]
}
```

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `song_id` | string | 楽曲の識別ID（16桁） |
| `stats` | array | レーティング帯別の統計配列。**先頭要素は必ず `rating_band: "ALL"`（全プレイヤー統計）** |
| `stats[].rating_band` | string | レーティング帯ラベル。`"ALL"`（全体）または個別帯（例: "15.0", "17.6+"） |
| `stats[].rank` | object | ランク別人数統計（aaal, s, sp, ss, ssp, sss, sssp, max） |
| `stats[].combo` | object | コンボランプ別人数統計（none, fc, aj） |
| `stats[].clear` | object | クリアランプ別人数統計（failed, clear, hard, brave, absolute, catastrophy） |
| `stats[].average_score` | number\|null | レーティング帯別平均スコア（レコード数が0件の場合はnull） |
| `stats[].player_count` | number | レーティング帯別プレイヤー数 |

**難易度パラメータについて**:
- パス内では小文字で指定: `basic`, `advanced`, `expert`, `master`, `ultima`, `worldsend`

- **主なエラー**:
  - 400 Bad Request (`invalid_difficulty`): 無効な難易度パラメータ
  - 404 Not Found (`song_not_found`): 楽曲が見つからない
  - 404 Not Found (`chart_not_found`): 指定された難易度の譜面が存在しない
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### POST `/internal/songs`
- **認証**: Firebase Bearer 必須
- **権限**: ADMIN 権限が必要
- **概要**: 新規楽曲（WORLD'S ENDを除く）を追加します。`display_id` はサーバーが自動生成します。
- **リクエスト**: JSON オブジェクト

```json
{
  "official_idx": "1234567890",
  "title": "楽曲タイトル",
  "reading": "ガッキョクタイトル",
  "artist": "アーティスト名",
  "genre": "POPS & ANIME",
  "bpm": 180,
  "released_at": "2024-01-01",
  "jacket": "ce21ae87308e7599",
  "charts": [
    {
      "difficulty": "MASTER",
      "const": 14.9,
      "is_const_unknown": false,
      "notes": 1234,
      "notes_designer": "デザイナー名"
    }
  ]
}
```

| フィールド | 型 | 必須 | 説明 |
|---|---|---|---|
| `official_idx` | string | ✅ | 公式ID（最大10文字） |
| `title` | string | ✅ | 楽曲タイトル |
| `reading` | string | - | 楽曲名の読み（最大300文字、省略可） |
| `artist` | string | ✅ | アーティスト名 |
| `genre` | string | ✅ | ジャンル名（マスターデータと一致する必要あり） |
| `bpm` | int | - | BPM（省略可） |
| `released_at` | string | - | リリース日（`YYYY-MM-DD` 形式、省略可） |
| `jacket` | string | - | ジャケット画像識別子（最大20文字、拡張子なし、省略可） |
| `charts` | array | - | 譜面情報配列（省略可） |
| `charts[].difficulty` | string | ✅ | 難易度（`BASIC` / `ADVANCED` / `EXPERT` / `MASTER` / `ULTIMA`） |
| `charts[].const` | float64 | ✅ | 譜面定数（0以上） |
| `charts[].is_const_unknown` | bool | ✅ | 定数が不明な場合 `true`（`const` には暫定値を設定） |
| `charts[].notes` | int | - | ノーツ数（省略可） |
| `charts[].notes_designer` | string | - | ノーツデザイナー名（最大100文字、省略可） |

- **レスポンス**: `201 Created` — 作成された楽曲情報（EditorSong形式）

レスポンスフィールドの詳細は GET `/internal/editor/songs/:displayid` と同様です。

- **エラー**:
  - 400 Bad Request (`bad_request`): リクエスト形式が不正
  - 400 Bad Request (`validation_failed`): バリデーションエラー
  - 400 Bad Request (`invalid_difficulty`): 難易度またはジャンルが無効
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 403 Forbidden (`forbidden`): 権限不足（ADMIN権限が必要）
  - 409 Conflict (`duplicate_official_idx`): `official_idx` が既に存在する
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### PUT `/internal/songs`
- **認証**: Firebase Bearer 必須
- **権限**: EDITOR または ADMIN 権限が必要
- **概要**: 通常楽曲（WORLD'S ENDを除く）の楽曲情報と譜面情報を一括更新します。既存データの修正専用で、新規追加・削除は行いません。
- **リクエスト**: JSON配列

```json
[
  {
    "id": "0123456789abcdef",
    "title": "楽曲タイトル",
    "reading": "ガッキョクタイトル",
    "artist": "アーティスト名",
    "genre": "POPS & ANIME",
    "bpm": 180,
    "released_at": "2024-01-01",
    "jacket": "jacket_img_name",
    "charts": {
      "EXPERT": {
        "const": 14.5,
        "is_const_unknown": false,
        "notes": 1234,
        "notes_designer": "譜面作者A"
      }
    }
  }
]
```

**リクエストフィールド（UpdateSongRequest）**:

| フィールド | 型 | 必須 | 説明 |
| ---------- | -- | ---- | ---- |
| `id` | string | ✓ | 楽曲の表示用ID（16文字の16進数文字列） |
| `title` | string | ✓ | 楽曲名 |
| `reading` | string \| null | | 楽曲名の読み（300文字以下、nullの場合DBをNULLに更新） |
| `artist` | string | ✓ | アーティスト名 |
| `genre` | string \| null | | ジャンル名（マスタに存在する必要がある） |
| `bpm` | int \| null | | BPM（正の整数、nullの場合DBをNULLに更新） |
| `released_at` | string \| null | | リリース日（YYYY-MM-DD形式、nullの場合DBをNULLに更新） |
| `jacket` | string \| null | | ジャケット画像ファイル名（nullの場合DBをNULLに更新） |
| `charts` | Map<string, UpdateChartRequest> | | 更新する譜面情報のマップ |

**UpdateChartRequest**:

| フィールド | 型 | 必須 | 説明 |
| ---------- | -- | ---- | ---- |
| `const` | float | ✓ | 譜面定数（0以上。小数1桁表記を推奨） |
| `is_const_unknown` | bool | ✓ | 譜面定数が未確定かどうか |
| `notes` | int \| null | | ノーツ数（0以上、nullの場合DBをNULLに更新） |
| `notes_designer` | string \| null | | 譜面製作者名（100文字以下、nullの場合DBをNULLに更新） |

**注意事項**:
- リクエスト配列内で `id`（display_id）が重複している場合はエラーになります。
- WORLD'S END楽曲（`is_worldsend = 1`）の `id` を指定した場合、このエンドポイントでは更新できずエラーになります。
- マスタに存在しないジャンル名を指定するとエラーになります。
- `charts` のキーは難易度名（`BASIC`, `ADVANCED`, `EXPERT`, `MASTER`, `ULTIMA`）を指定します。
- ポインタ型フィールド（`genre`, `bpm`, `released_at`, `jacket`, `notes`, `notes_designer`）にnullを指定すると、DBの該当カラムがNULLに更新されます。

- **レスポンス**: 204 No Content（成功時）

- **主なエラー**:
  - 400 Bad Request (`bad_request`): リクエスト形式不正（JSONパースエラー）
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 403 Forbidden (`forbidden`): 権限不足（PLAYER権限ではアクセス不可）
  - 422 Unprocessable Entity (`validation_failed`): バリデーションエラー
  - 500 Internal Server Error (`internal_error`): 楽曲・譜面・マスタ不整合などのサーバー内部エラー

### DELETE `/internal/songs/:displayid`
- **認証**: Firebase Bearer 必須
- **権限**: ADMIN 権限が必要
- **パスパラメータ**: `displayid` - 楽曲の表示用ID
- **概要**: 指定されたDisplayIDの楽曲を論理削除します。物理削除ではなく、`is_deleted` フラグを `true` に設定します。
- **レスポンス**: 204 No Content（成功時）

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 403 Forbidden (`forbidden`): 権限不足（ADMIN権限が必要）
  - 404 Not Found (`song_not_found`): 楽曲が見つからない
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### POST `/internal/songs/:displayid/restore`
- **認証**: Firebase Bearer 必須
- **権限**: EDITOR または ADMIN 権限が必要
- **パスパラメータ**: `displayid` - 楽曲の表示用ID
- **概要**: 指定されたDisplayIDの削除済み楽曲を復活させます。`is_deleted` フラグを `false` に設定します。
- **レスポンス**: 204 No Content（成功時）

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 403 Forbidden (`forbidden`): 権限不足（PLAYER権限ではアクセス不可）
  - 404 Not Found (`song_not_found`): 楽曲が見つからない
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/internal/worldsend-songs`
- **認証**: Firebase Bearer (任意)
- **レートリミット**: 認証なしは1分10回/IP
- **クエリパラメータ**: 
  - `include_deleted` (bool, optional): `true` を指定すると削除済み楽曲も含めて取得。ただし、EDITOR 権限が必要です。権限がない場合は自動的に `false` として処理されます。デフォルト: `false`
- **概要**: 全 WORLD'S END 楽曲を譜面情報付きで取得します。WORLD'S END は1曲1譜面が保証されています。
- **レスポンス**: 200 OK

```json
{
  "songs": [
    {
      "id": "0123456789abcdef",
      "title": "楽曲名",
      "reading": "ガッキョクメイ",
      "artist": "アーティスト名",
      "genre": "ジャンル名",
      "bpm": 180,
      "release": "2024-01-15",
      "jacket": "img_filename",
      "official_idx": "123",
      "charts": {
        "WORLDSEND": {
          "attribute": "狂",
          "level_star": 5,
          "notes": 2000
        }
      }
    }
  ]
}
```

**WorldsendSongDTO フィールド**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `id` | string | 楽曲の表示用ID |
| `title` | string | 楽曲名 |
| `reading` | string \| null | 楽曲名の読み |
| `artist` | string | アーティスト名 |
| `genre` | string \| null | ジャンル名（IDではなく名称） |
| `bpm` | int \| null | BPM |
| `release` | string \| null | リリース日（YYYY-MM-DD形式） |
| `jacket` | string \| null | ジャケット画像ファイル名 |
| `official_idx` | string | 公式インデックス |
| `charts` | Map<string, WorldsendChartDTO> | 譜面情報のマップ。キーは "WORLDSEND" 固定（1曲1譜面） |

**WorldsendChartDTO**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `attribute` | string \| null | WORLD'S END 属性（光、蔵、改、狂、etc.） |
| `level_star` | int \| null | WORLD'S END レベル（1～5） |
| `notes` | int \| null | ノーツ数 |
| `notes_designer` | string \| null | 譜面製作者名 |

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/internal/worldsend-songs/:displayid`
- **認証**: Firebase Bearer (任意)
- **レートリミット**: 認証なしは1分10回/IP
- **パスパラメータ**: `displayid` - 楽曲の表示用ID
- **概要**: 指定された DisplayID の WORLD'S END 楽曲を譜面情報付きで取得します。削除済み楽曲も取得可能です。
- **レスポンス**: 200 OK

```json
{
  "id": "0123456789abcdef",
  "title": "楽曲名",
  "artist": "アーティスト名",
  "genre": "ジャンル名",
  "bpm": 180,
  "release": "2024-01-15",
  "jacket": "img_filename",
  "official_idx": "123",
  "charts": {
    "WORLDSEND": {
      "attribute": "狂",
      "level_star": 5,
      "notes": 2000,
      "notes_designer": "譜面作者A"
    }
  }
}
```

- **主なエラー**:
  - 404 Not Found (`song_not_found`): 楽曲が見つからない
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### POST `/internal/worldsend-songs`
- **認証**: Firebase Bearer 必須
- **権限**: ADMIN 権限が必要
- **概要**: 新規 WORLD'S END 楽曲を追加します。`display_id` はサーバーが自動生成します。
- **リクエスト**: JSON オブジェクト

```json
{
  "official_idx": "1234567890",
  "title": "楽曲タイトル",
  "reading": "ガッキョクタイトル",
  "artist": "アーティスト名",
  "genre": "POPS & ANIME",
  "bpm": 180,
  "released_at": "2024-01-01",
  "jacket": "ce21ae87308e7599",
  "chart": {
    "attribute": "red",
    "level_star": 5,
    "notes": 567,
    "notes_designer": "デザイナー名"
  }
}
```

| フィールド | 型 | 必須 | 説明 |
|---|---|---|---|
| `official_idx` | string | ✅ | 公式ID（最大10文字） |
| `title` | string | ✅ | 楽曲タイトル |
| `reading` | string | - | 楽曲名の読み（最大300文字、省略可） |
| `artist` | string | ✅ | アーティスト名 |
| `genre` | string | ✅ | ジャンル名（マスターデータと一致する必要あり） |
| `bpm` | int | - | BPM（省略可） |
| `released_at` | string | - | リリース日（`YYYY-MM-DD` 形式、省略可） |
| `jacket` | string | - | ジャケット画像識別子（最大20文字、拡張子なし、省略可） |
| `chart` | object | - | 譜面情報（省略可、省略時は空行を挿入） |
| `chart.attribute` | string | - | アトリビュート（省略可） |
| `chart.level_star` | int | - | レベル星数（1〜5、省略可） |
| `chart.notes` | int | - | ノーツ数（省略可） |
| `chart.notes_designer` | string | - | ノーツデザイナー名（最大100文字、省略可） |

- **レスポンス**: `201 Created` — 作成された WORLD'S END 楽曲情報（EditorWorldsendSong形式）

レスポンスフィールドの詳細は GET `/internal/editor/worldsend-songs/:displayid` と同様です。

- **エラー**:
  - 400 Bad Request (`bad_request`): リクエスト形式が不正
  - 400 Bad Request (`validation_failed`): バリデーションエラーまたはジャンルが無効
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 403 Forbidden (`forbidden`): 権限不足（ADMIN権限が必要）
  - 409 Conflict (`duplicate_official_idx`): `official_idx` が既に存在する
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### PUT `/internal/worldsend-songs`
- **認証**: Firebase Bearer 必須
- **権限**: EDITOR または ADMIN 権限が必要
- **概要**: WORLD'S END 楽曲および譜面情報を一括更新します。既存データの修正専用で、新規追加・削除は行いません。
- **リクエスト**: JSON配列

```json
[
  {
    "id": "0123456789abcdef",
    "title": "楽曲タイトル",
    "reading": "ガッキョクタイトル",
    "artist": "アーティスト名",
    "genre": "POPS & ANIME",
    "bpm": 180,
    "released_at": "2024-01-01",
    "jacket": "jacket_img_name",
    "charts": {
      "WORLDSEND": {
        "attribute": "狂",
        "level_star": 5,
        "notes": 2000,
        "notes_designer": "譜面作者A"
      }
    }
  }
]
```

**リクエストフィールド（UpdateWorldsendSongRequest）**:

| フィールド | 型 | 必須 | 説明 |
| ---------- | -- | ---- | ---- |
| `id` | string | ✓ | 楽曲の表示用ID（16文字の16進数文字列） |
| `title` | string | ✓ | 楽曲名 |
| `reading` | string \| null | | 楽曲名の読み（300文字以下、nullの場合DBをNULLに更新） |
| `artist` | string | ✓ | アーティスト名 |
| `genre` | string \| null | | ジャンル名（マスタに存在する必要がある） |
| `bpm` | int \| null | | BPM（正の整数、nullの場合DBをNULLに更新） |
| `released_at` | string \| null | | リリース日（YYYY-MM-DD形式、nullの場合DBをNULLに更新） |
| `jacket` | string \| null | | ジャケット画像ファイル名（nullの場合DBをNULLに更新） |
| `charts` | Map<string, UpdateWorldsendChartRequest> | | 更新する譜面情報のマップ。キーは `WORLDSEND` のみ指定可能 |

**UpdateWorldsendChartRequest**:

| フィールド | 型 | 必須 | 説明 |
| ---------- | -- | ---- | ---- |
| `attribute` | string \| null | | WORLD'S END 属性（光、蔵、改、狂、etc.） |
| `level_star` | int \| null | | WORLD'S END レベル（1〜5、nullの場合DBをNULLに更新） |
| `notes` | int \| null | | ノーツ数（0以上、nullの場合DBをNULLに更新） |
| `notes_designer` | string \| null | | 譜面製作者名（100文字以下、nullの場合DBをNULLに更新） |

**注意事項**:
- `charts` を省略または `null` にした場合、譜面情報は更新されません（楽曲情報のみ更新されます）
- `charts` を指定する場合は `WORLDSEND` キーのみ指定可能です（大文字固定）
- `charts` で `WORLDSEND` 以外のキーを指定するとエラーになります
- リクエスト配列内で `id`（display_id）が重複している場合はエラーになります
- マスタに存在しないジャンル名を指定するとエラーになります
- ポインタ型フィールド（`genre`, `bpm`, `released_at`, `jacket`, `attribute`, `level_star`, `notes`, `notes_designer`）にnullを指定すると、DBの該当カラムがNULLに更新されます

- **レスポンス**: 204 No Content（成功時）

- **主なエラー**:
  - 400 Bad Request (`bad_request`): リクエスト形式不正（JSONパースエラー）
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 403 Forbidden (`forbidden`): 権限不足（PLAYER権限ではアクセス不可）
  - 404 Not Found (`song_not_found`): 楽曲が見つからない
  - 422 Unprocessable Entity (`validation_failed`): バリデーションエラー
  - 500 Internal Server Error (`internal_error`): 楽曲・譜面・マスタ不整合などのサーバー内部エラー

### DELETE `/internal/worldsend-songs/:displayid`
- **認証**: Firebase Bearer 必須
- **権限**: ADMIN 権限が必要
- **パスパラメータ**: `displayid` - 楽曲の表示用ID
- **概要**: 指定された DisplayID の WORLD'S END 楽曲を論理削除します。物理削除ではなく、`is_deleted` フラグを `true` に設定します。
- **レスポンス**: 204 No Content（成功時）

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 403 Forbidden (`forbidden`): 権限不足（ADMIN権限が必要）
  - 404 Not Found (`song_not_found`): 楽曲が見つからない
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### POST `/internal/worldsend-songs/:displayid/restore`
- **認証**: Firebase Bearer 必須
- **権限**: EDITOR または ADMIN 権限が必要
- **パスパラメータ**: `displayid` - 楽曲の表示用ID
- **概要**: 指定された DisplayID の削除済み WORLD'S END 楽曲を復活させます。`is_deleted` フラグを `false` に設定します。
- **レスポンス**: 204 No Content（成功時）

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 403 Forbidden (`forbidden`): 権限不足（PLAYER権限ではアクセス不可）
  - 404 Not Found (`song_not_found`): 楽曲が見つからない
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

---

## `/internal/honors` グループ

### GET `/internal/honors`
- **認証**: Firebase Bearer 必須
- **権限**: ADMIN 権限が必要
- **概要**: 称号マスタをID昇順で全件取得します。
- **レスポンス**: 200 OK

```json
{
  "honors": [
    {
      "id": 1,
      "name": "称号名",
      "type_name": "gold",
      "image_url": "https://example.com/honor.png",
      "created_at": "2025-11-27T12:00:00+09:00"
    }
  ]
}
```

### GET `/internal/honors/:id`
- **認証**: Firebase Bearer 必須
- **権限**: ADMIN 権限が必要
- **パスパラメータ**: `id` - 称号ID
- **概要**: 指定IDの称号を取得します。
- **レスポンス**: 200 OK (`HonorDTO`)

### POST `/internal/honors`
- **認証**: Firebase Bearer 必須
- **権限**: ADMIN 権限が必要
- **概要**: 称号を新規追加します。`type_name` は `GET /internal/master/honor-types` の `name` を指定します。
- **リクエストボディ**:

```json
{
  "name": "称号名",
  "type_name": "gold",
  "image_url": "https://example.com/honor.png"
}
```

- **レスポンス**: 201 Created (`HonorDTO`)

### PUT `/internal/honors/:id`
- **認証**: Firebase Bearer 必須
- **権限**: ADMIN 権限が必要
- **パスパラメータ**: `id` - 称号ID
- **概要**: 指定IDの称号を更新します。`type_name` は `GET /internal/master/honor-types` の `name` を指定します。
- **リクエストボディ**: `POST /internal/honors` と同一
- **レスポンス**: 200 OK (`HonorDTO`)

### DELETE `/internal/honors/:id`
- **認証**: Firebase Bearer 必須
- **権限**: ADMIN 権限が必要
- **パスパラメータ**: `id` - 称号ID
- **概要**: 指定IDの称号を物理削除します。プレイヤーに割り当て済みの称号は削除できません。
- **レスポンス**: 204 No Content

**HonorDTO**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `id` | int | 称号ID |
| `name` | string | 称号名 |
| `type_name` | string | 称号タイプ名 |
| `image_url` | string | 称号画像URL。未設定時は空文字 |
| `created_at` | string \| null | 作成日時 |

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 403 Forbidden (`forbidden`): 権限不足（ADMIN権限が必要）
  - 404 Not Found (`not_found`): 称号が見つからない
  - 409 Conflict (`conflict`): 重複する称号、または割り当て済み称号の削除
  - 422 Unprocessable Entity (`validation_failed`): 入力値が不正
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

---

## `/internal/editor/songs` グループ

### GET `/internal/editor/songs`
- **認証**: Firebase Bearer 必須
- **権限**: EDITOR または ADMIN 権限が必要
- **概要**: 編集者向けに、WORLD'S END以外の全楽曲を削除済みも含めて取得します。
- **レスポンス**: 200 OK

**レスポンスフィールド（トップレベル）**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `songs` | EditorSongDTO[] | 楽曲情報の配列 |

**EditorSongDTO**:

`EditorSongDTO` は `SongDTO` を embed（埋め込み）したDTOです。レスポンスJSONでは `SongDTO` の全フィールド（`id`, `title`, `reading`, `artist`, `genre`, `bpm`, `release`, `jacket`, `official_idx`, `maxop`, `is_maxop_unknown`, `op_target_difficulty`）がトップレベルにそのまま展開されます。さらに編集者向けとして、楽曲自体の `updated_at`、論理削除状態を表す `is_deleted`、および譜面ごとの `updated_at` を含む `charts` を返します。

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `is_deleted` | bool | 論理削除済みかどうか |
| `updated_at` | string \| null | 楽曲の更新日時 (ISO8601) |
| `charts` | object | 難易度ごとの譜面情報。キーは `BASIC` / `ADVANCED` / `EXPERT` / `MASTER` / `ULTIMA` |

`charts` の各値は `EditorChartDTO \| null` です。譜面が存在しない難易度は `null` になります。

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `const` | number | 譜面定数 |
| `is_const_unknown` | bool | 譜面定数が不明かどうか |
| `notes` | int \| null | ノーツ数 |
| `notes_designer` | string \| null | ノーツデザイナー名 |
| `updated_at` | string \| null | 譜面の更新日時 (ISO8601) |

`SongDTO` の各フィールドの詳細は GET `/internal/songs` の `SongDTO` を参照してください。

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 403 Forbidden (`forbidden`): 権限不足（PLAYER権限ではアクセス不可）
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/internal/editor/songs/:displayid`
- **認証**: Firebase Bearer 必須
- **権限**: EDITOR または ADMIN 権限が必要
- **パスパラメータ**: `displayid` - 楽曲の表示用ID
- **概要**: 編集者向けに、指定されたDisplayIDの通常楽曲を取得します。削除済みも取得対象です。
- **レスポンス**: 200 OK (`EditorSongDTO`)

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 403 Forbidden (`forbidden`): 権限不足（PLAYER権限ではアクセス不可）
  - 404 Not Found (`song_not_found`): 楽曲が見つからない
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/internal/editor/worldsend-songs`
- **認証**: Firebase Bearer 必須
- **権限**: EDITOR または ADMIN 権限が必要
- **概要**: 編集者向けに、全 WORLD'S END 楽曲を削除済みも含めて取得します。
- **レスポンス**: 200 OK

**レスポンスフィールド（トップレベル）**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `songs` | EditorWorldsendSongDTO[] | WORLD'S END 楽曲情報の配列 |

**EditorWorldsendSongDTO**:

`EditorWorldsendSongDTO` は `WorldsendSongDTO` を embed（埋め込み）したDTOです。レスポンスJSONでは `WorldsendSongDTO` の全フィールド（`id`, `title`, `reading`, `artist`, `genre`, `bpm`, `release`, `jacket`, `official_idx`）がトップレベルにそのまま展開されます。さらに編集者向けとして、楽曲自体の `updated_at`、論理削除状態を表す `is_deleted`、および WORLD'S END 譜面の `updated_at` を含む `charts` を返します。

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `is_deleted` | bool | 論理削除済みかどうか |
| `updated_at` | string \| null | 楽曲の更新日時 (ISO8601) |
| `charts` | object | WORLD'S END 譜面情報。`WORLDSEND` キーのみを持ちます |

`charts.WORLDSEND` は `EditorWorldsendChartDTO` です。

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `attribute` | string \| null | WORLD'S END 属性 |
| `level_star` | int \| null | WORLD'S END レベル |
| `notes` | int \| null | ノーツ数 |
| `notes_designer` | string \| null | ノーツデザイナー名 |
| `updated_at` | string \| null | 譜面の更新日時 (ISO8601) |

`WorldsendSongDTO` の各フィールドの詳細は GET `/internal/worldsend-songs` の `WorldsendSongDTO` を参照してください。

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 403 Forbidden (`forbidden`): 権限不足（PLAYER権限ではアクセス不可）
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/internal/editor/worldsend-songs/:displayid`
- **認証**: Firebase Bearer 必須
- **権限**: EDITOR または ADMIN 権限が必要
- **パスパラメータ**: `displayid` - 楽曲の表示用ID
- **概要**: 編集者向けに、指定されたDisplayIDの WORLD'S END 楽曲を取得します。削除済みも取得対象です。
- **レスポンス**: 200 OK (`EditorWorldsendSongDTO`)

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 403 Forbidden (`forbidden`): 権限不足（PLAYER権限ではアクセス不可）
  - 404 Not Found (`song_not_found`): 楽曲が見つからない
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

---

## `/internal/master` グループ

### GET `/internal/master`

- **認証**: 不要
- **概要**: フロントエンド向けにマスタデータ（ジャンル、難易度、アカウント種別、バージョン、レーティング帯、成果種別、クラスエンブレム、クリアランプ、コンボランプ、フルチェインランプ、スロット、称号タイプ）を返却します。
- `achievement_types` は目標APIの `achievement_type` を表示・入力補助するための辞書として利用します。
- **レスポンス**: 200 OK

```json
{
  "genres": [
    { "id": 1, "name": "POPS & ANIME" },
    { "id": 2, "name": "niconico" },
    { "id": 3, "name": "東方Project" }
  ],
  "difficulties": [
    { "id": 1, "name": "BASIC" },
    { "id": 2, "name": "ADVANCED" },
    { "id": 3, "name": "EXPERT" },
    { "id": 4, "name": "MASTER" },
    { "id": 5, "name": "ULTIMA" }
  ],
  "account_types": [
    { "id": 1, "name": "PLAYER" },
    { "id": 2, "name": "EDITOR" },
    { "id": 3, "name": "ADMIN" }
  ],
  "versions": [
    { "id": 1, "name": "CHUNITHM", "released_at": "2015-07-16T00:00:00+09:00" },
    { "id": 2, "name": "CHUNITHM PLUS", "released_at": "2016-02-04T00:00:00+09:00" },
    { "id": 3, "name": "CHUNITHM AIR", "released_at": "2016-08-25T00:00:00+09:00" }
  ],
  "rating_bands": [
    { "id": 1, "label": "～14.9", "min_inclusive": null, "max_exclusive": 15.0, "sort_order": 1 },
    { "id": 2, "label": "15.0", "min_inclusive": 15.0, "max_exclusive": 15.1, "sort_order": 2 },
    { "id": 28, "label": "17.6+", "min_inclusive": 17.6, "max_exclusive": null, "sort_order": 28 }
  ],
  "achievement_types": [
    { "id": 1, "name": "rank_count" },
    { "id": 2, "name": "score_count" },
    { "id": 3, "name": "avg_score" }
  ],
  "class_emblems": [
    { "id": 1, "name": "1" },
    { "id": 2, "name": "2" },
    { "id": 3, "name": "3" },
    { "id": 4, "name": "4" },
    { "id": 5, "name": "5" },
    { "id": 6, "name": "inf" }
  ],
  "class_emblem_bases": [
    { "id": 1, "name": "1" },
    { "id": 2, "name": "2" },
    { "id": 3, "name": "3" },
    { "id": 4, "name": "4" },
    { "id": 5, "name": "5" },
    { "id": 6, "name": "inf" }
  ],
  "clear_lamps": [
    { "id": 1, "name": "FAILED" },
    { "id": 2, "name": "CLEAR" },
    { "id": 3, "name": "HARD" },
    { "id": 4, "name": "BRAVE" },
    { "id": 5, "name": "ABSOLUTE" },
    { "id": 6, "name": "CATASTROPHY" }
  ],
  "combo_lamps": [
    { "id": 1, "name": "NONE" },
    { "id": 2, "name": "FULL COMBO" },
    { "id": 3, "name": "ALL JUSTICE" }
  ],
  "full_chains": [
    { "id": 1, "name": "NONE" },
    { "id": 2, "name": "FULL CHAIN GOLD" },
    { "id": 3, "name": "FULL CHAIN PLATINUM" }
  ],
  "slots": [
    { "id": 1, "name": "none" },
    { "id": 2, "name": "best" },
    { "id": 3, "name": "best_candidate" },
    { "id": 4, "name": "new" },
    { "id": 5, "name": "new_candidate" }
  ],
  "honor_types": [
    { "id": 1, "name": "normal" },
    { "id": 2, "name": "copper" },
    { "id": 3, "name": "silver" },
    { "id": 4, "name": "gold" },
    { "id": 5, "name": "platina" },
    { "id": 6, "name": "rainbow" }
  ]
}
```

**レスポンスフィールド**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `genres` | MasterItemDTO[] | ジャンル一覧（表示順） |
| `difficulties` | MasterItemDTO[] | 難易度一覧（sort_order順） |
| `account_types` | MasterItemDTO[] | アカウント種別一覧（ID順） |
| `versions` | VersionDTO[] | バージョン一覧（リリース日昇順） |
| `rating_bands` | RatingBandDTO[] | レーティング帯マスタ一覧（sort_order順） |
| `achievement_types` | MasterItemDTO[] | 成果種別一覧（ID順）。`name` には `achievement_types.code` の値が入ります |
| `class_emblems` | MasterItemDTO[] | クラスエンブレム一覧（sort_order順）。`PlayerDTO.class_emblem_id` の解決に使用 |
| `class_emblem_bases` | MasterItemDTO[] | クラスエンブレムベース一覧（sort_order順）。`PlayerDTO.class_emblem_base_id` の解決に使用 |
| `clear_lamps` | MasterItemDTO[] | クリアランプ一覧（sort_order順）。`PlayerRecordDTO.clear_lamp` の取りうる値 |
| `combo_lamps` | MasterItemDTO[] | コンボランプ一覧（sort_order順）。`PlayerRecordDTO.combo_lamp` の取りうる値 |
| `full_chains` | MasterItemDTO[] | フルチェインランプ一覧（sort_order順）。`PlayerRecordDTO.full_chain` の取りうる値 |
| `slots` | MasterItemDTO[] | スロット一覧（ID順）。`PlayerRecordDTO.slot` の取りうる値 |
| `honor_types` | MasterItemDTO[] | 称号タイプ一覧（ID順）。`HonorDTO.type_name` の取りうる値 |

**MasterItemDTO**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `id` | int | マスタID |
| `name` | string | マスタ名称。`achievement_types` の場合は表示名ではなく成果種別コード |

**VersionDTO**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `id` | int | バージョンID |
| `name` | string | バージョン名称 |
| `released_at` | string | リリース日時（ISO8601形式） |

**RatingBandDTO**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `id` | int | レーティング帯ID |
| `label` | string | 表示ラベル（例: "15.0", "17.6+"） |
| `min_inclusive` | number\|null | 下限（未設定の場合は下限なし） |
| `max_exclusive` | number\|null | 上限（未設定の場合は上限なし） |
| `sort_order` | int | 表示順 |

- **主なエラー**:
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/internal/master/versions`

- **認証**: 不要
- **概要**: `/internal/master` の `versions` を単独で取得します。フロントエンドが内部マスタ全体に依存せず、バージョン一覧だけを段階的に分離取得するためのエンドポイントです。
- **レスポンス**: 200 OK。レスポンス形式は後述の `GET /v1/master/versions` と同一です。

- **主なエラー**:
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/internal/master/honor-types`

- **認証**: 不要
- **概要**: `/internal/master` の `honor_types` を単独で取得します。管理者向け称号CRUDの `type_name` 入力候補として利用します。
- **レスポンス**: 200 OK

```json
{
  "honor_types": [
    { "id": 1, "name": "normal" },
    { "id": 2, "name": "copper" },
    { "id": 3, "name": "silver" }
  ]
}
```

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `honor_types` | MasterItemDTO[] | 称号タイプ一覧（ID昇順） |

- **主なエラー**:
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

---

## 公開API `/v1`

公開APIはAPIトークン認証を使用します。トークンは `Authorization: Bearer <token>` ヘッダーで送信してください。

### GET `/v1/master/versions`
- **認証**: APIトークン必須
- **概要**: バージョン一覧をリリース日昇順で返します。クライアントがバージョン辞書だけを独立取得する用途を想定しており、`id` は含みません。
- **レスポンス**: 200 OK

```json
{
  "versions": [
    { "name": "CHUNITHM", "released_at": "2015-07-16T00:00:00+09:00" },
    { "name": "CHUNITHM PLUS", "released_at": "2016-02-04T00:00:00+09:00" },
    { "name": "CHUNITHM AIR", "released_at": "2016-08-25T00:00:00+09:00" }
  ]
}
```

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `versions` | VersionSummaryDTO[] | バージョン一覧（リリース日昇順） |

- **主なエラー**:
  - 401 Unauthorized (`missing_token`): APIトークン未指定
  - 401 Unauthorized (`invalid_token`): 無効なAPIトークン
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/v1/songs`
- **認証**: APIトークン必須
- **概要**: WORLD'S END以外の全楽曲を取得します（削除済み楽曲は除外）。
- **レスポンス**: 200 OK

```json
{
  "songs": [
    {
      "id": "0000000000000001",
      "title": "楽曲名",
      "reading": "ガッキョクメイ",
      "artist": "アーティスト名",
      "genre": "ジャンル名",
      "bpm": 180,
      "release": "2024-01-15",
      "jacket": "jacket_001.png",
      "official_idx": "123",
      "maxop": 86.25,
      "is_maxop_unknown": false,
      "op_target_difficulty": "MASTER",
      "charts": {
        "MASTER": {
          "const": 14.5,
          "is_const_unknown": false,
          "notes": 1500,
          "notes_designer": "譜面作者A"
        },
        "BASIC": {
          "const": 8.5,
          "is_const_unknown": false,
          "notes": 450,
          "notes_designer": "譜面作者B"
        }
      }
    }
  ]
}
```

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `songs` | array | 楽曲オブジェクトの配列 |
| `songs[].id` | string | 楽曲の識別ID（16桁） |
| `songs[].title` | string | 楽曲名 |
| `songs[].reading` | string\|null | 楽曲名の読み |
| `songs[].artist` | string | アーティスト名 |
| `songs[].genre` | string\|null | ジャンル名 |
| `songs[].bpm` | number\|null | BPM |
| `songs[].release` | string\|null | リリース日（YYYY-MM-DD形式） |
| `songs[].jacket` | string\|null | ジャケット画像ファイル名 |
| `songs[].official_idx` | string | 公式インデックス |
| `songs[].maxop` | number | その曲の全譜面のうち最も定数が高い譜面で理論値(AJC)を取ったときのOP値 |
| `songs[].is_maxop_unknown` | bool | `maxop` が暫定値である可能性があるかどうか。MASTERまたはULTIMAの譜面定数が未判明（`is_const_unknown=true`）の場合に`true` |
| `songs[].op_target_difficulty` | string\|null | `maxop` の算出対象となった譜面の難易度。譜面が存在しない場合は `null` |
| `songs[].charts` | Map<string, ChartDTO> | 譜面情報のマップ。キーはBASIC, ADVANCED, EXPERT, MASTER, ULTIMA（大文字）の順序で固定されます。譜面が存在しない難易度はnullとなります |
| `songs[].charts[key].const` | number | 譜面定数（小数点以下1桁表記） |
| `songs[].charts[key].is_const_unknown` | boolean | 定数が推定値の場合true |
| `songs[].charts[key].notes` | number\|null | ノーツ数 |
| `songs[].charts[key].notes_designer` | string\|null | 譜面製作者名 |

- **主なエラー**:
  - 401 Unauthorized (`missing_token`): APIトークン未指定
  - 401 Unauthorized (`invalid_token`): 無効なAPIトークン
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### PUT `/v1/songs`
- **認証**: APIトークン必須
- **権限**: EDITOR または ADMIN 権限が必要
- **概要**: 通常楽曲（WORLD'S ENDを除く）の楽曲情報と譜面情報を一括更新します。既存データの修正専用で、新規追加・削除は行いません。
- **リクエスト**: JSON配列。形式は PUT `/internal/songs` と同じです。

```json
[
  {
    "id": "0123456789abcdef",
    "title": "楽曲タイトル",
    "reading": "ガッキョクタイトル",
    "artist": "アーティスト名",
    "genre": "POPS & ANIME",
    "bpm": 180,
    "released_at": "2024-01-01",
    "jacket": "jacket_img_name",
    "charts": {
      "MASTER": {
        "const": 14.5,
        "is_const_unknown": false,
        "notes": 1234,
        "notes_designer": "譜面作者A"
      }
    }
  }
]
```

- **レスポンス**: 204 No Content（成功時）
- **主なエラー**:
  - 400 Bad Request (`bad_request`): リクエスト形式不正（JSONパースエラー）
  - 400 Bad Request (`validation_failed`): バリデーションエラー
  - 401 Unauthorized (`missing_token`): APIトークン未指定
  - 401 Unauthorized (`invalid_token`): 無効なAPIトークン
  - 403 Forbidden (`forbidden`): 権限不足（PLAYER権限ではアクセス不可）
  - 500 Internal Server Error (`internal_error`): 楽曲・譜面・マスタ不整合などのサーバー内部エラー

### GET `/v1/worldsend-songs`
- **認証**: APIトークン必須
- **概要**: 全 WORLD'S END 楽曲を取得します（削除済み楽曲は除外）。WORLD'S END は1曲1譜面が保証されています。
- **レスポンス**: 200 OK

```json
{
  "songs": [
    {
      "id": "0123456789abcdef",
      "title": "楽曲名",
      "reading": "ガッキョクメイ",
      "artist": "アーティスト名",
      "genre": "ジャンル名",
      "bpm": 180,
      "release": "2024-01-15",
      "jacket": "https://example.com/jacket.png",
      "official_idx": "123",
      "charts": {
        "WORLDSEND": {
          "attribute": "狂",
          "level_star": 5,
          "notes": 2000
        }
      }
    }
  ]
}
```

**WorldsendSongDTO フィールド**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `id` | string | 楽曲の表示用ID |
| `title` | string | 楽曲名 |
| `reading` | string \| null | 楽曲名の読み |
| `artist` | string | アーティスト名 |
| `genre` | string \| null | ジャンル名（IDではなく名称） |
| `bpm` | int \| null | BPM |
| `release` | string \| null | リリース日（YYYY-MM-DD形式） |
| `jacket` | string \| null | ジャケット画像URL |
| `official_idx` | string | 公式インデックス |
| `charts` | Map<string, WorldsendChartDTO> | 譜面情報のマップ。キーは "WORLDSEND" 固定（1曲1譜面） |

**WorldsendChartDTO**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `attribute` | string \| null | WORLD'S END 属性（光、蔵、改、狂、etc.） |
| `level_star` | int \| null | WORLD'S END レベル（1～5） |
| `notes` | int \| null | ノーツ数 |
| `notes_designer` | string \| null | 譜面製作者名 |

- **主なエラー**:
  - 401 Unauthorized (`missing_token`): APIトークン未指定
  - 401 Unauthorized (`invalid_token`): 無効なAPIトークン
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/v1/worldsend-songs/:displayid`
- **認証**: APIトークン必須
- **パスパラメータ**: `displayid` - 楽曲の表示用ID
- **概要**: 指定された DisplayID の WORLD'S END 楽曲を譜面情報付きで取得します。
- **レスポンス**: 200 OK

```json
{
  "id": "0123456789abcdef",
  "title": "楽曲名",
  "artist": "アーティスト名",
  "genre": "ジャンル名",
  "bpm": 180,
  "release": "2024-01-15",
  "jacket": "https://example.com/jacket.png",
  "official_idx": "123",
  "charts": {
    "WORLDSEND": {
      "attribute": "狂",
      "level_star": 5,
      "notes": 2000
    }
  }
}
```

- **主なエラー**:
  - 401 Unauthorized (`missing_token`): APIトークン未指定
  - 401 Unauthorized (`invalid_token`): 無効なAPIトークン
  - 404 Not Found (`song_not_found`): 楽曲が見つからない
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/v1/songs/:displayid`
- **認証**: APIトークン必須
- **パスパラメータ**:

| パラメータ | 型 | 説明 |
| ---------- | -- | ---- |
| `displayid` | string | 楽曲の表示用ID |

- **概要**: 指定楽曲の詳細を取得します。
- **レスポンス**: 200 OK

```json
{
  "id": "0000000000000001",
  "title": "楽曲名",
  "artist": "アーティスト名",
  "genre": "ジャンル名",
  "bpm": 180,
  "release": "2024-01-15T00:00:00Z",
  "jacket": "https://example.com/jacket.png",
  "official_idx": "123",
  "maxop": 86.25,
  "is_maxop_unknown": false,
  "charts": {
    "MASTER": {
      "const": 14.5,
      "is_const_unknown": false,
      "notes": 1500
    }
  }
}
```

- **主なエラー**:
  - 401 Unauthorized (`missing_token`): APIトークン未指定
  - 401 Unauthorized (`invalid_token`): 無効なAPIトークン
  - 404 Not Found (`song_not_found`): 楽曲が見つからない
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/v1/songs/:displayid/stats/:difficulty`
- **認証**: APIトークン必須
- **概要**: 指定楽曲の特定難易度のレーティング帯別統計を取得します。
- **パスパラメータ**:

| パラメータ | 型 | 説明 |
| ---------- | -- | ---- |
| `displayid` | string | 楽曲の表示用ID |
| `difficulty` | string | 難易度名（小文字）: `basic`, `advanced`, `expert`, `master`, `ultima`, `worldsend` |

- **レスポンス**: 200 OK

レスポンス形式は GET `/internal/songs/:displayid/stats/:difficulty` と同様です。

- **主なエラー**:
  - 400 Bad Request (`invalid_difficulty`): 無効な難易度パラメータ
  - 401 Unauthorized (`missing_token`): APIトークン未指定
  - 401 Unauthorized (`invalid_token`): 無効なAPIトークン
  - 404 Not Found (`song_not_found`): 楽曲が見つからない
  - 404 Not Found (`chart_not_found`): 指定された難易度の譜面が存在しない
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/v1/users/:username`
- **認証**: APIトークン必須
- **概要**: 指定されたユーザーのプロファイルとスコアレコードを取得します。非公開設定のユーザーは本人（APIトークンの所有者）以外 404 を返します。プレイヤー未連携の場合は `200 OK` で `player` と `records` が `null` になります。
- **パスパラメータ**:

| パラメータ | 型 | 説明 |
| ---------- | -- | ---- |
| `username` | string | ユーザー名 |

- **クエリパラメータ**:
    - `include_noplay` (任意): `true` を指定すると、`records.standard` と `records.worldsend` に未プレイ譜面を補完して返します。未プレイ補完データは `is_played=false` となり、`updated_at` / `clear_lamp` は `null` になります。

- **レスポンス**: 200 OK

```json
{
  "username": "sample_user",
  "player": {
    "name": "プレイヤー名",
    "level": 50,
    "rating": 16.50,
    "class_emblem_id": 3,
    "class_emblem_base_id": 1,
    "last_played_at": "2024-12-01T15:30:00Z",
    "overpower_value": 1234.56,
    "overpower_percent": 98.76,
    "honors": [
      {
        "slot": 1,
        "name": "称号名",
        "type_name": "gold",
        "image_url": "https://example.com/honor.png"
      }
    ],
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-12-20T10:00:00Z"
  },
  "records": {
    "updated_at": "2024-12-20T10:00:00Z",
    "best": [
      {
        "updated_at": "2024-12-20T10:00:00Z",
        "difficulty": "MASTER",
        "id": "0000000000000001",
        "title": "楽曲名",
        "artist": "アーティスト名",
        "const": 14.5,
        "is_const_unknown": false,
        "score": 1009500,
        "rating": 17.14,
        "overpower": 5.67,
        "overpower_percent": 98.2857,
        "img": "https://example.com/jacket.png",
        "clear_lamp": "CLEAR",
        "combo_lamp": "FULL COMBO",
        "full_chain": null,
        "slot": "best"
      }
    ],
    "best_candidate": [],
    "new": [],
    "new_candidate": [],
    "standard": [],
    "worldsend": []
  },
  "updated_at": "2024-12-20T10:00:00Z"
}
```

#### プレイヤー未連携時のレスポンス例

```json
{
  "username": "sample_user",
  "player": null,
  "records": null,
  "updated_at": null
}
```

- **主なエラー**:
  - 401 Unauthorized (`missing_token`): APIトークン未指定
  - 401 Unauthorized (`invalid_token`): 無効なAPIトークン
  - 404 Not Found (`user_not_found`): ユーザーが見つからない（非公開ユーザー含む）
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

---

## chunirec互換API `/compat/chunirec/2.0`

chunirec互換APIはchunirecとの互換性を持つエンドポイントです。APIトークン認証を使用し、`Authorization: Bearer <token>` ヘッダーで送信してください。

### GET `/compat/chunirec/2.0/music/showall`
- **認証**: APIトークン必須
- **概要**: WORLD'S END以外の全楽曲をchunirec互換形式で取得します（削除済み楽曲は除外）。
- **レスポンス**: 200 OK

```json
[
  {
    "meta": {
      "id": "0000000000000001",
      "title": "楽曲名",
      "genre": "POPS & ANIME",
      "artist": "アーティスト名",
      "release": "2015-07-16",
      "bpm": 180.0
    },
    "data": {
      "MAS": {
        "level": 14.5,
        "const": 14.5,
        "maxcombo": 1234,
        "is_const_unknown": false
      },
      "BAS": {
        "level": 8.0,
        "const": 8.5,
        "maxcombo": 456,
        "is_const_unknown": false
      }
    }
  }
]
```

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `meta.id` | string | 楽曲の識別ID（16桁） |
| `meta.title` | string | 楽曲名 |
| `meta.genre` | string\|null | ジャンル名 |
| `meta.artist` | string | アーティスト名 |
| `meta.release` | string\|null | リリース日（YYYY-MM-DD形式） |
| `meta.bpm` | number\|null | BPM |
| `data.BAS` | object\|null | BASIC譜面データ |
| `data.ADV` | object\|null | ADVANCED譜面データ |
| `data.EXP` | object\|null | EXPERT譜面データ |
| `data.MAS` | object\|null | MASTER譜面データ |
| `data.ULT` | object\|null | ULTIMA譜面データ |
| `data.*.level` | number | 表記レベル（.0または.5） |
| `data.*.const` | number | 譜面定数 |
| `data.*.maxcombo` | number\|null | ノーツ数 |
| `data.*.is_const_unknown` | boolean | 定数が推定値の場合true |

- **主なエラー**:
  - 401 Unauthorized (`missing_token`): APIトークン未指定
  - 401 Unauthorized (`invalid_token`): 無効なAPIトークン
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/compat/chunirec/2.0/music/show`
- **認証**: APIトークン必須
- **概要**: 指定された1楽曲のchunirec互換形式の情報を取得します（WORLD'S END除く）。
- **クエリパラメータ**:
  - `id` (string, required): 楽曲のDisplay ID（16桁）
- **レスポンス**: 200 OK

```json
{
  "meta": {
    "id": "0000000000000001",
    "title": "楽曲名",
    "genre": "POPS & ANIME",
    "artist": "アーティスト名",
    "release": "2015-07-16",
    "bpm": 180.0
  },
  "data": {
    "MAS": {
      "level": 14.5,
      "const": 14.5,
      "maxcombo": 1234,
      "is_const_unknown": false
    },
    "BAS": {
      "level": 8.0,
      "const": 8.5,
      "maxcombo": 456,
      "is_const_unknown": false
    }
  }
}
```

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `meta.id` | string | 楽曲の識別ID（16桁） |
| `meta.title` | string | 楽曲名 |
| `meta.genre` | string\|null | ジャンル名 |
| `meta.artist` | string | アーティスト名 |
| `meta.release` | string\|null | リリース日（YYYY-MM-DD形式） |
| `meta.bpm` | number\|null | BPM |
| `data.BAS` | object\|null | BASIC譜面データ |
| `data.ADV` | object\|null | ADVANCED譜面データ |
| `data.EXP` | object\|null | EXPERT譜面データ |
| `data.MAS` | object\|null | MASTER譜面データ |
| `data.ULT` | object\|null | ULTIMA譜面データ |
| `data.*.level` | number | 表記レベル（.0または.5） |
| `data.*.const` | number | 譜面定数 |
| `data.*.maxcombo` | number\|null | ノーツ数 |
| `data.*.is_const_unknown` | boolean | 定数が推定値の場合true |

- **主なエラー**:
  - 400 Bad Request (`validation_failed`): クエリパラメータ`id`が未指定
  - 401 Unauthorized (`missing_token`): APIトークン未指定
  - 401 Unauthorized (`invalid_token`): 無効なAPIトークン
  - 404 Not Found: 指定されたDisplay IDの楽曲が見つからない
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/compat/chunirec/2.0/users/show`
- **認証**: APIトークン必須
- **概要**: 指定されたユーザーのプロフィールをchunirec互換形式で取得します。
- **クエリパラメータ**:
  - `user_name` (string, optional): 取得対象のユーザー名。未指定の場合はAPIトークン所有者のプロフィールを返します。
- **レスポンス**: 200 OK

```json
{
  "user_id": 283,
  "player_name": "Ｕ＋ＦＦ３１",
  "title": "邪気眼",
  "title_rarity": "platinum",
  "level": 229,
  "rating": "17.23",
  "rating_max": "17.23",
  "classemblem": "inf",
  "classemblem_base": null,
  "is_joined_team": null,
  "updated_at": "2026-01-24T18:39:52+09:00"
}
```

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `user_id` | number | 内部ユーザーID |
| `player_name` | string | プレイヤー名 |
| `title` | string\|null | 1番目の称号（スロット1） |
| `title_rarity` | string\|null | 1番目の称号のレアリティ（normal, copper, silver, gold, platinum, rainbow等）。ChuniSupport内部では"platina"を"platinum"に変換 |
| `level` | number | プレイヤーレベル |
| `rating` | string\|null | レーティング（小数点以下2桁の文字列） |
| `rating_max` | string\|null | 最大レーティング（現在はratingと同じ値） |
| `classemblem` | string\|null | クラスエンブレム（"1", "2", "3", "4", "5", "inf"） |
| `classemblem_base` | string\|null | クラスエンブレムベース（"1", "2", "3", "4", "5", "inf"） |
| `is_joined_team` | null | チーム参加状態（ChuniSupportでは保持しないため常にnull） |
| `updated_at` | string | プレイヤーデータの最終更新日時（RFC3339形式） |

- **主なエラー**:
  - 401 Unauthorized (`missing_token`): APIトークン未指定
  - 401 Unauthorized (`invalid_token`): 無効なAPIトークン
  - 404 Not Found (`user_not_found`): ユーザーが見つからない（非公開ユーザー・プレイヤー未紐付けを含む）
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

---

フロントエンド開発の参考として、主要なDTO型をTypeScriptで定義した例を示します。

```typescript
// ユーザー関連
interface UserDTO {
  username: string;
  player: PlayerDTO | null;
}

// ユーザー一覧レスポンス（ADMIN用）
interface AdminUserListResponse {
  username: string;
  account_type: string;
  created_at: string;
  updated_at: string;
  player_name: string | null;
  rating: number | null;
  overpower_value: number | null;
  is_suspicious: boolean;
  is_private: boolean;
  firebase_uid: string | null;
  is_deleted: boolean;
}

// プロファイル＋レコード統合レスポンス
interface UserProfileWithRecordsDTO {
  username: string;
  player: PlayerDTO | null;
  records: UserRecordResponseDTO | null;
  updated_at: string | null;
}

interface UserRatingDTO {
  best: PlayerRecordDTO[];
  best_candidate: PlayerRecordDTO[];
  new: PlayerRecordDTO[];
  new_candidate: PlayerRecordDTO[];
  meta: UserRatingMetaDTO;
}

interface UserRatingMetaDTO {
  updated_at: string | null;
}

interface UserRecordDTO {
  standard: PlayerRecordDTO[];
  worldsend: WorldsendRecordDTO[];
  meta: UserRecordMetaDTO;
}

interface UserRecordMetaDTO {
  updated_at: string | null;
}

interface PlayerDTO {
  name: string;
  level: number;
  rating: number;
  class_emblem_id: number | null;
  class_emblem_base_id: number | null;
  last_played_at: string | null;
  overpower_value: number | null;
  overpower_percent: number | null;
  honors: HonorDTO[];
  created_at: string;
  updated_at: string;
}

interface HonorDTO {
  slot: number;
  name: string;
  type_name: string;
  image_url: string;
}

// レコード関連
interface PlayerRecordDTO {
  is_played: boolean;
  updated_at: string | null;
  difficulty: string;
  id: string;
  title: string;
  artist: string;
  const: number;
  is_const_unknown: boolean;
  score: number;
  justice_count: number | null;
  rating: number;
  overpower: number;
  overpower_percent: number;
  img: string;
  clear_lamp: string | null;
  combo_lamp: string | null;  // マスタ値が「NONE」の場合はnull
  full_chain: string | null;  // マスタ値が「NONE」の場合はnull
  slot: string | null;        // マスタ値が「none」の場合はnull
}

interface UserRecordResponseDTO {
  updated_at: string;
  best: PlayerRecordDTO[];
  best_candidate: PlayerRecordDTO[];
  new: PlayerRecordDTO[];
  new_candidate: PlayerRecordDTO[];
  standard: PlayerRecordDTO[];
  worldsend: WorldsendRecordDTO[];  // WORLD'S END レコード（レーティング計算対象外）
}

// WORLD'S END レコード（スロット分類なし、レーティング計算なし）
interface WorldsendRecordDTO {
  is_played: boolean;
  updated_at: string | null;
  id: string;
  title: string;
  artist: string;
  level_star: number | null;      // WORLD'S END レベル（1～5）
  attribute: string | null;       // WORLD'S END 属性（光、蔵、改、狂、etc.）
  notes: number | null;
  score: number;
  justice_count: number | null;
  img: string;
  clear_lamp: string | null;
  combo_lamp: string | null;      // マスタ値が「NONE」の場合はnull
  full_chain: string | null;      // マスタ値が「NONE」の場合はnull
}

// エラーレスポンス
interface ErrorResponse {
  error: {
    status: number;
    code: string;  // エラーコード (例: "invalid_token", "validation_failed")
    message?: string; // validation_failed の場合のみ返却されることがある
    details?: {
      field: string;
      message: string;
    }[];
  }
}

// プレイヤーデータ登録結果
interface PlayerDataResult {
  player_id: number;
  app_ver: string;
  imported_at: string;
  profile: PlayerDataProfile;
  summary: PlayerDataSummary;
  statistics: PlayerDataStatistics;
  counts: PlayerDataCounts;
  changes: PlayerDataRecordChange[];
  skipped_records: SkippedRecord[];
}

interface PlayerDataProfile {
  player_id: number;
  name: string;
  level: number;
  rating: number | null;
  class_emblem_id: number | null;
  class_emblem_base_id: number | null;
  last_played_at: string | null;
  overpower_value: number | null;
  overpower_percent: number | null;
}

interface PlayerDataSummary {
  name: string;
  level: number;
  rating: number | null;
  last_played_at: string | null;
  overpower_value: number | null;
  overpower_percentage: number | null;
}

interface PlayerDataStatistics {
  total_high_score: number;
  lamp_counts: {
    clear: Record<string, number>;
    combo: Record<string, number>;
    full_chain: Record<string, number>;
  };
}

interface PlayerDataCounts {
  standard_records_upserted: number;
  worldsend_records_upserted: number;
  standard_records_skipped: number;
  worldsend_records_skipped: number;
  honors_skipped: number;
  standard_records_actually_changed: number;
  worldsend_records_actually_changed: number;
}

interface PlayerDataRecordChange {
  record_type: 'standard' | 'worldsend';
  change_type: 'new' | 'updated';
  idx: string;
  diff: string;
  before: PlayerDataRecordState | null;
  after: PlayerDataRecordState;
}

interface PlayerDataRecordState {
  score: number;
  clear_lamp: string | null;
  combo_lamp: string | null;
  full_chain: string | null;
}

interface SkippedRecord {
  record_type: 'standard' | 'worldsend' | 'honor';
  reason: string;
  details: string;
}
```

---

## 運用上の注意

- エラーコードと内部理由コードの最新一覧は `docs/error_code_reason_codes.md` を参照してください。
- CORSの許可オリジンは環境ごとに設定ファイルで管理します。
- ユーザーを物理削除すると、ログインはできなくなり、関連データも削除されます。
