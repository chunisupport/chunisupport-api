package models

import (
	"time"

	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/domain/vo/passwordhash"
	"github.com/Qman110101/chunisupport-api/internal/domain/vo/username"
)

// UserModel はデータベース用のUserモデルです。
type UserModel struct {
	ID            int       `db:"id"`
	Username      string    `db:"username"`
	PasswordHash  string    `db:"password_hash"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
	PlayerID      *int      `db:"player_id"`
	AccountTypeID int       `db:"account_type_id"`
	IsDeleted     bool      `db:"is_deleted"`
	IsPrivate     bool      `db:"is_private"`
}

// ToEntity はUserModelをentity.Userに変換します。
func (m *UserModel) ToEntity() (*entity.User, error) {
	uname, err := username.NewUserName(m.Username)
	if err != nil {
		return nil, err
	}

	phash, err := passwordhash.NewPasswordHash(m.PasswordHash)
	if err != nil {
		return nil, err
	}

	return &entity.User{
		ID:            m.ID,
		Username:      uname,
		PasswordHash:  phash,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		PlayerID:      m.PlayerID,
		AccountTypeID: m.AccountTypeID,
		IsDeleted:     m.IsDeleted,
		IsPrivate:     m.IsPrivate,
	}, nil
}

// FromEntity はentity.UserをUserModelに変換します。
func FromUserEntity(e *entity.User) *UserModel {
	return &UserModel{
		ID:            e.ID,
		Username:      e.Username.String(),
		PasswordHash:  e.PasswordHash.String(),
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
		PlayerID:      e.PlayerID,
		AccountTypeID: e.AccountTypeID,
		IsDeleted:     e.IsDeleted,
		IsPrivate:     e.IsPrivate,
	}
}

// UserWithPlayerRow はユーザーとプレイヤー情報のJOIN結果を格納するモデルです。
// StructScanでLEFT JOIN結果を取得するために使用します。
type UserWithPlayerRow struct {
	// ユーザー情報
	UserID       int    `db:"user_id"`
	Username     string `db:"username"`
	UserPlayerID *int   `db:"user_player_id"`

	// プレイヤー情報（LEFT JOINなのでnull許容）
	PlayerID             *int     `db:"player_id"`
	PlayerName           *string  `db:"player_name"`
	PlayerOfficialRating *float64 `db:"player_official_rating"`
	PlayerOverpowerValue *float64 `db:"player_overpower_value"`
}
