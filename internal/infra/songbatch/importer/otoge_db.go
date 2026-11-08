package importer

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chunisupport/chunisupport-api/internal/domain/songbatch"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// OtogeDbImporter はotoge-dbデータソースによって生成されたデータをロードします
type OtogeDbImporter struct{}

// NewOtogeDbImporter はOtogeDbImporterのインスタンスを返します
func NewOtogeDbImporter() *OtogeDbImporter {
	return &OtogeDbImporter{}
}

// Import はotoge-dbのJSONファイルを読み込み、ImportResultを生成します
func (oi *OtogeDbImporter) Import(filePath string) (*songbatch.ImportedSource, error) {
	slog.Info("Starting otoge-db data import", "file", filePath)

	if _, err := os.Stat(filePath); errors.Is(err, fs.ErrNotExist) {
		slog.Warn("Otoge-db datasource file not found, skipping import", "path", filePath)
		return &songbatch.ImportedSource{Type: songbatch.DataSourceOtogeDb, Data: nil}, nil
	}

	data, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to read otoge-db data file %s: %w", filePath, err)
	}

	data = removeBOM(data)

	otogeDbData, discarded, err := decodeOtogeDbData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal otoge-db JSON: %w", err)
	}
	if len(discarded) > 0 {
		// 楽曲ごとに出すと通知が大量になるため、1回の取り込みにつき1件にまとめます。
		slog.Warn("Discarded otoge-db fields with unexpected types", "count", len(discarded), "fields", strings.Join(discarded, ","))
	}

	for i := range otogeDbData {
		otogeDbData[i].ID = strings.TrimSpace(otogeDbData[i].ID)
		otogeDbData[i].Title = strings.TrimSpace(otogeDbData[i].Title)
		otogeDbData[i].DateAdded = strings.TrimSpace(otogeDbData[i].DateAdded)
		otogeDbData[i].Version = strings.TrimSpace(otogeDbData[i].Version)
		otogeDbData[i].Catname = strings.TrimSpace(otogeDbData[i].Catname)
		otogeDbData[i].Artist = strings.TrimSpace(otogeDbData[i].Artist)
		otogeDbData[i].WikiwikiURL = strings.TrimSpace(otogeDbData[i].WikiwikiURL)
		id, err := strconv.Atoi(otogeDbData[i].ID)
		if err != nil || id <= 0 || otogeDbData[i].Title == "" {
			return nil, fmt.Errorf("%w: otoge_db row %d requires a positive numeric id and title", songbatch.ErrSourceValidation, i+1)
		}
	}
	if len(otogeDbData) == 0 {
		return nil, fmt.Errorf("%w: otoge_db must contain at least one song", songbatch.ErrSourceValidation)
	}

	slog.Info("Successfully loaded otoge-db song data", "count", len(otogeDbData))

	return &songbatch.ImportedSource{
		Type: songbatch.DataSourceOtogeDb,
		Data: &otogeDbData,
	}, nil
}

// decodeOtogeDbData は楽曲ごと・フィールドごとに文字列として読み込みます。
// otoge-db では特定の楽曲のフィールドだけ型が崩れることがあります。
// 誤ったデータを解釈して取り込まないよう、型の合わない値はそのフィールドだけ捨て、
// データソース全体を失わないようにします。捨てたフィールドは「楽曲ID:キー」で返します。
func decodeOtogeDbData(data []byte) (OtogeDbData, []string, error) {
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, nil, err
	}

	songs := make(OtogeDbData, len(rows))
	var discarded []string
	for i, row := range rows {
		// jsonFields は id を先頭に返すため、他のフィールドを捨てる時点で ID は確定しています。
		// ID 自体を捨てた場合は後段の検証で行番号つきのエラーになります。
		for _, field := range songs[i].jsonFields() {
			value, ok := row[field.key]
			if !ok {
				continue
			}
			if err := json.Unmarshal(value, field.dst); err != nil {
				discarded = append(discarded, strings.TrimSpace(songs[i].ID)+":"+field.key)
			}
		}
	}
	return songs, discarded, nil
}
