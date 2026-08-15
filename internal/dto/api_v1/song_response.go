package api_v1

import (
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
)

// NewV1SongsResponse は通常楽曲エンティティを外部API v1の一覧レスポンスへ変換します。
// HTTPレスポンスと静的スナップショットで同じ変換規則を共有するために使用します。
func NewV1SongsResponse(
	songs []*entity.Song,
	genreNamesByID map[int]string,
	difficultyNamesByID map[int]string,
	calculateMaxOP func(*entity.Song) float64,
) *V1SongsResponse {
	songDTOs := make([]*V1SongDTO, 0, len(songs))
	for _, song := range songs {
		songDTOs = append(songDTOs, NewV1SongDTO(song, genreNamesByID, difficultyNamesByID, calculateMaxOP))
	}
	return &V1SongsResponse{Songs: songDTOs}
}

// NewV1SongDTO は通常楽曲エンティティを譜面情報を含む外部API v1のDTOへ変換します。
func NewV1SongDTO(
	song *entity.Song,
	genreNamesByID map[int]string,
	difficultyNamesByID map[int]string,
	calculateMaxOP func(*entity.Song) float64,
) *V1SongDTO {
	songDTO := ToV1SongDTO(song, genreNamesByID, calculateMaxOP(song))
	for difficultyID, difficultyName := range difficultyNamesByID {
		if difficultyID >= service.DifficultyIDBasic && difficultyID <= service.DifficultyIDUltima {
			songDTO.Charts[difficultyName] = nil
		}
	}
	for _, chart := range song.Charts {
		difficultyName, ok := difficultyNamesByID[chart.DifficultyID]
		if !ok {
			continue
		}
		songDTO.Charts[difficultyName] = ToV1ChartDTO(chart)
	}
	return songDTO
}

// NewV1WorldsendSongsResponse はWORLD'S END楽曲エンティティを外部API v1の一覧レスポンスへ変換します。
func NewV1WorldsendSongsResponse(
	songs []*entity.WorldsendSongWithChart,
	genreNamesByID map[int]string,
) *V1WorldsendSongsResponse {
	songDTOs := make([]*V1WorldsendSongDTO, 0, len(songs))
	for _, song := range songs {
		songDTOs = append(songDTOs, ToV1WorldsendSongDTO(song.Song, song.Chart, genreNamesByID))
	}
	return &V1WorldsendSongsResponse{Songs: songDTOs}
}
