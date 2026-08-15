package reiwa

import (
	"log/slog"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/notes"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
)

// ChunithmRecordDTO は楽曲×譜面の1レコードを表します
type ChunithmRecordDTO struct {
	Title      string  `json:"title"`
	Artist     string  `json:"artist"`
	Img        string  `json:"img"`
	Genre      string  `json:"genre"`
	Const      float64 `json:"const"`
	Level      float64 `json:"level"`
	Diff       string  `json:"diff"`
	Notes      int     `json:"notes"`
	Unknown    int     `json:"unknown"`
	ChunirecID string  `json:"chunirec_id"`
	Idx        string  `json:"idx"`
	BPM        int     `json:"bpm"`
	Release    int64   `json:"release"`
	Version    string  `json:"version"`
}

// ChunithmRecordList はレコードの配列です
type ChunithmRecordList []*ChunithmRecordDTO

// ToChunithmRecordOriginalResponse はWORLD'S END以外の全楽曲をフラットな譜面単位のDTO配列に変換します
func ToChunithmRecordOriginalResponse(songs []*entity.Song, masterCache *masterdata.Cache) ChunithmRecordList {
	records := make(ChunithmRecordList, 0)

	for _, s := range songs {
		if s.IsWorldsend {
			continue
		}

		genreName := resolveGenreName(s.GenreID, masterCache)
		versionName := resolveVersionName(s.ReleasedAt, masterCache)
		bpm := resolveBPM(s.BPM)
		release := resolveRelease(s.ReleasedAt)
		jacket := resolveJacket(s.Jacket)

		for _, c := range s.Charts {
			record := &ChunithmRecordDTO{
				Title:      s.Title,
				Artist:     s.Artist,
				Img:        jacket,
				Genre:      genreName,
				Const:      c.Const.Float64(),
				Level:      calculateLevel(c.Const.Float64()),
				Diff:       difficultyIDToDiff(c.DifficultyID),
				Notes:      resolveNotes(c.Notes),
				Unknown:    boolToInt(c.IsConstUnknown),
				ChunirecID: s.DisplayID,
				Idx:        s.OfficialIdx,
				BPM:        bpm,
				Release:    release,
				Version:    versionName,
			}
			records = append(records, record)
		}
	}

	sortChunithmRecords(records)
	return records
}

// resolveGenreName はGenreIDからジャンル名を解決します。POPS&ANIMEはPOPS & ANIMEに変換します
func resolveGenreName(genreID *int, cache *masterdata.Cache) string {
	if genreID == nil || cache == nil {
		return ""
	}
	name, ok := cache.GenreNamesByID[*genreID]
	if !ok {
		return ""
	}
	if name == "POPS&ANIME" {
		return "POPS & ANIME"
	}
	return name
}

// resolveVersionName はリリース日時点でアクティブなバージョン名から"CHUNITHM "を除去した名前を返します
func resolveVersionName(releasedAt *time.Time, cache *masterdata.Cache) string {
	if releasedAt == nil || cache == nil {
		return ""
	}

	var targetVersion *masterdata.Version
	for _, v := range cache.VersionsByID {
		if !v.ReleasedAt.After(*releasedAt) {
			if targetVersion == nil || v.ReleasedAt.After(targetVersion.ReleasedAt) {
				vCopy := v
				targetVersion = &vCopy
			}
		}
	}

	if targetVersion == nil {
		return ""
	}

	name := targetVersion.Name
	if name == "CHUNITHM" {
		return ""
	}
	return strings.TrimPrefix(name, "CHUNITHM ")
}

// resolveBPM はBPM値を解決します。nilの場合は0を返します
func resolveBPM(bpm *int) int {
	if bpm == nil {
		return 0
	}
	return *bpm
}

// resolveRelease はリリース日のJST0時のUnixタイムスタンプを100で割った値を返します
func resolveRelease(releasedAt *time.Time) int64 {
	if releasedAt == nil {
		return 0
	}

	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	releaseJST := time.Date(
		releasedAt.Year(), releasedAt.Month(), releasedAt.Day(),
		0, 0, 0, 0, jst,
	)
	return releaseJST.Unix() / 100
}

// resolveJacket はジャケット画像識別子を解決します。nilの場合は空文字を返します
func resolveJacket(jacket *string) string {
	if jacket == nil {
		return ""
	}
	return *jacket
}

// resolveNotes はノーツ数を解決します。nilの場合は0を返します
func resolveNotes(n *notes.Notes) int {
	if n == nil {
		return 0
	}
	return int(*n)
}

// calculateLevel は譜面定数から表記レベルを計算します
// .0〜.4は.0に、.5〜.9は.5に丸められます
func calculateLevel(constant float64) float64 {
	intPart := math.Floor(constant)
	fracPart := constant - intPart

	if fracPart >= 0.5 {
		return intPart + 0.5
	}
	return intPart
}

// difficultyIDToDiff は難易度IDを3文字の難易度名に変換します
func difficultyIDToDiff(difficultyID int) string {
	switch difficultyID {
	case 1:
		return "BAS"
	case 2:
		return "ADV"
	case 3:
		return "EXP"
	case 4:
		return "MAS"
	case 5:
		return "ULT"
	default:
		return ""
	}
}

// boolToInt はbool値を0/1に変換します
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// sortChunithmRecords はレコードをidx（数値昇順）→難易度順にソートします
func sortChunithmRecords(records ChunithmRecordList) {
	sort.Slice(records, func(i, j int) bool {
		idxI, errI := strconv.Atoi(records[i].Idx)
		if errI != nil {
			slog.Warn("failed to parse official idx as integer", "idx", records[i].Idx, "error", errI)
		}
		idxJ, errJ := strconv.Atoi(records[j].Idx)
		if errJ != nil {
			slog.Warn("failed to parse official idx as integer", "idx", records[j].Idx, "error", errJ)
		}
		if idxI != idxJ {
			return idxI < idxJ
		}
		return diffSortOrder(records[i].Diff) < diffSortOrder(records[j].Diff)
	})
}

// diffSortOrder は難易度名をソート用の数値に変換します
func diffSortOrder(diff string) int {
	switch diff {
	case "BAS":
		return 1
	case "ADV":
		return 2
	case "EXP":
		return 3
	case "MAS":
		return 4
	case "ULT":
		return 5
	default:
		return 99
	}
}
