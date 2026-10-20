package logger

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextHandler_リクエストIDをアプリケーションログへ付与する(t *testing.T) {
	tests := []struct {
		name      string
		requestID string
		useCtx    bool
		wantAttr  bool
	}{
		{
			name:      "コンテキストのIDをrequest_idとして出す",
			requestID: "req-123",
			useCtx:    true,
			wantAttr:  true,
		},
		{
			name:     "コンテキストにIDがないログには属性を付けない",
			useCtx:   false,
			wantAttr: false,
		},
		{
			name:      "空のIDは属性にしない",
			requestID: "",
			useCtx:    true,
			wantAttr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			var output bytes.Buffer
			logger := slog.New(newContextHandler(slog.NewTextHandler(&output, nil)))
			ctx := context.Background()
			if tt.useCtx {
				ctx = WithRequestID(ctx, tt.requestID)
			}

			// When
			logger.InfoContext(ctx, "処理失敗")

			// Then
			assert.Contains(t, output.String(), "処理失敗")
			if tt.wantAttr {
				assert.Contains(t, output.String(), "request_id="+tt.requestID)
			} else {
				assert.NotContains(t, output.String(), "request_id=")
			}
		})
	}
}

func TestRequestIDFromContext_未設定は空文字(t *testing.T) {
	assert.Empty(t, RequestIDFromContext(context.Background()))
	assert.Empty(t, RequestIDFromContext(nil))
}
