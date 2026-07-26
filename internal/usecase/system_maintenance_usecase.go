package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/maintenancecomment"
)

// SystemMaintenanceUsecase は現在状態の参照とADMINによる状態更新を扱います。
type SystemMaintenanceUsecase interface {
	MaintenanceStateProvider
	Update(
		ctx context.Context,
		actorUserID int,
		enabled bool,
		comment string,
	) (MaintenanceState, error)
}

type systemMaintenanceUsecase struct {
	repo          repository.SystemMaintenanceRepository
	clock         clock
	logger        *slog.Logger
	updateMu      sync.Mutex
	currentEntity *entity.SystemMaintenance
	currentState  atomic.Pointer[MaintenanceState]
}

// NewSystemMaintenanceUsecase はDBの永続状態を読み込み、読み取り用スナップショットを生成します。
// DBを正として起動するため、読み込みに失敗した場合は通常状態へフォールバックせずエラーを返します。
func NewSystemMaintenanceUsecase(
	ctx context.Context,
	repo repository.SystemMaintenanceRepository,
) (SystemMaintenanceUsecase, error) {
	return newSystemMaintenanceUsecase(ctx, repo, systemClock{}, slog.Default())
}

func newSystemMaintenanceUsecase(
	ctx context.Context,
	repo repository.SystemMaintenanceRepository,
	clock clock,
	logger *slog.Logger,
) (SystemMaintenanceUsecase, error) {
	if repo == nil {
		return nil, errors.Join(ErrInternalError, errors.New("system maintenance repository is nil"))
	}
	if clock == nil {
		return nil, errors.Join(ErrInternalError, errors.New("system maintenance clock is nil"))
	}
	if logger == nil {
		return nil, errors.Join(ErrInternalError, errors.New("system maintenance logger is nil"))
	}

	maintenance, err := repo.Find(ctx)
	if err != nil {
		return nil, fmt.Errorf("システムメンテナンス状態の読み込みに失敗しました: %w", err)
	}
	if maintenance == nil {
		return nil, errors.Join(ErrInternalError, errors.New("system maintenance repository returned nil entity"))
	}

	currentEntity := cloneSystemMaintenance(maintenance)
	currentState := maintenanceStateFromEntity(currentEntity)
	usecase := &systemMaintenanceUsecase{
		repo:          repo,
		clock:         clock,
		logger:        logger,
		currentEntity: currentEntity,
	}
	usecase.currentState.Store(&currentState)
	return usecase, nil
}

// Current はメモリ上の不変スナップショットをロックせず返します。
func (u *systemMaintenanceUsecase) Current() MaintenanceState {
	return *u.currentState.Load()
}

// Update は更新を直列化し、DB保存に成功した候補だけを新しいスナップショットとして公開します。
func (u *systemMaintenanceUsecase) Update(
	ctx context.Context,
	actorUserID int,
	enabled bool,
	comment string,
) (MaintenanceState, error) {
	u.updateMu.Lock()
	defer u.updateMu.Unlock()

	candidate := cloneSystemMaintenance(u.currentEntity)
	// DBのDATETIME(6)と公開中のスナップショットを一致させるため、
	// 保存前にUTCのマイクロ秒精度へ揃えます。
	now := u.clock.Now().UTC().Truncate(time.Microsecond)
	var err error
	if enabled {
		err = candidate.Enable(comment, actorUserID, now)
	} else {
		err = candidate.Disable(actorUserID, now)
	}
	if err != nil {
		if isInvalidMaintenanceCommentError(err) {
			return MaintenanceState{}, errors.Join(ErrInvalidMaintenanceComment, err)
		}
		return MaintenanceState{}, err
	}

	currentState := u.Current()
	candidateState := maintenanceStateFromEntity(candidate)
	if hasSamePublishedMaintenanceState(currentState, candidateState) {
		return currentState, nil
	}

	if err := u.repo.Save(ctx, candidate); err != nil {
		return MaintenanceState{}, err
	}

	u.currentEntity = cloneSystemMaintenance(candidate)
	u.currentState.Store(&candidateState)
	u.logger.InfoContext(
		ctx,
		"メンテナンス状態を更新しました",
		"actor_user_id",
		actorUserID,
		"enabled",
		candidateState.Enabled,
		"updated_at",
		candidateState.UpdatedAt,
	)
	return candidateState, nil
}

func maintenanceStateFromEntity(maintenance *entity.SystemMaintenance) MaintenanceState {
	return MaintenanceState{
		Enabled:   maintenance.IsEnabled(),
		Comment:   maintenance.Comment.String(),
		UpdatedAt: maintenance.UpdatedAt,
	}
}

func hasSamePublishedMaintenanceState(current MaintenanceState, candidate MaintenanceState) bool {
	return current.Enabled == candidate.Enabled && current.Comment == candidate.Comment
}

func isInvalidMaintenanceCommentError(err error) bool {
	return errors.Is(err, maintenancecomment.ErrRequired) ||
		errors.Is(err, maintenancecomment.ErrTooLong) ||
		errors.Is(err, maintenancecomment.ErrControlCharacter)
}

func cloneSystemMaintenance(source *entity.SystemMaintenance) *entity.SystemMaintenance {
	cloned := *source
	if source.UpdatedByUserID != nil {
		updaterUserID := *source.UpdatedByUserID
		cloned.UpdatedByUserID = &updaterUserID
	}
	return &cloned
}

var _ SystemMaintenanceUsecase = (*systemMaintenanceUsecase)(nil)
var _ MaintenanceStateProvider = (*systemMaintenanceUsecase)(nil)
