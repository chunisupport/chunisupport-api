package usecase

import (
	"context"
	"fmt"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

const unresolvedReferenceResponseLimit = 100

type UserDataTransferUsecase interface {
	Export(ctx context.Context, userID int) (*UserDataTransferExportOutput, error)
	Validate(ctx context.Context, userID int, encoded []byte) (*UserDataTransferValidationOutput, error)
	Import(ctx context.Context, userID int, encoded []byte) (*UserDataTransferImportOutput, error)
}

type UserDataTransferExportOutput struct {
	File []byte
}

type UserDataTransferCounts struct {
	Records                  int
	RecordHistories          int
	WorldsendRecords         int
	WorldsendRecordHistories int
	MetricHistories          int
	CourseRecords            int
	Honors                   int
	FavoriteSongs            int
	LockedSongs              int
	GoalGroups               int
	Goals                    int
	RecordFilters            int
}

type UserDataTransferValidationOutput struct {
	Importable               bool
	PlayerName               string
	Counts                   UserDataTransferCounts
	Blockers                 []string
	UnresolvedReferences     []string
	UnresolvedReferenceCount int
}

type UserDataTransferImportOutput struct {
	PlayerID int
	Counts   UserDataTransferCounts
}

type userDataTransferUsecase struct {
	codec         UserDataTransferCodec
	repo          repository.UserDataTransferRepository
	goalValidator UserDataTransferGoalValidator
}

func NewUserDataTransferUsecase(codec UserDataTransferCodec, repo repository.UserDataTransferRepository, goalValidators ...UserDataTransferGoalValidator) UserDataTransferUsecase {
	var goalValidator UserDataTransferGoalValidator
	if len(goalValidators) > 0 {
		goalValidator = goalValidators[0]
	}
	return &userDataTransferUsecase{codec: codec, repo: repo, goalValidator: goalValidator}
}

func (u *userDataTransferUsecase) Export(ctx context.Context, userID int) (*UserDataTransferExportOutput, error) {
	if u.codec == nil || u.repo == nil || userID <= 0 {
		return nil, ErrInternalError
	}
	snapshot, err := u.repo.ExportSnapshot(ctx, userID)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, ErrDataTransferPlayerNotFound
	}
	if err := snapshot.Validate(); err != nil {
		return nil, fmt.Errorf("%w: exported snapshot is invalid: %w", ErrDataTransferInvalidData, err)
	}
	encoded, err := u.codec.Encode(snapshot)
	if err != nil {
		return nil, err
	}
	return &UserDataTransferExportOutput{File: encoded}, nil
}

func (u *userDataTransferUsecase) Validate(ctx context.Context, userID int, encoded []byte) (*UserDataTransferValidationOutput, error) {
	snapshot, err := u.decode(encoded)
	if err != nil {
		return nil, err
	}
	unresolved, err := u.repo.FindUnresolvedReferences(ctx, snapshot)
	if err != nil {
		return nil, err
	}
	if len(unresolved) == 0 && u.goalValidator != nil {
		if err := u.goalValidator.ValidateTransferredGoals(ctx, snapshot.Goals); err != nil {
			return nil, err
		}
	}
	empty, err := u.repo.IsDestinationEmpty(ctx, userID)
	if err != nil {
		return nil, err
	}
	blockers := make([]string, 0, 2)
	if !empty {
		blockers = append(blockers, "destination_not_empty")
	}
	if len(unresolved) > 0 {
		blockers = append(blockers, "unresolved_references")
	}
	limited := unresolved
	if len(limited) > unresolvedReferenceResponseLimit {
		limited = limited[:unresolvedReferenceResponseLimit]
	}
	return &UserDataTransferValidationOutput{
		Importable:               len(blockers) == 0,
		PlayerName:               snapshot.Player.Name.String(),
		Counts:                   countUserDataTransfer(snapshot),
		Blockers:                 blockers,
		UnresolvedReferences:     append([]string(nil), limited...),
		UnresolvedReferenceCount: len(unresolved),
	}, nil
}

func (u *userDataTransferUsecase) Import(ctx context.Context, userID int, encoded []byte) (*UserDataTransferImportOutput, error) {
	snapshot, err := u.decode(encoded)
	if err != nil {
		return nil, err
	}
	unresolved, err := u.repo.FindUnresolvedReferences(ctx, snapshot)
	if err != nil {
		return nil, err
	}
	if len(unresolved) > 0 {
		return nil, ErrDataTransferUnresolvedReference
	}
	if u.goalValidator != nil {
		if err := u.goalValidator.ValidateTransferredGoals(ctx, snapshot.Goals); err != nil {
			return nil, err
		}
	}
	playerID, err := u.repo.ImportSnapshot(ctx, userID, snapshot)
	if err != nil {
		return nil, err
	}
	return &UserDataTransferImportOutput{PlayerID: playerID, Counts: countUserDataTransfer(snapshot)}, nil
}

func (u *userDataTransferUsecase) decode(encoded []byte) (*entity.UserDataTransferSnapshot, error) {
	if u.codec == nil || u.repo == nil {
		return nil, ErrInternalError
	}
	return u.codec.Decode(encoded)
}

func countUserDataTransfer(snapshot *entity.UserDataTransferSnapshot) UserDataTransferCounts {
	goals := len(snapshot.Goals.Ungrouped)
	for _, group := range snapshot.Goals.Groups {
		goals += len(group.Goals)
	}
	return UserDataTransferCounts{
		Records: len(snapshot.Records), RecordHistories: len(snapshot.RecordHistories),
		WorldsendRecords: len(snapshot.WorldsendRecords), WorldsendRecordHistories: len(snapshot.WorldsendRecordHistories),
		MetricHistories: len(snapshot.MetricHistories), CourseRecords: len(snapshot.CourseRecords), Honors: len(snapshot.Honors),
		FavoriteSongs: len(snapshot.FavoriteSongs), LockedSongs: len(snapshot.LockedSongs), GoalGroups: len(snapshot.Goals.Groups),
		Goals: goals, RecordFilters: len(snapshot.RecordFilters),
	}
}
