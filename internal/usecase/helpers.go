package usecase

import "github.com/chunisupport/chunisupport-api/internal/info"

// normalizeIncludeDeleted は includeDeleted フラグを正規化します。
// requesterAccountTypeID が EDITOR 権限を満たさない場合は常に false を返します。
func normalizeIncludeDeleted(includeDeleted bool, requesterAccountTypeID *int) bool {
	if !includeDeleted {
		return false
	}

	if requesterAccountTypeID == nil || !info.HasRole(*requesterAccountTypeID, info.AccountTypeEditor) {
		return false
	}

	return true
}
