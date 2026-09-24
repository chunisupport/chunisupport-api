# Firebase IDトークン検証の実装

## 1. 概要

本アプリケーションの内部 API は、Firebase ID トークンを `Authorization: Bearer <Firebase ID Token>` で受け取り、Firebase UID に紐づく有効ユーザーを解決します。

ID トークン検証には次の 2 経路があります。

- strict 経路: `VerifyIDTokenAndCheckRevoked` を利用し、トークンの失効とユーザー無効化も確認する。
- read 最適化経路: `VerifyIDToken` を利用し、失効確認を省いて read エンドポイントの検証コストを抑える。

strict 経路は失効確認のためにリクエストごとに Firebase へ問い合わせるため、応答が遅くなり、ページ遷移などによるクライアント切断（`context.Canceled`）を招きやすくなります。そのため read 最適化経路は、読み取り専用で権限変更を伴わない GET エンドポイント（公開参照と、ログインユーザー自身・フレンド情報の参照）に利用します。書き込み、管理操作、退会、API トークン操作などは strict 経路を維持します。

## 2. 実装構成

### 2.1 Middleware

`internal/app/middleware/firebase_auth_middleware.go` で Firebase Bearer 認証を扱います。

- `FirebaseIDTokenMiddleware`
  - Bearer トークン必須。
  - `FirebaseAuthenticator.Authenticate` でユーザーを解決する。
  - トークン未指定は `missing_token`、検証失敗は usecase エラーを API エラーへ変換する。
- `OptionalFirebaseIDTokenMiddleware`
  - Bearer トークンがない場合は匿名として後続処理へ進む。
  - Bearer トークンがある場合のみ `AuthenticateOptional` でユーザーを解決する。
  - 未登録 Firebase UID は匿名扱いになり、`userEntity` は設定されない。

### 2.2 Usecase

`internal/usecase/firebase_auth_usecase.go` の `TokenVerifier` は、ID トークンから UID を取得する抽象です。

```go
type TokenVerifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (string, error)
}
```

`FirebaseAuthUsecase` は `TokenVerifier.VerifyIDToken` だけを呼び出すため、失効チェック有無は usecase ではなく注入される実装で切り替えます。これにより、usecase 層は Firebase Admin SDK や検証モードの詳細に依存しません。

`internal/usecase/read_optimized_token_verifier.go` では read 最適化用のラッパーを定義しています。

- strict verifier が `VerifyIDTokenWithoutRevocationCheck` を実装している場合、`VerifyIDToken` 呼び出しを失効確認なし検証へ委譲する。
- 実装していない場合は、渡された strict verifier をそのまま返す。
- `nil` が渡された場合は `nil` を返し、既存の usecase 側の nil チェックに委ねる。

### 2.3 Infra

`internal/infra/firebaseauth/token_verifier.go` の `tokenVerifier` は Firebase Admin SDK の `auth.Client` を利用します。

- `VerifyIDToken`
  - `VerifyIDTokenAndCheckRevoked` を呼び出す strict 検証。
  - invalid / revoked / disabled は `usecase.ErrInvalidIDToken` として返す。
  - SDK 内部エラー、nil クライアント、空 UID は `usecase.ErrInternalError` として返す。
- `VerifyIDTokenWithoutRevocationCheck`
  - `VerifyIDToken` を呼び出す read 最適化検証。
  - invalid は `usecase.ErrInvalidIDToken` として返す。
  - SDK 内部エラー、nil クライアント、空 UID は `usecase.ErrInternalError` として返す。
- `VerifyRecentSignIn`
  - `VerifyIDTokenAndCheckRevoked` を呼び出す。
  - UID と `auth_time` を返し、退会など recent sign-in が必要な処理で利用する。

## 3. ルーティング適用状況

`internal/app/router.go` では、strict 用と read 最適化用の `FirebaseAuthUsecase` を別インスタンスとして生成します。

```go
firebaseAuthUsecaseStrict := usecase.NewFirebaseAuthUsecase(db, userRepo, firebaseTokenVerifier)
firebaseAuthUsecaseReadOptimized := usecase.NewFirebaseAuthUsecase(db, userRepo, usecase.NewReadOptimizedTokenVerifier(firebaseTokenVerifier))
```

`registerRoutes` では以下のミドルウェアを使い分けます。

- `firebaseAuthStrict`: strict 必須認証。
- `firebaseAuthReadOptimized`: read 最適化の必須認証。
- `optionalFirebaseAuthReadOptimized`: read 最適化の任意認証。

同じパスプレフィックスで読み取りと書き込みが混在する `/internal/me` と `/internal/friends` は、read 用と書き込み用の Group を分けて登録します。

