package usecase

import (
	"context"
	"time"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/dto/api_internal"
)

// PlayerDataPayload はCHUNITHMプレイヤーデータインポートの入力構造です。
type PlayerDataPayload struct {
	AppVersion  string                            `json:"app_ver"`
	Name        string                            `json:"name"`
	Level       int                               `json:"level"`
	Rating      *float64                          `json:"rating"`
	LastPlayed  string                            `json:"last_played"`
	Overpower   PlayerDataOverpowerPayload        `json:"overpower"`
	ClassEmblem PlayerDataClassPayload            `json:"class_emblem"`
	Team        PlayerDataTeamPayload             `json:"team"`
	Honors      map[string]PlayerDataHonorPayload `json:"honors"`
	Scores      PlayerDataScorePayload            `json:"scores"`
	UpdatedAt   string                            `json:"updated_at"`
}

// PlayerDataOverpowerPayload はオーバーパワー情報です。
type PlayerDataOverpowerPayload struct {
	Value      float64 `json:"value"`
	Percentage float64 `json:"percentage"`
}

// PlayerDataClassPayload はクラスエンブレム情報です。
type PlayerDataClassPayload struct {
	MedalClass string `json:"medal_class"`
	BaseClass  string `json:"base_class"`
}

// PlayerDataTeamPayload はチーム情報です。
type PlayerDataTeamPayload struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// PlayerDataHonorPayload は称号スロットの情報です。
type PlayerDataHonorPayload struct {
	Title string  `json:"title"`
	Class string  `json:"class"`
	Img   *string `json:"img_url"`
}

// PlayerDataScorePayload はスコア配列を保持します。
type PlayerDataScorePayload struct {
	Full      []PlayerDataScoreEntry `json:"full"`
	Worldsend []PlayerDataScoreEntry `json:"worldsend"`
}

// PlayerDataScoreEntry は1件のスコア情報です。
type PlayerDataScoreEntry struct {
	Diff      string  `json:"diff"`
	Idx       string  `json:"idx"`
	Score     int     `json:"score"`
	ClearLamp *string `json:"clear_lamp"`
	ComboLv   *int    `json:"cmb_lv"`
	FullChain *int    `json:"fch_lv"`
	Slot      *string `json:"slot"`
	Order     *int    `json:"order"`
}

// PlayerDataUsecase はCHUNITHMプレイヤーデータの登録ユースケースを表します。
type PlayerDataUsecase interface {
	Register(ctx context.Context, user *entity.User, payload *PlayerDataPayload, bodyHash string) (*api_internal.PlayerDataResult, error)
	// Delete はユーザーに紐づくプレイヤーと関連データを削除し、連携を解除します。
	Delete(ctx context.Context, user *entity.User) error
}

// PlayerDataValidationError は入力値検証に失敗した場合のエラーです。
type PlayerDataValidationError struct {
	Field   string
	Message string
}

func (e *PlayerDataValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return e.Field + ": " + e.Message
}

// PlayerDataNotFoundError はマスターデータなどが見つからない場合に発生します。
type PlayerDataNotFoundError struct {
	Resource string
	Key      string
}

func (e *PlayerDataNotFoundError) Error() string {
	return "resource not found: " + e.Resource + "(" + e.Key + ")"
}

// PlayerDataConflictError は矛盾した入力などで処理できない場合に返されます。
type PlayerDataConflictError struct {
	Reason string
}

func (e *PlayerDataConflictError) Error() string {
	return e.Reason
}

// PlayerDataSummaryInput はプレイヤー情報の更新値です。
type PlayerDataSummaryInput struct {
	Name             string
	Level            int
	OfficialRating   *float64
	LastPlayedAt     *time.Time
	OverpowerValue   *float64
	OverpowerPercent *float64
	ClassEmblemID    *int
	ClassBaseID      *int
}
