package entity

import (
	"errors"
	"strings"
	"time"
	"uuid"

	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
)

// SongBatchJobErrorMessageMaxLength は保存する失敗理由の最大文字数です。
// 外部サービスのエラー全文で行が肥大化しないよう、管理画面で状況を把握できる長さに制限します。
const SongBatchJobErrorMessageMaxLength = 1000

var (
	// ErrInvalidSongBatchJob は楽曲バッチジョブの不変条件を満たさないことを表します。
	ErrInvalidSongBatchJob = errors.New("invalid song batch job")
	// ErrSongBatchJobAlreadyFinished は終了済みのジョブを再度終了しようとしたことを表します。
	ErrSongBatchJobAlreadyFinished = errors.New("song batch job already finished")
)

// SongBatchJobStatus は楽曲バッチジョブの状態です。
type SongBatchJobStatus string

const (
	// SongBatchJobStatusRunning は実行中です。
	SongBatchJobStatusRunning SongBatchJobStatus = "RUNNING"
	// SongBatchJobStatusSucceeded は全データソースを利用して成功したことを表します。
	SongBatchJobStatusSucceeded SongBatchJobStatus = "SUCCEEDED"
	// SongBatchJobStatusSucceededWithWarnings は補完データソースを除外して成功したことを表します。
	SongBatchJobStatusSucceededWithWarnings SongBatchJobStatus = "SUCCEEDED_WITH_WARNINGS"
	// SongBatchJobStatusFailed は処理を完了できず、MySQL を更新しなかったことを表します。
	SongBatchJobStatusFailed SongBatchJobStatus = "FAILED"
	// SongBatchJobStatusInterrupted はプロセス停止などで処理が中断されたことを表します。
	// MySQL への同期は単一トランザクションのため、中断時はロールバック済みです。
	SongBatchJobStatusInterrupted SongBatchJobStatus = "INTERRUPTED"
)

func (s SongBatchJobStatus) isValid() bool {
	switch s {
	case SongBatchJobStatusRunning, SongBatchJobStatusSucceeded, SongBatchJobStatusSucceededWithWarnings, SongBatchJobStatusFailed, SongBatchJobStatusInterrupted:
		return true
	default:
		return false
	}
}

// SongBatchJobTrigger は楽曲バッチジョブの起動元です。
type SongBatchJobTrigger string

const (
	// SongBatchJobTriggerCLI は cron などから CLI で起動したことを表します。
	SongBatchJobTriggerCLI SongBatchJobTrigger = "CLI"
	// SongBatchJobTriggerAdmin は管理画面から起動したことを表します。
	SongBatchJobTriggerAdmin SongBatchJobTrigger = "ADMIN"
)

func (t SongBatchJobTrigger) isValid() bool {
	return t == SongBatchJobTriggerCLI || t == SongBatchJobTriggerAdmin
}

// SongBatchJobRequester は管理画面からジョブを起動したユーザーです。
type SongBatchJobRequester struct {
	UserID   int
	Username username.UserName
}

// SongBatchJob は楽曲バッチ1回分の実行を表す集約です。
type SongBatchJob struct {
	id           uuid.UUID
	request      songbatch.RunRequest
	trigger      SongBatchJobTrigger
	requester    *SongBatchJobRequester
	status       SongBatchJobStatus
	startedAt    time.Time
	finishedAt   *time.Time
	warningCount int
	errorMessage string
}

// StartSongBatchJobFromCLI は CLI から起動した実行中のジョブを生成します。
func StartSongBatchJobFromCLI(id uuid.UUID, request songbatch.RunRequest, startedAt time.Time) *SongBatchJob {
	return &SongBatchJob{
		id:        id,
		request:   request,
		trigger:   SongBatchJobTriggerCLI,
		status:    SongBatchJobStatusRunning,
		startedAt: startedAt,
	}
}

// StartSongBatchJobFromAdmin は管理画面から起動した実行中のジョブを生成します。
func StartSongBatchJobFromAdmin(id uuid.UUID, request songbatch.RunRequest, requester SongBatchJobRequester, startedAt time.Time) *SongBatchJob {
	return &SongBatchJob{
		id:        id,
		request:   request,
		trigger:   SongBatchJobTriggerAdmin,
		requester: &requester,
		status:    SongBatchJobStatusRunning,
		startedAt: startedAt,
	}
}

