package usecase

import "errors"

var (
	ErrTemporaryPlayerDataNotFound = errors.New("temporary player data not found")
	ErrTempDataPerIPLimitExceeded  = errors.New("temporary player data per ip limit exceeded")
	ErrTempDataCapacityExceeded    = errors.New("temporary player data capacity exceeded")
	ErrUnauthorizedOperation       = errors.New("unauthorized operation")
)
