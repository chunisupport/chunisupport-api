package repository

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransferGoalGroupID(t *testing.T) {
	tests := []struct {
		name     string
		id       int64
		expected uint32
		wantErr  bool
	}{
		{name: "正のIDを変換できる", id: 1, expected: 1},
		{name: "uint32の最大値を変換できる", id: math.MaxUint32, expected: math.MaxUint32},
		{name: "負のIDはエラー", id: -1, wantErr: true},
		{name: "uint32の最大値を超えるIDはエラー", id: int64(math.MaxUint32) + 1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			actual, err := transferGoalGroupID(tt.id)

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
