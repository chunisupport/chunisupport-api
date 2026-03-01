package usecase

// AccountTypeProvider はアカウント種別名の参照に必要な最小境界インターフェースです。
type AccountTypeProvider interface {
	GetAccountTypeNameByID(id int) string
}
