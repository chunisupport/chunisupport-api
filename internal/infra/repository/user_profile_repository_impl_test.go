package repository

import (
	"context"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// UPSERTはMySQL固有のため、X連携と同様にSQLiteで取得を検証し、削除も検証します。
func TestUserProfileRepository_FindAndDelete(t *testing.T) {
	tests := []struct {
		name    string
		id      int
		want    string
		invalid bool
	}{
		{name: "指定ユーザーを取得", id: 7, want: "よろしく"},
		{name: "別ユーザーを取得", id: 8, want: "別ユーザー"},
		{name: "未設定はnil", id: 9},
		{name: "不正な保存値はエラー", id: 10, invalid: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			db, err := sqlx.Open("sqlite", ":memory:")
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, db.Close()) })
			_, err = db.Exec(`CREATE TABLE user_profiles (user_id INTEGER PRIMARY KEY, comment TEXT NOT NULL);
INSERT INTO user_profiles (user_id,comment) VALUES (7,'よろしく'),(8,'別ユーザー'),(10,'');`)
			require.NoError(t, err)
			repo := NewUserProfileRepository()
			// When
			got, err := repo.FindByUserID(context.Background(), db, tt.id)
			// Then
			if tt.invalid {
				require.ErrorIs(t, err, repository.ErrRepositoryOperationFailed)
				return
			}
			require.NoError(t, err)
			if tt.want == "" {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, tt.id, got.UserID())
				assert.Equal(t, tt.want, got.Comment().String())
			}
			// When: 未設定の削除も繰り返せます。
			require.NoError(t, repo.DeleteByUserID(context.Background(), db, tt.id))
			require.NoError(t, repo.DeleteByUserID(context.Background(), db, tt.id))
			got, err = repo.FindByUserID(context.Background(), db, tt.id)
			// Then
			require.NoError(t, err)
			assert.Nil(t, got)
			other, err := repo.FindByUserID(context.Background(), db, 8)
			require.NoError(t, err)
			if tt.id != 8 {
				require.NotNil(t, other)
			}
		})
	}
}
