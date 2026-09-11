package displayid

import (
	"github.com/stretchr/testify/assert"
	"testing"
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
			name:    "大文字を含む",
			input:   "0123456789ABCDEF",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDisplayID(tt.input)
			if (err != nil) != tt.wantErr {
				assert.Failf(t, "アサーション失敗", "NewDisplayID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.String() != tt.input {
				assert.Failf(t, "アサーション失敗", "NewDisplayID() = %v, want %v", got, tt.input)
			}
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
