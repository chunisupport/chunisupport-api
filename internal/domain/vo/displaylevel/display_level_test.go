package displaylevel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantString string
		wantLower  int
		wantUpper  int
		wantErr    bool
	}{
		{name: "整数レベル", input: "13", wantString: "13", wantLower: 130, wantUpper: 134},
		{name: "プラスレベル", input: "13+", wantString: "13+", wantLower: 135, wantUpper: 139},
		{name: "10未満のプラスレベル", input: "9+", wantString: "9+", wantLower: 95, wantUpper: 95},
		{name: "空文字は無効", input: "", wantErr: true},
		{name: "小数表記は無効", input: "13.5", wantErr: true},
		{name: "先頭ゼロは無効", input: "013", wantErr: true},
		{name: "16プラスは範囲外", input: "16+", wantErr: true},
		{name: "17は範囲外", input: "17", wantErr: true},
		{name: "前後の空白は無効", input: " 14+ ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantString, got.String())
			assert.Equal(t, tt.wantLower, got.LowerTenths())
			assert.Equal(t, tt.wantUpper, got.UpperTenths())
		})
	}
}
