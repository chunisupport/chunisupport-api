package repository

import (
	"context"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
)

// AppAnnouncementRepository は期間限定のお知らせを永続化します。
type AppAnnouncementRepository interface {
	// FindUnexpired は now 時点で終了していないお知らせを、開始日時の新しい順（同時刻は id 順）に返します。
	// 開始前のものも含めるのは、登録直後に不正な行を検出できるようにするためです。
	FindUnexpired(ctx context.Context, now time.Time) ([]*entity.AppAnnouncement, error)
	// FindAll は終了済み・開始前を含むすべてのお知らせを、開始日時の新しい順（同時刻は id の降順）に返します。
	FindAll(ctx context.Context) ([]*entity.AppAnnouncement, error)
	// FindByDisplayIDForUpdate は並行更新で revision が巻き戻らないよう、対象行をロックして取得します。
	FindByDisplayIDForUpdate(ctx context.Context, exec Executor, displayID displayid.DisplayID) (*entity.AppAnnouncement, error)
	// Create はお知らせを追加します。
	Create(ctx context.Context, announcement *entity.AppAnnouncement) error
	// Update はお知らせの内容を上書きします。存在しない場合は ErrAppAnnouncementNotFound を返します。
	Update(ctx context.Context, exec Executor, announcement *entity.AppAnnouncement) error
	// Delete はお知らせを削除します。存在しない場合は ErrAppAnnouncementNotFound を返します。
	Delete(ctx context.Context, displayID displayid.DisplayID) error
}
