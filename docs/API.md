# chunisupport-api API仕様書

このドキュメントは `chunisupport-api` が提供する内部API(`/internal` プレフィックス)と関連ユースケースの挙動をまとめたものです。

**最終更新日**: 2026年01月17日

## ベースURLと環境

アプリケーションは `.config/<environment>.settings.json` の `app_port` で待ち受けポートを決定します。ローカル開発で `app_port` に `3002` を設定した場合のベースURLは `http://localhost:3002` になります。

想定している環境名は以下の3種類です。

| 環境名 | 設定ファイル | 例示ポート |
| ------ | ------------ | ---------- |
| develop | `.config/develop.settings.json` | `http://localhost:<app_port>` |
| staging | `.config/staging.settings.json` | `https://staging.example.com:<app_port>` |
| production | `.config/production.settings.json` | `https://api.example.com:<app_port>` |

`APP_ENV=<name> go run main.go` で設定ファイルを切り替えます（環境変数は必須です）。

以降のリクエスト例では `${APP_PORT}` を設定済みポート番号のプレースホルダーとして使用します。

主要なパス構成:

- 監視用API: `http://localhost:${APP_PORT}/`
- 内部向けAPI: `http://localhost:${APP_PORT}/internal`
- 公開API (APIトークン認証): `http://localhost:${APP_PORT}/v1`

## CORS

すべてのエンドポイントでCORSが有効化されています。設定は環境ごとの設定ファイルで管理されます。

| 設定項目 | 説明 |
| -------- | ---- |
| `cors.allow_origins` | 許可するオリジンの配列 |
| `cors.allow_credentials` | Cookie送信を許可するか (内部APIでは `true` 必須) |
| `cors.max_age` | プリフライトキャッシュ秒数 |

**許可メソッド**: `GET`, `POST`, `PUT`, `DELETE`, `OPTIONS`

**許可ヘッダー**: `Origin`, `Content-Type`, `Accept`, `Authorization`

**公開ヘッダー**: `Content-Length`

フロントエンドからのリクエストでは、Cookie認証を使用する場合 `credentials: 'include'` を必ず設定してください。

```javascript
// fetch の例
fetch('http://localhost:3002/internal/me', {
  credentials: 'include'
});
```

## 認証

### 内部API (`/internal`)

- ログイン成功時に `token` という名前の HTTPOnly Cookie を発行します。
- Cookie属性は以下の通りです。
  - `Path=/`
  - `HttpOnly` は常に付与
  - `Secure` は `auth.cookie_secure` 設定値に追従 (開発環境では `false`, HTTPS運用時は `true` を推奨)
  - `SameSite` は `auth.cookie_same_site` の値 (`lax`/`strict`/`none`)
- セッションはサーバー側 (`sessions` テーブル) に保存され、JWTにはセッションIDが含まれます。
- すべての認証必須エンドポイントでは `JWTMiddleware` が Cookie を検証し、`userEntity`（`entity.User`）をリクエストコンテキストに格納します。
- `/internal/users/:username` `/internal/songs` `/internal/songs/:displayid` `/internal/songs/worldsend` `/internal/songs/worldsend/:displayid` は Cookie 任意です。未認証時は1分10回/IPのレートリミットが適用されます。

### 公開API (`/v1`)

- `Authorization: Bearer <token>` ヘッダーで API トークンを送信します。
- ミドルウェアがトークンを検証し、認証済みユーザーとトークン情報をコンテキストに格納します。
- トークンは `/internal/auth/api-tokens` で発行します。1ユーザーにつき1件のみ保持され、再発行すると旧トークンは無効化されます。
- **レートリミット**: ADMINアカウントは無制限、その他のアカウントは15分間に150リクエストまでに制限されます。

```javascript
// fetch の例
fetch('http://localhost:3002/v1/songs', {
  headers: {
    'Authorization': 'Bearer your-api-token-here'
  }
});
```

## 共通レスポンス仕様

- コンテンツタイプは `application/json`。
- カスタムエラーハンドラーは以下形式を返します。

```json
{
  "error": {
    "status": 401,
    "code": "invalid_token"
  }
}
```

`error` オブジェクト内の `code` フィールドには機械処理しやすいスネークケースのエラーコードが入ります。`status` フィールドにはHTTPステータスコードが入ります。詳細なエラーメッセージはサーバーログにのみ記録され、クライアントには返却されません。

### エラーコード一覧

| カテゴリ | コード | HTTPステータス | 発生条件 |
| -------- | ------ | -------------- | -------- |
| 汎用 | `bad_request` | 400 | リクエスト形式不正、JSON構文エラー |
| 汎用 | `internal_error` | 500 | サーバー内部エラー |
| 認証 | `unauthorized` | 401 | 認証が必要 |
| 認証 | `invalid_credentials` | 401 | ユーザー名またはパスワードが不正 |
| 認証 | `invalid_token` | 401 | トークンが無効 |
| 認証 | `token_expired` | 401 | トークン期限切れ |
| 認証 | `missing_token` | 401 | トークン未指定 |
| 認証 | `invalid_session` | 401 | セッション無効/期限切れ（詳細隠蔽） |
| 認証 | `invalid_recovery_credentials` | 401 | リカバリーコードが無効/使用済み/ユーザー不在 |
| 権限 | `forbidden` | 403 | アクセス権限なし |
| ユーザー | `registration_failed` | 400 | ユーザー登録失敗（詳細隠蔽） |
| ユーザー | `user_not_found` | 404 | ユーザーが見つからない（非公開含む） |
| ユーザー | `operation_failed` | 400 | 操作失敗（詳細隠蔽） |
| プレイヤー | `player_not_found` | 404 | プレイヤーが見つからない |
| データ | `validation_failed` | 422 | 入力値バリデーションエラー |
| データ | `resource_not_found` | 400 | マスターデータが見つからない |
| データ | `conflict` | 409 | データ競合（例: 別ユーザーのプレイヤーデータと衝突） |
| データ | `api_token_not_found` | 404 | APIトークンが見つからない |
| データ | `payload_too_large` | 413 | リクエストボディサイズ超過 |
| 入力検証 | `username_empty` | 400 | ユーザー名が空 |
| 入力検証 | `username_too_short` | 400 | ユーザー名が短すぎる（5文字未満） |
| 入力検証 | `username_too_long` | 400 | ユーザー名が長すぎる（50文字超過） |
| 入力検証 | `username_invalid_char` | 400 | ユーザー名に使用できない文字が含まれている（小文字英数字のみ可） |
| 入力検証 | `password_too_short` | 400 | パスワードが短すぎる（8文字未満） |
| 入力検証 | `password_too_long` | 400 | パスワードが長すぎる（128文字超過） |
| 入力検証 | `invalid_password` | 400 | パスワードが無効（詳細隠蔽） |
| その他 | `not_found` | 404 | リソースが存在しない |
| その他 | `method_not_allowed` | 405 | HTTPメソッドが許可されていない |
| その他 | `unsupported_media_type` | 415 | サポートされていないメディアタイプ |
| その他 | `too_many_requests` | 429 | リクエスト制限超過 |
| その他 | `service_unavailable` | 503 | サービス利用不可 |

