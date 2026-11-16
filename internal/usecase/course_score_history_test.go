package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestBuildCourseHistories(t *testing.T) {
	now := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	current := repository.CourseRecordState{Score: 2_900_000, IsClear: true, ComboLampID: 2, UpdatedAt: now}
	tests := []struct {
		name string
		// Given: 保存前の状態と、今回の登録内容
		before map[int]repository.CourseRecordState
		after  repository.CourseRecordState
		// Then: 退避される更新前の状態
		want []repository.PlayerCourseRecordHistory
	}{
		{
			name:   "初めて登録するコースは退避しない",
			before: map[int]repository.CourseRecordState{},
			after:  repository.CourseRecordState{Score: 2_900_000, IsClear: true, ComboLampID: 2},
		},
		{
			name:   "成績が同じ再登録は退避しない",
			before: map[int]repository.CourseRecordState{501: current},
			after:  repository.CourseRecordState{Score: 2_900_000, IsClear: true, ComboLampID: 2, UpdatedAt: now.Add(time.Hour)},
		},
		{
			name:   "スコアが変わると更新前の状態を退避する",
			before: map[int]repository.CourseRecordState{501: current},
			after:  repository.CourseRecordState{Score: 2_950_000, IsClear: true, ComboLampID: 2},
			want:   []repository.PlayerCourseRecordHistory{{PlayerID: 77, CourseID: 501, State: current}},
		},
		{
			name:   "クリア状態だけが変わっても退避する",
			before: map[int]repository.CourseRecordState{501: current},
			after:  repository.CourseRecordState{Score: 2_900_000, IsClear: false, ComboLampID: 2},
			want:   []repository.PlayerCourseRecordHistory{{PlayerID: 77, CourseID: 501, State: current}},
		},
		{
			name:   "コンボランプだけが変わっても退避する",
			before: map[int]repository.CourseRecordState{501: current},
			after:  repository.CourseRecordState{Score: 2_900_000, IsClear: true, ComboLampID: 3},
			want:   []repository.PlayerCourseRecordHistory{{PlayerID: 77, CourseID: 501, State: current}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			rows, courseIDs := buildCourseHistories(77, tt.before, []repository.CourseRecordForUpsert{{PlayerID: 77, CourseID: 501, State: tt.after}})

			// Then
			if tt.want == nil {
				assert.Empty(t, rows)
				assert.Empty(t, courseIDs)
				return
			}
			assert.Equal(t, tt.want, rows)
			assert.Equal(t, []int{501}, courseIDs)
		})
	}
}

func TestRegister_コース成績が変化したときだけ履歴を保存して上限整理する(t *testing.T) {
	now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	entry := PlayerDataCourseEntry{Idx: "course-idx", Score: 3_000_000, IsClear: true, ComboLv: 2}
	previous := repository.CourseRecordState{Score: 2_900_000, IsClear: false, ComboLampID: 1, UpdatedAt: now.Add(-24 * time.Hour)}
	tests := []struct {
		name        string
		before      map[int]repository.CourseRecordState
		wantRows    []repository.PlayerCourseRecordHistory
		wantPruneID []int
	}{
		{
			name:        "変化したコースは更新前の成績を履歴へ追加する",
			before:      map[int]repository.CourseRecordState{501: previous},
			wantRows:    []repository.PlayerCourseRecordHistory{{PlayerID: 77, CourseID: 501, State: previous}},
			wantPruneID: []int{501},
		},
		{
			name:        "変化なしの再登録では履歴を増やさない",
			before:      map[int]repository.CourseRecordState{501: {Score: 3_000_000, IsClear: true, ComboLampID: 2, UpdatedAt: now.Add(-24 * time.Hour)}},
			wantRows:    []repository.PlayerCourseRecordHistory{},
			wantPruneID: []int{},
		},
		{
			name:        "新規登録では履歴を増やさない",
			wantRows:    []repository.PlayerCourseRecordHistory{},
			wantPruneID: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			fixture := newRegistrationFixture(t)
			fixture.uc.courseRepo = &courseRegistrationRepository{t: t, tx: fixture.tx, before: tt.before}
			histories := &stubScoreHistoryRepositoryForApplyScoresTest{}
			fixture.uc.scoreHistoryRepo = histories
			fixture.payload.Scores.Course = []PlayerDataCourseEntry{entry}

			// When
			_, err := fixture.uc.Register(context.Background(), fixture.user, fixture.payload, "course-history-hash")

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.wantRows, histories.courseRows)
			assert.Equal(t, 77, histories.pruneCoursePlayer)
			assert.Equal(t, tt.wantPruneID, histories.pruneCourseIDs)
		})
	}
}

// courseHistoryReadRepository はコース履歴取得で使う表示用IDからの解決だけを差し替えます。
type courseHistoryReadRepository struct {
	repository.CourseRepository
	course *entity.Course
	calls  int
}

