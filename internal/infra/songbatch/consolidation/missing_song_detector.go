package consolidation

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	apirepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
)

// missingSong はデータソースに含まれなくなった MySQL 上の楽曲です。
type missingSong struct {
	DisplayID   string `db:"display_id"`
	OfficialIdx string `db:"official_idx"`
	Title       string `db:"title"`
	Artist      string `db:"artist"`
	IsWorldsend int    `db:"is_worldsend"`
}

// findSongsMissingFromSources は未削除の MySQL 楽曲のうち、official と additional_songs のどちらにも含まれない楽曲を返します。
// 統合時はジャンル不明などの楽曲をワークスペースから除外するため、ワークスペースではなく取り込んだままのデータソースの ID で判定します。
// 同期は official_idx で照合するため、official_idx が振り直されると旧 official_idx の行が重複として残ります。
// その行も削除が必要なため、タイトルなどでは除外せず official_idx だけで判定します。
func findSongsMissingFromSources(ctx context.Context, mysql apirepo.Executor, sources ConsolidationSources) ([]missingSong, error) {
	sourceIdx := make(map[string]struct{})
	if sources.Official != nil {
		for _, song := range *sources.Official {
			sourceIdx[song.ID] = struct{}{}
		}
	}
	if sources.AdditionalSongs != nil {
		for _, song := range sources.AdditionalSongs.Songs {
			sourceIdx[strings.TrimSpace(song.ID)] = struct{}{}
		}
	}

	var activeSongs []missingSong
	if err := mysql.SelectContext(ctx, &activeSongs, `SELECT display_id, official_idx, title, artist, is_worldsend FROM songs WHERE is_deleted = 0 ORDER BY official_idx`); err != nil {
		return nil, fmt.Errorf("failed to load active mysql songs: %w", err)
	}

	var missing []missingSong
	for _, song := range activeSongs {
		if _, ok := sourceIdx[song.OfficialIdx]; !ok {
			missing = append(missing, song)
		}
	}
	return missing, nil
}

// logSongsMissingFromSources はデータソースから消えた楽曲を「削除された可能性がある楽曲」として警告ログに出力します。
// データソースの一時的な欠落でも誤って削除しないよう、自動では論理削除せず、判断は管理者に委ねます。
func logSongsMissingFromSources(ctx context.Context, mysql apirepo.Executor, sources ConsolidationSources) error {
	missing, err := findSongsMissingFromSources(ctx, mysql, sources)
	if err != nil {
		return err
	}
	if len(missing) == 0 {
		return nil
	}

	slog.Warn("Detected songs that may have been deleted from datasources", "count", len(missing))
	for _, song := range missing {
		slog.Warn("Song may have been deleted",
			"display_id", song.DisplayID,
			"official_idx", song.OfficialIdx,
			"title", song.Title,
			"artist", song.Artist,
			"is_worldsend", song.IsWorldsend == 1)
	}
	return nil
}
