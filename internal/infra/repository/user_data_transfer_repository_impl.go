package repository

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/goalgroupname"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/jmoiron/sqlx"
)

type userDataTransferRepository struct{ db *sqlx.DB }

func NewUserDataTransferRepository(db *sqlx.DB) domainrepo.UserDataTransferRepository {
	return &userDataTransferRepository{db: db}
}

func (r *userDataTransferRepository) ExportSnapshot(ctx context.Context, userID int) (*entity.UserDataTransferSnapshot, error) {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("failed to begin data transfer export transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	snapshot, err := r.exportSnapshot(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit data transfer export transaction: %w", err)
	}
	return snapshot, nil
}

func (r *userDataTransferRepository) exportSnapshot(ctx context.Context, exec domainrepo.Executor, userID int) (*entity.UserDataTransferSnapshot, error) {
	var playerRow struct {
		ID                  int        `db:"id"`
		Name                string     `db:"player_name"`
		Level               int        `db:"player_level"`
		OfficialRating      float64    `db:"official_player_rating"`
		OfficialOverpower   float64    `db:"official_overpower"`
		ClassEmblemName     *string    `db:"class_emblem_name"`
		ClassEmblemBaseName *string    `db:"class_emblem_base_name"`
		LastPlayedAt        *time.Time `db:"last_played_at"`
		DataCollectedAt     *time.Time `db:"data_collected_at"`
		CreatedAt           time.Time  `db:"created_at"`
	}
	const playerQuery = `SELECT p.id, p.player_name, p.player_level, p.official_player_rating, p.official_overpower,
		ce.name AS class_emblem_name, ceb.name AS class_emblem_base_name, p.last_played_at, p.data_collected_at, p.created_at
		FROM players p INNER JOIN users u ON u.id = p.user_id AND u.player_id = p.id
		LEFT JOIN class_emblems ce ON ce.id = p.class_emblem_id
		LEFT JOIN class_emblem_bases ceb ON ceb.id = p.class_emblem_base_id WHERE u.id = ?`
	if err := exec.GetContext(ctx, &playerRow, playerQuery, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, usecase.ErrDataTransferPlayerNotFound
		}
		return nil, fmt.Errorf("failed to load transfer player: %w", err)
	}
	name, err := playername.NewPlayerName(playerRow.Name)
	if err != nil {
		return nil, fmt.Errorf("invalid transfer player name: %w", err)
	}
	snapshot := &entity.UserDataTransferSnapshot{
		Player: entity.UserDataTransferPlayer{Name: name, Level: playerRow.Level, OfficialRating: playerRow.OfficialRating, OfficialOverpower: playerRow.OfficialOverpower,
			ClassEmblemName: playerRow.ClassEmblemName, ClassEmblemBaseName: playerRow.ClassEmblemBaseName,
			LastPlayedAt: transferUTCOptional(playerRow.LastPlayedAt), DataCollectedAt: transferUTCOptional(playerRow.DataCollectedAt), CreatedAt: transferUTC(playerRow.CreatedAt)},
		Records: []entity.UserDataTransferRecord{}, RecordHistories: []entity.UserDataTransferRecordHistory{},
		WorldsendRecords: []entity.UserDataTransferWorldsendRecord{}, WorldsendRecordHistories: []entity.UserDataTransferWorldsendRecordHistory{},
		MetricHistories: []entity.UserDataTransferMetricHistory{}, CourseRecords: []entity.UserDataTransferCourseRecord{}, Honors: []entity.UserDataTransferHonor{},
		FavoriteSongs: []entity.UserDataTransferFavoriteSong{}, LockedSongs: []entity.UserDataTransferLockedSong{},
		Goals:         entity.UserDataTransferGoals{Groups: []entity.UserDataTransferGoalGroup{}, Ungrouped: []entity.UserDataTransferGoal{}},
		RecordFilters: []entity.UserDataTransferRecordFilter{},
	}
	if err := r.exportRecords(ctx, exec, playerRow.ID, snapshot); err != nil {
		return nil, err
	}
	if err := r.exportAuxiliaryPlayerData(ctx, exec, playerRow.ID, snapshot); err != nil {
		return nil, err
	}
	if err := r.exportGoals(ctx, exec, userID, snapshot); err != nil {
		return nil, err
	}
	if err := r.exportRecordFilters(ctx, exec, userID, snapshot); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (r *userDataTransferRepository) exportRecords(ctx context.Context, exec domainrepo.Executor, playerID int, snapshot *entity.UserDataTransferSnapshot) error {
	var records []struct {
		SongOfficialIdx string    `db:"song_official_idx"`
		Difficulty      string    `db:"difficulty"`
		Score           uint32    `db:"score"`
		ClearLampName   string    `db:"clear_lamp_name"`
		ComboLampName   string    `db:"combo_lamp_name"`
		FullChainName   string    `db:"full_chain_name"`
		SlotName        string    `db:"slot_name"`
		SlotOrder       *int      `db:"slot_order"`
		UpdatedAt       time.Time `db:"updated_at"`
	}
	const recordQuery = `SELECT s.official_idx AS song_official_idx, d.name AS difficulty, pr.score,
		cl.name AS clear_lamp_name, co.name AS combo_lamp_name, fc.name AS full_chain_name, sl.name AS slot_name, pr.slot_order, pr.updated_at
		FROM player_records pr INNER JOIN charts c ON c.id = pr.chart_id INNER JOIN songs s ON s.id = c.song_id
		INNER JOIN difficulties d ON d.id = c.difficulty_id INNER JOIN clear_lamp_types cl ON cl.id = pr.clear_lamp_id
		INNER JOIN combo_lamp_types co ON co.id = pr.combo_lamp_id INNER JOIN full_chain_types fc ON fc.id = pr.full_chain_id
		INNER JOIN slots sl ON sl.id = pr.slot_id WHERE pr.player_id = ? ORDER BY s.official_idx, d.sort_order`
	if err := exec.SelectContext(ctx, &records, recordQuery, playerID); err != nil {
		return fmt.Errorf("failed to export player records: %w", err)
	}
	for _, row := range records {
		score, err := newTransferScore(row.Score)
		if err != nil {
			return err
		}
		snapshot.Records = append(snapshot.Records, entity.UserDataTransferRecord{SongOfficialIdx: row.SongOfficialIdx, Difficulty: row.Difficulty, Score: score,
			ClearLampName: row.ClearLampName, ComboLampName: row.ComboLampName, FullChainName: row.FullChainName, SlotName: row.SlotName, SlotOrder: row.SlotOrder, UpdatedAt: transferUTC(row.UpdatedAt)})
	}
	var histories []struct {
		SongOfficialIdx string    `db:"song_official_idx"`
		Difficulty      string    `db:"difficulty"`
		Score           uint32    `db:"score"`
		ClearLampName   *string   `db:"clear_lamp_name"`
		ComboLampName   *string   `db:"combo_lamp_name"`
		FullChainName   *string   `db:"full_chain_name"`
		UpdatedAt       time.Time `db:"updated_at"`
	}
	const historyQuery = `SELECT s.official_idx AS song_official_idx, d.name AS difficulty, h.score,
		cl.name AS clear_lamp_name, co.name AS combo_lamp_name, fc.name AS full_chain_name, h.updated_at
		FROM player_record_histories h INNER JOIN charts c ON c.id = h.chart_id INNER JOIN songs s ON s.id = c.song_id
		INNER JOIN difficulties d ON d.id = c.difficulty_id LEFT JOIN clear_lamp_types cl ON cl.id = h.clear_lamp_id
		LEFT JOIN combo_lamp_types co ON co.id = h.combo_lamp_id LEFT JOIN full_chain_types fc ON fc.id = h.full_chain_id
		WHERE h.player_id = ? ORDER BY s.official_idx, d.sort_order, h.updated_at`
	if err := exec.SelectContext(ctx, &histories, historyQuery, playerID); err != nil {
		return fmt.Errorf("failed to export record histories: %w", err)
	}
	for _, row := range histories {
		if row.ClearLampName == nil || row.ComboLampName == nil || row.FullChainName == nil {
			return fmt.Errorf("record history contains an unresolved lamp reference")
		}
		score, err := newTransferScore(row.Score)
		if err != nil {
			return err
		}
		snapshot.RecordHistories = append(snapshot.RecordHistories, entity.UserDataTransferRecordHistory{SongOfficialIdx: row.SongOfficialIdx, Difficulty: row.Difficulty, Score: score,
			ClearLampName: *row.ClearLampName, ComboLampName: *row.ComboLampName, FullChainName: *row.FullChainName, UpdatedAt: transferUTC(row.UpdatedAt)})
	}
	return r.exportWorldsendRecords(ctx, exec, playerID, snapshot)
}

