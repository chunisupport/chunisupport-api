# 過剰実装の簡素化候補

## 目的

このレポートは、リポジトリ全体を確認した上で、DDD / クリーンアーキテクチャの意図を保ちながらも、
現時点では実装コストに対して効果が薄く、簡素化の余地がある箇所を優先順位順に整理したものです。

対象は「設計原則そのものの否定」ではなく、原則を守るために導入した抽象化や分割のうち、
現在は保守コストの方が大きくなっているものです。

## 結論

優先順位は次の通りです。

1. `legacyAuthService` 互換レイヤの撤去
2. `Executor` 抽象の整理
3. `UserUsecase` -> `PlayerUsecase` の依存整理
4. Usecase 層での `sql.ErrNoRows` 判定の削減

上から順に着手するのが妥当です。特に 1 と 2 は効果が大きく、設計上の見通しも改善しやすいです。

## 要約表

| 優先順位 | 候補 | 効果 | 影響範囲 | 着手しやすさ |
|---|---|---|---|---|
| 1 | `legacyAuthService` 撤去 | 高い | 小から中 | 高い |
| 2 | `Executor` 整理 | 高い | 大きい | 低い |
| 3 | `UserUsecase` -> `PlayerUsecase` 整理 | 中から高 | 中 | 中 |
| 4 | `sql.ErrNoRows` 判定削減 | 中 | 中 | 中 |

補足として、優先順位は「価値の大きさ」を基準にしています。
実際の実装順は「安全に進めやすいか」を加味して調整して構いません。

---

## 優先度1: `legacyAuthService` 互換レイヤの撤去

### 対象

- `internal/usecase/auth_service_compat.go`

### 現状

`legacyAuthService` は以下の Usecase を束ねるだけの互換ラッパーになっています。

- `AuthUsecase`
- `UserCredentialUsecase`
- `RecoveryUsecase`

各メソッドは単純委譲であり、独自の業務ルールや集約の調停は持っていません。
現在の主な利用箇所はテストで、アプリケーション本体の責務整理に対する寄与は限定的です。

### 過剰実装と判断した理由

- 追加レイヤがあるのに、責務の追加がない
- 依存関係を単純化するどころか、旧来の入口を維持するための構造だけが残っている
- テスト側もこの互換レイヤ前提になっており、新しい設計への移行を遅らせている

### 推奨方針

- `NewAuthService` の利用箇所を段階的に廃止する
- 本体コードとテストの双方で、必要な Usecase を直接組み立てる
- `legacyAuthService` を最終的に削除する

### 期待効果

- Auth 周りの構成が分かりやすくなる
- テストの依存関係が明示的になる
- 「過去互換のための façade」がなくなり、設計判断が単純になる

### 注意点

- テスト修正量はやや多い可能性がある
- 一括削除ではなく、参照箇所を潰してから最後に削除する方が安全

### 完了条件

- 本体コードから `NewAuthService` の参照がなくなっている
- テストも個別 Usecase の直接生成へ移行している
- `auth_service_compat.go` を削除できている

---

## 優先度2: `Executor` 抽象の整理

### 対象

- `internal/domain/repository/executor.go`
- `internal/usecase/transaction.go`
- `internal/infra/transaction/transaction_manager.go`
- `internal/domain/repository/*Repository`
- `internal/infra/repository/*`

### 現状

`Executor` は `*sqlx.DB` と `*sqlx.Tx` を共通化するための抽象ですが、
インターフェース自体が `*sqlx.Rows` や `*sqlx.Row` を返しており、完全には infra 依存を隠せていません。

結果として、次の中途半端な状態になっています。

- 依存は抽象化したつもりだが、実体は `sqlx` に強く依存している
- すべての Repository シグネチャが `exec Executor` を受け取るため、API が重くなっている
- Usecase から見たコード量と認知負荷が増えている

### 過剰実装と判断した理由

- 境界保護のための抽象化なのに、境界がきれいに切れていない
- その割に、全 Repository / Usecase に追加の引数と概念を持ち込んでいる
- 小規模アプリとしては、抽象の維持コストが相対的に高い

### 推奨方針

候補は2通りあります。

#### 案A

`Executor` を infra 側の概念として割り切り、domain 配下から外す。

- `Executor` を `internal/infra` 側へ移す
- domain/repository の純度を上げる
- 「cleanに見せるための抽象」ではなく「実務上必要な抽象」として扱う

#### 案B

より単純化し、通常系の Repository は `db` を内部保持、トランザクション利用時だけ別入口に寄せる。

- Repository 構造体は `*sqlx.DB` を持つ
- トランザクションが必要な処理だけ専用メソッドまたは Unit of Work 的な扱いに寄せる
- 全メソッドに `exec` を渡す形を減らす

現実的には、まず案Aで責務の位置を正し、その後必要なら案Bを検討するのが安全です。

### 期待効果

- レイヤ境界の説明が簡単になる
- Repository のシグネチャが読みやすくなる
- 「抽象のための抽象」が減る

### 注意点

- 影響範囲は広い
- 一気に全面変更すると差分が大きくなるため、段階的な移行が望ましい

### 完了条件

- `Executor` の責務の置き場所が整理されている
- domain 配下から `sqlx` 依存を追い出すか、少なくとも infra の責務として明確化できている
- Repository シグネチャの説明が今より簡潔になっている

