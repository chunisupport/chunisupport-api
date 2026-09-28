package repository

import (
	"errors"
	"testing"

	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestWrapGoalGroupDuplicateError(t *testing.T) {
	// Given
	err := &mysql.MySQLError{
		Number:  mysqlDuplicateEntryErrorNumber,
		Message: "Duplicate entry '1-攻略中' for key 'uq_goal_groups_user_name'",
	}

	// When
	got := wrapGoalGroupDuplicateError(err)

	// Then
	assert.ErrorIs(t, got, domainrepo.ErrGoalGroupConflict)
}

func TestMySQLErrorWrappers_原因エラーを保持する(t *testing.T) {
	tests := []struct {
		name string
		// Given: 変換対象のMySQLエラーとラップ関数
		wrap     func(error) error
		mysqlErr *mysql.MySQLError
		// Then: 期待するドメインエラー
		wantErr error
	}{
		{
			name:     "firebase_uid の重複",
			wrap:     wrapFirebaseUIDDuplicateError,
			mysqlErr: &mysql.MySQLError{Number: mysqlDuplicateEntryErrorNumber, Message: "Duplicate entry 'uid-1' for key 'uk_users_firebase_uid'"},
			wantErr:  domainrepo.ErrFirebaseUIDAlreadyLinked,
		},
		{
			name:     "username の重複",
			wrap:     wrapUsernameDuplicateError,
			mysqlErr: &mysql.MySQLError{Number: mysqlDuplicateEntryErrorNumber, Message: "Duplicate entry 'testuser' for key 'username'"},
			wantErr:  domainrepo.ErrDuplicateUsername,
		},
		{
			name:     "official_idx の重複",
			wrap:     wrapOfficialIdxDuplicateError,
			mysqlErr: &mysql.MySQLError{Number: mysqlDuplicateEntryErrorNumber, Message: "Duplicate entry '1' for key 'official_idx'"},
			wantErr:  domainrepo.ErrDuplicateOfficialIdx,
		},
		{
			name:     "称号の重複",
			wrap:     wrapHonorDuplicateError,
			mysqlErr: &mysql.MySQLError{Number: mysqlDuplicateEntryErrorNumber, Message: "Duplicate entry 'a-1' for key 'unique_honor_name_type'"},
			wantErr:  domainrepo.ErrHonorConflict,
		},
		{
			name:     "参照中の称号の削除",
			wrap:     wrapHonorReferencedError,
			mysqlErr: &mysql.MySQLError{Number: mysqlCannotDeleteOrUpdateParentRowErrorNumber, Message: "Cannot delete or update a parent row"},
			wantErr:  domainrepo.ErrHonorConflict,
		},
		{
			name:     "目標グループ名の重複",
			wrap:     wrapGoalGroupDuplicateError,
			mysqlErr: &mysql.MySQLError{Number: mysqlDuplicateEntryErrorNumber, Message: "Duplicate entry '1-攻略中' for key 'uq_goal_groups_user_name'"},
			wantErr:  domainrepo.ErrGoalGroupConflict,
		},
		{
			name:     "APIトークン名の重複",
			wrap:     wrapAPITokenDuplicateError,
			mysqlErr: &mysql.MySQLError{Number: mysqlDuplicateEntryErrorNumber, Message: "Duplicate entry '1-token' for key 'uq_api_tokens_user_name'"},
			wantErr:  domainrepo.ErrAPITokenConflict,
		},
		{
			name:     "バージョン名の重複",
			wrap:     wrapVersionDuplicateError,
			mysqlErr: &mysql.MySQLError{Number: mysqlDuplicateEntryErrorNumber, Message: "Duplicate entry 'CHUNITHM' for key 'name'"},
			wantErr:  domainrepo.ErrVersionConflict,
		},
		{
			name:     "スコア履歴の登録日時の重複",
			wrap:     wrapScoreHistoryInsertError,
			mysqlErr: &mysql.MySQLError{Number: mysqlDuplicateEntryErrorNumber, Message: "Duplicate entry for key 'PRIMARY'"},
			wantErr:  domainrepo.ErrScoreHistoryTimestampConflict,
		},
		{
			name:     "プレイヤー指標履歴の登録日時の重複",
			wrap:     wrapPlayerMetricHistoryInsertError,
			mysqlErr: &mysql.MySQLError{Number: mysqlDuplicateEntryErrorNumber, Message: "Duplicate entry for key 'PRIMARY'"},
			wantErr:  domainrepo.ErrPlayerMetricHistoryTimestampConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got := tt.wrap(tt.mysqlErr)

			// Then
			assert.ErrorIs(t, got, tt.wantErr)
			var mysqlErr *mysql.MySQLError
			require.ErrorAs(t, got, &mysqlErr)
			assert.Same(t, tt.mysqlErr, mysqlErr)
		})
	}
}
