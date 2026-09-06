package constants

const (
	AccountTypePlayer = 1
	AccountTypeEditor = 2
	AccountTypeAdmin  = 3
	AccountTypeExtDev = 4
)

// IsKnownAccountType は権限IDの大小に依存せず、許可された権限だけを受け入れます。
func IsKnownAccountType(accountTypeID int) bool {
	switch accountTypeID {
	case AccountTypePlayer, AccountTypeEditor, AccountTypeAdmin, AccountTypeExtDev:
		return true
	default:
		return false
	}
}
