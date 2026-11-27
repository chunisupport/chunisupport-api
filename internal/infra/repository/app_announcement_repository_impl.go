package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
	"github.com/chunisupport/chunisupport-api/internal/infra/models"
	"github.com/jmoiron/sqlx"
)

type appAnnouncementRepository struct {
	db *sqlx.DB
}

// NewAppAnnouncementRepository はお知らせリポジトリを生成します。
func NewAppAnnouncementRepository(db *sqlx.DB) domainrepo.AppAnnouncementRepository {
	return &appAnnouncementRepository{db: db}
}

const selectAppAnnouncementColumns = `
SELECT id, kind, title, body, target_url, starts_at, expires_at, revision
FROM app_announcements`

// FindUnexpired は expires_at のインデックスで終了前の行だけを取得します。
// 不正な行を黙って除外すると登録ミスに気付けないため、検証エラーはそのまま返します。
func (r *appAnnouncementRepository) FindUnexpired(ctx context.Context, now time.Time) ([]*entity.AppAnnouncement, error) {
	var rows []models.AppAnnouncementModel
	if err := r.db.SelectContext(ctx, &rows, selectAppAnnouncementColumns+`
WHERE expires_at > ?
ORDER BY starts_at DESC, id DESC
`, now.UTC()); err != nil {
		return nil, fmt.Errorf("%w: find unexpired app announcements: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}
	return restoreAppAnnouncements(rows)
}

// FindAll は管理画面向けに、終了済み・開始前を含むすべての行を取得します。
func (r *appAnnouncementRepository) FindAll(ctx context.Context) ([]*entity.AppAnnouncement, error) {
	var rows []models.AppAnnouncementModel
	if err := r.db.SelectContext(ctx, &rows, selectAppAnnouncementColumns+`
ORDER BY starts_at DESC, id DESC
`); err != nil {
		return nil, fmt.Errorf("%w: find all app announcements: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}
	return restoreAppAnnouncements(rows)
}

// FindByDisplayID は主キーで1件を取得します。
func (r *appAnnouncementRepository) FindByDisplayID(ctx context.Context, displayID displayid.DisplayID) (*entity.AppAnnouncement, error) {
	var row models.AppAnnouncementModel
	if err := r.db.GetContext(ctx, &row, selectAppAnnouncementColumns+`
WHERE id = ?
`, displayID.String()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domainrepo.ErrAppAnnouncementNotFound
		}
		return nil, fmt.Errorf("%w: find app announcement: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}
	announcement, err := row.ToEntity()
	if err != nil {
		return nil, fmt.Errorf("%w: restore app announcement %s: %w", domainrepo.ErrRepositoryOperationFailed, row.ID, err)
	}
	return announcement, nil
}

// Create はお知らせを1行追加します。主キーは乱数のためAUTO_INCREMENTを消費しません。
func (r *appAnnouncementRepository) Create(ctx context.Context, announcement *entity.AppAnnouncement) error {
	if _, err := r.db.NamedExecContext(ctx, `
INSERT INTO app_announcements (id, kind, title, body, target_url, starts_at, expires_at, revision)
VALUES (:id, :kind, :title, :body, :target_url, :starts_at, :expires_at, :revision)
`, models.FromAppAnnouncementEntity(announcement)); err != nil {
		return fmt.Errorf("%w: create app announcement: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}
	return nil
}

// Update はお知らせの内容を上書きします。
// DSNの clientFoundRows=true により値が変わらない更新でも一致行数が返るため、0件を存在しないと判定します。
func (r *appAnnouncementRepository) Update(ctx context.Context, announcement *entity.AppAnnouncement) error {
	result, err := r.db.NamedExecContext(ctx, `
UPDATE app_announcements
SET kind = :kind, title = :title, body = :body, target_url = :target_url,
    starts_at = :starts_at, expires_at = :expires_at, revision = :revision
WHERE id = :id
`, models.FromAppAnnouncementEntity(announcement))
	if err != nil {
		return fmt.Errorf("%w: update app announcement: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}
	return requireAppAnnouncementAffected(result, "update")
}

// Delete はお知らせを削除します。配信停止もこの操作で行います。
func (r *appAnnouncementRepository) Delete(ctx context.Context, displayID displayid.DisplayID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM app_announcements WHERE id = ?`, displayID.String())
	if err != nil {
		return fmt.Errorf("%w: delete app announcement: %w", domainrepo.ErrRepositoryOperationFailed, err)
	}
	return requireAppAnnouncementAffected(result, "delete")
}

// requireAppAnnouncementAffected は対象の行がなかった場合に ErrAppAnnouncementNotFound を返します。
func requireAppAnnouncementAffected(result sql.Result, operation string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: %s app announcement rows affected: %w", domainrepo.ErrRepositoryOperationFailed, operation, err)
	}
	if affected == 0 {
		return domainrepo.ErrAppAnnouncementNotFound
	}
	return nil
}

// restoreAppAnnouncements は取得した行を検証して復元します。
func restoreAppAnnouncements(rows []models.AppAnnouncementModel) ([]*entity.AppAnnouncement, error) {
	announcements := make([]*entity.AppAnnouncement, 0, len(rows))
	for i := range rows {
		announcement, err := rows[i].ToEntity()
		if err != nil {
			return nil, fmt.Errorf("%w: restore app announcement %s: %w", domainrepo.ErrRepositoryOperationFailed, rows[i].ID, err)
		}
		announcements = append(announcements, announcement)
	}
	return announcements, nil
}

var _ domainrepo.AppAnnouncementRepository = (*appAnnouncementRepository)(nil)
