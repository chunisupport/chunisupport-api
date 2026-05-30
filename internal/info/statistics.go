package info

import "strings"

const (
	// DifficultyNameUltima は内部で扱うULTIMA難易度名です。
	DifficultyNameUltima = "ULTIMA"

	// StatsDifficultyWorldsend はWORLD'S END譜面を表す難易度名です。
	// 通常の難易度（BASIC, ADVANCED, EXPERT, MASTER, ULTIMA）と異なり、
	// WORLD'S ENDは専用のマスタデータが存在しないため、定数として定義しています。
	StatsDifficultyWorldsend = "WORLD'S END"
)

// DifficultyPathParam はパスパラメータ用の難易度名です。
// すべて小文字で、WORLD'S ENDは"worldsend"として扱います。
type DifficultyPathParam string

// 有効な難易度パスパラメータ
const (
	DifficultyPathBasic     DifficultyPathParam = "basic"
	DifficultyPathAdvanced  DifficultyPathParam = "advanced"
	DifficultyPathExpert    DifficultyPathParam = "expert"
	DifficultyPathMaster    DifficultyPathParam = "master"
	DifficultyPathUltima    DifficultyPathParam = "ultima"
	DifficultyPathWorldsend DifficultyPathParam = "worldsend"
)

// difficultyPathToName はパスパラメータから内部難易度名へのマッピングです。
var difficultyPathToName = map[string]string{
	string(DifficultyPathBasic):     "BASIC",
	string(DifficultyPathAdvanced):  "ADVANCED",
	string(DifficultyPathExpert):    "EXPERT",
	string(DifficultyPathMaster):    "MASTER",
	string(DifficultyPathUltima):    DifficultyNameUltima,
	string(DifficultyPathWorldsend): StatsDifficultyWorldsend,
}

// ValidDifficultyPaths は有効な難易度パスパラメータのリストです。
// APIドキュメントやエラーメッセージで使用されます。
var ValidDifficultyPaths = []string{
	string(DifficultyPathBasic),
	string(DifficultyPathAdvanced),
	string(DifficultyPathExpert),
	string(DifficultyPathMaster),
	string(DifficultyPathUltima),
	string(DifficultyPathWorldsend),
}

// ParseDifficultyPath はパスパラメータを内部難易度名に変換します。
// 無効なパラメータの場合は空文字とfalseを返します。
func ParseDifficultyPath(path string) (difficultyName string, ok bool) {
	// パスパラメータは小文字で正規化して検索
	name, ok := difficultyPathToName[strings.ToLower(path)]
	return name, ok
}
