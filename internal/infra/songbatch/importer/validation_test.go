package importer

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeImportFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "source.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestOfficialImporterValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		valid   bool
	}{
		{name: "valid", content: `[{"id":" 1 ","title":" Song "}]`, valid: true},
		{name: "empty", content: `[]`},
		{name: "null", content: `null`},
		{name: "missing id", content: `[{"title":"Song"}]`},
		{name: "blank title", content: `[{"id":"1","title":" "}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			path := writeImportFixture(t, tt.content)

			// When
			src, err := NewOfficialImporter().Import(path)

			// Then
			assertValidationResult(t, err, tt.valid)
			if !tt.valid {
				return
			}
			// 成功戻り値の型と正規化値を捨てずに検証します。
			require.NotNil(t, src)
			assert.Equal(t, songbatch.DataSourceOfficial, src.Type)
			data, ok := src.Data.(*OfficialData)
			require.True(t, ok, "Dataは*OfficialDataであること")
			require.NotNil(t, data)
			require.Len(t, *data, 1)
			assert.Equal(t, "1", (*data)[0].ID)
			assert.Equal(t, "Song", (*data)[0].Title)
		})
	}
}

func TestMainframeImporterValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		valid   bool
	}{
		{name: "valid", content: `[{"title":" Song ","diff":" MAS ","const":14.2}]`, valid: true},
		{name: "empty", content: `[]`},
		{name: "null", content: `null`},
		{name: "missing title", content: `[{"diff":"MAS","const":14.2}]`},
		{name: "unsupported difficulty", content: `[{"title":"Song","diff":"WORLD'S END","const":14.2}]`},
		{name: "non-positive const", content: `[{"title":"Song","diff":"MAS","const":0}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			path := writeImportFixture(t, tt.content)

			// When
			src, err := NewMainframeImporter().Import(path)

			// Then
			assertValidationResult(t, err, tt.valid)
			if !tt.valid {
				return
			}
			// 成功戻り値の型と正規化値を捨てずに検証します。
			require.NotNil(t, src)
			assert.Equal(t, songbatch.DataSourceMainframe, src.Type)
			data, ok := src.Data.(*MainframeData)
			require.True(t, ok, "Dataは*MainframeDataであること")
			require.NotNil(t, data)
			require.Len(t, *data, 1)
			assert.Equal(t, "Song", (*data)[0].Title)
			assert.Equal(t, "MAS", (*data)[0].Diff)
			assert.Equal(t, 14.2, (*data)[0].Const)
		})
	}
}

func TestSt1027ImporterValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		valid   bool
	}{
		{name: "valid", content: `{"songs":[{"meta":{"official_id":"1"}}]}`, valid: true},
		{name: "missing songs", content: `{}`},
		{name: "null songs", content: `{"songs":null}`},
		{name: "empty songs", content: `{"songs":[]}`},
		{name: "missing official id", content: `{"songs":[{"meta":{}}]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			path := writeImportFixture(t, tt.content)

			// When
			src, err := NewSt1027Importer().Import(path)

			// Then
			assertValidationResult(t, err, tt.valid)
			if !tt.valid {
				return
			}
			// 成功戻り値の型と値を捨てずに検証します。
			require.NotNil(t, src)
			assert.Equal(t, songbatch.DataSourceSt1027, src.Type)
			data, ok := src.Data.(*St1027Data)
			require.True(t, ok, "Dataは*St1027Dataであること")
			require.NotNil(t, data)
			require.Len(t, data.Songs, 1)
			assert.Equal(t, "1", data.Songs[0].Meta.OfficialID)
		})
	}
}

func TestOtogeDbImporterValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		valid   bool
	}{
		{name: "valid", content: `[{"id":" 1 ","title":" Song "}]`, valid: true},
		{name: "empty", content: `[]`},
		{name: "null", content: `null`},
		{name: "non-numeric id", content: `[{"id":"x","title":"Song"}]`},
		{name: "non-string id", content: `[{"id":1,"title":"Song"}]`},
		{name: "non-positive id", content: `[{"id":"0","title":"Song"}]`},
		{name: "missing title", content: `[{"id":"1"}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			path := writeImportFixture(t, tt.content)

			// When
			src, err := NewOtogeDbImporter().Import(path)

			// Then
			assertValidationResult(t, err, tt.valid)
			if !tt.valid {
				return
			}
			// 成功戻り値の型と正規化値を捨てずに検証します。
			require.NotNil(t, src)
			assert.Equal(t, songbatch.DataSourceOtogeDb, src.Type)
			data, ok := src.Data.(*OtogeDbData)
			require.True(t, ok, "Dataは*OtogeDbDataであること")
			require.NotNil(t, data)
			require.Len(t, *data, 1)
			assert.Equal(t, "1", (*data)[0].ID)
			assert.Equal(t, "Song", (*data)[0].Title)
		})
	}
}

func TestAdditionalSongsImporterValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		valid   bool
	}{
		{name: "empty arrays are valid", content: `{"songs":[],"charts":[],"we_charts":[],"courses":[]}`, valid: true},
		{name: "missing array", content: `{"songs":[],"charts":[],"we_charts":[]}`},
		{name: "null array", content: `{"songs":[],"charts":[],"we_charts":[],"courses":null}`},
		{name: "missing songs", content: `{"charts":[],"we_charts":[],"courses":[]}`},
		{name: "missing charts", content: `{"songs":[],"we_charts":[],"courses":[]}`},
		{name: "missing we_charts", content: `{"songs":[],"charts":[],"courses":[]}`},
		{name: "null songs", content: `{"songs":null,"charts":[],"we_charts":[],"courses":[]}`},
		{name: "null charts", content: `{"songs":[],"charts":null,"we_charts":[],"courses":[]}`},
		{name: "null we_charts", content: `{"songs":[],"charts":[],"we_charts":null,"courses":[]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			path := writeImportFixture(t, tt.content)

			// When
			src, err := NewAdditionalSongsImporter().Import(path)

			// Then
			assertValidationResult(t, err, tt.valid)
			if !tt.valid {
				return
			}
			// 成功戻り値の型と値を捨てずに検証します。全配列空は有効のままです。
			require.NotNil(t, src)
			assert.Equal(t, songbatch.DataSourceAdditionalSongs, src.Type)
			data, ok := src.Data.(*AdditionalSongsData)
			require.True(t, ok, "Dataは*AdditionalSongsDataであること")
			require.NotNil(t, data)
			assert.Empty(t, data.Songs)
			assert.Empty(t, data.Charts)
			assert.Empty(t, data.WECharts)
			assert.Empty(t, data.Courses)
			assert.NotNil(t, data.Songs)
			assert.NotNil(t, data.Charts)
			assert.NotNil(t, data.WECharts)
			assert.NotNil(t, data.Courses)
		})
	}
}

func assertValidationResult(t *testing.T, err error, valid bool) {
	t.Helper()
	if valid {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}
	if !errors.Is(err, songbatch.ErrSourceValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
