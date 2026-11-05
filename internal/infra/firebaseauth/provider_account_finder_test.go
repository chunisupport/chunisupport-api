package firebaseauth

import (
	"context"
	"errors"
	"testing"

	firebaseauthsdk "firebase.google.com/go/v4/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubFirebaseUserGetter struct {
	record *firebaseauthsdk.UserRecord
	err    error
}

func (s *stubFirebaseUserGetter) GetUser(_ context.Context, _ string) (*firebaseauthsdk.UserRecord, error) {
	return s.record, s.err
}

func newUserRecordWithProviders(providers ...*firebaseauthsdk.UserInfo) *firebaseauthsdk.UserRecord {
	return &firebaseauthsdk.UserRecord{
		UserInfo:         &firebaseauthsdk.UserInfo{UID: "firebase-uid"},
		ProviderUserInfo: providers,
	}
}

func TestProviderAccountFinder_FindProviderUID(t *testing.T) {
	tests := []struct {
		name   string
		record *firebaseauthsdk.UserRecord
		want   string
	}{
		{
			name: "指定したプロバイダが連携されていればそのユーザーIDを返す",
			record: newUserRecordWithProviders(
				&firebaseauthsdk.UserInfo{ProviderID: "google.com", UID: "google-uid"},
				&firebaseauthsdk.UserInfo{ProviderID: "twitter.com", UID: "123"},
			),
			want: "123",
		},
		{
			name:   "指定したプロバイダが連携されていなければ空文字を返す",
			record: newUserRecordWithProviders(&firebaseauthsdk.UserInfo{ProviderID: "google.com", UID: "google-uid"}),
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			finder := &providerAccountFinder{client: &stubFirebaseUserGetter{record: tt.record}}

			// When
			got, err := finder.FindProviderUID(context.Background(), "firebase-uid", "twitter.com")

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProviderAccountFinder_FindProviderUID_取得失敗はエラーを返す(t *testing.T) {
	// Given
	wantErr := errors.New("firebase down")
	finder := &providerAccountFinder{client: &stubFirebaseUserGetter{err: wantErr}}

	// When
	got, err := finder.FindProviderUID(context.Background(), "firebase-uid", "twitter.com")

	// Then
	assert.ErrorIs(t, err, wantErr)
	assert.Empty(t, got)
}