## マスターデータ概要

主なマスタ定義は `migration/mysql/000001_init_schema.up.sql` に記載されています。

- アカウント種別: `PLAYER`, `EDITOR`, `ADMIN`
- クリアランプ: `FAILED`, `CLEAR`, `HARD`, `BRAVE`, `ABSOLUTE`, `CATASTROPHY`
- コンボランプ: `NONE`, `FULL COMBO`, `ALL JUSTICE`
- フルチェイン: `NONE`, `FULL CHAIN GOLD`, `FULL CHAIN PLATINUM`
- スロット: `none`, `best`, `best_candidate`, `new`, `new_candidate`

## エンドポイント一覧

| パス | メソッド | 認証 | 概要 |
| ---- | -------- | ---- | ---- |
| `/` | GET | 不要 | 監視向けにアプリケーション名を固定で返します。 |
| `/health` | GET | APIトークン(ADMIN) | DB接続を含むヘルスチェック。 |
| `/internal/auth/register` | POST | 不要 | ユーザー登録。 |
| `/internal/auth/login` | POST | 不要 | ログインしてCookieを発行。 |
| `/internal/auth/logout` | POST | Cookie | セッション失効。 |
| `/internal/auth/recovery-codes` | POST | 不要 | リカバリーコードでパスワード再設定。 |
| `/internal/auth/api-tokens` | POST | Cookie | APIトークン発行。 |
| `/internal/auth/api-tokens` | DELETE | Cookie | APIトークン削除。 |
| `/internal/me` | GET | Cookie | 自身のユーザー情報。 |
| `/internal/me/privacy` | PUT | Cookie | 非公開設定更新。 |
| `/internal/me/password` | PUT | Cookie | パスワード変更。 |
| `/internal/me/recovery-codes` | POST | Cookie | リカバリーコード発行。 |
| `/internal/me` | DELETE | Cookie | アカウント論理削除。 |
| `/internal/me/register-data` | POST | Cookie | CHUNITHMプレイヤーデータ登録。 |
| `/internal/me/player-data` | DELETE | Cookie | プレイヤー連携を解除し、プレイヤー関連レコードを削除。 |
| `/internal/me/sessions` | GET | Cookie | 有効なセッション数を取得。 |
| `/internal/me/sessions` | DELETE | Cookie | 現在のセッション以外をすべてログアウト。 |
| `/internal/users/` | GET | Cookie (ADMIN+) | 全ユーザー一覧取得（プライベート・削除済み・プレイヤー未紐付けを含む）。 |
| `/internal/users/:username` | GET | Cookie (任意) | プロファイルとレコードを一括取得。 |
| `/internal/users/:username` | DELETE | Cookie (ADMIN+) | ユーザーの論理削除。 |
| `/internal/users/:username/restore` | POST | Cookie (ADMIN+) | ユーザーの復活。 |
| `/internal/songs` | GET | Cookie (任意) | WORLD'S END以外の楽曲一覧取得（ページネーション対応）。 |
| `/internal/songs/:displayid` | GET | Cookie (任意) | 楽曲詳細取得。 |
| `/internal/songs/:displayid` | DELETE | Cookie (EDITOR+) | 楽曲の論理削除。 |
| `/internal/songs/:displayid/restore` | POST | Cookie (EDITOR+) | 楽曲の復活。 |
| `/v1/songs` | GET | APIトークン | 全楽曲一覧取得（WORLD'S END除く）。 |
| `/v1/songs/:songId` | GET | APIトークン | 楽曲詳細取得。 |
| `/v1/users/:username` | GET | APIトークン | ユーザープロファイルとレコード取得。 |

---

## 監視用エンドポイント

> **警告**: これらのエンドポイントはアプリケーションの稼働状況を確認するために使用されます。本番環境では、不正な情報漏洩を防ぐため、ネットワーク設定（例: ファイアウォール、ロードバランサ）によってアクセスを内部ネットワークや特定のIPアドレスに制限することが強く推奨されます。

### GET `/`
- **認証**: 不要
- **レスポンス**: 常に 200 OK で固定のアプリケーション名を返します（将来的に変更の可能性あり）。

```json
{
  "app_name": "chunisupport-api"
}
```

### GET `/health`
- **認証**: APIトークン (ADMIN)
- **レスポンス**:
  - 200 OK: 空レスポンス
  - 503 Service Unavailable: DB接続エラーを通知。

---

## 認証エンドポイント

### POST `/internal/auth/register`
- **認証**: 不要
- **リクエストボディ**:

```json
{
  "username": "sample_user",
  "password": "strongpassword"
}
```

| フィールド | 型 | 必須 | バリデーション |
| ---------- | -- | ---- | -------------- |
| `username` | string | ✓ | 5〜50文字、小文字英数字のみ |
| `password` | string | ✓ | 8〜128文字 |

- **レスポンス**: 201 Created。`UserDTO` を返します。登録成功時は自動的にログイン状態となり、`token` Cookie が設定されます。
- **レスポンスヘッダー**: `Set-Cookie: token=<JWT>; Path=/; HttpOnly; ...`

```json
{
  "username": "sample_user",
  "player": null
}
```

- **主なエラー**:
  - 400 Bad Request (`bad_request`): リクエスト形式不正（JSONパースエラー）
  - 400 Bad Request (`username_empty`): ユーザー名が空
  - 400 Bad Request (`username_too_short`): ユーザー名が5文字未満
  - 400 Bad Request (`username_too_long`): ユーザー名が50文字超過
  - 400 Bad Request (`username_invalid_char`): ユーザー名に使用できない文字が含まれている（小文字英数字のみ可）
  - 400 Bad Request (`password_too_short`): パスワードが8文字未満
  - 400 Bad Request (`password_too_long`): パスワードが128文字超過
  - 400 Bad Request (`registration_failed`): ユーザー登録失敗（詳細隠蔽）
  - 500 Internal Server Error (`internal_error`): 予期しないサーバーエラー

### POST `/internal/auth/login`
- **認証**: 不要
- **リクエストボディ**:

```json
{
  "username": "sample_user",
  "password": "strongpassword"
}
```

| フィールド | 型 | 必須 | バリデーション |
| ---------- | -- | ---- | -------------- |
| `username` | string | ✓ | 5〜50文字、小文字英数字のみ |
| `password` | string | ✓ | 8〜128文字 |

