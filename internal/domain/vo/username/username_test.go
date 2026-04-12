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

func TestMustNewUserName(t *testing.T) {
	t.Run("有効な入力", func(t *testing.T) {
		value := "testuser"
		got := MustNewUserName(value)
		assert.Equal(t, UserName{value: value}, got)
	})

	t.Run("無効な入力はパニックする", func(t *testing.T) {
		assert.Panics(t, func() {
			MustNewUserName("")
		})
	})
}

func TestUserName_Value(t *testing.T) {
	userName := UserName{value: "testuser"}
	assert.Equal(t, "testuser", userName.String())
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
			name: "nil",
			args: args{
				src: nil,
			},
			want:    UserName{value: ""},
			wantErr: assert.NoError,
		},
		{
			name: "5文字未満の文字列はエラー",
			args: args{
				src: "test",
			},
			want:    UserName{},
			wantErr: assert.Error,
		},
		{
			name: "5文字未満の[]byteはエラー",
			args: args{
				src: []byte("test"),
			},
			want:    UserName{},
			wantErr: assert.Error,
		},
		{
			name: "DBからの空文字列",
			args: args{
				src: "",
			},
			want:    UserName{value: ""},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &UserName{}
			err := u.Scan(tt.args.src)
			if !tt.wantErr(t, err, fmt.Sprintf("Scan(%v)", tt.args.src)) {
				return
			}
			assert.Equalf(t, tt.want, *u, "Scan(%v)", tt.args.src)
		})
	}
}

func TestValidateUserName(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "有効なユーザー名",
			value:   "testuser",
			wantErr: assert.NoError,
		},
		{
			name:    "無効な空文字列",
			value:   "",
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.wantErr(t, validateUserName(tt.value), fmt.Sprintf("validateUserName(%v)", tt.value))
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
