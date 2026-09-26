package main

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	"github.com/stretchr/testify/assert"
)

func TestParseRunRequest(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected songbatch.RunRequest
		wantErr  bool
	}{
		{
			name:     "引数なしは通常実行になる",
			args:     nil,
			expected: songbatch.RunRequest{Mode: songbatch.RunModeNormal},
		},
		{
			name:     "major-updateは大型更新になる",
			args:     []string{"--major-update"},
			expected: songbatch.RunRequest{Mode: songbatch.RunModeMajorUpdate},
		},
		{
			name:     "fill-missing-release-dateはリリース日補完を有効にする",
			args:     []string{"--fill-missing-release-date"},
			expected: songbatch.RunRequest{Mode: songbatch.RunModeNormal, FillMissingReleaseDate: true},
		},
		{
			name:    "未知のフラグはエラー",
			args:    []string{"--unknown"},
			wantErr: true,
		},
		{
			name:    "位置引数はエラー",
			args:    []string{"extra"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRunRequest(tt.args)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}
