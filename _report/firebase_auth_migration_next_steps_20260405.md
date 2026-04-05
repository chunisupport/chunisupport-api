# Firebase認証移行の不足事項と今後の対応

作成日: 2026-04-05

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
- `internal/domain/entity/user.go`
- `internal/infra/models/user_model.go`

## 結論

Firebase認証への移行で、現状もっとも不足しているのは以下の3点です。

1. **アカウントライフサイクルの仕様が閉じていないこと**
2. **削除・連携変更などの破壊的操作に再認証がないこと**
3. **「Firebaseを使う」のに認証の正本がまだアプリ内セッション側に残っていること**

特に1と2は、今のままフロント導線だけ足しても危険です。  
先に「削除したら何が起きるか」「同じFirebaseアカウントで再登録できるか」「どの操作で再認証を要求するか」を固定しないと、UIだけ整えても仕様事故になります。

## 現状整理

- `POST /internal/auth/firebase/login`
- `POST /internal/auth/firebase/register`
- `POST /internal/me/firebase/link`

上記は実装済みです。一方で、Firebase認証成功後はそのままFirebaseセッションを使うのではなく、`SessionIssuer` でアプリ独自のDBセッション + JWT Cookieを発行しています。

- つまり現状は「Firebaseを本人確認の入口に使い、その後は従来のCookieセッションで動く」構成です。
- `FirebaseIDTokenMiddleware` は存在しますが、実ルートでは未採用です。
- `firebase_register_usecase.go` にも「今後 Firebase に認証を一任し DB セッションを廃止する予定」というコメントがあり、移行途中の状態です。

また、アカウント削除は `DELETE /internal/me` で論理削除していますが、現在のセッションを落とすだけで、Firebaseアカウント自体の扱いは定義されていません。

## いま絶対に不足していること

### 1. アカウント削除後の扱い

現状の最大の穴はここです。

- 自分のアカウントを削除しても、Firebase UID の扱いが仕様として閉じていません。
- API仕様上も実装上も、**削除済みユーザーに紐づいた Firebase UID は再利用できません**。
- そのため、ユーザーが「退会したので後日同じGoogle/Appleアカウントで作り直したい」と思っても、現状は素直に再登録できない可能性があります。

これはFirebase認証を採用する上で致命的です。  
「削除は本当に不可逆なのか」「一定期間内は復元なのか」「同一Firebase UIDで再登録可能にするのか」を先に決める必要があります。

### 2. 削除や連携変更に再認証がない

`DELETE /internal/me` はログイン済みCookieだけで実行できます。  
Firebaseを認証バックエンドとして扱うなら、削除や認証手段変更は**直前の本人再確認**が必要です。

しかも現状の内部APIはCookieベースなので、削除のような状態変更系操作はCSRF観点でも慎重に扱う必要があります。  
少なくとも「ログイン中なら削除できる」ではなく、「今この場で本人確認し直したので削除できる」に寄せるべきです。

最低限必要なのは以下です。

- アカウント削除時に最新の Firebase ID トークン提出を必須にする
- Firebase連携の変更時にも同様に再認証を要求する
- 可能なら Firebase の recent login 相当の考え方を採用する

今の構成だと、Firebaseで本人確認しているのに、最も危険な操作だけはFirebaseによる再確認を使っていません。

### 3. 認証方式の正本が曖昧

現在の認証方式は次の2層構造です。

- 入口: Firebase IDトークン
- 継続ログイン: アプリ独自セッション + JWT Cookie

この構成自体は暫定策としては成立しますが、長期運用の正本としては曖昧です。

- Firebaseを正本にしたいのか
- 現行のアプリ内セッションを正本として残すのか
- 内部APIだけFirebaseセッションCookieまたはBearer IDトークンへ寄せるのか

ここを決めないまま機能を積み増すと、削除、ログアウト、セッション失効、複数端末管理、トラブルシュートが全部中途半端になります。

## 仕様上の不整合

### 認証手段の管理画面に必要な概念が足りない

現状は Firebase の `link` はありますが、次がありません。

- Firebase の `unlink`
- 現在どの認証手段を持っているかの取得
- 「Firebaseのみ」「パスワードのみ」「両方」のどれかを示す状態管理
- 最後の認証手段を外してアカウントを詰ませないための制約

このため、設定画面を作ろうとしても、表示すべき状態と許可すべき操作がまだ足りません。