- **レスポンス**: 200 OK。ボディは空で、`token` Cookie が設定されます。
- **レスポンスヘッダー**: `Set-Cookie: token=<JWT>; Path=/; HttpOnly; ...`
- **主なエラー**:
  - 400 Bad Request (`bad_request`): リクエスト形式不正（JSONパースエラー）
  - 401 Unauthorized (`invalid_credentials`): ユーザー名またはパスワードが不正

### POST `/internal/auth/logout`
- **認証**: Cookie 必須
- **レスポンス**: 200 OK。ボディは空です。
- Cookieは即時失効 (`Max-Age=-1`)。
- **主なエラー**:
  - 401 Unauthorized (`missing_token` / `invalid_token`): 認証が必要

### POST `/internal/auth/recovery-codes`
- **認証**: 不要
- **レートリミット**: 1分あたり5回/IP
- **リクエストボディ**:

```json
{
  "recovery_code": "A1B2-C3D4-E5F6",
  "new_password": "new-password"
}
```

| フィールド | 型 | 必須 | バリデーション |
| ---------- | -- | ---- | -------------- |
| `recovery_code` | string | ✓ | `XXXX-XXXX-XXXX` 形式の英数字 |
| `new_password` | string | ✓ | 8〜128文字 |

- **レスポンス**: 200 OK。ボディは空です。
- **主なエラー**:
  - 400 Bad Request (`bad_request`): `recovery_code` の形式不正
  - 400 Bad Request (`password_too_short`): パスワードが8文字未満
  - 400 Bad Request (`password_too_long`): パスワードが128文字超過
  - 400 Bad Request (`invalid_password`): パスワードが無効（詳細隠蔽）
  - 401 Unauthorized (`invalid_recovery_credentials`): コード不正/使用済み/ユーザー不在（詳細隠蔽）
  - 500 Internal Server Error (`internal_error`): 予期しないサーバーエラー

### POST `/internal/auth/api-tokens`
- **認証**: Cookie 必須
- **レスポンス**: 200 OK

```json
{"token":"plain-text-api-token"}
```

トークンはレスポンスでのみ平文が取得できます。

### DELETE `/internal/auth/api-tokens`
- **認証**: Cookie 必須
- **レスポンス**: 204 No Content
- 自分のAPIトークンを削除します。トークンが存在しない場合でも204を返します。
- **主なエラー**:
  - 401 Unauthorized (`missing_token` / `invalid_token`): 認証が必要

---

## `/internal/me` グループ

### GET `/internal/me`
- **認証**: Cookie 必須
- **レスポンス**: `UserDTO`

```json
{
  "username": "sample_user",
  "account_type": "PLAYER",
  "last_score_update": "2025-11-27T12:00:00+09:00"
}
```

**UserDTO スキーマ**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `username` | string | ユーザー名 |
| `account_type` | string | アカウントタイプ (PLAYER, EDITOR, ADMIN) |
| `last_score_update` | string \| null | プレイヤースコアの最終更新日時 (ISO8601)。プレイヤーが紐付いていない場合やレコードが存在しない場合は null |

### PUT `/internal/me/privacy`
- **認証**: Cookie 必須
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

### PUT `/internal/me/password`
- **認証**: Cookie 必須
- **リクエストボディ**:

```json
{
  "current_password": "oldpassword123",
  "new_password": "newpassword123"
}
```

| フィールド | 型 | 必須 | バリデーション |
| ---------- | -- | ---- | -------------- |
| `current_password` | string | ✓ | 8〜128文字 |
| `new_password` | string | ✓ | 8〜128文字 |

- **レスポンス**: 200 OK。ボディは空です。
- **主なエラー**:
  - 400 Bad Request (`bad_request`): リクエスト形式不正（JSONパースエラー）
  - 400 Bad Request (`password_too_short`): 新しいパスワードが8文字未満
  - 400 Bad Request (`password_too_long`): 新しいパスワードが128文字超過
  - 400 Bad Request (`invalid_password`): パスワードが無効（詳細隠蔽）
  - 401 Unauthorized (`missing_token` / `invalid_token`): 認証が必要
  - 401 Unauthorized (`invalid_credentials`): 現在のパスワードが不正

### POST `/internal/me/recovery-codes`
- **認証**: Cookie 必須
- **リクエストボディ**: なし
- **レスポンス**: 200 OK。リカバリーコード一覧を返却します。

```json
{
  "recovery_codes": [
    "A1B2-C3D4-E5F6",
    "G7H8-I9J0-K1L2"
  ]
}
```

- **主なエラー**:
  - 401 Unauthorized (`missing_token` / `invalid_token`): 認証が必要

### DELETE `/internal/me`
- **認証**: Cookie 必須
- **レスポンス**: 200 OK。ボディは空です。

ユーザーを論理削除し、セッションも無効化します。

- **主なエラー**:
  - 401 Unauthorized (`missing_token` / `invalid_token`): 認証が必要

### DELETE `/internal/me/player-data`
- **認証**: Cookie 必須
- **レスポンス**: 204 No Content（ボディなし）

ユーザーアカウントは残したまま、`users.player_id` を `NULL` にし、紐づく `players` および `player_records`/`player_worldsend_records`/`player_honors` を物理削除します。削除はトランザクション内で実行され、連携済みでない状態でも冪等に成功します。

- **主なエラー**:
  - 401 Unauthorized (`missing_token` / `invalid_token`): 認証が必要

### GET `/internal/me/sessions`
- **認証**: Cookie 必須
- **説明**: 現在有効なセッション数を取得します。
- **レスポンス**: 200 OK

#### レスポンス例

```json
{
  "count": 3
}
```

#### レスポンススキーマ

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `count` | number | 有効なセッション数（期限切れを除く） |

- **主なエラー**:
  - 401 Unauthorized (`missing_token` / `invalid_token`): 認証が必要
  - 500 Internal Server Error (`internal_error`): DB処理失敗

### DELETE `/internal/me/sessions`
- **認証**: Cookie 必須
- **説明**: 現在のセッション以外をすべてログアウトします（他の端末からログアウト）。
- **レスポンス**: 204 No Content（ボディなし）

現在使用中のセッションは削除されないため、このリクエストを実行した端末はログイン状態のままとなります。他の端末では次回リクエスト時に401エラーが返され、再ログインが必要になります。

- **主なエラー**:
  - 401 Unauthorized (`missing_token` / `invalid_token`): 認証が必要
  - 500 Internal Server Error (`internal_error`): DB処理失敗

### POST `/internal/me/register-data`
- **認証**: Cookie 必須
- **コンテンツタイプ**: 
  - デフォルト（クエリパラメータなし）: `application/octet-stream` または `text/plain`（base64+gzip形式）
  - `?format=json`: `application/json`（デバッグ用、通常は使用しない）
