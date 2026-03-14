# DOM-014 修正計画: `PlayerDataChart.Const` の VO 化

## 課題概要

`PlayerDataChart.Const` が `float64` で定義されており、通常の `Chart` エンティティが使用する `chartconstant.ChartConstant` VOと不整合。VOによるバリデーション（0以上チェック）が適用されない。

## 現状分析

### 型の不整合

| エンティティ | フィールド定義 | バリデーション |
|---|---|---|
| `Chart` | `Const chartconstant.ChartConstant` | `NewChartConstant` で 0 以上をチェック |
| `PlayerDataChart` | `Const float64` | なし |

### 影響範囲の調査結果

#### エンティティ定義
- `internal/domain/entity/player_data_master.go` L29: `Const float64`

#### インフラモデル（変換含む）
- `internal/infra/models/player_data_master_model.go` L63: `PlayerDataChartModel.Const float64`（`db:"const"`タグ付き）
- 同 L74: `ToEntity()` で `m.Const` をそのままエンティティに渡す
- 同 L84: `FromPlayerDataChartEntity()` で `e.Const` をそのままモデルに渡す

#### ユースケース層（参照箇所）
- `internal/usecase/player_data_usecase_impl.go` L135-136: `map[string]entity.PlayerDataChart` / `map[int]entity.PlayerDataChart`（キャッシュ）
- 同 L307-308: キャッシュの `make` 呼び出し
- 同 L710: `resolveChart()` の戻り値型
- 同 L718, 724, 730: ゼロ値 `entity.PlayerDataChart{}` の生成

#### `PlayerDataChart.Const` の実際の読み取り箇所
- **使用箇所なし**: `resolveChart()` で取得された `PlayerDataChart` の `Const` フィールドは、その後の処理（`PlayerRecordForUpsert` 生成）では参照されない。`chart.ID` のみが使用される。
- `player_data_usecase_impl.go` L851 の `chartConst = float64(rec.Chart.Const)` は `PlayerRecord.Chart`（`entity.Chart` 型）の `Const`（`ChartConstant` 型）であり、`PlayerDataChart` ではない。

### 重要な発見

`PlayerDataChart.Const` は現状、**スコア登録フローで直接的に計算には使われていない**。`chart.ID` で紐づく `entity.Chart`（`ChartConstant` 型の `Const`）が最終的な計算に使われる。ただし、以下の理由から型安全性の修正は依然として必要：

1. **ドメインモデルの一貫性**: 同一概念の「譜面定数」が2種類の型で表現されるのは設計上の不整合
2. **将来のリスク**: `PlayerDataChart.Const` が直接参照されるコードが追加された場合、バリデーションなしの値が使用される
3. **DOM-007 との関連**: `ChartConstant.Scan` のバリデーションバイパス問題（DOM-007）と合わせて修正することで、DB→エンティティ変換パスの安全性が包括的に向上する

## 修正計画

### Step 1: エンティティの型変更

**対象ファイル**: `internal/domain/entity/player_data_master.go`

```go
// Before
type PlayerDataChart struct {
	ID             int
	SongID         int
	DifficultyID   int
	Const          float64
	IsConstUnknown bool
	Notes          *notes.Notes
}

// After
type PlayerDataChart struct {
	ID             int
	SongID         int
	DifficultyID   int
	Const          chartconstant.ChartConstant
	IsConstUnknown bool
	Notes          *notes.Notes
}
```

- `chartconstant` パッケージの import を追加

### Step 2: インフラモデルの変換処理を更新

**対象ファイル**: `internal/infra/models/player_data_master_model.go`

インフラモデル `PlayerDataChartModel.Const` は `float64`（`db:"const"` タグ）のまま維持する（DBカラムの型に合わせるため）。

`ToEntity()` で `float64` → `ChartConstant` への変換にバリデーションを適用する：

```go
// Before
func (m *PlayerDataChartModel) ToEntity() *entity.PlayerDataChart {
	return &entity.PlayerDataChart{
		...
		Const: m.Const,
		...
	}
}

// After
func (m *PlayerDataChartModel) ToEntity() (*entity.PlayerDataChart, error) {
	cc, err := chartconstant.NewChartConstant(m.Const)
	if err != nil {
		return nil, fmt.Errorf("invalid chart constant (chart_id=%d): %w", m.ID, err)
	}
	return &entity.PlayerDataChart{
		...
		Const: cc,
		...
	}, nil
}
```

`FromPlayerDataChartEntity()` で `ChartConstant` → `float64` への変換：

```go
// Before
func FromPlayerDataChartEntity(e *entity.PlayerDataChart) *PlayerDataChartModel {
	return &PlayerDataChartModel{
		...
		Const: e.Const,
		...
	}
}

// After
func FromPlayerDataChartEntity(e *entity.PlayerDataChart) *PlayerDataChartModel {
	return &PlayerDataChartModel{
		...
		Const: float64(e.Const),
		...
	}
}
```

