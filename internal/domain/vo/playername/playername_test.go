package playername

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{
			name:    "有効な半角カナ範囲の直前（U+FF60）を含む",
			value:   "あ\uFF60う",
			want:    PlayerName{value: "あ\uFF60う"},
			wantErr: assert.NoError,
		},
		{
			name:    "有効な半角カナ範囲の直後（U+FFA0）を含む",
			value:   "あ\uFFA0う",
			want:    PlayerName{value: "あ\uFFA0う"},
			wantErr: assert.NoError,
		},
		{
			name:    "無効な半角カナ範囲の下端（U+FF61）を含む",
			value:   "あ\uFF61う",
			want:    PlayerName{},
			wantErr: assert.Error,
		},
		{
			name:    "無効な半角カナ範囲の上端（U+FF9F）を含む",
			value:   "あ\uFF9Fう",
			want:    PlayerName{},
			wantErr: assert.Error,
		},
		{
			name:    "無効な小文字を含む",
			value:   "太郎ab",
			want:    PlayerName{},
			wantErr: assert.Error,
		},
		{
			name:    "有効な絵文字8文字",
			value:   strings.Repeat("😀", 8),
			want:    PlayerName{value: strings.Repeat("😀", 8)},
			wantErr: assert.NoError,
		},
		{
			name:    "無効な絵文字9文字",
			value:   strings.Repeat("😀", 9),
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

func TestPlayerName_JSONRoundTrip(t *testing.T) {
	// Given: 有効な名前のJSON文字列
	payload := []byte(`"太郎１２３"`)

	// When: JSONから読み込む
	var got PlayerName
	err := json.Unmarshal(payload, &got)

	// Then: 文字列に戻り、再びJSON化しても元と同じになる
	require.NoError(t, err)
	assert.Equal(t, "太郎１２３", got.String())
	marshaled, err := json.Marshal(got)
	require.NoError(t, err)
	assert.Equal(t, string(payload), string(marshaled))
}

func TestPlayerName_UnmarshalJSONRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		// Given: 生成関数が拒否する代表的な値（種類の異なるもの）
		value string
	}{
		{
			name:  "空文字列は拒否する",
			value: "",
		},
		{
			name:  "半角英数字を含む値は拒否する",
			value: "太郎12AB",
		},
		{
			name:  "9文字は拒否する",
			value: strings.Repeat("😀", 9),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: 変更前の有効な値と、JSONペイロード
			before, err := NewPlayerName("太郎１２３")
			require.NoError(t, err)
			got := before
			payload, err := json.Marshal(tt.value)
			require.NoError(t, err)
			_, wantErr := NewPlayerName(tt.value)
			require.Error(t, wantErr)

			// When: JSONから読み込む
			err = json.Unmarshal(payload, &got)

			// Then: 生成関数と同じ内容のエラーになり、値は変更前のまま
			assert.EqualError(t, err, wantErr.Error())
			assert.Equal(t, before, got)
		})
	}
}

func TestPlayerName_UnmarshalJSONRejectsNonStringInput(t *testing.T) {
	tests := []struct {
		name string
		// Given: JSONペイロードそのもの
		payload string
	}{
		{
			name:    "nullは拒否する",
			payload: `null`,
		},
		{
			name:    "文字列でない値は拒否する",
			payload: `123`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: 変更前の有効な値
			before, err := NewPlayerName("太郎１２３")
			require.NoError(t, err)
			got := before

			// When: JSONから読み込む
			err = json.Unmarshal([]byte(tt.payload), &got)

			// Then: エラーになり、値は変更前のまま
			assert.Error(t, err)
			assert.Equal(t, before, got)
		})
	}
}
