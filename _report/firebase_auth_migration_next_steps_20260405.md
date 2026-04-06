# Firebase認証移行計画書

作成日: 2026-04-05

## 目的

Firebase 認証の導入を「ログイン入口の追加」で止めず、退会・認証手段管理・パスワード導線まで含めて、安全に運用できる状態まで段階的に進める。

この文書は次の役割を持つ。

- 現状実装の事実を固定する
- 先に決めるべき仕様を明文化する
- 実装フェーズと依存関係を整理する
- フロントエンドが安全に追従できる API 契約の土台にする

## 対象

- `docs/API.md`
- `internal/app/router.go`
- `internal/app/handler/api_internal/firebase_handler.go`
- `internal/app/handler/api_internal/profile_handler.go`
- `internal/app/middleware/firebase_auth_middleware.go`
- `internal/usecase/firebase_auth_usecase.go`
- `internal/usecase/firebase_login_usecase.go`
- `internal/usecase/firebase_register_usecase.go`
- `internal/usecase/firebase_link_usecase.go`
- `internal/usecase/user_credential_usecase.go`
- `internal/usecase/recovery_usecase.go`
- `internal/usecase/session_issuer.go`
- `internal/usecase/auth_usecase_impl.go`
- `internal/domain/entity/user.go`
- `internal/domain/repository/user_repository.go`
- `internal/domain/repository/session_repository.go`
- `internal/domain/repository/recovery_code_repository.go`
- `internal/domain/repository/api_token_repository.go`
- `internal/infra/models/user_model.go`
- `internal/infra/repository/user_repository_impl.go`

## 現状の確定事項

### 実装済み

- `POST /internal/auth/firebase/login`
- `POST /internal/auth/firebase/register`
- `POST /internal/me/firebase/link`

### 現在の認証構成

- 入口は Firebase ID トークン
- 継続認証はアプリ独自セッション + JWT Cookie

つまり、現状は Firebase を本人確認に使い、その後は従来の Cookie セッションで動作する構成である。

### コード確認で確定した補足

- `POST /internal/me/firebase/link` は無認証ではなく、Firebase ID トークン必須
- ただし「センシティブ操作向けの再認証」としては扱っていない
- `FirebaseIDTokenMiddleware` は存在するが、実ルートでは未採用
- `DELETE /internal/me` は `users` レコードを物理削除する
- ユーザー削除に伴い `players` / `player_records` / `player_worldsend_records` / `player_honors` / `sessions` / `api_tokens` / `user_recovery_codes` は外部キー制約により即時削除される
- Firebase UID が連携されている場合は Firebase ユーザー削除も試行するが、失敗時はサーバーログに記録し、API レスポンスは成功を維持する
- したがって、少なくとも DB 上で「物理削除後も `firebase_uid` が残り再利用を妨げる」状態ではない
- Firebase 専用ユーザーは空の `password_hash` を許容する
- その一方で、`PUT /internal/me/password` は `current_password` 必須であり、Firebase 専用ユーザーは直接使えない
- `POST /internal/auth/recovery-codes` では新しいパスワードを設定できるため、リカバリーコード経由ではパスワード認証を後付けできる

## この計画で解決すべき問題

### 1. アカウントライフサイクルが閉じていない

不足している論点:

- 退会後に同じ Firebase アカウントで再登録・再連携を許可するか
- 退会を物理削除のまま運用し続けるか
- Firebase アカウント自体の削除失敗を本アプリでどう扱うか

この仕様が未確定のままでは、退会 API も認証手段管理 UI も確定できない。

### 2. 破壊的操作の再認証が未整備

不足している論点:

- 退会時に Firebase の再認証を要求していない
- 認証手段変更時の再認証ポリシーがない
- recent sign-in 相当の考え方を API として扱っていない

Firebase を認証基盤として使うなら、削除や unlink のような操作は「ログイン中なら実行可能」ではなく、「直前に本人確認し直したので実行可能」に寄せる必要がある。

### 3. 認証手段モデルが API になっていない

現状コードから推測できる状態:

- `firebase_uid != nil` なら Firebase 連携あり
- `password_hash != ""` ならパスワードあり

ただし、これを「正式な認証手段一覧」として返す API は存在しない。  
このため、フロントエンドは状態表示も制約判定もできない。

### 4. 認証の正本が曖昧

現状は次の二層構造である。

- 入口: Firebase ID トークン
- 継続利用: 独自セッション + JWT Cookie

この構成自体は当面の暫定策として成立する。  
ただし、削除・ログアウト・セッション失効・複数端末管理を中途半端にしないため、当面の正本を明文化する必要がある。

