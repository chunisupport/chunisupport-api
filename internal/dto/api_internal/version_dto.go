package api_internal

import (
	"encoding/json"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// CreateVersionRequest はバージョン作成リクエストです。
type CreateVersionRequest struct {
	Name       string `json:"name"`
	ReleasedAt string `json:"released_at"`
}

// RenameVersionRequest はバージョン改名リクエストです。
type RenameVersionRequest struct {
	Name       string          `json:"name"`
	ReleasedAt json.RawMessage `json:"released_at,omitempty"`
}

// VersionDTO は管理者向けバージョン情報です。
type VersionDTO struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	ReleasedAt string `json:"released_at"`
}

func ToVersionDTO(version *entity.Version) *VersionDTO {
	if version == nil {
		return nil
	}
	return &VersionDTO{ID: version.ID, Name: version.Name, ReleasedAt: version.ReleasedAt.Format(time.DateOnly)}
}

func ToVersionDTOs(versions []*entity.Version) []*VersionDTO {
	dtos := make([]*VersionDTO, len(versions))
	for i, version := range versions {
		dtos[i] = ToVersionDTO(version)
	}
	return dtos
}
