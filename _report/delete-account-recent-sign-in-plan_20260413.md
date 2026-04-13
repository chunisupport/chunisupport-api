# 退会API recent sign-in 対応 実装計画書

## 1. 目的

本計画書は、`SEC-012` で指摘されている
`DELETE /internal/me` の再認証不足を解消するために、
**退会APIに対してのみ Firebase の recent sign-in を要求する**
実装方針を定義する。

対象:

- `DELETE /internal/me`
- 退会APIに必要な Firebase トークン検証拡張
- 退会導線に必要なフロントエンドの再認証対応
- 関連する API ドキュメント更新

非対象:

- 他APIへの recent sign-in 要求の横展開
- 認証基盤全体の全面リファクタリング
- Firebase 以外の認証方式追加
- 退会後の復旧機構や論理削除方式への変更

---

## 2. 背景

現状の退会APIは、Firebase Bearer トークンで認証済みであれば実行できる。

具体的には、`internal/app/handler/api_internal/profile_handler.go` の
`DeleteAccount` に recent sign-in の TODO が残っており、
有効な Bearer トークンのみで `DeleteOwnAccount` が呼ばれている。

この状態では、以下のリスクがある。

1. 盗難・流出した有効トークンだけでアカウント削除が可能
2. 共有端末や放置セッションでも、本人の再確認なしに退会できる
3. Firebase を認証基盤にしているにもかかわらず、破壊的操作に recent sign-in を使っていない

アカウント削除は不可逆な破壊的操作であるため、
通常のログイン状態確認だけでなく、
**直前に本人が再認証したこと**を保証する必要がある。

---

## 3. 現状整理

## 3.1 現在の認証フロー

- ルーターで Firebase Bearer トークン必須
- ミドルウェアで ID トークンを検証
- 検証後、`userEntity` をコンテキストへ設定
- `DeleteAccount` は `user.ID` を使って `DeleteOwnAccount` を呼ぶ

## 3.2 現状の不足

現在の `TokenVerifier` は UID を返すだけであり、
recent sign-in 判定に必要な `auth_time` を扱っていない。

そのため、以下の確認ができない。

- そのトークンが直近の再認証に基づくものか
- 再認証した Firebase ユーザーと、削除対象ユーザーが同一人物か

## 3.3 問題の本質

問題は「退会APIが認証されていないこと」ではない。
問題は「退会APIが通常認証だけで許可されており、
破壊的操作に必要な recent sign-in を要求していないこと」である。

したがって解決方針は、
**退会APIだけに recent sign-in を追加し、
他APIへの副作用を最小化すること**になる。

---

## 4. 採用方針

## 4.1 基本方針

以下の方針を採用する。

1. `DELETE /internal/me` にのみ recent sign-in を要求する
2. 通常の `Authorization: Bearer <login token>` は維持する
3. 退会専用の再認証トークンを追加で受け取る
4. バックエンドで `auth_time` と UID 一致を検証する
5. recent sign-in 検証は退会専用の責務として追加し、既存の通常認証処理は極力変更しない

## 4.2 採用理由

1. 他APIに影響を広げずに `SEC-012` を解消できる
2. 既存の Firebase 認証ミドルウェアを大きく壊さずに済む
3. 将来、他の高リスク操作にも同じ仕組みを再利用しやすい
4. recent sign-in の起点はフロントエンドであり、責務分離が明確になる

---

## 5. API 仕様方針

## 5.1 エンドポイント

- **Method**: `DELETE`
- **Path**: `/internal/me`
- **Auth**:
  - `Authorization: Bearer <通常のログイン用 Firebase ID トークン>`
  - `X-Reauth-Token: <再認証直後の Firebase ID トークン>`

## 5.2 採用する受け渡し方法

recent sign-in 用トークンは、JSON body ではなく **専用ヘッダ** で受け取る。

採用ヘッダ:

- `X-Reauth-Token`

理由:

1. `DELETE` に body を載せる前提を避けられる
2. 既存の Bearer 認証と役割を分けやすい
3. 退会API専用の追加認証情報であることが明確

## 5.3 recent sign-in の許容時間

初期値として **5分以内** を採用する。

理由:

1. 再認証直後の確認として十分に短い
2. 退会は日常操作ではなく、多少厳しめでも許容しやすい
3. 将来調整したい場合に `internal/info` へ切り出しやすい

