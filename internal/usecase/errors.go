package usecase

import "errors"

var (
	ErrInvalidPlayerName = errors.New("invalid player name")

	ErrInvalidDifficulty     = errors.New("invalid difficulty")
	ErrChartNotFound         = errors.New("chart not found")
	ErrInvalidWorldsendInput = errors.New("invalid worldsend input")
	ErrInvalidHonorInput     = errors.New("invalid honor input")

	ErrAdminRequired      = errors.New("admin permission required")
	ErrInvalidAccountType = errors.New("invalid account type")

	ErrRecordFilterNotFound      = errors.New("record filter not found")
	ErrRecordFilterLimitExceeded = errors.New("record filter limit exceeded")
	ErrInvalidRecordFilterInput  = errors.New("invalid record filter input")
	ErrInvalidRecordFilterID     = errors.New("invalid record filter id")
)
