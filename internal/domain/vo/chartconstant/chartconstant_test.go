package chartconstant

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChartConstant(t *testing.T) {
	tests := []struct {
		name        string
		input       float64
		expected    ChartConstant
		expectedErr string
		wantErr     bool
	}{
		{
			name:     "16.0なら生成できる",
			input:    16.0,
			expected: ChartConstant(16.0),
			wantErr:  false,
		},
		{
			name:        "小数点以下2桁ならエラーになる",
			input:       15.01,
			expectedErr: "chart constant must have at most one decimal place: 15.01",
			wantErr:     true,
		},
		{
			name:        "16.0を超える値ならエラーになる",
			input:       16.1,
			expectedErr: "chart constant must be between 1.0 and 16.0",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got, err := NewChartConstant(tt.input)

			// Then
			if tt.wantErr {
				require.Error(t, err)
				assert.EqualError(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestChartConstantScan(t *testing.T) {
	tests := []struct {
		name        string
		input       any
		expected    ChartConstant
		expectedErr string
		wantErr     bool
	}{
		{
			name:     "正の譜面定数値なら読み込める",
			input:    []byte("13.5"),
			expected: ChartConstant(13.5),
			wantErr:  false,
		},
		{
			name:        "0ならエラーになる",
			input:       float64(0),
			expectedErr: "chart constant must be between 1.0 and 16.0",
			wantErr:     true,
		},
		{
			name:        "負の値ならエラーになる",
			input:       float64(-1),
			expectedErr: "chart constant must be between 1.0 and 16.0",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			var got ChartConstant

			// When
			err := got.Scan(tt.input)

			// Then
			if (err != nil) != tt.wantErr {
				require.Failf(t, "前提条件失敗", "error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if err == nil || err.Error() != tt.expectedErr {
					require.Failf(t, "前提条件失敗", "error = %v, want %q", err, tt.expectedErr)
				}
				return
			}
			if got != tt.expected {
				require.Failf(t, "前提条件失敗", "got = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestChartConstantUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    ChartConstant
		expectedErr string
		wantErr     bool
	}{
		{
			name:     "正の譜面定数値なら復元できる",
			input:    "14.0",
			expected: ChartConstant(14.0),
			wantErr:  false,
		},
		{
			name:        "負の値ならエラーになる",
			input:       "-0.1",
			expectedErr: "chart constant must be between 1.0 and 16.0",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			var got ChartConstant

			// When
			err := got.UnmarshalJSON([]byte(tt.input))

			// Then
			if (err != nil) != tt.wantErr {
				require.Failf(t, "前提条件失敗", "error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if err == nil || err.Error() != tt.expectedErr {
					require.Failf(t, "前提条件失敗", "error = %v, want %q", err, tt.expectedErr)
				}
				return
			}
			if got != tt.expected {
				require.Failf(t, "前提条件失敗", "got = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestChartConstantRejectsInvalidValues(t *testing.T) {
	values := []float64{-1, 0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 16.1, math.NaN(), math.Inf(1), math.Inf(-1)}
	for _, value := range values {
		t.Run(fmt.Sprint(value), func(t *testing.T) {
			_, err := NewChartConstant(value)
			assert.Error(t, err)
			valid, err := NewChartConstant(1)
			require.NoError(t, err)
			assert.Error(t, valid.Scan(value))
			assert.Equal(t, ChartConstant(1), valid)
			_, err = ChartConstant(value).Value()
			assert.Error(t, err)
		})
	}
}

func TestChartConstantRejectsMissingValues(t *testing.T) {
	valid, err := NewChartConstant(1)
	require.NoError(t, err)
	assert.Error(t, valid.Scan(nil))
	assert.Equal(t, ChartConstant(1), valid)
	for _, data := range []string{"null", "0", "0.5", "0.9"} {
		assert.Error(t, json.Unmarshal([]byte(data), &valid))
		assert.Equal(t, ChartConstant(1), valid)
	}
}

func TestChartConstantLowerBoundary(t *testing.T) {
	value, err := NewChartConstant(1)
	require.NoError(t, err)
	assert.Equal(t, 1.0, value.Float64())
}
