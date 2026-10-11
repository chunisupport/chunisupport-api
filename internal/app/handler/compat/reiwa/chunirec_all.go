package reiwa

import (
	"cmp"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
)

type ChunirecAllMetaDTO struct {
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	Genre   string  `json:"genre"`
	Artist  string  `json:"artist"`
	Release string  `json:"release"`
	BPM     int     `json:"bpm"`
	Img     *string `json:"img,omitempty"`
	Idx     *string `json:"idx,omitempty"`
}

type ChunirecAllChartDTO struct {
	Level          float64 `json:"level"`
	Const          float64 `json:"const"`
	MaxCombo       int     `json:"maxcombo"`
	IsConstUnknown int     `json:"is_const_unknown"`
}

type ChunirecAllItemDTO struct {
	Meta ChunirecAllMetaDTO              `json:"meta"`
	Data map[string]*ChunirecAllChartDTO `json:"data"`
}

type ChunirecAllResponse []*ChunirecAllItemDTO

func ToChunirecAllResponse(songs []*entity.Song, worldsendSongs []*entity.WorldsendSongWithChart, cache *masterdata.Cache) ChunirecAllResponse {
	normal := slices.Clone(songs)
	slices.SortFunc(normal, func(a, b *entity.Song) int {
		ai, ae := strconv.Atoi(a.OfficialIdx)
		bi, be := strconv.Atoi(b.OfficialIdx)
		if ae == nil && be != nil {
			return -1
		}
		if ae != nil && be == nil {
			return 1
		}
		if ae == nil && be == nil {
			if order := cmp.Compare(ai, bi); order != 0 {
				return order
			}
		}
		return cmp.Compare(a.DisplayID, b.DisplayID)
	})
	we := slices.Clone(worldsendSongs)
	slices.SortFunc(we, func(a, b *entity.WorldsendSongWithChart) int { return cmp.Compare(a.Song.DisplayID, b.Song.DisplayID) })
	response := make(ChunirecAllResponse, 0, len(normal)+len(we))
	for _, song := range normal {
		if song.IsDeleted {
			continue
		}
		meta := chunirecAllMeta(song, cache)
		jacket := resolveJacket(song.Jacket)
		jacket = strings.TrimSuffix(jacket, path.Ext(jacket))
		meta.Img = &jacket
		meta.Idx = &song.OfficialIdx
		item := &ChunirecAllItemDTO{Meta: meta, Data: make(map[string]*ChunirecAllChartDTO, len(song.Charts))}
		for _, chart := range song.Charts {
			diff := difficultyIDToDiff(chart.DifficultyID)
			if diff == "" {
				continue
			}
			item.Data[diff] = &ChunirecAllChartDTO{Level: calculateLevel(chart.Const.Float64()), Const: chart.Const.Float64(), MaxCombo: resolveNotes(chart.Notes), IsConstUnknown: boolToInt(chart.IsConstUnknown)}
		}
		response = append(response, item)
	}
	for _, entry := range we {
		if entry.Song.IsDeleted {
			continue
		}
		meta := chunirecAllMeta(entry.Song, cache)
		meta.Genre = worldsendGenre
		if entry.Chart.Attribute != nil && *entry.Chart.Attribute != "" {
			meta.Title += "【" + *entry.Chart.Attribute + "】"
		}
		response = append(response, &ChunirecAllItemDTO{Meta: meta, Data: map[string]*ChunirecAllChartDTO{"WE": {MaxCombo: resolveNotes(entry.Chart.Notes)}}})
	}
	return response
}

func chunirecAllMeta(song *entity.Song, cache *masterdata.Cache) ChunirecAllMetaDTO {
	meta := ChunirecAllMetaDTO{ID: song.DisplayID, Title: song.Title, Artist: song.Artist, Genre: resolveGenreName(song.GenreID, cache), BPM: resolveBPM(song.BPM)}
	if song.ReleasedAt != nil {
		meta.Release = song.ReleasedAt.Format("2006-01-02")
	}
	return meta
}
