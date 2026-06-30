package service

const (
	// DifficultyIDBasic はBASIC難易度のマスターIDです。
	DifficultyIDBasic = 1
	// DifficultyIDAdvanced はADVANCED難易度のマスターIDです。
	DifficultyIDAdvanced = 2
	// DifficultyIDExpert はEXPERT難易度のマスターIDです。
	DifficultyIDExpert = 3
	// DifficultyIDMaster はMASTER難易度のマスターIDです。
	DifficultyIDMaster = 4
	// DifficultyIDUltima はULTIMA難易度のマスターIDです。
	DifficultyIDUltima = 5
)

const (
	playerRatingSlotCount = 50
)

const (
	playerRecordScoreMax     = 1010000
	playerRecordScoreSSSPlus = 1009000
	playerRecordScoreSSS     = 1007500
	playerRecordScoreSSPlus  = 1005000
	playerRecordScoreSS      = 1000000
	playerRecordScoreSPlus   = 990000
	playerRecordScoreS       = 975000
)

var playerRecordDifficultyNames = [...]string{"BASIC", "ADVANCED", "EXPERT", "MASTER", "ULTIMA"}

const (
	// comboLampAllJustice は ALL JUSTICE のコンボランプIDです。
	comboLampAllJustice = 3
	// comboLampFullCombo は FULL COMBO のコンボランプIDです。
	comboLampFullCombo = 2
)

var difficultyNamesByID = map[int]string{
	DifficultyIDBasic:    "BASIC",
	DifficultyIDAdvanced: "ADVANCED",
	DifficultyIDExpert:   "EXPERT",
	DifficultyIDMaster:   "MASTER",
	DifficultyIDUltima:   "ULTIMA",
}

// DifficultyNameByID は難易度IDに対応する難易度名を返します。
func DifficultyNameByID(difficultyID int) (string, bool) {
	name, ok := difficultyNamesByID[difficultyID]
	return name, ok
}
