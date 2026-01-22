package masterdata

import (
	"time"

	domainmasterdata "github.com/Qman110101/chunisupport-api/internal/domain/masterdata"
)

// Item は単一のマスタ項目を表します。
type Item = domainmasterdata.Item

// Version はバージョンマスタ項目を表します。
type Version struct {
	ID         uint8     `db:"id"`
	Name       string    `db:"name"`
	ReleasedAt time.Time `db:"released_at"`
}
