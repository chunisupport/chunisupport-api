# Firebase Auth 完全移行計画書

作成日: 2026-04-12

## 目的

認証方式を Firebase Authentication へ一本化する。

今回のゴールは次の 3 点である。

- 内部 API のユーザー認証を Firebase ID トークン検証へ移行する
- `users.password_hash` を削除する
- `sessions` テーブルと独自セッション管理を削除する

後方互換性は考慮しない。  
既存の Cookie / JWT / パスワード認証を前提にしたクライアントは破壊的に切り替える。

## 前提とスコープ

### 本計画で「Firebase Auth に移譲するもの」

- `/internal` 配下のユーザー認証
- ユーザー本人確認
- ログイン状態の継続判定
- 退会時の再認証

### 本計画で維持するもの

- `/v1` および `/compat/chunirec/2.0` の API トークン認証

理由:

- 現在の API トークンは「外部 API 利用者向けのアプリ内トークン」であり、ブラウザのログインセッションとは役割が異なる
- ユーザーの対話的認証を Firebase に寄せることと、機械利用向け API トークンを残すことは両立する

したがって、本計画の「すべての認証が Firebase Auth に移譲される」は、内部 API のユーザー認証について達成するものと定義する。

## 現状整理

### 現在の認証方式

現状は単一方式ではなく、次の 3 系統が混在している。

| 用途 | 現状の方式 | 主な実装 |
| --- | --- | --- |
| 内部 API (`/internal`) | Cookie + 独自 JWT + `sessions` テーブル | `internal/app/middleware/auth_middleware.go`, `internal/usecase/auth_usecase_impl.go`, `internal/usecase/session_issuer.go` |
| Firebase ログイン入口 | Firebase ID トークンを受け取って独自セッションへ変換 | `internal/usecase/firebase_login_usecase.go`, `internal/usecase/firebase_register_usecase.go` |
| 外部 API (`/v1`, `/compat/chunirec/2.0`) | `Authorization: Bearer <api_token>` | `internal/app/middleware/api_token_middleware.go`, `internal/usecase/api_token_usecase_impl.go` |

### 現在の内部 API 認証フロー

1. `/internal/auth/login` または `/internal/auth/firebase/login` を呼ぶ
2. バックエンドが独自 JWT を発行する
3. 同時に `sessions` テーブルへセッションを保存する
4. 以後の `/internal` は `token` Cookie を読む
5. JWT 内の `session_id` と `sessions` テーブルを照合して本人確認する

つまり、Firebase は現状では認証基盤ではなく「ログイン入口のひとつ」に留まっている。

### 現在の DB 状態

`users` テーブル:

- `firebase_uid` は nullable かつ unique
- `password_hash` は NOT NULL
- Firebase 専用ユーザーは実装上 `password_hash == ""` を許容している
- 現在は全ユーザーの `firebase_uid` が埋まっている
- Google 連携していなかったユーザーについても、存在しないメールアドレスとパスワードで Firebase Auth 側ユーザーを作成済みである

`sessions` テーブル:

- 内部 API の継続認証の正本
- `auth_middleware.go` が Cookie 内 JWT と組み合わせて使用

`user_recovery_codes` テーブル:

- パスワード再設定用
- パスワード認証を残す前提の機能

### 現在の主要エンドポイント

内部認証関連:

- `POST /internal/auth/register`
- `POST /internal/auth/login`
- `POST /internal/auth/firebase/login`
- `POST /internal/auth/firebase/register`
- `POST /internal/auth/logout`
- `POST /internal/auth/recovery-codes`
- `GET /internal/me/sessions`
- `DELETE /internal/me/sessions`
- `PUT /internal/me/password`
- `POST /internal/me/firebase/link`

### コード上の確定事項

