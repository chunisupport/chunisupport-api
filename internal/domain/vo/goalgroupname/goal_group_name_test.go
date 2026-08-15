package goalgroupname

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGoalGroupName(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		expected  string
		wantError bool
	}{
		{name: "前後の空白を除去する", value: "  攻略中  ", expected: "攻略中"},
		{name: "30文字を許可する", value: "あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほ", expected: "あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほ"},
		{name: "空文字を拒否する", value: "  ", wantError: true},
		{name: "31文字を拒否する", value: "あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほま", wantError: true},
		{name: "制御文字を拒否する", value: "攻略\n中", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			value, err := NewGoalGroupName(tt.value)

			// Then
			if tt.wantError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, value.String())
		})
	}
}
