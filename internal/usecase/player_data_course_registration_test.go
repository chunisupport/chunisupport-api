package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/usecase/playerdataresult"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type courseRegistrationRepository struct {
	repository.CourseRepository
	t      *testing.T
	tx     repository.Executor
	before map[int]repository.CourseRecordState
	saved  []repository.CourseRecordForUpsert
}

func (r *courseRegistrationRepository) FindByOfficialIdxList(_ context.Context, _ repository.Executor, indexes []string) (map[string]*entity.Course, error) {
	courses := map[string]*entity.Course{}
	for _, idx := range indexes {
		if idx == "course-idx" {
			courses[idx] = &entity.Course{ID: 501, OfficialIdx: "course-idx", CourseClass: &entity.CourseClass{Name: "extra"}}
		}
	}
	return courses, nil
}
func (r *courseRegistrationRepository) FindRecordStatesByCourseIDs(_ context.Context, exec repository.Executor, playerID int, ids []int) (map[int]repository.CourseRecordState, error) {
	assert.Same(r.t, r.tx, exec)
	assert.Equal(r.t, 77, playerID)
	assert.Equal(r.t, []int{501}, ids)
	return r.before, nil
}
func (r *courseRegistrationRepository) UpsertRecords(_ context.Context, exec repository.Executor, rows []repository.CourseRecordForUpsert) error {
	assert.Same(r.t, r.tx, exec)
	r.saved = rows
	return nil
}

func TestRegister_コースの保存と差分件数(t *testing.T) {
	now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	entry := PlayerDataCourseEntry{Idx: "course-idx", Score: 3_000_000, IsClear: true, ComboLv: 2}
	saved := []repository.CourseRecordForUpsert{{
		PlayerID: 77, CourseID: 501,
		State: repository.CourseRecordState{Score: 3_000_000, IsClear: true, ComboLampID: 2, UpdatedAt: now},
	}}
	newChange := playerdataresult.RecordChange{
		RecordType: "course", ChangeType: "new", Idx: "course-idx", CourseClass: "extra",
		After: playerdataresult.RecordState{Score: 3_000_000, IsClear: new(true), ComboLamp: new("full combo")},
	}
	updatedChange := playerdataresult.RecordChange{
		RecordType: "course", ChangeType: "updated", Idx: "course-idx", CourseClass: "extra",
		Before: &playerdataresult.RecordState{Score: 3_000_000, IsClear: new(false), ComboLamp: new("full combo")},
		After:  playerdataresult.RecordState{Score: 3_000_000, IsClear: new(true), ComboLamp: new("full combo")},
	}
	for _, tt := range []struct {
		name              string
		entries           []PlayerDataCourseEntry
		before            map[int]repository.CourseRecordState
		wantChanges       []playerdataresult.RecordChange
		wantSaved         []repository.CourseRecordForUpsert
		upserted, skipped int
		missingRepo       bool
	}{
		{
			name: "新規", entries: []PlayerDataCourseEntry{entry}, upserted: 1,
			wantSaved: saved, wantChanges: []playerdataresult.RecordChange{newChange},
		},
		{
			name: "同値なら更新差分なし", entries: []PlayerDataCourseEntry{entry}, upserted: 1,
			before:    map[int]repository.CourseRecordState{501: {Score: 3_000_000, IsClear: true, ComboLampID: 2, UpdatedAt: now.Add(-time.Hour)}},
			wantSaved: saved,
		},
		{
			name: "クリアだけの変更も差分", entries: []PlayerDataCourseEntry{entry}, upserted: 1,
			before:    map[int]repository.CourseRecordState{501: {Score: 3_000_000, IsClear: false, ComboLampID: 2}},
			wantSaved: saved, wantChanges: []playerdataresult.RecordChange{updatedChange},
		},
		{
			name: "スコアだけの変更も差分", entries: []PlayerDataCourseEntry{entry}, upserted: 1,
			before:    map[int]repository.CourseRecordState{501: {Score: 2_900_000, IsClear: true, ComboLampID: 2}},
			wantSaved: saved, wantChanges: []playerdataresult.RecordChange{{
				RecordType: "course", ChangeType: "updated", Idx: "course-idx", CourseClass: "extra",
				Before: &playerdataresult.RecordState{Score: 2_900_000, IsClear: new(true), ComboLamp: new("full combo")},
				After:  playerdataresult.RecordState{Score: 3_000_000, IsClear: new(true), ComboLamp: new("full combo")},
			}},
		},
		{
			name: "コンボランプだけの変更も差分", entries: []PlayerDataCourseEntry{entry}, upserted: 1,
			before:    map[int]repository.CourseRecordState{501: {Score: 3_000_000, IsClear: true, ComboLampID: 1}},
			wantSaved: saved, wantChanges: []playerdataresult.RecordChange{{
				RecordType: "course", ChangeType: "updated", Idx: "course-idx", CourseClass: "extra",
				Before: &playerdataresult.RecordState{Score: 3_000_000, IsClear: new(true)},
				After:  playerdataresult.RecordState{Score: 3_000_000, IsClear: new(true), ComboLamp: new("full combo")},
			}},
		},
		{
			name: "未知のidxをスキップ", entries: []PlayerDataCourseEntry{{Idx: "unknown", Score: 3_000_000, ComboLv: 2}},
			upserted: 1, skipped: 1,
		},
		{
			name: "重複は最後の記録を保存", entries: []PlayerDataCourseEntry{{Idx: "course-idx", Score: 2_900_000, ComboLv: 1}, entry},
			upserted: 2, wantSaved: saved, wantChanges: []playerdataresult.RecordChange{newChange},
		},
		{
			name: "リポジトリ未初期化は保存しない", entries: []PlayerDataCourseEntry{entry}, missingRepo: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newRegistrationFixture(t)
			courseRepo := &courseRegistrationRepository{t: t, tx: fixture.tx, before: tt.before}
			if !tt.missingRepo {
				fixture.uc.courseRepo = courseRepo
			}
			fixture.payload.Scores.Course = tt.entries

			result, err := fixture.uc.Register(context.Background(), fixture.user, fixture.payload, "course-registration-hash")

			if tt.missingRepo {
				require.Error(t, err)
				assert.Zero(t, fixture.data.saveCalls)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.upserted, result.Counts.CourseRecordsUpserted)
			assert.Equal(t, tt.skipped, result.Counts.CourseRecordsSkipped)
			assert.Equal(t, len(tt.wantChanges), result.Counts.CourseRecordsActuallyChanged)
			assert.Equal(t, tt.wantSaved, courseRepo.saved)
			assert.Equal(t, tt.wantChanges, append([]playerdataresult.RecordChange(nil), result.Changes...))
			if tt.skipped > 0 {
				assert.Equal(t, []playerdataresult.SkippedRecord{{RecordType: "course", Reason: "failed to resolve course", Details: "idx=unknown"}}, result.SkippedRecords)
			} else {
				assert.Empty(t, result.SkippedRecords)
			}
		})
	}
}
