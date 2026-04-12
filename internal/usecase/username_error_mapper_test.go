package usecase

import (
	"errors"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/stretchr/testify/assert"
)

func TestConvertUsernameError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected error
	}{
		{
			name:     "空文字エラーをユースケースエラーへ変換する",
			err:      username.ErrEmpty,
			expected: ErrUsernameEmpty,
		},
		{
			name:     "短すぎるエラーをユースケースエラーへ変換する",
			err:      username.ErrTooShort,
			expected: ErrUsernameTooShort,
		},
		{
			name:     "長すぎるエラーをユースケースエラーへ変換する",
			err:      username.ErrTooLong,
			expected: ErrUsernameTooLong,
		},
		{
			name:     "文字種エラーをユースケースエラーへ変換する",
			err:      username.ErrInvalidChar,
			expected: ErrUsernameInvalidChar,
		},
		{
			name:     "ラップされたVOエラーも変換する",
			err:      errors.Join(errors.New("wrapped"), username.ErrTooShort),
			expected: ErrUsernameTooShort,
		},
		{
			name:     "未知のエラーはそのまま返す",
			err:      errors.New("unknown"),
			expected: nil,
		},
		{
			name:     "nilはnilを返す",
			err:      nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converted := convertUsernameError(tt.err)

			if tt.expected == nil {
				assert.Equal(t, tt.err, converted)
				return
			}

			assert.ErrorIs(t, converted, tt.expected)
		})
	}
}
