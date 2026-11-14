package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserModel_ToEntity(t *testing.T) {
	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	firebaseUID := "firebase-uid"

	tests := []struct {
		name  string
		model UserModel
		// 復元結果の固定期待値です。モデル由来の値を参照せず、リテラルで固定します。
		wantFirebaseUID            *string
		wantPlayerID               *int
		wantAccountTypeID          int
		wantPersistedAccountTypeID int
		wantIsPrivate              bool
		wantIsSuspicious           bool
		// MarkPersisted検証のために変更する権限IDの固定値です。復元時と異なる既知の権限を指定します。
		changeTo int
	}{
		{
			name: "Firebase UID があるユーザーを復元できる",
			model: UserModel{
				ID:            1,
				Username:      "firebaseuser",
				FirebaseUID:   &firebaseUID,
				CreatedAt:     now,
				UpdatedAt:     now,
				PlayerID:      new(10),
				AccountTypeID: 2,
				IsSuspicious:  true,
				IsPrivate:     true,
			},
			wantFirebaseUID:            &firebaseUID,
			wantPlayerID:               new(10),
			wantAccountTypeID:          2,
			wantPersistedAccountTypeID: 2,
			wantIsPrivate:              true,
			wantIsSuspicious:           true,
			changeTo:                   1,
		},
		{
			name: "Firebase UID がないユーザーも復元できる",
			model: UserModel{
				ID:            2,
				Username:      "normaluser",
				CreatedAt:     now,
				UpdatedAt:     now,
				AccountTypeID: 1,
			},
			wantPlayerID:               nil,
			wantAccountTypeID:          1,
			wantPersistedAccountTypeID: 1,
			wantIsPrivate:              false,
			wantIsSuspicious:           false,
			changeTo:                   2,
		},
		{
			name: "Firebase UID が空白なら未連携として復元する",
			model: UserModel{
				ID:            3,
				Username:      "invaliduser",
				FirebaseUID:   new(" "),
				CreatedAt:     now,
				UpdatedAt:     now,
				AccountTypeID: 1,
				IsSuspicious:  true,
			},
			wantFirebaseUID:            nil,
			wantPlayerID:               nil,
			wantAccountTypeID:          1,
			wantPersistedAccountTypeID: 1,
			wantIsPrivate:              false,
			wantIsSuspicious:           true,
			changeTo:                   2,
		},
		{
			name: "Firebase UID の前後空白を除去して復元する",
			model: UserModel{
				ID:            4,
				Username:      "trimuser",
				FirebaseUID:   new("  firebase-uid  "),
				CreatedAt:     now,
				UpdatedAt:     now,
				AccountTypeID: 1,
			},
			wantFirebaseUID:            &firebaseUID,
			wantPlayerID:               nil,
			wantAccountTypeID:          1,
			wantPersistedAccountTypeID: 1,
			wantIsPrivate:              false,
			wantIsSuspicious:           false,
			changeTo:                   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			model := tt.model

			// When
			got, err := model.ToEntity()

			// Then
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantFirebaseUID, got.FirebaseUID)
			assert.Equal(t, tt.wantPlayerID, got.PlayerID)
			assert.Equal(t, tt.wantAccountTypeID, got.AccountTypeID)
			assert.Equal(t, tt.wantIsPrivate, got.IsPrivate)
			assert.Equal(t, tt.wantIsSuspicious, got.IsSuspicious)
			// 復元直後は保存済み権限が復元値として記録されることを検証します。
			assert.Equal(t, tt.wantPersistedAccountTypeID, got.PersistedAccountTypeID())

			// Then: MarkPersistedの検証として権限変更後も保存済み値が復元時のままであることを検証します。
			// 他者として変更するため、自己降格の拒否に当たりません。
			require.NoError(t, got.ChangeAccountType(999, tt.changeTo))
			assert.Equal(t, tt.changeTo, got.AccountTypeID)
			assert.Equal(t, tt.wantPersistedAccountTypeID, got.PersistedAccountTypeID())
		})
	}
}
