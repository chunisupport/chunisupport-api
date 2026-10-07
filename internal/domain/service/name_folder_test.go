package service_test

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/stretchr/testify/assert"
)

func TestResolveNameFolderCode(t *testing.T) {
	tests := []struct{ title, reading, want string }{
		{"", "ABCD", "ABCD"}, {"", "ECHO", "EFGH"}, {"", "LOVE", "IJKL"},
		{"", "PEACE", "MNOP"}, {"", "TIME", "QRST"}, {"", "ZERO", "UVWXYZ"},
		{"", "アイウエオ", "A"}, {"", "ココロ", "KA"}, {"", "ソラ", "SA"},
		{"", "トキ", "TA"}, {"", "ノハラ", "NA"}, {"", "ホシ", "HA"},
		{"", "モリ", "MA"}, {"", "ヨル", "YA"}, {"", "ロツク", "RA"}, {"", "ン", "WA"},
		{"", "39", "NUMBER"}, {"", "@TEST", "NUMBER"}, {"", "漢字", "NUMBER"},
		{"", "ガ", "NUMBER"}, {"", "あ", "NUMBER"}, {"", "abc", "NUMBER"},
		{"", "\uFEFFABC", "ABCD"}, {"", "\u0085ABC", "NUMBER"},
		{"", " ＡＢＣ ", "ABCD"}, {"", " ｶﾅ ", "KA"},
		{"", "ｶﾞ", "NUMBER"}, {"", "カ\u3099", "NUMBER"},
		{"ソラ", "  ", "SA"}, {"BRAND NEW DAY", "", "ABCD"},
		{"", "", "NUMBER"}, {"ソラ", "ECHO", "EFGH"},
	}
	for _, tt := range tests {
		t.Run(tt.title+"/"+tt.reading, func(t *testing.T) {
			assert.Equal(t, tt.want, service.ResolveNameFolderCode(tt.title, &tt.reading))
		})
	}
	assert.Equal(t, "ABCD", service.ResolveNameFolderCode("BRAND NEW DAY", nil))
}
