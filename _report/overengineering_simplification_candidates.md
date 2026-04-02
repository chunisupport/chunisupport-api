# 過剰実装の簡素化候補

## 目的

このレポートは、現時点のコードベースに残っている「価値に対して保守コストが高い抽象化や依存」を整理し、
簡素化の優先順位を明確にするためのものです。

解決済みの項目は含めず、未解決の候補だけに絞って再構成しています。

## 結論

現時点で優先して整理したい候補は次の1件です。

1. `Executor` 抽象の整理

Usecase 層での `sql.ErrNoRows` 直接判定は解消済みです。
`Executor` は価値のある見直し候補ですが、影響範囲が広く、先に周辺の責務整理を進めた方が差分を抑えられます。

## 要約表

| 優先順位 | 候補 | 現状の問題 | 効果 | 影響範囲 | 着手しやすさ |
|---|---|---|---|---|---|
| 1 | `Executor` 整理 | 抽象の割に `sqlx` 依存を隠し切れていない | 高い | 大きい | 低い |

---

## 解決済み: Usecase 層での `sql.ErrNoRows` 判定の削減

### 対象

- `internal/usecase/auth_usecase_impl.go`
- `internal/usecase/user_usecase_impl.go`
- `internal/usecase/recovery_usecase.go`
- `internal/usecase/api_token_usecase_impl.go`
- `internal/usecase/goal_usecase_impl.go`
- その他 `sql.ErrNoRows` を直接判定している Usecase 実装

### 現状

Repository 実装で not found 系エラーへ変換し、Usecase 層から `errors.Is(err, sql.ErrNoRows)` の直接判定を除去しました。
Usecase は `repository.ErrUserNotFound` などの repository エラーのみを扱う構成に統一しています。

### 過剰実装と判断した理由

- Usecase が infra 由来のエラー知識を持たなくなった
- Repository 境界で吸収すべき責務を明確化できた
- not found 判定の置き場所が統一された

### 推奨方針

- not found 系は Repository 実装で repository エラーへ変換済み
- Usecase は repository 層で定義されたエラーのみを扱う
- テストも repository エラー前提へ寄せた

### 期待効果

- エラー責務が整理された
- Usecase 実装が読みやすくなった
- DB 実装の詳細が上位層に漏れにくくなった

### 注意点

- `FindByUserID` のように「見つからない場合は `nil, nil`」を契約にしている箇所はそのまま維持する
- 今後も not found をエラーで返す Repository は repository エラーへ変換する

### 完了条件

- Usecase 実装から `sql.ErrNoRows` の直接判定がなくなっている
- Repository が not found 系エラーを自前のエラーへ変換している
- テストも repository エラー前提で統一されている

---

## 優先度1: `Executor` 抽象の整理

### 対象

- `internal/domain/repository/executor.go`
- `internal/usecase/transaction.go`
- `internal/infra/transaction/transaction_manager.go`
- `internal/domain/repository/*Repository`
- `internal/infra/repository/*`

### 現状

`Executor` は `*sqlx.DB` と `*sqlx.Tx` を共通化するための抽象ですが、
インターフェース自体が `*sqlx.Rows` や `*sqlx.Row` を返しており、domain 配下に置かれている割に `sqlx` 依存を露出しています。

さらに、Repository シグネチャ全体に `exec repository.Executor` が広く入り込んでいます。

### 過剰実装と判断した理由

- 境界を守るための抽象なのに、境界をきれいに切れていない
- その一方で、全体に追加引数と認知負荷を持ち込んでいる
- 小規模アプリとしては抽象維持コストが高い

### 推奨方針

候補は2段階で考えるのが現実的です。

#### 案A

`Executor` を infra 側の実装都合として位置付け直す。

- `Executor` を domain 配下から外す
- 「設計上きれいに見せる抽象」ではなく「トランザクション共有のための実務的な抽象」として扱う

#### 案B

通常系 Repository は `db` を内部保持し、トランザクションが必要な処理だけ別入口に寄せる。

- 全メソッドで `exec` を受け取る形を減らす
- トランザクションが必要な箇所だけ明示的に扱う

まずは案Aで責務の置き場所を正し、そのうえで本当に必要なら案Bを検討するのが安全です。

### 期待効果

- レイヤ境界の説明がしやすくなる
- Repository シグネチャが読みやすくなる
- 「抽象のための抽象」が減る

### 注意点

- 影響範囲が広い
- 一気に変更すると差分が大きくなりやすい
- 周辺の Usecase / error 整理を先に済ませた方が安全

### 完了条件

- `Executor` の責務の置き場所が整理されている
- domain 配下からの `sqlx` 依存露出が解消、または infra 責務として明確化されている
- Repository シグネチャの説明が今より簡潔になっている

---

## 着手順の提案

### フェーズ1

最後に `Executor` を整理します。
ここは横断的な変更なので、周辺責務を先に整えた現状から着手する方が安全です。

---

## 補足

今回の見直しでは、Entity / Value Object / Repository / Usecase という大枠の分割自体は過剰とは判断していません。
問題なのは、次のような「価値に対して重い」箇所です。

- 同一層での依存の連鎖
- 境界で吸収し切れていないエラー
- 抽象の意図と実装実態がずれているインターフェース

したがって方針としては、設計原則を崩すのではなく、価値の薄い抽象と不整合だけを削るのが適切です。

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
