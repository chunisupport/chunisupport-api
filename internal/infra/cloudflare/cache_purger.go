package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

type sleepFunc func(ctx context.Context, duration time.Duration) error

type retryDecision struct {
	retryable     bool
	delay         time.Duration
	delayProvided bool
}

// CachePurger は単一APIのためだけにCloudflare SDKへ依存しないよう、標準HTTPクライアントを保持します。
type CachePurger struct {
	apiToken      string
	publicBaseURL string
	httpClient    *http.Client
	endpoint      string
	sleep         sleepFunc
}

// NewCachePurger はR2のS3認証とパージ権限を共有させないため、専用設定からクライアントを生成します。
func NewCachePurger(cfg config.CloudflareCacheConfig) *CachePurger {
	endpoint := fmt.Sprintf("%s/zones/%s/purge_cache", info.CloudflareAPIBaseURL, cfg.ZoneID)
	return newCachePurger(
		cfg,
		&http.Client{Timeout: info.CloudflareAPITimeout},
		endpoint,
		sleepContext,
	)
}

func newCachePurger(
	cfg config.CloudflareCacheConfig,
	httpClient *http.Client,
	endpoint string,
	sleep sleepFunc,
) *CachePurger {
	return &CachePurger{
		apiToken:      cfg.APIToken,
		publicBaseURL: cfg.PublicBaseURL,
		httpClient:    httpClient,
		endpoint:      endpoint,
		sleep:         sleep,
	}
}

// Purge は同じZoneの無関係なキャッシュを残すため、全削除ではなく公開URLだけをまとめて削除します。
func (p *CachePurger) Purge(ctx context.Context, objectKeys []string) error {
	files := make([]string, 0, len(objectKeys))
	for _, objectKey := range objectKeys {
		fileURL, err := url.JoinPath(p.publicBaseURL, objectKey)
		if err != nil {
			return fmt.Errorf("failed to build public object URL for %s: %w", objectKey, err)
		}
		files = append(files, fileURL)
	}

	requestBody, err := json.Marshal(struct {
		Files []string `json:"files"`
	}{Files: files})
	if err != nil {
		return fmt.Errorf("failed to marshal cloudflare cache purge request: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= info.CloudflareCachePurgeMaxAttempts; attempt++ {
		decision, err := p.purgeOnce(ctx, requestBody)
		if err == nil {
			return nil
		}
		lastErr = err
		if !decision.retryable || attempt == info.CloudflareCachePurgeMaxAttempts {
			break
		}
		retryAfter := decision.delay
		if !decision.delayProvided {
			retryAfter = time.Duration(attempt) * info.CloudflareCachePurgeRetryBaseDelay
		}
		if err := p.sleep(ctx, retryAfter); err != nil {
			return fmt.Errorf("cloudflare cache purge retry interrupted: %w", err)
		}
	}

	return lastErr
}

func (p *CachePurger) purgeOnce(ctx context.Context, requestBody []byte) (retryDecision, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return retryDecision{}, fmt.Errorf("failed to create cloudflare cache purge request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return retryDecision{retryable: true}, fmt.Errorf("failed to call cloudflare cache purge API: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, info.CloudflareAPIResponseMaxBytes))
	if err != nil {
		return retryDecision{}, fmt.Errorf("failed to read cloudflare cache purge response: %w", err)
	}

	var apiResponse struct {
		Success bool `json:"success"`
		Errors  []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(responseBody, &apiResponse); err != nil {
		return retryPolicy(resp), fmt.Errorf("cloudflare cache purge API returned status %d with an invalid response: %w", resp.StatusCode, err)
	}
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices && apiResponse.Success {
		return retryDecision{}, nil
	}

	errorMessages := make([]string, 0, len(apiResponse.Errors))
	for _, apiError := range apiResponse.Errors {
		errorMessages = append(errorMessages, fmt.Sprintf("%d: %s", apiError.Code, apiError.Message))
	}
	if len(errorMessages) == 0 {
		errorMessages = append(errorMessages, "unknown error")
	}
	return retryPolicy(resp), fmt.Errorf(
		"cloudflare cache purge API returned status %d: %s",
		resp.StatusCode,
		strings.Join(errorMessages, "; "),
	)
}

func retryPolicy(resp *http.Response) retryDecision {
	serverError := resp.StatusCode >= http.StatusInternalServerError && resp.StatusCode < 600
	if resp.StatusCode != http.StatusTooManyRequests && !serverError {
		return retryDecision{}
	}

	retryAfter := strings.TrimSpace(resp.Header.Get("Retry-After"))
	if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds >= 0 {
		return retryDecision{retryable: true, delay: time.Duration(seconds) * time.Second, delayProvided: true}
	}
	if retryAt, err := http.ParseTime(retryAfter); err == nil {
		return retryDecision{retryable: true, delay: max(time.Until(retryAt), time.Duration(0)), delayProvided: true}
	}
	return retryDecision{retryable: true}
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