---

## 優先度3: `UserUsecase` -> `PlayerUsecase` の依存整理

### 対象

- `internal/usecase/user_usecase_impl.go`
- `internal/usecase/player_usecase.go`
- `internal/usecase/player_usecase_impl.go`

### 現状

`UserUsecase` はプロフィール取得の内部で `PlayerUsecase.GetPlayerByID` を呼び、
その結果として `dto.PlayerDTO` を受け取っています。

つまり Usecase 層が別の Usecase 層に依存し、しかも DTO を受け渡しています。

### 過剰実装と判断した理由

- Usecase の再利用より、依存の連鎖が強くなっている
- 同一層で DTO を渡すため、アプリケーションサービス同士の境界が曖昧になる
- `UserUsecase` が本当に必要なのは Player の取得であり、`PlayerUsecase` という振る舞いの再利用ではない

### 推奨方針

- `UserUsecase` から `PlayerUsecase` 依存を外す
- 必要なデータ取得は `PlayerRepository` または専用取得ロジックで完結させる
- DTO 変換は最終的な返却地点でのみ行う

### 期待効果

- Usecase 間依存が減る
- DTO の責務が明確になる
- プロフィール取得系の処理が追いやすくなる

### 注意点

- `GetPlayerByID` に honor 取得まで含まれているため、必要なら Player 取得処理を少し整理する必要がある
- 既存テストのモック構造も変わる

### 完了条件

- `UserUsecase` から `PlayerUsecase` への依存が消えている
- Usecase 間で DTO を受け渡す箇所がなくなっている
- プロフィール取得系のテストが repository / entity ベースで読める形になっている

---

## 優先度4: Usecase 層での `sql.ErrNoRows` 判定の削減

### 対象

- `internal/usecase/auth_usecase_impl.go`
- `internal/usecase/user_usecase_impl.go`
- `internal/usecase/recovery_usecase.go`
- `internal/usecase/api_token_usecase_impl.go`
- `internal/usecase/goal_usecase_impl.go`
- その他 `sql.ErrNoRows` を直接判定している Usecase 実装

### 現状

Usecase 層で `errors.Is(err, sql.ErrNoRows)` を直接判定している箇所が複数あります。
これは repository 実装都合のエラーが Usecase に漏れている状態です。

### 過剰実装と判断した理由

- DDD / クリーンアーキテクチャを志向している割に、エラー境界が整理し切れていない
- Usecase がインフラ由来の知識を持つため、抽象化の恩恵が減る
- `repository.ErrUserNotFound` などに寄せた方が責務が明快

### 推奨方針

- Repository 実装で `sql.ErrNoRows` を domain / repository エラーへ変換する
- Usecase 層は repository 定義のエラーだけを扱う
- テストも `sql.ErrNoRows` 前提から repository エラー前提へ寄せる

### 期待効果

- エラー責務が整理される
- Usecase 実装が読みやすくなる
- DB 実装の差し替え耐性が少し上がる

### 注意点

- 1件ごとの改善効果は限定的
- ただし他の簡素化と相性がよく、継続的に効く

### 完了条件

- Usecase 実装から `sql.ErrNoRows` の直接判定がなくなっている
- Repository が not found 系エラーを自前のエラーへ変換している
- テストも repository エラー前提で書かれている

---

## 着手順の提案

### フェーズ1

まず `legacyAuthService` を撤去します。
これは局所的で、効果に対してリスクが低いです。

### フェーズ2

次に `UserUsecase` -> `PlayerUsecase` 依存を外します。
Usecase 層の構造が素直になり、プロフィール系の可読性が上がります。

### フェーズ3

その後 `sql.ErrNoRows` の扱いを Repository 側へ寄せます。
これは設計の整流化として意味があります。

### フェーズ4

最後に `Executor` の整理に着手します。
影響範囲が最も広いため、先に周辺の小さな整理を終えてから着手する方が安全です。

### 優先順位と着手順が異なる理由

`Executor` の整理は価値自体は高いものの、横断的な変更になりやすく、差分も大きくなります。
そのため「重要度は高いが、実装は後ろに回す」という扱いにしています。

---

## 補足

今回の調査では、Entity / Value Object / Repository / Usecase という大枠の分割自体は過剰とは判断していません。
問題なのは分割そのものではなく、次のような箇所です。

- 役割のない互換レイヤ
- 境界を守り切れていない抽象
- 同一層での依存の連鎖

したがって、方針としては「DDD をやめる」ではなく、
「今の構造の中で、価値の薄い抽象だけを削る」のが適切です。

## 今回は簡素化対象にしないもの

次の要素は、現時点では過剰実装とは判断していません。

- `domain/entity`
  - ドメインの振る舞いを保持しており、貧血モデル化を防ぐ役割があるため
- `domain/vo`
  - バリデーション境界として実益があるため
- `infra/models`
  - domain の純粋性を守るために必要な分離であり、責務が明確なため
- `dto`
  - API ごとの差分吸収に実際に使われており、境界の役割があるため

このレポートの意図は、設計レイヤを潰して単純化することではなく、
役割の薄いレイヤや不完全な抽象を減らすことにあります。
