package entity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFriendRequest(t *testing.T) {
	now := time.Date(2026, 8, 30, 1, 2, 3, 0, time.UTC)

	friendship, err := NewFriendRequest(1, 2, now)

	require.NoError(t, err)
	assert.Equal(t, 1, friendship.UserID)
	assert.Equal(t, 2, friendship.FriendUserID)
	assert.Equal(t, FriendshipStatusPending, friendship.StatusID)
	assert.Equal(t, now, friendship.RequestedAt)
	assert.Nil(t, friendship.AcceptedAt)
	assert.Equal(t, now, friendship.CreatedAt)
	assert.Equal(t, now, friendship.UpdatedAt)
	assert.True(t, friendship.IsPending())
	assert.False(t, friendship.IsAccepted())
}

func TestNewAcceptedFriendship(t *testing.T) {
	requestedAt := time.Date(2026, 8, 29, 1, 2, 3, 0, time.UTC)
	acceptedAt := requestedAt.Add(time.Hour)

	friendship, err := NewAcceptedFriendship(1, 2, requestedAt, acceptedAt)

	require.NoError(t, err)
	assert.Equal(t, FriendshipStatusAccepted, friendship.StatusID)
	require.NotNil(t, friendship.AcceptedAt)
	assert.Equal(t, acceptedAt, *friendship.AcceptedAt)
	assert.Equal(t, requestedAt, friendship.CreatedAt)
	assert.Equal(t, acceptedAt, friendship.UpdatedAt)
	assert.False(t, friendship.IsPending())
	assert.True(t, friendship.IsAccepted())
}

func TestFriendship_Accept(t *testing.T) {
	requestedAt := time.Date(2026, 8, 29, 1, 2, 3, 0, time.UTC)
	acceptedAt := requestedAt.Add(time.Hour)
	friendship, err := NewFriendRequest(1, 2, requestedAt)
	require.NoError(t, err)

	err = friendship.Accept(acceptedAt)

	require.NoError(t, err)
	assert.True(t, friendship.IsAccepted())
	require.NotNil(t, friendship.AcceptedAt)
	assert.Equal(t, acceptedAt, *friendship.AcceptedAt)
	assert.Equal(t, acceptedAt, friendship.UpdatedAt)
}

func TestFriendship_Accept_承認済みでも時刻を更新できる(t *testing.T) {
	requestedAt := time.Date(2026, 8, 29, 1, 2, 3, 0, time.UTC)
	firstAcceptedAt := requestedAt.Add(time.Hour)
	secondAcceptedAt := firstAcceptedAt.Add(time.Hour)
	friendship, err := NewAcceptedFriendship(1, 2, requestedAt, firstAcceptedAt)
	require.NoError(t, err)

	err = friendship.Accept(secondAcceptedAt)

	require.NoError(t, err)
	require.NotNil(t, friendship.AcceptedAt)
	assert.Equal(t, secondAcceptedAt, *friendship.AcceptedAt)
}

func TestFriendship_Validate(t *testing.T) {
	now := time.Date(2026, 8, 30, 1, 2, 3, 0, time.UTC)
	tests := []struct {
		name       string
		friendship *Friendship
		wantErr    bool
	}{
		{name: "申請中は有効", friendship: &Friendship{UserID: 1, FriendUserID: 2, StatusID: FriendshipStatusPending, RequestedAt: now}},
		{name: "承認済みは承認日時があれば有効", friendship: &Friendship{UserID: 1, FriendUserID: 2, StatusID: FriendshipStatusAccepted, RequestedAt: now, AcceptedAt: &now}},
		{name: "ブロック中は有効", friendship: &Friendship{UserID: 1, FriendUserID: 2, StatusID: FriendshipStatusBlocked, RequestedAt: now}},
		{name: "nilは無効", friendship: nil, wantErr: true},
		{name: "ユーザーIDが0は無効", friendship: &Friendship{FriendUserID: 2, StatusID: FriendshipStatusPending, RequestedAt: now}, wantErr: true},
		{name: "フレンドユーザーIDが0は無効", friendship: &Friendship{UserID: 1, StatusID: FriendshipStatusPending, RequestedAt: now}, wantErr: true},
		{name: "自分自身は無効", friendship: &Friendship{UserID: 1, FriendUserID: 1, StatusID: FriendshipStatusPending, RequestedAt: now}, wantErr: true},
		{name: "申請中に承認日時があると無効", friendship: &Friendship{UserID: 1, FriendUserID: 2, StatusID: FriendshipStatusPending, RequestedAt: now, AcceptedAt: &now}, wantErr: true},
		{name: "承認済みに承認日時がないと無効", friendship: &Friendship{UserID: 1, FriendUserID: 2, StatusID: FriendshipStatusAccepted, RequestedAt: now}, wantErr: true},
		{name: "未知のステータスは無効", friendship: &Friendship{UserID: 1, FriendUserID: 2, StatusID: 99, RequestedAt: now}, wantErr: true},
		{name: "申請日時がないと無効", friendship: &Friendship{UserID: 1, FriendUserID: 2, StatusID: FriendshipStatusPending}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.friendship.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFriendship_Accept_承認できない状態(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name       string
		friendship *Friendship
	}{
		{name: "nil", friendship: nil},
		{name: "ブロック中", friendship: &Friendship{UserID: 1, FriendUserID: 2, StatusID: FriendshipStatusBlocked, RequestedAt: now}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, tt.friendship.Accept(now.Add(time.Hour)))
		})
	}
}

func TestFriendship_nilの状態判定(t *testing.T) {
	var friendship *Friendship

	assert.False(t, friendship.IsPending())
	assert.False(t, friendship.IsAccepted())
}
