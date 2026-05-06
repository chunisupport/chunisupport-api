package chartconstant

import (
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
			name:        "16.0を超える値ならエラーになる",
			input:       16.1,
			expectedErr: "chart constant must be between 0.0 and 16.0",
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
			name:     "0なら読み込める",
			input:    float64(0),
			expected: ChartConstant(0),
			wantErr:  false,
		},
		{
			name:        "負の値ならエラーになる",
			input:       float64(-1),
			expectedErr: "chart constant must be between 0.0 and 16.0",
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
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if err == nil || err.Error() != tt.expectedErr {
					t.Fatalf("error = %v, want %q", err, tt.expectedErr)
				}
				return
			}
			if got != tt.expected {
				t.Fatalf("got = %v, want %v", got, tt.expected)
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
			expectedErr: "chart constant must be between 0.0 and 16.0",
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
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if err == nil || err.Error() != tt.expectedErr {
					t.Fatalf("error = %v, want %q", err, tt.expectedErr)
				}
				return
			}
			if got != tt.expected {
				t.Fatalf("got = %v, want %v", got, tt.expected)
			}
		})
	}
}
