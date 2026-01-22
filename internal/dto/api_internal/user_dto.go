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

// UserDTO はユーザー情報を外部に公開するためのDTOです。
// パスワードハッシュなどの機密情報は含まれません。
type UserDTO struct {
	Username    string         `json:"username"`
	AccountType string         `json:"account_type"`
	Player      *dto.PlayerDTO `json:"player"` // 関連するプレイヤー情報
}

// ToUserDTO はエンティティからDTOへ変換します。
func ToUserDTO(user *entity.User, masterCache *masterdata.Cache) *UserDTO {
	if user == nil {
		return nil
	}

	dto := &UserDTO{
		Username:    user.Username.String(),
		AccountType: getAccountTypeName(user.AccountTypeID, masterCache),
	}

	// 今後、プレイヤー情報を含める場合はここで設定
	// 現在のユーザーエンティティにはプレイヤー情報は含まれていない

	return dto
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
