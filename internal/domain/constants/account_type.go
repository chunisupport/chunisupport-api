package constants

const (
	// AccountTypePlayer は一般ユーザー権限を表します。
	AccountTypePlayer = 1
	// AccountTypeEditor は編集者権限を表します。
	AccountTypeEditor = 2
	// AccountTypeAdmin は管理者権限を表します。
	AccountTypeAdmin = 3
)

// IsKnownAccountType は account_type_id がドメイン上で扱える既知ロールかを判定します。
func IsKnownAccountType(accountTypeID int) bool {
	switch accountTypeID {
	case AccountTypePlayer, AccountTypeEditor, AccountTypeAdmin:
		return true
	default:
		return false
	}
}
