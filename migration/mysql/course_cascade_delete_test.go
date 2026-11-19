package mysql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAddCourseCascadeDeleteMigration はコース物理削除時にスコアとスコア履歴も連鎖削除される外部キー制約へ更新することを確認します。
func TestAddCourseCascadeDeleteMigration(t *testing.T) {
	// Given
	up := readNormalizedMigrationSQL(t, "000069_add_course_cascade_delete.up.sql")
	down := readNormalizedMigrationSQL(t, "000069_add_course_cascade_delete.down.sql")
	schema := readNormalizedMigrationSQL(t, "../schema_mysql.sql")

	// When / Then: up で既存制約を削除してから ON DELETE CASCADE 付きで再作成する
	assert.Contains(t, up, "ALTER TABLE player_course_records DROP FOREIGN KEY fk_player_course_records_course;")
	assert.Contains(t, up, "ALTER TABLE player_course_records ADD CONSTRAINT fk_player_course_records_course FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE;")
	assert.Less(
		t,
		strings.Index(up, "ALTER TABLE player_course_records DROP FOREIGN KEY fk_player_course_records_course;"),
		strings.Index(up, "ALTER TABLE player_course_records ADD CONSTRAINT fk_player_course_records_course"),
	)

	assert.Contains(t, up, "ALTER TABLE player_course_record_histories DROP FOREIGN KEY fk_player_course_record_histories_course;")
	assert.Contains(t, up, "ALTER TABLE player_course_record_histories ADD CONSTRAINT fk_player_course_record_histories_course FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE;")
	assert.Less(
		t,
		strings.Index(up, "ALTER TABLE player_course_record_histories DROP FOREIGN KEY fk_player_course_record_histories_course;"),
		strings.Index(up, "ALTER TABLE player_course_record_histories ADD CONSTRAINT fk_player_course_record_histories_course"),
	)

	// When / Then: down で ON DELETE 指定なしの外部キー制約へ戻す
	assert.Contains(t, down, "ALTER TABLE player_course_record_histories DROP FOREIGN KEY fk_player_course_record_histories_course;")
	assert.Contains(t, down, "ALTER TABLE player_course_record_histories ADD CONSTRAINT fk_player_course_record_histories_course FOREIGN KEY (course_id) REFERENCES courses (id);")
	assert.Less(
		t,
		strings.Index(down, "ALTER TABLE player_course_record_histories DROP FOREIGN KEY fk_player_course_record_histories_course;"),
		strings.Index(down, "ALTER TABLE player_course_record_histories ADD CONSTRAINT fk_player_course_record_histories_course"),
	)

	assert.Contains(t, down, "ALTER TABLE player_course_records DROP FOREIGN KEY fk_player_course_records_course;")
	assert.Contains(t, down, "ALTER TABLE player_course_records ADD CONSTRAINT fk_player_course_records_course FOREIGN KEY (course_id) REFERENCES courses (id);")
	assert.Less(
		t,
		strings.Index(down, "ALTER TABLE player_course_records DROP FOREIGN KEY fk_player_course_records_course;"),
		strings.Index(down, "ALTER TABLE player_course_records ADD CONSTRAINT fk_player_course_records_course"),
	)

	// When / Then: schema_mysql.sql に ON DELETE CASCADE が反映されている
	assert.Contains(t, schema, "CONSTRAINT `fk_player_course_record_histories_course` FOREIGN KEY (`course_id`) REFERENCES `courses` (`id`) ON DELETE CASCADE")
	assert.Contains(t, schema, "CONSTRAINT `fk_player_course_records_course` FOREIGN KEY (`course_id`) REFERENCES `courses` (`id`) ON DELETE CASCADE")
}
