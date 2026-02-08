package chunirec

import (
	"fmt"
	"math"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
)

// MusicShowAllResponse は全楽曲情報のレスポンスを表します
type MusicShowAllResponse []*MusicItemDTO

// MusicItemDTO は個々の楽曲情報を表します
type MusicItemDTO struct {
	Meta MusicMetaDTO `json:"meta"`
	Data MusicDataDTO `json:"data"`
}

// MusicMetaDTO は楽曲のメタデータを表します
type MusicMetaDTO struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Genre   *string  `json:"genre"`
	Artist  string   `json:"artist"`
	Release *string  `json:"release"`
	BPM     *float64 `json:"bpm"`
}

// MusicDataDTO は楽曲の譜面データを表します
type MusicDataDTO struct {
	BAS *ChartDataDTO `json:"BAS,omitempty"`
	ADV *ChartDataDTO `json:"ADV,omitempty"`
	EXP *ChartDataDTO `json:"EXP,omitempty"`
	MAS *ChartDataDTO `json:"MAS,omitempty"`
	ULT *ChartDataDTO `json:"ULT,omitempty"`
}

// ChartDataDTO は個々の譜面情報を表します
type ChartDataDTO struct {
	Level          float64 `json:"level"`
	Const          float64 `json:"const"`
	MaxCombo       *int    `json:"maxcombo"`
	IsConstUnknown bool    `json:"is_const_unknown"`
}

// ToMusicShowAllResponse はドメインエンティティのリストをDTOに変換します
func ToMusicShowAllResponse(songs []*entity.Song, masters *domainmasterdata.SongMasters) MusicShowAllResponse {
	response := make(MusicShowAllResponse, 0, len(songs))
	var genres map[int]string
	if masters != nil {
		genres = masters.GenreNamesByID
	}

	for _, s := range songs {
		item := toMusicItemDTO(s, genres)
		response = append(response, item)
	}

	return response
}

// ToMusicShowResponse は単一のドメインエンティティをDTOに変換します（配列ではなくオブジェクトを返す）
func ToMusicShowResponse(song *entity.Song, masters *domainmasterdata.SongMasters) *MusicItemDTO {
	var genres map[int]string
	if masters != nil {
		genres = masters.GenreNamesByID
	}

	return toMusicItemDTO(song, genres)
}

// toMusicItemDTO は1曲のSongWithChartsをMusicItemDTOに変換する内部ヘルパー関数です
func toMusicItemDTO(s *entity.Song, genres map[int]string) *MusicItemDTO {
	item := &MusicItemDTO{
		Meta: MusicMetaDTO{
			ID:      s.DisplayID,
			Title:   s.Title,
			Artist:  s.Artist,
			Release: nil,
			BPM:     nil,
		},
		Data: MusicDataDTO{},
	}

	// Nullable fields handling
	if s.GenreID != nil {
		if genreName, ok := genres[*s.GenreID]; ok {
			item.Meta.Genre = &genreName
		}
	}
	if s.ReleasedAt != nil {
		dateStr := s.ReleasedAt.Format("2006-01-02")
		item.Meta.Release = &dateStr
	}
	if s.BPM != nil {
		bpmVal := float64(*s.BPM)
		item.Meta.BPM = &bpmVal
	}

	// Charts handling
	for _, c := range s.Charts {
		chartDTO := &ChartDataDTO{
			Const:          float64(c.Const),
			IsConstUnknown: c.IsConstUnknown,
			Level:          calculateLevel(float64(c.Const)),
		}

		if c.Notes != nil {
			maxCombo := int(*c.Notes)
			chartDTO.MaxCombo = &maxCombo
		}

		switch c.DifficultyID {
		case 1: // Basic
			item.Data.BAS = chartDTO
		case 2: // Advanced
			item.Data.ADV = chartDTO
		case 3: // Expert
			item.Data.EXP = chartDTO
		case 4: // Master
			item.Data.MAS = chartDTO
		case 5: // Ultima
			item.Data.ULT = chartDTO
		}
	}

	return item
}

// calculateLevel は定数から表記レベルを計算します
// .0 ~ .4 -> .0
// .5 ~ .9 -> .5
func calculateLevel(constant float64) float64 {
	intPart := math.Floor(constant)
	fracPart := constant - intPart

	if fracPart >= 0.5 {
		return intPart + 0.5
	}
	return intPart
}

// ChunirecUserDTO はchunirec互換のユーザープロフィールを表します
type ChunirecUserDTO struct {
	UserID          int     `json:"user_id"`
	PlayerName      string  `json:"player_name"`
	Title           *string `json:"title"`
	TitleRarity     *string `json:"title_rarity"`
	Level           int     `json:"level"`
	Rating          *string `json:"rating"`
	RatingMax       *string `json:"rating_max"`
	ClassEmblem     *string `json:"classemblem"`
	ClassEmblemBase *string `json:"classemblem_base"`
	IsJoinedTeam    *bool   `json:"is_joined_team"` // 常にnil（ChuniSupportでは保持しない）
	UpdatedAt       string  `json:"updated_at"`
}

// ToChunirecUserDTO は内部APIのUserProfileWithRecordsDTOをchunirec互換形式に変換します
func ToChunirecUserDTO(profile *api_internal.UserProfileWithRecordsDTO, userID int, masterCache *masterdata.Cache) *ChunirecUserDTO {
	if profile == nil || profile.Player == nil {
		return nil
	}

	dto := &ChunirecUserDTO{
		UserID:       userID,
		PlayerName:   profile.Player.Name,
		Level:        profile.Player.Level,
		IsJoinedTeam: nil, // ChuniSupportでは保持しないデータ
		UpdatedAt:    profile.Player.UpdatedAt.Format("2006-01-02T15:04:05-07:00"),
	}

	// Rating を文字列化（nullならnullのまま）
	if profile.Player.Rating != nil {
		ratingStr := formatRating(*profile.Player.Rating)
		dto.Rating = &ratingStr
		dto.RatingMax = &ratingStr // rating_max は rating と同じ
	}

	// ClassEmblem の名前を取得
	if profile.Player.ClassEmblemID != nil && masterCache != nil {
		emblemName := masterCache.GetClassEmblemNameByID(*profile.Player.ClassEmblemID)
		if emblemName != "" {
			dto.ClassEmblem = &emblemName
		}
	}

	// ClassEmblemBase の名前を取得
	if profile.Player.ClassEmblemBaseID != nil && masterCache != nil {
		emblemBaseName := masterCache.GetClassEmblemBaseNameByID(*profile.Player.ClassEmblemBaseID)
		if emblemBaseName != "" {
			dto.ClassEmblemBase = &emblemBaseName
		}
	}

	// 1番目の称号（スロット1）を取得
	if len(profile.Player.Honors) > 0 {
		for _, honor := range profile.Player.Honors {
			if honor.Slot == 1 {
				dto.Title = &honor.Name
				// "platina" のみ "platinum" に変換
				rarity := honor.TypeName
				if rarity == "platina" {
					rarity = "platinum"
				}
				dto.TitleRarity = &rarity
				break
			}
		}
	}

	return dto
}

// formatRating はレーティングを小数点以下2桁の文字列にフォーマットします
func formatRating(rating float64) string {
	return formatFloat(rating, 2)
}

// formatFloat は浮動小数点数を指定した小数点以下の桁数で文字列にフォーマットします
func formatFloat(value float64, precision int) string {
	format := "%." + string(rune('0'+precision)) + "f"
	return fmt.Sprintf(format, value)
}
