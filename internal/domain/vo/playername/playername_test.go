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
		},
		{
			name:    "無効な半角カタカナを含む",
			value:   "ﾞﾛｳ",
			want:    PlayerName{},
			wantErr: assert.Error,
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
			name: "DBのNULLはエラー",
			args: args{
				src: nil,
			},
			want:    PlayerName{value: "変更前"},
			wantErr: assert.Error,
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
			want:    PlayerName{value: "変更前"},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip("プレイヤー名の仕様が確定するまでスキップ")
			}
			p := &PlayerName{value: "変更前"}
			err := p.Scan(tt.args.src)
			if !tt.wantErr(t, err, fmt.Sprintf("Scan(%v)", tt.args.src)) {
				return
			}
			assert.Equalf(t, tt.want, *p, "Scan(%v)", tt.args.src)
		})
	}
}
