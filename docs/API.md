# chunisupport-api API仕様書

このドキュメントは `chunisupport-api` が提供する内部API(`/internal` プレフィックス)、公開API(`/v1` プレフィックス)、chunirec互換API(`/compat/chunirec/2.0` プレフィックス)、reiwa互換API(`/compat/reiwa/1` プレフィックス)の仕様をまとめたものです。

**最終更新日**: 2026年07月27日

## ベースURLと環境

アプリケーションは `.config/<APP_ENV>.settings.json` の `app_port` で待ち受けポートを決定します。`APP_ENV=<name> go run main.go` で環境を切り替えます。APIレスポンスの日時は同設定の `timezone` で指定したIANAタイムゾーンへ変換され、内部処理とDB保存ではUTCを使用します。日時はRFC3339で返され、UTC固定の`Z`ではなく設定に応じたオフセット（例: `+09:00`）になります。

ローカル開発の例: `.config/<APP_ENV>.settings.json` で `app_port: 3002` を指定している場合、`http://localhost:3002`

主要なパス構成:

- 監視用API: `http://localhost:<app_port>/`
- 内部向けAPI: `http://localhost:<app_port>/internal`
- 公開API (APIトークン認証): `http://localhost:<app_port>/v1`
- chunirec互換API (APIトークン認証): `http://localhost:<app_port>/compat/chunirec/2.0`
- reiwa互換API (APIトークン認証): `http://localhost:<app_port>/compat/reiwa/1`

## CORS

すべてのエンドポイントでCORSが有効です。基本設定は `cors.*` を参照してください（設定方法は `docs/configuration.md` を参照）。
ただし `GET /healthz`、`OPTIONS /healthz`、`POST /internal/player-data/temp`、`OPTIONS /internal/player-data/temp` は、設定された許可オリジンに加えて `https://new.chunithm-net.com` も常に許可します。

メンテナンス中の待機時間をブラウザから参照できるよう、CORSの公開レスポンスヘッダーには `Retry-After` を含みます。

## 認証

### 内部API (`/internal`)

- 認証必須エンドポイントでは `Authorization: Bearer <Firebase ID Token>` を送信します。
- 認証必須エンドポイントでは Firebase ID トークンを検証し、ユーザー情報をリクエストコンテキストに格納します。
- Bearer 任意のエンドポイントでは、未認証時にレートリミットが適用されます。
- `token` Cookie や独自セッションは使用しません。

### 公開API (`/v1`, `/compat/chunirec/2.0`, `/compat/reiwa/1`)

- `Authorization: Bearer <token>` ヘッダーで API トークンを送信します。
- `/v1`、`/compat/chunirec/2.0`、`/compat/reiwa/1` はすべて API トークン認証です。
- `/v1/*/score-history*` のスコア履歴取得と `/v1/users/:username/rating-op-history` の公式指標履歴取得はAPIトークンが任意です。非公開ユーザーを参照する場合は、本人または承認済みフレンドのAPIトークンを送信します。
- トークンは `/internal/auth/api-tokens` で1ユーザーあたり最大10個まで発行できます。発行済みトークンに有効期限はありません。

## レートリミット（現行実装値）

ルーター実装（`internal/app/router.go`）および定数定義（`internal/info/info.go`）に基づく主要なレートリミットは以下です。

- `/internal/auth/signup`: **1分あたり5回/IP**
- `/internal/me/register-data`: **30秒あたり1回/ユーザー**
- `/internal/player-data/temp`: **1分あたり30回/IP**
- `/internal/player-data/commit`: **30秒あたり1回/ユーザー**
- `/internal/users/*`、`/internal/songs/*` および `/internal/worldsend-songs/*` の公開参照系（Firebase Bearer任意）: **未認証時のみ1分あたり60回/IP**
- `/v1/*`: **15分あたり150回（PLAYER） / 3,000回（EDITOR/EXTDEV） / 150,000回（ADMIN）**
- `/compat/chunirec/2.0/*`: **`/v1` と同一**
- `/compat/reiwa/1/*`: **`/v1` と同一**

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
| `maintenance_mode` | APIメンテナンス中の利用制限 |
| `internal_error` | 予期しないサーバーエラー |
| `friendship_limit_exceeded` | フレンド枠の上限超過 |
| `friendship_conflict` | 既に申請中またはフレンド成立済み |
| `friend_request_not_found` | 対象のフレンド申請が見つからない |

## メンテナンスモード

メンテナンスモード全体の振る舞いと運用上の規則は、[メンテナンスモード仕様](maintenance_mode.md) を参照してください。本節では、API利用者向けのアクセス制御とレスポンス形式を示します。

### メンテナンス中のアクセス制御

| 利用者 | 通常のエンドポイント | 状態変更 |
| --- | --- | --- |
| ADMIN | 利用可 | 利用可 |
| EDITOR | 利用可 | 不可 |
| PLAYER | `503 Service Unavailable` | 不可 |
| EXTDEV | `503 Service Unavailable` | 不可 |
| 未認証・不正な認証情報 | `503 Service Unavailable` | 不可 |

ADMIN / EDITORの判定は、`/internal` と `/` ではFirebase IDトークン、`/v1`、`/compat`、`/version` ではAPIトークンを使用します。スタッフとしてメンテナンスゲートを通過しても、各エンドポイントに既存の権限要件がある場合は、その認可を引き続き適用します。

次のリクエストはメンテナンスゲートの例外です。

- すべての `OPTIONS` リクエスト
- `GET /healthz`
- `GET /internal/system/status`
- `POST /internal/auth/login`

`POST /internal/auth/signup` は例外ではなく、メンテナンス中はスタッフ以外に `503 maintenance_mode` を返します。未登録パスはメンテナンス中も `404 not_found` を返します。

ログインでは既存のレートリミット、Turnstile検証、Firebase認証を維持します。認証に成功したADMIN / EDITORはログインでき、PLAYER / EXTDEVは `503 maintenance_mode` になります。認証に失敗した場合は、メンテナンス状態にかかわらず従来の認証エラーを返します。

### 標準APIのメンテナンス応答

互換API以外のメンテナンス遮断では、次のレスポンスを返します。コメントはこのレスポンスに含めず、`GET /internal/system/status` から取得してください。

```http
HTTP/1.1 503 Service Unavailable
Retry-After: 60
Cache-Control: no-store
Content-Type: application/json
```

```json
{
  "error": {
    "status": 503,
    "code": "maintenance_mode"
  }
}
```

### 互換APIのメンテナンス応答

`/compat/chunirec/2.0` と `/compat/reiwa/1` は、それぞれの既存互換エラー形式を維持します。`Retry-After: 60` と `Cache-Control: no-store` は標準APIと同様に付与します。

```json
{
  "error": {
    "code": 503,
    "message": "service unavailable.",
    "additional_message": ""
  }
}
```

## マスターデータ概要

主なマスタ定義は `migration/mysql/000001_init_schema.up.sql` に記載されています。

## エンドポイント一覧

| パス | メソッド | 認証 | 概要 |
| ---- | -------- | ---- | ---- |
| `/` | GET | 通常時不要 | アプリケーション名とビルド日を返します。メンテナンス中はFirebase認証済みのADMIN / EDITORのみ利用可 |
| `/healthz` | GET | 不要 | 外部監視向けの軽量な死活チェック |
| `/version` | GET | APIトークン(ADMIN) | APIのバージョン識別子取得 |
| `/internal/system/status` | GET | 不要 | APIの運用状態とメンテナンスコメントを取得 |
| `/internal/auth/login` | POST | Firebase Bearer + Turnstile | Firebase IDトークンとTurnstileでログイン検証 |
| `/internal/auth/signup` | POST | Firebase Bearer | Firebase IDトークンで初回ユーザー登録 |
| `/internal/auth/api-tokens` | GET | Firebase Bearer | APIトークン一覧取得 |
| `/internal/auth/api-tokens` | POST | Firebase Bearer | 名前付きAPIトークン発行 |
| `/internal/auth/api-tokens/:id` | PATCH | Firebase Bearer | APIトークン名変更 |
| `/internal/auth/api-tokens/:id` | DELETE | Firebase Bearer | APIトークン削除 |
| `/internal/admin/build-info` | GET | Firebase Bearer (ADMIN+) | 管理者画面向けAPIビルド情報取得 |
| `/internal/admin/user-stats` | GET | Firebase Bearer (ADMIN+) | 管理者画面向けユーザー集計取得 |
| `/internal/admin/maintenance` | PUT | Firebase Bearer (ADMIN+) | メンテナンス状態を開始・終了 |
| `/internal/me` | GET | Firebase Bearer | 自身のユーザー情報 |
| `/internal/me/privacy` | PUT | Firebase Bearer | 非公開設定更新 |
| `/internal/me` | DELETE | Firebase Bearer + X-Reauth-Token | アカウント物理削除 |
| `/internal/me/register-data` | POST | Firebase Bearer | CHUNITHMプレイヤーデータ登録 |
| `/internal/me/player-data/latest-update` | GET | Firebase Bearer | 自分の最新プレイヤーデータ登録結果を取得 |
| `/internal/me/player-data` | DELETE | Firebase Bearer | プレイヤー連携を解除し、プレイヤー関連レコードを削除 |
| `/internal/me/locked-songs` | POST | Firebase Bearer | 自分の未解禁曲を登録 |
| `/internal/me/locked-songs/batch` | POST | Firebase Bearer | 自分の未解禁曲をまとめて登録・解除 |
| `/internal/me/locked-songs/:displayid` | DELETE | Firebase Bearer | 自分の未解禁曲を解除 |
| `/internal/me/favorite-songs` | POST | Firebase Bearer | 自分のお気に入り楽曲を登録 |
| `/internal/me/favorite-songs/:displayid` | DELETE | Firebase Bearer | 自分のお気に入り楽曲を解除 |
| `/internal/friends` | GET | Firebase Bearer | フレンド一覧取得 |
| `/internal/friends/:username` | DELETE | Firebase Bearer | フレンド解除 |
| `/internal/friends/requests` | POST | Firebase Bearer | username完全一致でフレンド申請 |
| `/internal/friends/requests/received` | GET | Firebase Bearer | 自分宛てのフレンド申請一覧取得 |
| `/internal/friends/requests/sent` | GET | Firebase Bearer | 自分が送ったフレンド申請一覧取得 |
| `/internal/friends/requests/:username/accept` | POST | Firebase Bearer | フレンド申請承認 |
| `/internal/friends/requests/:username/reject` | POST | Firebase Bearer | フレンド申請拒否 |
| `/internal/friends/requests/:username` | DELETE | Firebase Bearer | 自分が送ったフレンド申請取り消し |
| `/internal/friend-rankings/songs/:displayid/charts/:difficulty` | GET | Firebase Bearer | 通常譜面のフレンドランキング取得 |
| `/internal/friend-rankings/worldsend-songs/:displayid` | GET | Firebase Bearer | WORLD'S END譜面のフレンドランキング取得 |
| `/internal/player-data/temp` | POST | なし | 未ログインでプレイヤーデータを一時受付（gzip JSON） |
| `/internal/player-data/commit` | POST | Firebase Bearer | 一時受付したプレイヤーデータを確定保存 |
| `/internal/me/goals` | GET | Firebase Bearer | 目標一覧を取得 |
| `/internal/me/goals` | POST | Firebase Bearer | 目標を作成 |
| `/internal/me/goals/order` | PUT | Firebase Bearer | 目標を並び替え |
| `/internal/me/goals/:id` | PUT | Firebase Bearer | 目標を更新 |
| `/internal/me/goals/:id` | DELETE | Firebase Bearer | 目標を削除 |
| `/internal/me/goal-groups` | GET | Firebase Bearer | 目標グループ一覧を取得 |
| `/internal/me/goal-groups` | POST | Firebase Bearer | 目標グループを作成 |
| `/internal/me/goal-groups/order` | PUT | Firebase Bearer | 目標グループを並び替え |
| `/internal/me/goal-groups/:id` | PUT | Firebase Bearer | 目標グループ名を更新 |
| `/internal/me/goal-groups/:id` | DELETE | Firebase Bearer | 目標グループを削除 |
| `/internal/me/record-filters` | GET | Firebase Bearer | 保存済みレコードフィルタ一覧を取得 |
| `/internal/me/record-filters` | POST | Firebase Bearer | レコードフィルタを保存 |
| `/internal/me/record-filters/:id` | PUT | Firebase Bearer | 保存済みレコードフィルタを更新 |
| `/internal/me/record-filters/:id` | DELETE | Firebase Bearer | 保存済みレコードフィルタを削除 |
| `/internal/users/` | GET | Firebase Bearer (ADMIN+) | 全ユーザー一覧取得（プライベート・プレイヤー未紐付けを含む） |
| `/internal/users/:username/profile` | GET | Firebase Bearer (任意) | ユーザー名とプレイヤー情報のみ取得 |
| `/internal/users/:username/updated-at` | GET | Firebase Bearer (任意) | ユーザー関連データの最終更新日時のみ取得 |
| `/internal/users/:username/rating` | GET | Firebase Bearer (任意) | レーティング枠のみ取得 |
| `/internal/users/:username/rating-op-history` | GET | Firebase Bearer (任意) | 公式RATING・公式OVER POWER・公式OP%履歴取得 |
| `/internal/users/:username/record` | GET | Firebase Bearer (任意) | レコード枠のみ取得 |
| `/internal/users/:username/record/songs/:displayid` | GET | Firebase Bearer (任意) | 通常楽曲1曲分のレコード取得 |
| `/internal/users/:username/record/songs/:displayid/:difficulty/history` | GET | Firebase Bearer (任意) | 通常譜面スコア履歴取得 |
| `/internal/users/:username/record/worldsend-songs/:displayid` | GET | Firebase Bearer (任意) | WORLD'S END楽曲1曲分のレコード取得 |
| `/internal/users/:username/record/worldsend-songs/:displayid/history` | GET | Firebase Bearer (任意) | WORLD'S ENDスコア履歴取得 |
| `/internal/users/:username/locked-songs` | GET | Firebase Bearer (任意) | ユーザーの未解禁曲一覧を取得 |
| `/internal/users/:username/favorite-songs` | GET | Firebase Bearer (任意) | ユーザーのお気に入り楽曲一覧を取得 |
| `/internal/users/:username` | GET | Firebase Bearer (任意) | プロファイルとレコードを一括取得 |
| `/internal/users/:username` | DELETE | Firebase Bearer (ADMIN+) | ユーザーの物理削除 |
| `/internal/songs/updated-at` | GET | Firebase Bearer (任意) | 楽曲情報キャッシュ用の最終更新日時のみ取得 |
| `/internal/songs` | GET | Firebase Bearer (任意) | WORLD'S END以外の楽曲一覧取得 |
| `/internal/songs/:displayid` | GET | Firebase Bearer (任意) | 楽曲詳細取得 |
| `/internal/songs/:displayid/stats/:difficulty` | GET | Firebase Bearer (任意) | 難易度別楽曲統計取得 |
| `/internal/songs/:displayid/best-slot-stats/:difficulty` | GET | Firebase Bearer (任意) | 難易度別ベスト枠採用統計取得 |
| `/internal/best-slot-rankings` | GET | Firebase Bearer (任意) | ベスト枠平均レート帯別の譜面採用率ランキング取得 |
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
| `/internal/courses/updated-at` | GET | Firebase Bearer (任意) | コースマスタキャッシュ用の最終更新日時のみ取得 |
| `/internal/courses` | GET | Firebase Bearer (任意) | 有効なコース一覧取得 |
| `/internal/courses/:displayid` | GET | Firebase Bearer (任意) | 有効なコース詳細取得 |
| `/internal/courses` | POST | Firebase Bearer (ADMIN+) | コース追加 |
| `/internal/courses/:displayid` | PUT | Firebase Bearer (EDITOR+) | コース名称・クラス更新 |
| `/internal/courses/:displayid` | DELETE | Firebase Bearer (ADMIN+) | コース論理削除 |
| `/internal/courses/:displayid/restore` | POST | Firebase Bearer (EDITOR+) | コース復元 |
| `/internal/editor/courses` | GET | Firebase Bearer (EDITOR+) | 削除済みを含むコース一覧取得 |
| `/internal/editor/courses/:displayid` | GET | Firebase Bearer (EDITOR+) | 削除済みを含むコース詳細取得 |
| `/internal/users/:username/record/courses` | GET | Firebase Bearer (任意) | ユーザーのコースレコード取得 |
| `/internal/users/:username/record/courses/:displayid` | GET | Firebase Bearer (任意) | ユーザーのコースレコード単件取得 |
| `/internal/master` | GET | 不要 | フロントエンド向けマスターデータ取得 |
| `/internal/master/versions` | GET | 不要 | バージョン一覧取得 |
| `/internal/master/honor-types` | GET | 不要 | 称号タイプ一覧取得 |
| `/v1/songs` | GET | APIトークン | 全楽曲一覧取得（WORLD'S END除く） |
| `/v1/songs` | PUT | APIトークン (EDITOR+) | 楽曲情報と譜面情報の一括更新 |
| `/v1/songs/chart-constant` | PATCH | APIトークン (EDITOR+) | 公式IDと難易度接頭辞による譜面定数更新 |
| `/v1/songs/:id` | GET | APIトークン | 楽曲詳細取得 |
| `/v1/songs/:id/stats/:difficulty` | GET | APIトークン | 難易度別楽曲統計取得 |
| `/v1/songs/:id/score-history/:difficulty` | GET | APIトークン（任意） | 通常譜面スコア履歴取得 |
| `/v1/worldsend-songs` | GET | APIトークン | WORLD'S END楽曲一覧取得 |
| `/v1/worldsend-songs/:id` | GET | APIトークン | WORLD'S END楽曲詳細取得 |
| `/v1/worldsend-songs/:id/score-history` | GET | APIトークン（任意） | WORLD'S ENDスコア履歴取得 |
| `/v1/users/:username` | GET | APIトークン | ユーザープロファイルとレコード取得 |
| `/v1/users/:username/rating-op-history` | GET | APIトークン（任意） | 公式RATING・公式OVER POWER・公式OP%履歴取得 |
| `/v1/courses` | GET | APIトークン | 有効なコースマスタ一覧取得 |
| `/v1/courses/:id` | GET | APIトークン | コースマスタ単件取得 |
| `/v1/users/:username/records/courses` | GET | APIトークン | ユーザーのコースレコード取得 |
| `/v1/master/versions` | GET | APIトークン | バージョン一覧取得 |
| `/compat/chunirec/2.0/music/showall` | GET | APIトークン | chunirec互換：全楽曲一覧取得 |
| `/compat/chunirec/2.0/music/show` | GET | APIトークン | chunirec互換：1楽曲情報取得 |
| `/compat/chunirec/2.0/records/showall` | GET | APIトークン | chunirec互換：通常譜面全レコード取得 |
| `/compat/chunirec/2.0/users/show` | GET | APIトークン | chunirec互換：ユーザープロフィール取得 |
| `/compat/reiwa/1/chunithm_record/original` | GET | APIトークン | reiwa互換：通常譜面全楽曲一覧取得 |

