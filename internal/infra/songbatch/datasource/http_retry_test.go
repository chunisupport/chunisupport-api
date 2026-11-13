package datasource

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRetryableDownloadError(t *testing.T) {
	t.Parallel()

	if !isRetryableDownloadError(httpStatusError{status: http.StatusServiceUnavailable}) {
		t.Fatal("503 should be retryable")
	}
	if !isRetryableDownloadError(fmt.Errorf("failed to batch get sheet data: %w", httpStatusError{status: http.StatusServiceUnavailable})) {
		t.Fatal("wrapped 503 should be retryable")
	}
	if isRetryableDownloadError(httpStatusError{status: http.StatusNotFound}) {
		t.Fatal("404 should not be retryable")
	}
	if isRetryableDownloadError(fmt.Errorf("failed to execute request")) {
		t.Fatal("generic error should not be retryable")
	}
}

func TestRetryDownloadRetriesTransientError(t *testing.T) {
	original := retryBackoff
	retryBackoff = func(int) time.Duration { return 0 }
	t.Cleanup(func() { retryBackoff = original })

	attempts := 0
	err := retryDownload(context.Background(), "additional_songs", func() error {
		attempts++
		if attempts < 3 {
			return httpStatusError{status: http.StatusServiceUnavailable}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected retry to succeed, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestRetryDownloadStopsDuringBackoffWhenCanceled(t *testing.T) {
	original := retryBackoff
	retryBackoff = func(int) time.Duration { return time.Hour }
	t.Cleanup(func() { retryBackoff = original })

	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0
	err := retryDownload(ctx, "official", func() error {
		attempts++
		cancel()
		return httpStatusError{status: http.StatusServiceUnavailable}
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestWrapRequestErrorRedactsURLAndPreservesCause(t *testing.T) {
	cause := errors.New("connection refused")
	err := wrapRequestError(&url.Error{
		Op:  "Get",
		URL: "https://example.invalid/data?key=secret",
		Err: cause,
	})
	if !errors.Is(err, cause) {
		t.Fatalf("error = %v, want wrapped cause", err)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatalf("error contains secret URL: %v", err)
	}
}

func TestWithGoogleSheetsGateReturnsWhenCanceled(t *testing.T) {
	googleSheetsGate <- struct{}{}
	t.Cleanup(func() { <-googleSheetsGate })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := withGoogleSheetsGate(ctx, func() error {
		t.Fatal("canceled waiter must not enter the gate")
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestRetryDownloadGivesUpAfterMaxAttempts(t *testing.T) {
	// Given: 再試行対象のエラーを返し続けるダウンロード。待ち時間は0にして実時間で待たない。
	original := retryBackoff
	backoffCalls := 0
	retryBackoff = func(int) time.Duration {
		backoffCalls++
		return 0
	}
	t.Cleanup(func() { retryBackoff = original })

	// 試行ごとに異なる再試行対象ステータスを返し、最後のエラーを区別できるようにする。
	attempts := 0

	// When
	err := retryDownload(context.Background(), "additional_songs", func() error {
		attempts++
		switch attempts {
		case 1:
			return httpStatusError{status: http.StatusTooManyRequests}
		case 2:
			return httpStatusError{status: http.StatusBadGateway}
		default:
			return httpStatusError{status: http.StatusGatewayTimeout}
		}
	})

	// Then: 決められた3回で打ち切り、最後のエラーを返す。待ちは試行間の2回だけ。
	require.Error(t, err)
	assert.Equal(t, 3, attempts)
	assert.Equal(t, 2, backoffCalls)
	var statusErr httpStatusError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, http.StatusGatewayTimeout, statusErr.status)
}

func TestRetryDownloadDoesNotRetryNonRetryableError(t *testing.T) {
	// Given: 待ち時間の計算回数を数え、実時間では待たない。
	original := retryBackoff
	backoffCalls := 0
	retryBackoff = func(int) time.Duration {
		backoffCalls++
		return 0
	}
	t.Cleanup(func() { retryBackoff = original })

	attempts := 0

	// When: 再試行対象外のステータスを返す。
	err := retryDownload(context.Background(), "official", func() error {
		attempts++
		return httpStatusError{status: http.StatusNotFound}
	})

	// Then: 1回で止め、待ち時間の計算もせず、受けたエラーをそのまま返す。
	require.Error(t, err)
	assert.Equal(t, 1, attempts)
	assert.Equal(t, 0, backoffCalls)
	var statusErr httpStatusError
	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, http.StatusNotFound, statusErr.status)
}

func TestIsRetryableDownloadError_StatusTable(t *testing.T) {
	tests := []struct {
		name string
		// Given: 判定対象のステータスコード
		status int
		// Then: 期待する判定結果
		wantRetryable bool
	}{
		{
			name:          "429は再試行する",
			status:        http.StatusTooManyRequests,
			wantRetryable: true,
		},
		{
			name:          "502は再試行する",
			status:        http.StatusBadGateway,
			wantRetryable: true,
		},
		{
			name:          "504は再試行する",
			status:        http.StatusGatewayTimeout,
			wantRetryable: true,
		},
		{
			name:          "400は再試行しない",
			status:        http.StatusBadRequest,
			wantRetryable: false,
		},
		{
			name:          "500は再試行しない",
			status:        http.StatusInternalServerError,
			wantRetryable: false,
		},
		{
			name:          "501は再試行しない",
			status:        http.StatusNotImplemented,
			wantRetryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got := isRetryableDownloadError(httpStatusError{status: tt.status})

			// Then
			assert.Equal(t, tt.wantRetryable, got)
		})
	}
}
