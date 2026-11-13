package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoalUsecase_変更は対象ユーザーをロックしたトランザクションで行う(t *testing.T) {
	for _, operation := range []string{"update", "delete", "reorder"} {
		t.Run(operation, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), goalContextKey{}, "request")
			tx := &sqlx.Tx{}
			repo := &stubGoalRepo{goal: &entity.Goal{ID: 42, UserID: 7, SortOrder: 1, AchievementTypeID: 2, AchievementParams: []byte(`{"score":1000000,"count":1}`), Attributes: []byte(`{}`)}}
			tm := &trackingTM{passedFloor: tx}
			u := NewGoalUsecase(nil, tm, repo, &stubGoalMasterProvider{}, &stubGoalGroupRepo{})
			var err error
			switch operation {
			case "update":
				_, err = u.Update(ctx, 7, 42, &GoalInput{Title: "変更後", AchievementType: "score_count", AchievementParams: []byte(`{"score":1000000,"count":1}`), Attributes: []byte(`{}`)})
			case "delete":
				err = u.Delete(ctx, 7, 42)
			case "reorder":
				err = u.Reorder(ctx, 7, nil, []uint32{42})
			}
			require.NoError(t, err)
			require.NotEmpty(t, repo.calls)
			assert.Equal(t, goalRepoCall{operation: "lock", ctx: ctx, exec: tx, userID: 7}, repo.calls[0])
			for _, call := range repo.calls {
				assert.Same(t, ctx, call.ctx)
				assert.Same(t, tx, call.exec)
				if call.operation != "arrange" {
					assert.Equal(t, 7, call.userID)
				}
				if call.operation == "find" || call.operation == "save" || call.operation == "delete" {
					assert.Equal(t, uint32(42), call.id)
				}
			}
			switch operation {
			case "update":
				assert.Equal(t, "変更後", repo.goal.Title)
			case "delete":
				assert.Nil(t, repo.goal)
			case "reorder":
				assert.Equal(t, []uint32{42}, repo.savedOrder)
			}
		})
	}
}

func TestGoalUsecase_他人の目標を削除や並べ替えの対象にできない(t *testing.T) {
	for _, operation := range []string{"delete", "reorder"} {
		t.Run(operation, func(t *testing.T) {
			own := &entity.Goal{ID: 42, UserID: 7, SortOrder: 1}
			other := &entity.Goal{ID: 99, UserID: 8, SortOrder: 1}
			repo := &stubGoalRepo{goals: []*entity.Goal{own, other}}
			uc := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{}, &stubGoalGroupRepo{})
			var err error
			if operation == "delete" {
				err = uc.Delete(context.Background(), 7, 99)
				assert.ErrorIs(t, err, ErrGoalNotFound)
			} else {
				err = uc.Reorder(context.Background(), 7, nil, []uint32{99})
				assert.ErrorIs(t, err, ErrInvalidGoalOrder)
			}
			assert.Equal(t, []*entity.Goal{{ID: 42, UserID: 7, SortOrder: 1}, {ID: 99, UserID: 8, SortOrder: 1}}, repo.goals)
			assert.Nil(t, repo.savedOrder)
		})
	}
}

type goalContextKey struct{}

func TestGoalUsecase_ロックできなければ変更しない(t *testing.T) {
	cause := errors.New("ロック失敗")
	for _, operation := range []string{"update", "delete", "reorder"} {
		t.Run(operation, func(t *testing.T) {
			repo := &stubGoalRepo{lockErr: cause}
			u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{}, &stubGoalGroupRepo{})
			var err error
			switch operation {
			case "update":
				_, err = u.Update(context.Background(), 7, 42, &GoalInput{Title: "変更後", AchievementType: "score_count", AchievementParams: []byte(`{"score":1000000,"count":1}`), Attributes: []byte(`{}`)})
			case "delete":
				err = u.Delete(context.Background(), 7, 42)
			case "reorder":
				err = u.Reorder(context.Background(), 7, nil, []uint32{42})
			}
			assert.ErrorIs(t, err, cause)
			require.Len(t, repo.calls, 1)
			assert.Equal(t, "lock", repo.calls[0].operation)
		})
	}
}

func TestGoalUsecase_Updateは他人の目標を変更できない(t *testing.T) {
	original := &entity.Goal{ID: 42, UserID: 8, Title: "他人の目標"}
	repo := &stubGoalRepo{goal: original}
	u := NewGoalUsecase(nil, &stubTM{}, repo, &stubGoalMasterProvider{}, &stubGoalGroupRepo{})
	_, err := u.Update(context.Background(), 7, 42, &GoalInput{Title: "変更後", AchievementType: "score_count", AchievementParams: []byte(`{"score":1000000,"count":1}`), Attributes: []byte(`{}`)})
	assert.ErrorIs(t, err, ErrGoalNotFound)
	assert.Equal(t, &entity.Goal{ID: 42, UserID: 8, Title: "他人の目標"}, original)
	for _, call := range repo.calls {
		assert.NotEqual(t, "save", call.operation)
	}
}