---

## 6. バックエンド設計

## 6.1 追加する責務

退会用の recent sign-in 判定のため、通常の `TokenVerifier` とは別に、
退会専用の検証責務を追加する。

想定責務:

- 再認証トークンの Firebase 検証
- UID の取得
- `auth_time` の取得
- `auth_time` が許容時間内かの判定

## 6.2 推奨インターフェース

既存の `TokenVerifier` を直接広げるより、
退会用の専用 interface を追加する方が影響範囲が小さい。

例:

```go
type RecentSignInVerifier interface {
    VerifyRecentSignIn(ctx context.Context, idToken string) (*RecentSignInInfo, error)
}

type RecentSignInInfo struct {
    UID      string
    AuthTime time.Time
}
```

## 6.3 Usecase の変更方針

退会 Usecase は recent sign-in 用トークンを受け取る形へ変更する。

例:

```go
DeleteOwnAccount(ctx context.Context, userID int, reauthToken string) error
```

Usecase 内では以下を行う。

1. 対象ユーザーを取得する
2. そのユーザーの `FirebaseUID` を確認する
3. `reauthToken` を recent sign-in verifier で検証する
4. 取得した UID がユーザーの `FirebaseUID` と一致することを確認する
5. `auth_time` が許容時間内であることを確認する
6. 条件を満たした場合のみ削除処理を続行する

## 6.4 Handler の変更方針

`ProfileHandler.DeleteAccount` で `X-Reauth-Token` を受け取り、
Usecase に渡す。

Handler の責務は以下に留める。

- ヘッダ取得
- 必須チェック
- Usecase 呼び出し
- Usecase エラーの API エラー変換

`auth_time` の解釈や UID 一致判定は Handler に置かない。

## 6.5 Firebase 側の取得情報

Firebase Admin SDK の検証結果から、最低限以下を取り出す必要がある。

- `UID`
- `auth_time`

`auth_time` が取得できない場合は内部異常または不正トークンとして扱う。

## 6.6 エラー設計

少なくとも以下のエラーが必要になる。

- 再認証トークン未指定
- 再認証トークン不正
- recent sign-in 期限切れ
- 再認証 UID とユーザーの FirebaseUID 不一致
- ユーザーに FirebaseUID が紐付いていない

外部向けには、まずは以下の整理で十分である。

- 再認証不足・期限切れ・不正: `401 Unauthorized`
- 対象ユーザー不在: 既存どおり `404`
- その他: `500`

必要であれば `apierror` に
`recent_sign_in_required` 相当のコードを追加する。

---

## 7. フロントエンド設計

## 7.1 フロントエンド対応は必須

recent sign-in はバックエンド単独では成立しないため、
フロントエンドでの再認証処理が必須である。

## 7.2 必要な変更

1. 退会確認 UI を用意する
2. 退会直前に Firebase の再認証を実行する
3. 再認証直後の新しい ID トークンを取得する
4. `X-Reauth-Token` に設定して `DELETE /internal/me` を呼ぶ
5. 失敗時は再認証エラーと退会失敗を区別して表示する

## 7.3 認証方式ごとの考慮

Firebase のログイン方式に応じて再認証手段が変わる。

- メール/パスワード: パスワード再入力
- OAuth 系: popup または redirect による再認証

したがって、フロント側では「退会前に再認証が必要」という UX を明示し、
サインイン方式ごとに適切な再認証導線を実装する必要がある。

## 7.4 バックエンドとの責務境界

フロントエンドの責務:

- 本人に再認証を促す
- 再認証後トークンを退会APIへ送る

バックエンドの責務:

- トークンが recent sign-in 条件を満たすか検証する
- 対象ユーザーとの一致を確認する
- 条件成立時のみ削除を許可する

---

## 8. 他APIへの影響方針

## 8.1 今回は退会API限定

本対応では、recent sign-in を要求するのは `DELETE /internal/me` のみに限定する。

そのため、以下の API は直接の仕様変更対象にしない。

- `/internal/me` の取得
- `/internal/me/privacy`
- API token 関連
- goal 関連
- song / user / player data 関連

## 8.2 間接影響

共通コードには以下の変更が入る可能性がある。

