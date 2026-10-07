package middleware

import (
	"strconv"
	"sync"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/info"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/labstack/echo/v5"
)

// RateLimitConfig はレートリミットの設定を保持します
type RateLimitConfig struct {
	// Requests は時間枠内に許可されるリクエスト数です
	Requests int
	// Window は時間枠です
	Window time.Duration
}

// rateLimitEntry はFixed Window方式のレートリミット情報を保持します
type rateLimitEntry struct {
	Count       int
	WindowStart time.Time
	Limit       int
}

// FixedWindowStore はFixed Window方式のレートリミットストアです
type FixedWindowStore struct {
	mu      sync.RWMutex
	entries map[string]*rateLimitEntry
	window  time.Duration
}

var (
	fixedWindowStoreCleanupHook   = func(func()) {}
	fixedWindowStoreCleanupHookMu sync.RWMutex
)

// NewFixedWindowStore は新しいFixedWindowStoreを作成します
func NewFixedWindowStore(window time.Duration) *FixedWindowStore {
	return &FixedWindowStore{
		entries: make(map[string]*rateLimitEntry),
		window:  window,
	}
}

// newFixedWindowStoreWithCleanup はストア作成とクリーンアップgoroutineの起動をまとめます
func newFixedWindowStoreWithCleanup(window time.Duration) *FixedWindowStore {
	store := NewFixedWindowStore(window)
	ticker := time.NewTicker(window)
	done := make(chan struct{})
	var once sync.Once

	stop := func() {
		once.Do(func() {
			close(done)
			ticker.Stop()
		})
	}

	go func() {
		for {
			select {
			case <-ticker.C:
				store.Cleanup()
			case <-done:
				return
			}
		}
	}()

	fixedWindowStoreCleanupHookMu.RLock()
	hook := fixedWindowStoreCleanupHook
	fixedWindowStoreCleanupHookMu.RUnlock()
	hook(stop)

	return store
}

// Allow はリクエストを許可するか判定し、残り回数とリセット時刻を返します
func (s *FixedWindowStore) Allow(identifier string, limit int) (allowed bool, remaining int, resetTime time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	entry, exists := s.entries[identifier]

	if !exists || now.Sub(entry.WindowStart) >= s.window {
		entry = &rateLimitEntry{
			Count:       0,
			WindowStart: now,
			Limit:       limit,
		}
		s.entries[identifier] = entry
	}

	resetTime = entry.WindowStart.Add(s.window)

	if entry.Count >= entry.Limit {
		return false, 0, resetTime
	}

	entry.Count++
	remaining = entry.Limit - entry.Count

	return true, remaining, resetTime
}

// Cleanup は期限切れのエントリを削除します（メモリリーク防止）
func (s *FixedWindowStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for key, entry := range s.entries {
		if now.Sub(entry.WindowStart) >= s.window*2 {
			delete(s.entries, key)
		}
	}
}

// APIRateLimitMiddleware は外部API向けのレートリミットミドルウェアを提供します。
// ADMINアカウントはadminLimit、EDITOR/EXTDEVアカウントはeditorLimit、それ以外はnormalLimitが適用されます。
// レスポンスにX-RateLimit-*ヘッダーを追加します。
// このミドルウェアはAPITokenMiddlewareの後に使用することを想定しています。
func APIRateLimitMiddleware(normalLimit, editorLimit, adminLimit int, window time.Duration) echo.MiddlewareFunc {
	store := newFixedWindowStoreWithCleanup(window)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			userObj := c.Get("userEntity")
			if userObj == nil {
				return apierror.ErrUnauthorized
			}
			user, ok := userObj.(*entity.User)
			if !ok {
				return apierror.ErrUnauthorized
			}

			identifier := strconv.Itoa(user.ID)

			limit := normalLimit
			switch user.AccountTypeID {
			case info.AccountTypeAdmin:
				limit = adminLimit
			case info.AccountTypeEditor, info.AccountTypeExtDev:
				limit = editorLimit
			}

			allowed, remaining, resetTime := store.Allow(identifier, limit)

			c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			c.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			c.Response().Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

			if !allowed {
				return apierror.ErrTooManyRequests
			}

			return next(c)
		}
	}
}

// IPRateLimitMiddleware はIPアドレスベースのレートリミットミドルウェアを提供します。
// 未認証ユーザーのエンドポイント保護などに使用します。
// このミドルウェアはヘッダーを追加しません（外部APIのみヘッダー追加）。
func IPRateLimitMiddleware(config RateLimitConfig) echo.MiddlewareFunc {
	store := newFixedWindowStoreWithCleanup(config.Window)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			identifier := c.RealIP()

			allowed, _, _ := store.Allow(identifier, config.Requests)

			if !allowed {
				return apierror.ErrTooManyRequests
			}

			return next(c)
		}
	}
}

// UserRateLimitMiddleware はユーザーIDベースのレートリミットミドルウェアを提供します。
// 認証済みユーザー向けエンドポイントの保護に使用します。
func UserRateLimitMiddleware(config RateLimitConfig) echo.MiddlewareFunc {
	store := newFixedWindowStoreWithCleanup(config.Window)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			user, ok := c.Get("userEntity").(*entity.User)
			if !ok || user == nil {
				return apierror.ErrUnauthorized
			}

			identifier := strconv.Itoa(user.ID)

			allowed, _, _ := store.Allow(identifier, config.Requests)
			if !allowed {
				return apierror.ErrTooManyRequests
			}

			return next(c)
		}
	}
}

// AnonymousIPRateLimitMiddleware は未認証ユーザーにのみIPベースのレートリミットを適用します。
func AnonymousIPRateLimitMiddleware(config RateLimitConfig) echo.MiddlewareFunc {
	store := newFixedWindowStoreWithCleanup(config.Window)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			userObj := c.Get("userEntity")
			if userObj != nil {
				if _, ok := userObj.(*entity.User); !ok {
					return apierror.ErrUnauthorized
				}
				return next(c)
			}

			identifier := c.RealIP()

			allowed, _, _ := store.Allow(identifier, config.Requests)
			if !allowed {
				return apierror.ErrTooManyRequests
			}

			return next(c)
		}
	}
}
