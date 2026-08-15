package coursescore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCourseScore(t *testing.T) {
	tests := []struct {
		name    string
		value   uint32
		wantErr bool
	}{
		{name: "0点", value: 0},
		{name: "理論値", value: 3030000},
		{name: "上限超過", value: 3030001, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := New(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.value, score.Uint32())
		})
	}
}

func TestCourseScore_Scan(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{name: "理論値", value: int64(Max)},
		{name: "上限超過", value: int64(Max) + 1, wantErr: true},
		{name: "uint32の範囲を超える値", value: int64(1) << 32, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var score CourseScore

			err := score.Scan(tt.value)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, Max, score.Uint32())
		})
	}
}