func (r *userDataTransferRepository) exportWorldsendRecords(ctx context.Context, exec domainrepo.Executor, playerID int, snapshot *entity.UserDataTransferSnapshot) error {
	var records []struct {
		SongOfficialIdx string    `db:"song_official_idx"`
		Score           uint32    `db:"score"`
		ClearLampName   string    `db:"clear_lamp_name"`
		ComboLampName   string    `db:"combo_lamp_name"`
		FullChainName   string    `db:"full_chain_name"`
		UpdatedAt       time.Time `db:"updated_at"`
	}
	const query = `SELECT s.official_idx AS song_official_idx, pr.score, cl.name AS clear_lamp_name, co.name AS combo_lamp_name, fc.name AS full_chain_name, pr.updated_at
		FROM player_worldsend_records pr INNER JOIN worldsend_charts wc ON wc.id = pr.worldsend_chart_id INNER JOIN songs s ON s.id = wc.song_id
		INNER JOIN clear_lamp_types cl ON cl.id = pr.clear_lamp_id INNER JOIN combo_lamp_types co ON co.id = pr.combo_lamp_id
		INNER JOIN full_chain_types fc ON fc.id = pr.full_chain_id WHERE pr.player_id = ? ORDER BY s.official_idx`
	if err := exec.SelectContext(ctx, &records, query, playerID); err != nil {
		return fmt.Errorf("failed to export worldsend records: %w", err)
	}
	for _, row := range records {
		score, err := newTransferScore(row.Score)
		if err != nil {
			return err
		}
		snapshot.WorldsendRecords = append(snapshot.WorldsendRecords, entity.UserDataTransferWorldsendRecord{SongOfficialIdx: row.SongOfficialIdx, Score: score, ClearLampName: row.ClearLampName, ComboLampName: row.ComboLampName, FullChainName: row.FullChainName, UpdatedAt: transferUTC(row.UpdatedAt)})
	}
	var histories []struct {
		SongOfficialIdx string    `db:"song_official_idx"`
		Score           uint32    `db:"score"`
		ClearLampName   *string   `db:"clear_lamp_name"`
		ComboLampName   *string   `db:"combo_lamp_name"`
		FullChainName   *string   `db:"full_chain_name"`
		UpdatedAt       time.Time `db:"updated_at"`
	}
	const historyQuery = `SELECT s.official_idx AS song_official_idx, h.score, cl.name AS clear_lamp_name, co.name AS combo_lamp_name, fc.name AS full_chain_name, h.updated_at
		FROM player_worldsend_record_histories h INNER JOIN worldsend_charts wc ON wc.id = h.worldsend_chart_id INNER JOIN songs s ON s.id = wc.song_id
		LEFT JOIN clear_lamp_types cl ON cl.id = h.clear_lamp_id LEFT JOIN combo_lamp_types co ON co.id = h.combo_lamp_id LEFT JOIN full_chain_types fc ON fc.id = h.full_chain_id
		WHERE h.player_id = ? ORDER BY s.official_idx, h.updated_at`
	if err := exec.SelectContext(ctx, &histories, historyQuery, playerID); err != nil {
		return fmt.Errorf("failed to export worldsend histories: %w", err)
	}
	for _, row := range histories {
		if row.ClearLampName == nil || row.ComboLampName == nil || row.FullChainName == nil {
			return fmt.Errorf("worldsend history contains an unresolved lamp reference")
		}
		score, err := newTransferScore(row.Score)
		if err != nil {
			return err
		}
		snapshot.WorldsendRecordHistories = append(snapshot.WorldsendRecordHistories, entity.UserDataTransferWorldsendRecordHistory{SongOfficialIdx: row.SongOfficialIdx, Score: score, ClearLampName: *row.ClearLampName, ComboLampName: *row.ComboLampName, FullChainName: *row.FullChainName, UpdatedAt: transferUTC(row.UpdatedAt)})
	}
	return nil
}

