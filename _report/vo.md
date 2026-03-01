# 値オブジェクト移行計画書

## 1. 背景と目的

本計画は、DDD の観点で「エンティティとして扱っているが、実質的には値オブジェクト（以下 VO）として扱うべき型」を段階的に是正し、以下を達成することを目的とする。

- ドメインモデルの純度向上（同一性が不要な概念を VO 化）
- 文字列比較や ID 依存の分岐を型安全な振る舞いへ集約
- Clean Architecture に沿った責務分離（Entity / VO / DTO / QueryModel）
- 将来機能追加時の変更容易性向上

## 2. 対象範囲

### 2.1 VO 化の優先対象（第1群）

- `ChartDifficulty`
- `ClearLampType`
- `ComboLampType`
- `FullChainType`
- `Slot`

理由:

- 実装上は `ID + Name` が中心で、ライフサイクル管理の振る舞いが薄い
- 他エンティティの属性値として参照される性質が強い
- 例: `Slot.Name == "none"` のような文字列比較をドメイン知識として閉じ込めたい

### 2.2 VO 化の検討対象（第2群）

- `AccountType`
- `Genre`
- `HonorType`
- `ClassEmblem`
- `ClassEmblemBase`

理由:

- 現状の実装は第1群と同様に値的性質が強い
- 一方で将来、独立したライフサイクル管理や運用要件が入る可能性がある

### 2.3 Entity ではなく DTO / QueryModel に分離すべき対象

- `UserWithPlayer`
- `PlayerDataSong` / `PlayerDataChart` / `PlayerDataWorldsendChart`
- `SongChartStats` / `SingleChartStats` / `ChartStatsByRatingBand` 系

理由:

- 集約ルート・不変条件・同一性を持つドメイン概念ではなく、読み取りや搬送が主目的

## 3. 設計方針

### 3.1 VO 設計ルール

- 不変（Immutable）を前提にする
- 生成はコンストラクタ経由（`NewXxx`）
- バリデーション責務を型内に閉じ込める
- DB などの信頼できるソースから大量生成するケースでは、性能劣化を避けるため、
  バリデーションを省略可能なリポジトリ専用ファクトリ（例: `NewXxxFromRepository`）の導入を検討する
- 可能な限り振る舞い（例: `IsRankedTarget`）を型に持たせる
- 既存 VO（`username`, `playername`, `score`, `chartconstant`, `notes`）の実装様式に統一する

### 3.2 境界の整理

- `internal/domain/entity`: 集約ルートおよび実体として同一性が必要なもののみ
- `internal/domain/vo`: 値・分類・ルールを表す概念
- `internal/dto`: Usecase 境界をまたぐ搬送モデル
- `internal/domain/querymodel`（必要時のみ新設）: ドメイン知識を含む読み取り専用モデル

## 4. 段階的移行ロードマップ

### Phase 0: 事前整備（影響を最小化）

1. 依存関係マップを作成する（型ごとの参照箇所を一覧化）
2. 既存テストのベースラインを取得する（`go test ./...`）
3. 変更対象の合意（第1群から着手）

完了条件:

- 影響範囲一覧が揃っている
- 現行テストが安定してグリーン

## Phase 1: Slot の VO 化（最優先）

目的:

- `Slot.Name` の文字列比較を廃止し、意味的な API に置き換える

実施内容:

1. `internal/domain/vo/slot`（仮）を追加
2. `NewSlot` / `IsRankedTarget` / `Key` 等を定義
3. `PlayerRecord.IsRanked()` / `SlotKey()` で VO の振る舞いを利用
4. Infra 層で DB 値との相互変換を追加

テスト:

- テーブルテストで `none`, `best`, `new` 等の振る舞いを検証
- 既存 `PlayerRecord` 関連テストを回帰確認

完了条件:

- ドメイン層から `Slot.Name == "none"` が消える
- 既存 API 互換性を維持したままテストグリーン

## Phase 2: Difficulty / Lamp 系の VO 化

目的:

- 難易度・ランプの表現ゆれ防止
- 比較・判定ロジックを型で吸収

実施内容:

1. `ChartDifficulty`, `ClearLampType`, `ComboLampType`, `FullChainType` を順次 VO 化
2. 大文字運用ルール（BASIC, ADVANCED, EXPERT, MASTER, ULTIMA）を VO 内で保証
3. 変換コードを Infra 側に集約

テスト:

- 正常系: 妥当な値で生成可能
- 異常系: 不正値でエラー
- 互換性: API 入出力でのシリアライズ確認

完了条件:

- 比較処理が `string` 直接比較から VO メソッド利用に置換
- 難易度の大文字運用が型で保証される

## Phase 3: 第2群（Account/Genre/Honor/ClassEmblem）の再評価と適用

目的:

- 将来要件を踏まえた適切な分類（VO or Entity）を確定する

実施内容:

1. 運用要件（履歴管理・管理 UI・外部連携）を確認
2. 同一性を要しないもののみ VO 化
3. 同一性が必要なものは Entity 維持 + 意味をコメントで明示

完了条件:

- 各型の採用理由が設計ドキュメントに記録される

## Phase 4: DTO / QueryModel 分離

目的:

- `entity` パッケージの責務を純化する

実施内容:

1. `UserWithPlayer` などを DTO/QueryModel パッケージへ移設
2. Usecase 出力モデルとして責務を明示
3. 呼び出し側（handler/infra）の import を段階置換

完了条件:

- `entity` に読み取り専用モデルが残存しない
- 依存方向がより明確になる

## 5. 互換性とリスク管理

### 5.1 互換性ポリシー

- API の入力/出力仕様は原則維持
- DB スキーマ変更は原則不要（変換層で吸収）
- 既存 ID を即時撤廃せず、段階的に移行する

### 5.2 主なリスクと対策

1. **参照箇所の大量修正による回帰**
   - 対策: Phase を小さく刻み、1型ずつ移行
2. **Infra 変換漏れ**
   - 対策: `ToEntity` / `FromEntity` 相当の変換テストを追加
3. **難易度の大文字ルール逸脱**
   - 対策: VO コンストラクタで `strings.ToUpper` を適用し整合性を統一

## 6. テスト戦略（TDD 運用）

各 Phase で以下を徹底する。

1. **Red**: 先に失敗するテストを書く
2. **Green**: 最小実装で通す
3. **Refactor**: 責務分離・命名改善を行う

共通チェック:

- `go test ./...`
- `gofmt`（対象ファイル）

## 7. 受け入れ基準（Done 定義）

- 第1群の VO 化が完了し、既存要件との後方互換を維持
- 主要分岐がプリミティブ比較から VO の振る舞いへ移行
- `entity` / `vo` / `dto(querymodel)` の境界が説明可能
- テストが継続的にグリーン
- 本計画書に基づく進捗が `_report` に追記される

## 8. 実行順序（推奨）

1. Slot
2. Difficulty
3. Lamp（Clear/Combo/FullChain）
4. Account/Genre/Honor/ClassEmblem 再評価
5. DTO / QueryModel 分離

この順序により、ビジネスロジックへの影響が大きい部分から先に安全に改善できる。