## 本計画で採る前提

この計画では、移行を詰まらせないために次を推奨前提とする。

### 推奨前提 1

当面は「Firebase は入口、継続認証は独自セッション」の構成を維持する。

理由:

- 既存フロントの `fetchWithAuth` / Cookie 前提を崩さず進められる
- 退会再認証・認証手段管理・初回パスワード設定を先に片付けられる
- 正本の全面移行を後段へ分離できる

### 推奨前提 2

unlink はまず「アプリ内の Firebase UID 紐付け解除」を指す。

注意:

- Firebase provider unlink と同義ではない
- このフェーズでは DB 側の `firebase_uid` を外す API を扱う
- Firebase Auth 側 provider unlink は別途検討対象とする

### 推奨前提 3

退会は当面「物理削除」で継続する。

推奨は次を前提として固定する。

- DB 上のユーザーおよび関連データは物理削除する
- 同じ Firebase アカウントでの再登録・再連携可否を API と UI に明示する
- Firebase Auth 側ユーザー削除は試行し、失敗時の再試行・監視方針を決める

現状実装はこの前提に概ね沿っている。  
追加で詰めるべきは、Firebase Auth 側削除失敗時の運用と、再登録導線の扱いである。

## 着手前に固定すべき仕様

### P0-1. 認証手段の正式モデル

決めること:

- Firebase のみを正式に許可するか
- パスワードのみを正式に許可するか
- 併用を正式に許可するか
- 最後の認証手段を外せない制約をどう置くか

推奨:

- `firebase_only`
- `password_only`
- `hybrid`

の 3 状態を API で返せる形にする。

### P0-2. 退会ポリシー

決めること:

- 退会後に同じ Firebase アカウントで再登録・再連携を可能にするか
- リカバリーコードを残すか削除するか
- API トークンを残すか削除するか
- 全セッションを即時失効するか
- Firebase Auth 側ユーザー削除失敗をどう扱うか

推奨:

- 全セッション即時失効
- リカバリーコード即時削除
- API トークン即時削除
- Firebase Auth 側ユーザー削除は試行し、失敗時はログ記録に加えて再試行や監視方針を定義する

### P0-3. 再認証の適用範囲

決めること:

- 退会に必須とするか
- unlink に必須とするか
- password setup に必須とするか

推奨:

- 退会: 必須
- Firebase unlink: 必須
- password setup: 原則不要、必要なら別途仕様化

## 実装フェーズ

## フェーズ 1: 仕様確定と API 契約整理

### ゴール

バックエンドとフロントエンドが同じ前提で着手できる状態にする。

### 成果物

- 認証手段モデルの定義
- 退会ポリシーの定義
- 再認証が必要な操作一覧
- API レスポンス契約の定義
- `docs/API.md` 更新方針の確定

### 完了条件

- 退会後の Firebase UID 扱いが文書化されている
- `auth-methods` レスポンスの項目が決まっている
- unlink の意味が「アプリ内連携解除」として固定されている

## フェーズ 2: 退会処理の安全化

### ゴール

`DELETE /internal/me` を Firebase 再認証前提の安全な削除導線へ変える。

### 必須タスク

- `DELETE /internal/me` のリクエスト仕様を拡張する
- 最新の Firebase ID トークンを受ける
- トークン検証と削除対象ユーザー整合を確認する
- 削除処理をライフサイクル処理としてまとめる

### このフェーズで整理するデータ

- 現在セッションの削除
- 他セッションの即時失効
- リカバリーコード削除
- API トークン削除
- Firebase UID を残すか外すかの処理

### API 形の推奨

候補:

- `DELETE /internal/me` に JSON body で `id_token` を渡す
- もしくは `POST /internal/me/delete` のような専用操作 API を新設する

推奨:

- 既存の `DELETE /internal/me` を維持しつつ `id_token` を受ける形

理由:

- フロントの導線がシンプル
- Firebase 再認証を API に素直に反映できる

### 完了条件

- Cookie だけでは退会できない
- 再認証なしの退会が拒否される
- 削除後に全セッション・API トークン・リカバリーコードが意図どおり整理される

## フェーズ 3: 認証手段管理 API の追加

### ゴール

フロントが認証手段管理 UI を実装できるだけの API を揃える。

### 必須 API

#### 1. `GET /internal/me/auth-methods`

最低限返したい情報:

- Firebase 連携あり / なし
- パスワード設定あり / なし
- 認証手段種別
- Firebase unlink 可否
- password setup 可否