- `FirebaseIDTokenMiddleware` は既に存在し、`Authorization: Bearer <Firebase ID Token>` を読める
- しかし実ルートでは未採用
- `/internal` の本番導線は `JWTMiddleware` に依存している
- `session_issuer.go` は `sessions` テーブル保存と JWT 発行を同時に担っている
- `auth_handler.go` / `firebase_handler.go` はどちらも `token` Cookie を返す
- `profile_handler.go` の退会処理は、削除後に Cookie セッション失効を試みる
- `user_model.go` / `user_repository_impl.go` は `password_hash` カラム前提
- `recovery_usecase.go` は `PW_PEPPER` と `password_hash` に依存している

## 目標アーキテクチャ

### 認証の正本

内部 API の認証正本を Firebase ID トークンに置き換える。

新しい流れは次のとおり。

1. クライアントが Firebase SDK でログインする
2. クライアントが Firebase ID トークンを取得する
3. 内部 API 呼び出し時に `Authorization: Bearer <Firebase ID Token>` を付与する
4. バックエンドは Firebase Admin SDK でトークンを検証する
5. `firebase_uid` でアプリ内ユーザーを解決する

### 認証状態の持ち方

- バックエンドは独自 JWT を発行しない
- `sessions` テーブルは持たない
- `token` Cookie は廃止する
- セッション数管理 API も廃止する

### ユーザー識別子

アプリ内ユーザーの認証上の主キーは `users.firebase_uid` とする。

推奨:

- `firebase_uid` を `NOT NULL UNIQUE` にする
- Firebase 未連携ユーザーを残さない

この方針を採ると、全ユーザーが Firebase Auth を前提とする設計へ揃う。

## 破壊的変更の方針

### 廃止するもの

- `POST /internal/auth/register`
- `POST /internal/auth/login`
- `POST /internal/auth/firebase/login`
- `POST /internal/auth/logout`
- `POST /internal/auth/recovery-codes`
- `GET /internal/me/sessions`
- `DELETE /internal/me/sessions`
- `PUT /internal/me/password`
- `POST /internal/me/recovery-codes`
- `POST /internal/me/firebase/link`

補足:

- `POST /internal/auth/firebase/register` は現行の「Firebase ID トークンでユーザー作成して Cookie 発行」という役割を失う
- 完全移行後は「初回ログイン時のユーザー作成 API」へ再設計するか、自動プロビジョニングへ置き換える

### 新設または再設計するもの

- Firebase Bearer 認証前提の `/internal` 共通認証
- 初回ユーザー作成フロー
- 退会 API の再認証仕様

### 推奨する新しいオンボーディング方式

後方互換を捨てるなら、最も単純なのは次のどちらかである。

#### 案 A: 初回アクセス時自動プロビジョニング

- Firebase ID トークンが有効
- 対応する `firebase_uid` のユーザーが存在しない
- その場合は最小情報でユーザーを自動作成する
- その後に username 設定 API を踏ませる

#### 案 B: 明示的な初回登録 API

- `POST /internal/auth/signup`
- 認証は Bearer Firebase ID トークン必須
- リクエストで `username` を受け、`firebase_uid` でユーザーを作成する

本プロジェクトでは、既存に username 必須文化があるため、案 B を推奨する。  
ただし初回未登録ユーザーの取り扱いを明確にしたいなら、案 A も実装は可能である。

## 移行後 API の想定

### 認証

- `/internal` の認証必須エンドポイントは原則すべて `Authorization: Bearer <Firebase ID Token>` を必須にする
- Cookie 認証エンドポイントは削除する

### 残す候補

- `POST /internal/auth/signup`
  - Bearer Firebase ID トークン必須
  - 未作成の `firebase_uid` に対してローカルユーザーを作成する

### 退会

- `DELETE /internal/me`
  - Bearer Firebase ID トークン必須
  - Firebase 側で recent sign-in を満たした直後のトークンを要求する
  - アプリ DB のユーザー削除後、Firebase Auth 側ユーザー削除も試行する

### 不要になるエンドポイント

- パスワードログイン系
- Cookie ログアウト系
- セッション数管理系
- パスワード再設定 / リカバリーコード系
- Firebase 連携系

理由:

- すべてのアクティブユーザーが Firebase 前提なら、「連携」という概念そのものが不要になる

## DB 変更計画

### 最終形

`users` テーブル:

