package usecase

import "errors"

var (
	ErrInvalidPlayerName = errors.New("invalid player name")

	ErrInvalidDifficulty            = errors.New("invalid difficulty")
	ErrInvalidRatingBand            = errors.New("invalid rating band")
	ErrInvalidBestSlotRankingCursor = errors.New("invalid best-slot ranking cursor")
	ErrChartNotFound                = errors.New("chart not found")
	ErrInvalidWorldsendInput        = errors.New("invalid worldsend input")
	ErrInvalidHonorInput            = errors.New("invalid honor input")
	ErrInvalidVersionInput          = errors.New("invalid version input")
	ErrVersionNotLatest             = errors.New("version not latest")
	ErrVersionInUse                 = errors.New("version in use")

	ErrAdminRequired = errors.New("admin permission required")
	// ErrMaintenanceMode はメンテナンス中にスタッフ以外の利用を拒否した場合に返します。
	ErrMaintenanceMode = errors.New("maintenance mode")
	// ErrInvalidMaintenanceComment はメンテナンスコメントが公開仕様を満たさない場合に返します。
	ErrInvalidMaintenanceComment = errors.New("invalid maintenance comment")

	ErrRecordFilterNotFound      = errors.New("record filter not found")
	ErrRecordFilterLimitExceeded = errors.New("record filter limit exceeded")
	ErrInvalidRecordFilterInput  = errors.New("invalid record filter input")
	ErrInvalidRecordFilterID     = errors.New("invalid record filter id")

	ErrPlayerFavoriteSongLimitExceeded = errors.New("player favorite song limit exceeded")
	ErrPlayerLatestUpdateNotFound      = errors.New("player latest update not found")
)