### Firebase専用ユーザーのパスワード導線が不整合

Firebase専用ユーザーは空の `password_hash` を許容しています。  
一方で導線は次のようにねじれています。

- `PUT /internal/me/password` は現在のパスワード入力が必要なので、Firebase専用ユーザーは事実上使えません
- しかし `POST /internal/me/recovery-codes` は発行できます
- さらに `POST /internal/auth/recovery-codes` で新しいパスワードを設定できるため、リカバリーコード経由ではパスワード認証を後付けできます

つまり、**「Firebase専用アカウントにパスワードを持たせるか」の仕様が明示されていないまま、経路によってできたりできなかったりする** 状態です。

## 今後どうすべきか

### 優先度P0: 先に決めるべき仕様

以下を実装前に決めるべきです。

- アカウント種別をどう扱うか
- Firebaseのみ / パスワードのみ / 併用を正式に許可するか
- 退会時に Firebase UID を再利用可能にするか
- 退会を「論理削除 + 復元前提」にするか、「完全退会」に寄せるか
- Firebaseアカウント自体の削除は本アプリの責務にするか、外部IdP側の責務にするか

この仕様が決まらない限り、削除導線も設定画面も確定できません。

### 優先度P1: 実装が必須のもの

#### 1. 削除APIを再認証必須にする

最低限、`DELETE /internal/me` は次のどちらかに変更すべきです。

- 最新の Firebase ID トークンをリクエストボディで受ける
- 退会確認専用エンドポイントを作り、そこで再認証を通したうえで削除する

推薦は前者です。Firebaseを使う意味が最も素直に出ます。

#### 2. 削除時の認証関連データ整理を明示する

削除時は少なくとも以下を一括で整理すべきです。

- 全ローカルセッション無効化
- APIトークン無効化
- リカバリーコード無効化
- Firebase UID を残すのか外すのかを仕様通りに処理

現状は自分の現在セッションだけを明示的に落とす形なので、「使えなくはなるが後片付けが弱い」寄りです。  
削除処理をライフサイクル処理としてまとめ直した方がよいです。

#### 3. Firebase連携管理APIを増やす

最低限ほしいのは以下です。

- `GET /internal/me/auth-methods`
- `DELETE /internal/me/firebase/link` または `POST /internal/me/firebase/unlink`
- 必要なら `POST /internal/me/password/setup`

特に unlink がないと、Firebase移行後の「連携し直したい」「別のFirebaseアカウントへ切り替えたい」に答えられません。

#### 4. 認証方式の最終方針を決めて実装を寄せる

選択肢は大きく2つです。

- 当面は現行方式を維持し、Firebaseはログイン入口だけに使う
- 内部APIを Firebase セッション Cookie または Bearer ID Token ベースへ寄せる

小規模アプリとしての現実解は、まず前者で仕様を閉じ、その後に後者へ進む形です。  
ただし、その場合でも「Firebaseを認証バックエンドとして使う」と言うなら、削除再認証だけは早めにFirebase寄せにした方がよいです。

### 優先度P2: フロント導線として必要なもの

設定画面には最低限、以下が必要です。

- 現在の認証手段一覧
- Firebase連携済みかどうか
- パスワード設定済みかどうか
- 退会時の注意文
- 退会後に何が残り、何が再利用できないかの説明
- 退会前の再認証UI

特に「退会すると同じFirebaseアカウントで再登録できるのか」は、UIで必ず明示すべきです。

## 推奨する実装順

1. アカウント種別と削除後ポリシーを仕様として確定する
2. `DELETE /internal/me` を再認証必須にする
3. 削除時の認証関連データ一括無効化を実装する
4. 認証手段一覧APIと Firebase unlink API を追加する
5. フロント設定画面に認証手段管理と退会導線を追加する
6. 最後に、内部API認証をFirebaseへ寄せるかを再判断する

## まとめ

今のFirebase移行は、ログイン入口の追加としては進んでいますが、**アカウント管理**としてはまだ未完成です。

特に不足しているのは以下です。

- 退会後のFirebase UIDの扱い
- 破壊的操作の再認証
- 認証手段管理のUI/API
- Firebase専用アカウントとパスワード導線の整合
- Firebaseとローカルセッションのどちらを正本にするかの決定

削除導線を強くしたいのであれば、最優先はUI改善ではなく、  
**「退会仕様の確定」→「再認証付き退会API」→「認証手段管理API」** の順で進めるべきです。
