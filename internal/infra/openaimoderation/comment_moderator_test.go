package openaimoderation

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommentModerator_IsFlagged_判定結果を返す(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{name: "flaggedがtrueなら不適切", body: `{"results":[{"flagged":true,"categories":{"harassment":true}}]}`, want: true},
		{name: "flaggedがfalseなら問題なし", body: `{"results":[{"flagged":false}]}`, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			var gotPath, gotAuthorization, gotContentType, gotBody string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotAuthorization = r.Header.Get("Authorization")
				gotContentType = r.Header.Get("Content-Type")
				body, _ := io.ReadAll(r.Body)
				gotBody = string(body)
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(server.Close)
			moderator := newCommentModerator("api-key", server.URL, server.Client())

			// When
			got, err := moderator.IsFlagged(context.Background(), "よろしく")

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, "/v1/moderations", gotPath)
			assert.Equal(t, "Bearer api-key", gotAuthorization)
			assert.Equal(t, "application/json", gotContentType)
			assert.JSONEq(t, `{"model":"omni-moderation-latest","input":"よろしく"}`, gotBody)
		})
	}
}

func TestCommentModerator_IsFlagged_判定できない応答はエラー(t *testing.T) {
	// 非2xxの本文は正常な成功応答にし、ステータス検査の削除が結果欠落検査にすり抜けないようにします。
	// 本文が{}だと検査削除後も結果欠落でエラーになり、退行を検出できません。
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{name: "APIキーの拒否", status: http.StatusUnauthorized, body: `{"results":[{"flagged":false}]}`},
		{name: "回数制限", status: http.StatusTooManyRequests, body: `{"results":[{"flagged":false}]}`},
		{name: "サーバーエラー", status: http.StatusInternalServerError, body: `{"results":[{"flagged":false}]}`},
		{name: "JSONとして読めない応答", status: http.StatusOK, body: `not json`},
		{name: "判定結果のない成功応答", status: http.StatusOK, body: `{"results":[]}`},
		{name: "flaggedが欠落した成功応答", status: http.StatusOK, body: `{"results":[{}]}`},
		{name: "判定結果がnullの成功応答", status: http.StatusOK, body: `{"results":[null]}`},
		{name: "flaggedがnullの成功応答", status: http.StatusOK, body: `{"results":[{"flagged":null}]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(server.Close)
			moderator := newCommentModerator("api-key", server.URL, server.Client())

			// When
			got, err := moderator.IsFlagged(context.Background(), "よろしく")

			// Then
			require.Error(t, err)
			assert.False(t, got)
		})
	}
}

func TestCommentModerator_IsFlagged_キャンセルの原因を保持する(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	t.Cleanup(server.Close)
	moderator := newCommentModerator("api-key", server.URL, server.Client())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// When
	_, err := moderator.IsFlagged(ctx, "よろしく")

	// Then
	require.ErrorIs(t, err, context.Canceled)
}
