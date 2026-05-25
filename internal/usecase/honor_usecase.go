package usecase

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// HonorInput は称号作成・更新の入力です。
type HonorInput struct {
	Name     string
	TypeName string
	ImageURL string
}

// HonorUsecase は管理者向け称号CRUDのユースケースです。
type HonorUsecase interface {
	ListHonors(ctx context.Context) ([]*entity.Honor, error)
	GetHonor(ctx context.Context, id int) (*entity.Honor, error)
	CreateHonor(ctx context.Context, input HonorInput) (*entity.Honor, error)
	UpdateHonor(ctx context.Context, id int, input HonorInput) (*entity.Honor, error)
	DeleteHonor(ctx context.Context, id int) error
}
