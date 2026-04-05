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
	difficultyNamesByID := map[int]string{
		1:  "BASIC",
		2:  "ADVANCED",
		3:  "EXPERT",
		4:  "MASTER",
		5:  "ULTIMA",
		99: "MASTER",
		77: "ULTIMA",
	}

	tests := []struct {
		name               string
		charts             []*entity.Chart
		wantMaxChartConst  float64
		wantIsMaxOPUnknown bool
	}{
		{
			name:               "譜面なし楽曲はゼロ値",
			charts:             []*entity.Chart{},
			wantMaxChartConst:  0,
			wantIsMaxOPUnknown: false,
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
			wantMaxChartConst:  14.5,
			wantIsMaxOPUnknown: false,
		},
		{
			name: "MASTER known / ULTIMA unknown（暫定値が低いケース）でもis_maxop_unknown=true",
			charts: []*entity.Chart{
				helperChart(3, 10.5, false), // EXPERT
				helperChart(4, 14.6, false), // MASTER known
				helperChart(5, 14.5, true),  // ULTIMA unknown（暫定値）
			},
			wantMaxChartConst:  14.6,
			wantIsMaxOPUnknown: true,
		},
		{
			name: "MASTER unknown / ULTIMA knownでもis_maxop_unknown=true",
			charts: []*entity.Chart{
				helperChart(3, 10.5, false), // EXPERT
				helperChart(4, 13.5, true),  // MASTER unknown
				helperChart(5, 14.8, false), // ULTIMA known
			},
			wantMaxChartConst:  14.8,
			wantIsMaxOPUnknown: true,
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
			wantMaxChartConst:  14.5,
			wantIsMaxOPUnknown: false,
		},
		{
			name: "MASTER/ULTIMA両方unknownならis_maxop_unknown=true",
			charts: []*entity.Chart{
				helperChart(4, 13.5, true), // MASTER unknown
				helperChart(5, 14.5, true), // ULTIMA unknown
			},
			wantMaxChartConst:  14.5,
			wantIsMaxOPUnknown: true,
		},
		{
			name: "MASTERのみ存在しunknownならis_maxop_unknown=true",
			charts: []*entity.Chart{
				helperChart(3, 10.5, false), // EXPERT known
				helperChart(4, 13.5, true),  // MASTER unknown
			},
			wantMaxChartConst:  13.5,
			wantIsMaxOPUnknown: true,
		},
		{
			name: "MASTERのみ存在しknownならis_maxop_unknown=false",
			charts: []*entity.Chart{
				helperChart(3, 10.5, false), // EXPERT known
				helperChart(4, 13.8, false), // MASTER known
			},
			wantMaxChartConst:  13.8,
			wantIsMaxOPUnknown: false,
		},
		{
			name: "EXPERT以下のみの楽曲（MASTER/ULTIMAなし）はis_maxop_unknown=false",
			charts: []*entity.Chart{
				helperChart(1, 3.0, false),  // BASIC known
				helperChart(2, 6.0, false),  // ADVANCED known
				helperChart(3, 10.5, false), // EXPERT known
			},
			wantMaxChartConst:  10.5,
			wantIsMaxOPUnknown: false,
		},
		{
			name: "同一constのtie-break: max_chart_constは最大値を正しく返す",
			charts: []*entity.Chart{
				helperChart(4, 14.0, false), // MASTER known
				helperChart(5, 14.0, true),  // ULTIMA unknown（同定数）
			},
			wantMaxChartConst:  14.0,
			wantIsMaxOPUnknown: true,
		},
		{
			name: "MASTER/ULTIMAのIDが4/5以外でも難易度名でunknown判定できる",
			charts: []*entity.Chart{
				helperChart(99, 14.2, true), // MASTER unknown (非4)
				helperChart(77, 14.7, false),
			},
			wantMaxChartConst:  14.7,
			wantIsMaxOPUnknown: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := AggregateSongCharts(tt.charts, difficultyNamesByID)

			if agg.MaxChartConst != tt.wantMaxChartConst {
				t.Errorf("MaxChartConst = %v, want %v", agg.MaxChartConst, tt.wantMaxChartConst)
			}

			if agg.IsMaxOPUnknown != tt.wantIsMaxOPUnknown {
				t.Errorf("IsMaxOPUnknown = %v, want %v", agg.IsMaxOPUnknown, tt.wantIsMaxOPUnknown)
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

	difficultyNamesByID := map[int]string{4: "MASTER", 5: "ULTIMA"}
	ApplyAggregation(song, difficultyNamesByID)

	if song.MaxChartConst != 14.6 {
		t.Errorf("MaxChartConst = %v, want %v", song.MaxChartConst, 14.6)
	}

	if !song.IsMaxOPUnknown {
		t.Errorf("IsMaxOPUnknown = %v, want %v", song.IsMaxOPUnknown, true)
	}
}
