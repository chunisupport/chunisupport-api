package master

// AccountType はアカウントタイプマスタの値オブジェクトです。
type AccountType struct {
	ID   int
	Name string
}

// ChartDifficulty は譜面難易度マスタの値オブジェクトです。
type ChartDifficulty struct {
	ID   int
	Name string
}

// ClassEmblem はクラスエンブレムマスタの値オブジェクトです。
type ClassEmblem struct {
	ID   int
	Name string
}

// ClassEmblemBase はクラスエンブレムベースマスタの値オブジェクトです。
type ClassEmblemBase struct {
	ID   int
	Name string
}

// ClearLampType はクリアランプマスタの値オブジェクトです。
type ClearLampType struct {
	ID   int
	Name string
}

// ComboLampType はコンボランプマスタの値オブジェクトです。
type ComboLampType struct {
	ID   int
	Name string
}

// FullChainType はフルチェインランプマスタの値オブジェクトです。
type FullChainType struct {
	ID   int
	Name string
}

// Genre はジャンルマスタの値オブジェクトです。
type Genre struct {
	ID   int
	Name string
}

// HonorType は称号種類マスタの値オブジェクトです。
type HonorType struct {
	ID   int
	Name string
}

// Slot はプレイヤーレコードのスロット種別の値オブジェクトです。
type Slot struct {
	ID   int
	Name string
}
