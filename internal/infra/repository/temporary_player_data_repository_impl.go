package repository

import (
	"container/heap"
	"context"
	"sync"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

type temporaryPlayerDataRepository struct {
	mu                 sync.Mutex
	entriesByToken     map[string]*entity.TemporaryPlayerData
	tokensByIP         map[string]map[string]struct{}
	expiryItemsByToken map[string]*temporaryPlayerDataExpiryItem
	expiryHeap         temporaryPlayerDataExpiryHeap
	totalBytes         int
	maxEntriesPerIP    int
	maxTotalBytes      int
}

type temporaryPlayerDataExpiryItem struct {
	token     string
	expiresAt time.Time
	index     int
}

type temporaryPlayerDataExpiryHeap []*temporaryPlayerDataExpiryItem

func (h temporaryPlayerDataExpiryHeap) Len() int { return len(h) }

func (h temporaryPlayerDataExpiryHeap) Less(i, j int) bool {
	return h[i].expiresAt.Before(h[j].expiresAt)
}

func (h temporaryPlayerDataExpiryHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *temporaryPlayerDataExpiryHeap) Push(x any) {
	item, ok := x.(*temporaryPlayerDataExpiryItem)
	if !ok {
		return
	}
	item.index = len(*h)
	*h = append(*h, item)
}

func (h *temporaryPlayerDataExpiryHeap) Pop() any {
	old := *h
	n := len(old)
	if n == 0 {
		return nil
	}
	item := old[n-1]
	item.index = -1
	*h = old[:n-1]
	return item
}

// NewTemporaryPlayerDataRepository はインメモリ一時データリポジトリを生成します。
func NewTemporaryPlayerDataRepository(maxEntriesPerIP, maxTotalBytes int) domainrepo.TemporaryPlayerDataRepository {
	r := &temporaryPlayerDataRepository{
		entriesByToken:     make(map[string]*entity.TemporaryPlayerData),
		tokensByIP:         make(map[string]map[string]struct{}),
		expiryItemsByToken: make(map[string]*temporaryPlayerDataExpiryItem),
		maxEntriesPerIP:    maxEntriesPerIP,
		maxTotalBytes:      maxTotalBytes,
	}
	heap.Init(&r.expiryHeap)
	return r
}

func (r *temporaryPlayerDataRepository) Create(_ context.Context, _ domainrepo.Executor, data *entity.TemporaryPlayerData) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	r.cleanupExpiredLocked(now)

	ipTokens := r.tokensByIP[data.IPAddress]
	if len(ipTokens) >= r.maxEntriesPerIP {
		return domainrepo.ErrTemporaryPlayerDataPerIPLimitExceeded
	}

	payloadSize := len(data.Payload)
	if r.totalBytes+payloadSize > r.maxTotalBytes {
		return domainrepo.ErrTemporaryPlayerDataTotalSizeLimitExceeded
	}

	r.entriesByToken[data.Token] = data
	r.entriesByToken[copyData.Token] = &copyData
	if ipTokens == nil {
		ipTokens = make(map[string]struct{})
		r.tokensByIP[copyData.IPAddress] = ipTokens
	}
	ipTokens[copyData.Token] = struct{}{}

	expiryItem := &temporaryPlayerDataExpiryItem{token: copyData.Token, expiresAt: copyData.ExpiresAt}
	r.expiryItemsByToken[copyData.Token] = expiryItem
	heap.Push(&r.expiryHeap, expiryItem)

	r.totalBytes += payloadSize

	return nil
}

func (r *temporaryPlayerDataRepository) FindByToken(_ context.Context, _ domainrepo.Executor, token string) (*entity.TemporaryPlayerData, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	r.cleanupExpiredLocked(now)

	entry, ok := r.entriesByToken[token]
	if !ok {
		return nil, domainrepo.ErrTemporaryPlayerDataNotFound
	}

	copied := *entry
	copied.Payload = append([]byte(nil), entry.Payload...)
	return &copied, nil
}

func (r *temporaryPlayerDataRepository) ConsumeByToken(_ context.Context, _ domainrepo.Executor, token string) (*entity.TemporaryPlayerData, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	r.cleanupExpiredLocked(now)

	entry, ok := r.entriesByToken[token]
	if !ok {
		return nil, domainrepo.ErrTemporaryPlayerDataNotFound
	}

	copied := *entry
	copied.Payload = append([]byte(nil), entry.Payload...)
	r.deleteEntryLocked(token, entry)

	return &copied, nil
}

func (r *temporaryPlayerDataRepository) Delete(_ context.Context, _ domainrepo.Executor, token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	r.cleanupExpiredLocked(now)

	entry, ok := r.entriesByToken[token]
	if !ok {
		return domainrepo.ErrTemporaryPlayerDataNotFound
	}

	r.deleteEntryLocked(token, entry)
	return nil
}

func (r *temporaryPlayerDataRepository) cleanupExpiredLocked(now time.Time) {
	for r.expiryHeap.Len() > 0 {
		head := r.expiryHeap[0]
		if head.expiresAt.After(now) {
			return
		}

		heap.Pop(&r.expiryHeap)
		delete(r.expiryItemsByToken, head.token)

		entry, ok := r.entriesByToken[head.token]
		if !ok {
			continue
		}
		r.deleteEntryLocked(head.token, entry)
	}
}

func (r *temporaryPlayerDataRepository) deleteEntryLocked(token string, entry *entity.TemporaryPlayerData) {
	delete(r.entriesByToken, token)
	if ipTokens, ok := r.tokensByIP[entry.IPAddress]; ok {
		delete(ipTokens, token)
		if len(ipTokens) == 0 {
			delete(r.tokensByIP, entry.IPAddress)
		}
	}
	if expiryItem, ok := r.expiryItemsByToken[token]; ok {
		heap.Remove(&r.expiryHeap, expiryItem.index)
		delete(r.expiryItemsByToken, token)
	}

	r.totalBytes -= len(entry.Payload)
}

var _ domainrepo.TemporaryPlayerDataRepository = (*temporaryPlayerDataRepository)(nil)
