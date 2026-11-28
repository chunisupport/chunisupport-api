package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
)

// AppAnnouncementUsecase は現在配信中のお知らせを提供します。
type AppAnnouncementUsecase interface {
	ListActive(ctx context.Context) ([]*entity.AppAnnouncement, error)
}

// AppAnnouncementInput は管理画面から登録・更新するお知らせの内容です。
type AppAnnouncementInput struct {
	Kind      string
	Title     string
	Body      *string
	TargetURL *string
	StartsAt  time.Time
	ExpiresAt time.Time
}

// AdminAppAnnouncementUsecase はADMIN専用のお知らせ管理を提供します。
type AdminAppAnnouncementUsecase interface {
	ListAll(ctx context.Context) ([]*entity.AppAnnouncement, error)
	Create(ctx context.Context, input AppAnnouncementInput) (*entity.AppAnnouncement, error)
	// Update は内容を上書きします。bumpRevision が true の場合は、閉じた利用者にも再表示させるため revision を1つ増やします。
	Update(ctx context.Context, id string, input AppAnnouncementInput, bumpRevision bool) (*entity.AppAnnouncement, error)
	Delete(ctx context.Context, id string) error
}

type appAnnouncementUsecase struct {
	repo  repository.AppAnnouncementRepository
	tm    TransactionManager
	clock clock
}

// NewAppAnnouncementUsecase はAPIサーバー時刻を基準に配信判定するユースケースを生成します。
func NewAppAnnouncementUsecase(repo repository.AppAnnouncementRepository) AppAnnouncementUsecase {
	return &appAnnouncementUsecase{repo: repo, clock: systemClock{}}
}

// NewAdminAppAnnouncementUsecase はADMIN専用のお知らせ管理ユースケースを生成します。
func NewAdminAppAnnouncementUsecase(repo repository.AppAnnouncementRepository, tm TransactionManager) AdminAppAnnouncementUsecase {
	return &appAnnouncementUsecase{repo: repo, tm: tm}
}

// ListActive は配信有効かつ表示期間内のお知らせを、開始日時の新しい順に返します。
func (u *appAnnouncementUsecase) ListActive(ctx context.Context) ([]*entity.AppAnnouncement, error) {
	now := u.clock.Now()
	announcements, err := u.repo.FindUnexpired(ctx, now)
	if err != nil {
		return nil, err
	}
	active := make([]*entity.AppAnnouncement, 0, len(announcements))
	for _, announcement := range announcements {
		if announcement.IsActiveAt(now) {
			active = append(active, announcement)
		}
	}
	return active, nil
}

// ListAll は終了済み・開始前を含むすべてのお知らせを返します。
func (u *appAnnouncementUsecase) ListAll(ctx context.Context) ([]*entity.AppAnnouncement, error) {
	return u.repo.FindAll(ctx)
}

// Create はIDを乱数で採番し、revision 1 のお知らせを追加します。
func (u *appAnnouncementUsecase) Create(ctx context.Context, input AppAnnouncementInput) (*entity.AppAnnouncement, error) {
	id, err := generateDisplayID()
	if err != nil {
		return nil, err
	}
	announcement, err := restoreAppAnnouncementInput(id, input, 1)
	if err != nil {
		return nil, err
	}
	if err := u.repo.Create(ctx, announcement); err != nil {
		return nil, err
	}
	return announcement, nil
}

// Update は既存のお知らせを上書きします。
func (u *appAnnouncementUsecase) Update(ctx context.Context, id string, input AppAnnouncementInput, bumpRevision bool) (*entity.AppAnnouncement, error) {
	displayID, err := parseAppAnnouncementID(id)
	if err != nil {
		return nil, err
	}
	var announcement *entity.AppAnnouncement
	err = u.tm.Transactional(ctx, func(tx repository.Executor) error {
		current, err := u.repo.FindByDisplayIDForUpdate(ctx, tx, displayID)
		if err != nil {
			return err
		}
		revision := current.Revision
		if bumpRevision {
			revision++
		}
		announcement, err = restoreAppAnnouncementInput(displayID.String(), input, revision)
		if err != nil {
			return err
		}
		return u.repo.Update(ctx, tx, announcement)
	})
	if err != nil {
		return nil, err
	}
	return announcement, nil
}

// Delete はお知らせを削除します。配信中なら即時に停止します。
func (u *appAnnouncementUsecase) Delete(ctx context.Context, id string) error {
	displayID, err := parseAppAnnouncementID(id)
	if err != nil {
		return err
	}
	return u.repo.Delete(ctx, displayID)
}

// parseAppAnnouncementID は形式の不正なIDを、存在しないお知らせとして扱います。
func parseAppAnnouncementID(id string) (displayid.DisplayID, error) {
	displayID, err := displayid.NewDisplayID(id)
	if err != nil {
		return "", fmt.Errorf("%w: %w", repository.ErrAppAnnouncementNotFound, err)
	}
	return displayID, nil
}

// restoreAppAnnouncementInput は入力を検証し、検証エラーを入力エラーとして包みます。
// id は採番済みまたは検証済みのため、ここでのエラーはすべて入力の誤りです。
func restoreAppAnnouncementInput(id string, input AppAnnouncementInput, revision int) (*entity.AppAnnouncement, error) {
	announcement, err := entity.RestoreAppAnnouncement(id, input.Kind, input.Title, input.Body, input.TargetURL, input.StartsAt, input.ExpiresAt, revision)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidAppAnnouncementInput, err)
	}
	return announcement, nil
}
