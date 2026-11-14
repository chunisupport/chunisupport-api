package username

import (
	"encoding/json"
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
			name:    "アンダーバーを含むユーザー名は有効",
			value:   "test_user",
			want:    UserName{value: "test_user"},
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

func TestNewUserName_AcceptsBoundaryLengths(t *testing.T) {
	tests := []struct {
		name string
		// Given: 前提条件となる境界長の入力
		value string
	}{
		{
			name:  "5文字ちょうどは受理される",
			value: "abcde",
		},
		{
			name:  "50文字ちょうどは受理される",
			value: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: 境界長のユーザー名
			// When: NewUserNameで生成する
			got, err := NewUserName(tt.value)

			// Then: エラーなく入力と同じ値が返る
			require.NoError(t, err)
			assert.Equal(t, tt.value, got.String())
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
		{
			name:    "アンダーバー以外の記号を含むとErrInvalidChar",
			value:   "test-user",
			wantErr: ErrInvalidChar,
		},
		{
			name:    "アンダーバーのみで英数字を含まないとErrInvalidChar",
			value:   "_____",
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

func TestUserName_JSONRoundTrip(t *testing.T) {
	// Given: 有効な名前のJSON文字列
	payload := []byte(`"testuser"`)

	// When: JSONから読み込む
	var got UserName
	err := json.Unmarshal(payload, &got)

	// Then: 文字列に戻り、再びJSON化しても元と同じになる
	require.NoError(t, err)
	assert.Equal(t, "testuser", got.String())
	marshaled, err := json.Marshal(got)
	require.NoError(t, err)
	assert.Equal(t, string(payload), string(marshaled))
}

func TestUserName_UnmarshalJSONRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		// Given: 生成関数が拒否する代表的な値（種類の異なるもの）
		value string
		// Then: JSON経由でも同じ型のエラーになること
		wantErr error
	}{
		{
			name:    "空文字列はErrEmptyで拒否する",
			value:   "",
			wantErr: ErrEmpty,
		},
		{
			name:    "4文字はErrTooShortで拒否する",
			value:   "test",
			wantErr: ErrTooShort,
		},
		{
			name:    "英大文字を含む値はErrInvalidCharで拒否する",
			value:   "Testuser",
			wantErr: ErrInvalidChar,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: 変更前の有効な値と、JSONペイロード
			before, err := NewUserName("testuser")
			require.NoError(t, err)
			got := before
			payload, err := json.Marshal(tt.value)
			require.NoError(t, err)

			// When: JSONから読み込む
			err = json.Unmarshal(payload, &got)

			// Then: 生成関数と同じ型のエラーになり、値は変更前のまま
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, before, got)
		})
	}
}

func TestUserName_UnmarshalJSONRejectsNonStringInput(t *testing.T) {
	tests := []struct {
		name string
		// Given: JSONペイロードそのもの
		payload string
		// Then: 期待する型のエラー（nilの場合は内容を問わずエラー）
		wantErr error
	}{
		{
			name:    "nullはErrEmptyで拒否する",
			payload: `null`,
			wantErr: ErrEmpty,
		},
		{
			name:    "文字列でない値は拒否する",
			payload: `123`,
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: 変更前の有効な値
			before, err := NewUserName("testuser")
			require.NoError(t, err)
			got := before

			// When: JSONから読み込む
			err = json.Unmarshal([]byte(tt.payload), &got)

			// Then: エラーになり、値は変更前のまま
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.Error(t, err)
			}
			assert.Equal(t, before, got)
		})
	}
}
