package app

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type stubReopenable struct {
	reopenCalled bool
	closeCalled  bool
	reopenErr    error
	closeErr     error
}

func (s *stubReopenable) Write(p []byte) (int, error) {
	return len(p), nil
}

func (s *stubReopenable) Reopen() error {
	s.reopenCalled = true
	return s.reopenErr
}

func (s *stubReopenable) Close() error {
	s.closeCalled = true
	return s.closeErr
}

func TestLogManager_ReopenAll(t *testing.T) {
	appErr := errors.New("app reopen failed")
	accessErr := errors.New("access reopen failed")
	tests := []struct {
		name             string
		appHandler       *stubReopenable
		accessWriter     *stubReopenable
		wantErrs         []error
		wantAppCalled    bool
		wantAccessCalled bool
	}{
		{
			name:             "アプリログとアクセスログを開き直す",
			appHandler:       &stubReopenable{},
			accessWriter:     &stubReopenable{},
			wantAppCalled:    true,
			wantAccessCalled: true,
		},
		{
			name:             "アプリログがnilでもアクセスログを開き直す",
			accessWriter:     &stubReopenable{},
			wantAccessCalled: true,
		},
		{
			name:       "両方のエラーをまとめて返す",
			appHandler: &stubReopenable{reopenErr: appErr},
			accessWriter: &stubReopenable{
				reopenErr: accessErr,
			},
			wantErrs:         []error{appErr, accessErr},
			wantAppCalled:    true,
			wantAccessCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &LogManager{}
			if tt.appHandler != nil {
				manager.AppHandler = tt.appHandler
			}
			if tt.accessWriter != nil {
				manager.AccessWriter = tt.accessWriter
			}

			err := manager.ReopenAll()

			if len(tt.wantErrs) == 0 {
				assert.NoError(t, err)
			}
			for _, wantErr := range tt.wantErrs {
				assert.ErrorIs(t, err, wantErr)
			}
			if tt.appHandler != nil {
				assert.Equal(t, tt.wantAppCalled, tt.appHandler.reopenCalled)
			}
			if tt.accessWriter != nil {
				assert.Equal(t, tt.wantAccessCalled, tt.accessWriter.reopenCalled)
			}
		})
	}
}

func TestLogManager_Close(t *testing.T) {
	appErr := errors.New("app close failed")
	accessErr := errors.New("access close failed")
	tests := []struct {
		name             string
		appHandler       *stubReopenable
		accessWriter     *stubReopenable
		wantErrs         []error
		wantAppCalled    bool
		wantAccessCalled bool
	}{
		{
			name:             "アプリログとアクセスログを閉じる",
			appHandler:       &stubReopenable{},
			accessWriter:     &stubReopenable{},
			wantAppCalled:    true,
			wantAccessCalled: true,
		},
		{
			name:             "アプリログがnilでもアクセスログを閉じる",
			accessWriter:     &stubReopenable{},
			wantAccessCalled: true,
		},
		{
			name:       "両方のエラーをまとめて返す",
			appHandler: &stubReopenable{closeErr: appErr},
			accessWriter: &stubReopenable{
				closeErr: accessErr,
			},
			wantErrs:         []error{appErr, accessErr},
			wantAppCalled:    true,
			wantAccessCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &LogManager{}
			if tt.appHandler != nil {
				manager.AppHandler = tt.appHandler
			}
			if tt.accessWriter != nil {
				manager.AccessWriter = tt.accessWriter
			}

			err := manager.Close()

			if len(tt.wantErrs) == 0 {
				assert.NoError(t, err)
			}
			for _, wantErr := range tt.wantErrs {
				assert.ErrorIs(t, err, wantErr)
			}
			if tt.appHandler != nil {
				assert.Equal(t, tt.wantAppCalled, tt.appHandler.closeCalled)
			}
			if tt.accessWriter != nil {
				assert.Equal(t, tt.wantAccessCalled, tt.accessWriter.closeCalled)
			}
		})
	}
}
