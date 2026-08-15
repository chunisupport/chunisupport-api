package entity

import (
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/maintenancecomment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestoreSystemMaintenance(t *testing.T) {
	updatedAt := time.Date(2026, 7, 26, 12, 0, 0, 123000000, time.UTC)
	updaterID := 10

	tests := []struct {
		name            string
		id              int
		enabled         bool
		comment         string
		updatedByUserID *int
		updatedAt       time.Time
		wantErr         error
	}{
		{
			name:      "初期状態を復元できる",
			id:        SystemMaintenanceID,
			enabled:   false,
			comment:   "",
			updatedAt: updatedAt,
		},
		{
			name:            "有効状態を復元できる",
			id:              SystemMaintenanceID,
			enabled:         true,
			comment:         "更新中です",
			updatedByUserID: &updaterID,
			updatedAt:       updatedAt,
		},
		{
			name:      "singleton以外のIDを拒否する",
			id:        2,
			enabled:   false,
			comment:   "",
			updatedAt: updatedAt,
			wantErr:   ErrSystemMaintenanceIDInvalid,
		},
		{
			name:      "有効状態の空コメントを拒否する",
			id:        SystemMaintenanceID,
			enabled:   true,
			comment:   "",
			updatedAt: updatedAt,
			wantErr:   ErrSystemMaintenanceCommentRequired,
		},
		{
			name:      "無効状態の非空コメントを拒否する",
			id:        SystemMaintenanceID,
			enabled:   false,
			comment:   "更新中です",
			updatedAt: updatedAt,
			wantErr:   ErrSystemMaintenanceCommentMustBeEmpty,
		},
		{
			name:      "更新日時がない状態を拒否する",
			id:        SystemMaintenanceID,
			enabled:   false,
			comment:   "",
			updatedAt: time.Time{},
			wantErr:   ErrSystemMaintenanceUpdatedAtRequired,
		},
		{
			name:            "不正な更新者IDを拒否する",
			id:              SystemMaintenanceID,
			enabled:         false,
			comment:         "",
			updatedByUserID: new(int),
			updatedAt:       updatedAt,
			wantErr:         ErrSystemMaintenanceUpdaterIDInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			maintenance, err := RestoreSystemMaintenance(
				tt.id,
				tt.enabled,
				tt.comment,
				tt.updatedByUserID,
				tt.updatedAt,
			)

			// Then
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.id, maintenance.ID)
			assert.Equal(t, tt.enabled, maintenance.Enabled)
			assert.Equal(t, tt.comment, maintenance.Comment.String())
			assert.Equal(t, tt.updatedByUserID, maintenance.UpdatedByUserID)
			assert.Equal(t, tt.updatedAt, maintenance.UpdatedAt)
			assert.Equal(t, tt.enabled, maintenance.IsEnabled())
		})
	}
}

func TestSystemMaintenance_Enable(t *testing.T) {
	// Given
	previousAt := time.Date(2026, 7, 26, 11, 0, 0, 0, time.UTC)
	maintenance, err := RestoreSystemMaintenance(SystemMaintenanceID, false, "", nil, previousAt)
	require.NoError(t, err)
	updatedAt := previousAt.Add(time.Hour)

	// When
	err = maintenance.Enable(" 更新中\r\nしばらくお待ちください。 ", 10, updatedAt)

	// Then
	require.NoError(t, err)
	assert.True(t, maintenance.IsEnabled())
	assert.Equal(t, "更新中\nしばらくお待ちください。", maintenance.Comment.String())
	require.NotNil(t, maintenance.UpdatedByUserID)
	assert.Equal(t, 10, *maintenance.UpdatedByUserID)
	assert.Equal(t, updatedAt, maintenance.UpdatedAt)
}

func TestSystemMaintenance_Enable_不正なコメントでは状態を変更しない(t *testing.T) {
	// Given
	updatedAt := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	maintenance, err := RestoreSystemMaintenance(SystemMaintenanceID, false, "", nil, updatedAt)
	require.NoError(t, err)

	// When
	err = maintenance.Enable(" \r\n ", 10, updatedAt.Add(time.Hour))

	// Then
	assert.ErrorIs(t, err, maintenancecomment.ErrRequired)
	assert.False(t, maintenance.Enabled)
	assert.True(t, maintenance.Comment.IsEmpty())
	assert.Nil(t, maintenance.UpdatedByUserID)
	assert.Equal(t, updatedAt, maintenance.UpdatedAt)
}

func TestSystemMaintenance_Disable(t *testing.T) {
	// Given
	previousAt := time.Date(2026, 7, 26, 11, 0, 0, 0, time.UTC)
	previousUpdaterID := 10
	maintenance, err := RestoreSystemMaintenance(
		SystemMaintenanceID,
		true,
		"更新中です",
		&previousUpdaterID,
		previousAt,
	)
	require.NoError(t, err)
	updatedAt := previousAt.Add(time.Hour)

	// When
	err = maintenance.Disable(20, updatedAt)

	// Then
	require.NoError(t, err)
	assert.False(t, maintenance.IsEnabled())
	assert.True(t, maintenance.Comment.IsEmpty())
	require.NotNil(t, maintenance.UpdatedByUserID)
	assert.Equal(t, 20, *maintenance.UpdatedByUserID)
	assert.Equal(t, updatedAt, maintenance.UpdatedAt)
}

func TestSystemMaintenance_更新者と時刻の不正値では状態を変更しない(t *testing.T) {
	previousAt := time.Date(2026, 7, 26, 11, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		enable  bool
		updater int
		now     time.Time
		wantErr error
	}{
		{
			name:    "有効化で更新者IDが不正",
			enable:  true,
			updater: 0,
			now:     previousAt.Add(time.Hour),
			wantErr: ErrSystemMaintenanceUpdaterIDInvalid,
		},
		{
			name:    "有効化で更新日時が不正",
			enable:  true,
			updater: 10,
			now:     time.Time{},
			wantErr: ErrSystemMaintenanceUpdatedAtRequired,
		},
		{
			name:    "無効化で更新者IDが不正",
			enable:  false,
			updater: 0,
			now:     previousAt.Add(time.Hour),
			wantErr: ErrSystemMaintenanceUpdaterIDInvalid,
		},
		{
			name:    "無効化で更新日時が不正",
			enable:  false,
			updater: 10,
			now:     time.Time{},
			wantErr: ErrSystemMaintenanceUpdatedAtRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			var maintenance *SystemMaintenance
			var err error
			if tt.enable {
				maintenance, err = RestoreSystemMaintenance(SystemMaintenanceID, false, "", nil, previousAt)
			} else {
				updaterID := 5
				maintenance, err = RestoreSystemMaintenance(SystemMaintenanceID, true, "更新中です", &updaterID, previousAt)
			}
			require.NoError(t, err)
			before := *maintenance

			// When
			if tt.enable {
				err = maintenance.Enable("更新中です", tt.updater, tt.now)
			} else {
				err = maintenance.Disable(tt.updater, tt.now)
			}

			// Then
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, before, *maintenance)
		})
	}
}
