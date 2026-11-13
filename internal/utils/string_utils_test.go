package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscapeLike_SQLのワイルドカードとエスケープ文字だけをリテラル化する(t *testing.T) {
	for _, tt := range []struct{ name, input, want string }{
		{"空文字", "", ""},
		{"通常文字", "楽曲ABC'", "楽曲ABC'"},
		{"パーセント", "100%", "100\\%"},
		{"アンダースコア", "a_b", "a\\_b"},
		{"バックスラッシュ", `a\b`, `a\\b`},
		{"混在した文字を二重にエスケープしない", `\%_\`, `\\\%\_\\`},
	} {
		t.Run(tt.name, func(t *testing.T) { assert.Equal(t, tt.want, EscapeLike(tt.input)) })
	}
}
