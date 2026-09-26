package api_internal

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
)

// StartSongBatchJobRequest は管理画面からの楽曲バッチ実行要求です。
type StartSongBatchJobRequest struct {
	Mode                   string `json:"mode"`
	FillMissingReleaseDate bool   `json:"fill_missing_release_date"`
}

// SongBatchJobDTO は楽曲バッチジョブの状態です。
// 要求者は内部のユーザーIDではなくユーザー名で返します。
type SongBatchJobDTO struct {
	ID                     string     `json:"id"`
	Mode                   string     `json:"mode"`
	FillMissingReleaseDate bool       `json:"fill_missing_release_date"`
	Trigger                string     `json:"trigger"`
	RequestedBy            *string    `json:"requested_by"`
	Status                 string     `json:"status"`
	StartedAt              time.Time  `json:"started_at"`
	FinishedAt             *time.Time `json:"finished_at"`
	WarningCount           int        `json:"warning_count"`
	ErrorMessage           *string    `json:"error_message"`
}

// SongBatchJobListDTO は楽曲バッチジョブの履歴です。
type SongBatchJobListDTO struct {
	Jobs []*SongBatchJobDTO `json:"jobs"`
}

// ToSongBatchJobDTO はジョブをレスポンス形式へ変換します。
func ToSongBatchJobDTO(job *entity.SongBatchJob) *SongBatchJobDTO {
	result := &SongBatchJobDTO{
		ID:                     job.ID().String(),
		Mode:                   string(job.Request().Mode),
		FillMissingReleaseDate: job.Request().FillMissingReleaseDate,
		Trigger:                string(job.Trigger()),
		Status:                 string(job.Status()),
		StartedAt:              job.StartedAt(),
		FinishedAt:             job.FinishedAt(),
		WarningCount:           job.WarningCount(),
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

// ToSongBatchJobListDTO はジョブ一覧をレスポンス形式へ変換します。
func ToSongBatchJobListDTO(jobs []*entity.SongBatchJob) *SongBatchJobListDTO {
	dtos := make([]*SongBatchJobDTO, len(jobs))
	for i, job := range jobs {
		dtos[i] = ToSongBatchJobDTO(job)
	}
	return &SongBatchJobListDTO{Jobs: dtos}
}