### 3.1 read 最適化の任意認証

以下の公開 GET に適用されています。

- `GET /internal/songs/updated-at`
- `GET /internal/courses/updated-at`
- `GET /internal/songs`
- `GET /internal/songs/:id`
- `GET /internal/songs/:id/stats/:difficulty`
- `GET /internal/songs/:id/best-slot-stats/:difficulty`
- `GET /internal/best-slot-rankings`
- `GET /internal/worldsend-songs`
- `GET /internal/worldsend-songs/:id`
- `GET /internal/courses`
- `GET /internal/courses/:id`
- `GET /internal/users/:username/updated-at`
- `GET /internal/users/:username/profile`
- `GET /internal/users/:username/rating`
- `GET /internal/users/:username/rating-op-history`
- `GET /internal/users/:username/record`
- `GET /internal/users/:username/record/songs/:id`
- `GET /internal/users/:username/record/songs/:id/:difficulty/history`
- `GET /internal/users/:username/record/worldsend-songs/:id`
- `GET /internal/users/:username/record/worldsend-songs/:id/history`
- `GET /internal/users/:username/record/courses`
- `GET /internal/users/:username/record/courses/:id`
- `GET /internal/users/:username/locked-songs`
- `GET /internal/users/:username/favorite-songs`
- `GET /internal/users/:username`

これらは Firebase Bearer 任意の公開参照エンドポイントです。Bearer トークンがある場合のみ失効確認なしで UID を検証し、未認証時は匿名として処理します。未認証時には匿名 IP レートリミットが適用されます。

### 3.2 read 最適化の必須認証

以下のログインユーザー向け GET に適用されています。

- `GET /internal/me`
- `GET /internal/me/player-data/latest-update`
- `GET /internal/me/goals`
- `GET /internal/me/goal-groups`
- `GET /internal/me/record-filters`
- `GET /internal/friends`
- `GET /internal/friends/requests/received`
- `GET /internal/friends/requests/sent`
- `/internal/friend-rankings` 配下
- `/internal/friend-comparisons` 配下

### 3.3 strict 必須認証

以下は strict 必須認証です。

- `/internal/auth/api-tokens` の GET / POST と `/internal/auth/api-tokens/:id` の PATCH / DELETE
- `/internal/me` 配下の書き込み操作（3.2 の GET を除く）
- `POST /internal/player-data/commit`
- `/internal/users` の管理系操作
- `/internal/songs` の編集系操作
- `/internal/worldsend-songs` の編集系操作
- `/internal/courses` の編集系操作
- `/internal/editor/songs` 配下
- `/internal/editor/worldsend-songs` 配下
- `/internal/editor/courses` 配下
- `/internal/honors` 配下
- `/internal/admin` 配下
- `/internal/friends` 配下の書き込み操作（3.2 の GET を除く）

`/internal/master` 配下と `GET /internal/system/status` は認証不要です。

退会処理では通常の strict Bearer 認証に加えて、`X-Reauth-Token` の recent sign-in 検証も行います。

## 4. セキュリティ上の扱い

read 最適化経路では、Firebase ID トークン自体の署名、有効期限、基本的な形式は検証しますが、失効確認は行いません。そのため、トークン失効直後や Firebase ユーザー無効化直後でも、ID トークンの有効期限内（最長 1 時間）は read エンドポイントの認証を通る可能性があります。ただし DB 上で退会済みのユーザーは Firebase UID からユーザーを解決できないため拒否されます。

このリスクを限定するため、以下の方針で適用範囲を制限しています。

- 書き込み操作、管理操作、権限（EDITOR / ADMIN）が必要な操作には read 最適化経路を使わない。
- API トークン発行・削除、退会、プレイヤーデータ登録などの重要操作は strict 経路を使う。
- recent sign-in が必要な処理では `VerifyRecentSignIn` を使い、失効確認と `auth_time` 検証を行う。

## 5. テスト

関連テストは以下にあります。

- `internal/infra/firebaseauth/token_verifier_test.go`
  - strict 検証、recent sign-in 検証、失効確認なし検証のエラー変換と UID 返却を確認する。
- `internal/usecase/read_optimized_token_verifier_test.go`
  - read 最適化対応 verifier のラップ、非対応 verifier のフォールバック、エラー伝播を確認する。
- `internal/usecase/firebase_auth_usecase_test.go`
  - ID トークン検証後の Firebase UID によるユーザー解決、任意認証時の未登録ユーザー扱い、エラー変換を確認する。
- `internal/app/middleware/firebase_auth_middleware_test.go`
  - 必須認証と任意認証の HTTP ミドルウェア挙動を確認する。

