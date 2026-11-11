package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestUserProfilesMigration は未設定を行なしで表す主キーと削除時の連鎖を確認します。
func TestUserProfilesMigration(t *testing.T) {
	// Given
	up := readNormalizedMigrationSQL(t, "000063_create_user_profiles.up.sql")
	down := readNormalizedMigrationSQL(t, "000063_create_user_profiles.down.sql")
	schema := readNormalizedMigrationSQL(t, "../schema_mysql.sql")
	// When / Then
	for _, sql := range []string{up, schema} {
		assert.Contains(t, sql, "CREATE TABLE `user_profiles`")
		assert.Contains(t, sql, "`comment` varchar(100) COLLATE utf8mb4_unicode_ci NOT NULL")
		assert.Contains(t, sql, "CONSTRAINT `fk_user_profiles_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE")
	}
	assert.Contains(t, up, "PRIMARY KEY (`user_id`)")
	assert.NotContains(t, up, "AUTO_INCREMENT")
	assert.Contains(t, down, "DROP TABLE IF EXISTS `user_profiles`")
}
