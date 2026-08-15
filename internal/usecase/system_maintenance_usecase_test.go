package usecase

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSystemMaintenanceUsecase_起動時の状態を読み込む(t *testing.T) {
	// Given
	updatedAt := time.Date(2026, time.July, 26, 3, 0, 0, 0, time.UTC)
	initial := mustRestoreSystemMaintenance(t, true, "データ更新中です", new(10), updatedAt)
	repo := &systemMaintenanceRepositoryStub{found: initial}

	// When
	maintenanceUsecase, err := NewSystemMaintenanceUsecase(context.Background(), repo)

	// Then
	require.NoError(t, err)
	assert.Equal(t, 1, repo.findCalls)
	assert.Equal(t, MaintenanceState{
		Enabled:   true,
		Comment:   "データ更新中です",
		UpdatedAt: updatedAt,
	}, maintenanceUsecase.Current())

	// リポジトリが返した可変エンティティや呼び出し元へ返した値から、公開中の状態を変更できないことを確認します。
	initial.Enabled = false
	initial.UpdatedAt = updatedAt.Add(time.Hour)
	copiedState := maintenanceUsecase.Current()
	copiedState.Enabled = false
	assert.True(t, maintenanceUsecase.Current().Enabled)
	assert.Equal(t, updatedAt, maintenanceUsecase.Current().UpdatedAt)
	assert.Equal(t, 1, repo.findCalls)
}

func TestNewSystemMaintenanceUsecase_初期化失敗を返す(t *testing.T) {
	findErr := errors.New("find failed")
	tests := []struct {
		name    string
		repo    repository.SystemMaintenanceRepository
		wantErr error
	}{
		{
			name:    "リポジトリがnil",
			wantErr: ErrInternalError,
		},
		{
			name:    "Findが失敗",
			repo:    &systemMaintenanceRepositoryStub{findErr: findErr},
			wantErr: findErr,
		},
		{
			name:    "Findがnilを返す",
			repo:    &systemMaintenanceRepositoryStub{},
			wantErr: ErrInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got, err := NewSystemMaintenanceUsecase(context.Background(), tt.repo)

			// Then
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, got)
		})
	}
}

func TestSystemMaintenanceUsecase_Update_開始状態を保存後に公開する(t *testing.T) {
	// Given
	initialAt := time.Date(2026, time.July, 26, 3, 0, 0, 0, time.UTC)
	updateAt := initialAt.Add(time.Hour)
	initial := mustRestoreSystemMaintenance(t, false, "", nil, initialAt)
	repo := &systemMaintenanceRepositoryStub{found: initial}
	var logBuffer bytes.Buffer
	maintenanceUsecase := mustNewSystemMaintenanceUsecase(
		t,
		repo,
		fixedClock{now: updateAt},
		slog.New(slog.NewTextHandler(&logBuffer, nil)),
	)

	// When
	got, err := maintenanceUsecase.Update(
		context.Background(),
		42,
		true,
		"  データ更新中です\r\nしばらくお待ちください。  ",
	)

	// Then
	require.NoError(t, err)
	assert.Equal(t, MaintenanceState{
		Enabled:   true,
		Comment:   "データ更新中です\nしばらくお待ちください。",
		UpdatedAt: updateAt,
	}, got)
	assert.Equal(t, got, maintenanceUsecase.Current())
	require.Len(t, repo.saved, 1)
	assert.True(t, repo.saved[0].Enabled)
	assert.Equal(t, "データ更新中です\nしばらくお待ちください。", repo.saved[0].Comment.String())
	assert.Equal(t, new(42), repo.saved[0].UpdatedByUserID)
	assert.Equal(t, updateAt, repo.saved[0].UpdatedAt)
	assert.Contains(t, logBuffer.String(), "メンテナンス状態を更新しました")
	assert.Contains(t, logBuffer.String(), "actor_user_id=42")
	assert.NotContains(t, logBuffer.String(), "データ更新中です")
}

