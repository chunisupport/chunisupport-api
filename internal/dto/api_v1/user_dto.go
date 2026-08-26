package api_v1

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/dto"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

// V1HonorDTO は外部API v1 用の称号情報DTOです。
type V1HonorDTO struct {
	Slot     int    `json:"slot"`
	Name     string `json:"name"`
	TypeName string `json:"type_name"`
	ImageURL string `json:"image_url"`
}

// V1PlayerDTO は外部API v1 用のプレイヤー情報DTOです。
type V1PlayerDTO struct {
	Name              string        `json:"name"`
	Level             int           `json:"level"`
	Rating            *float64      `json:"rating"`
	ClassEmblemID     *int          `json:"class_emblem_id"`
	ClassEmblemBaseID *int          `json:"class_emblem_base_id"`
	LastPlayedAt      *time.Time    `json:"last_played_at"`
	OverpowerValue    *float64      `json:"overpower_value"`
	OverpowerPercent  *float64      `json:"overpower_percent"`
	Honors            []*V1HonorDTO `json:"honors"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

// V1PlayerRecordDTO は外部API v1 用のプレイヤーレコードDTOです。
type V1PlayerRecordDTO struct {
	UpdatedAt        *time.Time                  `json:"updated_at"`
	IsPlayed         bool                        `json:"is_played"`
	IsOPTarget       bool                        `json:"is_op_target"`
	Difficulty       string                      `json:"difficulty"`
	ID               string                      `json:"id"`
	Title            string                      `json:"title"`
	Artist           string                      `json:"artist"`
	Const            chartconstant.ChartConstant `json:"const"`
	IsConstUnknown   bool                        `json:"is_const_unknown"`
	Score            uint32                      `json:"score"`
	JusticeCount     *int                        `json:"justice_count"`
	Rating           float64                     `json:"rating"`
	Overpower        float64                     `json:"overpower"`
	OverpowerPercent float64                     `json:"overpower_percent"`
	Img              string                      `json:"img"`
	ClearLamp        *string                     `json:"clear_lamp"`
	ComboLamp        *string                     `json:"combo_lamp"`
	FullChain        *string                     `json:"full_chain"`
	Slot             *string                     `json:"slot"`
}

// V1WorldsendRecordDTO は外部API v1 用の WORLD'S END レコードDTOです。
type V1WorldsendRecordDTO struct {
	UpdatedAt    *time.Time `json:"updated_at"`
	IsPlayed     bool       `json:"is_played"`
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Artist       string     `json:"artist"`
	LevelStar    *int       `json:"level_star"`
	Attribute    *string    `json:"attribute"`
	Notes        *int       `json:"notes"`
	Score        uint32     `json:"score"`
	JusticeCount *int       `json:"justice_count"`
	Img          string     `json:"img"`
	ClearLamp    *string    `json:"clear_lamp"`
	ComboLamp    *string    `json:"combo_lamp"`
	FullChain    *string    `json:"full_chain"`
}

// V1UserRecordResponseDTO は外部API v1 用のユーザーレコードレスポンスDTOです。
type V1UserRecordResponseDTO struct {
	UpdatedAt     time.Time               `json:"updated_at"`
	Best          []*V1PlayerRecordDTO    `json:"best"`
	BestCandidate []*V1PlayerRecordDTO    `json:"best_candidate"`
	New           []*V1PlayerRecordDTO    `json:"new"`
	NewCandidate  []*V1PlayerRecordDTO    `json:"new_candidate"`
	All           []*V1PlayerRecordDTO    `json:"standard"`
	WorldsEnd     []*V1WorldsendRecordDTO `json:"worldsend"` // WORLD'S END レコード（全件）
	Courses       []*V1CourseRecordDTO    `json:"course"`
}

// V1UserProfileDTO は外部API v1 用のユーザープロファイルDTOです。
type V1UserProfileDTO struct {
	Username  string                   `json:"username"`
	Player    *V1PlayerDTO             `json:"player"`
	Records   *V1UserRecordResponseDTO `json:"records"`
	UpdatedAt *time.Time               `json:"updated_at"`
}

// V1UserRatingMetaDTO は外部API v1 のレーティング更新情報です。
type V1UserRatingMetaDTO struct {
	UpdatedAt *time.Time `json:"updated_at"`
}

// V1UserRatingDTO は外部API v1 のレーティング枠レスポンスです。
type V1UserRatingDTO struct {
	Rating        *float64             `json:"rating"`
	BestAverage   *float64             `json:"best_average"`
	NewAverage    *float64             `json:"new_average"`
	Best          []*V1PlayerRecordDTO `json:"best"`
	BestCandidate []*V1PlayerRecordDTO `json:"best_candidate"`
	New           []*V1PlayerRecordDTO `json:"new"`
	NewCandidate  []*V1PlayerRecordDTO `json:"new_candidate"`
	Meta          *V1UserRatingMetaDTO `json:"meta"`
}

// ToV1PlayerDTO はエンティティから V1PlayerDTO へ変換します。
// Honors フィールドはこの関数では設定されません。呼び出し元で別途設定してください。
func ToV1PlayerDTO(player *entity.Player) *V1PlayerDTO {
	if player == nil {
		return nil
	}

	return &V1PlayerDTO{
		Name:              player.Name.String(),
		Level:             player.Level,
		Rating:            player.CalculatedRating,
		ClassEmblemID:     player.ClassEmblemID,
		ClassEmblemBaseID: player.ClassEmblemBaseID,
		LastPlayedAt:      player.LastPlayedAt,
		OverpowerValue:    player.OverpowerValue,
		OverpowerPercent:  player.OverpowerPercent,
		Honors:            []*V1HonorDTO{},
		CreatedAt:         player.CreatedAt,
		UpdatedAt:         player.UpdatedAt,
	}
}

// ToV1HonorDTO はエンティティから V1HonorDTO へ変換します。
func ToV1HonorDTO(honor *dto.HonorDTO) *V1HonorDTO {
	if honor == nil {
		return nil
	}
	return &V1HonorDTO{
		Slot:     honor.Slot,
		Name:     honor.Name,
		TypeName: honor.TypeName,
		ImageURL: honor.ImageURL,
	}
}

// ToV1PlayerRecordDTO は既存の PlayerRecordDTO から V1PlayerRecordDTO へ変換します。
func ToV1PlayerRecordDTO(record *dto.PlayerRecordDTO) *V1PlayerRecordDTO {
	if record == nil {
		return nil
	}
	return &V1PlayerRecordDTO{
		UpdatedAt:        record.UpdatedAt,
		IsPlayed:         record.IsPlayed,
		IsOPTarget:       record.IsOPTarget,
		Difficulty:       record.Difficulty,
		ID:               record.ID,
		Title:            record.Title,
		Artist:           record.Artist,
		Const:            record.Const,
		IsConstUnknown:   record.IsConstUnknown,
		Score:            record.Score,
		JusticeCount:     record.JusticeCount,
		Rating:           record.Rating,
		Overpower:        record.Overpower,
		OverpowerPercent: record.OverpowerPercent,
		Img:              record.Img,
		ClearLamp:        record.ClearLamp,
		ComboLamp:        record.ComboLamp,
		FullChain:        record.FullChain,
		Slot:             record.Slot,
	}
}

// ToV1WorldsendRecordDTO は既存の WorldsendRecordDTO から V1WorldsendRecordDTO へ変換します。
func ToV1WorldsendRecordDTO(record *dto.WorldsendRecordDTO) *V1WorldsendRecordDTO {
	if record == nil {
		return nil
	}
	return &V1WorldsendRecordDTO{
		UpdatedAt:    record.UpdatedAt,
		IsPlayed:     record.IsPlayed,
		ID:           record.ID,
		Title:        record.Title,
		Artist:       record.Artist,
		LevelStar:    record.LevelStar,
		Attribute:    record.Attribute,
		Notes:        record.Notes,
		Score:        record.Score,
		JusticeCount: record.JusticeCount,
		Img:          record.Img,
		ClearLamp:    record.ClearLamp,
		ComboLamp:    record.ComboLamp,
		FullChain:    record.FullChain,
	}
}

// ToV1UserRecordResponseDTO は既存の UserRecordResponseDTO から V1UserRecordResponseDTO へ変換します。
func ToV1UserRecordResponseDTO(records *dto.UserRecordResponseDTO) *V1UserRecordResponseDTO {
	if records == nil {
		return nil
	}

	convertSlice := func(src []*dto.PlayerRecordDTO) []*V1PlayerRecordDTO {
		result := make([]*V1PlayerRecordDTO, len(src))
		for i, r := range src {
			result[i] = ToV1PlayerRecordDTO(r)
		}
		return result
	}

	convertWorldsendSlice := func(src []*dto.WorldsendRecordDTO) []*V1WorldsendRecordDTO {
		result := make([]*V1WorldsendRecordDTO, len(src))
		for i, r := range src {
			result[i] = ToV1WorldsendRecordDTO(r)
		}
		return result
	}

	convertCourseSlice := func(src []*dto.CourseRecordDTO) []*V1CourseRecordDTO {
		result := make([]*V1CourseRecordDTO, len(src))
		for i, course := range src {
			result[i] = ToV1CourseRecordDTO(course)
		}
		return result
	}

	return &V1UserRecordResponseDTO{
		UpdatedAt:     records.UpdatedAt,
		Best:          convertSlice(records.Best),
		BestCandidate: convertSlice(records.BestCandidate),
		New:           convertSlice(records.New),
		NewCandidate:  convertSlice(records.NewCandidate),
		All:           convertSlice(records.All),
		WorldsEnd:     convertWorldsendSlice(records.WorldsEnd),
		Courses:       convertCourseSlice(records.Courses),
	}
}

// ToV1UserRatingDTO はユースケース出力から外部API v1 のレーティング枠DTOへ変換します。
func ToV1UserRatingDTO(output *usecase.UserProfileRatingViewOutput) *V1UserRatingDTO {
	if output == nil {
		return nil
	}

	convertSlice := func(src []*usecase.PlayerRecordOutput) []*V1PlayerRecordDTO {
		result := make([]*V1PlayerRecordDTO, len(src))
		for i, recordOutput := range src {
			if recordOutput == nil || recordOutput.PlayerRecord == nil {
				continue
			}
			record := dto.ToPlayerRecordDTO(recordOutput.PlayerRecord)
			record.UpdatedAt = recordOutput.UpdatedAt
			record.IsPlayed = recordOutput.IsPlayed
			record.IsOPTarget = recordOutput.IsOPTarget
			record.Difficulty = recordOutput.Difficulty
			record.Const = recordOutput.Const
			record.ClearLamp = recordOutput.ClearLamp
			record.ComboLamp = recordOutput.ComboLamp
			record.FullChain = recordOutput.FullChain
			record.Slot = recordOutput.Slot
			result[i] = ToV1PlayerRecordDTO(record)
		}
		return result
	}

	rating := &V1UserRatingDTO{
		Best:          []*V1PlayerRecordDTO{},
		BestCandidate: []*V1PlayerRecordDTO{},
		New:           []*V1PlayerRecordDTO{},
		NewCandidate:  []*V1PlayerRecordDTO{},
		Meta:          &V1UserRatingMetaDTO{UpdatedAt: output.UpdatedAt},
	}
	if output.Player != nil && output.Player.Player != nil {
		rating.Rating = output.Player.Player.CalculatedRating
		rating.BestAverage = output.Player.Player.BestAverageRating
		rating.NewAverage = output.Player.Player.NewAverageRating
	}
	if output.Records == nil {
		return rating
	}

	rating.Best = convertSlice(output.Records.Best)
	rating.BestCandidate = convertSlice(output.Records.BestCandidate)
	rating.New = convertSlice(output.Records.New)
	rating.NewCandidate = convertSlice(output.Records.NewCandidate)
	rating.Meta.UpdatedAt = &output.Records.UpdatedAt
	return rating
}

// ToV1UserProfileDTO は既存の UserProfileWithRecordsDTO から V1UserProfileDTO へ変換します。
func ToV1UserProfileDTO(profile *api_internal.UserProfileWithRecordsDTO) *V1UserProfileDTO {
	if profile == nil {
		return nil
	}

	var v1Player *V1PlayerDTO
	if profile.Player != nil {
		v1Player = &V1PlayerDTO{
			Name:              profile.Player.Name,
			Level:             profile.Player.Level,
			Rating:            profile.Player.CalculatedRating,
			ClassEmblemID:     profile.Player.ClassEmblemID,
			ClassEmblemBaseID: profile.Player.ClassEmblemBaseID,
			LastPlayedAt:      profile.Player.LastPlayedAt,
			OverpowerValue:    profile.Player.OverpowerValue,
			OverpowerPercent:  profile.Player.OverpowerPercent,
			Honors:            make([]*V1HonorDTO, len(profile.Player.Honors)),
			CreatedAt:         profile.Player.CreatedAt,
			UpdatedAt:         profile.Player.UpdatedAt,
		}
		for i, h := range profile.Player.Honors {
			v1Player.Honors[i] = ToV1HonorDTO(h)
		}
	}

	return &V1UserProfileDTO{
		Username:  profile.Username,
		Player:    v1Player,
		Records:   ToV1UserRecordResponseDTO(profile.Records),
		UpdatedAt: profile.UpdatedAt,
	}
}
