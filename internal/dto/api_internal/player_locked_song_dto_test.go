package api_internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStrictBool_UnmarshalParam(t *testing.T) {
	tests := []struct {
		name     string
		param    string
		expected bool
	}{
		{name: "trueを受け付ける", param: "true", expected: true},
		{name: "falseを受け付ける", param: "false", expected: false},
		{name: "1をtrueとして受け付ける", param: "1", expected: true},
		{name: "0をfalseとして受け付ける", param: "0", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actual strictBool
			err := actual.UnmarshalParam(tt.param)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, bool(actual))
		})
	}
}

func TestStrictBool_UnmarshalParam_不正な値では変更しない(t *testing.T) {
	actual := strictBool(true)

	err := actual.UnmarshalParam("yes")

	assert.Error(t, err)
	assert.True(t, bool(actual))
}
