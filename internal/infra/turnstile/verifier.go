package turnstile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

const (
	siteverifyEndpoint = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	requestTimeout     = 5 * time.Second
)

type verifier struct {
	secretKey string
	endpoint  string
	client    *http.Client
}

type siteverifyResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

// NewVerifier はCloudflare Turnstileの検証器を生成します。
func NewVerifier(secretKey string) usecase.TurnstileVerifier {
	return &verifier{
		secretKey: strings.TrimSpace(secretKey),
		endpoint:  siteverifyEndpoint,
		client: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

func (v *verifier) VerifyTurnstile(ctx context.Context, token string, remoteIP string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return usecase.ErrInvalidTurnstileToken
	}

	form := url.Values{}
	form.Set("secret", v.secretKey)
	form.Set("response", token)
	if remoteIP = strings.TrimSpace(remoteIP); remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return errors.Join(usecase.ErrInternalError, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := v.client
	if client == nil {
		client = &http.Client{Timeout: requestTimeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Join(usecase.ErrInternalError, err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		if err := resp.Body.Close(); err != nil {
			slog.Warn("failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return errors.Join(usecase.ErrInternalError, fmt.Errorf("turnstile siteverify returned status %d", resp.StatusCode))
	}

	var result siteverifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return errors.Join(usecase.ErrInternalError, err)
	}
	if !result.Success {
		slog.Warn("turnstile verification failed", "error_codes", result.ErrorCodes)
		return usecase.ErrInvalidTurnstileToken
	}

	return nil
}

var _ usecase.TurnstileVerifier = (*verifier)(nil)
