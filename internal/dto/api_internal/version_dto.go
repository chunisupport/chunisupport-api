package api_internal

import (
	"encoding/json"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// CreateVersionRequest はバージョン作成リクエストです。
type CreateVersionRequest struct {
	Name       string `json:"name"`
	ShortName  string `json:"short_name"`
	ReleasedAt string `json:"released_at"`
}

// RenameVersionRequest はバージョン改名リクエストです。
type RenameVersionRequest struct {
	Name       string          `json:"name"`
	ShortName  string          `json:"short_name"`
	ReleasedAt json.RawMessage `json:"released_at,omitempty"`
}

// VersionDTO は管理者向けバージョン情報です。
// id にはDBの数値主キーではなく公開ID（display_id）を返します。
type VersionDTO struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ShortName  string `json:"short_name"`
	ReleasedAt string `json:"released_at"`
}

func ToVersionDTO(version *entity.Version) *VersionDTO {
	if version == nil {
		return nil
	}
	return &VersionDTO{ID: version.DisplayID.String(), Name: version.Name, ShortName: version.ShortName, ReleasedAt: version.ReleasedAt.Format(time.DateOnly)}
}

func ToVersionDTOs(versions []*entity.Version) []*VersionDTO {
	dtos := make([]*VersionDTO, len(versions))
	for i, version := range versions {
		dtos[i] = ToVersionDTO(version)
	}
	return dtos
}
