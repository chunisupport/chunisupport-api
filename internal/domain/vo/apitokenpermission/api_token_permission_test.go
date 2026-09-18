package apitokenpermission

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAPITokenPermission(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected APITokenPermission
		canWrite bool
	}{
		{name: "read", value: "read", expected: Read, canWrite: false},
		{name: "read_write", value: "read_write", expected: ReadWrite, canWrite: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			permission, err := NewAPITokenPermission(tt.value)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, permission)
			assert.Equal(t, tt.canWrite, permission.CanWrite())
			assert.Equal(t, tt.value, permission.String())
		})
	}
}

func TestNewAPITokenPermission_Invalid(t *testing.T) {
	_, err := NewAPITokenPermission("")

	assert.ErrorIs(t, err, ErrInvalidAPITokenPermission)
}