func (r *userDataTransferRepository) exportAuxiliaryPlayerData(ctx context.Context, exec domainrepo.Executor, playerID int, snapshot *entity.UserDataTransferSnapshot) error {
	var metrics []struct {
		OfficialRating    float64   `db:"official_rating"`
		OfficialOverpower float64   `db:"official_overpower"`
		DataCollectedAt   time.Time `db:"data_collected_at"`
	}
	if err := exec.SelectContext(ctx, &metrics, `SELECT official_rating, official_overpower, data_collected_at FROM player_metric_histories WHERE player_id = ? ORDER BY data_collected_at`, playerID); err != nil {
		return err
	}
	for _, row := range metrics {
		snapshot.MetricHistories = append(snapshot.MetricHistories, entity.UserDataTransferMetricHistory{OfficialRating: row.OfficialRating, OfficialOverpower: row.OfficialOverpower, DataCollectedAt: transferUTC(row.DataCollectedAt)})
	}
	var courses []struct {
		OfficialIdx   string    `db:"official_idx"`
		Score         uint32    `db:"score"`
		IsClear       bool      `db:"is_clear"`
		ComboLampName string    `db:"combo_lamp_name"`
		UpdatedAt     time.Time `db:"updated_at"`
	}
	if err := exec.SelectContext(ctx, &courses, `SELECT c.official_idx, pr.score, pr.is_clear, cl.name AS combo_lamp_name, pr.updated_at FROM player_course_records pr INNER JOIN courses c ON c.id = pr.course_id INNER JOIN combo_lamp_types cl ON cl.id = pr.combo_lamp_id WHERE pr.player_id = ? ORDER BY c.official_idx`, playerID); err != nil {
		return err
	}
	for _, row := range courses {
		score, err := newTransferCourseScore(row.Score)
		if err != nil {
			return err
		}
		snapshot.CourseRecords = append(snapshot.CourseRecords, entity.UserDataTransferCourseRecord{CourseOfficialIdx: row.OfficialIdx, Score: score, IsClear: row.IsClear, ComboLampName: row.ComboLampName, UpdatedAt: transferUTC(row.UpdatedAt)})
	}
	var honors []struct {
		Slot       int       `db:"slot"`
		ImageURL   *string   `db:"image_url"`
		Name       string    `db:"name"`
		TypeName   string    `db:"type_name"`
		EquippedAt time.Time `db:"equipped_at"`
	}
	if err := exec.SelectContext(ctx, &honors, `SELECT ph.slot, NULLIF(h.image_url, '') AS image_url, h.name, ht.name AS type_name, ph.created_at AS equipped_at FROM player_honors ph INNER JOIN honors h ON h.id = ph.honor_id INNER JOIN honor_types ht ON ht.id = h.honor_type_id WHERE ph.player_id = ? ORDER BY ph.slot`, playerID); err != nil {
		return err
	}
	for _, row := range honors {
		snapshot.Honors = append(snapshot.Honors, entity.UserDataTransferHonor{Slot: row.Slot, ImageURL: row.ImageURL, Name: row.Name, TypeName: row.TypeName, EquippedAt: transferUTC(row.EquippedAt)})
	}
	var favorites []struct {
		OfficialIdx string    `db:"official_idx"`
		FavoritedAt time.Time `db:"favorited_at"`
	}
	if err := exec.SelectContext(ctx, &favorites, `SELECT s.official_idx, f.created_at AS favorited_at FROM player_favorite_songs f INNER JOIN songs s ON s.id = f.song_id WHERE f.player_id = ? ORDER BY s.official_idx`, playerID); err != nil {
		return err
	}
	for _, row := range favorites {
		snapshot.FavoriteSongs = append(snapshot.FavoriteSongs, entity.UserDataTransferFavoriteSong{SongOfficialIdx: row.OfficialIdx, FavoritedAt: transferUTC(row.FavoritedAt)})
	}
	var locked []struct {
		OfficialIdx string `db:"official_idx"`
		IsUltima    bool   `db:"is_ultima"`
	}
	if err := exec.SelectContext(ctx, &locked, `SELECT s.official_idx, l.is_ultima FROM player_locked_songs l INNER JOIN songs s ON s.id = l.song_id WHERE l.player_id = ? ORDER BY s.official_idx, l.is_ultima`, playerID); err != nil {
		return err
	}
	for _, row := range locked {
		snapshot.LockedSongs = append(snapshot.LockedSongs, entity.UserDataTransferLockedSong{SongOfficialIdx: row.OfficialIdx, IsUltima: row.IsUltima})
	}
	return nil
}

