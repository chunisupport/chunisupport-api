package songbatch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDisplayLevelBandContains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		officialConst float64
		chartConst    float64
		want          bool
	}{
		{name: "公式13の下限13.0は許容する", officialConst: 13.0, chartConst: 13.0, want: true},
		{name: "公式13の上限13.4は許容する", officialConst: 13.0, chartConst: 13.4, want: true},
		{name: "公式13に13.5は矛盾する", officialConst: 13.0, chartConst: 13.5, want: false},
		{name: "公式13に13.9は矛盾する", officialConst: 13.0, chartConst: 13.9, want: false},
		{name: "公式13+の下限13.5は許容する", officialConst: 13.5, chartConst: 13.5, want: true},
		{name: "公式13+の上限13.9は許容する", officialConst: 13.5, chartConst: 13.9, want: true},
		{name: "公式13+に13.4は矛盾する", officialConst: 13.5, chartConst: 13.4, want: false},
		{name: "公式14の下限14.0は許容する", officialConst: 14.0, chartConst: 14.0, want: true},
		{name: "公式14の上限14.4は許容する", officialConst: 14.0, chartConst: 14.4, want: true},
		{name: "公式14に13.9は矛盾する", officialConst: 14.0, chartConst: 13.9, want: false},
		{name: "公式14に14.5は矛盾する", officialConst: 14.0, chartConst: 14.5, want: false},
		{name: "公式14+の下限14.5は許容する", officialConst: 14.5, chartConst: 14.5, want: true},
		{name: "公式14+の上限14.9は許容する", officialConst: 14.5, chartConst: 14.9, want: true},
		{name: "公式14+に14.4は矛盾する", officialConst: 14.5, chartConst: 14.4, want: false},
		{name: "公式14+に15.0は矛盾する", officialConst: 14.5, chartConst: 15.0, want: false},
		{name: "10未満の公式7は7.0のみ許容する", officialConst: 7.0, chartConst: 7.0, want: true},
		{name: "10未満の公式7に7.4は矛盾する", officialConst: 7.0, chartConst: 7.4, want: false},
		{name: "10未満の公式7に7.5は矛盾する", officialConst: 7.0, chartConst: 7.5, want: false},
		{name: "10未満の公式9+は9.5のみ許容する", officialConst: 9.5, chartConst: 9.5, want: true},
		{name: "10未満の公式9+に9.9は矛盾する", officialConst: 9.5, chartConst: 9.9, want: false},
		{name: "公式10は10.4まで許容する", officialConst: 10.0, chartConst: 10.4, want: true},
		{name: "公式10に10.5は矛盾する", officialConst: 10.0, chartConst: 10.5, want: false},
		{name: "帯の途中の14.2は同じ公式14の14.0を許容する", officialConst: 14.2, chartConst: 14.0, want: true},
		{name: "帯の途中の14.2は公式14+の14.5を拒否する", officialConst: 14.2, chartConst: 14.5, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, DisplayLevelBandContains(tt.officialConst, tt.chartConst))
		})
	}
}

func TestFormatOfficialDisplayLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		officialConst float64
		want          string
	}{
		{name: "整数レベルはプラスなし", officialConst: 14.0, want: "14"},
		{name: "0.5始まりはプラス表記", officialConst: 13.5, want: "13+"},
		{name: "帯の途中でも所属する表示レベルを返す", officialConst: 14.2, want: "14"},
		{name: "プラス帯の上限もプラス表記", officialConst: 14.9, want: "14+"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, FormatOfficialDisplayLevel(tt.officialConst))
		})
	}
}
