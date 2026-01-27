package api_internal

import (
	"time"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/dto"
	"github.com/Qman110101/chunisupport-api/internal/infra/masterdata"
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
func ToUserDTO(user *entity.User, masterCache *masterdata.Cache, lastScoreUpdate *time.Time) *UserDTO {
	if user == nil {
		return nil
	}

	return &UserDTO{
		Username:        user.Username.String(),
		AccountType:     getAccountTypeName(user.AccountTypeID, masterCache),
		LastScoreUpdate: lastScoreUpdate,
	}
}

// getAccountTypeName はアカウントタイプIDから名前を取得します。
func getAccountTypeName(accountTypeID int, masterCache *masterdata.Cache) string {
	if masterCache == nil {
		return "UNKNOWN"
	}
	for _, item := range masterCache.AccountTypes {
		if item.ID == accountTypeID {
			return item.Name
		}
	}
	return "UNKNOWN"
}