- **制限**: リクエストボディ最大5MB（圧縮前のJSONデータに対して適用）。空ボディや余分なデータは 400。ファイルサイズ超過で 413。
- **リクエストボディ**: 
  - **デフォルト形式（推奨）**: JSONデータをgzip圧縮後、base64エンコードした文字列
  - **デバッグ形式（`?format=json`）**: `PlayerDataPayload` 構造に準拠した生JSON。公式アプリのエクスポートJSONをそのまま送信する想定。

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
    body: base64Data,
    credentials: 'include'
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
    "full": [
      {
        "diff": "MAS",
        "idx": "2849",
        "score": 1002345,
        "clear_lamp": "brave",
        "cmb_lv": 1,
        "fch_lv": 0,
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
        "cmb_lv": 0,
        "fch_lv": 0
      }
    ]
  },
  "updated_at": "2025-11-27T10:30:03+09:00"
}
```

#### リクエストボディスキーマ

| フィールド | 型 | 必須 | 説明 |
| ---------- | -- | ---- | ---- |
| `app_ver` | string | ✓ | インポートアプリのバージョン |
| `name` | string | ✓ | プレイヤー名（最大20文字） |
| `level` | number | ✓ | プレイヤーレベル |
| `rating` | number | ✓ | レーティング |
| `last_played` | string | ✓ | 最終プレイ日時 (`YYYY/MM/DD HH:mm` 形式) |
| `overpower.value` | number | ✓ | オーバーパワー値 |
| `overpower.percentage` | number | ✓ | オーバーパワー割合 |
| `class_emblem.medal_class` | string | ✓ | クラスエンブレム（0埋め2桁） |
| `class_emblem.base_class` | string | ✓ | クラスエンブレムベース（0埋め2桁） |
| `team.name` | string | | チーム名 |
| `team.color` | string | | チームカラー |
| `honors` | object | | 称号情報（キー: スロット番号 "1"〜"3"） |
| `scores.full` | array | ✓ | 通常譜面スコア配列 |
| `scores.worldsend` | array | ✓ | WORLD'S END スコア配列 |
| `updated_at` | string | ✓ | 更新日時 (ISO8601) |

**スコアエントリスキーマ (`scores.full` / `scores.worldsend` の各要素)**:

| フィールド | 型 | 必須 | 説明 |
| ---------- | -- | ---- | ---- |
| `diff` | string | ✓ | 難易度 (`BAS`, `ADV`, `EXP`, `MAS`, `ULT`, `WE`) |
| `idx` | string | ✓ | 楽曲の公式インデックス |
| `score` | number | ✓ | スコア (0〜1,010,000) |
| `clear_lamp` | string \| null | | クリアランプ (`clear`, `hard`, `brave`, `absolute`, `catastrophy`, `null`=FAILED) |
| `cmb_lv` | number \| null | | コンボランプ (0=NONE, 1=FULL COMBO, 2=ALL JUSTICE) |
| `fch_lv` | number \| null | | フルチェイン (0=NONE, 1=GOLD, 2=PLATINUM) |
| `slot` | string \| null | | スロット (`best`, `best_candidate`, `new`, `new_candidate`, `null`=none) |
| `order` | number \| null | | スロット内順序 |

- **レスポンス**: 200 OK。登録結果 `PlayerDataResult` を返します。

#### レスポンス例

```json
{
  "player_id": 42,
  "app_ver": "0.0.1a",
  "imported_at": "2025-11-27T10:45:00+09:00",
  "summary": {
    "name": "プレイヤー名",
    "level": 217,
    "rating": 17.29,
    "last_played_at": "2025-11-02T16:42:00+09:00",
    "overpower_value": 96123.91,
    "overpower_percentage": 76.27
  },
  "counts": {
    "full_records_upserted": 1185,
    "worldsend_records_upserted": 120,
    "full_records_skipped": 0,
    "worldsend_records_skipped": 0,
    "honors_skipped": 0
  },
  "skipped_records": [
    {
      "record_type": "full",
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
| `summary` | object | プレイヤーサマリー情報 |
| `counts` | object | 各種レコードの処理件数 |
| `skipped_records` | array | スキップされたレコード情報（存在する場合） |

> **Note**: 差分情報（変更前後の比較）は返却されません。差分を取得する場合は、登録前後でスコア一覧API（`GET /internal/users/:username`）を呼び出し、クライアント側で比較してください。

- **主なエラー**:
  - 400 Bad Request (`bad_request` / `resource_not_found`): JSON構文不備・未知フィールド・楽曲マスタ未登録など
  - 401 Unauthorized (`missing_token` / `invalid_token`): Cookie欠如
  - 409 Conflict (`conflict`): 別ユーザーのプレイヤーデータと競合
  - 413 Request Entity Too Large (`payload_too_large`): ボディサイズ5MB超過
  - 422 Unprocessable Entity (`validation_failed`): バリデーションエラー（スコア範囲外など）

---

## `/internal/users` グループ

### GET `/internal/users/`
- **認証**: Cookie 必須（ADMIN権限必須）
- **説明**: ADMIN専用のエンドポイントです。プライベートアカウント、削除済みアカウント、プレイヤー未紐付けアカウントを含む全ユーザーの一覧を取得します。
- **クエリパラメータ**:
    - `page` (任意): ページ番号 (デフォルト: 1)
    - `name` (任意): ユーザー名またはプレイヤー名の前方一致検索
- **レスポンス**: `AdminUserListResponse` の配列を返します。

#### レスポンス例

```json
[
  {
    "username": "user1",
    "player_name": "player1",
    "rating": 17.25,
    "overpower_value": 9500.00,
    "is_private": false,
    "is_deleted": false
  },
  {
    "username": "user2",
    "player_name": "",
    "rating": null,
    "overpower_value": null,
    "is_private": true,
    "is_deleted": false
  },
  {
    "username": "deleted_user",
    "player_name": "deleted_player",
    "rating": 15.00,
    "overpower_value": 7500.00,
    "is_private": false,
    "is_deleted": true
  }
]
```

#### AdminUserListResponse スキーマ

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `username` | string | ユーザー名 |
| `player_name` | string | プレイヤー名（未連携の場合は空文字） |
| `rating` | number \| null | レーティング（未連携の場合は null） |
| `overpower_value` | number \| null | オーバーパワー値（未連携の場合は null） |
| `is_private` | boolean | プライベートアカウントかどうか |
| `is_deleted` | boolean | 削除済みアカウントかどうか |

---

### GET `/internal/users/:username`
- **認証**: Cookie (任意)
- **レートリミット**: 認証なしは1分10回/IP
- **パスパラメータ**: `username` - 対象ユーザーのユーザー名
- **レスポンス**: ユーザープロファイルとプレイヤーレコードを一括で返します。非公開設定のユーザーは本人以外 404 を返します。

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
    "team_name": "チーム名",
    "team_color": "green",
    "honors": [
      { "slot": 1, "name": "称号名（上段）", "type_name": "gold", "image_url": "https://..." },
      { "slot": 2, "name": "称号名（中段）", "type_name": "platina", "image_url": "https://..." },
      { "slot": 3, "name": "称号名（下段）", "type_name": "rainbow", "image_url": null }
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
    "all": [
      {
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
| `player` | PlayerDTO | プレイヤー情報 |
| `records` | UserRecordResponseDTO | スロット別レコード |
| `updated_at` | string | プレイヤーデータの最終更新日時 (ISO8601) |

#### UserRecordResponseDTO スキーマ

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `updated_at` | string | player_records の updated_at の最大値（ISO8601）。レコードが存在しない場合は player.updated_at |
| `best` | PlayerRecordDTO[] | ベスト枠レコード |
| `best_candidate` | PlayerRecordDTO[] | ベスト候補枠レコード |
| `new` | PlayerRecordDTO[] | 新曲枠レコード |
| `new_candidate` | PlayerRecordDTO[] | 新曲候補枠レコード |
| `all` | PlayerRecordDTO[] | 全レコード |

#### PlayerRecordDTO スキーマ

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `updated_at` | string | 更新日時 (ISO8601) |
| `difficulty` | string | 難易度名称 |
| `id` | string | 楽曲表示用ID |
| `title` | string | 楽曲タイトル |
| `artist` | string | アーティスト名 |
| `const` | number | 譜面定数 |
| `is_const_unknown` | boolean | 譜面定数が不明か |
| `score` | number | スコア |
| `rating` | number | 単曲レーティング（譜面定数とスコアから計算） |
| `overpower` | number | 単曲OVER POWER（譜面定数・スコア・コンボランプから計算） |
| `img` | string | 楽曲画像ID |
| `clear_lamp` | string | クリアランプ名称 |
| `combo_lamp` | string \| null | コンボランプ名称（マスタ値が「NONE」の場合は `null`） |
| `full_chain` | string \| null | フルチェイン名称（マスタ値が「NONE」の場合は `null`） |
| `slot` | string \| null | スロット名称（マスタ値が「none」の場合は `null`） |

- **主なエラー**:
  - 401 Unauthorized (`missing_token` / `invalid_token`): 認証が必要
  - 404 Not Found (`user_not_found`): ユーザーが見つからない（非公開/プレイヤー未紐付含む）

### DELETE `/internal/users/:username`
- **認証**: Cookie 必須
- **権限**: ADMIN (account_type_id = 3) 以上
- **パスパラメータ**: `username` - 削除対象ユーザーのユーザー名
- **レスポンス**: 204 No Content

**説明**: 指定されたユーザー名のユーザーを論理削除（`is_deleted = TRUE`）します。物理削除は行わず、関連データ（プレイヤー、セッション）は保持されます。削除済みユーザーはログインできなくなります。

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): Cookie欠如または無効
  - 403 Forbidden (`forbidden`): ADMIN権限が不足
  - 404 Not Found (`user_not_found`): ユーザーが存在しない
  - 400 Bad Request (`operation_failed`): 操作失敗（詳細隠蔽）

### POST `/internal/users/:username/restore`
- **認証**: Cookie 必須
- **権限**: ADMIN (account_type_id = 3) 以上
- **パスパラメータ**: `username` - 復活対象ユーザーのユーザー名
- **レスポンス**: 204 No Content

**説明**: 論理削除されたユーザーを復活（`is_deleted = FALSE`）させます。復活後はログインが可能になります。

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): Cookie欠如または無効
  - 403 Forbidden (`forbidden`): ADMIN権限が不足
  - 404 Not Found (`user_not_found`): ユーザーが存在しない
  - 400 Bad Request (`operation_failed`): 操作失敗（詳細隠蔽）

---

## `/internal/songs` グループ

### GET `/internal/songs`
- **認証**: Cookie (任意)
- **レートリミット**: 認証なしは1分10回/IP
- **概要**: WORLD'S END以外の全楽曲を譜面情報付きで取得します。デフォルトでは削除済み楽曲は除外されます。
- **クエリパラメータ**:
  - `include_deleted` (bool, optional): `true` で削除済み楽曲も含めます。デフォルト: `false`
- **レスポンス**: 200 OK

**レスポンス例**:
```json
{
  "songs": [
    {
      "id": "0123456789abcdef",
      "title": "楽曲名",
      "artist": "アーティスト名",
      "genre": "ジャンル名",
      "bpm": 180,
      "release": "2024-01-15T00:00:00Z",
      "jacket": "img_filename",
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
| `artist` | string | アーティスト名 |
| `genre` | string | ジャンル名（IDではなく名称） |
| `bpm` | int \| null | BPM（未設定の場合null） |
| `release` | string \| null | リリース日（ISO8601形式、未設定の場合null） |
| `jacket` | string \| null | ジャケット画像ファイル名（未設定の場合null） |
| `charts` | Map<string, ChartDTO> | 譜面情報のマップ。キーはBASIC, ADVANCED, EXPERT, MASTER, ULTIMA（大文字）の順序で固定されます。譜面が存在しない難易度はnullとなります |

**ChartDTO**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `const` | float | 譜面定数（小数点以下1桁表記） |
| `is_const_unknown` | bool | 譜面定数が未確定かどうか |
| `notes` | int \| null | ノーツ数（未設定の場合null/省略） |
| `statistics` | ChartStatisticsDTO \| null | 統計データは GET `/internal/songs/:displayid` の `content=full` 指定時のみ返却されます（譜面定数10.0未満はnull/省略）。 |

**ChartStatisticsDTO**:

`ChartStatisticsDTO` はレーティング帯をキーとするマップです。キーは "15.0", "15.1", ..., "17.6", "17.7+" の全てが含まれます（データがないレーティング帯は0で埋められます）。

**Map<string, ChartStatisticsByRatingDTO>**: キーはレーティング帯（"15.0" ~ "17.7+"）

**ChartStatisticsByRatingDTO**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `rank` | ChartRankStatisticsDTO | ランク別人数統計 |
| `lamp` | ChartLampStatisticsDTO | ランプ別人数統計 |

**ChartRankStatisticsDTO**: `{"s": int, "s_plus": int, "ss": int, "ss_plus": int, "sss": int, "sss_plus": int}`

**ChartLampStatisticsDTO**: `{"aj": int, "fc": int, "other": int}`

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/internal/songs/:displayid`
- **認証**: Cookie (任意)
- **レートリミット**: 認証なしは1分10回/IP
- **パスパラメータ**: `displayid` - 楽曲の表示用ID
- **クエリパラメータ**:
  - `content` (string, optional): `full` を指定すると統計データを含めます
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

レスポンスフィールドの詳細は GET `/internal/songs` と同様です。`content=full` 指定時のみ統計データを含めます。

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 500 Internal Server Error (`internal_error`): 楽曲が存在しない、またはサーバー内部エラー

### DELETE `/internal/songs/:displayid`
- **認証**: Cookie 必須
- **権限**: EDITOR (2) または ADMIN (3) 以上が必要
- **パスパラメータ**: `displayid` - 楽曲の表示用ID
- **概要**: 指定されたDisplayIDの楽曲を論理削除します。物理削除ではなく、`is_deleted` フラグを `true` に設定します。
- **レスポンス**: 204 No Content（成功時）

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 403 Forbidden (`forbidden`): 権限不足（PLAYER権限ではアクセス不可）
  - 500 Internal Server Error (`internal_error`): 楽曲が存在しない、またはサーバー内部エラー

### POST `/internal/songs/:displayid/restore`
- **認証**: Cookie 必須
- **権限**: EDITOR (2) または ADMIN (3) 以上が必要
- **パスパラメータ**: `displayid` - 楽曲の表示用ID
- **概要**: 指定されたDisplayIDの削除済み楽曲を復活させます。`is_deleted` フラグを `false` に設定します。
- **レスポンス**: 204 No Content（成功時）

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 403 Forbidden (`forbidden`): 権限不足（PLAYER権限ではアクセス不可）
  - 500 Internal Server Error (`internal_error`): 楽曲が存在しない、またはサーバー内部エラー

### PUT `/internal/songs`
- **認証**: Cookie 必須
- **権限**: EDITOR (2) または ADMIN (3) 以上が必要
- **概要**: 楽曲および譜面情報を一括更新します。既存データの修正専用で、新規追加・削除は行いません。
- **リクエスト**: JSON配列

```json
[
  {
    "id": "0123456789abcdef",
    "title": "楽曲タイトル",
    "artist": "アーティスト名",
    "genre_id": 1,
    "bpm": 180,
    "released_at": "2024-01-01T00:00:00Z",
    "jacket": "jacket_img_name",
    "charts": [
      {
        "difficulty_id": 3,
        "const": 14.5,
        "is_const_unknown": false,
        "notes": 1234
      }
    ]
  }
]
```

**リクエストフィールド（UpdateSongRequest）**:

| フィールド | 型 | 必須 | 説明 |
| ---------- | -- | ---- | ---- |
| `id` | string | ✓ | 楽曲の表示用ID（16文字の16進数文字列） |
| `title` | string | ✓ | 楽曲名 |
| `artist` | string | ✓ | アーティスト名 |
| `genre_id` | int \| null | | ジャンルID（マスタに存在する必要がある） |
| `bpm` | int \| null | | BPM（正の整数、nullの場合DBをNULLに更新） |
| `released_at` | string \| null | | リリース日（ISO8601形式、nullの場合DBをNULLに更新） |
| `jacket` | string \| null | | ジャケット画像ファイル名（nullの場合DBをNULLに更新） |
| `charts` | UpdateChartRequest[] | | 更新する譜面情報の配列 |

**UpdateChartRequest**:

| フィールド | 型 | 必須 | 説明 |
| ---------- | -- | ---- | ---- |
| `difficulty_id` | int | ✓ | 難易度ID（マスタに存在する必要がある） |
| `const` | float | ✓ | 譜面定数（0以上、小数点以下1桁表記） |
| `is_const_unknown` | bool | ✓ | 譜面定数が未確定かどうか |
| `notes` | int \| null | | ノーツ数（0以上、nullの場合DBをNULLに更新） |

- **レスポンス**: 204 No Content（成功時）

- **主なエラー**:
  - 400 Bad Request (`validation_failed`): バリデーションエラー
  - 400 Bad Request (`internal_error`): 存在しない楽曲・譜面・マスタIDの指定

### GET `/internal/songs/worldsend`
- **認証**: Cookie (任意)
- **レートリミット**: 認証なしは1分10回/IP
- **クエリパラメータ**: `include_deleted` - `true` を指定すると削除済み楽曲も含めて取得（オプション、デフォルト: `false`）
- **概要**: 全 WORLD'S END 楽曲を譜面情報付きで取得します。WORLD'S END は1曲1譜面が保証されています。
- **レスポンス**: 200 OK

```json
{
  "songs": [
    {
      "id": "0123456789abcdef",
      "title": "楽曲名",
      "artist": "アーティスト名",
      "genre_id": 1,
      "bpm": 180,
      "released_at": "2024-01-15T00:00:00Z",
      "official_idx": "123",
      "jacket": "img_filename",
      "we_star": 5,
      "we_kanji": "狂",
      "notes": 2000,
      "is_deleted": false
    }
  ]
}
```

**WorldsendSongDTO フィールド**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `id` | string | 楽曲の表示用ID |
| `title` | string | 楽曲名 |
| `artist` | string | アーティスト名 |
| `genre_id` | int \| null | ジャンルID |
| `bpm` | int \| null | BPM |
| `released_at` | string \| null | リリース日（ISO8601形式） |
| `official_idx` | string \| null | 公式インデックス |
| `jacket` | string \| null | ジャケット画像ファイル名 |
| `we_star` | int \| null | WORLD'S END 星の数（1～5） |
| `we_kanji` | string \| null | WORLD'S END カテゴリ漢字（光、蔵、改、狂、etc.） |
| `notes` | int \| null | ノーツ数 |
| `is_deleted` | bool | 削除フラグ |

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/internal/songs/worldsend/:displayid`
- **認証**: Cookie (任意)
- **レートリミット**: 認証なしは1分10回/IP
- **パスパラメータ**: `displayid` - 楽曲の表示用ID
- **概要**: 指定された DisplayID の WORLD'S END 楽曲を譜面情報付きで取得します。削除済み楽曲も取得可能です。
- **レスポンス**: 200 OK

```json
{
  "id": "0123456789abcdef",
  "title": "楽曲名",
  "artist": "アーティスト名",
  "genre_id": 1,
  "bpm": 180,
  "released_at": "2024-01-15T00:00:00Z",
  "official_idx": "123",
  "jacket": "img_filename",
  "we_star": 5,
  "we_kanji": "狂",
  "notes": 2000,
  "is_deleted": false
}
```

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 500 Internal Server Error (`internal_error`): 楽曲が存在しない、またはサーバー内部エラー

### DELETE `/internal/songs/worldsend/:displayid`
- **認証**: Cookie 必須
- **権限**: EDITOR (2) または ADMIN (3) 以上が必要
- **パスパラメータ**: `displayid` - 楽曲の表示用ID
- **概要**: 指定された DisplayID の WORLD'S END 楽曲を論理削除します。物理削除ではなく、`is_deleted` フラグを `true` に設定します。
- **レスポンス**: 204 No Content（成功時）

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 403 Forbidden (`forbidden`): 権限不足（PLAYER権限ではアクセス不可）
  - 500 Internal Server Error (`internal_error`): 楽曲が存在しない、またはサーバー内部エラー

### POST `/internal/songs/worldsend/:displayid/restore`
- **認証**: Cookie 必須
- **権限**: EDITOR (2) または ADMIN (3) 以上が必要
- **パスパラメータ**: `displayid` - 楽曲の表示用ID
- **概要**: 指定された DisplayID の削除済み WORLD'S END 楽曲を復活させます。`is_deleted` フラグを `false` に設定します。
- **レスポンス**: 204 No Content（成功時）

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 403 Forbidden (`forbidden`): 権限不足（PLAYER権限ではアクセス不可）
  - 500 Internal Server Error (`internal_error`): 楽曲が存在しない、またはサーバー内部エラー
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 403 Forbidden (`forbidden`): 権限不足（PLAYER権限ではアクセス不可）
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

**注意事項**:
- リクエストに含まれない譜面は変更されません（削除もされません）
- 存在しない `id` や `difficulty_id` を指定するとエラーになります
- マスタに存在しない `genre_id` や `difficulty_id` を指定するとエラーになります
- ポインタ型フィールド（`bpm`, `notes` など）にnullを指定すると、DBの該当カラムがNULLに更新されます

---

## `/internal/master` グループ

### GET `/internal/master`
- **認証**: Cookie 必須
- **概要**: フロントエンド向けにマスタデータ（ジャンル、難易度、定数不明選択肢、アカウント種別）を返却します。
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
  "is_const_unknown": [
    { "value": false, "label": "確定" },
    { "value": true, "label": "調査中" }
  ],
  "account_types": [
    { "id": 1, "name": "PLAYER" },
    { "id": 2, "name": "EDITOR" },
    { "id": 3, "name": "ADMIN" }
  ]
}
```

**レスポンスフィールド**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `genres` | MasterItemDTO[] | ジャンル一覧（ID順） |
| `difficulties` | MasterItemDTO[] | 難易度一覧（ID順） |
| `is_const_unknown` | BooleanChoiceDTO[] | 定数不明の選択肢 |
| `account_types` | MasterItemDTO[] | アカウント種別一覧（ID順） |

**MasterItemDTO**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `id` | int | マスタID |
| `name` | string | マスタ名称 |

**BooleanChoiceDTO**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `value` | bool | 真偽値 |
| `label` | string | 表示ラベル |

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): 認証が必要
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

---

## 公開API `/v1`

公開APIはAPIトークン認証を使用します。トークンは `Authorization: Bearer <token>` ヘッダーで送信してください。

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
      "artist": "アーティスト名",
      "genre": "ジャンル名",
      "bpm": 180,
      "release": "2024-01-15",
      "jacket": "jacket_001.png",
      "charts": {
        "MASTER": {
          "const": 14.5,
          "is_const_unknown": false,
          "notes": 1500
        },
        "BASIC": {
          "const": 8.5,
          "is_const_unknown": false,
          "notes": 450
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
| `songs[].artist` | string | アーティスト名 |
| `songs[].genre` | string\|null | ジャンル名 |
| `songs[].bpm` | number\|null | BPM |
| `songs[].release` | string\|null | リリース日（YYYY-MM-DD形式） |
| `songs[].jacket` | string\|null | ジャケット画像ファイル名 |
| `songs[].charts` | Map<string, ChartDTO> | 譜面情報のマップ。キーはBASIC, ADVANCED, EXPERT, MASTER, ULTIMA（大文字）の順序で固定されます。譜面が存在しない難易度はnullとなります |
| `songs[].charts[key].const` | number | 譜面定数（小数点以下1桁表記） |
| `songs[].charts[key].is_const_unknown` | boolean | 定数が推定値の場合true |
| `songs[].charts[key].notes` | number\|null | ノーツ数 |
| `songs[].charts[key].statistics` | Map<string, object>\|null | 統計データは GET `/v1/songs/:songId` の `content=full` 指定時のみ返却されます（譜面定数10.0未満はnull/省略）。 |
| `songs[].charts[key].statistics[tier]` | object | レーティング帯別の統計情報。キーは "15.0", "15.1", ..., "17.6", "17.7+" です（15.0未満のプレイヤーは集計対象外） |
| `songs[].charts[key].statistics[tier].rank` | object | ランク別人数統計 |
| `songs[].charts[key].statistics[tier].rank.s` | number | Sランク人数 (975,000-989,999) |
| `songs[].charts[key].statistics[tier].rank.s_plus` | number | S+ランク人数 (990,000-999,999) |
| `songs[].charts[key].statistics[tier].rank.ss` | number | SSランク人数 (1,000,000-1,004,999) |
| `songs[].charts[key].statistics[tier].rank.ss_plus` | number | SS+ランク人数 (1,005,000-1,007,499) |
| `songs[].charts[key].statistics[tier].rank.sss` | number | SSSランク人数 (1,007,500-1,008,999) |
| `songs[].charts[key].statistics[tier].rank.sss_plus` | number | SSS+ランク人数 (1,009,000+) |
| `songs[].charts[key].statistics[tier].lamp` | object | ランプ別人数統計 |
| `songs[].charts[key].statistics[tier].lamp.aj` | number | ALL JUSTICE人数 |
| `songs[].charts[key].statistics[tier].lamp.fc` | number | FULL COMBO人数 |
| `songs[].charts[key].statistics[tier].lamp.other` | number | その他ランプ人数 |

**統計情報について（GET `/v1/songs/:songId` の `content=full` 指定時のみ）**:
- 統計データは定期的なバッチ処理によって更新されます（リアルタイムではありません）
- レーティング帯は、プレイヤーの「ベスト枠平均レーティング」を基準に分類されます
- 集計対象: 譜面定数10.0以上の譜面 × ベスト枠平均15.0以上のプレイヤー
- レーティング帯17.7以上は "17.7+" として一括集計されます

- **主なエラー**:
  - 401 Unauthorized (`missing_token`): APIトークン未指定
  - 401 Unauthorized (`invalid_token`): 無効なAPIトークン
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/v1/songs/worldsend`
- **認証**: APIトークン必須
- **概要**: 全 WORLD'S END 楽曲を取得します（削除済み楽曲は除外）。WORLD'S END は1曲1譜面が保証されています。
- **レスポンス**: 200 OK

```json
{
  "songs": [
    {
      "id": "0123456789abcdef",
      "title": "楽曲名",
      "artist": "アーティスト名",
      "genre_id": 1,
      "bpm": 180,
      "released_at": "2024-01-15T00:00:00Z",
      "official_idx": "123",
      "jacket": "https://example.com/jacket.png",
      "we_star": 5,
      "we_kanji": "狂",
      "notes": 2000,
      "is_deleted": false
    }
  ]
}
```

**WorldsendSongDTO フィールド**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `id` | string | 楽曲の表示用ID |
| `title` | string | 楽曲名 |
| `artist` | string | アーティスト名 |
| `genre_id` | int \| null | ジャンルID |
| `bpm` | int \| null | BPM |
| `released_at` | string \| null | リリース日（ISO8601形式） |
| `official_idx` | string \| null | 公式インデックス |
| `jacket` | string \| null | ジャケット画像URL |
| `we_star` | int \| null | WORLD'S END 星の数（1～5） |
| `we_kanji` | string \| null | WORLD'S END カテゴリ漢字（光、蔵、改、狂、etc.） |
| `notes` | int \| null | ノーツ数 |
| `is_deleted` | bool | 削除フラグ（常にfalse） |

- **主なエラー**:
  - 401 Unauthorized (`missing_token`): APIトークン未指定
  - 401 Unauthorized (`invalid_token`): 無効なAPIトークン
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/v1/songs/worldsend/:displayid`
- **認証**: APIトークン必須
- **パスパラメータ**: `displayid` - 楽曲の表示用ID
- **概要**: 指定された DisplayID の WORLD'S END 楽曲を取得します（削除済み楽曲は除外）。
- **レスポンス**: 200 OK

```json
{
  "id": "0123456789abcdef",
  "title": "楽曲名",
  "artist": "アーティスト名",
  "genre_id": 1,
  "bpm": 180,
  "released_at": "2024-01-15T00:00:00Z",
  "official_idx": "123",
  "jacket": "https://example.com/jacket.png",
  "we_star": 5,
  "we_kanji": "狂",
  "notes": 2000,
  "is_deleted": false
}
```

- **主なエラー**:
  - 401 Unauthorized (`missing_token`): APIトークン未指定
  - 401 Unauthorized (`invalid_token`): 無効なAPIトークン
  - 404 Not Found (`internal_error`): 楽曲が存在しない
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/v1/songs/:songId`
- **認証**: APIトークン必須
- **パスパラメータ**:

| パラメータ | 型 | 説明 |
| ---------- | -- | ---- |
| `songId` | string | 楽曲の識別ID（16桁） |

- **????????**: `content=full` ?????????????????
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
  "charts": {
    "master": {
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

### GET `/v1/users/:username`
- **認証**: APIトークン必須
- **概要**: 指定されたユーザーのプロファイルとスコアレコードを取得します。非公開設定のユーザーは本人（APIトークンの所有者）以外 404 を返します。
- **パスパラメータ**:

| パラメータ | 型 | 説明 |
| ---------- | -- | ---- |
| `username` | string | ユーザー名 |

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
    "team_name": "チーム名",
    "team_color": "#FF5500",
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
    "all": []
  },
  "updated_at": "2024-12-20T10:00:00Z"
}
```

- **主なエラー**:
  - 401 Unauthorized (`missing_token`): APIトークン未指定
  - 401 Unauthorized (`invalid_token`): 無効なAPIトークン
  - 404 Not Found (`user_not_found`): ユーザーが見つからない（非公開ユーザー含む）
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

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
  player_name: string;
  rating: number | null;
  overpower_value: number | null;
  is_private: boolean;
  is_deleted: boolean;
}

// プロファイル＋レコード統合レスポンス
interface UserProfileWithRecordsDTO {
  username: string;
  player: PlayerDTO;
  records: UserRecordResponseDTO;
  updated_at: string;
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
  team_name: string | null;
  team_color: string | null;
  honors: HonorDTO[];
  created_at: string;
  updated_at: string;
}

interface HonorDTO {
  slot: number;
  name: string;
  type_name: string;
  image_url: string | null;
}

// レコード関連
interface PlayerRecordDTO {
  updated_at: string;
  difficulty: string;
  id: string;
  title: string;
  artist: string;
  const: number;
  is_const_unknown: boolean;
  score: number;
  rating: number;
  overpower: number;
  img: string;
  clear_lamp: string;
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
  all: PlayerRecordDTO[];
  worldsend: WorldsendRecordDTO[];  // WORLD'S END レコード（レーティング計算対象外）
}

// WORLD'S END レコード（スロット分類なし、レーティング計算なし）
interface WorldsendRecordDTO {
  updated_at: string;
  id: string;
  title: string;
  artist: string;
  we_star: number | null;         // WORLD'S END 星の数（1～5）
  we_kanji: string | null;        // WORLD'S END カテゴリ漢字（光、蔵、改、狂、etc.）
  notes: number | null;
  score: number;
  img: string;
  clear_lamp: string;
  combo_lamp: string | null;      // マスタ値が「NONE」の場合はnull
  full_chain: string | null;      // マスタ値が「NONE」の場合はnull
}

// エラーレスポンス
interface ErrorResponse {
  code: string;  // エラーコード (例: "invalid_token", "validation_failed")
}

// プレイヤーデータ登録結果
interface PlayerDataResult {
  player_id: number;
  app_ver: string;
  imported_at: string;
  summary: PlayerDataSummary;
  counts: PlayerDataCounts;
  diff_records: PlayerDataDiffSet;
  skipped_records: SkippedRecord[];
}

interface PlayerDataSummary {
  name: string;
  level: number;
  rating: number;
  last_played_at: string | null;
  overpower_value: number | null;
  overpower_percentage: number | null;
}

interface PlayerDataCounts {
  full_records_upserted: number;
  worldsend_records_upserted: number;
  full_records_changed: number;
  worldsend_records_changed: number;
  full_records_skipped: number;
  worldsend_records_skipped: number;
  honors_skipped: number;
}

interface PlayerDataDiffSet {
  full: PlayerDataDiff[];
  worldsend: PlayerDataDiff[];
}

// レスポンスサイズ削減のため、PlayerRecordDTOより軽量な専用型
interface PlayerDataDiffRecord {
  difficulty: string;
  title: string;
  const: number;
  is_const_unknown: boolean;
  score: number;
  clear_lamp: string;
  combo_lamp: string | null;  // マスタ値が「NONE」の場合はnull
  full_chain: string | null;  // マスタ値が「NONE」の場合はnull
}

interface PlayerDataDiff {
  before: PlayerDataDiffRecord | null;
  after: PlayerDataDiffRecord;
  changed_fields: string[];
}

interface SkippedRecord {
  record_type: 'full' | 'worldsend' | 'honor';
  reason: string;
  details: string;
}
```

---

## 運用上の注意

- `.env` の `JWT_SECRET` と `PW_PEPPER` は32文字以上の強度を推奨します。
- CORSの許可オリジンやCookie属性は環境ごとに設定ファイルで管理します。
- ユーザーを論理削除するとログインは失敗し、既存セッションも無効化されます。
