package main

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseExportMode(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected exportMode
		wantErr  bool
	}{
		{name: "引数なしは定期データ", expected: exportModeStaticData},
		{name: "chart-stats指定は統計データ", args: []string{"--chart-stats"}, expected: exportModeChartStats},
		{name: "未知の引数はエラー", args: []string{"--unknown"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			actual, err := parseExportMode(tt.args)

			// Then
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestLockNameForMode_更新対象ごとにロックを分ける(t *testing.T) {
	assert.Equal(t, info.StaticDataExportBatchLockName, lockNameForMode(exportModeStaticData))
	assert.Equal(t, info.ChartStatsExportBatchLockName, lockNameForMode(exportModeChartStats))
}
