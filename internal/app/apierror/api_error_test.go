package apierror

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPIError_Is(t *testing.T) {
	tests := []struct {
		name string
		// Given
		err    error
		target error
		// Then
		expected bool
	}{
		{
			name:     "内部エラー付きでも同じコードの定義済みエラーに一致する",
			err:      ErrInternalError.WithInternal(context.Canceled),
			target:   ErrInternalError,
			expected: true,
		},
		{
			name:     "内部エラーの原因にも一致する",
			err:      ErrInternalError.WithInternal(context.Canceled),
			target:   context.Canceled,
			expected: true,
		},
		{
			name:     "異なるコードの定義済みエラーには一致しない",
			err:      ErrInternalError.WithInternal(context.Canceled),
			target:   ErrBadRequest,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			result := errors.Is(tt.err, tt.target)

			// Then
			assert.Equal(t, tt.expected, result)
		})
	}
}
