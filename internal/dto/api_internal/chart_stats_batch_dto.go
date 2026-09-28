package api_internal

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// ChartStatsBatchJobDTO は譜面統計バッチジョブの状態です。
// 要求者は内部のユーザーIDではなくユーザー名で返します。
type ChartStatsBatchJobDTO struct {
	ID           string     `json:"id"`
	Trigger      string     `json:"trigger"`
	RequestedBy  *string    `json:"requested_by"`
	Status       string     `json:"status"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	ErrorMessage *string    `json:"error_message"`
}

// ChartStatsBatchJobListDTO は譜面統計バッチジョブの履歴です。
type ChartStatsBatchJobListDTO struct {
	Jobs []*ChartStatsBatchJobDTO `json:"jobs"`
}

// ToChartStatsBatchJobDTO はジョブをレスポンス形式へ変換します。
func ToChartStatsBatchJobDTO(job *entity.ChartStatsBatchJob) *ChartStatsBatchJobDTO {
	result := &ChartStatsBatchJobDTO{
		ID:         job.ID().String(),
		Trigger:    string(job.Trigger()),
		Status:     string(job.Status()),
		StartedAt:  job.StartedAt(),
		FinishedAt: job.FinishedAt(),
	}
	if requester := job.Requester(); requester != nil {
		name := requester.Username.String()
		result.RequestedBy = &name
	}
	if message := job.ErrorMessage(); message != "" {
		result.ErrorMessage = &message
	}
	return result
}

// ToChartStatsBatchJobListDTO はジョブ一覧をレスポンス形式へ変換します。
func ToChartStatsBatchJobListDTO(jobs []*entity.ChartStatsBatchJob) *ChartStatsBatchJobListDTO {
	dtos := make([]*ChartStatsBatchJobDTO, len(jobs))
	for i, job := range jobs {
		dtos[i] = ToChartStatsBatchJobDTO(job)
	}
	return &ChartStatsBatchJobListDTO{Jobs: dtos}
}
