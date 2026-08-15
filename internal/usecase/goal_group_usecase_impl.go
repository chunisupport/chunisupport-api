package usecase

import (
	"context"
	"errors"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

var (
	ErrGoalGroupNotFound      = errors.New("goal group not found")
	ErrGoalGroupLimitExceeded = errors.New("goal group limit exceeded")
	ErrInvalidGoalGroupName   = errors.New("invalid goal group name")
	ErrGoalGroupConflict      = errors.New("goal group conflict")
	ErrInvalidGoalGroupOrder  = errors.New("invalid goal group order")
)

type goalGroupUsecase struct {
	db        repository.Executor
	tm        TransactionManager
	groupRepo repository.GoalGroupRepository
	goalRepo  repository.GoalRepository
}

// NewGoalGroupUsecase は目標グループユースケースを生成します。
func NewGoalGroupUsecase(db repository.Executor, tm TransactionManager, groupRepo repository.GoalGroupRepository, goalRepo repository.GoalRepository) GoalGroupUsecase {
	return &goalGroupUsecase{db: db, tm: tm, groupRepo: groupRepo, goalRepo: goalRepo}
}

func (u *goalGroupUsecase) List(ctx context.Context, userID int) ([]*GoalGroupOutput, error) {
	groups, err := u.groupRepo.ListByUserID(ctx, u.db, userID)
	if err != nil {
		return nil, err
	}
	return toGoalGroupOutputs(groups), nil
}

func (u *goalGroupUsecase) Create(ctx context.Context, userID int, name string) (*GoalGroupOutput, error) {
	group, err := entity.NewGoalGroup(userID, name)
	if err != nil {
		return nil, ErrInvalidGoalGroupName
	}
	err = u.tm.Transactional(ctx, func(tx repository.Executor) error {
		if err := u.goalRepo.LockUserByID(ctx, tx, userID); err != nil {
			return err
		}
		count, err := u.groupRepo.CountByUserID(ctx, tx, userID)
		if err != nil {
			return err
		}
		if count >= info.GoalGroupMaxPerUser {
			return ErrGoalGroupLimitExceeded
		}
		// ユーザー単位の上限を直前で検証しているため、uint16の範囲を超えません。
		group.SortOrder = uint16(count + 1) // #nosec G115
		if err := u.groupRepo.Save(ctx, tx, group); err != nil {
			if errors.Is(err, repository.ErrGoalGroupConflict) {
				return ErrGoalGroupConflict
			}
			return err
		}
		created, err := u.groupRepo.FindByIDAndUserID(ctx, tx, group.ID, userID)
		if err != nil {
			return err
		}
		group = created
		return nil
	})
	if err != nil {
		return nil, err
	}
	return toGoalGroupOutput(group), nil
}

func (u *goalGroupUsecase) Update(ctx context.Context, userID int, id uint32, name string) (*GoalGroupOutput, error) {
	var group *entity.GoalGroup
	err := u.tm.Transactional(ctx, func(tx repository.Executor) error {
		if err := u.goalRepo.LockUserByID(ctx, tx, userID); err != nil {
			return err
		}
		found, err := u.groupRepo.FindByIDAndUserID(ctx, tx, id, userID)
		if errors.Is(err, repository.ErrGoalGroupNotFound) {
			return ErrGoalGroupNotFound
		}
		if err != nil {
			return err
		}
		if err := found.Rename(name); err != nil {
			return ErrInvalidGoalGroupName
		}
		if err := u.groupRepo.Save(ctx, tx, found); err != nil {
			if errors.Is(err, repository.ErrGoalGroupConflict) {
				return ErrGoalGroupConflict
			}
			return err
		}
		group = found
		return nil
	})
	if err != nil {
		return nil, err
	}
	return toGoalGroupOutput(group), nil
}

func (u *goalGroupUsecase) Delete(ctx context.Context, userID int, id uint32) error {
	return u.tm.Transactional(ctx, func(tx repository.Executor) error {
		if err := u.goalRepo.LockUserByID(ctx, tx, userID); err != nil {
			return err
		}
		groups, err := u.groupRepo.ListByUserID(ctx, tx, userID)
		if err != nil {
			return err
		}
		groupOrder, err := entity.NewGoalGroupOrder(userID, groups)
		if err != nil {
			return ErrInternalError
		}
		if err := groupOrder.Remove(id); errors.Is(err, entity.ErrGoalGroupOrderMissing) {
			return ErrGoalGroupNotFound
		} else if err != nil {
			return ErrInternalError
		}
		goals, err := u.goalRepo.ListByUserID(ctx, tx, userID)
		if err != nil {
			return err
		}
		arrangement, err := entity.NewGoalArrangement(userID, goals)
		if err != nil {
			return ErrInternalError
		}
		arrangement.RemoveGroup(id)
		if err := u.goalRepo.SaveGoalArrangement(ctx, tx, arrangement); err != nil {
			return err
		}
		if err := u.groupRepo.DeleteByIDAndUserID(ctx, tx, id, userID); err != nil {
			if errors.Is(err, repository.ErrGoalGroupNotFound) {
				return ErrGoalGroupNotFound
			}
			return err
		}
		return u.groupRepo.SaveOrder(ctx, tx, groupOrder)
	})
}

func (u *goalGroupUsecase) Reorder(ctx context.Context, userID int, orderedGroupIDs []uint32) error {
	return u.tm.Transactional(ctx, func(tx repository.Executor) error {
		if err := u.goalRepo.LockUserByID(ctx, tx, userID); err != nil {
			return err
		}
		groups, err := u.groupRepo.ListByUserID(ctx, tx, userID)
		if err != nil {
			return err
		}
		order, err := entity.NewGoalGroupOrder(userID, groups)
		if err != nil {
			return ErrInternalError
		}
		if err := order.Reorder(orderedGroupIDs); err != nil {
			return ErrInvalidGoalGroupOrder
		}
		return u.groupRepo.SaveOrder(ctx, tx, order)
	})
}

func toGoalGroupOutputs(groups []*entity.GoalGroup) []*GoalGroupOutput {
	outputs := make([]*GoalGroupOutput, 0, len(groups))
	for _, group := range groups {
		outputs = append(outputs, toGoalGroupOutput(group))
	}
	return outputs
}

func toGoalGroupOutput(group *entity.GoalGroup) *GoalGroupOutput {
	return &GoalGroupOutput{ID: group.ID, Name: group.Name.String(), SortOrder: group.SortOrder, CreatedAt: group.CreatedAt}
}
