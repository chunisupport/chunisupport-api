package service

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
)

// helperChart はテスト用の譜面エンティティを簡易生成します。
func helperChart(difficultyID int, constVal float64, isConstUnknown bool) *entity.Chart {
	cc, _ := chartconstant.NewChartConstant(constVal)
	return &entity.Chart{
		DifficultyID:   difficultyID,
		Const:          cc,
		IsConstUnknown: isConstUnknown,
	}
}

func TestAggregateSongCharts(t *testing.T) {
	tests := []struct {
		name                     string
		charts                   []*entity.Chart
		wantMaxChartConst        float64
		wantIsMaxOPUnknown       bool
		wantOpTargetDifficultyID int
	}{
		{
			name:                     "譜面なし楽曲はゼロ値",
			charts:                   []*entity.Chart{},
			wantMaxChartConst:        0,
			wantIsMaxOPUnknown:       false,
			wantOpTargetDifficultyID: 0,
		},
		{
			name: "全譜面がknownならis_maxop_unknown=false",
			charts: []*entity.Chart{
				helperChart(1, 3.0, false),  // BASIC
				helperChart(2, 6.0, false),  // ADVANCED
				helperChart(3, 10.5, false), // EXPERT
				helperChart(4, 13.8, false), // MASTER
				helperChart(5, 14.5, false), // ULTIMA
			},
			wantMaxChartConst:        14.5,
			wantIsMaxOPUnknown:       false,
			wantOpTargetDifficultyID: DifficultyIDUltima,
		},
		{
			name: "MASTER known / ULTIMA unknown（暫定値が低いケース）でもis_maxop_unknown=true",
			charts: []*entity.Chart{
				helperChart(3, 10.5, false), // EXPERT
				helperChart(4, 14.6, false), // MASTER known
				helperChart(5, 14.5, true),  // ULTIMA unknown（暫定値）
			},
			wantMaxChartConst:        14.6,
			wantIsMaxOPUnknown:       true,
			wantOpTargetDifficultyID: DifficultyIDMaster,
		},
		{
			name: "MASTER unknown / ULTIMA knownでもis_maxop_unknown=true",
			charts: []*entity.Chart{
				helperChart(3, 10.5, false), // EXPERT
				helperChart(4, 13.5, true),  // MASTER unknown
				helperChart(5, 14.8, false), // ULTIMA known
			},
			wantMaxChartConst:        14.8,
			wantIsMaxOPUnknown:       true,
			wantOpTargetDifficultyID: DifficultyIDUltima,
		},
		{
			name: "EXPERT以下のみunknownでMASTER/ULTIMAがknownならis_maxop_unknown=false",
			charts: []*entity.Chart{
				helperChart(1, 3.0, true),   // BASIC unknown
				helperChart(2, 6.0, true),   // ADVANCED unknown
				helperChart(3, 10.5, true),  // EXPERT unknown
				helperChart(4, 13.8, false), // MASTER known
				helperChart(5, 14.5, false), // ULTIMA known
			},
			wantMaxChartConst:        14.5,
			wantIsMaxOPUnknown:       false,
			wantOpTargetDifficultyID: DifficultyIDUltima,
		},
		{
			name: "MASTER/ULTIMA両方unknownならis_maxop_unknown=true",
			charts: []*entity.Chart{
				helperChart(4, 13.5, true), // MASTER unknown
				helperChart(5, 14.5, true), // ULTIMA unknown
			},
			wantMaxChartConst:        14.5,
			wantIsMaxOPUnknown:       true,
			wantOpTargetDifficultyID: DifficultyIDUltima,
		},
		{
			name: "MASTERのみ存在しunknownならis_maxop_unknown=true",
			charts: []*entity.Chart{
				helperChart(3, 10.5, false), // EXPERT known
				helperChart(4, 13.5, true),  // MASTER unknown
			},
			wantMaxChartConst:        13.5,
			wantIsMaxOPUnknown:       true,
			wantOpTargetDifficultyID: DifficultyIDMaster,
		},
		{
			name: "MASTERのみ存在しknownならis_maxop_unknown=false",
			charts: []*entity.Chart{
				helperChart(3, 10.5, false), // EXPERT known
				helperChart(4, 13.8, false), // MASTER known
			},
			wantMaxChartConst:        13.8,
			wantIsMaxOPUnknown:       false,
			wantOpTargetDifficultyID: DifficultyIDMaster,
		},
		{
			name: "EXPERT以下のみの楽曲（MASTER/ULTIMAなし）はis_maxop_unknown=false",
			charts: []*entity.Chart{
				helperChart(1, 3.0, false),  // BASIC known
				helperChart(2, 6.0, false),  // ADVANCED known
				helperChart(3, 10.5, false), // EXPERT known
			},
			wantMaxChartConst:        10.5,
			wantIsMaxOPUnknown:       false,
			wantOpTargetDifficultyID: DifficultyIDExpert,
		},
		{
			name: "同一constのtie-break: 難易度IDが大きい譜面をOP対象とする",
			charts: []*entity.Chart{
				helperChart(4, 14.0, false), // MASTER known
				helperChart(5, 14.0, true),  // ULTIMA unknown（同定数）
			},
			wantMaxChartConst:        14.0,
			wantIsMaxOPUnknown:       true,
			wantOpTargetDifficultyID: DifficultyIDUltima,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := AggregateSongCharts(tt.charts)

			if agg.MaxChartConst != tt.wantMaxChartConst {
				t.Errorf("MaxChartConst = %v, want %v", agg.MaxChartConst, tt.wantMaxChartConst)
			}

			if agg.IsMaxOPUnknown != tt.wantIsMaxOPUnknown {
				t.Errorf("IsMaxOPUnknown = %v, want %v", agg.IsMaxOPUnknown, tt.wantIsMaxOPUnknown)
			}

			if agg.OpTargetDifficultyID != tt.wantOpTargetDifficultyID {
				t.Errorf("OpTargetDifficultyID = %v, want %v", agg.OpTargetDifficultyID, tt.wantOpTargetDifficultyID)
			}
		})
	}
}

func TestApplyAggregation(t *testing.T) {
	song := &entity.Song{
		Charts: []*entity.Chart{
			helperChart(4, 14.6, false), // MASTER known
			helperChart(5, 14.5, true),  // ULTIMA unknown
		},
	}

	ApplyAggregation(song)

	if song.MaxChartConst != 14.6 {
		t.Errorf("MaxChartConst = %v, want %v", song.MaxChartConst, 14.6)
	}

	if !song.IsMaxOPUnknown {
		t.Errorf("IsMaxOPUnknown = %v, want %v", song.IsMaxOPUnknown, true)
	}

	if song.OpTargetDifficultyID != DifficultyIDMaster {
		t.Errorf("OpTargetDifficultyID = %v, want %v", song.OpTargetDifficultyID, DifficultyIDMaster)
	}
}
