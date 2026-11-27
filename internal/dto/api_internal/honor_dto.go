package api_internal

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// HonorDTO は管理者向け称号情報DTOです。
type HonorDTO struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	TypeName  string     `json:"type_name"`
	ImageURL  string     `json:"image_url"`
	CreatedAt *time.Time `json:"created_at"`
}

// HonorsResponse は称号一覧レスポンスです。
type HonorsResponse struct {
	Honors []*HonorDTO `json:"honors"`
}

// HonorRequest は称号作成・更新リクエストです。
type HonorRequest struct {
	Name     string `json:"name" validate:"required,max=500"`
	TypeName string `json:"type_name" validate:"required"`
	ImageURL string `json:"image_url" validate:"max=255"`
}

// ToHonorDTO は称号エンティティをDTOに変換します。
func ToHonorDTO(honor *entity.Honor) *HonorDTO {
	if honor == nil {
		return nil
	}
	return &HonorDTO{
		ID:        honor.ID,
		Name:      honor.Name,
		TypeName:  honor.TypeName,
		ImageURL:  honor.ImageURL,
		CreatedAt: honor.CreatedAt,
	}
}

// ToHonorDTOs は称号エンティティのスライスをDTOに変換します。
func ToHonorDTOs(honors []*entity.Honor) []*HonorDTO {
	dtos := make([]*HonorDTO, len(honors))
	for i, honor := range honors {
		dtos[i] = ToHonorDTO(honor)
	}
	return dtos
}
