package entity

import (
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username/usernametest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var chartStatsBatchJobStartedAt = time.Date(2026, 9, 28, 3, 0, 0, 0, time.UTC)

func TestStartChartStatsBatchJob(t *testing.T) {
	requester := ChartStatsBatchJobRequester{UserID: 10, Username: usernametest.New(t, "adminuser")}
	tests := []struct {
		name              string
		job               *ChartStatsBatchJob
		expectedTrigger   ChartStatsBatchJobTrigger
		expectedRequester *ChartStatsBatchJobRequester
	}{
		{
			name:            "CLIから起動したジョブは要求者を持たない",
			job:             StartChartStatsBatchJobFromCLI(uuid.NewV4(), chartStatsBatchJobStartedAt),
			expectedTrigger: ChartStatsBatchJobTriggerCLI,
		},
		{
			name:              "管理画面から起動したジョブは要求者を持つ",
			job:               StartChartStatsBatchJobFromAdmin(uuid.NewV4(), requester, chartStatsBatchJobStartedAt),
			expectedTrigger:   ChartStatsBatchJobTriggerAdmin,
			expectedRequester: &requester,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Then
			assert.Equal(t, tt.expectedTrigger, tt.job.Trigger())
			assert.Equal(t, tt.expectedRequester, tt.job.Requester())
			assert.Equal(t, ChartStatsBatchJobStatusRunning, tt.job.Status())
			assert.True(t, tt.job.IsRunning())
			assert.Equal(t, chartStatsBatchJobStartedAt, tt.job.StartedAt())
			assert.Nil(t, tt.job.FinishedAt())
		})
	}
}

func TestChartStatsBatchJob_Finish(t *testing.T) {
	finishedAt := chartStatsBatchJobStartedAt.Add(time.Minute)
	tests := []struct {
		name            string
		finish          func(job *ChartStatsBatchJob) error
		expectedStatus  ChartStatsBatchJobStatus
		expectedMessage string
	}{
		{
			name:           "完了を記録する",
			finish:         func(job *ChartStatsBatchJob) error { return job.Complete(finishedAt) },
			expectedStatus: ChartStatsBatchJobStatusSucceeded,
		},
		{
			name:            "失敗理由の前後の空白を除いて失敗を記録する",
			finish:          func(job *ChartStatsBatchJob) error { return job.Fail("  db down \n", finishedAt) },
			expectedStatus:  ChartStatsBatchJobStatusFailed,
			expectedMessage: "db down",
		},
		{
			name:            "失敗理由は上限文字数で切り詰める",
			finish:          func(job *ChartStatsBatchJob) error { return job.Fail(strings.Repeat("あ", 1001), finishedAt) },
			expectedStatus:  ChartStatsBatchJobStatusFailed,
			expectedMessage: strings.Repeat("あ", ChartStatsBatchJobErrorMessageMaxLength),
		},
		{
			name: "中断の理由とともに中断を記録する",
			finish: func(job *ChartStatsBatchJob) error {
				return job.Interrupt(" read failed: context canceled\n", finishedAt)
			},
			expectedStatus:  ChartStatsBatchJobStatusInterrupted,
			expectedMessage: "read failed: context canceled",
		},
		{
			name:           "取り残されたジョブは理由なしで中断を記録する",
			finish:         func(job *ChartStatsBatchJob) error { return job.Interrupt("", finishedAt) },
			expectedStatus: ChartStatsBatchJobStatusInterrupted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			job := StartChartStatsBatchJobFromCLI(uuid.NewV4(), chartStatsBatchJobStartedAt)

			// When
			err := tt.finish(job)

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, job.Status())
			assert.Equal(t, tt.expectedMessage, job.ErrorMessage())
			require.NotNil(t, job.FinishedAt())
			assert.Equal(t, finishedAt, *job.FinishedAt())
		})
	}
}

func TestChartStatsBatchJob_Finish_エラー(t *testing.T) {
	tests := []struct {
		name        string
		prepare     func(job *ChartStatsBatchJob)
		finish      func(job *ChartStatsBatchJob) error
		expectedErr error
	}{
		{
			name:        "空の失敗理由は記録できない",
			finish:      func(job *ChartStatsBatchJob) error { return job.Fail(" ", chartStatsBatchJobStartedAt) },
			expectedErr: ErrInvalidChartStatsBatchJob,
		},
		{
			name:        "終了済みのジョブは再度終了できない",
			prepare:     func(job *ChartStatsBatchJob) { _ = job.Complete(chartStatsBatchJobStartedAt) },
			finish:      func(job *ChartStatsBatchJob) error { return job.Interrupt("", chartStatsBatchJobStartedAt) },
			expectedErr: ErrChartStatsBatchJobAlreadyFinished,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			job := StartChartStatsBatchJobFromCLI(uuid.NewV4(), chartStatsBatchJobStartedAt)
			if tt.prepare != nil {
				tt.prepare(job)
			}

			// When
			err := tt.finish(job)

			// Then
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func TestReconstructChartStatsBatchJob(t *testing.T) {
	finishedAt := chartStatsBatchJobStartedAt.Add(time.Minute)
	tests := []struct {
		name       string
		trigger    ChartStatsBatchJobTrigger
		status     ChartStatsBatchJobStatus
		finishedAt *time.Time
		wantErr    bool
	}{
		{name: "終了済みのジョブを復元できる", trigger: ChartStatsBatchJobTriggerAdmin, status: ChartStatsBatchJobStatusSucceeded, finishedAt: &finishedAt},
		{name: "実行中のジョブを復元できる", trigger: ChartStatsBatchJobTriggerCLI, status: ChartStatsBatchJobStatusRunning},
		{name: "未知の起動元はエラー", trigger: "CRON", status: ChartStatsBatchJobStatusRunning, wantErr: true},
		{name: "未知の状態はエラー", trigger: ChartStatsBatchJobTriggerCLI, status: "SUCCEEDED_WITH_WARNINGS", finishedAt: &finishedAt, wantErr: true},
		{name: "実行中なのに終了日時があるとエラー", trigger: ChartStatsBatchJobTriggerCLI, status: ChartStatsBatchJobStatusRunning, finishedAt: &finishedAt, wantErr: true},
		{name: "終了済みなのに終了日時がないとエラー", trigger: ChartStatsBatchJobTriggerCLI, status: ChartStatsBatchJobStatusFailed, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			job, err := ReconstructChartStatsBatchJob(uuid.NewV4(), tt.trigger, nil, tt.status, chartStatsBatchJobStartedAt, tt.finishedAt, "")

			// Then
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidChartStatsBatchJob)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.status, job.Status())
			assert.Equal(t, tt.finishedAt, job.FinishedAt())
		})
	}
}