### Step 3: `ToEntity()` 呼び出し元のエラーハンドリング対応

**対象ファイル**: `internal/infra/repository/player_data_repository_impl.go`

`ToEntity()` の戻り値が `(*entity.PlayerDataChart, error)` に変わるため、呼び出し元で `error` をハンドリングする必要がある。

```go
// Before
for _, model := range chartModels {
	chart := model.ToEntity()
	key := fmt.Sprintf("%d:%d", chart.SongID, chart.DifficultyID)
	result.ChartsByKey[key] = *chart
	result.ChartsByID[chart.ID] = *chart
}

// After
for _, model := range chartModels {
	chart, err := model.ToEntity()
	if err != nil {
		return nil, fmt.Errorf("failed to convert chart model to entity: %w", err)
	}
	key := fmt.Sprintf("%d:%d", chart.SongID, chart.DifficultyID)
	result.ChartsByKey[key] = *chart
	result.ChartsByID[chart.ID] = *chart
}
```

### Step 4: ユースケース層のゼロ値生成への対応

**対象ファイル**: `internal/usecase/player_data_usecase_impl.go`

`resolveChart()` 内の `entity.PlayerDataChart{}` ゼロ値生成は、エラー返却時の値なので問題ない（呼び出し元は `err != nil` で使用しない）。`ChartConstant` のゼロ値は `0.0` であり、`float64` のゼロ値と同等なので、**変更不要**。

### Step 5: テストの追加・更新

**対象ファイル**: `internal/infra/models/player_data_master_model_test.go`（新規作成）

`ToEntity()` のバリデーション動作をテスト：

| テストケース | 入力 | 期待結果 |
|---|---|---|
| 正常な譜面定数でエンティティが生成される | `Const: 13.5` | `err == nil`, `entity.Const == ChartConstant(13.5)` |
| 0の譜面定数でエンティティが生成される | `Const: 0.0` | `err == nil`, `entity.Const == ChartConstant(0.0)` |
| 負の譜面定数でエラーが返される | `Const: -1.0` | `err != nil` |

**対象ファイル**: `internal/infra/models/player_data_master_model.go` の `FromPlayerDataChartEntity()` テスト

| テストケース | 入力 | 期待結果 |
|---|---|---|
| ChartConstant から float64 に正しく変換される | `Const: ChartConstant(13.5)` | `model.Const == 13.5` |

### Step 6: refactor.md の更新

`refactor.md` からDOM-014を削除する。

## 変更ファイル一覧

| # | ファイル | 変更内容 |
|---|---|---|
| 1 | `internal/domain/entity/player_data_master.go` | `Const` の型を `float64` → `chartconstant.ChartConstant` に変更 |
| 2 | `internal/infra/models/player_data_master_model.go` | `ToEntity()` にバリデーション追加（戻り値にerror追加）、`FromPlayerDataChartEntity()` に `float64()` キャスト追加 |
| 3 | `internal/infra/repository/player_data_repository_impl.go` | `ToEntity()` のエラーハンドリング追加 |
| 4 | `internal/infra/models/player_data_master_model_test.go` | 新規テストファイル作成 |
| 5 | `_report/refactor.md` | DOM-014 の内容を削除 |

## 注意事項

- **インフラモデルの `Const` 型は変更しない**: `PlayerDataChartModel.Const` は `float64`（`db:"const"` タグ付き）のまま維持する。DBカラムの型と一致させる必要があるため。VOへの変換はモデル→エンティティの境界で行う。
- **`ChartConstant` に `Float64()` アクセサがない**: 現状 `float64(c.Const)` による明示的キャスト（`ChartConstant` の基底型は `float64`）で対応可能。INFRA-009 で `Float64()` アクセサを追加する場合はそちらに合わせて更新する。
- **DOM-007 との併行修正を推奨**: `ChartConstant.Scan` / `UnmarshalJSON` がバリデーションをバイパスする問題は、本修正と同時に対応することで整合性が向上する。ただし本修正は `ToEntity()` 内で `NewChartConstant()` を経由するため、DOM-007 未修正でも `PlayerDataChart` パスのバリデーションは担保される。
- **破壊的変更の範囲**: `ToEntity()` の戻り値シグネチャが変わるため、呼び出し元が他にないか確認済み（`player_data_repository_impl.go` のみ）。

## リスク評価

| リスク | 影響度 | 対策 |
|---|---|---|
| DB に負値の chart constant が存在する場合、マスタ読み込みが失敗する | 低（運用上負値は存在しないが念のため） | マイグレーション前にDBデータの検証クエリを実行: `SELECT * FROM charts WHERE const < 0` |
| `ToEntity()` のシグネチャ変更による他ファイルへの影響 | 低（呼び出し箇所は1箇所のみ確認済み） | `go build` とテスト全通で確認 |

## 見積り影響

- 変更行数は約30行（テスト除く）
- 影響範囲が限定的であり、低リスクな修正