---

## 監視用エンドポイント

> **警告**: これらのエンドポイントはアプリケーションの稼働状況を確認するために使用されます。本番環境では、不正な情報漏洩を防ぐため、ネットワーク設定（例: ファイアウォール、ロードバランサ）によってアクセスを内部ネットワークや特定のIPアドレスに制限することが強く推奨されます。

### GET `/`
- **認証**: 通常時は不要。メンテナンス中はFirebase IDトークンで認証済みのADMIN / EDITORのみ利用できます。
- **レスポンス**: 通常時とメンテナンス中のADMIN / EDITORには 200 OK で、アプリケーション名とビルド日を返します。リビジョン（Git短縮ハッシュ）は公開しません。それ以外の利用者はメンテナンス中に 503 Service Unavailable (`maintenance_mode`) となります。

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
  - 204 No Content: 空レスポンス。メンテナンス中も同じレスポンスを維持します。

### GET `/version`
- **認証**: APIトークン (ADMIN)
- **レスポンス**:
  - 200 OK: APIのビルド識別子とGoバージョンを返します。

```json
{
  "app_name": "chunisupport-api",
  "build_date": "20240528",
  "commit_hash": "a1b2c3d",
  "go_version": "go1.27.0"
}
```

---

## システム状態・メンテナンスエンドポイント

### GET `/internal/system/status`

- **認証**: 不要
- **概要**: APIの運用状態、公開用コメント、最終更新日時を取得します。
- **メンテナンスゲート**: 例外。通常時・メンテナンス中ともに 200 OK を返します。
- **レスポンスヘッダー**: `Cache-Control: no-store`

通常時:

```json
{
  "status": "operational",
  "comment": "",
  "updated_at": "2026-07-26T12:00:00+09:00"
}
```

メンテナンス中:

```json
{
  "status": "maintenance",
  "comment": "データ更新のためメンテナンスを実施しています。",
  "updated_at": "2026-07-26T12:30:00+09:00"
}
```

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `status` | string | `operational` または `maintenance` |
| `comment` | string | 公開用メンテナンスコメント。通常時は空文字 |
| `updated_at` | string | 状態の最終更新日時（RFC3339） |

内部監査用の更新者IDは公開レスポンスに含めません。

### PUT `/internal/admin/maintenance`

- **認証**: Firebase Bearer (ADMIN)
- **概要**: メンテナンス状態の開始、稼働中コメントの更新、または終了を行います。EDITORは実行できません。
- **リクエストヘッダー**:
  - `Authorization: Bearer <Firebase ID Token>`
  - `Content-Type: application/json`

開始・稼働中コメント更新:

```json
{
  "enabled": true,
  "comment": "データ更新のためメンテナンスを実施しています。"
}
```

終了:

```json
{
  "enabled": false,
  "comment": ""
}
```

| フィールド | 型 | 必須 | バリデーション |
| ---------- | -- | ---- | -------------- |
| `enabled` | boolean | ✓ | `true` で開始、`false` で終了 |
| `comment` | string | `enabled: true`（開始・更新）時は必須 | 最大1,000 Unicodeコードポイント。改行可 |

コメントは `CRLF` と `CR` を `LF` へ正規化し、前後の空白を除去します。`LF` 以外の制御文字は使用できず、開始時に正規化後の空文字は指定できません。終了時はリクエストのコメントにかかわらず、保存値とレスポンスを空文字へ統一します。

- **レスポンス**: 200 OK。`GET /internal/system/status` と同じ形式を返します。
- **レスポンスヘッダー**: `Cache-Control: no-store`
- **冪等性**: 現在と `enabled` が同じで、正規化・無効化時の空文字化を反映したコメントも同じ場合はno-opとなり、最終更新日時は変更されません。
- **主なエラー**:
  - 400 Bad Request (`bad_request`): JSON不正、`enabled` 未指定、またはコメント不正
  - 401 Unauthorized: 通常時のFirebase認証失敗
  - 403 Forbidden (`forbidden`): EDITORを含むADMIN以外による状態変更
  - 503 Service Unavailable (`maintenance_mode`): メンテナンス中のPLAYER / EXTDEV / 未認証・不正な認証情報

---

## 管理者向け情報エンドポイント

### GET `/internal/admin/user-stats`

- **認証**: Firebase Bearer (ADMIN)
- **概要**: ユーザー数、プレイヤーデータ連携済みユーザー数、直近30日以内に更新されたプレイヤーデータ数を返します。
- **レスポンス**: 200 OK

```json
{
  "total_users": 100,
  "users_with_player_data": 80,
  "active_player_data_last_30_days": 50
}
```

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `total_users` | integer | 全ユーザー数 |
| `users_with_player_data` | integer | プレイヤーデータが紐付けられているユーザー数 |
| `active_player_data_last_30_days` | integer | `players.data_collected_at` が取得時点から30日前以降のプレイヤーデータ数（境界日時を含む） |

### GET `/internal/admin/build-info`
- **認証**: Firebase Bearer (ADMIN)
- **概要**: 管理者画面で表示するAPIのビルド情報を取得します。
- **レスポンス**: 200 OK

