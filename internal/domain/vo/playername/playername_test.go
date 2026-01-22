package playername

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPlayerName(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    PlayerName
		wantErr assert.ErrorAssertionFunc
		skip    bool
	}{
		{
			name:    "有効な1文字",
			value:   "太",
			want:    PlayerName{value: "太"},
			wantErr: assert.NoError,
		},
		{
			name:    "有効な8文字",
			value:   "あいうえおかきく",
			want:    PlayerName{value: "あいうえおかきく"},
			wantErr: assert.NoError,
		},
		{
			name:    "有効な全角混在文字",
			value:   "太郎１２３",
			want:    PlayerName{value: "太郎１２３"},
			wantErr: assert.NoError,
		},
		{
			name:    "無効な空文字列",
			value:   "",
			want:    PlayerName{},
			wantErr: assert.Error,
		},
		{
			name:    "無効な9文字",
			value:   "あいうえおかきくけ",
			want:    PlayerName{},
			wantErr: assert.Error,
		},
		{
			name:    "無効な半角英数字を含む",
			value:   "太郎12AB",
			want:    PlayerName{},
			wantErr: assert.Error,
			skip:    true,
		},
		{
			name:    "無効な半角カタカナを含む",
			value:   "ﾀﾛｳ",
			want:    PlayerName{},
			wantErr: assert.Error,
			skip:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip("プレイヤー名の仕様が確定するまでスキップ")
			}
			got, err := NewPlayerName(tt.value)
			if !tt.wantErr(t, err, fmt.Sprintf("NewPlayerName(%v)", tt.value)) {
				return
			}
			assert.Equalf(t, tt.want, got, "NewPlayerName(%v)", tt.value)
		})
	}
}

func TestMustNewPlayerName(t *testing.T) {
	t.Run("有効な入力", func(t *testing.T) {
		value := "太郎１２３"
		got := MustNewPlayerName(value)
		assert.Equal(t, PlayerName{value: value}, got)
	})

	t.Run("無効な入力はパニックする", func(t *testing.T) {
		assert.Panics(t, func() {
			MustNewPlayerName("")
		})
	})
}

func TestPlayerName_Value(t *testing.T) {
	playerName := PlayerName{value: "太郎１２３"}
	assert.Equal(t, "太郎１２３", playerName.String())
}

func TestPlayerName_Scan(t *testing.T) {
	type args struct {
		src any
	}
	tests := []struct {
		name    string
		args    args
		want    PlayerName
		wantErr assert.ErrorAssertionFunc
		skip    bool
	}{
		{
			name: "nil",
			args: args{
				src: nil,
			},
			want:    PlayerName{value: ""},
			wantErr: assert.NoError,
		},
		{
			name: "有効な文字列",
			args: args{
				src: "太郎１２３",
			},
			want:    PlayerName{value: "太郎１２３"},
			wantErr: assert.NoError,
		},
		{
			name: "有効な[]byte",
			args: args{
				src: []byte("太郎１２３"),
			},
			want:    PlayerName{value: "太郎１２３"},
			wantErr: assert.NoError,
		},
		{
			name: "無効な半角文字列",
			args: args{
				src: "太郎12AB",
			},
			want:    PlayerName{},
			wantErr: assert.Error,
			skip:    true,
		},
		{
			name: "DBからの空文字列",
			args: args{
				src: "",
			},
			want:    PlayerName{value: ""},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip("プレイヤー名の仕様が確定するまでスキップ")
			}
			p := &PlayerName{}
			err := p.Scan(tt.args.src)
			if !tt.wantErr(t, err, fmt.Sprintf("Scan(%v)", tt.args.src)) {
				return
			}
			assert.Equalf(t, tt.want, *p, "Scan(%v)", tt.args.src)
		})
	}
}

func TestValidatePlayerName(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr assert.ErrorAssertionFunc
		skip    bool
	}{
		{
			name:    "有効な1文字",
			value:   "太",
			wantErr: assert.NoError,
		},
		{
			name:    "有効な8文字",
			value:   "あいうえおかきく",
			wantErr: assert.NoError,
		},
		{
			name:    "有効な全角混在文字列",
			value:   "太郎１２３",
			wantErr: assert.NoError,
		},
		{
			name:    "無効な空文字列",
			value:   "",
			wantErr: assert.Error,
		},
		{
			name:    "無効な9文字",
			value:   "あいうえおかきくけ",
			wantErr: assert.Error,
		},
		{
			name:    "無効な半角英数字を含む",
			value:   "太郎123A",
			wantErr: assert.Error,
			skip:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip("プレイヤー名の仕様が確定するまでスキップ")
			}
			tt.wantErr(t, validatePlayerName(tt.value), fmt.Sprintf("validatePlayerName(%v)", tt.value))
		})
	}
}
