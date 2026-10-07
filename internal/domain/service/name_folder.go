package service

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// ResolveNameFolderCode は正規化済みの読みを前提とする名前順フォルダ分類を返します。
// 読みが空なら曲名を使います。分類できない文字を数字フォルダへ含めるのは、既存の名前順フォルダとの互換性のためです。
func ResolveNameFolderCode(title string, reading *string) string {
	source := ""
	if reading != nil {
		source = strings.TrimFunc(*reading, isSongReadingSpace)
	}
	if source == "" {
		source = strings.TrimFunc(title, isSongReadingSpace)
	}
	first, _ := utf8.DecodeRuneInString(norm.NFKC.String(source))
	for _, group := range nameFolderGroups {
		if strings.ContainsRune(group.characters, first) {
			return group.code
		}
	}
	return "NUMBER"
}

var nameFolderGroups = [...]struct{ code, characters string }{
	{"ABCD", "ABCD"}, {"EFGH", "EFGH"}, {"IJKL", "IJKL"},
	{"MNOP", "MNOP"}, {"QRST", "QRST"}, {"UVWXYZ", "UVWXYZ"},
	{"A", "アイウエオ"}, {"KA", "カキクケコ"}, {"SA", "サシスセソ"},
	{"TA", "タチツテト"}, {"NA", "ナニヌネノ"}, {"HA", "ハヒフヘホ"},
	{"MA", "マミムメモ"}, {"YA", "ヤユヨ"}, {"RA", "ラリルレロ"}, {"WA", "ワヰヱヲン"},
}

// isSongReadingSpace はフロントのtrimと揃え、BOMを空白として扱い、NELを含めません。
func isSongReadingSpace(r rune) bool {
	return r == '\uFEFF' || r != '\u0085' && unicode.IsSpace(r)
}