- Firebase 検証 interface 追加
- Usecase エラー追加
- `apierror` 追加
- API ドキュメント更新

ただしこれは内部共通部品の拡張であり、
**他APIに recent sign-in を必須化するものではない。**

---

## 9. 推奨実装ステップ

## 9.1 Step 1: 仕様確定

決めるべき事項:

- `X-Reauth-Token` を正式採用する
- 許容時間を 5 分にする
- エラーコードを追加するか、既存 `unauthorized` に寄せるか決める

## 9.2 Step 2: Domain / Usecase 境界の追加

実施内容:

- recent sign-in 検証用 interface 追加
- recent sign-in 結果 DTO 追加
- Usecase エラー追加

## 9.3 Step 3: Infra 実装追加

実施内容:

- Firebase Admin SDK を使った recent sign-in verifier 実装
- `auth_time` の抽出と検証

## 9.4 Step 4: Handler / Router 反映

実施内容:

- `ProfileHandler.DeleteAccount` を更新
- `X-Reauth-Token` 必須化
- 既存ルートは維持

## 9.5 Step 5: テスト追加

実施内容:

- Handler テスト
- Usecase テスト
- verifier テスト

## 9.6 Step 6: フロント実装

実施内容:

- 退会確認モーダルまたは画面
- 再認証処理
- 再認証トークン付き退会API呼び出し
- エラー表示

## 9.7 Step 7: ドキュメント更新

実施内容:

- `docs/API.md`
- 必要に応じてフロントの実装メモや運用手順

---

## 10. テスト方針

## 10.1 バックエンドで追加すべき観点

- `X-Reauth-Token` 未指定で 401 になる
- 再認証トークンが不正なら 401 になる
- `auth_time` が古ければ 401 になる
- UID が一致しなければ 401 または 403 になる
- recent sign-in が有効なら退会成功する
- 退会後に Firebase ユーザー削除連携が従来どおり動く

## 10.2 フロントエンドで確認すべき観点

- 再認証成功後に退会APIを呼べる
- 再認証キャンセル時に退会が実行されない
- 再認証失敗時に適切なエラー表示になる
- 期限切れトークン時に再認証を再要求できる

---

## 11. 実装タスク分解

1. `DELETE /internal/me` の recent sign-in 仕様を確定する
2. recent sign-in 用の verifier interface と DTO を追加する
3. Firebase Admin SDK で `auth_time` を取得する verifier を実装する
4. `DeleteOwnAccount` を再認証トークン必須のシグネチャへ変更する
5. UID 一致と recent sign-in 時刻検証を Usecase に実装する
6. `ProfileHandler.DeleteAccount` で `X-Reauth-Token` を受け取る
7. API エラー変換を追加する
8. Handler / Usecase / Infra テストを追加する
9. `docs/API.md` に退会APIの新要件を追記する
10. フロントエンドの退会導線を再認証必須へ更新する

---

## 12. 未決事項

## 12.1 FirebaseUID 未連携ユーザーの扱い

退会APIは Firebase Bearer 認証前提であるため、
通常は FirebaseUID が存在する想定だが、
不整合データがある場合の扱いを明確にする必要がある。

候補:

1. 内部異常として `500`
2. 不正状態として `401`
3. ドメインエラーを追加して明示的に扱う

運用上は 3 が望ましい。

## 12.2 エラーコード名

候補:

- `recent_sign_in_required`
- `reauthentication_required`
- `invalid_reauth_token`

クライアント実装を簡潔にするには、
まずは `recent_sign_in_required` に寄せるのが扱いやすい。

## 12.3 許容時間の設定化

初期実装では定数で十分だが、
運用調整の必要があるなら `internal/info` へ移す余地がある。

---

## 13. 結論

`SEC-012` の本質は、退会APIが通常のログイン状態だけで実行でき、
破壊的操作に必要な recent sign-in を要求していないことにある。

最も筋の良い対応は、以下である。

- 退会APIにのみ recent sign-in を要求する
- フロントエンドで再認証を実施し、再認証直後のトークンを送る
- バックエンドで `auth_time` と UID 一致を検証する
- 他APIへの仕様変更は広げない

この方針であれば、`SEC-012` を的確に解消しつつ、
既存API群への副作用を最小限に抑えられる。
