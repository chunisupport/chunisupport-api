package usecase

import "errors"

var (
	ErrInvalidPlayerName = errors.New("invalid player name")

	ErrInvalidDifficulty = errors.New("invalid difficulty")
	ErrChartNotFound     = errors.New("chart not found")

	ErrAdminRequired = errors.New("admin permission required")

	ErrAppVersionUnsupported = errors.New("unsupported app version")
)
