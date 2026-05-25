package usecase_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/master"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type honorNoopExecutor struct{}

func (e *honorNoopExecutor) GetContext(context.Context, any, string, ...any) error {
	return nil
}

func (e *honorNoopExecutor) SelectContext(context.Context, any, string, ...any) error {
	return nil
}

func (e *honorNoopExecutor) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, nil
}

func (e *honorNoopExecutor) NamedExecContext(context.Context, string, any) (sql.Result, error) {
	return nil, nil
}

func (e *honorNoopExecutor) Rebind(query string) string {
	return query
}

func (e *honorNoopExecutor) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, nil
}

func (e *honorNoopExecutor) QueryxContext(context.Context, string, ...any) (*sqlx.Rows, error) {
	return nil, nil
}

func (e *honorNoopExecutor) QueryRowxContext(context.Context, string, ...any) *sqlx.Row {
	return nil
}

type honorPassthroughTransactionManager struct {
	tx repository.Executor
}

func (m *honorPassthroughTransactionManager) Transactional(ctx context.Context, fn func(tx repository.Executor) error) error {
	return fn(m.tx)
}

type honorRepositoryMock struct {
	findByIDResult *entity.Honor
	findByIDErr    error
	createResult   *entity.Honor
	createErr      error
	saveErr        error
	deleteErr      error

	createCalled bool
	saveCalled   bool
	deleteCalled bool
	createdHonor *entity.Honor
	savedHonor   *entity.Honor
}

func (m *honorRepositoryMock) FindAll(context.Context, repository.Executor) ([]*entity.Honor, error) {
	return []*entity.Honor{}, nil
}

func (m *honorRepositoryMock) FindByID(context.Context, repository.Executor, int) (*entity.Honor, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	return m.findByIDResult, nil
}

func (m *honorRepositoryMock) Create(_ context.Context, _ repository.Executor, honor *entity.Honor) (*entity.Honor, error) {
	m.createCalled = true
	m.createdHonor = honor
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.createResult, nil
}

func (m *honorRepositoryMock) Save(_ context.Context, _ repository.Executor, honor *entity.Honor) error {
	m.saveCalled = true
	m.savedHonor = honor
	return m.saveErr
}

func (m *honorRepositoryMock) Delete(context.Context, repository.Executor, int) error {
	m.deleteCalled = true
	return m.deleteErr
}

func (m *honorRepositoryMock) EnsureHonor(context.Context, repository.Executor, string, int, *string) (int, error) {
	return 0, nil
}

func (m *honorRepositoryMock) DeletePlayerHonors(context.Context, repository.Executor, int) error {
	return nil
}

func (m *honorRepositoryMock) BulkAssignHonors(context.Context, repository.Executor, []repository.HonorAssignment) error {
	return nil
}

func newHonorUsecaseForTest(repo repository.HonorRepository, masters *masterdata.MasterDataMasters) usecase.HonorUsecase {
	exec := &honorNoopExecutor{}
	return usecase.NewHonorUsecase(
		repo,
		&masterDataMasterProviderMock{masters: masters},
		&honorPassthroughTransactionManager{tx: exec},
		exec,
	)
}

func TestHonorUsecase_CreateHonor_入力を正規化して称号を作成する(t *testing.T) {
	// Given
	repo := &honorRepositoryMock{}
	uc := newHonorUsecaseForTest(repo, &masterdata.MasterDataMasters{
		HonorTypes: map[string]master.HonorType{
			"gold": {ID: 2, Name: "gold"},
		},
	})
	repo.createResult = &entity.Honor{ID: 10, Name: "称号A", HonorTypeID: 2, TypeName: "gold", ImageURL: "https://example.com/honor.png"}

	// When
	got, err := uc.CreateHonor(context.Background(), usecase.HonorInput{
		Name:     " 称号A ",
		TypeName: " gold ",
		ImageURL: " https://example.com/honor.png ",
	})

	// Then
	require.NoError(t, err)
	assert.Equal(t, 10, got.ID)
	require.NotNil(t, repo.createdHonor)
	assert.Equal(t, "称号A", repo.createdHonor.Name)
	assert.Equal(t, 2, repo.createdHonor.HonorTypeID)
	assert.Equal(t, "gold", repo.createdHonor.TypeName)
	assert.Equal(t, "https://example.com/honor.png", repo.createdHonor.ImageURL)
}

