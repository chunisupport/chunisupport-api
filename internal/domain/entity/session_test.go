package entity

import (
	"testing"
	"time"
)

func TestSessionExpiryJudgmentAtGivenTime(t *testing.T) {
	baseTime := time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		// Given: セッションの有効期限
		expiresAt time.Time
		// When: 判定時刻
		now time.Time
		// Then
		expected bool
	}{
		{
			name:      "有効期限が現在時刻より前の場合は期限切れ",
			expiresAt: baseTime.Add(-1 * time.Hour),
			now:       baseTime,
			expected:  true,
		},
		{
			name:      "有効期限が現在時刻より後の場合は有効",
			expiresAt: baseTime.Add(1 * time.Hour),
			now:       baseTime,
			expected:  false,
		},
		{
			name:      "有効期限と現在時刻が同一の場合は有効",
			expiresAt: baseTime,
			now:       baseTime,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			session := &Session{
				ID:        "session-1",
				UserID:    1,
				ExpiresAt: tt.expiresAt,
				CreatedAt: baseTime.Add(-24 * time.Hour),
			}

			// When
			result := session.IsExpired(tt.now)

			// Then
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}