// ReconstructSongBatchJob は永続化済みデータから不変条件を満たすジョブを復元します。
// 要求者のユーザーが削除された場合、管理画面ジョブでも requester は nil になります。
func ReconstructSongBatchJob(
	id uuid.UUID,
	request songbatch.RunRequest,
	trigger SongBatchJobTrigger,
	requester *SongBatchJobRequester,
	status SongBatchJobStatus,
	startedAt time.Time,
	finishedAt *time.Time,
	warningCount int,
	errorMessage string,
) (*SongBatchJob, error) {
	if !trigger.isValid() || !status.isValid() || warningCount < 0 {
		return nil, ErrInvalidSongBatchJob
	}
	if (status == SongBatchJobStatusRunning) != (finishedAt == nil) {
		return nil, ErrInvalidSongBatchJob
	}
	return &SongBatchJob{
		id:           id,
		request:      request,
		trigger:      trigger,
		requester:    requester,
		status:       status,
		startedAt:    startedAt,
		finishedAt:   finishedAt,
		warningCount: warningCount,
		errorMessage: errorMessage,
	}, nil
}

// ID はジョブIDを返します。
func (j *SongBatchJob) ID() uuid.UUID { return j.id }

// Request は実行条件を返します。
func (j *SongBatchJob) Request() songbatch.RunRequest { return j.request }

// Trigger は起動元を返します。
func (j *SongBatchJob) Trigger() SongBatchJobTrigger { return j.trigger }

// Requester は管理画面から起動したユーザーを返します。CLI 起動や要求者削除後は nil です。
func (j *SongBatchJob) Requester() *SongBatchJobRequester { return j.requester }

// Status は状態を返します。
func (j *SongBatchJob) Status() SongBatchJobStatus { return j.status }

// StartedAt は開始日時を返します。
func (j *SongBatchJob) StartedAt() time.Time { return j.startedAt }

// FinishedAt は終了日時を返します。実行中は nil です。
func (j *SongBatchJob) FinishedAt() *time.Time { return j.finishedAt }

// WarningCount は除外した補完データソースの件数を返します。
func (j *SongBatchJob) WarningCount() int { return j.warningCount }

// ErrorMessage は失敗理由を返します。失敗以外では空文字です。
func (j *SongBatchJob) ErrorMessage() string { return j.errorMessage }

// IsRunning は実行中かどうかを返します。
func (j *SongBatchJob) IsRunning() bool { return j.status == SongBatchJobStatusRunning }

// Complete は処理の完了を記録します。補完データソースを除外した場合は警告付き成功になります。
func (j *SongBatchJob) Complete(warningCount int, finishedAt time.Time) error {
	if warningCount < 0 {
		return ErrInvalidSongBatchJob
	}
	status := SongBatchJobStatusSucceeded
	if warningCount > 0 {
		status = SongBatchJobStatusSucceededWithWarnings
	}
	return j.finish(status, warningCount, "", finishedAt)
}

// Fail は処理の失敗を記録します。失敗理由は上限文字数で切り詰めます。
func (j *SongBatchJob) Fail(warningCount int, message string, finishedAt time.Time) error {
	message = strings.TrimSpace(message)
	if warningCount < 0 || message == "" {
		return ErrInvalidSongBatchJob
	}
	if runes := []rune(message); len(runes) > SongBatchJobErrorMessageMaxLength {
		message = string(runes[:SongBatchJobErrorMessageMaxLength])
	}
	return j.finish(SongBatchJobStatusFailed, warningCount, message, finishedAt)
}

// Interrupt は処理の中断を記録します。
func (j *SongBatchJob) Interrupt(finishedAt time.Time) error {
	return j.finish(SongBatchJobStatusInterrupted, j.warningCount, "", finishedAt)
}

func (j *SongBatchJob) finish(status SongBatchJobStatus, warningCount int, message string, finishedAt time.Time) error {
	if !j.IsRunning() {
		return ErrSongBatchJobAlreadyFinished
	}
	j.status = status
	j.warningCount = warningCount
	j.errorMessage = message
	j.finishedAt = &finishedAt
	return nil
}
