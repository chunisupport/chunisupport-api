package entity

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displaylevel"
)

// CourseTrackSelectionType はコース内の譜面選択方式を表します。
type CourseTrackSelectionType string

const (
	CourseTrackFixed        CourseTrackSelectionType = "FIXED"
	CourseTrackRandomFilter CourseTrackSelectionType = "RANDOM_FILTER"
	CourseTrackRandomPool   CourseTrackSelectionType = "RANDOM_POOL"

	// CourseTrackDifficultyWorldsend はコースに登録できるWORLD'S END譜面の難易度名です。
	CourseTrackDifficultyWorldsend = "WORLDSEND"
)

// CourseTrackFilter はランダム選曲に使う表示レベル・ジャンル条件です。
type CourseTrackFilter struct {
	MinLevel *displaylevel.Level
	MaxLevel *displaylevel.Level
	Genres   []string
}

// CourseChartRef はコースで選択する公開譜面の識別子です。
type CourseChartRef struct {
	SongID     string
	Difficulty string
}

// CourseTrack はコースの1トラックを表します。
type CourseTrack struct {
	Number        int
	SelectionType CourseTrackSelectionType
	Filter        *CourseTrackFilter
	Charts        []CourseChartRef
}

// CourseChartCandidate はランダム選曲の結果を表示するための候補です。
// 集約の永続状態には含めず、選曲処理の入出力でだけ使用します。
type CourseChartCandidate struct {
	CourseChartRef
	Title          string
	Const          float64
	IsConstUnknown bool
	Jacket         *string
	Attribute      *string
	LevelStar      *int
	Notes          *int
}

// Validate はトラックの選択方式と参照条件が整合していることを検証します。
func (t CourseTrack) Validate() error {
	if t.Number < 1 || t.Number > 3 {
		return errors.New("トラック番号は1から3の範囲である必要があります")
	}

	switch t.SelectionType {
	case CourseTrackFixed:
		if t.Filter != nil {
			return errors.New("固定トラックにフィルターは指定できません")
		}
		if len(t.Charts) != 1 {
			return errors.New("固定トラックには譜面参照が1件必要です")
		}
	case CourseTrackRandomFilter:
		if t.Filter == nil {
			return errors.New("ランダムフィルタートラックにはフィルターが必要です")
		}
		if err := t.Filter.Validate(); err != nil {
			return fmt.Errorf("ランダムフィルターが不正です: %w", err)
		}
		if len(t.Charts) != 0 {
			return errors.New("ランダムフィルタートラックに譜面参照は指定できません")
		}
	case CourseTrackRandomPool:
		if t.Filter != nil {
			return errors.New("ランダムプールトラックにフィルターは指定できません")
		}
		if len(t.Charts) == 0 {
			return errors.New("ランダムプールトラックには譜面参照が1件以上必要です")
		}
	default:
		return fmt.Errorf("トラック選択方式が不正です: %q", t.SelectionType)
	}

	if t.SelectionType == CourseTrackFixed || t.SelectionType == CourseTrackRandomPool {
		return validateCourseChartRefs(t.Charts)
	}
	return nil
}

// Validate はランダム選曲条件が空でなく、範囲も正しいことを検証します。
func (f CourseTrackFilter) Validate() error {
	if f.MinLevel == nil && f.MaxLevel == nil && len(f.Genres) == 0 {
		return errors.New("フィルターは少なくとも1つの条件が必要です")
	}
	if f.MinLevel != nil && !f.MinLevel.IsValid() {
		return errors.New("フィルターの最小レベルが不正です")
	}
	if f.MaxLevel != nil && !f.MaxLevel.IsValid() {
		return errors.New("フィルターの最大レベルが不正です")
	}
	if f.MinLevel != nil && f.MaxLevel != nil && f.MinLevel.LowerTenths() > f.MaxLevel.LowerTenths() {
		return errors.New("フィルターの最小レベルは最大レベル以下である必要があります")
	}

	genres := make(map[string]struct{}, len(f.Genres))
	for _, genre := range f.Genres {
		trimmed := strings.TrimSpace(genre)
		if trimmed == "" {
			return errors.New("フィルターのジャンルは空にできません")
		}
		if _, exists := genres[trimmed]; exists {
			return fmt.Errorf("フィルターのジャンルが重複しています: %q", genre)
		}
		genres[trimmed] = struct{}{}
	}
	return nil
}

func validateCourseChartRefs(charts []CourseChartRef) error {
	refs := make(map[CourseChartRef]struct{}, len(charts))
	for _, chart := range charts {
		if _, err := displayid.NewDisplayID(chart.SongID); err != nil {
			return fmt.Errorf("譜面参照の楽曲IDが不正です: %w", err)
		}
		if !isRegularCourseDifficulty(chart.Difficulty) {
			return fmt.Errorf("譜面参照の難易度が不正です: %q", chart.Difficulty)
		}
		if _, exists := refs[chart]; exists {
			return fmt.Errorf("譜面参照が重複しています: %s/%s", chart.SongID, chart.Difficulty)
		}
		refs[chart] = struct{}{}
	}
	return nil
}

func isRegularCourseDifficulty(value string) bool {
	switch value {
	case "BASIC", "ADVANCED", "EXPERT", "MASTER", "ULTIMA", CourseTrackDifficultyWorldsend:
		return true
	default:
		return false
	}
}

func validateCourseTracks(tracks []CourseTrack) error {
	if len(tracks) != 0 && len(tracks) != 3 {
		return errors.New("トラックは0件または3件である必要があります")
	}

	seenNumbers := make(map[int]struct{}, len(tracks))
	for _, track := range tracks {
		if err := track.Validate(); err != nil {
			return err
		}
		if _, exists := seenNumbers[track.Number]; exists {
			return fmt.Errorf("トラック番号が重複しています: %d", track.Number)
		}
		seenNumbers[track.Number] = struct{}{}
	}
	return nil
}

func cloneCourseTracks(tracks []CourseTrack) []CourseTrack {
	if len(tracks) == 0 {
		return nil
	}
	cloned := make([]CourseTrack, len(tracks))
	for i, track := range tracks {
		cloned[i] = track
		cloned[i].Charts = append([]CourseChartRef(nil), track.Charts...)
		if track.Filter != nil {
			filter := *track.Filter
			filter.Genres = append([]string(nil), track.Filter.Genres...)
			cloned[i].Filter = &filter
		}
	}
	return cloned
}

func sortCourseTracks(tracks []CourseTrack) {
	sort.Slice(tracks, func(i, j int) bool {
		return tracks[i].Number < tracks[j].Number
	})
}
