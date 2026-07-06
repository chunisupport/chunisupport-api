package api_internal

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/dto"
)

// UserSongRecordMetaDTO は楽曲単位レコードの更新日時を表します。
type UserSongRecordMetaDTO struct {
	UpdatedAt *time.Time `json:"updated_at"`
}

// UserSongRecordDTO は通常楽曲1曲分のレコードを表します。
type UserSongRecordDTO struct {
	Standard []*dto.PlayerRecordDTO `json:"standard"`
	Meta     *UserSongRecordMetaDTO `json:"meta"`
}

// UserWorldsendSongRecordDTO は WORLD'S END 楽曲1曲分のレコードを表します。
type UserWorldsendSongRecordDTO struct {
	Worldsend *dto.WorldsendRecordDTO `json:"worldsend"`
	Meta      *UserSongRecordMetaDTO  `json:"meta"`
}
