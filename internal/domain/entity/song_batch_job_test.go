package entity

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
	"uuid"

	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var songBatchJobStartedAt = time.Date(2026, 9, 26, 3, 0, 0, 0, time.UTC)

func newRunningSongBatchJob() *SongBatchJob {
	return StartSongBatchJobFromCLI(uuid.NewV4(), songbatch.NewRunRequest(false, false), songBatchJobStartedAt)
}

func TestStartSongBatchJob(t *testing.T) {
	requester := SongBatchJobRequester{UserID: 10, Username: username.MustNewUserName("adminuser")}

	tests := []struct {
		name              string
		start             func(id uuid.UUID) *SongBatchJob
		expectedTrigger   SongBatchJobTrigger
		expectedRequester *SongBatchJobRequester
	}{
		{
			name: "CLIから開始したジョブは要求者を持たない",
			start: func(id uuid.UUID) *SongBatchJob {
				return StartSongBatchJobFromCLI(id, songbatch.NewRunRequest(true, false), songBatchJobStartedAt)
			},
			expectedTrigger: SongBatchJobTriggerCLI,
		},
		{
			name: "管理画面から開始したジョブは要求者を持つ",
			start: func(id uuid.UUID) *SongBatchJob {
				return StartSongBatchJobFromAdmin(id, songbatch.NewRunRequest(true, false), requester, songBatchJobStartedAt)
			},
			expectedTrigger:   SongBatchJobTriggerAdmin,
			expectedRequester: &requester,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			id := uuid.NewV4()

			// When
			job := tt.start(id)

			// Then
			assert.Equal(t, id, job.ID())
			assert.Equal(t, songbatch.NewRunRequest(true, false), job.Request())
			assert.Equal(t, tt.expectedTrigger, job.Trigger())
			assert.Equal(t, tt.expectedRequester, job.Requester())
			assert.Equal(t, SongBatchJobStatusRunning, job.Status())
			assert.True(t, job.IsRunning())
			assert.Equal(t, songBatchJobStartedAt, job.StartedAt())
			assert.Nil(t, job.FinishedAt())
		})
	}
}

func TestSongBatchJob_Finish(t *testing.T) {
	finishedAt := songBatchJobStartedAt.Add(3 * time.Minute)

	tests := []struct {
		name                 string
		finish               func(job *SongBatchJob) error
		expectedStatus       SongBatchJobStatus
		expectedWarningCount int
		expectedErrorMessage string
	}{
		{
			name:           "警告なしで完了すると成功になる",
			finish:         func(job *SongBatchJob) error { return job.Complete(0, finishedAt) },
			expectedStatus: SongBatchJobStatusSucceeded,
		},
		{
			name:                 "警告ありで完了すると警告付き成功になる",
			finish:               func(job *SongBatchJob) error { return job.Complete(2, finishedAt) },
			expectedStatus:       SongBatchJobStatusSucceededWithWarnings,
			expectedWarningCount: 2,
		},
		{
			name:                 "失敗すると失敗理由を保持する",
			finish:               func(job *SongBatchJob) error { return job.Fail(1, "required datasource official failed", finishedAt) },
			expectedStatus:       SongBatchJobStatusFailed,
			expectedWarningCount: 1,
			expectedErrorMessage: "required datasource official failed",
		},
		{
			name:                 "中断すると中断までの警告件数とともに中断状態になる",
			finish:               func(job *SongBatchJob) error { return job.Interrupt(1, finishedAt) },
			expectedStatus:       SongBatchJobStatusInterrupted,
			expectedWarningCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			job := newRunningSongBatchJob()

			// When
			err := tt.finish(job)

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, job.Status())
			assert.False(t, job.IsRunning())
			assert.Equal(t, tt.expectedWarningCount, job.WarningCount())
			assert.Equal(t, tt.expectedErrorMessage, job.ErrorMessage())
			require.NotNil(t, job.FinishedAt())
			assert.Equal(t, finishedAt, *job.FinishedAt())
		})
	}
}