func TestSystemMaintenanceUsecase_Update_終了時は入力コメントを保存しない(t *testing.T) {
	// Given
	initialAt := time.Date(2026, time.July, 26, 3, 0, 0, 0, time.UTC)
	updateAt := initialAt.Add(time.Hour)
	initial := mustRestoreSystemMaintenance(t, true, "データ更新中です", new(10), initialAt)
	repo := &systemMaintenanceRepositoryStub{found: initial}
	maintenanceUsecase := mustNewSystemMaintenanceUsecase(
		t,
		repo,
		fixedClock{now: updateAt},
		discardMaintenanceLogger(),
	)

	// When
	got, err := maintenanceUsecase.Update(context.Background(), 42, false, "この入力は保存しない")

	// Then
	require.NoError(t, err)
	assert.Equal(t, MaintenanceState{Enabled: false, Comment: "", UpdatedAt: updateAt}, got)
	require.Len(t, repo.saved, 1)
	assert.False(t, repo.saved[0].Enabled)
	assert.Empty(t, repo.saved[0].Comment.String())
	assert.Equal(t, new(42), repo.saved[0].UpdatedByUserID)
	assert.Equal(t, got, maintenanceUsecase.Current())
}

func TestSystemMaintenanceUsecase_Update_更新日時をUTCのマイクロ秒精度へ揃える(t *testing.T) {
	// Given
	initialAt := time.Date(2026, time.July, 26, 3, 0, 0, 0, time.UTC)
	location := time.FixedZone("JST", 9*60*60)
	clockTime := time.Date(2026, time.July, 26, 12, 30, 0, 123456789, location)
	expectedAt := time.Date(2026, time.July, 26, 3, 30, 0, 123456000, time.UTC)
	initial := mustRestoreSystemMaintenance(t, false, "", nil, initialAt)
	repo := &systemMaintenanceRepositoryStub{found: initial}
	maintenanceUsecase := mustNewSystemMaintenanceUsecase(
		t,
		repo,
		fixedClock{now: clockTime},
		discardMaintenanceLogger(),
	)

	// When
	got, err := maintenanceUsecase.Update(context.Background(), 42, true, "データ更新中です")

	// Then
	require.NoError(t, err)
	assert.Equal(t, expectedAt, got.UpdatedAt)
	assert.Equal(t, expectedAt, maintenanceUsecase.Current().UpdatedAt)
	require.Len(t, repo.saved, 1)
	assert.Equal(t, expectedAt, repo.saved[0].UpdatedAt)
}

func TestSystemMaintenanceUsecase_Update_不正な開始コメントなら保存も公開もしない(t *testing.T) {
	// Given
	initialAt := time.Date(2026, time.July, 26, 3, 0, 0, 0, time.UTC)
	initial := mustRestoreSystemMaintenance(t, false, "", nil, initialAt)
	repo := &systemMaintenanceRepositoryStub{found: initial}
	maintenanceUsecase := mustNewSystemMaintenanceUsecase(
		t,
		repo,
		fixedClock{now: initialAt.Add(time.Hour)},
		discardMaintenanceLogger(),
	)

	// When
	got, err := maintenanceUsecase.Update(context.Background(), 42, true, " \r\n ")

	// Then
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidMaintenanceComment)
	assert.Zero(t, got)
	assert.Empty(t, repo.saved)
	assert.Equal(t, MaintenanceState{Enabled: false, Comment: "", UpdatedAt: initialAt}, maintenanceUsecase.Current())
}

func TestSystemMaintenanceUsecase_Update_DB保存失敗時は公開状態を変更しない(t *testing.T) {
	// Given
	initialAt := time.Date(2026, time.July, 26, 3, 0, 0, 0, time.UTC)
	saveErr := errors.New("save failed")
	initial := mustRestoreSystemMaintenance(t, false, "", nil, initialAt)
	repo := &systemMaintenanceRepositoryStub{found: initial, saveErr: saveErr}
	var logBuffer bytes.Buffer
	maintenanceUsecase := mustNewSystemMaintenanceUsecase(
		t,
		repo,
		fixedClock{now: initialAt.Add(time.Hour)},
		slog.New(slog.NewTextHandler(&logBuffer, nil)),
	)

	// When
	got, err := maintenanceUsecase.Update(context.Background(), 42, true, "データ更新中です")

	// Then
	require.Error(t, err)
	assert.ErrorIs(t, err, saveErr)
	assert.Zero(t, got)
	assert.Equal(t, MaintenanceState{Enabled: false, Comment: "", UpdatedAt: initialAt}, maintenanceUsecase.Current())
	assert.Empty(t, logBuffer.String())
}