#### 2. `DELETE /internal/me/firebase/link` または `POST /internal/me/firebase/unlink`

挙動:

- アプリ DB の `firebase_uid` を解除する
- 最後の認証手段を失う場合は拒否する
- 必要なら再認証を要求する

#### 3. `POST /internal/me/password/setup`

対象:

- Firebase 専用ユーザー

挙動:

- `current_password` なしでパスワードを初回設定できる
- 既にパスワードがある場合の扱いを明示する

### 完了条件

- フロントが現在状態と許可操作を API だけで判断できる
- Firebase 専用ユーザーの初回パスワード設定が既存 `PUT /internal/me/password` と分離される

## フェーズ 4: ドキュメントとエラー契約の更新

### ゴール

API 利用者が誤解なく実装できる状態にする。

### 必須更新

- `docs/API.md`
- エラーコード一覧
- 退会 API の再認証要件
- auth-methods のレスポンス例
- unlink の制約
- password setup のレスポンス例

### 追加したいエラー例

- `firebase_uid_already_linked`
- `last_auth_method_cannot_be_removed`
- `recent_login_required`
- `password_already_configured`

エラー名は例であり、既存命名規則に合わせて調整してよい。

## フェーズ 5: 認証の正本見直しを再判断

### 位置づけ

このフェーズは必須ではない。  
ただし、Firebase を将来的な正本に寄せたい場合は、ここで別計画として切り出す。

### 検討対象

- 内部 API を Firebase セッション Cookie または Bearer ID トークンへ寄せるか
- `FirebaseIDTokenMiddleware` を実ルート採用するか
- フロントの `fetchWithAuth` 前提を崩すか

### この計画でやらないこと

- 現時点で全内部 API を Firebase 認証に置き換えること
- 現行の Cookie 認証をこの計画の途中で廃止すること

## API 設計メモ

### `GET /internal/me/auth-methods` のレスポンス案

```json
{
  "primary_mode": "hybrid",
  "firebase": {
    "linked": true,
    "provider": "google"
  },
  "password": {
    "configured": true
  },
  "actions": {
    "can_unlink_firebase": true,
    "can_setup_password": false,
    "requires_reauth_for_unlink": true
  }
}
```

`provider` は将来拡張用で、当面は `google` 固定でもよい。  
不要なら最初の段階では省いてもよい。

### `DELETE /internal/me` のリクエスト案

```json
{
  "id_token": "<Firebase ID Token>"
}
```

### `POST /internal/me/password/setup` のリクエスト案

```json
{
  "new_password": "new-password"
}
```

## テスト観点

### バックエンド単体テストで必須

- 物理削除後ユーザーの Firebase login / register 挙動確認
- 再認証なし退会の拒否
- 再認証あり退会の成功
- 退会時の他セッション削除
- 退会時のリカバリーコード削除
- 退会時の API トークン削除
- 最後の認証手段 unlink 拒否
- Firebase 専用ユーザーの password setup 成功
- パスワード既設定ユーザーの password setup 拒否

### 結合テストで必須

- Firebase login -> auth-methods 取得
- Firebase register -> password setup -> password login
- hybrid ユーザーで unlink
- 退会後の再ログイン挙動

## フロントエンドへの依頼事項

バックエンド側が先に確定すべきもの:

- auth-methods のレスポンス形
- 退会 API の再認証仕様
- unlink の制約
- password setup の可否ルール
- 退会後に同じ Firebase UID で再登録可能か

これが決まれば、フロントは認証手段管理 UI、退会 UI、初回パスワード設定 UI を安全に実装できる。

## 推奨実装順

1. 認証手段モデルと退会ポリシーを固定する
2. `DELETE /internal/me` を再認証必須にする
3. 退会時のセッション / API トークン / リカバリーコード整理を入れる
4. `GET /internal/me/auth-methods` を追加する
5. Firebase unlink API を追加する
6. Firebase 専用ユーザー向け password setup API を追加する
7. `docs/API.md` とエラー契約を更新する
8. 最後に認証の正本見直しを別判断する

## まとめ

今の Firebase 導入は、ログイン入口としては成立している。  
未完成なのは「アカウント管理」と「退会ライフサイクル」である。

この計画では、まず次を片付ける。

- 退会ポリシー確定
- 退会時再認証
- 認証手段一覧 API
- Firebase unlink API
- Firebase 専用ユーザー向け password setup API

当面は Cookie セッション構成を維持し、Firebase 正本化は後段判断に分離する。  
この順番で進めるのが、現状コードベースでは最も安全で詰まりにくい。
