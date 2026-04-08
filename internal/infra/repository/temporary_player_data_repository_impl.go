package repository

import (
	"context"
	"sync"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

type temporaryPlayerDataRepository struct {
	mu              sync.Mutex
	entriesByToken  map[string]*entity.TemporaryPlayerData
	tokensByIP      map[string]map[string]struct{}
	totalBytes      int
	maxEntriesPerIP int
	maxTotalBytes   int
}

// NewTemporaryPlayerDataRepository はインメモリ一時データリポジトリを生成します。
func NewTemporaryPlayerDataRepository(maxEntriesPerIP, maxTotalBytes int) domainrepo.TemporaryPlayerDataRepository {
	return &temporaryPlayerDataRepository{
		entriesByToken:  make(map[string]*entity.TemporaryPlayerData),
		tokensByIP:      make(map[string]map[string]struct{}),
		maxEntriesPerIP: maxEntriesPerIP,
		maxTotalBytes:   maxTotalBytes,
	}
}

func (r *temporaryPlayerDataRepository) Create(_ context.Context, data *entity.TemporaryPlayerData) error {
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

	copyData := *data
	copyData.Payload = append([]byte(nil), data.Payload...)
	r.entriesByToken[copyData.Token] = &copyData
	if ipTokens == nil {
		ipTokens = make(map[string]struct{})
		r.tokensByIP[copyData.IPAddress] = ipTokens
	}
	ipTokens[copyData.Token] = struct{}{}
	r.totalBytes += payloadSize

	return nil
}

func (r *temporaryPlayerDataRepository) FindByToken(_ context.Context, token string) (*entity.TemporaryPlayerData, error) {
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

func (r *temporaryPlayerDataRepository) Delete(_ context.Context, token string) error {
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
	for token, entry := range r.entriesByToken {
		if entry.IsExpired(now) {
			r.deleteEntryLocked(token, entry)
		}
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
	r.totalBytes -= len(entry.Payload)
	if r.totalBytes < 0 {
		r.totalBytes = 0
	}
}

var _ domainrepo.TemporaryPlayerDataRepository = (*temporaryPlayerDataRepository)(nil)
