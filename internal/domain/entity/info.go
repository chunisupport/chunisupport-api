package entity

const (
	// DefaultPlayerLevel は新規作成時に適用するプレイヤーの初期レベルです。
	DefaultPlayerLevel = 1
	// DefaultPossessionID はポゼッション normal の固定IDです。
	DefaultPossessionID = 1
	// PossessionNameNormal はデフォルトのポゼッション名です。
	PossessionNameNormal = "normal"
	DifficultyIDBasic    = 1
	DifficultyIDAdvanced = 2
	DifficultyIDExpert   = 3
	DifficultyIDMaster   = 4
	DifficultyIDUltima   = 5
	// TeamNameMaxLength はチーム名の最大コードポイント数です（teams.name の VARCHAR 長と一致させます）。
	TeamNameMaxLength = 64
	// TeamColorMaxLength はチームカラーの最大バイト数です（teams.color の VARCHAR 長と一致させます）。
	TeamColorMaxLength = 16
)
