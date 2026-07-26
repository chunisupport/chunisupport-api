package entity

import (
	"errors"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/maintenancecomment"
)

// SystemMaintenanceID はメンテナンス状態を保持するsingleton集約の固定IDです。
const SystemMaintenanceID = 1

var (
	ErrSystemMaintenanceIDInvalid          = errors.New("system maintenance id is invalid")
	ErrSystemMaintenanceCommentRequired    = errors.New("enabled system maintenance requires a comment")
	ErrSystemMaintenanceCommentMustBeEmpty = errors.New("disabled system maintenance requires an empty comment")
	ErrSystemMaintenanceUpdaterIDInvalid   = errors.New("system maintenance updater user id is invalid")
	ErrSystemMaintenanceUpdatedAtRequired  = errors.New("system maintenance updated_at is required")
)

// SystemMaintenance はシステム全体のメンテナンス状態を表す単一の集約です。
type SystemMaintenance struct {
	ID              int
	Enabled         bool
	Comment         maintenancecomment.MaintenanceComment
	UpdatedByUserID *int
	UpdatedAt       time.Time
}

// RestoreSystemMaintenance は永続化済みデータから不変条件を満たす状態を復元します。
func RestoreSystemMaintenance(
	id int,
	enabled bool,
	comment string,
	updatedByUserID *int,
	updatedAt time.Time,
) (*SystemMaintenance, error) {
	if id != SystemMaintenanceID {
		return nil, ErrSystemMaintenanceIDInvalid
	}
	validatedComment, err := maintenancecomment.RestoreMaintenanceComment(comment)
	if err != nil {
		return nil, err
	}
	if enabled && validatedComment.IsEmpty() {
		return nil, ErrSystemMaintenanceCommentRequired
	}
	if !enabled && !validatedComment.IsEmpty() {
		return nil, ErrSystemMaintenanceCommentMustBeEmpty
	}
	if err := validateSystemMaintenanceUpdate(updatedByUserID, updatedAt); err != nil {
		return nil, err
	}

	return &SystemMaintenance{
		ID:              id,
		Enabled:         enabled,
		Comment:         validatedComment,
		UpdatedByUserID: cloneIntPointer(updatedByUserID),
		UpdatedAt:       updatedAt,
	}, nil
}

// Enable は検証済みコメントを設定してメンテナンスを開始します。
func (m *SystemMaintenance) Enable(comment string, updaterUserID int, now time.Time) error {
	validatedComment, err := maintenancecomment.NewMaintenanceComment(comment)
	if err != nil {
		return err
	}
	if err := validateSystemMaintenanceUpdate(&updaterUserID, now); err != nil {
		return err
	}

	m.Enabled = true
	m.Comment = validatedComment
	m.UpdatedByUserID = &updaterUserID
	m.UpdatedAt = now
	return nil
}

// Disable はコメントを空にしてメンテナンスを終了します。
func (m *SystemMaintenance) Disable(updaterUserID int, now time.Time) error {
	if err := validateSystemMaintenanceUpdate(&updaterUserID, now); err != nil {
		return err
	}
	emptyComment, err := maintenancecomment.RestoreMaintenanceComment("")
	if err != nil {
		return err
	}

	m.Enabled = false
	m.Comment = emptyComment
	m.UpdatedByUserID = &updaterUserID
	m.UpdatedAt = now
	return nil
}

// IsEnabled は現在メンテナンス中かを返します。
func (m *SystemMaintenance) IsEnabled() bool {
	return m.Enabled
}

func validateSystemMaintenanceUpdate(updatedByUserID *int, updatedAt time.Time) error {
	if updatedByUserID != nil && *updatedByUserID <= 0 {
		return ErrSystemMaintenanceUpdaterIDInvalid
	}
	if updatedAt.IsZero() {
		return ErrSystemMaintenanceUpdatedAtRequired
	}
	return nil
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
