package models

import (
	"fmt"
	"time"
	"uuid"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
)

// ChartStatsBatchJobModel は譜面統計バッチジョブのデータベースモデルです。
// RequestedByUsername は users との JOIN で取得する読み取り専用の値です。
type ChartStatsBatchJobModel struct {
	ID                  []byte     `db:"id"`
	TriggerType         string     `db:"trigger_type"`
	Status              string     `db:"status"`
	RequestedByUserID   *int       `db:"requested_by_user_id"`
	RequestedByUsername *string    `db:"requested_by_username"`
	StartedAt           time.Time  `db:"started_at"`
	FinishedAt          *time.Time `db:"finished_at"`
	ErrorMessage        *string    `db:"error_message"`
}

// ToEntity は永続化モデルを不変条件が検証されたジョブへ変換します。
func (m *ChartStatsBatchJobModel) ToEntity() (*entity.ChartStatsBatchJob, error) {
	if len(m.ID) != len(uuid.UUID{}) {
		return nil, fmt.Errorf("invalid chart stats batch job id length: %d", len(m.ID))
	}
	var requester *entity.ChartStatsBatchJobRequester
	if m.RequestedByUserID != nil && m.RequestedByUsername != nil {
		name, err := username.NewUserName(*m.RequestedByUsername)
		if err != nil {
			return nil, err
		}
		requester = &entity.ChartStatsBatchJobRequester{UserID: *m.RequestedByUserID, Username: name}
	}
	var errorMessage string
	if m.ErrorMessage != nil {
		errorMessage = *m.ErrorMessage
	}
	return entity.ReconstructChartStatsBatchJob(
		uuid.UUID(m.ID),
		entity.ChartStatsBatchJobTrigger(m.TriggerType),
		requester,
		entity.ChartStatsBatchJobStatus(m.Status),
		m.StartedAt,
		m.FinishedAt,
		errorMessage,
	)
}

// FromChartStatsBatchJobEntity はジョブをデータベースモデルへ変換します。
func FromChartStatsBatchJobEntity(job *entity.ChartStatsBatchJob) *ChartStatsBatchJobModel {
	id := job.ID()
	model := &ChartStatsBatchJobModel{
		ID:          id[:],
		TriggerType: string(job.Trigger()),
		Status:      string(job.Status()),
		StartedAt:   job.StartedAt(),
		FinishedAt:  job.FinishedAt(),
	}
	if requester := job.Requester(); requester != nil {
		model.RequestedByUserID = &requester.UserID
	}
	if message := job.ErrorMessage(); message != "" {
		model.ErrorMessage = &message
	}
	return model
}
