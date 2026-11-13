package displayid

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDisplayID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "有効な16進数16文字",
			input:   "0123456789abcdef",
			wantErr: false,
		},
		{
			name:    "15文字(短すぎる)",
			input:   "0123456789abcde",
			wantErr: true,
		},
		{
			name:    "17文字(長すぎる)",
			input:   "0123456789abcdef0",
			wantErr: true,
		},
		{
			name:    "大文字を含む",
			input:   "0123456789ABCDEF",
			wantErr: true,
		},
		{
			name:    "16進数でない文字gを含む",
			input:   "0123456789abcdeg",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got, err := NewDisplayID(tt.input)

			// Then
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.input, got.String())
		})
	}
}

func TestDisplayID_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		id    DisplayID
		valid bool
	}{
		{
			name:  "有効なID",
			id:    "0123456789abcdef",
			valid: true,
		},
		{
			name:  "無効なID(大文字)",
			id:    "0123456789ABCDEF",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.IsValid(); got != tt.valid {
				assert.Failf(t, "アサーション失敗", "DisplayID.IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}
