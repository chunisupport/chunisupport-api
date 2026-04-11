package api_internal

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/dto"
)

// UserProfileWithRecordsDTO はユーザープロファイルとレコードを統合したDTOです。
type UserProfileWithRecordsDTO struct {
	UserID    int                        `json:"user_id"`
	Username  string                     `json:"username"`
	Player    *dto.PlayerDTO             `json:"player"`
	Records   *dto.UserRecordResponseDTO `json:"records"`
	UpdatedAt *time.Time                 `json:"updated_at"` // プレイヤーデータの最終更新日時
}

// UserRatingRecordResponseDTO はレーティング関連のレコードDTOです。
type UserRatingRecordResponseDTO struct {
	UpdatedAt     time.Time              `json:"updated_at"`
	Best          []*dto.PlayerRecordDTO `json:"best"`
	BestCandidate []*dto.PlayerRecordDTO `json:"best_candidate"`
	New           []*dto.PlayerRecordDTO `json:"new"`
	NewCandidate  []*dto.PlayerRecordDTO `json:"new_candidate"`
}

// UserRecordViewResponseDTO はレコードビュー向けのレコードDTOです。
type UserRecordViewResponseDTO struct {
	UpdatedAt time.Time                 `json:"updated_at"`
	All       []*dto.PlayerRecordDTO    `json:"all"`
	Worldsend []*dto.WorldsendRecordDTO `json:"worldsend"`
}

// UserProfileRatingViewDTO はレーティングビュー用のユーザープロファイルDTOです。
type UserProfileRatingViewDTO struct {
	Username  string                       `json:"username"`
	Player    *dto.PlayerDTO               `json:"player"`
	Records   *UserRatingRecordResponseDTO `json:"records"`
	UpdatedAt *time.Time                   `json:"updated_at"` // プレイヤーデータの最終更新日時
}

// UserProfileRecordViewDTO はレコードビュー用のユーザープロファイルDTOです。
type UserProfileRecordViewDTO struct {
	Username  string                     `json:"username"`
	Player    *dto.PlayerDTO             `json:"player"`
	Records   *UserRecordViewResponseDTO `json:"records"`
	UpdatedAt *time.Time                 `json:"updated_at"` // プレイヤーデータの最終更新日時
}

// UserProfileDTO はユーザー名とプレイヤー情報のみを含む軽量なプロファイルDTOです。
type UserProfileDTO struct {
	Username string         `json:"username"`
	Player   *dto.PlayerDTO `json:"player"`
}

// UserDTO はユーザー情報を外部に公開するためのDTOです。
// パスワードハッシュなどの機密情報は含まれません。
type UserDTO struct {
	Username        string     `json:"username"`
	AccountType     string     `json:"account_type"`
	IsPrivate       bool       `json:"is_private"`        // 非公開設定
	LastScoreUpdate *time.Time `json:"last_score_update"` // プレイヤースコアの最終更新日時
}

// ToUserDTO はエンティティからDTOへ変換します。
// accountTypeNameはUsecase層で解決された値を受け取ります。
func ToUserDTO(user *entity.User, accountTypeName string, isPrivate bool, lastScoreUpdate *time.Time) *UserDTO {
	if user == nil {
		return nil
	}

	return &UserDTO{
		Username:        user.Username.String(),
		AccountType:     accountTypeName,
		IsPrivate:       isPrivate,
		LastScoreUpdate: lastScoreUpdate,
	}
}
