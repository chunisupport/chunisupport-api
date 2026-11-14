package firebaseauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	firebase "firebase.google.com/go/v4"
	firebaseauthsdk "firebase.google.com/go/v4/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubFirebaseUserGetter struct {
	record *firebaseauthsdk.UserRecord
	err    error
	gotCtx context.Context
	gotUID string
}

func (s *stubFirebaseUserGetter) GetUser(ctx context.Context, uid string) (*firebaseauthsdk.UserRecord, error) {
	s.gotCtx = ctx
	s.gotUID = uid
	return s.record, s.err
}

// finderCtxKey は呼び出し元から渡した context がスタブまで届いたことを識別するためのキーです。
type finderCtxKey struct{}

func newUserRecordWithProviders(providers ...*firebaseauthsdk.UserInfo) *firebaseauthsdk.UserRecord {
	return &firebaseauthsdk.UserRecord{
		UserInfo:         &firebaseauthsdk.UserInfo{UID: "firebase-uid"},
		ProviderUserInfo: providers,
	}
}

// newUserNotFoundError は本物の Firebase Admin SDK が IsUserNotFound と認識するエラーを生成します。
// エミュレーター向きの公開 API (FIREBASE_AUTH_EMULATOR_HOST + firebase.NewApp + app.Auth) だけを使い、
// httptest サーバーが空のユーザー一覧を返すことで SDK 内部のユーザー不在分岐を通します。
// internal パッケージの直接 import や偽の文字列エラーは使いません。
func newUserNotFoundError(t *testing.T) error {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	t.Setenv("FIREBASE_AUTH_EMULATOR_HOST", strings.TrimPrefix(srv.URL, "http://"))

	app, err := firebase.NewApp(context.Background(), &firebase.Config{ProjectID: "test-project"})
	require.NoError(t, err)
	client, err := app.Auth(context.Background())
	require.NoError(t, err)

	_, err = client.GetUser(context.Background(), "missing-user")
	require.Error(t, err)
	require.True(t, firebaseauthsdk.IsUserNotFound(err), "SDK がユーザー不在と認識できるエラーであること")
	return err
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
			stub := &stubFirebaseUserGetter{record: tt.record}
			finder := &providerAccountFinder{client: stub}
			ctx := context.WithValue(context.Background(), finderCtxKey{}, "caller")

			// When
			got, err := finder.FindProviderUID(ctx, "firebase-uid", "twitter.com")

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, "firebase-uid", stub.gotUID)
			assert.Equal(t, "caller", stub.gotCtx.Value(finderCtxKey{}))
		})
	}
}

func TestProviderAccountFinder_FindProviderUID_取得失敗はエラーを返す(t *testing.T) {
	// Given
	wantErr := errors.New("firebase down")
	stub := &stubFirebaseUserGetter{err: wantErr}
	finder := &providerAccountFinder{client: stub}
	ctx := context.WithValue(context.Background(), finderCtxKey{}, "caller")

	// When
	got, err := finder.FindProviderUID(ctx, "firebase-uid", "twitter.com")

	// Then
	assert.ErrorIs(t, err, wantErr)
	assert.Empty(t, got)
	assert.Equal(t, "firebase-uid", stub.gotUID)
	assert.Equal(t, "caller", stub.gotCtx.Value(finderCtxKey{}))
}

func TestProviderAccountFinder_FindProviderUID_ユーザーが存在しない場合は空文字を返す(t *testing.T) {
	// Given
	notFoundErr := newUserNotFoundError(t)
	stub := &stubFirebaseUserGetter{err: notFoundErr}
	finder := &providerAccountFinder{client: stub}
	ctx := context.WithValue(context.Background(), finderCtxKey{}, "caller")

	// When
	got, err := finder.FindProviderUID(ctx, "firebase-uid", "twitter.com")

	// Then
	require.NoError(t, err)
	assert.Empty(t, got)
	assert.Equal(t, "firebase-uid", stub.gotUID)
	assert.Equal(t, "caller", stub.gotCtx.Value(finderCtxKey{}))
}
