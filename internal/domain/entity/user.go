package entity

import (
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/passwordhash"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
)

// User はユーザーのエンティティを表します。
type User struct {
	ID            int
	Username      username.UserName
	PasswordHash  passwordhash.PasswordHash
	CreatedAt     time.Time
	UpdatedAt     time.Time
	PlayerID      *int
	AccountTypeID int
	IsDeleted     bool
	IsPrivate     bool
}

// IsActive はユーザーが有効（削除されていない）かを判定します。
func (u *User) IsActive() bool {
	return !u.IsDeleted
}

// IsPublic はユーザーが公開設定かを判定します。
func (u *User) IsPublic() bool {
	return !u.IsPrivate
}

// HasLinkedPlayer はユーザーにプレイヤーが紐づいているかを判定します。
func (u *User) HasLinkedPlayer() bool {
	return u.PlayerID != nil
}

// ChangePrivacy はユーザーの公開/非公開設定を変更します。
func (u *User) ChangePrivacy(isPrivate bool) {
	u.IsPrivate = isPrivate
	u.UpdatedAt = time.Now()
}

// ChangePassword はユーザーのパスワードハッシュを変更します。
func (u *User) ChangePassword(hash passwordhash.PasswordHash) {
	u.PasswordHash = hash
	u.UpdatedAt = time.Now()
}

// Delete はユーザーを論理削除します。
func (u *User) Delete() {
	u.IsDeleted = true
	u.UpdatedAt = time.Now()
}

// Restore はユーザーを復活させます。
func (u *User) Restore() {
	u.IsDeleted = false
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
