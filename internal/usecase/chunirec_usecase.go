package usecase

import (
	"context"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
)

type ChunirecRecordOutput struct {
	*repository.ChunirecRecord
	Rating float64
}

type ChunirecUsecase interface {
	GetProfile(context.Context, string, *entity.User) (*repository.ChunirecProfile, error)
	GetRecords(context.Context, string, *entity.User) ([]*ChunirecRecordOutput, error)
}

type chunirecUsecase struct {
	db             repository.Executor
	userRepo       repository.UserRepository
	friendshipRepo repository.FriendshipRepository
	query          repository.ChunirecQueryService
}

func NewChunirecUsecase(db repository.Executor, userRepo repository.UserRepository, friendshipRepo repository.FriendshipRepository, query repository.ChunirecQueryService) ChunirecUsecase {
	return &chunirecUsecase{db: db, userRepo: userRepo, friendshipRepo: friendshipRepo, query: query}
}

func (s *chunirecUsecase) GetProfile(ctx context.Context, username string, requester *entity.User) (*repository.ChunirecProfile, error) {
	user, err := getAccessibleUser(ctx, s.db, s.userRepo, s.friendshipRepo, username, requester)
	if err != nil {
		return nil, err
	}
	if !user.HasLinkedPlayer() {
		return nil, nil
	}
	return s.query.FindProfileByPlayerID(ctx, *user.PlayerID)
}

func (s *chunirecUsecase) GetRecords(ctx context.Context, username string, requester *entity.User) ([]*ChunirecRecordOutput, error) {
	user, err := getAccessibleUser(ctx, s.db, s.userRepo, s.friendshipRepo, username, requester)
	if err != nil {
		return nil, err
	}
	output := make([]*ChunirecRecordOutput, 0)
	if !user.HasLinkedPlayer() {
		return output, nil
	}
	records, err := s.query.ListRecordsByPlayerID(ctx, *user.PlayerID)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		output = append(output, &ChunirecRecordOutput{ChunirecRecord: record, Rating: service.CalcSingleRating(record.Score, record.Const.Float64())})
	}
	return output, nil
}
