package api_internal

import (
	"time"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/dto"
)

// UserProfileWithRecordsDTO はユーザープロファイルとレコードを統合したDTOです。
type UserProfileWithRecordsDTO struct {
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

// UserProfileRatingViewDTO はレーティングビュー用のユーザープロファイルDTOです。
type UserProfileRatingViewDTO struct {
	Username  string                       `json:"username"`
	Player    *dto.PlayerDTO               `json:"player"`
	Records   *UserRatingRecordResponseDTO `json:"records"`
	UpdatedAt *time.Time                   `json:"updated_at"` // プレイヤーデータの最終更新日時
}

// UserDTO はユーザー情報を外部に公開するためのDTOです。
// パスワードハッシュなどの機密情報は含まれません。
type UserDTO struct {
	Username        string     `json:"username"`
	AccountType     string     `json:"account_type"`
	LastScoreUpdate *time.Time `json:"last_score_update"` // プレイヤースコアの最終更新日時
}

// ToUserDTO はエンティティからDTOへ変換します。
// accountTypeNameはUsecase層で解決された値を受け取ります。
func ToUserDTO(user *entity.User, accountTypeName string, lastScoreUpdate *time.Time) *UserDTO {
	if user == nil {
		return nil
	}

	return &UserDTO{
		Username:        user.Username.String(),
		AccountType:     accountTypeName,
		LastScoreUpdate: lastScoreUpdate,
	}
}
