package usecase

import "errors"

var (
	ErrTemporaryPlayerDataNotFound = errors.New("temporary player data not found")
	// ErrTemporaryPlayerDataInProgress は同じトークンの確定処理が実行中であることを表します。
	ErrTemporaryPlayerDataInProgress = errors.New("temporary player data commit in progress")
	ErrTempDataPerIPLimitExceeded    = errors.New("temporary player data per ip limit exceeded")
	ErrTempDataCapacityExceeded      = errors.New("temporary player data capacity exceeded")
	ErrTempDataPayloadInvalidJSON    = errors.New("temporary player data payload is invalid json")
	ErrUnauthorizedOperation         = errors.New("unauthorized operation")
)
