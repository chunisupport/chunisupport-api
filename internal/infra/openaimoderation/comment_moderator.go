package openaimoderation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

const (
	moderationsPath  = "/v1/moderations"
	maxResponseBytes = 64 * 1024
)

type commentModerator struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

type moderationRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type moderationResponse struct {
	// 欠落や null を false と区別するため、flagged はポインタで受けます。
	Results []struct {
		Flagged *bool `json:"flagged"`
	} `json:"results"`
}

// NewCommentModerator は OpenAI Moderation API でコメントを判定する CommentModerator を生成します。
// 送信するのはコメント本文だけで、利用者を特定できる情報は含めません。
func NewCommentModerator(apiKey string) usecase.CommentModerator {
	return newCommentModerator(apiKey, info.OpenAIAPIBaseURL, &http.Client{Timeout: info.OpenAIModerationRequestTimeout})
}

func newCommentModerator(apiKey string, baseURL string, client *http.Client) *commentModerator {
	return &commentModerator{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
	}
}

// IsFlagged は Moderation API の flagged をそのまま返します。
// 通信の失敗、2xx 以外の応答、判定結果のない応答、flagged が明示的な true/false でない応答
// （欠落・null・results の要素が null）は、判定できなかったものとしてエラーにします。
// 判定できないまま「問題なし」と扱わない（fail-closed）ためです。
func (m *commentModerator) IsFlagged(ctx context.Context, comment string) (bool, error) {
	body, err := json.Marshal(moderationRequest{Model: info.OpenAIModerationModel, Input: comment})
	if err != nil {
		return false, fmt.Errorf("encode moderation request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+moderationsPath, bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("build moderation request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("moderation request: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		if err := resp.Body.Close(); err != nil {
			slog.WarnContext(ctx, "failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return false, fmt.Errorf("moderation api returned status %d", resp.StatusCode)
	}

	var result moderationResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&result); err != nil {
		return false, fmt.Errorf("decode moderation response: %w", err)
	}
	if len(result.Results) == 0 {
		return false, errors.New("moderation response has no result")
	}

	flagged := result.Results[0].Flagged
	if flagged == nil {
		return false, errors.New("moderation response has no flagged")
	}

	return *flagged, nil
}

var _ usecase.CommentModerator = (*commentModerator)(nil)
