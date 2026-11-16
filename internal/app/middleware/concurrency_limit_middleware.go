package middleware

import (
	"strconv"
	"sync"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/labstack/echo/v5"
)

// concurrencyLimiter は送信元ごとと全体の処理中リクエスト数を数えます
type concurrencyLimiter struct {
	mu             sync.Mutex
	inFlight       map[string]int
	total          int
	perClientLimit int
	globalLimit    int
}

// acquire は上限内であれば処理中の枠を1つ確保します
func (l *concurrencyLimiter) acquire(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.total >= l.globalLimit || l.inFlight[key] >= l.perClientLimit {
		return false
	}
	l.inFlight[key]++
	l.total++
	return true
}

// release は確保した枠を返却します。処理中が0件になった送信元はマップから消し、メモリの増加を防ぎます
func (l *concurrencyLimiter) release(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.inFlight[key]--
	if l.inFlight[key] == 0 {
		delete(l.inFlight, key)
	}
	l.total--
}

// ConcurrencyLimitMiddleware は重い参照処理の同時実行数を送信元ごとと全体で制限します。
// 認証済みならユーザーID、未認証なら送信元IPを単位とします。重い処理がDB接続を占有し続けないよう、
// 上限超過時は待たせずに429を返します。同じ上限を共有するルートには同じインスタンスを渡してください。
func ConcurrencyLimitMiddleware(perClientLimit, globalLimit int) echo.MiddlewareFunc {
	limiter := &concurrencyLimiter{
		inFlight:       make(map[string]int),
		perClientLimit: perClientLimit,
		globalLimit:    globalLimit,
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// ユーザーIDとIPが同じ文字列にならないよう、種別の接頭辞を付けます。
			key := "ip:" + ClientIPKey(c)
			if userObj := c.Get("userEntity"); userObj != nil {
				user, ok := userObj.(*entity.User)
				if !ok {
					return apierror.ErrUnauthorized
				}
				key = "user:" + strconv.Itoa(user.ID)
			}

			if !limiter.acquire(key) {
				return apierror.ErrTooManyRequests
			}
			defer limiter.release(key)

			return next(c)
		}
	}
}
