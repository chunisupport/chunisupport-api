package chartstatsbatch

// clear_lamps マスタのIDです。統計のクリアランプ分布は ID で分類します。
const (
	clearLampIDFailed      = 1
	clearLampIDClear       = 2
	clearLampIDHard        = 3
	clearLampIDBrave       = 4
	clearLampIDAbsolute    = 5
	clearLampIDCatastrophy = 6
)

// combo_lamps マスタのIDです。
const (
	comboLampIDNone       = 1
	comboLampIDFullCombo  = 2
	comboLampIDAllJustice = 3
)

// ランク分布の下限スコアです。
const (
	scoreMax     = 1010000
	scoreSSSPlus = 1009000
	scoreSSS     = 1007500
	scoreSSPlus  = 1005000
	scoreSS      = 1000000
	scoreSPlus   = 990000
	scoreS       = 975000
)

// bestPlayerPercentageScale は採用率を小数第4位で丸めるための倍率です。
// 保存先の best_player_percentage が DECIMAL(7,4) のため、丸めた値をそのまま保存できるようにします。
const bestPlayerPercentageScale = 10000
