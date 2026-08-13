package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type transferCodecStub struct {
	encoded []byte
	decoded *entity.UserDataTransferSnapshot
}

func (s *transferCodecStub) Encode(snapshot *entity.UserDataTransferSnapshot) ([]byte, error) {
	return append([]byte(nil), s.encoded...), nil
}
func (s *transferCodecStub) Decode(encoded []byte) (*entity.UserDataTransferSnapshot, error) {
	return s.decoded, nil
}

type transferRepositoryStub struct {
	snapshot         *entity.UserDataTransferSnapshot
	unresolved       []string
	empty            bool
	importedPlayerID int
	importCalled     bool
}

func (s *transferRepositoryStub) ExportSnapshot(context.Context, int) (*entity.UserDataTransferSnapshot, error) {
	return s.snapshot, nil
}
func (s *transferRepositoryStub) FindUnresolvedReferences(context.Context, *entity.UserDataTransferSnapshot) ([]string, error) {
	return append([]string(nil), s.unresolved...), nil
}
func (s *transferRepositoryStub) IsDestinationEmpty(context.Context, int) (bool, error) {
	return s.empty, nil
}
func (s *transferRepositoryStub) ImportSnapshot(context.Context, int, *entity.UserDataTransferSnapshot) (int, error) {
	s.importCalled = true
	return s.importedPlayerID, nil
}

func TestUserDataTransferUsecaseValidateReturnsBlockersAndLimitsReferences(t *testing.T) {
	snapshot := validTransferSnapshot(t)
	unresolved := make([]string, 105)
	for i := range unresolved {
		unresolved[i] = fmt.Sprintf("song:%03d", i)
	}
	repo := &transferRepositoryStub{snapshot: snapshot, unresolved: unresolved, empty: false}
	codec := &transferCodecStub{decoded: snapshot}

	usecase := NewUserDataTransferUsecase(codec, repo)

	output, err := usecase.Validate(context.Background(), 1, []byte("signed"))

	require.NoError(t, err)
	assert.False(t, output.Importable)
	assert.Equal(t, []string{"destination_not_empty", "unresolved_references"}, output.Blockers)
	assert.Len(t, output.UnresolvedReferences, 100)
	assert.Equal(t, 105, output.UnresolvedReferenceCount)
	assert.Equal(t, "テスト", output.PlayerName)
}

func TestUserDataTransferUsecaseImportLocksAndSavesInTransaction(t *testing.T) {
	snapshot := validTransferSnapshot(t)
	repo := &transferRepositoryStub{snapshot: snapshot, empty: true, importedPlayerID: 42}
	codec := &transferCodecStub{decoded: snapshot}

	usecase := NewUserDataTransferUsecase(codec, repo)

	output, err := usecase.Import(context.Background(), 1, []byte("signed"))

	require.NoError(t, err)
	assert.True(t, repo.importCalled)
	assert.Equal(t, 42, output.PlayerID)
}

func TestUserDataTransferUsecaseExportEncodesSnapshot(t *testing.T) {
	snapshot := validTransferSnapshot(t)
	repo := &transferRepositoryStub{snapshot: snapshot}
	codec := &transferCodecStub{encoded: []byte("exported")}
	usecaseImpl := NewUserDataTransferUsecase(codec, repo)

	output, err := usecaseImpl.Export(context.Background(), 1)

	require.NoError(t, err)
	assert.Equal(t, []byte("exported"), output.File)
}

func TestUserDataTransferUsecaseExportPreservesSnapshotValidationReason(t *testing.T) {
	snapshot := validTransferSnapshot(t)
	imageURL := ""
	snapshot.Honors = []entity.UserDataTransferHonor{{
		Slot:       1,
		ImageURL:   &imageURL,
		Name:       "称号",
		TypeName:   "normal",
		EquippedAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	}}
	transfer := NewUserDataTransferUsecase(&transferCodecStub{}, &transferRepositoryStub{snapshot: snapshot})

	_, err := transfer.Export(context.Background(), 1)

	assert.ErrorIs(t, err, ErrDataTransferInvalidData)
	assert.ErrorIs(t, err, entity.ErrInvalidUserDataTransfer)
	assert.Contains(t, err.Error(), "honors[0].image_url is invalid")
}

type transferGoalValidatorStub struct {
	called bool
}

func (s *transferGoalValidatorStub) ValidateTransferredGoals(context.Context, entity.UserDataTransferGoals) error {
	s.called = true
	return nil
}

func TestUserDataTransferUsecaseValidateReportsUnresolvedReferencesBeforeGoalValidation(t *testing.T) {
	snapshot := validTransferSnapshot(t)
	repo := &transferRepositoryStub{unresolved: []string{"goal.diff:UNKNOWN"}, empty: true}
	codec := &transferCodecStub{decoded: snapshot}
	validator := &transferGoalValidatorStub{}
	transfer := NewUserDataTransferUsecase(codec, repo, validator)

	output, err := transfer.Validate(context.Background(), 1, []byte("signed"))

	require.NoError(t, err)
	assert.False(t, output.Importable)
	assert.Equal(t, []string{"unresolved_references"}, output.Blockers)
	assert.False(t, validator.called)
}

func TestUserDataTransferUsecaseImportRejectsUnresolvedReferencesBeforeSaving(t *testing.T) {
	snapshot := validTransferSnapshot(t)
	repo := &transferRepositoryStub{unresolved: []string{"song:999"}, empty: true}
	codec := &transferCodecStub{decoded: snapshot}
	validator := &transferGoalValidatorStub{}
	transfer := NewUserDataTransferUsecase(codec, repo, validator)

	_, err := transfer.Import(context.Background(), 1, []byte("signed"))

	assert.ErrorIs(t, err, ErrDataTransferUnresolvedReference)
	assert.False(t, validator.called)
	assert.False(t, repo.importCalled)
}
func validTransferSnapshot(t *testing.T) *entity.UserDataTransferSnapshot {
	t.Helper()
	name, err := playername.NewPlayerName("テスト")
	require.NoError(t, err)
	return &entity.UserDataTransferSnapshot{
		Player:                   entity.UserDataTransferPlayer{Name: name, Level: 1, CreatedAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)},
		Records:                  []entity.UserDataTransferRecord{},
		RecordHistories:          []entity.UserDataTransferRecordHistory{},
		WorldsendRecords:         []entity.UserDataTransferWorldsendRecord{},
		WorldsendRecordHistories: []entity.UserDataTransferWorldsendRecordHistory{},
		MetricHistories:          []entity.UserDataTransferMetricHistory{},
		CourseRecords:            []entity.UserDataTransferCourseRecord{},
		Honors:                   []entity.UserDataTransferHonor{},
		FavoriteSongs:            []entity.UserDataTransferFavoriteSong{},
		LockedSongs:              []entity.UserDataTransferLockedSong{},
		Goals:                    entity.UserDataTransferGoals{Groups: []entity.UserDataTransferGoalGroup{}, Ungrouped: []entity.UserDataTransferGoal{}},
		RecordFilters:            []entity.UserDataTransferRecordFilter{},
	}
}
