package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCompatQuery struct {
	profile                   *repository.ChunirecProfile
	records                   []*repository.ChunirecRecord
	err                       error
	profileCalls, recordCalls int
	playerID                  int
}

func (s *stubCompatQuery) FindProfileByPlayerID(_ context.Context, id int) (*repository.ChunirecProfile, error) {
	s.profileCalls++
	s.playerID = id
	return s.profile, s.err
}
func (s *stubCompatQuery) ListRecordsByPlayerID(_ context.Context, id int) ([]*repository.ChunirecRecord, error) {
	s.recordCalls++
	s.playerID = id
	return s.records, s.err
}

func TestChunirecUsecase_Access(t *testing.T) {
	tests := []struct {
		name             string
		target           *entity.User
		requester        *entity.User
		accepted         bool
		repoErr, wantErr error
		wantCalls        int
	}{
		{name: "公開ユーザー", target: &entity.User{ID: 1, PlayerID: new(10)}, wantCalls: 1},
		{name: "非公開の本人", target: &entity.User{ID: 1, PlayerID: new(10), IsPrivate: true}, requester: &entity.User{ID: 1}, wantCalls: 1},
		{name: "承認済みフレンド", target: &entity.User{ID: 1, PlayerID: new(10), IsPrivate: true}, requester: &entity.User{ID: 2}, accepted: true, wantCalls: 1},
		{name: "未承認フレンド", target: &entity.User{ID: 1, PlayerID: new(10), IsPrivate: true}, requester: &entity.User{ID: 2}, wantErr: ErrUserPrivate},
		{name: "非公開の匿名アクセス", target: &entity.User{ID: 1, PlayerID: new(10), IsPrivate: true}, wantErr: ErrUserPrivate},
		{name: "ユーザー不存在", repoErr: repository.ErrUserNotFound, wantErr: ErrUserNotFound},
		{name: "nilユーザー", wantErr: ErrUserNotFound},
		{name: "未連携", target: &entity.User{ID: 1}},
		{name: "切断", repoErr: context.Canceled, wantErr: context.Canceled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, records := range []bool{false, true} {
				query := &stubCompatQuery{}
				friends := &stubFriendshipRepo{exists: map[[2]int]bool{{2, 1}: tt.accepted}}
				uc := NewChunirecUsecase(nil, &stubUserRepository{user: tt.target, err: tt.repoErr}, friends, query)
				var err error
				if records {
					_, err = uc.GetRecords(context.Background(), "tester", tt.requester)
				} else {
					_, err = uc.GetProfile(context.Background(), "tester", tt.requester)
				}
				if tt.wantErr != nil {
					assert.ErrorIs(t, err, tt.wantErr)
				} else {
					assert.NoError(t, err)
				}
				assert.Equal(t, tt.wantCalls, query.profileCalls+query.recordCalls)
				if tt.wantCalls > 0 {
					assert.Equal(t, 10, query.playerID)
				}
			}
		})
	}
}

func TestChunirecUsecase_専用Queryの結果と原因エラーを保持する(t *testing.T) {
	constant, err := chartconstant.NewChartConstant(10.7)
	require.NoError(t, err)
	profile := &repository.ChunirecProfile{Name: "プレイヤー"}
	query := &stubCompatQuery{profile: profile, records: []*repository.ChunirecRecord{{Const: constant, Score: 1003215, UpdatedAt: time.Now()}}}
	uc := NewChunirecUsecase(nil, &stubUserRepository{user: &entity.User{PlayerID: new(10)}}, nil, query)
	got, err := uc.GetProfile(context.Background(), "tester", nil)
	require.NoError(t, err)
	assert.Same(t, profile, got)
	assert.Zero(t, query.recordCalls)
	records, err := uc.GetRecords(context.Background(), "tester", nil)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.InDelta(t, 12.02, records[0].Rating, 0.000001)
	assert.Equal(t, 1, query.profileCalls)
	query.err = context.Canceled
	_, err = uc.GetProfile(context.Background(), "tester", nil)
	assert.ErrorIs(t, err, context.Canceled)
	_, err = uc.GetRecords(context.Background(), "tester", nil)
	assert.ErrorIs(t, err, context.Canceled)
}