func TestHonorUsecase_CreateHonor_未知のTypeNameはErrInvalidHonorInputを返す(t *testing.T) {
	// Given
	repo := &honorRepositoryMock{}
	uc := newHonorUsecaseForTest(repo, &masterdata.MasterDataMasters{
		HonorTypes: map[string]master.HonorType{
			"gold": {ID: 2, Name: "gold"},
		},
	})

	// When
	got, err := uc.CreateHonor(context.Background(), usecase.HonorInput{
		Name:     "称号A",
		TypeName: "unknown",
	})

	// Then
	assert.Nil(t, got)
	assert.ErrorIs(t, err, usecase.ErrInvalidHonorInput)
	assert.False(t, repo.createCalled)
}

func TestHonorUsecase_UpdateHonor_既存称号を変更して保存する(t *testing.T) {
	// Given
	repo := &honorRepositoryMock{
		findByIDResult: &entity.Honor{ID: 10, Name: "旧称号", HonorTypeID: 1, TypeName: "normal", ImageURL: ""},
	}
	uc := newHonorUsecaseForTest(repo, &masterdata.MasterDataMasters{
		HonorTypes: map[string]master.HonorType{
			"gold": {ID: 2, Name: "gold"},
		},
	})

	// When
	got, err := uc.UpdateHonor(context.Background(), 10, usecase.HonorInput{
		Name:     " 新称号 ",
		TypeName: " gold ",
		ImageURL: " https://example.com/new.png ",
	})

	// Then
	require.NoError(t, err)
	assert.Equal(t, "新称号", got.Name)
	require.NotNil(t, repo.savedHonor)
	assert.Equal(t, 10, repo.savedHonor.ID)
	assert.Equal(t, "新称号", repo.savedHonor.Name)
	assert.Equal(t, 2, repo.savedHonor.HonorTypeID)
	assert.Equal(t, "gold", repo.savedHonor.TypeName)
	assert.Equal(t, "https://example.com/new.png", repo.savedHonor.ImageURL)
}

func TestHonorUsecase_UpdateHonor_不正IDはErrInvalidHonorInputを返す(t *testing.T) {
	// Given
	repo := &honorRepositoryMock{}
	uc := newHonorUsecaseForTest(repo, &masterdata.MasterDataMasters{})

	// When
	got, err := uc.UpdateHonor(context.Background(), 0, usecase.HonorInput{Name: "称号A", TypeName: "gold"})

	// Then
	assert.Nil(t, got)
	assert.ErrorIs(t, err, usecase.ErrInvalidHonorInput)
	assert.False(t, repo.saveCalled)
}

func TestHonorUsecase_DeleteHonor_リポジトリエラーを返す(t *testing.T) {
	// Given
	repo := &honorRepositoryMock{deleteErr: repository.ErrHonorConflict}
	uc := newHonorUsecaseForTest(repo, &masterdata.MasterDataMasters{})

	// When
	err := uc.DeleteHonor(context.Background(), 10)

	// Then
	assert.ErrorIs(t, err, repository.ErrHonorConflict)
	assert.True(t, repo.deleteCalled)
}

func TestHonorUsecase_DeleteHonor_不正IDはErrInvalidHonorInputを返す(t *testing.T) {
	// Given
	repo := &honorRepositoryMock{deleteErr: errors.New("should not be called")}
	uc := newHonorUsecaseForTest(repo, &masterdata.MasterDataMasters{})

	// When
	err := uc.DeleteHonor(context.Background(), 0)

	// Then
	assert.ErrorIs(t, err, usecase.ErrInvalidHonorInput)
	assert.False(t, repo.deleteCalled)
}
