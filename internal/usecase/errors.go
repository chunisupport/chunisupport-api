package usecase

import "errors"

var (
	ErrInvalidPlayerName = errors.New("invalid player name")

	ErrInvalidDifficulty     = errors.New("invalid difficulty")
	ErrChartNotFound         = errors.New("chart not found")
	ErrInvalidWorldsendInput = errors.New("invalid worldsend input")

	ErrAdminRequired = errors.New("admin permission required")

	ErrAppVersionUnsupported = errors.New("unsupported app version")
)
