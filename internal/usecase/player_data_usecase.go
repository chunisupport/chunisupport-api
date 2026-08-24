package usecase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	playerdataresult "github.com/chunisupport/chunisupport-api/internal/usecase/playerdataresult"
)

// PlayerDataPayload はCHUNITHMプレイヤーデータインポートの入力構造です。
type PlayerDataPayload struct {
	AppVersion  string
	Name        string
	Level       int
	Rating      *float64
	LastPlayed  string
	Overpower   PlayerDataOverpowerPayload
	ClassEmblem PlayerDataClassPayload
	Team        PlayerDataTeamPayload
	Honors      map[string]PlayerDataHonorPayload
	Scores      PlayerDataScorePayload
	UpdatedAt   string
}

// PlayerDataOverpowerPayload はオーバーパワー情報です。
type PlayerDataOverpowerPayload struct {
	Value      *float64
	Percentage *float64
}

// PlayerDataClassPayload はクラスエンブレム情報です。
type PlayerDataClassPayload struct {
	MedalClass string
	BaseClass  string
}

// PlayerDataTeamPayload はチーム情報です。
type PlayerDataTeamPayload struct {
	Name  string
	Color string
}

// PlayerDataHonorPayload は称号スロットの情報です。
type PlayerDataHonorPayload struct {
	Title string
	Class string
	Img   *string
}

// PlayerDataScorePayload はスコア配列を保持します。
type PlayerDataScorePayload struct {
	Standard  []PlayerDataScoreEntry
	Worldsend []PlayerDataScoreEntry
	Course    []PlayerDataCourseEntry
}

// PlayerDataCourseEntry は1件のコースレコード入力です。
type PlayerDataCourseEntry struct {
	Score   int
	IsClear bool
	ComboLv int
	Idx     string
}

// PlayerDataScoreEntry は1件のスコア情報です。
type PlayerDataScoreEntry struct {
	Diff      string
	Idx       string
	Score     int
	ClearLamp *string
	ComboLv   *int
	FullChain *int
	Slot      *string
	Order     *int
}

// PlayerDataUsecase はCHUNITHMプレイヤーデータの登録ユースケースを表します。
type PlayerDataUsecase interface {
	Register(ctx context.Context, user *entity.User, payload *PlayerDataPayload, bodyHash string) (*playerdataresult.Result, error)
	// GetLatestUpdate はユーザーに紐づくプレイヤーの最新データ登録結果を返します。
	GetLatestUpdate(ctx context.Context, user *entity.User) (json.RawMessage, error)
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
	Name                     string
	Level                    int
	OfficialRating           float64
	OfficialOverpower        float64
	OfficialOverpowerPercent float64
	LastPlayedAt             *time.Time
	OverpowerValue           *float64
	OverpowerPercent         *float64
	ClassEmblemID            *int
	ClassBaseID              *int
}
