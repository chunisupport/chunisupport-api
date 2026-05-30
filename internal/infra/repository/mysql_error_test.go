package repository

import (
	"errors"
	"testing"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
)

func TestWrapFirebaseUIDDuplicateError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantErr error
	}{
		{
			name: "firebase_uid の UNIQUE 制約違反はドメインエラーへ変換する",
			err: &mysql.MySQLError{
				Number:  mysqlDuplicateEntryErrorNumber,
				Message: "Duplicate entry 'uid-1' for key 'uk_users_firebase_uid'",
			},
			wantErr: domainrepo.ErrFirebaseUIDAlreadyLinked,
		},
		{
			name: "他キーの duplicate entry は変換しない",
			err: &mysql.MySQLError{
				Number:  mysqlDuplicateEntryErrorNumber,
				Message: "Duplicate entry 'foo' for key 'username'",
			},
		},
		{
			name:    "MySQL 以外のエラーは変換しない",
			err:     errors.New("other error"),
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapFirebaseUIDDuplicateError(tt.err)

			if tt.wantErr != nil {
				assert.ErrorIs(t, got, tt.wantErr)
				return
			}

			assert.ErrorIs(t, got, tt.err)
			assert.NotErrorIs(t, got, domainrepo.ErrFirebaseUIDAlreadyLinked)
		})
	}
}

func TestWrapUsernameDuplicateError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantErr error
	}{
		{
			name: "username の UNIQUE 制約違反はドメインエラーへ変換する",
			err: &mysql.MySQLError{
				Number:  mysqlDuplicateEntryErrorNumber,
				Message: "Duplicate entry 'testuser' for key 'username'",
			},
			wantErr: domainrepo.ErrDuplicateUsername,
		},
		{
			name: "他キーの duplicate entry は変換しない",
			err: &mysql.MySQLError{
				Number:  mysqlDuplicateEntryErrorNumber,
				Message: "Duplicate entry 'uid-1' for key 'uk_users_firebase_uid'",
			},
		},
		{
			name:    "MySQL 以外のエラーは変換しない",
			err:     errors.New("other error"),
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapUsernameDuplicateError(tt.err)

			if tt.wantErr != nil {
				assert.ErrorIs(t, got, tt.wantErr)
				return
			}

			assert.ErrorIs(t, got, tt.err)
			assert.NotErrorIs(t, got, domainrepo.ErrDuplicateUsername)
		})
	}
}