- `firebase_uid` は `NOT NULL UNIQUE`
- `password_hash` を削除

削除するテーブル:

- `sessions`
- `user_recovery_codes`

維持するテーブル:

- `api_tokens`

補足:

- 外部 API (`/v1`, `/compat/chunirec/2.0`) を維持するため、`api_tokens` は削除しない

### マイグレーション方針

#### Step 1. スキーマ変更

推奨順:

1. 既存 `sessions` を全削除する
2. 既存 `user_recovery_codes` を全削除する
3. `users.firebase_uid` を `NOT NULL` に変更する
4. `users.password_hash` を削除する
5. `sessions` テーブルを削除する
6. `user_recovery_codes` テーブルを削除する
7. `cleanup_expired_sessions` イベントを削除する

補足:

- `sessions` と `user_recovery_codes` は互換維持不要のため、事前に強制全削除で問題ない
- これらは移行対象データではなく、不要データとして扱う

#### Step 2. ドキュメントと ER 図更新

更新対象:

- `docs/API.md`
- `docs/configuration.md`
- `docs/er_diagram.puml`
- `migration/MIGRATION.md`
- `docs/recovery_code_spec.md`（廃止またはアーカイブ）

## アプリケーション変更計画

### 1. ルーター

`internal/app/router.go` で実施すること:

- `JWTMiddleware` / `OptionalJWTMiddleware` の適用をやめる
- 認証必須の `/internal` グループへ `FirebaseIDTokenMiddleware` を適用する
- Cookie 前提エンドポイントを削除する
- `AuthHandler` / `SessionHandler` / `RecoveryHandler` への依存を削減または除去する

### 2. ミドルウェア

残す:

- `firebase_auth_middleware.go`
- `bearer_token.go`
- `api_token_middleware.go`

削除対象:

- `auth_middleware.go`

### 3. ユースケース

削除対象:

- `auth_usecase_impl.go`
- `session_issuer.go`
- `session_usecase_impl.go`
- パスワード変更・リカバリーコード依存部分

再設計対象:

- `firebase_register_usecase.go`
  - Cookie セッション発行をやめる
  - 初回ユーザー作成専用へ変更する
- `firebase_auth_usecase.go`
  - `/internal` の標準認証として利用する
- `user_credential_usecase.go`
  - `sessionRepo` / `recoveryCodeRepo` / `pepper` 依存の整理

### 4. ハンドラー

削除対象:

- `auth_handler.go`
- `session_handler.go`
- パスワード変更 / リカバリーコード発行 / Cookie 失効処理

再設計対象:

- `firebase_handler.go`
  - ログイン Cookie 発行ではなく、必要最小限の signup 系 API へ縮小
- `profile_handler.go`
  - 退会時の Cookie 削除を除去
  - Firebase Bearer 認証前提へ簡素化

### 5. ドメインとリポジトリ

`entity.User`:

- `PasswordHash` を削除する
- `ChangePassword` を削除する
- `NewUser` / `NewFirebaseUser` を統合し、Firebase UID 必須の生成へ寄せる

`models.UserModel`:

- `PasswordHash` フィールドを削除する
- `ToEntity()` の空ハッシュ分岐を削除する

`user_repository_impl.go`:

- `SELECT` / `INSERT` / `UPDATE` から `password_hash` を除去する
- `firebase_uid` を前提にした CRUD へ寄せる

## 設定変更計画

削除候補:

- `JWT_SECRET`
- `PW_PEPPER`
- `auth.jwt_expiration_hour`
- `auth.session_expiration_hour`
- `auth.cookie_secure`
- `auth.cookie_same_site`

維持:

- `FIREBASE_CREDENTIALS_FILE`

確認事項:

- `PW_PEPPER` はパスワード関連実装を全廃するなら不要
- Cookie 設定も `token` Cookie 廃止と同時に不要

## 推奨フェーズ

### フェーズ 1: 現状確認と切替条件確定

実施内容:

- `firebase_uid IS NULL` が 0 件であることを最終確認する
- `sessions` / `user_recovery_codes` の削除前件数を把握する
- 完全移行日を決める

