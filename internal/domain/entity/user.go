package entity

import (
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/passwordhash"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
)

// User はユーザーのエンティティを表します。
type User struct {
	ID            int
	Username      username.UserName
	FirebaseUID   *string
	PasswordHash  passwordhash.PasswordHash
	CreatedAt     time.Time
	UpdatedAt     time.Time
	PlayerID      *int
	AccountTypeID int
	IsSuspicious  bool
	IsPrivate     bool
}

// NewUser は必須項目が設定された新規ユーザーを生成します。
func NewUser(userName username.UserName, hash passwordhash.PasswordHash, accountTypeID int) *User {
	now := time.Now()

	return &User{
		Username:      userName,
		PasswordHash:  hash,
		CreatedAt:     now,
		UpdatedAt:     now,
		AccountTypeID: accountTypeID,
	}
}

// NewFirebaseUser はFirebase UID紐付け済みのパスワードなし新規ユーザーを生成します。
func NewFirebaseUser(userName username.UserName, uid string, accountTypeID int) *User {
	now := time.Now()
	normalizedUID := strings.TrimSpace(uid)

	return &User{
		Username:      userName,
		PasswordHash:  passwordhash.NewEmptyPasswordHash(),
		FirebaseUID:   &normalizedUID,
		CreatedAt:     now,
		UpdatedAt:     now,
		AccountTypeID: accountTypeID,
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