func TestSongBatchJob_FinishedJobCannotTransition(t *testing.T) {
	finishedAt := songBatchJobStartedAt.Add(time.Minute)

	tests := []struct {
		name   string
		finish func(job *SongBatchJob) error
	}{
		{name: "完了済みジョブは再度完了できない", finish: func(job *SongBatchJob) error { return job.Complete(0, finishedAt) }},
		{name: "完了済みジョブは失敗にできない", finish: func(job *SongBatchJob) error { return job.Fail(0, "error", finishedAt) }},
		{name: "完了済みジョブは中断にできない", finish: func(job *SongBatchJob) error { return job.Interrupt(0, finishedAt) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			job := newRunningSongBatchJob()
			require.NoError(t, job.Complete(0, finishedAt))

			// When
			err := tt.finish(job)

			// Then
			assert.ErrorIs(t, err, ErrSongBatchJobAlreadyFinished)
			assert.Equal(t, SongBatchJobStatusSucceeded, job.Status())
		})
	}
}

func TestSongBatchJob_FailValidation(t *testing.T) {
	tests := []struct {
		name                 string
		warningCount         int
		message              string
		wantErr              bool
		expectedErrorMessage string
	}{
		{
			name:    "空の失敗理由はエラー",
			message: "   ",
			wantErr: true,
		},
		{
			name:         "負の警告件数はエラー",
			warningCount: -1,
			message:      "error",
			wantErr:      true,
		},
		{
			name:                 "上限を超える失敗理由は切り詰める",
			message:              strings.Repeat("あ", SongBatchJobErrorMessageMaxLength+10),
			expectedErrorMessage: strings.Repeat("あ", SongBatchJobErrorMessageMaxLength),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			job := newRunningSongBatchJob()

			// When
			err := job.Fail(tt.warningCount, tt.message, songBatchJobStartedAt.Add(time.Minute))

			// Then
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidSongBatchJob)
				assert.True(t, job.IsRunning())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedErrorMessage, job.ErrorMessage())
			assert.Equal(t, SongBatchJobErrorMessageMaxLength, utf8.RuneCountInString(job.ErrorMessage()))
		})
	}
}

func TestSongBatchJob_CompleteRejectsNegativeWarningCount(t *testing.T) {
	// Given
	job := newRunningSongBatchJob()

	// When
	err := job.Complete(-1, songBatchJobStartedAt.Add(time.Minute))

	// Then
	assert.ErrorIs(t, err, ErrInvalidSongBatchJob)
	assert.True(t, job.IsRunning())
}

func TestReconstructSongBatchJob(t *testing.T) {
	finishedAt := songBatchJobStartedAt.Add(time.Minute)
	requester := &SongBatchJobRequester{UserID: 1, Username: username.MustNewUserName("adminuser")}

	tests := []struct {
		name       string
		trigger    SongBatchJobTrigger
		requester  *SongBatchJobRequester
		status     SongBatchJobStatus
		finishedAt *time.Time
		wantErr    bool
	}{
		{name: "実行中のジョブを復元できる", trigger: SongBatchJobTriggerCLI, status: SongBatchJobStatusRunning},
		{name: "終了済みのジョブを復元できる", trigger: SongBatchJobTriggerAdmin, requester: requester, status: SongBatchJobStatusSucceeded, finishedAt: &finishedAt},
		{name: "要求者が削除された管理画面ジョブも復元できる", trigger: SongBatchJobTriggerAdmin, status: SongBatchJobStatusFailed, finishedAt: &finishedAt},
		{name: "未知の状態はエラー", trigger: SongBatchJobTriggerCLI, status: "UNKNOWN", wantErr: true},
		{name: "未知の起動元はエラー", trigger: "CRON", status: SongBatchJobStatusRunning, wantErr: true},
		{name: "終了済みなのに終了日時がない場合はエラー", trigger: SongBatchJobTriggerCLI, status: SongBatchJobStatusSucceeded, wantErr: true},
		{name: "実行中なのに終了日時がある場合はエラー", trigger: SongBatchJobTriggerCLI, status: SongBatchJobStatusRunning, finishedAt: &finishedAt, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			job, err := ReconstructSongBatchJob(
				uuid.NewV4(),
				songbatch.NewRunRequest(false, true),
				tt.trigger,
				tt.requester,
				tt.status,
				songBatchJobStartedAt,
				tt.finishedAt,
				0,
				"",
			)

			// Then
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidSongBatchJob)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.status, job.Status())
			assert.Equal(t, tt.requester, job.Requester())
		})
	}
}
