# Firebase IDトークン検証最適化 設計・実装案

## 1. 背景と目的

現状の認証実装は `VerifyIDTokenAndCheckRevoked` を利用しており、IDトークン検証時に revoke / disabled の確認まで行っています。これにより安全性は高い一方で、読み取り系エンドポイントでも毎回外部問い合わせ寄りの検証コストが発生します。

本提案の目的は、**read系エンドポイントでは `VerifyIDToken` を利用してサーバ内検証中心に切り替え、レイテンシと外部依存を低減する**ことです。

## 2. 現状整理

### 2.1 認証フロー

- `FirebaseIDTokenMiddleware` / `OptionalFirebaseIDTokenMiddleware` が Bearer トークンを受け取り、`FirebaseAuthenticator` を呼び出す。
- `FirebaseAuthUsecase` が `TokenVerifier.VerifyIDToken` を通じて UID を取得し、ユーザーを解決する。
- `tokenVerifier` 実装は `VerifyIDTokenAndCheckRevoked` を使用している。

### 2.2 ルーティング上の適用候補

`/internal/users` と `/internal/songs` の公開GET群は `optionalFirebaseAuth` を使う read 系であり、`VerifyIDToken` 適用優先度が高い。

## 3. 要件定義

### 3.1 機能要件

- 読み取り系 API の認証では `VerifyIDToken` を利用可能にする。
- 書き込み系 API の認証では従来どおり revoke / disabled を確認する。
- 退会など recent sign-in を要する処理は既存の厳格検証を維持する。

### 3.2 非機能要件

- Clean Architecture の依存方向を維持する。
- 既存の公開 API 仕様は変更しない。
- 既存テストを壊さない。

## 4. 設計案

## 4.1 Usecase 構成（案Bに統一）

`internal/usecase/firebase_auth_usecase.go` の `TokenVerifier` インターフェース自体は変更せず、実装の注入を切り替える構成とする。

- revoke チェックありの厳格実装と、revoke チェックなしの read 最適化実装を用意する。
- ルーター組み立て時に read 用 / write 用で `FirebaseAuthUsecase` を別インスタンスとして生成し、適切な実装を注入する。
- ユースケース層はインフラ詳細（失効チェック有無）を意識せず、`TokenVerifier.VerifyIDToken` の呼び出しに統一する。

**利点**: Clean Architecture の依存方向を保ちながら、既存の usecase 抽象への影響を最小化できる。

## 4.2 Infra 実装の拡張

`internal/infra/firebaseauth/token_verifier.go` に以下を追加する。

- `VerifyIDTokenWithoutRevocationCheck(ctx, idToken)`
  - Firebase Admin SDK の `VerifyIDToken` を使用。
  - 既存のエラー変換ポリシー（`ErrInvalidIDToken` / `ErrInternalError`）を踏襲。

既存の `VerifyRecentSignIn` と `VerifyIDTokenAndCheckRevoked` ベースの厳格経路は残す。

## 4.3 Router での適用分離

`internal/app/router.go` の `registerRoutes` で、認証ミドルウェアを read/write で分離する。

- `firebaseAuthStrict`（従来: revoke チェックあり）
- `firebaseAuthReadOptimized`（新規: revoke チェックなし）
- `optionalFirebaseAuthReadOptimized`（新規: revoke チェックなし）

### 適用対象（推奨）

#### `VerifyIDToken` 適用

- `/internal/users` 公開GET群
  - `/:username/profile`
  - `/:username/updated-at`
  - `/:username/rating`
  - `/:username/record`
  - `/:username/locked-songs`
  - `/:username`
- `/internal/songs` 公開GET群
  - `/updated-at`
  - `/:displayid`
  - `/:displayid/stats/:difficulty`
  - `/worldsend`
  - `/worldsend/:displayid`

#### strict 維持

- `/internal/me` の PUT/POST/DELETE
- `/internal/users` の管理系操作
- `/internal/songs` の編集系操作
- `/internal/me` の `DELETE`（recent sign-in 要件あり）

### 条件付き（運用判断）

- `/internal/me` の GET (`/me`, `/me/goals`)
- `/internal/master` の GET (`/master`, `/master/versions`)

## 5. 実装手順（TDD）

1. **Red**: `token_verifier` の新メソッドに対するテストを先に追加する。
2. **Green**: `VerifyIDTokenWithoutRevocationCheck` を実装し、エラー変換を揃える。
3. **Red**: `firebase_auth_usecase` に read最適化経路のテストを追加する。
4. **Green**: usecase を実装し、既存ケースが壊れないことを確認する。
5. **Red**: `router` で read系に新認証を適用するテストを追加する。
6. **Green**: ルート配線を実装する。
7. **Refactor**: 命名整理・重複削減。
8. `go test ./...` 実行。
9. `gofmt -s -w .` 実行。

## 6. リスクと対策

- リスク: revoke 直後のトークンが read系で一時的に通る可能性。
  - 対策: 書き込み・権限操作は strict 経路を維持。
- リスク: 実装分岐が増えて認証責務が複雑化。
  - 対策: 認証モードを enum/定数化し、ユースケースの責務境界を明示。

## 7. 受け入れ条件

- read系対象ルートで `VerifyIDToken` が利用される。
- write/critical ルートで revoke チェックが維持される。
- 既存 API 仕様とレスポンス互換性を維持。
- 既存・追加テストがすべて成功。

## 8. 将来拡張案

- read経路に短TTLのサーバ内キャッシュを追加し、同一トークン検証のCPUコストをさらに低減する。
- メトリクス（strict/read の検証回数、失敗率、レイテンシ）を導入して、適用範囲を継続的に評価する。
