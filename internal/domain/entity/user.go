package entity

import (
	"errors"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/constants"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
)

var ErrInvalidAccountType = errors.New("invalid account type")

// User はユーザーのエンティティを表します。
type User struct {
	ID            int
	Username      username.UserName
	FirebaseUID   *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	PlayerID      *int
	AccountTypeID int
	// OriginalAccountTypeID は永続化から復元した時点の権限IDです。
	// Save時の競合検知に使い、権限変更の巻き戻しを防ぎます。
	OriginalAccountTypeID int
	IsSuspicious          bool
	IsPrivate             bool
}

// NewUser は必須項目が設定された新規ユーザーを生成します。
func NewUser(userName username.UserName, accountTypeID int) *User {
	now := time.Now()

	return &User{
		Username:              userName,
		CreatedAt:             now,
		UpdatedAt:             now,
		AccountTypeID:         accountTypeID,
		OriginalAccountTypeID: accountTypeID,
	}
}

// NewFirebaseUser はFirebase UID紐付け済みの新規ユーザーを生成します。
func NewFirebaseUser(userName username.UserName, uid string, accountTypeID int) *User {
	now := time.Now()
	normalizedUID := strings.TrimSpace(uid)

	return &User{
		Username:              userName,
		FirebaseUID:           &normalizedUID,
		CreatedAt:             now,
		UpdatedAt:             now,
		AccountTypeID:         accountTypeID,
		OriginalAccountTypeID: accountTypeID,
	}
}

// IsPublic はユーザーが公開設定かを判定します。
func (u *User) IsPublic() bool {
	return !u.IsPrivate
}

// HasLinkedPlayer はユーザーにプレイヤーが紐づいているかを判定します。
func (u *User) HasLinkedPlayer() bool {
	return u.PlayerID != nil
}

// HasLinkedFirebase はユーザーに Firebase UID が紐づいているかを判定します。
func (u *User) HasLinkedFirebase() bool {
	return u.FirebaseUID != nil && *u.FirebaseUID != ""
}

// ChangePrivacy はユーザーの公開/非公開設定を変更します。
func (u *User) ChangePrivacy(isPrivate bool) {
	u.IsPrivate = isPrivate
	u.UpdatedAt = time.Now()
}

// ChangeAccountType はユーザー権限を変更します。
// 権限の正当性はユーザー集約の不変条件なので、ハンドラではなくドメインで検証します。
func (u *User) ChangeAccountType(accountTypeID int) error {
	if !constants.IsKnownAccountType(accountTypeID) {
		return ErrInvalidAccountType
	}
	u.AccountTypeID = accountTypeID
	u.UpdatedAt = time.Now()
	return nil
}

// LinkFirebaseUID はユーザーに Firebase UID を紐付けます。
func (u *User) LinkFirebaseUID(uid string) {
	normalizedUID := strings.TrimSpace(uid)
	if normalizedUID == "" {
		u.FirebaseUID = nil
	} else {
		u.FirebaseUID = &normalizedUID
	}
	u.UpdatedAt = time.Now()
}

// LinkPlayer はユーザーにプレイヤーを紐付けます。
func (u *User) LinkPlayer(playerID int) {
	u.PlayerID = &playerID
	u.UpdatedAt = time.Now()
}

// UnlinkPlayer はユーザーからプレイヤーとの紐付けを解除します。
func (u *User) UnlinkPlayer() {
	u.PlayerID = nil
	u.UpdatedAt = time.Now()
}