func (r *userDataTransferRepository) exportGoals(ctx context.Context, exec domainrepo.Executor, userID int, snapshot *entity.UserDataTransferSnapshot) error {
	masters, err := loadTransferMasterData(ctx, exec)
	if err != nil {
		return err
	}
	var groups []struct {
		ID        uint32    `db:"id"`
		Name      string    `db:"name"`
		SortOrder uint16    `db:"sort_order"`
		CreatedAt time.Time `db:"created_at"`
	}
	if err := exec.SelectContext(ctx, &groups, `SELECT id, name, sort_order, created_at FROM goal_groups WHERE user_id = ? ORDER BY sort_order, id`, userID); err != nil {
		return err
	}
	indexes := make(map[uint32]int, len(groups))
	for _, row := range groups {
		name, err := goalgroupname.NewGoalGroupName(row.Name)
		if err != nil {
			return err
		}
		indexes[row.ID] = len(snapshot.Goals.Groups)
		snapshot.Goals.Groups = append(snapshot.Goals.Groups, entity.UserDataTransferGoalGroup{Name: name, SortOrder: row.SortOrder, CreatedAt: transferUTC(row.CreatedAt), Goals: []entity.UserDataTransferGoal{}})
	}
	var goals []struct {
		GroupID           *uint32   `db:"group_id"`
		Title             string    `db:"title"`
		AchievementType   string    `db:"achievement_type"`
		AchievementParams []byte    `db:"achievement_params"`
		Attributes        []byte    `db:"attributes"`
		InvertValue       bool      `db:"invert_value"`
		InvertPercentage  bool      `db:"invert_percentage"`
		SortOrder         uint16    `db:"sort_order"`
		CreatedAt         time.Time `db:"created_at"`
	}
	if err := exec.SelectContext(ctx, &goals, `SELECT g.group_id, g.title, at.code AS achievement_type, g.achievement_params, g.attributes, g.invert_value, g.invert_percentage, g.sort_order, g.created_at FROM goals g INNER JOIN achievement_types at ON at.id = g.achievement_type_id LEFT JOIN goal_groups gg ON gg.id = g.group_id AND gg.user_id = g.user_id WHERE g.user_id = ? ORDER BY (g.group_id IS NULL), gg.sort_order, g.sort_order, g.id`, userID); err != nil {
		return err
	}
	for _, row := range goals {
		attributes, err := masters.externalizeGoalAttributes(row.Attributes)
		if err != nil {
			return err
		}
		goal := entity.UserDataTransferGoal{Title: row.Title, AchievementType: row.AchievementType, AchievementParams: append(json.RawMessage(nil), row.AchievementParams...), Attributes: attributes, InvertValue: row.InvertValue, InvertPercentage: row.InvertPercentage, SortOrder: row.SortOrder, CreatedAt: transferUTC(row.CreatedAt)}
		if row.GroupID == nil {
			snapshot.Goals.Ungrouped = append(snapshot.Goals.Ungrouped, goal)
			continue
		}
		index, ok := indexes[*row.GroupID]
		if !ok {
			return fmt.Errorf("goal references an unknown group")
		}
		snapshot.Goals.Groups[index].Goals = append(snapshot.Goals.Groups[index].Goals, goal)
	}
	return nil
}