func TestSystemMaintenanceUsecase_Update_同一の最終状態は何も更新しない(t *testing.T) {
	tests := []struct {
		name           string
		initialEnabled bool
		initialComment string
		enabled        bool
		comment        string
	}{
		{
			name:           "正規化後の開始コメントが同じ",
			initialEnabled: true,
			initialComment: "データ更新中です\nしばらくお待ちください。",
			enabled:        true,
			comment:        "  データ更新中です\r\nしばらくお待ちください。  ",
		},
		{
			name:           "終了済みの場合は入力コメントを無視して同じ状態になる",
			initialEnabled: false,
			initialComment: "",
			enabled:        false,
			comment:        "保存されないコメント",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			initialAt := time.Date(2026, time.July, 26, 3, 0, 0, 0, time.UTC)
			initial := mustRestoreSystemMaintenance(t, tt.initialEnabled, tt.initialComment, new(10), initialAt)
			repo := &systemMaintenanceRepositoryStub{found: initial}
			var logBuffer bytes.Buffer
			maintenanceUsecase := mustNewSystemMaintenanceUsecase(
				t,
				repo,
				fixedClock{now: initialAt.Add(time.Hour)},
				slog.New(slog.NewTextHandler(&logBuffer, nil)),
			)

			// When
			got, err := maintenanceUsecase.Update(context.Background(), 42, tt.enabled, tt.comment)

			// Then
			require.NoError(t, err)
			assert.Equal(t, MaintenanceState{
				Enabled:   tt.initialEnabled,
				Comment:   tt.initialComment,
				UpdatedAt: initialAt,
			}, got)
			assert.Empty(t, repo.saved)
			assert.Empty(t, logBuffer.String())
			assert.Equal(t, got, maintenanceUsecase.Current())
		})
	}
}

func TestSystemMaintenanceUsecase_Update_並行更新を保存完了順に直列化する(t *testing.T) {
	// Given
	initialAt := time.Date(2026, time.July, 26, 3, 0, 0, 0, time.UTC)
	firstAt := initialAt.Add(time.Hour)
	secondAt := firstAt.Add(time.Hour)
	initial := mustRestoreSystemMaintenance(t, false, "", nil, initialAt)
	repo := newBlockingSystemMaintenanceRepository(initial)
	maintenanceUsecase := mustNewSystemMaintenanceUsecase(
		t,
		repo,
		&sequenceMaintenanceClock{times: []time.Time{firstAt, secondAt}},
		discardMaintenanceLogger(),
	)

	firstDone := make(chan error, 1)
	go func() {
		_, err := maintenanceUsecase.Update(context.Background(), 10, true, "データ更新中です")
		firstDone <- err
	}()
	require.Equal(t, 1, <-repo.entered)

	secondDone := make(chan error, 1)
	go func() {
		_, err := maintenanceUsecase.Update(context.Background(), 20, false, "無視されます")
		secondDone <- err
	}()

	// Then: 1件目のSave中は公開状態を変えず、2件目もSaveへ進みません。
	assert.Equal(t, MaintenanceState{Enabled: false, Comment: "", UpdatedAt: initialAt}, maintenanceUsecase.Current())
	select {
	case call := <-repo.entered:
		require.Fail(t, "2件目の更新が直列化されていません", "save call %d", call)
	case <-time.After(50 * time.Millisecond):
	}

	repo.release <- struct{}{}
	require.NoError(t, <-firstDone)
	require.Equal(t, 2, <-repo.entered)
	assert.Equal(t, MaintenanceState{Enabled: true, Comment: "データ更新中です", UpdatedAt: firstAt}, maintenanceUsecase.Current())
	repo.release <- struct{}{}
	require.NoError(t, <-secondDone)

	require.Len(t, repo.savedEntities(), 2)
	assert.True(t, repo.savedEntities()[0].Enabled)
	assert.False(t, repo.savedEntities()[1].Enabled)
	assert.Equal(t, secondAt, repo.savedEntities()[1].UpdatedAt)
	assert.Equal(t, MaintenanceState{Enabled: false, Comment: "", UpdatedAt: secondAt}, maintenanceUsecase.Current())
}