```json
{
  "app_name": "chunisupport-api",
  "build_date": "20240528",
  "commit_hash": "a1b2c3d",
  "go_version": "go1.27.0"
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
- **メンテナンス中**:
  - Firebase認証に成功したADMIN / EDITORはログインできます。
  - Firebase認証に成功したPLAYER / EXTDEVは 503 Service Unavailable (`maintenance_mode`) になります。
  - TurnstileまたはFirebase認証に失敗した場合は、従来の認証エラーを返します。

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
  - 503 Service Unavailable (`maintenance_mode`): メンテナンス中に認証済みのPLAYER / EXTDEVがログインを試みた
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
| `username` | string | ✓ | 5〜50文字、小文字英数字のみ。設定された禁止語に該当しないこと |
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
  - 503 Service Unavailable (`maintenance_mode`): メンテナンス中のスタッフ以外によるアクセス
  - 500 Internal Server Error (`internal_error`): 予期しないサーバーエラー

### POST `/internal/auth/api-tokens`
- **認証**: Firebase Bearer 必須
- **リクエスト**:

```json
{"name":"Discord Bot"}
```

- `name` は前後の空白を除いた1〜50文字で、同一ユーザー内で一意です。
- **レスポンス**: 201 Created

```json
{
  "id": 42,
  "name": "Discord Bot",
  "token": "plain-text-api-token",
  "token_prefix": "plain",
  "last_used_at": null,
  "created_at": "2026-07-22T12:34:56+09:00"
}
```

平文の `token` はこのレスポンスでのみ取得できます。サーバーにはSHA-256ハッシュと表示用の先頭5文字だけを保存します。

- **主なエラー**:
  - 400 Bad Request (`invalid_api_token_name`): 名前が不正
  - 400 Bad Request (`api_token_limit_exceeded`): 10個発行済み
  - 409 Conflict (`api_token_name_conflict`): 同名のトークンが存在する

### GET `/internal/auth/api-tokens`
- **認証**: Firebase Bearer 必須
- **レスポンス**: 200 OK

```json
{
  "tokens": [
    {
      "id": 42,
      "name": "Discord Bot",
      "token_prefix": "plain",
      "last_used_at": "2026-07-22T13:00:00+09:00",
      "created_at": "2026-07-22T12:34:56+09:00"
    },
    {
      "id": 1,
      "name": "既存のトークン",
      "token_prefix": null,
      "last_used_at": null,
      "created_at": "2026-04-16T12:34:56+09:00"
    }
  ]
}
```

- 未発行の場合は `tokens` が空配列になります。
- 旧仕様から移行したトークンは平文を復元できないため `token_prefix=null` のままです。認証には引き続き使用できます。
- `last_used_at` は認証成功時に更新されます。DB書き込みを抑えるため、最大1時間の遅延があります。
- **主なエラー**:
  - 401 Unauthorized (`missing_token` / `invalid_token`): 認証が必要
  - 500 Internal Server Error (`internal_error`): 予期しないサーバーエラー

### PATCH `/internal/auth/api-tokens/:id`
- **認証**: Firebase Bearer 必須
- **リクエスト**: `POST` と同じ `name`
- **レスポンス**: 200 OK。変更後のトークン管理情報を返します。平文の `token` は返しません。
- **主なエラー**:
  - 400 Bad Request (`invalid_api_token_id` / `invalid_api_token_name`): IDまたは名前が不正
  - 404 Not Found (`api_token_not_found`): 自分が所有する対象トークンが存在しない
  - 409 Conflict (`api_token_name_conflict`): 同名のトークンが存在する

### DELETE `/internal/auth/api-tokens/:id`
- **認証**: Firebase Bearer 必須
- **レスポンス**: 204 No Content
- 自分が所有するAPIトークンをID指定で削除します。削除後はそのトークンを認証に使用できません。
- **主なエラー**:
  - 401 Unauthorized (`missing_token` / `invalid_token`): 認証が必要
  - 400 Bad Request (`invalid_api_token_id`): IDが不正
  - 404 Not Found (`api_token_not_found`): 自分が所有する対象トークンが存在しない

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
| `account_type` | string | アカウントタイプ (PLAYER, EDITOR, ADMIN, EXTDEV) |
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

### GET `/internal/users/:username/favorite-songs`

対象ユーザーのお気に入り楽曲一覧を取得します。対象ユーザーが非公開の場合は、本人または承認済みフレンドのみ閲覧できます。

#### 認証

Firebase Bearer Token（任意）
- トークンあり: 非公開ユーザーでも本人または承認済みフレンドが取得可能
- トークンなし: 公開ユーザーのみ取得可能

#### リクエスト

| パラメータ | 型 | 必須 | 説明 |
| --- | --- | --- | --- |
| `username` | パス | 必須 | 対象ユーザー名（半角英数字4〜16文字） |

#### レスポンス（200 OK）

```json
{
  "items": [
    {
      "display_id": "0000000000000123",
      "title": "楽曲名",
      "jacket": "example.jpg",
      "favorited_at": "2026-07-05T12:34:56Z"
    }
  ]
}
```

| フィールド | 型 | 説明 |
| --- | --- | --- |
| `items` | array | お気に入り楽曲リスト。空の場合は `{"items":[]}` |
| `.display_id` | string | 楽曲識別子 |
| `.title` | string | 楽曲タイトル |
| `.jacket` | string or null | ジャケット画像ファイル名 |
| `.favorited_at` | string (ISO 8601) | お気に入り登録日時 |

#### エラー

| コード | HTTP | 条件 |
| --- | --- | --- |
| `user_not_found` | 404 | 対象ユーザーが存在しない、または非公開ユーザーを本人・承認済みフレンド以外が取得 |
| `player_not_linked` | 404 | ユーザーにプレイヤーが紐づいていない |

削除済み楽曲やWORLD'S END楽曲はレスポンスに含まれません。

### POST `/internal/me/favorite-songs`

認証済みユーザーのお気に入りに楽曲を登録します。

#### 認証

Firebase Bearer Token（必須）

#### リクエスト

```json
{
  "display_id": "0000000000000123"
}
```

| フィールド | 型 | 必須 | 説明 |
| --- | --- | --- | --- |
| `display_id` | string | 必須 | 楽曲識別子（16桁16進数） |

未知のトップレベルキーは拒否されます。

#### レスポンス（204 No Content）

成功時はレスポンスボディなし。

#### エラー

| コード | HTTP | 条件 |
| --- | --- | --- |
| `unauthorized` | 401 | 認証情報がない |
| `bad_request` | 400 | JSONデコード失敗 |
| `validation_failed` | 422 | `display_id` の形式不正 |
| `player_not_linked` | 404 | ユーザーにプレイヤーが紐づいていない |
| `song_not_found` | 404 | 楽曲が存在しない、論理削除済み、またはWORLD'S END |
| `favorite_song_limit_exceeded` | 400 | お気に入りが100件に達している（再登録時は発生しない） |

#### 備考

- 登録済み楽曲の再登録は成功し、登録日時を変更しません（冪等）
- お気に入りはプレイヤー単位で保持され、最大100件です

### DELETE `/internal/me/favorite-songs/:displayid`

認証済みユーザーのお気に入りから楽曲を解除します。

#### 認証

Firebase Bearer Token（必須）

#### リクエスト

| パラメータ | 型 | 必須 | 説明 |
| --- | --- | --- | --- |
| `displayid` | パス | 必須 | 楽曲識別子（16桁16進数） |

#### レスポンス（204 No Content）

成功時はレスポンスボディなし。

#### エラー

| コード | HTTP | 条件 |
| --- | --- | --- |
| `unauthorized` | 401 | 認証情報がない |
| `validation_failed` | 422 | `display_id` の形式不正 |
| `player_not_linked` | 404 | ユーザーにプレイヤーが紐づいていない |

#### 備考

- 未登録楽曲の解除も成功します（冪等）
- 論理削除済み、物理削除済み楽曲の解除も成功します

## `/internal/friends` グループ

フレンド関係は片方向レコード2件で管理します。申請中は `pending` の片方向レコード、承認後は双方向 `accepted` レコードです。拒否は申請レコード削除で表現し、`rejected` 状態は持ちません。`blocked` はDB上の予約ステータスですが、現時点のAPIでは作成・更新しません。

フレンド枠の上限は、自分から外向きの `pending` / `accepted` 合計100件です。`blocked` は将来仕様を検討するため、上限カウント対象外です。

非公開ユーザーのプロフィール、レーティング、レコード、スコア履歴、未解禁曲、お気に入り楽曲は、承認済みフレンドからは公開ユーザーと同じように閲覧できます。未認証または非フレンドからの参照では、ユーザー列挙を避けるため従来通り `user_not_found` 相当になります。

一覧で返す相手ユーザー概要は以下です。

```json
{
  "username": "frienduser",
  "player_level": 42,
  "player_name": "PLAYER",
  "rating": 15.25,
  "is_private": false,
  "requested_at": "2026-07-08T12:00:00Z",
  "accepted_at": "2026-07-08T12:05:00Z"
}
```

`username` とアカウントの存在は公開情報です。数値の内部ユーザーIDはレスポンスおよび操作パスへ公開しません。未承認の送受信申請では、非公開ユーザーの `player_level`、`player_name`、`rating` をすべて `null` とし、公開ユーザーの概要だけを表示できます。承認済みフレンドは非公開設定でも概要を返します。

### GET `/internal/friends`

承認済みフレンド一覧を、成立日時降順で取得します。

### POST `/internal/friends/requests`

`username` の完全一致でフレンド申請します。相手から申請中の場合は、即時に双方向承認します。

```json
{
  "username": "targetuser"
}
```

成功時は `204 No Content` です。

| コード | HTTP | 条件 |
| --- | --- | --- |
| `validation_failed` | 422 | 自分自身への申請、または不正な `username` |
| `user_not_found` | 404 | 対象ユーザーが存在しない |
| `friendship_limit_exceeded` | 400 | 自分の外向き `pending` / `accepted` が100件に達している |
| `friendship_conflict` | 409 | 既に申請中、承認済み、または相手から承認済み関係がある |

### GET `/internal/friends/requests/received`

自分宛ての申請一覧を申請日時降順で取得します。

### GET `/internal/friends/requests/sent`

自分が送った申請一覧を申請日時降順で取得します。

### POST `/internal/friends/requests/:username/accept`

指定ユーザーからの申請を承認し、双方向の `accepted` レコードを作成します。承認時に自分の外向き `pending` / `accepted` が100件に達している場合は失敗します。

成功時は `204 No Content` です。

| コード | HTTP | 条件 |
| --- | --- | --- |
| `validation_failed` | 400 | 自分自身を指定 |
| `username_too_short` / `username_too_long` / `username_invalid_char` | 400 | `username` の形式不正 |
| `user_not_found` | 404 | ロック対象ユーザーが存在しない |
| `friend_request_not_found` | 404 | 指定ユーザーからの `pending` 申請が存在しない |
| `friendship_limit_exceeded` | 400 | 自分の外向き `pending` / `accepted` が100件に達している |

### POST `/internal/friends/requests/:username/reject`

指定ユーザーからの `pending` 申請を削除します。成功時は `204 No Content` です。

| コード | HTTP | 条件 |
| --- | --- | --- |
| `validation_failed` | 400 | 自分自身を指定 |
| `username_too_short` / `username_too_long` / `username_invalid_char` | 400 | `username` の形式不正 |
| `user_not_found` | 404 | ロック対象ユーザーが存在しない |
| `friend_request_not_found` | 404 | 指定ユーザーからの `pending` 申請が存在しない |

### DELETE `/internal/friends/requests/:username`

指定ユーザーへの自分からの `pending` 申請を取り消します。成功時は `204 No Content` です。

| コード | HTTP | 条件 |
| --- | --- | --- |
| `validation_failed` | 400 | 自分自身を指定 |
| `username_too_short` / `username_too_long` / `username_invalid_char` | 400 | `username` の形式不正 |
| `user_not_found` | 404 | ロック対象ユーザーが存在しない |
| `friend_request_not_found` | 404 | 指定ユーザーへの自分からの `pending` 申請が存在しない |

### DELETE `/internal/friends/:username`

指定ユーザーとの双方向フレンド関係を削除します。成功時は `204 No Content` です。

| コード | HTTP | 条件 |
| --- | --- | --- |
| `validation_failed` | 400 | 自分自身を指定 |
| `username_too_short` / `username_too_long` / `username_invalid_char` | 400 | `username` の形式不正 |

## `/internal/friend-rankings` グループ

フレンドランキングは、自分と双方向 `accepted` のフレンドのうち、対象譜面をプレイ済みのユーザーだけを返します。未プレイユーザーは返しません。

### GET `/internal/friend-rankings/songs/:displayid/charts/:difficulty`

- **認証**: Firebase Bearer 必須
- **概要**: 通常譜面1つについて、自分と承認済みフレンド内の現在スコアランキングを取得します。
- **パスパラメータ**:

| パラメータ | 型 | 説明 |
| ---------- | -- | ---- |
| `displayid` | string | 楽曲の表示用ID |
| `difficulty` | string | 難易度（`BASIC`, `ADVANCED`, `EXPERT`, `MASTER`, `ULTIMA`。短縮形は既存のパス難易度変換に従う） |

- **ソート・順位**:
  - `score` 降順
  - 同点内の表示順は `updated_at` 降順、`username` 昇順
  - 同点は同順位とし、次順位は件数分進めます（例: `1, 1, 3`）

- **レスポンス**: 200 OK

```json
{
  "song": {
    "id": "0000000000000001",
    "title": "楽曲名",
    "artist": "アーティスト名"
  },
  "chart": {
    "difficulty": "MASTER",
    "const": 14.5,
    "is_const_unknown": false,
    "is_worldsend": false
  },
  "ranking": [
    {
      "rank": 1,
      "username": "frienduser",
      "player_name": "PLAYER",
      "score": 1009500,
      "rating": 16.65,
      "overpower": 87.123,
      "overpower_percent": 98.7654,
      "clear_lamp": "CLEAR",
      "combo_lamp": "ALL JUSTICE",
      "full_chain": null,
      "updated_at": "2026-07-09T12:00:00Z",
      "is_self": false
    }
  ],
  "my_rank": null,
  "total": 1
}
```

`my_rank` は自分が対象譜面を未プレイの場合 `null` です。`combo_lamp` と `full_chain` はマスタ値が `NONE` の場合 `null` です。

- **主なエラー**:
  - 422 Unprocessable Entity (`validation_failed`): `displayid` の形式不正
  - 400 Bad Request (`invalid_difficulty`): 難易度が不正
  - 401 Unauthorized (`missing_token` / `invalid_token`): 認証が必要
  - 404 Not Found (`chart_not_found`): 対象譜面が存在しない、または削除済み・WORLD'S END楽曲

### GET `/internal/friend-rankings/worldsend-songs/:displayid`

- **認証**: Firebase Bearer 必須
- **概要**: WORLD'S END譜面1つについて、自分と承認済みフレンド内の現在スコアランキングを取得します。
- **パスパラメータ**:

| パラメータ | 型 | 説明 |
| ---------- | -- | ---- |
| `displayid` | string | WORLD'S END楽曲の表示用ID |

- **ソート・順位**:
  - `score` 降順
  - 同点内の表示順は `updated_at` 降順、`username` 昇順
  - 同点は同順位とし、次順位は件数分進めます（例: `1, 1, 3`）

- **レスポンス**: 200 OK

```json
{
  "song": {
    "id": "0000000000000002",
    "title": "WORLD'S END楽曲名",
    "artist": "アーティスト名"
  },
  "chart": {
    "difficulty": "WORLD'S END",
    "level_star": 5,
    "attribute": "狂",
    "is_worldsend": true
  },
  "ranking": [
    {
      "rank": 1,
      "username": "frienduser",
      "player_name": "PLAYER",
      "score": 1009500,
      "clear_lamp": "CLEAR",
      "combo_lamp": "ALL JUSTICE",
      "full_chain": null,
      "updated_at": "2026-07-09T12:00:00Z",
      "is_self": false
    }
  ],
  "my_rank": null,
  "total": 1
}
```

WORLD'S END はレーティング・OVER POWER計算の対象外のため、通常譜面で返す `const` / `is_const_unknown` / `rating` / `overpower` / `overpower_percent` は返しません。ランキング表示に必要なスコアとランプは `ranking` の各要素に含まれます。

- **主なエラー**:
  - 422 Unprocessable Entity (`validation_failed`): `displayid` の形式不正
  - 401 Unauthorized (`missing_token` / `invalid_token`): 認証が必要
  - 404 Not Found (`chart_not_found`): 対象譜面が存在しない、または削除済み・通常楽曲
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/internal/users/:username/locked-songs`
- **認証**: Firebase Bearer 任意
- **概要**: 指定ユーザーのプレイヤーに紐づく未解禁曲一覧を取得します。通常未解禁とULTIMA未解禁は `is_ultima` で区別されます。対象ユーザーが非公開設定の場合、本人または承認済みフレンド以外にはユーザー未発見として扱われます。
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

スコア差分、集計差分、件数の厳密な定義は [プレイヤーデータ登録時の差分仕様](./player_data_registration_diff_specification.md) を参照してください。

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

プレイヤーデータ登録時と日次再計算バッチで、公式本枠から以下の3つのレーティング値を計算して `players` テーブルに保存します:

| カラム名 | 型 | 説明 |
| -------- | -- | ---- |
| `calculated_player_rating` | DECIMAL(6,4) | プレイヤーレーティング（ベスト枠30曲 + 新曲枠20曲の加重平均） |
| `best_average_rating` | DECIMAL(6,4) | `best` 本枠の平均レーティング |
| `new_average_rating` | DECIMAL(6,4) | `new` 本枠の平均レーティング |

**計算の詳細**:

1. **本枠の判定**:
   - 登録時は公式 `best` と `new` だけを集計し、候補枠を含めません
   - 日次バッチでは現行版プレイヤーの公式枠を保持し、旧版プレイヤーだけを最新マスタで再構築します
   - バッチの新曲判定にはリリース日を使います

