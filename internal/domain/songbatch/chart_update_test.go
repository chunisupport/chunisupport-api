package songbatch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestResolveChartUpdateBoundaries は大型更新の10.0と0.5幅の境界で、保存候補と保存方針が維持されることを検証します。
func TestResolveChartUpdateBoundaries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		existing *ChartUpdateValues
		incoming ChartUpdateValues
		want     ChartUpdateResult
	}{
		{
			name:     "新規の10未満は既知にする",
			incoming: ChartUpdateValues{Const: 9.9, IsConstUnknown: true},
			want:     ChartUpdateResult{Values: ChartUpdateValues{Const: 9.9}, Action: ChartInsert},
		},
		{
			name:     "新規の10ちょうどは不明にする",
			incoming: ChartUpdateValues{Const: 10},
			want:     ChartUpdateResult{Values: ChartUpdateValues{Const: 10, IsConstUnknown: true}, Action: ChartInsert},
		},
		{
			name:     "既存の10未満は公式値を採用する",
			existing: &ChartUpdateValues{Const: 10, IsConstUnknown: true},
			incoming: ChartUpdateValues{Const: 9.9, IsConstUnknown: true},
			want:     ChartUpdateResult{Values: ChartUpdateValues{Const: 9.9}, Action: ChartUpdate},
		},
		{
			name:     "範囲下端を含み変更なしならスキップする",
			existing: &ChartUpdateValues{Const: 10, IsConstUnknown: true},
			incoming: ChartUpdateValues{Const: 10},
			want:     ChartUpdateResult{Values: ChartUpdateValues{Const: 10, IsConstUnknown: true}, Action: ChartSkip},
		},
		{
			name:     "範囲上端の直前は既存定数を保持する",
			existing: &ChartUpdateValues{Const: 10.49},
			incoming: ChartUpdateValues{Const: 10},
			want:     ChartUpdateResult{Values: ChartUpdateValues{Const: 10.49, IsConstUnknown: true}, Action: ChartUpdate},
		},
		{
			name:     "範囲外の既存不明値は公式値へ戻す",
			existing: &ChartUpdateValues{Const: 9.9, IsConstUnknown: true},
			incoming: ChartUpdateValues{Const: 10},
			want:     ChartUpdateResult{Values: ChartUpdateValues{Const: 10, IsConstUnknown: true}, Action: ChartUpdate},
		},
		{
			name:     "範囲上端ちょうどは公式値へ戻す",
			existing: &ChartUpdateValues{Const: 10.5, IsConstUnknown: true},
			incoming: ChartUpdateValues{Const: 10},
			want:     ChartUpdateResult{Values: ChartUpdateValues{Const: 10, IsConstUnknown: true}, Action: ChartUpdate},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: 大型更新の境界にある既存値と取り込み値。
			input := ChartUpdateInput{Existing: tt.existing, Incoming: tt.incoming, MajorUpdate: true}
			// When: DBを介さず更新方針を決定する。
			got := ResolveChartUpdate(input)
			// Then: 保存候補と挿入・更新・スキップの選択を維持する。
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestResolveChartUpdateComplementaryValues は取得済みの0を欠落と区別し、既存の製作者と入力値を保持することを検証します。
func TestResolveChartUpdateComplementaryValues(t *testing.T) {
	t.Parallel()
	// Given: 不明定数の既存譜面と、定数確定・補完値の取り込み。
	existing := ChartUpdateValues{Const: 14, IsConstUnknown: true, Notes: new(int64(700)), NotesDesigner: new("既存")}
	incoming := ChartUpdateValues{Const: 14.2, Notes: new(int64(0)), NotesDesigner: new("")}
	// When: 更新判定する。
	got := ResolveChartUpdate(ChartUpdateInput{Existing: &existing, Incoming: incoming})
	// Then: ノーツの0は欠落と区別し、既存の製作者を保持する。入力は変更しない。
	assert.Equal(t, ChartUpdateResult{Values: ChartUpdateValues{Const: 14.2, Notes: new(int64(0)), NotesDesigner: new("既存")}, Action: ChartUpdate}, got)
	assert.Equal(t, ChartUpdateValues{Const: 14, IsConstUnknown: true, Notes: new(int64(700)), NotesDesigner: new("既存")}, existing)
	assert.Equal(t, ChartUpdateValues{Const: 14.2, Notes: new(int64(0)), NotesDesigner: new("")}, incoming)
}

// TestResolveChartUpdateFillsEmptyDesigner は取得済みの空文字を未取得と区別し、製作者の補完として更新することを検証します。
func TestResolveChartUpdateFillsEmptyDesigner(t *testing.T) {
	t.Parallel()
	// Given: 製作者が未取得の譜面へ、取得済みの空文字が届く。
	existing := ChartUpdateValues{Const: 14}
	incoming := ChartUpdateValues{Const: 14, NotesDesigner: new("")}
	// When: 通常更新を判定する。
	got := ResolveChartUpdate(ChartUpdateInput{Existing: &existing, Incoming: incoming})
	// Then: 空文字を欠落として扱わず、補完による更新にする。
	assert.Equal(t, ChartUpdateResult{Values: incoming, Action: ChartUpdate}, got)
}

// TestResolveChartUpdateKeepsKnownConstOnMismatch は通常更新で既知定数同士が一致しない場合に既存値を保持することを検証します。
func TestResolveChartUpdateKeepsKnownConstOnMismatch(t *testing.T) {
	t.Parallel()
	// Given: 既知定数の既存譜面へ異なる既知定数が届く。
	existing := ChartUpdateValues{Const: 14}
	incoming := ChartUpdateValues{Const: 14.2}
	// When: 通常更新を判定する。
	got := ResolveChartUpdate(ChartUpdateInput{Existing: &existing, Incoming: incoming})
	// Then: 既知定数を上書きせずスキップする。
	assert.Equal(t, ChartUpdateResult{Values: ChartUpdateValues{Const: 14}, Action: ChartSkip}, got)
}
