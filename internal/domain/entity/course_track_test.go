package entity

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displaylevel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCourseReplaceTracks_空または3件を番号順に保持する(t *testing.T) {
	course := validCourseForTracks()
	tracks := []CourseTrack{
		validFixedTrack(3, "0000000000000003", "MASTER"),
		validFixedTrack(1, "0000000000000001", "BASIC"),
		validFixedTrack(2, "0000000000000002", "EXPERT"),
	}

	err := course.ReplaceTracks(tracks)
	require.NoError(t, err)
	require.Len(t, course.Tracks, 3)
	assert.Equal(t, []int{1, 2, 3}, []int{course.Tracks[0].Number, course.Tracks[1].Number, course.Tracks[2].Number})

	err = course.ReplaceTracks(nil)
	require.NoError(t, err)
	assert.Empty(t, course.Tracks)
}

func TestCourseReplaceTracks_番号が不足または重複する場合は変更しない(t *testing.T) {
	course := validCourseForTracks()
	course.Tracks = []CourseTrack{validFixedTrack(1, "0000000000000001", "BASIC"), validFixedTrack(2, "0000000000000002", "EXPERT"), validFixedTrack(3, "0000000000000003", "MASTER")}
	before := course.Tracks

	err := course.ReplaceTracks([]CourseTrack{
		validFixedTrack(1, "0000000000000001", "BASIC"),
		validFixedTrack(1, "0000000000000002", "EXPERT"),
		validFixedTrack(3, "0000000000000003", "MASTER"),
	})

	assert.Error(t, err)
	assert.Equal(t, before, course.Tracks)
}

func TestCourseTrackValidate_選択方式ごとの条件を検証する(t *testing.T) {
	minLevel, err := displaylevel.Parse("13")
	require.NoError(t, err)
	maxLevel, err := displaylevel.Parse("13+")
	require.NoError(t, err)
	validFilter := &CourseTrackFilter{MinLevel: &minLevel, MaxLevel: &maxLevel, Genres: []string{"POPS"}}

	tests := []struct {
		name  string
		track CourseTrack
		want  bool
	}{
		{name: "固定は譜面参照を持つ", track: validFixedTrack(1, "0000000000000001", "BASIC"), want: true},
		{name: "固定は譜面参照を複数持てない", track: CourseTrack{Number: 1, SelectionType: CourseTrackFixed, Charts: []CourseChartRef{{SongID: "0000000000000001", Difficulty: "BASIC"}, {SongID: "0000000000000002", Difficulty: "MASTER"}}}, want: false},
		{name: "ランダムフィルターは空でない条件を持つ", track: CourseTrack{Number: 1, SelectionType: CourseTrackRandomFilter, Filter: validFilter}, want: true},
		{name: "ランダムプールは譜面を持つ", track: CourseTrack{Number: 1, SelectionType: CourseTrackRandomPool, Charts: []CourseChartRef{{SongID: "0000000000000001", Difficulty: "MASTER"}}}, want: true},
		{name: "固定にフィルターは指定できない", track: CourseTrack{Number: 1, SelectionType: CourseTrackFixed, Filter: validFilter, Charts: []CourseChartRef{{SongID: "0000000000000001", Difficulty: "BASIC"}}}, want: false},
		{name: "フィルターの下限が上限を超える場合は無効", track: CourseTrack{Number: 1, SelectionType: CourseTrackRandomFilter, Filter: &CourseTrackFilter{MinLevel: &maxLevel, MaxLevel: &minLevel}}, want: false},
		{name: "プールに譜面がない場合は無効", track: CourseTrack{Number: 1, SelectionType: CourseTrackRandomPool}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.want {
				assert.NoError(t, tt.track.Validate())
			} else {
				assert.Error(t, tt.track.Validate())
			}
		})
	}
}

func TestCourseTrackValidate_ランダムプールの譜面参照は重複を拒否する(t *testing.T) {
	track := CourseTrack{
		Number:        1,
		SelectionType: CourseTrackRandomPool,
		Charts: []CourseChartRef{
			{SongID: "0000000000000001", Difficulty: "MASTER"},
			{SongID: "0000000000000001", Difficulty: "MASTER"},
		},
	}

	assert.Error(t, track.Validate())
}

func TestCourseTrackValidate_譜面参照を検証する(t *testing.T) {
	tests := []struct {
		name  string
		chart CourseChartRef
		want  bool
	}{
		{name: "通常難易度の大文字表記", chart: CourseChartRef{SongID: "0000000000000001", Difficulty: "MASTER"}, want: true},
		{name: "ULTIMAも通常譜面集合として許可", chart: CourseChartRef{SongID: "0000000000000001", Difficulty: "ULTIMA"}, want: true},
		{name: "WORLDSENDも固定譜面として許可", chart: CourseChartRef{SongID: "0000000000000001", Difficulty: CourseTrackDifficultyWorldsend}, want: true},
		{name: "小文字は無効", chart: CourseChartRef{SongID: "0000000000000001", Difficulty: "master"}, want: false},
		{name: "短縮WORLD'S END表記は無効", chart: CourseChartRef{SongID: "0000000000000001", Difficulty: "WE"}, want: false},
		{name: "公開ID形式でない楽曲IDは無効", chart: CourseChartRef{SongID: "song-1", Difficulty: "MASTER"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			track := CourseTrack{Number: 1, SelectionType: CourseTrackFixed, Charts: []CourseChartRef{tt.chart}}
			if tt.want {
				assert.NoError(t, track.Validate())
			} else {
				assert.Error(t, track.Validate())
			}
		})
	}
}

func TestCourseTrackValidate_固定とプールは通常譜面とWORLDSENDを混在できる(t *testing.T) {
	track := CourseTrack{
		Number:        1,
		SelectionType: CourseTrackRandomPool,
		Charts: []CourseChartRef{
			{SongID: "0000000000000001", Difficulty: "MASTER"},
			{SongID: "0000000000000002", Difficulty: CourseTrackDifficultyWorldsend},
		},
	}

	assert.NoError(t, track.Validate())
}

func TestCourseTrackValidate_フィルターのジャンルは空と重複を拒否する(t *testing.T) {
	tests := []struct {
		name   string
		genres []string
		want   bool
	}{
		{name: "一意なジャンル", genres: []string{"POPS", "VARIETY"}, want: true},
		{name: "空文字のジャンル", genres: []string{"POPS", ""}, want: false},
		{name: "重複ジャンル", genres: []string{"POPS", "POPS"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			track := CourseTrack{Number: 1, SelectionType: CourseTrackRandomFilter, Filter: &CourseTrackFilter{Genres: tt.genres}}
			if tt.want {
				assert.NoError(t, track.Validate())
			} else {
				assert.Error(t, track.Validate())
			}
		})
	}
}

func TestCourseValidate_Tracksも検証する(t *testing.T) {
	course := validCourseForTracks()
	course.Tracks = []CourseTrack{{Number: 1, SelectionType: CourseTrackRandomPool}}

	assert.Error(t, course.Validate())
}

func validCourseForTracks() *Course {
	return &Course{
		DisplayID:     "0123456789abcdef",
		OfficialIdx:   "50020",
		Name:          "コース",
		CourseClassID: 1,
	}
}

func validFixedTrack(number int, songID, difficulty string) CourseTrack {
	return CourseTrack{Number: number, SelectionType: CourseTrackFixed, Charts: []CourseChartRef{{SongID: songID, Difficulty: difficulty}}}
}
