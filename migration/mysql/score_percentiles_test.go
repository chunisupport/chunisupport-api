package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestScorePercentilesMigration は上位%の境界テーブルと統計変更世代テーブルのマイグレーションを確認します。
func TestScorePercentilesMigration(t *testing.T) {
	// Given
	up := readNormalizedMigrationSQL(t, "000072_create_score_percentiles.up.sql")
	down := readNormalizedMigrationSQL(t, "000072_create_score_percentiles.down.sql")
	schema := readNormalizedMigrationSQL(t, "../schema_mysql.sql")

	// When / Then
	assert.Contains(t, up, "CREATE TABLE chart_stats_revision")
	assert.Contains(t, up, "revision BIGINT UNSIGNED NOT NULL")
	assert.Contains(t, up, "CHECK (id = 1)")
	assert.Contains(t, up, "INSERT INTO chart_stats_revision (id, revision) VALUES (1, 0);")
	assert.Contains(t, up, "CREATE TABLE chart_score_percentiles")
	assert.Contains(t, up, "FOREIGN KEY (chart_id) REFERENCES charts (id) ON DELETE CASCADE")
	assert.Contains(t, up, "CREATE TABLE worldsend_chart_score_percentiles")
	assert.Contains(t, up, "FOREIGN KEY (worldsend_chart_id) REFERENCES worldsend_charts (id) ON DELETE CASCADE")
	assert.Contains(t, up, "CREATE TABLE course_score_percentiles")
	assert.Contains(t, up, "FOREIGN KEY (course_id) REFERENCES courses (id) ON DELETE CASCADE")
	assert.NotContains(t, up, "AUTO_INCREMENT")

	assert.Contains(t, schema, "CREATE TABLE `chart_stats_revision`")
	assert.Contains(t, schema, "CREATE TABLE `chart_score_percentiles`")
	assert.Contains(t, schema, "CREATE TABLE `worldsend_chart_score_percentiles`")
	assert.Contains(t, schema, "CREATE TABLE `course_score_percentiles`")

	assert.Contains(t, down, "DROP TABLE IF EXISTS course_score_percentiles")
	assert.Contains(t, down, "DROP TABLE IF EXISTS worldsend_chart_score_percentiles")
	assert.Contains(t, down, "DROP TABLE IF EXISTS chart_score_percentiles")
	assert.Contains(t, down, "DROP TABLE IF EXISTS chart_stats_revision")
}
