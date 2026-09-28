package entity

import (
	"errors"
	"strings"
	"time"
	"uuid"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
)

// ChartStatsBatchJobErrorMessageMaxLength は保存する失敗・中断の理由の最大文字数です。
// DBエラー全文で行が肥大化しないよう、管理画面で状況を把握できる長さに制限します。
const ChartStatsBatchJobErrorMessageMaxLength = 1000

var (
	// ErrInvalidChartStatsBatchJob は譜面統計バッチジョブの不変条件を満たさないことを表します。
	ErrInvalidChartStatsBatchJob = errors.New("invalid chart stats batch job")
	// ErrChartStatsBatchJobAlreadyFinished は終了済みのジョブを再度終了しようとしたことを表します。
	ErrChartStatsBatchJobAlreadyFinished = errors.New("chart stats batch job already finished")
)

// ChartStatsBatchJobStatus は譜面統計バッチジョブの状態です。
type ChartStatsBatchJobStatus string

const (
	// ChartStatsBatchJobStatusRunning は実行中です。
	ChartStatsBatchJobStatusRunning ChartStatsBatchJobStatus = "RUNNING"
	// ChartStatsBatchJobStatusSucceeded は統計テーブルを再構築したことを表します。
	ChartStatsBatchJobStatusSucceeded ChartStatsBatchJobStatus = "SUCCEEDED"
	// ChartStatsBatchJobStatusFailed は処理を完了できず、統計テーブルを更新しなかったことを表します。
	ChartStatsBatchJobStatusFailed ChartStatsBatchJobStatus = "FAILED"
	// ChartStatsBatchJobStatusInterrupted はプロセス停止などで処理が中断されたことを表します。
	// 実行中のキャンセルで中断した場合、統計テーブルの入れ替えは単一トランザクションのためロールバック済みです。
	// 終了を記録できずに取り残されたジョブを後から中断扱いにした場合は、入れ替えの成否は不明です。
	ChartStatsBatchJobStatusInterrupted ChartStatsBatchJobStatus = "INTERRUPTED"
)

func (s ChartStatsBatchJobStatus) isValid() bool {
	switch s {
	case ChartStatsBatchJobStatusRunning, ChartStatsBatchJobStatusSucceeded, ChartStatsBatchJobStatusFailed, ChartStatsBatchJobStatusInterrupted:
		return true
	default:
		return false
	}
}

// ChartStatsBatchJobTrigger は譜面統計バッチジョブの起動元です。
type ChartStatsBatchJobTrigger string

const (
	// ChartStatsBatchJobTriggerCLI は cron などから CLI で起動したことを表します。
	ChartStatsBatchJobTriggerCLI ChartStatsBatchJobTrigger = "CLI"
	// ChartStatsBatchJobTriggerAdmin は管理画面から起動したことを表します。
	ChartStatsBatchJobTriggerAdmin ChartStatsBatchJobTrigger = "ADMIN"
)

func (t ChartStatsBatchJobTrigger) isValid() bool {
	return t == ChartStatsBatchJobTriggerCLI || t == ChartStatsBatchJobTriggerAdmin
}

// ChartStatsBatchJobRequester は管理画面からジョブを起動したユーザーです。
type ChartStatsBatchJobRequester struct {
	UserID   int
	Username username.UserName
}

// ChartStatsBatchJob は譜面統計バッチ1回分の実行を表す集約です。
type ChartStatsBatchJob struct {
	id           uuid.UUID
	trigger      ChartStatsBatchJobTrigger
	requester    *ChartStatsBatchJobRequester
	status       ChartStatsBatchJobStatus
	startedAt    time.Time
	finishedAt   *time.Time
	errorMessage string
}

// StartChartStatsBatchJobFromCLI は CLI から起動した実行中のジョブを生成します。
func StartChartStatsBatchJobFromCLI(id uuid.UUID, startedAt time.Time) *ChartStatsBatchJob {
	return &ChartStatsBatchJob{
		id:        id,
		trigger:   ChartStatsBatchJobTriggerCLI,
		status:    ChartStatsBatchJobStatusRunning,
		startedAt: startedAt,
	}
}

// StartChartStatsBatchJobFromAdmin は管理画面から起動した実行中のジョブを生成します。
func StartChartStatsBatchJobFromAdmin(id uuid.UUID, requester ChartStatsBatchJobRequester, startedAt time.Time) *ChartStatsBatchJob {
	return &ChartStatsBatchJob{
		id:        id,
		trigger:   ChartStatsBatchJobTriggerAdmin,
		requester: &requester,
		status:    ChartStatsBatchJobStatusRunning,
		startedAt: startedAt,
	}
}

