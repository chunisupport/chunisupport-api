package entity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUser_LinkFirebaseUID(t *testing.T) {
	baseTime := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		uid            string
		expectedLinked bool
		expectedUID    *string
	}{
		{
			name:           "UID を設定すると連携済みになる",
			uid:            "firebase-uid-1",
			expectedLinked: true,
			expectedUID:    stringPtr("firebase-uid-1"),
		},
		{
			name:           "空白だけのUIDは未連携として扱う",
			uid:            "  ",
			expectedLinked: false,
			expectedUID:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			user := &User{UpdatedAt: baseTime}

			// When
			user.LinkFirebaseUID(tt.uid)

			// Then
			assert.Equal(t, tt.expectedLinked, user.HasLinkedFirebase())
			assert.Equal(t, tt.expectedUID, user.FirebaseUID)
			require.False(t, user.UpdatedAt.IsZero())
			assert.NotEqual(t, baseTime, user.UpdatedAt)
		})
	}
}

func stringPtr(value string) *string {
	return &value
}
