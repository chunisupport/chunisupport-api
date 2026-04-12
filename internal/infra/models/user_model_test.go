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
		name            string
		model           UserModel
		wantPassword    string
		wantFirebaseUID *string
	}{
		{
			name: "Firebase UID があるユーザーはパスワードなしで復元する",
			model: UserModel{
				ID:            1,
				Username:      "firebaseuser",
				FirebaseUID:   &firebaseUID,
				CreatedAt:     now,
				UpdatedAt:     now,
				AccountTypeID: 1,
			},
			wantPassword:    "",
			wantFirebaseUID: &firebaseUID,
		},
		{
			name: "Firebase UID がないユーザーもパスワードなしで復元する",
			model: UserModel{
				ID:            2,
				Username:      "normaluser",
				CreatedAt:     now,
				UpdatedAt:     now,
				AccountTypeID: 1,
			},
			wantPassword: "",
		},
		{
			name: "Firebase UID が空白でもパスワードなしで復元する",
			model: UserModel{
				ID:            3,
				Username:      "invaliduser",
				FirebaseUID:   ptr(" "),
				CreatedAt:     now,
				UpdatedAt:     now,
				AccountTypeID: 1,
			},
			wantPassword:    "",
			wantFirebaseUID: ptr(" "),
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
			assert.Equal(t, tt.wantPassword, got.PasswordHash.String())
			assert.Equal(t, tt.wantFirebaseUID, got.FirebaseUID)
		})
	}
}

func ptr(value string) *string {
	return &value
}
