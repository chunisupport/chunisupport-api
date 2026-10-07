package models

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
)

func TestSongModelConvertsNameFolderCode(t *testing.T) {
	model := &SongModel{NameFolderCode: "KA"}
	assert.Equal(t, "KA", model.ToEntity().NameFolderCode)

	song := &entity.Song{NameFolderCode: "SA"}
	assert.Equal(t, "SA", FromSongEntity(song).NameFolderCode)
}
