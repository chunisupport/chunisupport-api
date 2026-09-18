package username

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUserName(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    UserName
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "有効なユーザー名",
			value:   "testuser",
			want:    UserName{value: "testuser"},
			wantErr: assert.NoError,
		},
		{
			name:    "無効な空文字列",
			value:   "",
			want:    UserName{},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewUserName(tt.value)
			if !tt.wantErr(t, err, fmt.Sprintf("NewUserName(%v)", tt.value)) {
				return
			}
			assert.Equalf(t, tt.want, got, "NewUserName(%v)", tt.value)
		})
	}
}

func TestUserName_Scan(t *testing.T) {
	type args struct {
		src any
	}
	tests := []struct {
		name    string
		args    args
		want    UserName
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "DBのNULLはエラー",
			args: args{
				src: nil,
			},
			want:    UserName{value: "before"},
			wantErr: assert.Error,
		},
		{
			name: "有効な文字列",
			args: args{
				src: "testuser",
			},
			want:    UserName{value: "testuser"},
			wantErr: assert.NoError,
		},
		{
			name: "有効な[]byte",
			args: args{
				src: []byte("testuser"),
			},
			want:    UserName{value: "testuser"},
			wantErr: assert.NoError,
		},
		{
			name: "5文字未満の文字列はエラー",
			args: args{
				src: "test",
			},
			want:    UserName{value: "before"},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &UserName{value: "before"}
			err := u.Scan(tt.args.src)
			if !tt.wantErr(t, err, fmt.Sprintf("Scan(%v)", tt.args.src)) {
				return
			}
			assert.Equalf(t, tt.want, *u, "Scan(%v)", tt.args.src)
		})
	}
}

func TestValidateUserNameReturnsTypedErrors(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr error
	}{
		{
			name:    "空文字はErrEmpty",
			value:   "",
			wantErr: ErrEmpty,
		},
		{
			name:    "4文字はErrTooShort",
			value:   "test",
			wantErr: ErrTooShort,
		},
		{
			name:    "51文字はErrTooLong",
			value:   "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxy",
			wantErr: ErrTooLong,
		},
		{
			name:    "英大文字を含むとErrInvalidChar",
			value:   "Testuser",
			wantErr: ErrInvalidChar,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUserName(tt.value)

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}
