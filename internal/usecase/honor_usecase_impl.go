package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

type honorUsecaseImpl struct {
	honorRepo       repository.HonorRepository
	masterProvider  repository.MasterDataMasterProvider
	tm              TransactionManager
	defaultExecutor repository.Executor
}

// NewHonorUsecase は新しい HonorUsecase を生成します。
func NewHonorUsecase(honorRepo repository.HonorRepository, masterProvider repository.MasterDataMasterProvider, tm TransactionManager, defaultExecutor repository.Executor) HonorUsecase {
	return &honorUsecaseImpl{
		honorRepo:       honorRepo,
		masterProvider:  masterProvider,
		tm:              tm,
		defaultExecutor: defaultExecutor,
	}
}

// ListHonors は称号を全件取得します。
func (u *honorUsecaseImpl) ListHonors(ctx context.Context) ([]*entity.Honor, error) {
	return u.honorRepo.FindAll(ctx, u.defaultExecutor)
}

// GetHonor は指定IDの称号を取得します。
func (u *honorUsecaseImpl) GetHonor(ctx context.Context, id int) (*entity.Honor, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidHonorInput)
	}
	return u.honorRepo.FindByID(ctx, u.defaultExecutor, id)
}

// CreateHonor は称号を新規登録します。
func (u *honorUsecaseImpl) CreateHonor(ctx context.Context, input HonorInput) (*entity.Honor, error) {
	honor, err := u.buildHonor(input)
	if err != nil {
		return nil, err
	}

	var created *entity.Honor
	if err := u.tm.Transactional(ctx, func(tx repository.Executor) error {
		result, err := u.honorRepo.Create(ctx, tx, honor)
		if err != nil {
			return err
		}
		created = result
		return nil
	}); err != nil {
		return nil, err
	}
	return created, nil
}

// UpdateHonor は称号を更新します。
func (u *honorUsecaseImpl) UpdateHonor(ctx context.Context, id int, input HonorInput) (*entity.Honor, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidHonorInput)
	}

	newHonor, err := u.buildHonor(input)
	if err != nil {
		return nil, err
	}

	var updated *entity.Honor
	if err := u.tm.Transactional(ctx, func(tx repository.Executor) error {
		current, err := u.honorRepo.FindByID(ctx, tx, id)
		if err != nil {
			return err
		}
		current.Rename(newHonor.Name)
		current.ChangeType(newHonor.HonorTypeID, newHonor.TypeName)
		current.ChangeImageURL(newHonor.ImageURL)

		if err := u.honorRepo.Save(ctx, tx, current); err != nil {
			return err
		}
		updated, err = u.honorRepo.FindByID(ctx, tx, id)
		return err
	}); err != nil {
		return nil, err
	}
	return updated, nil
}

// DeleteHonor は称号を物理削除します。
func (u *honorUsecaseImpl) DeleteHonor(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidHonorInput)
	}
	return u.tm.Transactional(ctx, func(tx repository.Executor) error {
		return u.honorRepo.Delete(ctx, tx, id)
	})
}

func (u *honorUsecaseImpl) buildHonor(input HonorInput) (*entity.Honor, error) {
	name := strings.TrimSpace(input.Name)
	typeName := strings.TrimSpace(input.TypeName)
	imageURL := strings.TrimSpace(input.ImageURL)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidHonorInput)
	}
	if typeName == "" {
		return nil, fmt.Errorf("%w: type_name is required", ErrInvalidHonorInput)
	}

	masters := u.masterProvider.MasterDataMasters()
	if masters == nil {
		return nil, fmt.Errorf("%w: master cache is not initialized", ErrInvalidHonorInput)
	}
	honorType, ok := masters.HonorTypes[typeName]
	if !ok {
		return nil, fmt.Errorf("%w: unknown type_name=%s", ErrInvalidHonorInput, typeName)
	}

	return entity.NewHonor(name, honorType.ID, honorType.Name, imageURL), nil
}