func (r *courseHistoryReadRepository) FindByDisplayID(_ context.Context, _ repository.Executor, _ string, includeDeleted bool) (*entity.Course, error) {
	r.calls++
	if r.course == nil || (r.course.IsDeleted && !includeDeleted) {
		return nil, repository.ErrCourseNotFound
	}
	return r.course, nil
}

func newCourseHistoryReadMasterProvider() repository.PlayerDataMasterProvider {
	return &scoreHistoryReadMasterProvider{masters: &masterdata.PlayerDataMasters{
		ComboLampNamesByID: map[int]string{1: "NONE", 2: "FULL COMBO", 3: "ALL JUSTICE"},
	}}
}

func TestScoreHistoryUsecase_GetCourse_現行ベストと過去ベストをコース用の形で返す(t *testing.T) {
	// Given
	ctx := context.Background()
	exec := &MockExecutor{}
	userRepo := new(MockUserRepository)
	historyRepo := new(mockScoreHistoryRepository)
	courseRepo := &courseHistoryReadRepository{course: &entity.Course{ID: 501}}
	setScoreHistoryReadUserExpectation(userRepo, exec, newScoreHistoryReadUser(t))
	now := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	historyRepo.On("FindCourseTimeline", ctx, 10, 501).Return([]entity.CourseScoreHistoryEntry{
		{Score: 3_010_000, IsClear: true, ComboLampID: 3, UpdatedAt: now},
		{Score: 2_900_000, IsClear: false, ComboLampID: 1, UpdatedAt: now.Add(-time.Hour)},
	}, nil).Once()
	uc := NewScoreHistoryUsecase(exec, userRepo, nil, nil, courseRepo, historyRepo, &stubFriendshipRepo{}, newCourseHistoryReadMasterProvider())

	// When
	got, err := uc.GetCourse(ctx, "testuser", nil, "0123456789abcdef")

	// Then: ランプなしは null として返す。
	require.NoError(t, err)
	assert.Equal(t, []CourseScoreHistoryEntry{
		{Score: 3_010_000, IsClear: true, ComboLamp: new("ALL JUSTICE"), UpdatedAt: now},
		{Score: 2_900_000, IsClear: false, ComboLamp: nil, UpdatedAt: now.Add(-time.Hour)},
	}, got)
	userRepo.AssertExpectations(t)
	historyRepo.AssertExpectations(t)
}

func TestScoreHistoryUsecase_GetCourse_履歴を返さない分岐ごとのエラー(t *testing.T) {
	tests := []struct {
		name string
		// Given
		private      bool
		playerLinked bool
		course       *entity.Course
		timeline     []entity.CourseScoreHistoryEntry
		// Then
		wantErr         error
		wantCourseCalls int
	}{
		{name: "非公開ユーザーは第三者に履歴を返さない", private: true, playerLinked: true, course: &entity.Course{ID: 501}, wantErr: ErrUserPrivate},
		{name: "プレイヤー未連携は履歴なし", course: &entity.Course{ID: 501}, wantErr: ErrScoreHistoryNotFound},
		{name: "存在しないコースはコースなし", playerLinked: true, wantErr: repository.ErrCourseNotFound, wantCourseCalls: 1},
		{name: "削除済みコースはコースなし", playerLinked: true, course: &entity.Course{ID: 501, IsDeleted: true}, wantErr: repository.ErrCourseNotFound, wantCourseCalls: 1},
		{name: "未プレイのコースは履歴なし", playerLinked: true, course: &entity.Course{ID: 501}, timeline: []entity.CourseScoreHistoryEntry{}, wantErr: ErrScoreHistoryNotFound, wantCourseCalls: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: 期待を設定していない履歴の読み出しが呼ばれるとテストは失敗する。
			ctx := context.Background()
			exec := &MockExecutor{}
			userRepo := new(MockUserRepository)
			historyRepo := new(mockScoreHistoryRepository)
			courseRepo := &courseHistoryReadRepository{course: tt.course}
			user := testUser(t, 1, "testuser")
			if tt.playerLinked {
				user = newScoreHistoryReadUser(t)
			}
			user.ChangePrivacy(tt.private)
			userRepo.On("FindByUsername", mock.Anything, exec, "testuser").Return(user, nil).Once()
			if tt.timeline != nil {
				historyRepo.On("FindCourseTimeline", ctx, 10, 501).Return(tt.timeline, nil).Once()
			}
			uc := NewScoreHistoryUsecase(exec, userRepo, nil, nil, courseRepo, historyRepo, &stubFriendshipRepo{}, newCourseHistoryReadMasterProvider())

			// When
			got, err := uc.GetCourse(ctx, "testuser", nil, "0123456789abcdef")

			// Then
			assert.Nil(t, got)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantCourseCalls, courseRepo.calls)
			historyRepo.AssertExpectations(t)
		})
	}
}
