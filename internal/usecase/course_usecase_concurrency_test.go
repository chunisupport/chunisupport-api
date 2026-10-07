package usecase

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type courseMutationTx struct {
	repository.Executor
	locked bool
}

type courseMutationRepository struct {
	*courseRepositoryStub
	row       sync.Mutex
	first     sync.Once
	attempted chan struct{}
	acquired  chan struct{}
	release   chan struct{}
	course    entity.Course
}

func (r *courseMutationRepository) Transactional(_ context.Context, fn func(repository.Executor) error) error {
	tx := &courseMutationTx{}
	defer func() {
		if tx.locked {
			r.row.Unlock()
		}
	}()
	return fn(tx)
}

func (r *courseMutationRepository) FindByDisplayIDForUpdate(_ context.Context, exec repository.Executor, _ string) (*entity.Course, error) {
	tx, ok := exec.(*courseMutationTx)
	if !ok {
		return nil, errors.New("transaction required")
	}
	r.attempted <- struct{}{}
	r.row.Lock()
	tx.locked = true
	r.first.Do(func() {
		close(r.acquired)
		<-r.release
	})
	course := r.course
	return &course, nil
}

func (r *courseMutationRepository) Save(_ context.Context, exec repository.Executor, course *entity.Course) error {
	tx, ok := exec.(*courseMutationTx)
	if !ok || !tx.locked {
		return errors.New("locked transaction required")
	}
	r.course = *course
	return nil
}

func (r *courseMutationRepository) FindClassByName(_ context.Context, exec repository.Executor, _ string) (*entity.CourseClass, error) {
	tx, ok := exec.(*courseMutationTx)
	if !ok || !tx.locked {
		return nil, errors.New("locked transaction required")
	}
	return &entity.CourseClass{ID: 7, Name: "extra"}, nil
}

func (r *courseMutationRepository) FindByOfficialIdx(_ context.Context, exec repository.Executor, _ string, _ bool) (*entity.Course, error) {
	tx, ok := exec.(*courseMutationTx)
	if !ok || !tx.locked {
		return nil, errors.New("locked transaction required")
	}
	course := r.course
	return &course, nil
}

func TestCourseUsecase_ConcurrentMutationsPreserveIndependentFields(t *testing.T) {
	tests := []struct {
		name string
		// Given
		deleted   bool
		editFirst bool
	}{
		{name: "編集が先に行ロックを取得しても削除状態を保持する", deleted: true, editFirst: true},
		{name: "削除が先に行ロックを取得しても編集内容を保持する", deleted: true, editFirst: false},
		{name: "編集が先に行ロックを取得しても復元状態を保持する", deleted: false, editFirst: true},
		{name: "復元が先に行ロックを取得しても編集内容を保持する", deleted: false, editFirst: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			id, err := displayid.NewDisplayID("0123456789abcdef")
			require.NoError(t, err)
			repo := &courseMutationRepository{
				courseRepositoryStub: &courseRepositoryStub{},
				attempted:            make(chan struct{}, 2),
				acquired:             make(chan struct{}),
				release:              make(chan struct{}),
				course:               entity.Course{DisplayID: id, OfficialIdx: "50020", Name: "変更前", CourseClassID: 1, IsDeleted: !tt.deleted},
			}
			var release sync.Once
			unblock := func() { release.Do(func() { close(repo.release) }) }
			defer unblock()
			uc := NewCourseUsecase(nil, repo, repo, nil, nil)
			edit := func() error {
				output, err := uc.Update(context.Background(), id.String(), UpdateCourseInput{Name: "変更後", Class: "extra"})
				if err == nil && (output.Name != "変更後" || output.Class != "extra") {
					return errors.New("unexpected edit response")
				}
				return err
			}
			state := func() error {
				if tt.deleted {
					return uc.Delete(context.Background(), id.String())
				}
				return uc.Restore(context.Background(), id.String())
			}
			first, second := edit, state
			if !tt.editFirst {
				first, second = state, edit
			}

			// When
			done := make(chan error, 2)
			go func() { done <- first() }()
			waitCourseSignal(t, repo.acquired)
			waitCourseSignal(t, repo.attempted)
			go func() { done <- second() }()
			waitCourseSignal(t, repo.attempted)
			unblock()

			// Then
			for range 2 {
				select {
				case err := <-done:
					require.NoError(t, err)
				case <-time.After(5 * time.Second):
					require.FailNow(t, "コース更新が完了しません")
				}
			}
			assert.Equal(t, "変更後", repo.course.Name)
			assert.Equal(t, 7, repo.course.CourseClassID)
			assert.Equal(t, tt.deleted, repo.course.IsDeleted)
		})
	}
}

func waitCourseSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "コースロックの同期が完了しません")
	}
}

type courseFailingTransactionManager struct {
	err           error
	afterCallback bool
}

func (m courseFailingTransactionManager) Transactional(_ context.Context, fn func(repository.Executor) error) error {
	if m.afterCallback {
		if err := fn(nil); err != nil {
			return err
		}
	}
	return m.err
}

func TestCourseUsecase_TransactionFailureDoesNotReturnSuccess(t *testing.T) {
	tests := []struct {
		name string
		// Given
		afterCallback bool
		// When
		operation string
	}{
		{name: "開始に失敗した編集はエラーを返す", operation: "update"},
		{name: "開始に失敗した削除はエラーを返す", operation: "delete"},
		{name: "開始に失敗した復元はエラーを返す", operation: "restore"},
		{name: "コミットに失敗した編集はエラーを返す", afterCallback: true, operation: "update"},
		{name: "コミットに失敗した削除はエラーを返す", afterCallback: true, operation: "delete"},
		{name: "コミットに失敗した復元はエラーを返す", afterCallback: true, operation: "restore"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			id, err := displayid.NewDisplayID("0123456789abcdef")
			require.NoError(t, err)
			repo := &courseRepositoryStub{displayIDCourse: &entity.Course{DisplayID: id, OfficialIdx: "50020", Name: "コース", CourseClassID: 1}}
			tm := courseFailingTransactionManager{err: context.Canceled, afterCallback: tt.afterCallback}
			uc := NewCourseUsecase(nil, tm, repo, nil, nil)

			// When
			var output *CourseOutput
			switch tt.operation {
			case "update":
				output, err = uc.Update(context.Background(), id.String(), UpdateCourseInput{Name: "変更後", Class: "1"})
			case "delete":
				err = uc.Delete(context.Background(), id.String())
			case "restore":
				err = uc.Restore(context.Background(), id.String())
			}

			// Then
			assert.ErrorIs(t, err, context.Canceled)
			assert.Nil(t, output)
		})
	}
}
