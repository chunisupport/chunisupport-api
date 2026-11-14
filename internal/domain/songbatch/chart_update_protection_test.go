package songbatch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveChartUpdate_既知定数への不明値取り込みは補完も保存しない(t *testing.T) {
	for _, tt := range []struct {
		name    string
		unknown bool
		action  ChartUpdateAction
	}{
		{"既知値なら補完する", false, ChartUpdate},
		{"不明値なら補完ごとスキップする", true, ChartSkip},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// 定数を一致させ、補完可能な値も与えて、別条件によるスキップを避ける。
			existing := ChartUpdateValues{Const: 14}
			incoming := ChartUpdateValues{Const: 14, IsConstUnknown: tt.unknown, Notes: new(int64(800)), NotesDesigner: new("譜面製作者")}
			got := ResolveChartUpdate(ChartUpdateInput{Existing: &existing, Incoming: incoming})
			assert.Equal(t, tt.action, got.Action)
			// スキップ時の候補値は保存されないため、内部の組み立て方は固定しない。
			if tt.action == ChartUpdate {
				assert.Equal(t, ChartUpdateValues{Const: 14, Notes: new(int64(800)), NotesDesigner: new("譜面製作者")}, got.Values)
			}
			assert.Equal(t, ChartUpdateValues{Const: 14}, existing)
		})
	}
}
