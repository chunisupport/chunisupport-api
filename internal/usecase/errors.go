package usecase

import (
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

var (
	ErrInvalidPlayerName = errors.New("invalid player name")

	ErrInvalidDifficulty     = errors.New("invalid difficulty")
	ErrChartNotFound         = errors.New("chart not found")
	ErrInvalidWorldsendInput = errors.New("invalid worldsend input")
	ErrInvalidHonorInput     = errors.New("invalid honor input")

	ErrAdminRequired = errors.New("admin permission required")
	// ErrInvalidAccountType はドメイン層の権限種別エラーと同一インスタンスに揃えます。
	// ユースケース境界で再公開する名前を残しつつ、errors.Is によるAPIエラー変換が
	// ドメイン由来のエラーでも同じ結果になるようにします。
	ErrInvalidAccountType = entity.ErrInvalidAccountType
	ErrLastAdminRequired  = errors.New("last admin must remain")

	ErrRecordFilterNotFound      = errors.New("record filter not found")
	ErrRecordFilterLimitExceeded = errors.New("record filter limit exceeded")
	ErrInvalidRecordFilterInput  = errors.New("invalid record filter input")
	ErrInvalidRecordFilterID     = errors.New("invalid record filter id")
)