func (r *userDataTransferRepository) exportRecordFilters(ctx context.Context, exec domainrepo.Executor, userID int, snapshot *entity.UserDataTransferSnapshot) error {
	var rows []struct {
		Name        string    `db:"name"`
		Value       []byte    `db:"filter_value_gzip"`
		IsWorldsend bool      `db:"is_worldsend"`
		CreatedAt   time.Time `db:"created_at"`
		UpdatedAt   time.Time `db:"updated_at"`
	}
	if err := exec.SelectContext(ctx, &rows, `SELECT name, filter_value_gzip, is_worldsend, created_at, updated_at FROM record_filters WHERE user_id = ? ORDER BY updated_at, id`, userID); err != nil {
		return err
	}
	for _, row := range rows {
		raw, err := gunzipTransferFilter(row.Value)
		if err != nil {
			return err
		}
		var payload struct {
			SchemaVersion int             `json:"schema_version"`
			Filter        json.RawMessage `json:"filter"`
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			return err
		}
		filterType := usecase.RecordFilterTypeStandard
		if row.IsWorldsend {
			filterType = usecase.RecordFilterTypeWorldsend
		}
		snapshot.RecordFilters = append(snapshot.RecordFilters, entity.UserDataTransferRecordFilter{Name: row.Name, FilterType: filterType, SchemaVersion: payload.SchemaVersion, Filter: payload.Filter, CreatedAt: transferUTC(row.CreatedAt), UpdatedAt: transferUTC(row.UpdatedAt)})
	}
	return nil
}

func transferUTC(value time.Time) time.Time { return value.UTC().Truncate(time.Second) }
func transferUTCOptional(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	normalized := transferUTC(*value)
	return &normalized
}

func gunzipTransferFilter(value []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(value))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	raw, err := io.ReadAll(io.LimitReader(reader, int64(info.RecordFilterMaxPayloadBytes)+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > info.RecordFilterMaxPayloadBytes {
		return nil, fmt.Errorf("record filter exceeds size limit")
	}
	return raw, nil
}

var _ domainrepo.UserDataTransferRepository = (*userDataTransferRepository)(nil)
