package master

// BaseMasterVO は、IDとNameを持つ単純なマスタ値オブジェクトの基本構造体です。
type BaseMasterVO struct {
	ID   int
	Name string
}

// AccountType はアカウントタイプマスタの値オブジェクトです。
type AccountType BaseMasterVO

// ChartDifficulty は譜面難易度マスタの値オブジェクトです。
type ChartDifficulty BaseMasterVO

// ClassEmblem はクラスエンブレムマスタの値オブジェクトです。
type ClassEmblem BaseMasterVO

// ClassEmblemBase はクラスエンブレムベースマスタの値オブジェクトです。
type ClassEmblemBase BaseMasterVO

// ClearLampType はクリアランプマスタの値オブジェクトです。
type ClearLampType BaseMasterVO

// ComboLampType はコンボランプマスタの値オブジェクトです。
type ComboLampType BaseMasterVO

// FullChainType はフルチェインランプマスタの値オブジェクトです。
type FullChainType BaseMasterVO

// Genre はジャンルマスタの値オブジェクトです。
type Genre BaseMasterVO

// HonorType は称号種類マスタの値オブジェクトです。
type HonorType BaseMasterVO

// Slot はプレイヤーレコードのスロット種別の値オブジェクトです。
type Slot BaseMasterVO
