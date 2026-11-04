package xapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/dghubble/oauth1"

	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

const (
	authenticatedUserPath = "/2/users/me"
	maxResponseBytes      = 10 * 1024
)

type userClient struct {
	config  *oauth1.Config
	baseURL string
	base    *http.Client
}

type authenticatedUserResponse struct {
	Data struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	} `json:"data"`
}

// NewUserClient は利用者本人の OAuth 1.0a トークンで X API を呼び出す XUserClient を生成します。
// Firebase の X ログインが発行するトークンは OAuth 1.0a のため、同じ X アプリのコンシューマーキーで署名します。
// 本人情報の取得には X API で無料とされている GET /2/users/me だけを使います。
func NewUserClient(consumerKey string, consumerSecret string) usecase.XUserClient {
	return newUserClient(consumerKey, consumerSecret, info.XAPIBaseURL, http.DefaultClient)
}

func newUserClient(consumerKey string, consumerSecret string, baseURL string, base *http.Client) *userClient {
	return &userClient{
		config:  oauth1.NewConfig(consumerKey, consumerSecret),
		baseURL: strings.TrimRight(baseURL, "/"),
		base:    base,
	}
}

func (c *userClient) GetAuthenticatedUser(ctx context.Context, accessToken string, accessTokenSecret string) (usecase.XAuthenticatedUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+authenticatedUserPath, nil)
	if err != nil {
		return usecase.XAuthenticatedUser{}, fmt.Errorf("%w: build request: %w", usecase.ErrXAPIUnavailable, err)
	}

	client := c.config.Client(context.WithValue(ctx, oauth1.HTTPClient, c.base), oauth1.NewToken(accessToken, accessTokenSecret))
	client.Timeout = info.XAPIRequestTimeout
	resp, err := client.Do(req)
	if err != nil {
		return usecase.XAuthenticatedUser{}, fmt.Errorf("%w: request: %w", usecase.ErrXAPIUnavailable, err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		if err := resp.Body.Close(); err != nil {
			slog.WarnContext(ctx, "failed to close x api response body", "error", err)
		}
	}()

	// 401 はトークンの失効・取り消しや署名不一致で返るため、利用者が X で認証し直せば解消します。
	// それ以外の失敗は X 側の障害や回数制限、アプリ設定の問題として扱います。
	if resp.StatusCode == http.StatusUnauthorized {
		return usecase.XAuthenticatedUser{}, usecase.ErrInvalidXCredentials
	}
	if resp.StatusCode != http.StatusOK {
		return usecase.XAuthenticatedUser{}, fmt.Errorf("%w: x api returned status %d", usecase.ErrXAPIUnavailable, resp.StatusCode)
	}

	var body authenticatedUserResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&body); err != nil {
		return usecase.XAuthenticatedUser{}, fmt.Errorf("%w: decode response: %w", usecase.ErrXAPIUnavailable, err)
	}
	// 本人情報のない成功応答は X 側の異常であり、トークンの問題として利用者に再認証させても解消しない。
	if body.Data.ID == "" {
		return usecase.XAuthenticatedUser{}, fmt.Errorf("%w: response has no user", usecase.ErrXAPIUnavailable)
	}

	return usecase.XAuthenticatedUser{ID: body.Data.ID, Username: body.Data.Username}, nil
}