完了条件:

- 全ユーザーが Firebase 化済みであることが確認できている

### フェーズ 2: 内部 API を Firebase Bearer 認証へ切替

実施内容:

- `/internal` の認証ミドルウェアを `FirebaseIDTokenMiddleware` へ統一する
- フロントエンドの API 呼び出しを Bearer ID トークン前提へ変更する
- Cookie 発行 / Cookie 読み取りコードを削除する

完了条件:

- `/internal` の認証必須 API がすべて Firebase ID トークンで通る
- `token` Cookie を使うコードが消えている

### フェーズ 3: 認証系 API を整理

実施内容:

- パスワードログイン・リカバリーコード・セッション管理 API を削除する
- 必要なら `POST /internal/auth/signup` を追加する

完了条件:

- 独自ログイン導線が消えている
- 初回ユーザー作成導線が確定している

### フェーズ 4: ドメイン・DB の不要物を削除

実施内容:

- `PasswordHash` 関連コード削除
- `Session` 関連コード削除
- `RecoveryCode` 関連コード削除
- マイグレーション適用

完了条件:

- `users.password_hash` が消えている
- `sessions` テーブルが消えている
- `user_recovery_codes` テーブルが消えている

### フェーズ 5: ドキュメント更新

実施内容:

- API 仕様の全面更新
- 設定値の全面更新
- 廃止エンドポイント明記

完了条件:

- 実装と `docs/API.md` が一致している

## リスクと対策

### リスク 1: 退会時の recent sign-in 条件不足

影響:

- クライアントではログイン済みでも、Firebase 側ユーザー削除が拒否される

対策:

- クライアントで reauthenticate を必須にする
- 退会 API は Firebase Bearer 認証必須にし、必要なら recent sign-in 不足を専用エラーへ変換する

### リスク 2: ID トークン期限切れ時の UX 変化

影響:

- Cookie のような長期セッション前提の UI が崩れる

対策:

- クライアント側で Firebase SDK による ID トークン再取得を標準化する
- サーバー側は短命トークン前提でシンプルに扱う

## テスト観点

### 必須テスト

- Firebase ID トークン付きで `/internal/me` が通る
- Firebase ID トークンなしで `/internal/me` が 401 になる
- Cookie を送っても認証されない
- 未登録 `firebase_uid` の signup が成功する
- 既存 `firebase_uid` で重複 signup が拒否される
- 退会でアプリユーザー削除と Firebase 側削除試行が行われる
- `/v1` と `/compat/chunirec/2.0` は既存 API トークンで引き続き通る

### 削除対象テスト

- JWT Cookie 認証テスト
- セッション数テスト
- パスワードログインテスト
- リカバリーコードテスト

## 実施順の推奨

1. `firebase_uid IS NULL` が 0 件であることを確認する
2. `sessions` と `user_recovery_codes` の全削除を実施する
3. `/internal` の標準認証を Firebase Bearer に切り替える
4. フロントエンドを Firebase SDK + Bearer 送信へ切り替える
5. Cookie / JWT / session 系 API と実装を削除する
6. `password_hash` と recovery code 系実装を削除する
7. DB マイグレーションで `users.password_hash` / `sessions` / `user_recovery_codes` を削除する
8. ドキュメントを更新する

## まとめ

現状の Firebase 対応は「独自認証の入口に Firebase を追加した状態」であり、完全移譲ではない。  
完全移譲を達成するには、Firebase ログイン API を増やすのではなく、内部 API の認証正本そのものを Firebase ID トークンへ置き換える必要がある。

後方互換性を捨てる前提なら、目指すべき最終形は明確である。

- `/internal` は Bearer Firebase ID トークンで認証する
- 独自 JWT と Cookie を廃止する
- `sessions` テーブルを削除する
- `users.password_hash` を削除する
- リカバリーコード系も原則廃止する

この順で進めれば、認証の責務は Firebase 側へ寄り、アプリ側は `firebase_uid` に紐づくドメインデータ管理へ集中できる。