// ReconstructChartStatsBatchJob は永続化済みデータから不変条件を満たすジョブを復元します。
// 要求者のユーザーが削除された場合、管理画面ジョブでも requester は nil になります。
func ReconstructChartStatsBatchJob(
	id uuid.UUID,
	trigger ChartStatsBatchJobTrigger,
	requester *ChartStatsBatchJobRequester,
	status ChartStatsBatchJobStatus,
	startedAt time.Time,
	finishedAt *time.Time,
	errorMessage string,
) (*ChartStatsBatchJob, error) {
	if !trigger.isValid() || !status.isValid() {
		return nil, ErrInvalidChartStatsBatchJob
	}
	if (status == ChartStatsBatchJobStatusRunning) != (finishedAt == nil) {
		return nil, ErrInvalidChartStatsBatchJob
	}
	return &ChartStatsBatchJob{
		id:           id,
		trigger:      trigger,
		requester:    requester,
		status:       status,
		startedAt:    startedAt,
		finishedAt:   finishedAt,
		errorMessage: errorMessage,
	}, nil
}

// ID はジョブIDを返します。
func (j *ChartStatsBatchJob) ID() uuid.UUID { return j.id }

// Trigger は起動元を返します。
func (j *ChartStatsBatchJob) Trigger() ChartStatsBatchJobTrigger { return j.trigger }

// Requester は管理画面から起動したユーザーを返します。CLI 起動や要求者削除後は nil です。
func (j *ChartStatsBatchJob) Requester() *ChartStatsBatchJobRequester { return j.requester }

// Status は状態を返します。
func (j *ChartStatsBatchJob) Status() ChartStatsBatchJobStatus { return j.status }

// StartedAt は開始日時を返します。
func (j *ChartStatsBatchJob) StartedAt() time.Time { return j.startedAt }

// FinishedAt は終了日時を返します。実行中は nil です。
func (j *ChartStatsBatchJob) FinishedAt() *time.Time { return j.finishedAt }

// ErrorMessage は失敗・中断の理由を返します。成功・実行中、および理由なしで中断扱いにしたジョブでは空文字です。
func (j *ChartStatsBatchJob) ErrorMessage() string { return j.errorMessage }

// IsRunning は実行中かどうかを返します。
func (j *ChartStatsBatchJob) IsRunning() bool { return j.status == ChartStatsBatchJobStatusRunning }

// Complete は処理の完了を記録します。
func (j *ChartStatsBatchJob) Complete(finishedAt time.Time) error {
	return j.finish(ChartStatsBatchJobStatusSucceeded, "", finishedAt)
}

// Fail は処理の失敗を記録します。失敗理由は上限文字数で切り詰めます。
func (j *ChartStatsBatchJob) Fail(message string, finishedAt time.Time) error {
	message = normalizeChartStatsBatchJobMessage(message)
	if message == "" {
		return ErrInvalidChartStatsBatchJob
	}
	return j.finish(ChartStatsBatchJobStatusFailed, message, finishedAt)
}

// Interrupt は処理の中断を記録します。
// キャンセルと同時に別のエラーが起きた場合も原因を追えるよう、実行が返したエラーを理由として残します。
// 取り残されたジョブを後から中断扱いにする場合は理由が分からないため、空文字を渡します。
func (j *ChartStatsBatchJob) Interrupt(message string, finishedAt time.Time) error {
	return j.finish(ChartStatsBatchJobStatusInterrupted, normalizeChartStatsBatchJobMessage(message), finishedAt)
}

// normalizeChartStatsBatchJobMessage は理由の前後の空白を除き、上限文字数で切り詰めます。
func normalizeChartStatsBatchJobMessage(message string) string {
	message = strings.TrimSpace(message)
	if runes := []rune(message); len(runes) > ChartStatsBatchJobErrorMessageMaxLength {
		message = string(runes[:ChartStatsBatchJobErrorMessageMaxLength])
	}
	return message
}

func (j *ChartStatsBatchJob) finish(status ChartStatsBatchJobStatus, message string, finishedAt time.Time) error {
	if !j.IsRunning() {
		return ErrChartStatsBatchJobAlreadyFinished
	}
	j.status = status
	j.errorMessage = message
	j.finishedAt = &finishedAt
	return nil
}
