# マスタデータ再読込機能 実装計画書

## 1. 目的

本計画書は、マスタデータ再読込機能を本番運用可能な品質で実装するための作業項目を定義する。
主目的は以下の3点である。

- 並行アクセス時の安全性を保証し、panic（concurrent map iteration and map write）を防止する。
- マスタ再読込後、全てのUsecase/Handlerが最新マスタを参照する一貫性を保証する。
- 動的マスタ（MySQL）と静的マスタ（SQLite）を同時更新し、不整合を防止する。

## 2. 対象範囲

- 対象レイヤー
  - Usecase層（マスタ参照境界の定義）
  - Infra層（RuntimeCache実装・Loader連携）
  - App層（DI構成、内部管理API）
  - 起動処理（main）
- 対象外
  - キャッシュ更新契機の自動化（ファイル監視・定期ジョブ）は本計画の対象外
  - マスタ構造自体の仕様変更（カラム追加等）は本計画の対象外

## 3. 設計方針

### 3.1 依存関係方針（Clean Architecture準拠）

- Usecase層は `internal/infra` の具体実装に依存しない。
- Usecase層に `MasterDataProvider` インターフェースを定義し、Infra層が実装する。
- Handler層にビジネスロジックを置かず、再読込判断・処理はUsecase層へ集約する。

### 3.2 キャッシュ更新方針

- 再読込は RuntimeCache を唯一の更新窓口とする。
- RuntimeCache内で動的マスタと静的マスタを同一クリティカルセクションでスワップする。
- 読み取り側は Provider 経由で毎回スナップショットを取得する。

### 3.3 一貫性方針

- `*masterdata.Cache` / `*masterdata.StaticCache` を各Usecaseへ直接注入しない。
- 再読込失敗時は旧スナップショットを保持し、利用者に破壊的影響を与えない。

## 4. 実装タスク

## 4.1 Usecase境界の追加

1. `internal/usecase` に `MasterDataProvider` インターフェースを追加する。
2. 既存Usecaseコンストラクタの引数を `*Cache` 直接依存から `MasterDataProvider` 依存へ変更する。
3. テストダブル（モック/スタブ）を新インターフェースで差し替える。

成果物:
- Usecase層の境界定義
- コンストラクタ更新
- 既存テスト修正

## 4.2 RuntimeCache適合

1. `internal/infra/masterdata/runtime_cache.go` を `MasterDataProvider` の実装として整備する。
2. 読み取りAPIで外部へ内部可変状態が露出しないことを確認する。
3. `Reload` 失敗時の不変性（旧値維持）をテストで保証する。

成果物:
- Provider実装
- 並行安全性テスト
- 失敗時の不変性テスト

## 4.3 DI配線の更新

1. `main.go` で RuntimeCache を生成し、DIに注入する。
2. `internal/app/router.go` の引数・初期化処理を Provider 注入へ変更する。
3. `masterCache` 直接注入箇所を全て置換する。

成果物:
- mainの初期化更新
- routerのDI更新
- 直接参照経路の排除

## 4.4 再読込Usecaseと内部API追加

1. `ReloadMasterDataUsecase` を追加し、再読込責務をUsecase層に実装する。
2. `POST /internal/master/reload` を追加する。
3. 認可は `ADMIN` 権限を必須とする。
4. 実行成功/失敗のレスポンス仕様を定義し、共通エラーハンドラへ接続する。

成果物:
- 再読込Usecase
- 内部管理API
- ルーティング更新

## 4.5 ドキュメント更新

1. API追加に伴い `API.md`（または同等のAPI仕様書）へ反映する。
2. 運用手順（実行者権限、失敗時対応）を `docs/` または `_report/` に記述する。

成果物:
- API仕様更新
- 運用メモ

## 5. テスト計画（TDD）

## 5.1 先に作成するテスト

- RuntimeCache並行安全性
  - 複数goroutineで参照中に `Reload` を繰り返してもpanicしない。
- 再読込反映性
  - 旧データ参照後、`Reload` 実行で新データがUsecase出力に反映される。
- 原子性
  - `Reload` 失敗時に旧スナップショットが維持される。
- 権限
  - `POST /internal/master/reload` はADMINのみ実行可能。
- 互換性
  - 既存マスタ参照APIが仕様通りのレスポンスを維持する。

## 5.2 実行コマンド

- `go test ./...`
- `gofmt -w <変更ファイル>`

## 6. 受け入れ基準

以下を全て満たした場合に完了とする。

1. `go test ./...` が成功する。
2. 再読込前後で全Usecaseが最新マスタを参照する。
3. 並行参照 + 再読込でpanicが発生しない。
4. 動的/静的マスタが同時に更新され、不整合が再現しない。
5. `POST /internal/master/reload` がADMIN権限でのみ実行できる。
6. API仕様書が更新されている。

## 7. リスクと対策

- リスク: 既存Usecaseのコンストラクタ変更で呼び出し側が広範囲に影響を受ける。
  - 対策: コンストラクタ変更を先に行い、コンパイルエラー起点で順次置換する。
- リスク: テストデータ依存により並行テストが不安定化する。
  - 対策: テスト専用Loaderを用意し、I/O依存を排除する。
- リスク: 再読込APIの誤用で運用負荷が上がる。
  - 対策: 管理者限定、監査ログ、レート制限を導入する。

## 8. 実施順序

1. Usecase境界（Provider interface）追加
2. RuntimeCache実装適合とテスト追加
3. Router/mainのDI置換
4. ReloadUsecase追加
5. Reload API追加と認可適用
6. APIドキュメント更新
7. 全体テスト・整形・自己レビュー

