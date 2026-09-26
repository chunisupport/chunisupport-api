package apierror

import (
	"net/http"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromUsecaseError_楽曲バッチ関連エラー(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "実行中の楽曲バッチがある場合は409",
			err:            usecase.ErrSongBatchAlreadyRunning,
			expectedStatus: http.StatusConflict,
			expectedCode:   CodeSongBatchAlreadyRunning,
		},
		{
			name:           "サーバー停止処理中は503",
			err:            usecase.ErrSongBatchUnavailable,
			expectedStatus: http.StatusServiceUnavailable,
			expectedCode:   CodeServiceUnavailable,
		},
		{
			name:           "ジョブIDが不正な場合は400",
			err:            usecase.ErrInvalidSongBatchJobID,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   CodeInvalidSongBatchJobID,
		},
		{
			name:           "ジョブが存在しない場合は404",
			err:            repository.ErrSongBatchJobNotFound,
			expectedStatus: http.StatusNotFound,
			expectedCode:   CodeSongBatchJobNotFound,
		},
		{
			name:           "実行モードが不正な場合は400",
			err:            songbatch.ErrInvalidRunMode,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   CodeInvalidSongBatchMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			apiErr := FromUsecaseError(tt.err)

			// Then
			require.NotNil(t, apiErr)
			assert.Equal(t, tt.expectedStatus, apiErr.HTTPStatus)
			assert.Equal(t, tt.expectedCode, apiErr.Code)
		})
	}
}
