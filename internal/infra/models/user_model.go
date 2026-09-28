package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username"
)

// UserModel はデータベース用のUserモデルです。
type UserModel struct {
	ID            int       `db:"id"`
	Username      string    `db:"username"`
	FirebaseUID   *string   `db:"firebase_uid"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
	PlayerID      *int      `db:"player_id"`
	AccountTypeID int       `db:"account_type_id"`
	IsSuspicious  bool      `db:"is_suspicious"`
	IsPrivate     bool      `db:"is_private"`
}

func (m *UserModel) ToEntity() (*entity.User, error) {
	uname, err := username.NewUserName(m.Username)
	if err != nil {
		return nil, err
	}

	var firebaseUID *string
	if m.FirebaseUID != nil {
		normalizedUID := strings.TrimSpace(*m.FirebaseUID)
		if normalizedUID != "" {
			firebaseUID = &normalizedUID
		}
	}

	user := &entity.User{
		ID:            m.ID,
		Username:      uname,
		FirebaseUID:   firebaseUID,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
		PlayerID:      m.PlayerID,
		AccountTypeID: m.AccountTypeID,
		IsSuspicious:  m.IsSuspicious,
		IsPrivate:     m.IsPrivate,
	}
	user.MarkPersisted()
	return user, nil
}

// FromEntity はentity.UserをUserModelに変換します。
func FromUserEntity(e *entity.User) *UserModel {
	return &UserModel{
		ID:            e.ID,
		Username:      e.Username.String(),
		FirebaseUID:   e.FirebaseUID,
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
		PlayerID:      e.PlayerID,
		AccountTypeID: e.AccountTypeID,
		IsSuspicious:  e.IsSuspicious,
		IsPrivate:     e.IsPrivate,
	}
}

// UserWithPlayerRow はユーザーとプレイヤー情報のJOIN結果を格納するモデルです。
// StructScanでLEFT JOIN結果を取得するために使用します。
type UserWithPlayerRow struct {
	// ユーザー情報
	UserID            int       `db:"user_id"`
	Username          string    `db:"username"`
	FirebaseUID       *string   `db:"firebase_uid"`
	UserAccountTypeID int       `db:"user_account_type_id"`
	UserPlayerID      *int      `db:"user_player_id"`
	UserCreatedAt     time.Time `db:"user_created_at"`
	UserUpdatedAt     time.Time `db:"user_updated_at"`
	UserIsSuspicious  bool      `db:"user_is_suspicious"`
	UserIsPrivate     bool      `db:"user_is_private"`

	// プレイヤー情報（LEFT JOINなのでnull許容）
	PlayerID               *int     `db:"player_id"`
	PlayerName             *string  `db:"player_name"`
	PlayerCalculatedRating *float64 `db:"player_calculated_rating"`
	PlayerOverpowerValue   *float64 `db:"player_overpower_value"`
}

// ToEntity はJOIN結果をユーザーとプレイヤーの組へ変換します。
// ユーザーの変換は UserModel と共通にし、値オブジェクトの検証を一か所に保ちます。
func (r *UserWithPlayerRow) ToEntity() (entity.UserWithPlayer, error) {
	userModel := UserModel{
		ID:            r.UserID,
		Username:      r.Username,
		FirebaseUID:   r.FirebaseUID,
		CreatedAt:     r.UserCreatedAt,
		UpdatedAt:     r.UserUpdatedAt,
		PlayerID:      r.UserPlayerID,
		AccountTypeID: r.UserAccountTypeID,
		IsSuspicious:  r.UserIsSuspicious,
		IsPrivate:     r.UserIsPrivate,
	}
	user, err := userModel.ToEntity()
	if err != nil {
		return entity.UserWithPlayer{}, fmt.Errorf("failed to create user: %w", err)
	}

	result := entity.UserWithPlayer{User: *user}
	if r.PlayerID == nil {
		return result, nil
	}

	player := &entity.Player{
		ID:               *r.PlayerID,
		CalculatedRating: r.PlayerCalculatedRating,
		OverpowerValue:   r.PlayerOverpowerValue,
	}
	if r.PlayerName != nil {
		player.Name, err = playername.NewPlayerName(*r.PlayerName)
		if err != nil {
			return entity.UserWithPlayer{}, fmt.Errorf("failed to create player name: %w", err)
		}
	}
	result.Player = player
	return result, nil
}
