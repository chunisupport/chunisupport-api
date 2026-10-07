package masterdata

import (
	"context"
	"testing"

	domainmasterdata "github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreload_NameFolders(t *testing.T) {
	db := setupPreloadSQLite(t)
	insertPreloadMasterRows(t, db, 1, "rank_count")
	_, err := db.Exec(`INSERT INTO name_folders (id, code, name, sort_order) VALUES (8, 'KA', 'か行', 8)`)
	require.NoError(t, err)
	cache, err := Preload(context.Background(), db)
	require.NoError(t, err)
	want := domainmasterdata.NameFolder{ID: 8, Code: "KA", Name: "か行", SortOrder: 8}
	snapshot := cache.MasterDataMasters()
	assert.Equal(t, want, snapshot.NameFolders["KA"])
	delete(snapshot.NameFolders, "KA")
	assert.Equal(t, want, cache.MasterDataMasters().NameFolders["KA"])
}
