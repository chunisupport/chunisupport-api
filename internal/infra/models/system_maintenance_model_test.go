package models

import (
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemMaintenanceModel_ToEntity(t *testing.T) {
	// Given
	updatedAt := time.Date(2026, 7, 26, 12, 0, 0, 123000000, time.UTC)
	updaterID := 10
	model := &SystemMaintenanceModel{
		ID:              entity.SystemMaintenanceID,
		Enabled:         true,
		Comment:         "更新中です",
		UpdatedByUserID: &updaterID,
		UpdatedAt:       updatedAt,
	}

	// When
	maintenance, err := model.ToEntity()

	// Then
	require.NoError(t, err)
	assert.Equal(t, model.ID, maintenance.ID)
	assert.Equal(t, model.Enabled, maintenance.Enabled)
	assert.Equal(t, model.Comment, maintenance.Comment.String())
	assert.Equal(t, model.UpdatedByUserID, maintenance.UpdatedByUserID)
	assert.Equal(t, model.UpdatedAt, maintenance.UpdatedAt)
}

func TestSystemMaintenanceModel_ToEntity_不正な永続状態を拒否する(t *testing.T) {
	updatedAt := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		enabled bool
		comment string
		wantErr error
	}{
		{
			name:    "有効なのにコメントが空",
			enabled: true,
			comment: "",
			wantErr: entity.ErrSystemMaintenanceCommentRequired,
		},
		{
			name:    "無効なのにコメントがある",
			enabled: false,
			comment: "更新中です",
			wantErr: entity.ErrSystemMaintenanceCommentMustBeEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			model := &SystemMaintenanceModel{
				ID:        entity.SystemMaintenanceID,
				Enabled:   tt.enabled,
				Comment:   tt.comment,
				UpdatedAt: updatedAt,
			}

			// When
			_, err := model.ToEntity()

			// Then
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestFromSystemMaintenanceEntity(t *testing.T) {
	// Given
	updatedAt := time.Date(2026, 7, 26, 12, 0, 0, 123000000, time.UTC)
	updaterID := 10
	maintenance, err := entity.RestoreSystemMaintenance(
		entity.SystemMaintenanceID,
		true,
		"更新中です",
		&updaterID,
		updatedAt,
	)
	require.NoError(t, err)

	// When
	model := FromSystemMaintenanceEntity(maintenance)

	// Then
	assert.Equal(t, maintenance.ID, model.ID)
	assert.Equal(t, maintenance.Enabled, model.Enabled)
	assert.Equal(t, maintenance.Comment.String(), model.Comment)
	assert.Equal(t, maintenance.UpdatedByUserID, model.UpdatedByUserID)
	assert.Equal(t, maintenance.UpdatedAt, model.UpdatedAt)
}
