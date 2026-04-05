package master

// BaseMasterVO は、IDとNameを持つ単純なマスタ値オブジェクトの基本構造体です。
// sort_order を持たないマスタ（genres, honor_types, slots など）に使用します。
type BaseMasterVO struct {
	ID   int
	Name string
}

// SortedMasterVO は表示順を持つマスタ値オブジェクトの基本構造体です。
// DBに sort_order カラムが存在するマスタ（difficulties, class_emblems 等）に使用します。
type SortedMasterVO struct {
	ID        int
	Name      string
	SortOrder int
}

// AccountType はアカウントタイプマスタの値オブジェクトです。
type AccountType BaseMasterVO

// ChartDifficulty は譜面難易度マスタの値オブジェクトです。
type ChartDifficulty SortedMasterVO

// ClassEmblem はクラスエンブレムマスタの値オブジェクトです。
type ClassEmblem SortedMasterVO

// ClassEmblemBase はクラスエンブレムベースマスタの値オブジェクトです。
type ClassEmblemBase SortedMasterVO

// ClearLampType はクリアランプマスタの値オブジェクトです。
type ClearLampType SortedMasterVO

// ComboLampType はコンボランプマスタの値オブジェクトです。
type ComboLampType SortedMasterVO

// FullChainType はフルチェインランプマスタの値オブジェクトです。
type FullChainType SortedMasterVO

// Genre はジャンルマスタの値オブジェクトです。
type Genre BaseMasterVO

// HonorType は称号種類マスタの値オブジェクトです。
type HonorType BaseMasterVO

// Slot はプレイヤーレコードのスロット種別の値オブジェクトです。
type Slot BaseMasterVO
