package models

import (
	"fmt"
	"time"
	"uuid"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
)

// SongBatchJobModel は楽曲バッチジョブのデータベースモデルです。
// RequestedByUsername は users との JOIN で取得する読み取り専用の値です。
type SongBatchJobModel struct {
	ID                     []byte     `db:"id"`
	Mode                   string     `db:"mode"`
	FillMissingReleaseDate bool       `db:"fill_missing_release_date"`
	TriggerType            string     `db:"trigger_type"`
	Status                 string     `db:"status"`
	RequestedByUserID      *int       `db:"requested_by_user_id"`
	RequestedByUsername    *string    `db:"requested_by_username"`
	StartedAt              time.Time  `db:"started_at"`
	FinishedAt             *time.Time `db:"finished_at"`
	WarningCount           int        `db:"warning_count"`
	ErrorMessage           *string    `db:"error_message"`
}

// ToEntity は永続化モデルを不変条件が検証されたジョブへ変換します。
func (m *SongBatchJobModel) ToEntity() (*entity.SongBatchJob, error) {
	if len(m.ID) != len(uuid.UUID{}) {
		return nil, fmt.Errorf("invalid song batch job id length: %d", len(m.ID))
	}
	mode, err := songbatch.ParseRunMode(m.Mode)
	if err != nil {
		return nil, err
	}
	var requester *entity.SongBatchJobRequester
	if m.RequestedByUserID != nil && m.RequestedByUsername != nil {
		name, err := username.NewUserName(*m.RequestedByUsername)
		if err != nil {
			return nil, err
		}
		requester = &entity.SongBatchJobRequester{UserID: *m.RequestedByUserID, Username: name}
	}
	var errorMessage string
	if m.ErrorMessage != nil {
		errorMessage = *m.ErrorMessage
	}
	return entity.ReconstructSongBatchJob(
		uuid.UUID(m.ID),
		songbatch.RunRequest{Mode: mode, FillMissingReleaseDate: m.FillMissingReleaseDate},
		entity.SongBatchJobTrigger(m.TriggerType),
		requester,
		entity.SongBatchJobStatus(m.Status),
		m.StartedAt,
		m.FinishedAt,
		m.WarningCount,
		errorMessage,
	)
}

// FromSongBatchJobEntity はジョブをデータベースモデルへ変換します。
func FromSongBatchJobEntity(job *entity.SongBatchJob) *SongBatchJobModel {
	id := job.ID()
	model := &SongBatchJobModel{
		ID:                     id[:],
		Mode:                   string(job.Request().Mode),
		FillMissingReleaseDate: job.Request().FillMissingReleaseDate,
		TriggerType:            string(job.Trigger()),
		Status:                 string(job.Status()),
		StartedAt:              job.StartedAt(),
		FinishedAt:             job.FinishedAt(),
		WarningCount:           job.WarningCount(),
	}
	if requester := job.Requester(); requester != nil {
		model.RequestedByUserID = &requester.UserID
	}
	if message := job.ErrorMessage(); message != "" {
		model.ErrorMessage = &message
	}
	return model
}
