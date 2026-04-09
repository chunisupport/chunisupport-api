package entity

// PlayerHonor はプレイヤーの称号情報を表す構造体です。
type PlayerHonor struct {
	Slot     int     // 称号スロット: 1=上段, 2=中段, 3=下段
	Name     string  // 称号名
	TypeName string  // 称号タイプ名 (normal, copper, silver, gold, platina, rainbow, etc.)
	ImageURL *string // 称号画像URL
}