func mustNewSystemMaintenanceUsecase(
	t *testing.T,
	repo repository.SystemMaintenanceRepository,
	clock clock,
	logger *slog.Logger,
) SystemMaintenanceUsecase {
	t.Helper()

	got, err := newSystemMaintenanceUsecase(context.Background(), repo, clock, logger)
	require.NoError(t, err)
	return got
}

func mustRestoreSystemMaintenance(
	t *testing.T,
	enabled bool,
	comment string,
	updatedByUserID *int,
	updatedAt time.Time,
) *entity.SystemMaintenance {
	t.Helper()

	got, err := entity.RestoreSystemMaintenance(1, enabled, comment, updatedByUserID, updatedAt)
	require.NoError(t, err)
	return got
}

func discardMaintenanceLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type systemMaintenanceRepositoryStub struct {
	found     *entity.SystemMaintenance
	findErr   error
	saveErr   error
	findCalls int
	saved     []*entity.SystemMaintenance
}

func (r *systemMaintenanceRepositoryStub) Find(context.Context) (*entity.SystemMaintenance, error) {
	r.findCalls++
	return r.found, r.findErr
}

func (r *systemMaintenanceRepositoryStub) Save(_ context.Context, maintenance *entity.SystemMaintenance) error {
	r.saved = append(r.saved, cloneSystemMaintenanceForTest(maintenance))
	return r.saveErr
}

type blockingSystemMaintenanceRepository struct {
	found   *entity.SystemMaintenance
	entered chan int
	release chan struct{}
	mu      sync.Mutex
	saved   []*entity.SystemMaintenance
}

func newBlockingSystemMaintenanceRepository(found *entity.SystemMaintenance) *blockingSystemMaintenanceRepository {
	return &blockingSystemMaintenanceRepository{
		found:   found,
		entered: make(chan int, 2),
		release: make(chan struct{}, 2),
	}
}

func (r *blockingSystemMaintenanceRepository) Find(context.Context) (*entity.SystemMaintenance, error) {
	return r.found, nil
}

func (r *blockingSystemMaintenanceRepository) Save(_ context.Context, maintenance *entity.SystemMaintenance) error {
	r.mu.Lock()
	r.saved = append(r.saved, cloneSystemMaintenanceForTest(maintenance))
	call := len(r.saved)
	r.mu.Unlock()
	r.entered <- call
	<-r.release
	return nil
}

func (r *blockingSystemMaintenanceRepository) savedEntities() []*entity.SystemMaintenance {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]*entity.SystemMaintenance(nil), r.saved...)
}

type sequenceMaintenanceClock struct {
	mu    sync.Mutex
	times []time.Time
}

func (c *sequenceMaintenanceClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.times[0]
	c.times = c.times[1:]
	return now
}

func cloneSystemMaintenanceForTest(source *entity.SystemMaintenance) *entity.SystemMaintenance {
	if source == nil {
		return nil
	}

	cloned := *source
	if source.UpdatedByUserID != nil {
		updaterUserID := *source.UpdatedByUserID
		cloned.UpdatedByUserID = &updaterUserID
	}
	return &cloned
}

var _ repository.SystemMaintenanceRepository = (*systemMaintenanceRepositoryStub)(nil)
var _ repository.SystemMaintenanceRepository = (*blockingSystemMaintenanceRepository)(nil)
