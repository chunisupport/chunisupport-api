package usecase

import (
	"context"
	"strings"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/profilecomment/profilecommenttest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubUserProfileRepository struct {
	found              *entity.UserProfile
	err                error
	saved              *entity.UserProfile
	deletedID, foundID int
}

func (s *stubUserProfileRepository) FindByUserID(_ context.Context, _ repository.Executor, id int) (*entity.UserProfile, error) {
	s.foundID = id
	return s.found, s.err
}
func (s *stubUserProfileRepository) Save(_ context.Context, _ repository.Executor, p *entity.UserProfile) error {
	s.saved = p
	return s.err
}
func (s *stubUserProfileRepository) DeleteByUserID(_ context.Context, _ repository.Executor, id int) error {
	s.deletedID = id
	return s.err
}

type stubCommentModerator struct {
	flagged bool
	err     error
	calls   []string
}

func (s *stubCommentModerator) IsFlagged(_ context.Context, comment string) (bool, error) {
	s.calls = append(s.calls, comment)
	return s.flagged, s.err
}

func TestUserProfileCommentUsecase_Update_内容判定(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		flagged       bool
		moderationErr error
		wantErr       error
		wantCalls     []string
	}{
		{name: "問題なしなら正規化後の本文を判定して保存", input: " よろしく ", wantCalls: []string{"よろしく"}},
		{name: "不適切と判定されたら保存しない", input: "abc", flagged: true, wantErr: ErrCommentInappropriate, wantCalls: []string{"abc"}},
		{name: "判定できなければ原因を保持して保存しない", input: "abc", moderationErr: context.Canceled, wantErr: ErrCommentModerationUnavailable, wantCalls: []string{"abc"}},
		{name: "削除は判定しない", input: " "},
		{name: "検証エラーは判定しない", input: "a\nb", wantErr: ErrCommentInvalidChar},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			repo := &stubUserProfileRepository{}
			moderator := &stubCommentModerator{flagged: tt.flagged, err: tt.moderationErr}
			uc := NewUserProfileCommentUsecase(nil, repo, nil, nil, moderator)
			// When
			_, err := uc.Update(context.Background(), 7, tt.input)
			// Then
			assert.Equal(t, tt.wantCalls, moderator.calls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				if tt.moderationErr != nil {
					assert.ErrorIs(t, err, tt.moderationErr)
				}
				assert.Nil(t, repo.saved)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUserProfileCommentUsecase_Update(t *testing.T) {
	tests := []struct {
		name, input      string
		repoErr, wantErr error
		want             string
		deleted          bool
	}{
		{name: "空白を除去して保存", input: "　よろしく 🙂 ", want: "よろしく 🙂"},
		{name: "空なら削除", input: "", deleted: true},
		{name: "空白だけなら削除", input: "　 \t\n", deleted: true},
		{name: "長すぎる", input: strings.Repeat("あ", 101), wantErr: ErrCommentTooLong},
		{name: "制御文字", input: "a\nb", wantErr: ErrCommentInvalidChar},
		{name: "保存失敗の原因保持", input: "abc", repoErr: context.Canceled, wantErr: context.Canceled},
		{name: "削除失敗の原因保持", input: "", repoErr: context.Canceled, wantErr: context.Canceled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			repo := &stubUserProfileRepository{err: tt.repoErr}
			uc := NewUserProfileCommentUsecase(nil, repo, nil, nil, &stubCommentModerator{})
			// When
			got, err := uc.Update(context.Background(), 7, tt.input)
			// Then
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				if tt.repoErr == nil {
					assert.Nil(t, repo.saved)
					assert.Zero(t, repo.deletedID)
				} else {
					assert.ErrorIs(t, err, ErrInternalError)
				}
				return
			}
			require.NoError(t, err)
			if tt.deleted {
				assert.Nil(t, got)
				assert.Equal(t, 7, repo.deletedID)
				assert.Nil(t, repo.saved)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tt.want, *got)
			require.NotNil(t, repo.saved)
			assert.Equal(t, 7, repo.saved.UserID())
			assert.Equal(t, tt.want, repo.saved.Comment().String())
		})
	}
}

func TestUserProfileCommentUsecase_Get(t *testing.T) {
	saved, err := entity.NewUserProfile(7, profilecommenttest.New(t, "よろしく"))
	require.NoError(t, err)
	public := &entity.User{ID: 7}
	private := &entity.User{ID: 7, IsPrivate: true}
	tests := []struct {
		name                      string
		target, requester         *entity.User
		userErr, repoErr, wantErr error
		saved                     bool
	}{
		{name: "公開ユーザーは匿名で閲覧", target: public, saved: true},
		{name: "未設定", target: public},
		{name: "プレイヤー未連携でも本人は閲覧", target: private, requester: &entity.User{ID: 7}, saved: true},
		{name: "相互フレンドは閲覧", target: private, requester: &entity.User{ID: 2}, saved: true},
		{name: "非公開は匿名で不可", target: private, wantErr: ErrUserPrivate},
		{name: "非公開は他人に不可", target: private, requester: &entity.User{ID: 3}, wantErr: ErrUserPrivate},
		{name: "存在しない", wantErr: ErrUserNotFound},
		{name: "検索で未発見", userErr: repository.ErrUserNotFound, wantErr: ErrUserNotFound},
		{name: "検索キャンセル", userErr: context.Canceled, wantErr: context.Canceled},
		{name: "取得キャンセル", target: public, repoErr: context.Canceled, wantErr: context.Canceled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			repo := &stubUserProfileRepository{err: tt.repoErr}
			if tt.saved {
				repo.found = saved
			}
			uc := NewUserProfileCommentUsecase(nil, repo, &stubUserRepository{user: tt.target, err: tt.userErr}, &stubFriendshipRepo{exists: map[[2]int]bool{{2, 7}: true}}, nil)
			// When
			got, err := uc.Get(context.Background(), "tester", tt.requester)
			// Then
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				if tt.repoErr == nil {
					assert.Zero(t, repo.foundID)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, 7, repo.foundID)
			if tt.saved {
				require.NotNil(t, got)
				assert.Equal(t, "よろしく", *got)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}
