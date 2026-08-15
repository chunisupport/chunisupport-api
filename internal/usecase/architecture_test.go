package usecase

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUsecaseDoesNotDependOnDTO はAPI表現の変更がユースケース層へ波及しないことを保証します。
func TestUsecaseDoesNotDependOnDTO(t *testing.T) {
	files, err := productionGoFiles(".")
	require.NoError(t, err)

	for _, file := range files {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		require.NoError(t, err)
		for _, imported := range parsed.Imports {
			path, err := strconv.Unquote(imported.Path.Value)
			require.NoError(t, err)
			assert.Falsef(t, strings.Contains(path, "/internal/dto"), "%s がAPI DTOへ依存しています: %s", file, path)
		}

		// astを明示的に参照し、parserが返すimport情報の型をコンパイル時にも固定します。
		assert.IsType(t, []*ast.ImportSpec{}, parsed.Imports)
	}
}

// TestUsecaseStructsDoNotHavePresentationTags はAPIやDBの表現がユースケース境界へ流入しないことを保証します。
func TestUsecaseStructsDoNotHavePresentationTags(t *testing.T) {
	files, err := productionGoFiles(".")
	require.NoError(t, err)

	// 既存の内部JSON処理はARCH-003のAPI DTO依存とは別課題のため、境界型からの分離後に個別移行します。
	legacyInternalJSONFiles := map[string]bool{
		"goal_usecase_impl.go":          true,
		"overpower_record_converter.go": true,
		"record_filter_usecase.go":      true,
	}

	for _, file := range files {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ParseComments)
		require.NoError(t, err)
		ast.Inspect(parsed, func(node ast.Node) bool {
			field, ok := node.(*ast.Field)
			if !ok || field.Tag == nil {
				return true
			}
			tag, err := strconv.Unquote(field.Tag.Value)
			require.NoError(t, err)
			for _, forbidden := range []string{"json", "db"} {
				if forbidden == "json" && legacyInternalJSONFiles[filepath.Base(file)] {
					continue
				}
				assert.NotContainsf(t, tag, forbidden+":", "%s にプレゼンテーションまたはDBタグがあります: %s", file, tag)
			}
			return true
		})
	}
}

func productionGoFiles(root string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
