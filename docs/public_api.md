# 公開API仕様書

このドキュメントは `/v1` 配下で提供する外部向けAPIの仕様です。

**最終更新日**: 2026年01月17日

## ベースURL

アプリケーションの待ち受けポートは `.config/<environment>.settings.json` の `app_port` で設定します。ローカル開発では `http://localhost:${APP_PORT}/v1` がベースURLとなります。ステージング／本番ではアプリケーションドメインに同じパスを付与してください。

例:

- 開発: `http://localhost:${APP_PORT}/v1`
- ステージング: `https://staging.chunisupport.net/v1`
- 本番: `https://api.chunisupport.net/v1`

## 認証

- `Authorization: Bearer <token>` ヘッダーでAPIトークンを送信してください。
- トークンは `/internal/auth/api-tokens` で発行します。1ユーザーあたり常に1件のみ有効です。

### エラー時のレスポンス

`CustomHTTPErrorHandler` により以下のJSONが返ります。

```json
{
  "code": "invalid_token"
}
```

`code` フィールドにスネークケースのエラーコードが格納されます。詳細なエラーメッセージはクライアントには返却されません。

## レートリミット

外部APIには、サービス安定性のためレートリミットが適用されます。

| アカウント種別 | 制限 |
|---------------|------|
| ADMIN | 無制限 |
| PLAYER, EDITOR | 15分間に150リクエスト |

### 制限超過時のレスポンス

レートリミットを超過した場合、`429 Too Many Requests` が返されます。

```json
{
  "code": "too_many_requests"
}
```

### 注意事項

- レートリミットはAPIトークンに紐づくユーザーごとに適用されます。
- トークンバケットアルゴリズムを使用しているため、一時的なバーストは許容されます。
- 制限超過後、時間経過とともにリクエスト可能数が回復します。

## エンドポイント

| エンドポイント | メソッド | 認証 | 説明 |
|-----------------|----------|------|------|
| `/songs`        | GET      | 必須 | 楽曲一覧を取得します。 |
| `/songs/:songId`| GET      | 必須 | 楽曲詳細を取得します。 |
| `/users/:username` | GET   | 必須 | ユーザープロファイルとレコードを取得します。 |

---

### GET `/songs`
- **認証**: 必須
- **概要**: WORLD'S END以外の全楽曲を取得します（削除済み楽曲は除外）。
- **レスポンス**: 200 OK

**レスポンス例**:
```json
{
  "songs": [
    {
      "id": "0000000000000001",
      "title": "楽曲名",
      "artist": "アーティスト名",
      "genre": "ジャンル名",
      "bpm": 180,
      "release": "2024-01-15T00:00:00Z",
      "jacket": "https://example.com/jacket.png",
      "charts": {
        "MASTER": {
          "const": 14.5,
          "is_const_unknown": false,
          "notes": 1500
        }
      }
    }
  ]
}
```

※ `bpm`, `release`, `jacket` は `null` になる可能性があります。
※ `charts` 配下の `notes` は `null` になる可能性があります。
※ `charts` オブジェクトのキーはBASIC, ADVANCED, EXPERT, MASTER, ULTIMA（大文字）の順序で固定されます。
※ `const` は小数点以下1桁表記です。
※ 統計データは GET `/songs/:songId` の `content=full` 指定時のみ返却され、事前集計されたものです（リアルタイムではありません）。

---

### GET `/songs/:songId`
- **認証**: 必須
- **概要**: 指定された楽曲の詳細を取得します。
- **パスパラメータ**:
  - `songId`: 楽曲の識別ID（16桁）
- **クエリパラメータ**:
  - `content` (オプション): `full` を指定すると統計データを含めます
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
    "MASTER": {
      "const": 14.5,
      "is_const_unknown": false,
      "notes": 1500
    }
  }
}
```

レスポンスフィールドの詳細は GET /songs と同様です。`content=full` を指定することで統計データを含めることができます。

- **主なエラー**:
  - 404 Not Found (`song_not_found`): 楽曲が見つからない

---

### GET `/users/:username`
- **認証**: 必須
- **概要**: 指定されたユーザーのプロファイルとスコアレコードを取得します。非公開設定のユーザーは本人（APIトークンの所有者）以外 404 を返します。
- **パスパラメータ**:
  - `username`: ユーザー名
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
  - 404 Not Found (`user_not_found`): ユーザーが見つからない（非公開ユーザー含む）

---

## 今後の予定

- スコアランキングAPI
- Webhook / イベント配信

仕様が確定次第、本ドキュメントを更新します。
