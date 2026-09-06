package entity

import (
	"errors"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/constants"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
)

var (
	ErrInvalidAccountType   = errors.New("invalid account type")
	ErrCannotDemoteOwnAdmin = errors.New("cannot demote own admin account")
)

// User はユーザーのエンティティを表します。
type User struct {
	ID            int
	Username      username.UserName
	FirebaseUID   *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	PlayerID      *int
	AccountTypeID int
	IsSuspicious  bool
	IsPrivate     bool

	// persistedAccountTypeID は保存済み集約から復元した権限IDです。
	// Save時に変更前の値で競合を検出するため、外部へは公開しません。
	persistedAccountTypeID int
}

// NewUser は必須項目が設定された新規ユーザーを生成します。
func NewUser(userName username.UserName, accountTypeID int) *User {
	now := time.Now().UTC()

	return &User{
		Username:      userName,
		CreatedAt:     now,
		UpdatedAt:     now,
		AccountTypeID: accountTypeID,
	}
}

// NewFirebaseUser はFirebase UID紐付け済みの新規ユーザーを生成します。
func NewFirebaseUser(userName username.UserName, uid string, accountTypeID int) *User {
	now := time.Now().UTC()
	normalizedUID := strings.TrimSpace(uid)

	return &User{
		Username:      userName,
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

// ChangeAccountType は管理者による権限変更を適用します。
// 自分自身がADMINである状態からADMIN未満へ変更すると、管理者が不在になる事故を防ぐため拒否します。
func (u *User) ChangeAccountType(requesterID int, accountTypeID int) error {
	if !constants.IsKnownAccountType(accountTypeID) {
		return ErrInvalidAccountType
	}
	if u.ID == requesterID && u.AccountTypeID == constants.AccountTypeAdmin && accountTypeID != constants.AccountTypeAdmin {
		return ErrCannotDemoteOwnAdmin
	}

	u.AccountTypeID = accountTypeID
	u.UpdatedAt = time.Now().UTC()
	return nil
}

// MarkPersisted はリポジトリが復元または保存した権限IDを記録します。
// 権限変更後も更新前の値で競合検出できるよう、永続化境界でだけ呼び出します。
func (u *User) MarkPersisted() {
	u.persistedAccountTypeID = u.AccountTypeID
}

// PersistedAccountTypeID は最後に永続化された権限IDを返します。
// テストなどで直接構築された既存集約は、現在の値を保存済み値として扱います。
func (u *User) PersistedAccountTypeID() int {
	if u.persistedAccountTypeID == 0 {
		return u.AccountTypeID
	}
	return u.persistedAccountTypeID
}

// ChangePrivacy はユーザーの公開/非公開設定を変更します。
func (u *User) ChangePrivacy(isPrivate bool) {
	u.IsPrivate = isPrivate
	u.UpdatedAt = time.Now().UTC()
}

// ChangeUsername は検証済みのユーザー名へ変更し、更新日時を更新します。
func (u *User) ChangeUsername(userName username.UserName) {
	u.Username = userName
	u.UpdatedAt = time.Now().UTC()
}

// LinkFirebaseUID はユーザーに Firebase UID を紐付けます。
func (u *User) LinkFirebaseUID(uid string) {
	normalizedUID := strings.TrimSpace(uid)
	if normalizedUID == "" {
		u.FirebaseUID = nil
	} else {
		u.FirebaseUID = &normalizedUID
	}
	u.UpdatedAt = time.Now().UTC()
}

// LinkPlayer はユーザーにプレイヤーを紐付けます。
func (u *User) LinkPlayer(playerID int) {
	u.PlayerID = &playerID
	u.UpdatedAt = time.Now().UTC()
}

// UnlinkPlayer はユーザーからプレイヤーとの紐付けを解除します。
func (u *User) UnlinkPlayer() {
	u.PlayerID = nil
	u.UpdatedAt = time.Now().UTC()
}
