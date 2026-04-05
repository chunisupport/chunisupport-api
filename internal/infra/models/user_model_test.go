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
		wantErr         string
	}{
		{
			name: "Firebase UID があるユーザーは空の password_hash を許容する",
			model: UserModel{
				ID:            1,
				Username:      "firebaseuser",
				FirebaseUID:   &firebaseUID,
				PasswordHash:  "",
				CreatedAt:     now,
				UpdatedAt:     now,
				AccountTypeID: 1,
			},
			wantPassword:    "",
			wantFirebaseUID: &firebaseUID,
		},
		{
			name: "Firebase UID がないユーザーの空の password_hash はエラーになる",
			model: UserModel{
				ID:            2,
				Username:      "normaluser",
				PasswordHash:  "",
				CreatedAt:     now,
				UpdatedAt:     now,
				AccountTypeID: 1,
			},
			wantErr: "password hash cannot be empty",
		},
		{
			name: "Firebase UID が空文字だけのユーザーの空の password_hash はエラーになる",
			model: UserModel{
				ID:            3,
				Username:      "invaliduser",
				FirebaseUID:   ptr(" "),
				PasswordHash:  "",
				CreatedAt:     now,
				UpdatedAt:     now,
				AccountTypeID: 1,
			},
			wantErr: "password hash cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			model := tt.model

			// When
			got, err := model.ToEntity()

			// Then
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}

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