2. **単曲レーティングの計算**: 
   - CHUNITHMのWiki記載の公式計算式に準拠（実装: [rating_service.go](../internal/domain/service/rating_service.go)）
   - 譜面定数が不明（`is_const_unknown=true`）な譜面も計算に含めます（除外するとより不正確になるため）

3. **プレイヤーレーティングの計算式**:
   ```
   プレイヤーレーティング = (ベスト枠30曲の合計 + 新曲枠20曲の合計) / 50
   ```

4. **ベスト枠平均の計算**:
   - `best` 本枠の実在件数で平均を算出

5. **新曲枠平均の計算**:
   - `new` 本枠の実在件数で平均を算出

**注意事項**:
- 日次バッチはキーセットページングでプレイヤーを順次処理します
- `official_player_rating` は入力データの `rating` フィールドから設定され、`calculated_player_rating` とは独立して保存されます
- `calculated_player_rating`、`best_average_rating`、`new_average_rating` は単曲レーティングを集計し、小数点以下4桁で切り捨てて保存されます

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
    ],
    "course": [
      {
        "score": 3023238,
        "is_clear": true,
        "cmb_lv": 1,
        "idx": "50020"
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
| `overpower.value` | number | ✓ | 公式オーバーパワー値（`players.official_overpower` に保存。通常譜面スコアから再計算する `overpower_value` とは別管理） |
| `overpower.percentage` | number | ✓ | CHUNITHM-NETに表示された公式OP%（`players.official_overpower_percent` に保存。通常譜面スコアから再計算する `overpower_percent` とは別管理） |
| `class_emblem.medal_class` | string | ✓ | クラスエンブレム（0埋め2桁） |
| `class_emblem.base_class` | string | ✓ | クラスエンブレムベース（0埋め2桁） |
| `team.name` | string | | チーム名 |
| `team.color` | string | | チームカラー |
| `honors` | object | | 称号情報（キー: スロット番号 "1"〜"3"） |
| `scores.standard` | array | ✓ | 通常譜面スコア配列 |
| `scores.worldsend` | array | ✓ | WORLD'S END スコア配列 |
| `scores.course` | array | | コーススコア配列。省略時は空配列として扱う |
| `updated_at` | string | ✓ | 更新日時 (ISO8601) |

`rating`、`overpower.value`、`overpower.percentage`は常にセットかつ小数第2位までの値として必要です。いずれかの省略・`null`・小数第3位以下を含む値は`422 validation_failed`となり、既存の公式値を更新しません。`overpower.percentage`は0以上100以下です。既存の`data_collected_at`より古い`updated_at`、同一`updated_at`で異なる公式指標を送信した場合、または同一`updated_at`で保存済み本文と異なる本文を送信した場合は`409 conflict`となり、現在値と履歴の時系列を維持します。同一`updated_at`の再送は、本文のSHA-256ハッシュも一致する場合のみ冪等な入力として受け付けます。

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

**コースエントリスキーマ (`scores.course` の各要素)**:

| フィールド | 型 | 必須 | 説明 |
| ---------- | -- | ---- | ---- |
| `idx` | string | ✓ | コースの公式インデックス |
| `score` | number | ✓ | 3曲合計スコア (0〜3,030,000) |
| `is_clear` | boolean | ✓ | コースクリア状態。コンボランプとは独立して保存する |
| `cmb_lv` | number | ✓ | 1=NONE、2=FULL COMBO、3=ALL JUSTICE |

- **レスポンス**: 200 OK。登録結果 `PlayerDataResult` を返します。
  - `profile.rating` と `summary.rating` は保存済み全スコアから再計算した `calculated_player_rating` です。入力データの公式RATINGではありません。
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
    "overpower_percent": 76.27011
  },
  "summary": {
    "name": "プレイヤー名",
    "level": 217,
    "rating": 17.29,
    "last_played_at": "2025-11-02T16:42:00+09:00",
    "overpower_value": 96123.91,
    "overpower_percentage": 76.27011
  },
  "metric_diffs": {
    "rating": { "before": 17.28, "after": 17.29, "delta": 0.01 },
    "overpower_value": { "before": 96120.123, "after": 96123.91, "delta": 3.787 },
    "overpower_percent": { "before": 76.26789, "after": 76.27011, "delta": 0.00222 }
  },
  "statistics": {
    "overall": {
      "total_high_score": { "before": 1183268650, "after": 1183287650, "delta": 19000 },
      "record_statistics": {
        "aj": { "before": 64, "after": 65, "delta": 1 },
        "fc": { "before": 284, "after": 285, "delta": 1 },
        "clr": { "before": 1173, "after": 1173, "delta": 0 },
        "fch": { "before": 25, "after": 25, "delta": 0 },
        "max": { "before": 3, "after": 3, "delta": 0 },
        "sss_plus": { "before": 120, "after": 121, "delta": 1 },
        "sss": { "before": 300, "after": 301, "delta": 1 },
        "ss_plus": { "before": 450, "after": 451, "delta": 1 },
        "ss": { "before": 700, "after": 701, "delta": 1 },
        "s_plus": { "before": 900, "after": 901, "delta": 1 },
        "s": { "before": 1050, "after": 1051, "delta": 1 }
      }
    },
    "by_difficulty": {
      "BASIC": {
        "total_high_score": { "before": 1000000, "after": 1000000, "delta": 0 },
        "record_statistics": {
          "aj": { "before": 1, "after": 1, "delta": 0 }, "fc": { "before": 1, "after": 1, "delta": 0 },
          "clr": { "before": 1, "after": 1, "delta": 0 }, "fch": { "before": 0, "after": 0, "delta": 0 },
          "max": { "before": 0, "after": 0, "delta": 0 }, "sss_plus": { "before": 0, "after": 0, "delta": 0 },
          "sss": { "before": 0, "after": 0, "delta": 0 }, "ss_plus": { "before": 0, "after": 0, "delta": 0 },
          "ss": { "before": 1, "after": 1, "delta": 0 },
          "s_plus": { "before": 1, "after": 1, "delta": 0 },
          "s": { "before": 1, "after": 1, "delta": 0 }
        }
      },
      "WE": {
        "total_high_score": { "before": 118000000, "after": 118019000, "delta": 19000 },
        "record_statistics": {
          "aj": { "before": 8, "after": 8, "delta": 0 }, "fc": { "before": 30, "after": 31, "delta": 1 },
          "clr": { "before": 100, "after": 101, "delta": 1 }, "fch": { "before": 2, "after": 2, "delta": 0 },
          "max": { "before": 1, "after": 1, "delta": 0 }, "sss_plus": { "before": 10, "after": 10, "delta": 0 },
          "sss": { "before": 25, "after": 25, "delta": 0 }, "ss_plus": { "before": 40, "after": 40, "delta": 0 },
          "ss": { "before": 60, "after": 60, "delta": 0 },
          "s_plus": { "before": 80, "after": 81, "delta": 1 },
          "s": { "before": 90, "after": 91, "delta": 1 }
        }
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
| `metric_diffs` | object | 計算レート、OVER POWER値、OP%の登録前後差分。各項目は `before` / `after` / `delta` を含みます |
| `statistics` | object | 通常譜面とWORLD'S ENDの登録前後集計。全体と難易度別の `before` / `after` / `delta` を含みます |
| `counts` | object | 各種レコードの処理件数。`*_actually_changed` は保存前状態と比較して `new` または `updated` になった件数 |
| `changes` | array | 実際に新規追加または更新されたスコア差分。0件の場合は空配列。詳細は最大100件 |
| `skipped_records` | array | スキップされたレコード情報。0件の場合は空配列 |

`statistics.overall` は通常譜面の全難易度、`statistics.by_difficulty` は通常難易度別およびWORLD'S ENDの集計です。`by_difficulty` にはデータの有無にかかわらず `BASIC` / `ADVANCED` / `EXPERT` / `MASTER` / `ULTIMA` / `WE` の6キーを返します。`WE` はWORLD'S ENDを表し、`overall` には含まれません。例では通常難易度を簡略化して `BASIC` だけ記載しています。

`total_high_score` は削除済み楽曲を除く対象グループのスコア合計です。`record_statistics` は `aj` / `fc` / `clr` / `fch` / `max` / `sss_plus` / `sss` / `ss_plus` / `ss` / `s_plus` / `s` の累積達成件数です。スコアランクは各ボーダー以上を数え、`s_plus` は990,000点以上、`s` は975,000点以上です。各値は `delta = after - before` で、減少時は負数になります。

`metric_diffs.rating` は保存済み全スコアから計算したレート、`metric_diffs.overpower_value` は通常楽曲レコードから再集計したOVER POWER値の差分です。`metric_diffs.overpower_percent` の `before` は更新前OVER POWER値を登録処理時点の `after` と同じ最大OVER POWER合計で割合へ変換した値であり、`delta` は小数点以下5桁で丸めたパーセントポイント差です。初回登録など登録前の値が存在しない場合、`before` と `delta` は `null` になります。

**`changes` の要素スキーマ**:

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `record_type` | string | `standard`、`worldsend`、または`course` |
| `change_type` | string | 未登録レコードは `new`、保存済みレコードの比較対象カラムが変化した場合は `updated` |
| `idx` | string | 楽曲の公式インデックス |
| `diff` | string | 通常譜面は大文字難易度名、WORLD'S ENDは`WE`。コースでは省略 |
| `course_class` | string | コースの場合のみコースクラス |
| `before` | object \| null | 更新前状態。`change_type=new` では `null` |
| `after` | object | 登録後状態 |

通常譜面とWORLD'S ENDの`before` / `after` は常に `score`, `clear_lamp`, `combo_lamp`, `full_chain` を含みます。コースではこれに`is_clear`を追加し、`course_class`でクラスを返します。ランプ名はマスタの `Name` を返し、`none`相当・未設定は`null`です。`slot` / `order`は保存されますが、差分判定および`changes`には含めません。同一payload内で同じ対象キーが複数回現れた場合は、最後の1件を保存・差分表示の対象にします。`changes`は`idx`を数値として昇順に並べ、同一`idx`の場合は`record_type`、`diff`の順で並びます。`idx`を数値として解釈できない値は末尾に並びます。`counts.*_actually_changed`は実際に変化した全件数で、`changes`はレスポンスサイズ抑制のため最大100件です。

- **主なエラー**:
  - 400 Bad Request (`bad_request` / `resource_not_found`): JSON構文不備・楽曲マスタ未登録など
  - 401 Unauthorized (`missing_token` / `invalid_token`): Bearerトークン欠如または無効
  - 409 Conflict (`conflict`): 別ユーザーのプレイヤーデータと競合
  - 413 Request Entity Too Large (`payload_too_large`): ボディサイズ5MB超過
  - 422 Unprocessable Entity (`validation_failed`): バリデーションエラー（スコア範囲外など）

---

### GET `/internal/me/player-data/latest-update`

認証済みユーザーに紐づくプレイヤーの最新データ登録結果を返します。

- **認証**: Firebase Bearer必須
- **成功レスポンス**: `200 OK`
- **レスポンス形式**: `POST /internal/me/register-data` の成功レスポンスから `skipped_records` を除き、保存形式の `schema_version` を追加したJSON
- `counts` 内の各スキップ件数はそのまま含みます
- 保存済みの結果がない場合は `204 No Content` を返します。既存プレイヤーが本機能の導入後にまだデータ登録していない場合などが該当します

```json
{
  "schema_version": 3,
  "player_id": 42,
  "app_ver": "0.1.0",
  "imported_at": "2026-07-16T12:00:00+09:00",
  "profile": {},
  "summary": {},
  "metric_diffs": {},
  "statistics": {},
  "counts": {},
  "changes": []
}
```

schema version 1の保存済み結果も取得できますが、`metric_diffs` は含まれません。schema version 2には `rating` と `overpower_value` の差分だけが含まれ、`overpower_percent` はschema version 3から追加されます。

- **主なエラー**:
  - 401 Unauthorized (`missing_token` / `invalid_token`): Bearerトークン欠如または無効
  - 404 Not Found (`player_not_linked`): プレイヤー未連携

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
- 1ユーザーあたり目標グループ上限は **20件** です。空グループも保持されます。
- 目標は「属性（`attributes`）」と「成果（`achievement`）」を持ちます。
- 外部API（`/v1`）には公開しません。

### GoalGroup オブジェクト

```json
{
  "id": 3,
  "name": "攻略中",
  "sort_order": 1,
  "created_at": "2026-01-01T09:00:00+09:00"
}
```

- `name` はtrim後1〜30文字で、制御文字を許可しません。
- 同一ユーザー内の名前は大文字小文字を区別せず重複不可です。
- `sort_order` はユーザー内で1から始まる連番です。
- 未分類はグループレコードを持たず、Goalの `group_id: null` で表します。表示上は常にグループの末尾です。

### Goal オブジェクト

```json
{
  "id": 1,
  "group_id": 3,
  "title": "マスター14+ 100枚",
  "achievement_type": "score_count",
  "achievement_params": { "score": 1007500, "count": 100 },
  "attributes": { "diff": 4, "const": { "min": 14.0, "max": 14.9 } },
  "invert_value": false,
  "invert_percentage": true,
  "sort_order": 1,
  "created_at": "2026-01-01T09:00:00+09:00"
}
```

| フィールド | 型 | 方向 | 説明 |
|---|---|---|---|
| `id` | `integer` | レスポンスのみ | 目標ID（自動採番） |
| `group_id` | `integer \| null` | 双方向 | 所属グループID。`null` または省略時は未分類 |
| `title` | `string` | 双方向 | 目標タイトル。trim後30文字以内、空文字不可、制御文字不可 |
| `achievement_type` | `string` | 双方向 | 成果種別コード（`achievement_types.code` と完全一致。大文字小文字の混在不可） |
| `achievement_params` | `object` | 双方向 | 成果種別ごとの可変パラメータ（詳細は後述） |
| `attributes` | `object` | 双方向 | 対象譜面の絞り込み条件（詳細は後述）。空オブジェクト `{}` は全譜面対象 |
| `invert_value` | `boolean` | 双方向 | `28/100` などの実数値表示用反転フラグ。サーバー側の達成判定には影響しない |
| `invert_percentage` | `boolean` | 双方向 | パーセンテージ表示用反転フラグ。サーバー側の達成判定には影響しない |
| `sort_order` | `integer` | レスポンスのみ | 同一グループ内での表示順。1から始まる連番 |
| `created_at` | `string` | レスポンスのみ | 作成日時（RFC3339、タイムゾーンオフセット付き） |

**作成・更新リクエストでの省略可否**:

- `title` / `achievement_type` / `achievement_params` は必須です。
- `attributes` は省略可能です。省略時は絞り込み条件なしとして扱います。明示する場合は空オブジェクト `{}` を推奨します。
- `group_id` は省略可能です。省略または `null` の場合は未分類として扱います。
- `invert_value` / `invert_percentage` はそれぞれ省略可能です。省略時は `false` として扱います。
- `id` / `sort_order` / `created_at` はレスポンス専用です。作成・更新リクエストには含めません。

### `achievement_type` 一覧

| code | 意味 |
|---|---|
| `rank_count` | 指定ランク（スコア）以上の譜面数 |
| `score_count` | 指定スコア以上の譜面数 |
| `avg_score` | 全譜面の平均スコア |
| `hardlamp_count` | 指定ハードランプの達成数 |
| `combolamp_count` | 指定コンボランプの達成数 |
| `rainbow_count` | BASIC〜MASTERと、存在する場合はULTIMAがすべてALL JUSTICEの楽曲数 |
| `total_score` | 全譜面のスコア合計 |
| `overpower_value` | 全譜面のOverPower値合計 |
| `overpower_percent` | 全譜面に対するOverPower達成割合（%） |

### `achievement_params` 仕様

`achievement_params` オブジェクト自体は必須です。ただし、成果種別によってはオブジェクト内の一部パラメータを省略または `null` にできます。省略可能なパラメータは以下の通りです。

| `achievement_type` | 省略可能なパラメータ | 省略/null時の扱い |
|---|---|---|
| `rank_count` / `score_count` | `count` | 対象譜面数（動的上限） |
| `hardlamp_count` / `combolamp_count` | `count` | 対象譜面数（動的上限） |
| `rainbow_count` | `count` | 対象楽曲数（動的上限） |
| `total_score` | `total` | 対象譜面数 × 1,010,000（動的上限） |
| `overpower_value` | `total` | 対象譜面の理論値OP合計（動的上限） |

上記以外のパラメータは必須です。例えば `score_count` の `score`、`avg_score` の `score`、`overpower_percent` の `total` は省略できません。

`rank_count` / `score_count` / `hardlamp_count` / `combolamp_count` / `rainbow_count` では、絶対目標値の `count` に代えて次のいずれかを指定できます。

- `remaining`: 動的上限から差し引く残数
- `percent`: 動的上限に対する目標割合（%）

`total_score` / `overpower_value` でも同様に、`total` に代えて `remaining` または `percent` を指定できます。絶対目標値、`remaining`、`percent` の非 `null` 値は相互排他です。`null` は未指定として扱うため、例えば `{"total": null, "remaining": 100}` は有効です。いずれも未指定の場合は動的上限そのものを目標値として扱います。

評価時の絶対目標値は、`remaining` の場合は「動的上限 - remaining」、`percent` の場合は「動的上限 × percent / 100」で算出します。件数系と `total_score` で割合計算結果が小数になる場合は切り上げ、`overpower_value` は小数値のまま扱います。

#### `rank_count` / `score_count`

`rank_count` と `score_count` は同じ構造・同じ判定ロジックです。`rank_count` はUIが「ランク由来の目標」として判別するために分けています。ランク境界はフロントエンドが保持し、バックエンドはスコア閾値のみを扱います。

```json
{ "score": 1000000, "count": 100 }
```

| パラメータ | 型 | 範囲 | 説明 |
|---|---|---|---|
| `score` | `integer` | 0〜1,010,000 | スコア閾値 |
| `count` | `integer \| null` | null または 1〜対象譜面数 | 目標件数。省略/null時は「対象譜面数（動的上限）」として扱います |
| `remaining` | `integer \| null` | null または 0〜対象譜面数 | 動的上限から差し引く残数 |
| `percent` | `number \| null` | null または 0〜100 | 動的上限に対する目標割合 |

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
| `remaining` | `integer \| null` | null または 0〜対象譜面数 | 動的上限から差し引く残数 |
| `percent` | `number \| null` | null または 0〜100 | 動的上限に対する目標割合 |

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
| `remaining` | `integer \| null` | null または 0〜対象譜面数 | 動的上限から差し引く残数 |
| `percent` | `number \| null` | null または 0〜100 | 動的上限に対する目標割合 |

**コンボランプ略称**:

| 略称 | マスタ名（`combo_lamp_types.name`） |
|---|---|
| `FC` | `FULL COMBO` |
| `AJ` | `ALL JUSTICE` |

#### `rainbow_count`

```json
{ "count": 100 }
```

BASIC・ADVANCED・EXPERT・MASTERがすべて存在する通常楽曲を対象にします。ULTIMAが存在する楽曲ではULTIMAも判定対象に加え、必要な全譜面のコンボランプが`ALL JUSTICE`なら達成楽曲として数えます。未プレイ譜面は未達成として扱います。

| パラメータ | 型 | 範囲 | 説明 |
|---|---|---|---|
| `count` | `integer \| null` | null または 1〜対象楽曲数 | 目標楽曲数。省略/null時は「対象楽曲数（動的上限）」として扱います |
| `remaining` | `integer \| null` | null または 0〜対象楽曲数 | 動的上限から差し引く残り楽曲数 |
| `percent` | `number \| null` | null または 0〜100 | 動的上限に対する目標割合 |

`attributes`は`genre`と`ver`のみ指定できます。`diff`、`const`、`chart_target`は指定できません。削除済み楽曲とBASIC〜MASTERのいずれかが存在しない楽曲は、対象楽曲数から除外します。固定`count`で保存済みの目標は、後から対象楽曲数が減少しても自動補正しません。

#### `total_score`

```json
{ "total": 100000000 }
```

| パラメータ | 型 | 範囲 | 説明 |
|---|---|---|---|
| `total` | `integer \| null` | null または 0〜対象譜面数 × 1,010,000 | スコア合計目標値。省略/null時は「対象譜面数 × 1,010,000（動的上限）」として扱います |
| `remaining` | `integer \| null` | null または 0〜対象譜面数 × 1,010,000 | 動的上限から差し引く残りスコア |
| `percent` | `number \| null` | null または 0〜100 | 動的上限に対する目標割合 |

#### `overpower_value`

```json
{ "total": 1000000.000 }
```

| パラメータ | 型 | 範囲 | 説明 |
|---|---|---|---|
| `total` | `number \| null` | null または 0〜対象譜面の理論値OP合計（小数点以下3桁まで） | OverPower合計目標値。省略/null時は「対象譜面の理論値OP合計（動的上限）」として扱います |
| `remaining` | `number \| null` | null または 0〜対象譜面の理論値OP合計（小数点以下3桁まで） | 動的上限から差し引く残りOverPower値 |
| `percent` | `number \| null` | null または 0〜100（小数点以下3桁まで） | 動的上限に対する目標割合 |

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

**許可キーは `diff` / `chart_target` / `const` / `genre` / `ver` のみ**です。未知キーは `goal_invalid_attributes` エラーになります。

```json
{
  "chart_target": "OP_TARGET",
  "const": { "min": 14.0, "max": 14.4 },
  "genre": [1, 2],
  "ver": [20, 21]
}
```

| フィールド | 型 | 必須 | 説明 |
|---|---|---|---|
| `diff` | `integer \| integer[]` | 任意 | 難易度ID（`difficulties.id` と同値、1〜5）。単一値または配列で指定可能。省略時は全難易度対象 |
| `chart_target` | `"OP_TARGET"` | 任意 | 曲ごとの理論OVER POWER対象譜面のみを対象にする。`diff` との同時指定は不可 |
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
- `chart_target` は `"OP_TARGET"` のみ許可。`diff` と同時指定した場合は `goal_invalid_attributes` エラー。

**配列入力の正規化**:
- `diff` / `genre` / `ver` は単一値（例: `"diff": 4`）と配列（例: `"diff": [3, 4]`）の両方を受け付けます。
- 配列は重複除去 + 昇順ソートで正規化されます。
- 要素数1の配列は単一値に正規化されます（例: `"diff": [4]` → `"diff": 4`）。
- 配列の実質上限は、対応するマスタデータの全件数です。
- レスポンスの `attributes` は正規化後の形式で返却されます（要素1ならスカラー、複数なら配列）。

### バリデーション方針

#### 境界（Handler/DTO）での検査

- リクエストボディは厳格デコード（`BindStrictJSON`）されるため、`group_id` / `title` / `achievement_type` / `achievement_params` / `attributes` / `invert_value` / `invert_percentage` 以外の未知キーを含むと `bad_request` になります。

#### Usecase層での業務ルール検査

1. **`title`**: trim後に空文字・30ルーン超・制御文字を含む場合はエラー
2. **`achievement_type`**: マスタキャッシュで検証。完全一致のみ許可（例: `score_count` は可、`Score_Count` は不可）
3. **`attributes`**: 許可キーのみ。各値をマスタ検証。`diff` / `genre` / `ver` は `integer | integer[]` を受け付け、配列は重複除去+昇順ソートで正規化（要素1はスカラー化）。`chart_target` は `"OP_TARGET"` のみ許可し、`diff` とは排他。`const` は小数1桁に丸め、`min <= max`、有効範囲 `[1.0, 16.0]`
4. **`achievement_params`**: `achievement_type` に対応する構造体へデコードし、パラメータ値を検証
5. **動的上限チェック**: `attributes` で絞り込まれた対象譜面数をもとに以下を検証
   - `rank_count` / `score_count` / `hardlamp_count` / `combolamp_count` の `count` ≤ 対象譜面数
   - `rainbow_count.count` ≤ 対象楽曲数
   - 譜面件数系成果種別の `remaining` ≤ 対象譜面数
   - `rainbow_count.remaining` ≤ 対象楽曲数
   - `total_score.total` ≤ 対象譜面数 × 1,010,000
   - `total_score.remaining` ≤ 対象譜面数 × 1,010,000
   - `overpower_value.total` ≤ 対象譜面の理論値OverPower合計
   - `overpower_value.remaining` ≤ 対象譜面の理論値OverPower合計
   - `percent` は 0〜100 の固定範囲
   - `overpower_percent.total` は 0〜100 の固定上限

#### 100件上限の担保

作成トランザクション内で `SELECT id FROM users WHERE id = ? FOR UPDATE` によりユーザー行をロックした後、`SELECT COUNT(*)` で件数を確認します。作成・削除・並び替えは同じユーザー行ロックを使用するため、件数と表示順を変更する同一ユーザーのリクエストは直列化されます。

### GET `/internal/me/goals`

自分が作成した目標を全件返します。グループは `goal_groups.sort_order` 昇順、未分類は末尾、その中でGoalの `sort_order` 昇順です。同順位が存在する不整合時のみ `id` 昇順を使用します。

**レスポンス**: 200 OK

```json
{
  "goals": [
    {
      "id": 1,
      "group_id": 3,
      "title": "マスター14+ 100枚",
      "achievement_type": "score_count",
      "achievement_params": { "score": 1007500, "count": 100 },
      "attributes": { "diff": 4, "const": { "min": 14.0, "max": 14.9 } },
      "invert_value": false,
      "invert_percentage": true,
      "sort_order": 1,
      "created_at": "2026-01-01T09:00:00+09:00"
    }
  ]
}
```

### POST `/internal/me/goals`

目標を新規作成します。100件上限を超える場合は `goal_limit_exceeded` エラーを返します。

新しい目標は指定グループ（未指定時は未分類）の末尾へ追加され、レスポンスにはサーバーが採番した `sort_order` が含まれます。

**リクエストボディ**: Goal オブジェクト（`id` / `sort_order` / `created_at` 除く）

```json
{
  "group_id": 3,
  "title": "マスター14+ 100枚",
  "achievement_type": "score_count",
  "achievement_params": { "score": 1007500, "count": 100 },
  "attributes": { "diff": 4, "const": { "min": 14.0, "max": 14.9 } },
  "invert_value": false,
  "invert_percentage": true
}
```

**レスポンス**: 201 Created（作成された Goal オブジェクト）

### PUT `/internal/me/goals/:id`

指定IDの目標を完全上書き更新します。他ユーザーの目標を指定した場合は `goal_not_found` を返します。`group_id` を変更した場合、移動元の順番を詰めて移動先の末尾へ追加します。

**リクエストボディ**: Goal オブジェクト（`id` / `sort_order` / `created_at` 除く）

**レスポンス**: 200 OK（更新後の Goal オブジェクト）

### PUT `/internal/me/goals/order`

指定グループ内の目標を並び替えます。そのグループに現在所属するすべての目標IDを、希望する表示順で1回ずつ指定します。`group_id: null` または省略時は未分類を対象にします。

```json
{
  "group_id": 3,
  "goal_ids": [12, 5, 9]
}
```

`goal_ids` に重複、欠落、存在しないID、または他ユーザー所有のIDが含まれる場合は `goal_invalid_order` を返し、並び順は変更しません。処理はユーザー行ロックを取得した単一トランザクションで実行されます。

`goal_ids` の欠落または `null` はリクエスト形式不正として `bad_request` を返します。

**レスポンス**: 204 No Content

### DELETE `/internal/me/goals/:id`

指定IDの目標を削除します。他ユーザーの目標を指定した場合は `goal_not_found` を返します。削除後、同じグループに残った目標の `sort_order` は1からの連番へ詰め直されます。

**レスポンス**: 204 No Content

### GoalGroup API

- `GET /internal/me/goal-groups`: 空グループを含む全グループを `sort_order` 順で返します。
- `POST /internal/me/goal-groups`: `{"name":"攻略中"}` で末尾に作成します。
- `PUT /internal/me/goal-groups/:id`: 同じ形式で名前を完全上書き更新します。
- `PUT /internal/me/goal-groups/order`: `{"group_ids":[3,1,2]}` のように所有する全グループIDを1回ずつ指定します。
- `DELETE /internal/me/goal-groups/:id`: 所属目標を現在の未分類末尾へ順序を保って移動し、グループを削除します。

作成・更新・削除・並び替えはユーザー行ロック下の単一トランザクションで実行します。

### Goal API エラーコード

| エラーコード | HTTP | 説明 |
|---|---|---|
| `goal_not_found` | 404 | 指定した goal が存在しない（他ユーザーの goal も含む） |
| `goal_limit_exceeded` | 400 | 100件上限を超えて作成しようとした |
| `goal_invalid_title` | 400 | `title` が trim 後に空文字、30文字超、または制御文字を含む |
| `goal_invalid_achievement_type` | 400 | `achievement_type` が不正（マスタに存在しない・大文字小文字不一致） |
| `goal_invalid_achievement_params` | 400 | `achievement_params` の形式不正・範囲不正・動的上限超過・`achievement_type` との組み合わせ不一致 |
| `goal_invalid_attributes` | 400 | `attributes` の形式不正・マスタ不整合・未許可キー・`const` 範囲外・`diff` 範囲外 |
| `goal_invalid_order` | 400 | `goal_ids` が指定グループ（`group_id` の省略・`null` は未分類）に現在所属する目標IDの集合と一致しない、または重複している |
| `invalid_goal_input` | 400 | goal 入力全般の不正（JSONデコード失敗など） |
| `goal_group_not_found` | 404 | グループが存在しない、または他ユーザー所有 |
| `goal_group_limit_exceeded` | 400 | 20件上限を超えてグループを作成しようとした |
| `goal_group_invalid_name` | 400 | グループ名が空、30文字超、または制御文字を含む |
| `goal_group_conflict` | 409 | 同一ユーザー内でグループ名が重複している |
| `goal_group_invalid_order` | 400 | `group_ids` が現在所有するグループIDの集合と一致しない、または重複している |

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
    "is_private": false
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
    "is_private": true
  }
]
```

#### AdminUserListResponse スキーマ

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `username` | string | ユーザー名 |
| `account_type` | string | アカウント種別（`PLAYER` / `EDITOR` / `ADMIN` / `EXTDEV`） |
| `created_at` | string | ユーザー作成日時 (ISO8601) |
| `updated_at` | string | ユーザー更新日時 (ISO8601) |
| `player_name` | string \| null | プレイヤー名（未連携の場合は `null`） |
| `rating` | number \| null | 保存済みスコアから算出したレーティング（未連携または未計算の場合は `null`） |
| `overpower_value` | number \| null | オーバーパワー値（未連携の場合は null） |
| `is_suspicious` | boolean | 不審アカウントフラグ |
| `is_private` | boolean | プライベートアカウントかどうか |

---

### GET `/internal/users/:username`
- **認証**: Firebase Bearer (任意)
- **レートリミット**: 認証なしは1分60回/IP
- **パスパラメータ**: `username` - 対象ユーザーのユーザー名
- **クエリパラメータ**:
    - `view` (任意): `rating` を指定すると、`records` は `updated_at`/`best`/`best_candidate`/`new`/`new_candidate` のみを返します（`standard`/`worldsend` は返しません）。`record` を指定すると、`records` は `updated_at`/`standard`/`worldsend` のみを返します。
    - `include_noplay` (任意): `true` を指定すると、`records.standard` と `records.worldsend` に未プレイ譜面を補完して返します。未プレイ補完データは `is_played=false` となり、`updated_at` / `clear_lamp` は `null` になります。`view=rating` と併用した場合は `include_noplay` は無視されます。`view=record` と併用した場合も補完されます。
- **レスポンス**: ユーザープロファイルとプレイヤーレコードを一括で返します。非公開設定のユーザーは本人または承認済みフレンド以外 404 を返します。プレイヤー未連携の場合は `200 OK` で `player` と `records` が `null` になります。
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
        "is_op_target": true,
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
- **レートリミット**: 認証なしで1分間60回/IP
- **パスパラメータ**: `username` - 対象ユーザーのユーザー名
- **レスポンス**: ユーザー名とプレイヤー情報のみを返します。非公開設定のユーザーは本人または承認済みフレンド以外 404 を返します。プレイヤー未連携の場合は `200 OK` で `player` が `null` になります。

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
- **レートリミット**: 認証なしで1分間60回/IP
- **パスパラメータ**: `username` - 対象ユーザーのユーザー名
- **レスポンス**: `profile.updated_at` と `rating/record` 系の元になるレコード最終更新日時のうち、新しい方のみを返します。非公開設定のユーザーは本人または承認済みフレンド以外 404 を返します。プレイヤー未連携の場合は `200 OK` で `updated_at` が `null` になります。

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
- **レートリミット**: 認証なしで1分間60回/IP
- **パスパラメータ**: `username` - 対象ユーザーのユーザー名
- **レスポンス**: レーティング枠のみを返します。非公開設定のユーザーは本人または承認済みフレンド以外 404 を返します。プレイヤー未連携の場合は各配列が空、`meta.updated_at` が `null` になります。

#### レスポンス例

```json
{
  "rating": 17.1234,
  "best_average": 17.2345,
  "new_average": 16.9567,
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
| `rating` | number \| null | BEST枠とNEW枠から計算したプレイヤーRATING（小数点以下4桁） |
| `best_average` | number \| null | BEST枠の平均RATING（小数点以下4桁） |
| `new_average` | number \| null | NEW枠の平均RATING（小数点以下4桁） |
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
  "rating": null,
  "best_average": null,
  "new_average": null,
  "best": [],
  "best_candidate": [],
  "new": [],
  "new_candidate": [],
  "meta": {
    "updated_at": null
  }
}
```

### GET `/internal/users/:username/rating-op-history`
- **認証**: Firebase Bearer (任意)
- **レートリミット**: 認証なしで1分間60回/IP
- **概要**: CHUNITHM-NETから取得した公式RATING・公式OVER POWER・公式OP%の履歴を、現在値を先頭に新しい順で返します。計算RATING・計算OVER POWER・計算OP%は含みません。非公開設定のユーザーは本人または承認済みフレンド以外 404 を返します。
- **パスパラメータ**: `username` - 対象ユーザーのユーザー名
- **レスポンス**: 200 OK

```json
{
  "entries": [
    {
      "rating": 17.25,
      "overpower": 12345.67,
      "overpower_percent": 98.76,
      "data_collected_at": "2026-08-08T12:00:00Z"
    }
  ]
}
```

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `entries` | array | 公式指標のスナップショット配列（現在値を先頭に新しい順） |
| `entries[].rating` | number | 公式RATING |
| `entries[].overpower` | number | 公式OVER POWER |
| `entries[].overpower_percent` | number \| null | 公式OP%。記録開始前の履歴は `null` |
| `entries[].data_collected_at` | string | CHUNITHM-NETからのデータ取得完了日時（ISO8601） |

- **主なエラー**:
  - 404 Not Found (`player_metric_history_not_found`): プレイヤー未連携などにより履歴が存在しない
  - 404 Not Found (`user_not_found`): ユーザーが存在しない、または非公開設定で閲覧できない

### GET `/internal/users/:username/record`
- **認証**: Firebase Bearer (任意)
- **レートリミット**: 認証なしで1分間60回/IP
- **概要**: 指定されたユーザーのレコード枠のみを取得します。非公開設定のユーザーは本人または承認済みフレンド以外 404 を返します。プレイヤー未連携の場合は `200 OK` で `standard` / `worldsend` が空配列、`meta.updated_at` が `null` になります。
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

### GET `/internal/users/:username/record/songs/:displayid`

- **認証**: Firebase Bearer (任意)
- **レートリミット**: 認証なしで1分間60回/IP
- **概要**: 指定した通常楽曲に属するユーザーレコードだけを返します。
- **クエリパラメータ**:
  - `include_noplay` (任意): `true` の場合は未プレイ譜面も補完します。
  - `difficulty` (任意): `BASIC` / `ADVANCED` / `EXPERT` / `MASTER` / `ULTIMA`。大文字小文字は区別しません。指定した難易度の譜面が曲に存在しない場合は `400 invalid_difficulty` を返します。
- **レスポンス**: `standard` は最大5件です。`meta.updated_at` は返却したプレイ済みレコードの最終更新日時で、該当レコードがなければ `null` です。

```json
{
  "standard": [
    {
      "is_played": true,
      "difficulty": "MASTER",
      "id": "0000000000000001",
      "score": 1009500,
      "updated_at": "2026-06-20T10:00:00Z"
    }
  ],
  "meta": {
    "updated_at": "2026-06-20T10:00:00Z"
  }
}
```

### GET `/internal/users/:username/record/worldsend-songs/:displayid`

- **認証**: Firebase Bearer (任意)
- **レートリミット**: 認証なしで1分間60回/IP
- **概要**: 指定した WORLD'S END 楽曲のユーザーレコードを返します。
- **クエリパラメータ**:
  - `include_noplay` (任意): `true` の場合は未プレイレコードを補完します。
- **レスポンス**: レコードがなければ `worldsend` は `null` です。`include_noplay=true` の場合は未プレイオブジェクトを返します。

```json
{
  "worldsend": null,
  "meta": {
    "updated_at": null
  }
}
```

両APIともユーザー不存在・非公開ユーザー・楽曲不存在・パスと楽曲種別の不一致は `404` を返します。`username` または `displayid` の形式不正は `400 validation_failed` です。

### GET `/internal/songs/updated-at`
- **認証**: Firebase Bearer (任意)
- **レートリミット**: 認証なしで1分間60回/IP
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

### GET `/internal/courses/updated-at`
- **認証**: Firebase Bearer (任意)
- **レートリミット**: 認証なしで1分間60回/IP
- **レスポンス**: `courses.updated_at` の最大値のみを返します。コースマスタキャッシュの更新判定に使用できます。削除済みコースも含みます。

#### レスポンス例

```json
{
  "updated_at": "2026-07-14T12:34:56Z"
}
```

#### CourseUpdatedAtDTO スキーマ

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `updated_at` | string \| null | `courses.updated_at` の最大値 (ISO8601)。コースが0件の場合は `null` |

### GET `/internal/users/:username/record/courses`
- **認証**: Firebase Bearer (任意)
- **レートリミット**: 認証なしで1分間60回/IP
- **パスパラメータ**: `username` - 対象ユーザーのユーザー名
- **クエリパラメータ**: `include_noplay` - `true` のとき未プレイコースを補完して返す
- **レスポンス**: 対象ユーザーのコースレコード一覧を返します。非公開設定のユーザーは本人または承認済みフレンド以外 404 を返します。プレイヤー未連携の場合は `courses` が空配列です。

#### レスポンス例

```json
{
  "courses": [
    {
      "display_id": "0123456789abcdef",
      "idx": "50020",
      "name": "CLASS I COURSE",
      "class": "1",
      "is_played": true,
      "score": 3029000,
      "is_clear": true,
      "combo_lamp": "FULL COMBO",
      "updated_at": "2026-07-10T08:00:00Z"
    }
  ],
  "meta": {
    "updated_at": "2026-07-14T10:00:00Z"
  }
}
```

#### CourseRecordListResponse スキーマ

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `courses` | CourseRecordDTO[] | コースレコード配列 |
| `meta` | UserRecordMetaDTO | メタ情報 |

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `meta.updated_at` | string \| null | COURSE マスタ更新日時（`courses.updated_at` 最大値）と、対象プレイヤーの COURSE レコード更新日時（`player_course_records.updated_at` 最大値）のうち新しい方。どちらもなければ `null`。プレイヤー未連携でもマスタ側のみ存在すればその値を返す |

各要素の `courses[i].updated_at` は当該プレイ済みレコードの更新日時であり、`meta.updated_at` とは独立する。未プレイ補完データでは `null`。

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
| `is_op_target` | boolean | 同一楽曲内でプレイヤーのOVER POWER合計に採用される通常譜面レコードかどうか。判定は全通常プレイ済みレコードを母集団に行い、未プレイ補完データは常に `false` |
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

**説明**: 指定されたユーザー名のユーザーを物理削除します。関連する目標を同一トランザクション内で先に削除し、目標グループ・プレイヤー・レコード・APIトークンなどの関連データも外部キー制約により削除されます。Firebase UID が連携されている場合は Firebase ユーザー削除も試行します（失敗時はサーバーログに記録し、APIレスポンスは成功を維持します）。

- **主なエラー**:
  - 401 Unauthorized (`unauthorized`): Bearerトークン欠如または無効
  - 403 Forbidden (`forbidden`): ADMIN権限が不足
  - 404 Not Found (`user_not_found`): ユーザーが存在しない
  - 400 Bad Request (`operation_failed`): 操作失敗（詳細隠蔽）

---

## `/internal/songs` グループ

### GET `/internal/songs`
- **認証**: Firebase Bearer (任意)
- **レートリミット**: 認証なしは1分60回/IP
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
      "is_new": true,
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
| `is_new` | bool | 新曲枠の対象かどうか |
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
- **レートリミット**: 認証なしは1分60回/IP
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
- **レートリミット**: 認証なしは1分60回/IP
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
        "aj": 25,
        "ajc": 3
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
      "median_score": 1007000,
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
        "aj": 4,
        "ajc": 1
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
      "median_score": 1008000,
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
| `stats[].combo` | object | コンボランプ別人数統計（none, fc, aj, ajc）。`aj` は AJC を除く ALL JUSTICE、`ajc` は ALL JUSTICE かつ 1,010,000 点の人数で、両者は排他的です |
| `stats[].clear` | object | クリアランプ別人数統計（failed, clear, hard, brave, absolute, catastrophy） |
| `stats[].average_score` | number\|null | レーティング帯別平均スコア（レコード数が0件の場合はnull） |
| `stats[].median_score` | number\|null | レーティング帯別中央スコア（レコード数が0件の場合はnull） |
| `stats[].player_count` | number | レーティング帯別プレイヤー数 |

**難易度パラメータについて**:
- パス内では小文字で指定: `basic`, `advanced`, `expert`, `master`, `ultima`, `worldsend`

- **主なエラー**:
  - 400 Bad Request (`invalid_difficulty`): 無効な難易度パラメータ
  - 404 Not Found (`song_not_found`): 楽曲が見つからない
  - 404 Not Found (`chart_not_found`): 指定された難易度の譜面が存在しない
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/internal/best-slot-rankings`

- **認証**: Firebase Bearer (任意)
- **レートリミット**: 認証なしは1分60回/IP
- **概要**: 指定したベスト枠平均レート帯について、各譜面がベスト枠に採用されているプレイヤーの割合を降順で取得します。削除済み楽曲、WORLD'S END譜面、採用人数0人の譜面は返しません。
- **クエリパラメータ**:
  - `rating_band` (必須): `/internal/master` の `rating_bands[].label`（例: `17.0`, `17.6+`）。`17.6+` を直接URLへ記述する場合、`+` は `%2B` にエンコードしてください。
  - `limit` (任意): 1～100。デフォルト50
  - `cursor` (任意): 前回レスポンスの `next_cursor`
- **ソート順**: `best_player_percentage` 降順 → `best_player_count` 降順 → 楽曲識別ID昇順 → 難易度名昇順
- **レスポンス**: 200 OK

```json
{
  "rating_band": "17.0",
  "eligible_player_count": 40,
  "ranking": [
    {
      "rank": 1,
      "song": {
        "id": "0000000000000001",
        "title": "楽曲名"
      },
      "chart": {
        "difficulty": "MASTER",
        "const": 14.8,
        "is_const_unknown": false
      },
      "best_player_count": 10,
      "best_player_percentage": 25.0,
      "average_score": 1007500.5
    }
  ],
  "next_cursor": "opaque-cursor"
}
```

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `rating_band` | string | 集計に使用したベスト枠平均レート帯のラベル |
| `eligible_player_count` | number | 指定レート帯の集計対象プレイヤー数 |
| `ranking` | array | ベスト枠採用率順の譜面一覧 |
| `ranking[].rank` | number | ランキング順位 |
| `ranking[].song.id` | string | 楽曲の識別ID（16桁） |
| `ranking[].song.title` | string | 楽曲名 |
| `ranking[].chart.difficulty` | string | 難易度（大文字） |
| `ranking[].chart.const` | number | 譜面定数 |
| `ranking[].chart.is_const_unknown` | boolean | 譜面定数が推定値の場合true |
| `ranking[].best_player_count` | number | 指定レート帯でこの譜面をベスト枠に持つプレイヤー数 |
| `ranking[].best_player_percentage` | number | `best_player_count / eligible_player_count * 100`（小数点以下4桁まで） |
| `ranking[].average_score` | number\|null | 指定レート帯でこの譜面をプレイした全プレイヤーの平均スコア（レコード数が0件の場合はnull） |
| `next_cursor` | string\|null | 次ページがある場合の不透明カーソル |

- **主なエラー**:
  - 422 Unprocessable Entity (`validation_failed`): `rating_band`、`limit`、`cursor` が不正
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/internal/songs/:displayid/best-slot-stats/:difficulty`

- **認証**: Firebase Bearer (任意)
- **レートリミット**: 認証なしは1分60回/IP
- **概要**: 指定した通常譜面について、ベスト枠平均レート帯別の採用人数、集計対象人数、採用率を取得します。WORLD'S ENDは対象外です。
- **パスパラメータ**:
  - `displayid`: 楽曲の表示用ID
  - `difficulty`: `basic`, `advanced`, `expert`, `master`, `ultima`
- **レスポンス**: 200 OK

```json
{
  "song_id": "0000000000000001",
  "stats": [
    {
      "rating_band": "ALL",
      "best_player_count": 520,
      "eligible_player_count": 1640,
      "best_player_percentage": 31.7073
    },
    {
      "rating_band": "17.0",
      "best_player_count": 10,
      "eligible_player_count": 40,
      "best_player_percentage": 25.0
    }
  ]
}
```

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `song_id` | string | 楽曲の識別ID（16桁） |
| `stats` | array | レート帯の表示順に並んだ統計。先頭は `ALL` |
| `stats[].rating_band` | string | ベスト枠平均レート帯のラベル |
| `stats[].best_player_count` | number | この譜面をベスト枠に持つプレイヤー数 |
| `stats[].eligible_player_count` | number | レート帯の集計対象プレイヤー数 |
| `stats[].best_player_percentage` | number\|null | ベスト枠採用率。集計対象が0人の場合はnull |

- **主なエラー**:
  - 400 Bad Request (`invalid_difficulty`): 無効な難易度またはWORLD'S END
  - 404 Not Found (`song_not_found`): 楽曲が見つからない
  - 404 Not Found (`chart_not_found`): 指定された難易度の譜面が存在しない
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/internal/users/:username/record/songs/:displayid/:difficulty/history`
- **認証**: Firebase Bearer（任意）
- **レート制限**: 認証なしの場合 1分間60回/IP
- **概要**: パスで指定したユーザーについて、通常譜面の現行ベストと過去のベストを新しい順で取得します。公開ユーザーは未認証で参照でき、非公開ユーザーは本人または承認済みフレンドが参照できます。
- **パスパラメータ**:
  - `username`: 対象ユーザー名
  - `displayid`: 楽曲の表示用ID
  - `difficulty`: `expert`, `master`, `ultima`
- **レスポンス**: 200 OK。形式は GET `/v1/songs/:id/score-history/:difficulty` と同一です。
- **主なエラー**:
  - 400 Bad Request (`validation_failed`): `username` が不正
  - 400 Bad Request (`invalid_difficulty`): 無効な難易度パラメータ
  - 400 Bad Request (`score_history_unsupported_difficulty`): 履歴対象外の難易度
  - 404 Not Found (`score_history_not_found`): スコア履歴が存在しない
  - 404 Not Found (`user_not_found`): ユーザーが存在しない、または非公開設定で閲覧できない

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
  "is_new": true,
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
| `is_new` | bool | - | 新曲枠の対象かどうか（省略時はfalse） |
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
    "is_new": true,
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
| `is_new` | bool \| null | | 新曲枠の対象かどうか（省略またはnullの場合はfalseとして更新） |
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
- **レートリミット**: 認証なしは1分60回/IP
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
- **レートリミット**: 認証なしは1分60回/IP
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

### GET `/internal/users/:username/record/worldsend-songs/:displayid/history`
- **認証**: Firebase Bearer（任意）
- **レート制限**: 認証なしの場合 1分間60回/IP
- **概要**: パスで指定したユーザーについて、WORLD'S END譜面の現行ベストと過去のベストを新しい順で取得します。公開ユーザーは未認証で参照でき、非公開ユーザーは本人または承認済みフレンドが参照できます。
- **パスパラメータ**:
  - `username`: 対象ユーザー名
  - `displayid`: WORLD'S END楽曲の表示用ID
- **レスポンス**: 200 OK。形式は GET `/v1/worldsend-songs/:id/score-history` と同一です。
- **主なエラー**:
  - 400 Bad Request (`validation_failed`): `username` が不正
  - 404 Not Found (`score_history_not_found`): スコア履歴が存在しない
  - 404 Not Found (`user_not_found`): ユーザーが存在しない、または非公開設定で閲覧できない

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

`EditorSongDTO` は `SongDTO` を embed（埋め込み）したDTOです。レスポンスJSONでは `SongDTO` の全フィールド（`id`, `title`, `reading`, `artist`, `genre`, `bpm`, `release`, `jacket`, `official_idx`, `maxop`, `is_maxop_unknown`, `op_target_difficulty`, `is_new`）がトップレベルにそのまま展開されます。さらに編集者向けとして、楽曲自体の `updated_at`、論理削除状態を表す `is_deleted`、および譜面ごとの `updated_at` を含む `charts` を返します。

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
    { "id": 3, "name": "ADMIN" },
    { "id": 4, "name": "EXTDEV" }
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
| `versions` | VersionDTO[] | バージョン一覧（起動日時点でリリース済みのバージョンをリリース日昇順） |
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
- **概要**: `/internal/master` の `versions` を単独で取得します。フロントエンドが内部マスタ全体に依存せず、バージョン一覧だけを段階的に分離取得するためのエンドポイントです。起動日時点でリリース済みのバージョンのみを返します。
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
- **概要**: バージョン一覧をリリース日昇順で返します。クライアントがバージョン辞書だけを独立取得する用途を想定しており、`id` は含みません。起動日時点でリリース済みのバージョンのみを返します。
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
| `versions` | VersionSummaryDTO[] | バージョン一覧（起動日時点でリリース済みのバージョンをリリース日昇順） |

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
      "is_new": true,
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
| `songs[].is_new` | boolean | 新曲枠の対象かどうか |
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

### PATCH `/v1/songs/chart-constant`
- **認証**: APIトークン必須
- **権限**: EDITOR または ADMIN 権限が必要
- **概要**: 通常楽曲の既存譜面について、公式ID、難易度名の先頭3文字、譜面定数だけを指定して更新します。更新後は `is_const_unknown` が `false` になります。

```json
{
  "official_idx": "1234567890",
  "difficulty": "MAS",
  "const": 14.7
}
```

| フィールド | 型 | 必須 | 説明 |
| ---------- | -- | ---- | ---- |
| `official_idx` | string | ✅ | 公式ID（最大10文字） |
| `difficulty` | string | ✅ | 難易度名の先頭3文字。`BAS` / `ADV` / `EXP` / `MAS` / `ULT`（大文字・小文字を区別しない） |
| `const` | number | ✅ | 譜面定数（0.0～16.0、小数第1位まで） |

- **レスポンス**: 200 OK（成功時）。更新後の楽曲オブジェクトを返します。
- **主なエラー**:
  - 400 Bad Request (`bad_request`): JSON形式またはContent-Typeが不正
  - 400 Bad Request (`validation_failed`): 必須項目、文字数などが不正
  - 400 Bad Request (`invalid_difficulty`): 難易度または譜面定数が不正
  - 401 Unauthorized (`missing_token`): APIトークン未指定
  - 401 Unauthorized (`invalid_token`): 無効なAPIトークン
  - 403 Forbidden (`forbidden`): 権限不足
  - 404 Not Found (`song_not_found`): 公式IDに対応する通常楽曲が存在しない
  - 404 Not Found (`chart_not_found`): 対象難易度の譜面が存在しない

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
| `id` | string | 楽曲ID |
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

### GET `/v1/worldsend-songs/:id`
- **認証**: APIトークン必須
- **パスパラメータ**: `id` - 楽曲ID
- **概要**: 指定されたIDの WORLD'S END 楽曲を譜面情報付きで取得します。
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

### GET `/v1/songs/:id`
- **認証**: APIトークン必須
- **パスパラメータ**:

| パラメータ | 型 | 説明 |
| ---------- | -- | ---- |
| `id` | string | 楽曲ID |

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

### GET `/v1/songs/:id/stats/:difficulty`
- **認証**: APIトークン必須
- **概要**: 指定楽曲の特定難易度のレーティング帯別統計を取得します。
- **パスパラメータ**:

| パラメータ | 型 | 説明 |
| ---------- | -- | ---- |
| `id` | string | 楽曲ID |
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

### GET `/v1/songs/:id/score-history/:difficulty`
- **認証**: APIトークン（任意）
- **概要**: 指定楽曲の指定難易度のスコア履歴を取得します。各譜面の現行ベストと過去のベストを新しい順で返します。公開ユーザーは未認証で参照できます。非公開ユーザーは本人または承認済みフレンドが参照できます。
- **制限**: 履歴は譜面ごとに最大50件で、レスポンス先頭の現行ベストを含めると最大51件です。
- **パスパラメータ**:

| パラメータ | 型 | 説明 |
| ---------- | -- | ---- |
| `id` | string | 楽曲ID |
| `difficulty` | string | 難易度名（小文字）: `expert`, `master`, `ultima` |

- **クエリパラメータ**:

| パラメータ | 型 | 必須 | 説明 |
| ---------- | -- | ---- | ---- |
| `username` | string | ✓ | 対象ユーザー名 |

- **レスポンス**: 200 OK

```json
{
  "entries": [
    {
      "score": 1009000,
      "clear_lamp": "ABSOLUTE",
      "combo_lamp": "ALL JUSTICE",
      "full_chain": "FULL CHAIN GOLD",
      "updated_at": "2026-04-27T12:34:56Z"
    }
  ]
}
```

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `entries` | array | スコア履歴エントリの配列（新しい順） |
| `entries[].score` | number | スコア |
| `entries[].clear_lamp` | string \| null | クリアランプ名称。未設定または解決不能の場合は `null` |
| `entries[].combo_lamp` | string \| null | コンボランプ名称。未設定または解決不能の場合は `null` |
| `entries[].full_chain` | string \| null | フルチェイン名称。未設定または解決不能の場合は `null` |
| `entries[].updated_at` | string | 更新日時（ISO8601） |

- **主なエラー**:
  - 400 Bad Request (`validation_failed`): `username` 未指定
  - 400 Bad Request (`score_history_unsupported_difficulty`): 指定された難易度が `expert`, `master`, `ultima` 以外
  - 404 Not Found (`score_history_not_found`): スコア履歴が存在しない（未プレイ）
  - 404 Not Found (`user_not_found`): ユーザーが存在しない、または非公開設定で閲覧できない

### GET `/v1/worldsend-songs/:id/score-history`
- **認証**: APIトークン（任意）
- **概要**: 指定WORLD'S END楽曲のスコア履歴を取得します。公開ユーザーは未認証で参照できます。非公開ユーザーは本人または承認済みフレンドが参照できます。
- **制限**: 履歴は譜面ごとに最大50件で、レスポンス先頭の現行ベストを含めると最大51件です。
- **パスパラメータ**:

| パラメータ | 型 | 説明 |
| ---------- | -- | ---- |
| `id` | string | WORLD'S END楽曲ID |

- **クエリパラメータ**:

| パラメータ | 型 | 必須 | 説明 |
| ---------- | -- | ---- | ---- |
| `username` | string | ✓ | 対象ユーザー名 |

- **レスポンス**: 200 OK（スキーマは通常譜面のスコア履歴と同一）
- **主なエラー**:
  - 400 Bad Request (`validation_failed`): `username` 未指定
  - 404 Not Found (`score_history_not_found`): スコア履歴が存在しない（未プレイ）
  - 404 Not Found (`user_not_found`): ユーザーが存在しない、または非公開設定で閲覧できない

### GET `/v1/users/:username/rating-op-history`
- **認証**: APIトークン（任意）
- **概要**: CHUNITHM-NETから取得した公式RATING・公式OVER POWER・公式OP%の履歴を、現在値を先頭に新しい順で返します。公開ユーザーは未認証で参照できます。非公開ユーザーは本人または承認済みフレンドが参照できます。
- **パスパラメータ**:

| パラメータ | 型 | 説明 |
| ---------- | -- | ---- |
| `username` | string | 対象ユーザー名 |

- **レスポンス**: 200 OK（スキーマはinternal APIの公式RATING・公式OVER POWER・公式OP%履歴と同一）
- **主なエラー**:
  - 404 Not Found (`player_metric_history_not_found`): プレイヤー未連携などにより履歴が存在しない
  - 404 Not Found (`user_not_found`): ユーザーが存在しない、または非公開設定で閲覧できない

### GET `/v1/users/:username`
- **認証**: APIトークン必須
- **概要**: 指定されたユーザーのプロファイルとスコアレコードを取得します。非公開設定のユーザーは本人（APIトークンの所有者）以外 404 を返します。プレイヤー未連携の場合は `200 OK` で `player` と `records` が `null` になります。
- `player.rating` は保存済みスコアから算出した `calculated_player_rating` です。入力データの公式RATINGではありません。
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

chunirec互換APIはchunirec APIとの互換性を持つエンドポイントです。認証方法は`/v1`と同様です。  
なお、楽曲IDにはChuniSupportのIDを使用します。chunirecの楽曲IDとは異なるため、他のchunirec互換APIから取得した楽曲IDを使用してください。

メンテナンス中の遮断でも既存の互換エラー形式を維持し、文字列の `maintenance_mode` はレスポンス本文へ含めません。本文とヘッダーは「互換APIのメンテナンス応答」を参照してください。

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
      "genre": "POPS&ANIME",
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
| `meta.genre` | string\|null | ジャンル名（`POPS & ANIME`は`POPS&ANIME`に変換） |
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
    "genre": "POPS&ANIME",
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
| `meta.genre` | string\|null | ジャンル名（`POPS & ANIME`は`POPS&ANIME`に変換） |
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

### GET `/compat/chunirec/2.0/records/showall`
- **認証**: APIトークン必須
- **概要**: 指定されたユーザーの通常譜面の全レコードをchunirec互換形式で取得します。未プレイ譜面とWORLD'S ENDは返しません。
- **クエリパラメータ**:
  - `user_name` (string, optional): 取得対象のユーザー名。未指定の場合はAPIトークン所有者のレコードを返します。
- **レスポンス**: 200 OK

```json
{
  "records": [
    {
      "id": "6a88218b1a936bd3",
      "diff": "EXP",
      "level": 10,
      "title": "B.B.K.K.B.K.K.",
      "const": 10,
      "score": 1003215,
      "rating": 11.32,
      "is_const_unknown": true,
      "is_clear": true,
      "is_fullcombo": false,
      "is_alljustice": false,
      "is_fullchain": false,
      "genre": "VARIETY",
      "updated_at": "1970-01-01T09:00:00+0900",
      "is_played": true
    }
  ]
}
```

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `records` | array | 通常譜面のプレイ済みレコード一覧 |
| `records[].id` | string | 楽曲の識別ID（16桁） |
| `records[].diff` | string | 難易度（`BAS`, `ADV`, `EXP`, `MAS`, `ULT`） |
| `records[].level` | number | 表記レベル（.0または.5） |
| `records[].title` | string | 楽曲名 |
| `records[].const` | number | 譜面定数 |
| `records[].score` | number | スコア |
| `records[].rating` | number | 単曲レーティング |
| `records[].is_const_unknown` | boolean | 定数が推定値の場合true |
| `records[].is_clear` | boolean | クリアランプが付いている場合true |
| `records[].is_fullcombo` | boolean | コンボランプがFULL COMBOまたはALL JUSTICEの場合true |
| `records[].is_alljustice` | boolean | コンボランプがALL JUSTICEの場合true |
| `records[].is_fullchain` | boolean | フルチェインランプが付いている場合true |
| `records[].genre` | string | ジャンル名（`POPS & ANIME`は`POPS&ANIME`に変換） |
| `records[].updated_at` | string | レコード更新日時（`YYYY-MM-DDTHH:mm:ss+0900`形式） |
| `records[].is_played` | boolean | 常にtrue |

- **主なエラー**:
  - 401 Unauthorized (`missing_token`): APIトークン未指定
  - 401 Unauthorized (`invalid_token`): 無効なAPIトークン
  - 404 Not Found (`user_not_found`): ユーザーが見つからない（非公開ユーザー含む）
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

### GET `/compat/chunirec/2.0/users/show`
- **認証**: APIトークン必須
- **概要**: 指定されたユーザーのプロフィールをchunirec互換形式で取得します。
- **クエリパラメータ**:
  - `user_name` (string, optional): 取得対象のユーザー名。未指定の場合はAPIトークン所有者のプロフィールを返します。
- **レスポンス**: 200 OK

```json
{
  "user_id": 0,
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
| `user_id` | number | 内部ユーザーIDを公開しないため常に `0` |
| `player_name` | string | プレイヤー名 |
| `title` | string\|null | 1番目の称号（スロット1） |
| `title_rarity` | string\|null | 1番目の称号のレアリティ（normal, copper, silver, gold, platinum, rainbow等）。ChuniSupport内部では"platina"を"platinum"に変換 |
| `level` | number | プレイヤーレベル |
| `rating` | string\|null | 保存済みスコアから算出したレーティング（小数点以下2桁の文字列） |
| `rating_max` | string\|null | 最大レーティング（現在はratingと同じ値） |
| `classemblem` | string\|null | クラスエンブレム（"1", "2", "3", "4", "5", "inf"） |
| `classemblem_base` | string\|null | クラスエンブレムベース（"1", "2", "3", "4", "5", "inf"） |
| `is_joined_team` | null | チーム参加状態（ChuniSupportでは保持しないため常にnull） |
| `updated_at` | string | プレイヤーデータの最終更新日時（RFC3339形式） |

- **主なエラー**:
  - 401 Unauthorized (`missing_token`): APIトークン未指定
  - 401 Unauthorized (`invalid_token`): 無効なAPIトークン
  - 404 Not Found (`user_not_found`): ユーザーが見つからない（非公開ユーザーを含む）
  - 500 Internal Server Error (`internal_error`): サーバー内部エラー

プレイヤー未連携ユーザーでは `200 OK` とJSONの `null` を返します。

---

## reiwa互換API `/compat/reiwa/1`

reiwa互換APIは外部ツールとの互換性を持つエンドポイントです。APIトークン認証を使用し、`Authorization: Bearer <token>` ヘッダーで送信してください。

メンテナンス中の遮断でも既存の互換エラー形式を維持し、文字列の `maintenance_mode` はレスポンス本文へ含めません。本文とヘッダーは「互換APIのメンテナンス応答」を参照してください。

### GET `/compat/reiwa/1/chunithm_record/original`

- **認証**: APIトークン必須
- **概要**: WORLD'S END以外の全楽曲の通常譜面情報を、譜面単位のフラットな配列で取得します（削除済み楽曲は除外）。
- **レスポンス**: 200 OK

```json
[
  {
    "title": "B.B.K.K.B.K.K.",
    "artist": "nora2r",
    "img": "d739ba44da6798a0",
    "genre": "VARIETY",
    "const": 4,
    "level": 4,
    "diff": "BAS",
    "notes": 333,
    "unknown": 0,
    "chunirec_id": "6a88218b1a936bd3",
    "idx": "3",
    "bpm": 170,
    "release": 14370588,
    "version": ""
  }
]
```

| フィールド | 型 | 説明 |
| ---------- | -- | ---- |
| `title` | string | 楽曲タイトル |
| `artist` | string | アーティスト名 |
| `img` | string | ジャケット画像識別子 |
| `genre` | string | ジャンル名（`POPS & ANIME`は`POPS&ANIME`に変換） |
| `const` | number | 譜面定数 |
| `level` | number | 表記レベル（.5区切り、例: 13+ → 13.5） |
| `diff` | string | 難易度（"BAS", "ADV", "EXP", "MAS", "ULT"） |
| `notes` | number | ノーツ数 |
| `unknown` | number | 譜面定数不明フラグ（0: 既知, 1: 不明） |
| `chunirec_id` | string | 楽曲の内部ID |
| `idx` | string | 公式インデックス |
| `bpm` | number | BPM |
| `release` | number | リリース日のJST0時Unixタイムスタンプ÷100 |
| `version` | string | バージョン名から"CHUNITHM "を除去した名称（初代CHUNITHMは空文字） |

- **ソート順**: `idx` を数値として昇順 → 難易度順（BAS → ADV → EXP → MAS → ULT）
- **主なエラー**:
  - 401 Unauthorized: APIトークン未指定または無効
  - 500 Internal Server Error: サーバー内部エラー

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
}

// ユーザー集計レスポンス（ADMIN用）
interface AdminUserStatisticsResponse {
  total_users: number;
  users_with_player_data: number;
  active_player_data_last_30_days: number;
}

// プロファイル＋レコード統合レスポンス
interface UserProfileWithRecordsDTO {
  username: string;
  player: PlayerDTO | null;
  records: UserRecordResponseDTO | null;
  updated_at: string | null;
}

interface UserRatingDTO {
  rating: number | null;
  best_average: number | null;
  new_average: number | null;
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
  is_op_target: boolean;
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

// システム状態・メンテナンス
interface SystemStatusResponse {
  status: 'operational' | 'maintenance';
  comment: string;
  updated_at: string;
}

type SystemMaintenanceUpdateRequest =
  | { enabled: true; comment: string }
  | { enabled: false; comment?: string };

// エラーレスポンス
interface ErrorResponse {
  error: {
    status: number;
    code: string;  // エラーコード (例: "invalid_token", "validation_failed", "maintenance_mode")
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
  metric_diffs: PlayerDataMetricDiffs;
  statistics: PlayerDataStatistics;
  counts: PlayerDataCounts;
  changes: PlayerDataRecordChange[];
  skipped_records: SkippedRecord[];
}

interface PlayerDataFloat64Diff {
  before: number | null;
  after: number | null;
  delta: number | null;
}

interface PlayerDataMetricDiffs {
  rating: PlayerDataFloat64Diff;
  overpower_value: PlayerDataFloat64Diff;
  overpower_percent: PlayerDataFloat64Diff;
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
  overall: PlayerDataStatisticsGroup;
  by_difficulty: Record<'BASIC' | 'ADVANCED' | 'EXPERT' | 'MASTER' | 'ULTIMA', PlayerDataStatisticsGroup>;
}

interface PlayerDataStatisticsGroup {
  total_high_score: PlayerDataNumberDiff;
  record_statistics: Record<'aj' | 'fc' | 'clr' | 'fch' | 'max' | 'sss_plus' | 'sss' | 'ss_plus' | 'ss' | 's_plus' | 's', PlayerDataNumberDiff>;
}

interface PlayerDataNumberDiff {
  before: number;
  after: number;
  delta: number;
}

interface PlayerDataCounts {
  standard_records_upserted: number;
  worldsend_records_upserted: number;
  standard_records_skipped: number;
  worldsend_records_skipped: number;
  honors_skipped: number;
  standard_records_actually_changed: number;
  worldsend_records_actually_changed: number;
  course_records_upserted: number;
  course_records_skipped: number;
  course_records_actually_changed: number;
}

interface PlayerDataRecordChange {
  record_type: 'standard' | 'worldsend' | 'course';
  change_type: 'new' | 'updated';
  idx: string;
  diff?: string;
  course_class?: string;
  before: PlayerDataRecordState | null;
  after: PlayerDataRecordState;
}

interface PlayerDataRecordState {
  score: number;
  clear_lamp: string | null;
  combo_lamp: string | null;
  full_chain: string | null;
  is_clear?: boolean;
}

interface SkippedRecord {
  record_type: 'standard' | 'worldsend' | 'course' | 'honor';
  reason: string;
  details: string;
}
```

---

## 運用上の注意

- エラーコードと内部理由コードの最新一覧は `docs/error_code_reason_codes.md` を参照してください。
- CORSの許可オリジンは環境ごとに設定ファイルで管理します。
- ユーザーを物理削除すると、ログインはできなくなり、関連データも削除されます。

## ユーザーデータ移行API

すべてのエンドポイントでFirebase Bearer認証が必要です。3つの操作ルートは常時利用できます。

### POST /internal/me/data-transfer/export

認証ユーザーの移行対象データを、HMAC-SHA256署名付きJSONファイルとして返します。成功時のContent-Typeはapplication/json、Content-Dispositionは`attachment; filename="chunisupport-transfer.json"`です。現在のスキーマバージョンは2で、プレイヤー現在値と公式指標履歴に`official_overpower_percent`（記録開始前は`null`）を含みます。移行元にプレイヤーがない場合は400を返します。

### POST /internal/me/data-transfer/validate

エクスポートされたJSONファイル本体をリクエストボディへ指定します。署名、形式、集約の不変条件、移行先マスター参照、移行先アカウントの空状態を検証します。称号本体は移行先に未登録でも検証を通過しますが、称号タイプは移行先に存在する必要があります。

形式が正しくても移行できない場合は200を返し、importableをfalseにします。blockersにはdestination_not_emptyまたはunresolved_referencesが入り、unresolved_referencesは先頭100件まで、unresolved_reference_countは全件数を表します。

### POST /internal/me/data-transfer/import

validateと同じファイルを再送します。すべての検証を再実行し、ユーザー行をロックして空状態を再確認した後、移行対象を1トランザクションで保存します。移行先に存在しない称号は、移行データの名称、称号タイプ、画像URLを使用して同じトランザクション内で称号マスタへ追加します。成功時は新しいplayer_idとセクション別保存件数を返します。

移行先が空でない場合は409 data_transfer_destination_not_empty、参照を解決できない場合は400 data_transfer_unresolved_referenceを返します。リクエスト上限は32MiB、3つの操作APIはユーザー単位で1分間に5回までです。
