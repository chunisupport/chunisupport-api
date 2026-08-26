package repository

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"
	"uuid"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	domainrepo "github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/coursescore"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/score"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

func (r *userDataTransferRepository) FindUnresolvedReferences(ctx context.Context, snapshot *entity.UserDataTransferSnapshot) ([]string, error) {
	return r.findUnresolvedReferences(ctx, r.db, snapshot)
}

func (r *userDataTransferRepository) findUnresolvedReferences(ctx context.Context, exec domainrepo.Executor, snapshot *entity.UserDataTransferSnapshot) ([]string, error) {
	masters, err := loadTransferMasterData(ctx, exec)
	if err != nil {
		return nil, fmt.Errorf("failed to load transfer master data: %w", err)
	}
	return findTransferUnresolvedReferences(snapshot, masters), nil
}

func findTransferUnresolvedReferences(snapshot *entity.UserDataTransferSnapshot, masters *transferMasterData) []string {
	unresolved := make(map[string]struct{})
	add := func(kind, value string, exists bool) {
		if !exists {
			unresolved[kind+":"+value] = struct{}{}
		}
	}
	if snapshot.Player.ClassEmblemName != nil {
		_, ok := masters.classEmblemIDs[*snapshot.Player.ClassEmblemName]
		add("class_emblem", *snapshot.Player.ClassEmblemName, ok)
	}
	if snapshot.Player.ClassEmblemBaseName != nil {
		_, ok := masters.classEmblemBaseIDs[*snapshot.Player.ClassEmblemBaseName]
		add("class_emblem_base", *snapshot.Player.ClassEmblemBaseName, ok)
	}
	for _, record := range snapshot.Records {
		_, ok := masters.charts[transferChartKey(record.SongOfficialIdx, record.Difficulty)]
		add("chart", record.SongOfficialIdx+"/"+record.Difficulty, ok)
		checkTransferRecordMasters(unresolved, masters, record.ClearLampName, record.ComboLampName, record.FullChainName)
		_, ok = masters.slotIDs[record.SlotName]
		add("slot", record.SlotName, ok)
	}
	for _, history := range snapshot.RecordHistories {
		_, ok := masters.charts[transferChartKey(history.SongOfficialIdx, history.Difficulty)]
		add("chart", history.SongOfficialIdx+"/"+history.Difficulty, ok)
		checkTransferRecordMasters(unresolved, masters, history.ClearLampName, history.ComboLampName, history.FullChainName)
	}
	for _, record := range snapshot.WorldsendRecords {
		_, ok := masters.worldsendChartIDs[record.SongOfficialIdx]
		add("worldsend_chart", record.SongOfficialIdx, ok)
		checkTransferRecordMasters(unresolved, masters, record.ClearLampName, record.ComboLampName, record.FullChainName)
	}
	for _, history := range snapshot.WorldsendRecordHistories {
		_, ok := masters.worldsendChartIDs[history.SongOfficialIdx]
		add("worldsend_chart", history.SongOfficialIdx, ok)
		checkTransferRecordMasters(unresolved, masters, history.ClearLampName, history.ComboLampName, history.FullChainName)
	}
	for _, course := range snapshot.CourseRecords {
		_, ok := masters.courseIDs[course.CourseOfficialIdx]
		add("course", course.CourseOfficialIdx, ok)
		_, ok = masters.comboLampIDs[course.ComboLampName]
		add("combo_lamp", course.ComboLampName, ok)
	}
	for _, honor := range snapshot.Honors {
		_, ok := masters.honorTypeIDs[honor.TypeName]
		add("honor_type", honor.TypeName, ok)
	}
	for _, favorite := range snapshot.FavoriteSongs {
		_, ok := masters.songIDs[favorite.SongOfficialIdx]
		add("song", favorite.SongOfficialIdx, ok)
	}
	for _, locked := range snapshot.LockedSongs {
		_, ok := masters.songIDs[locked.SongOfficialIdx]
		add("song", locked.SongOfficialIdx, ok)
	}
	checkGoal := func(goal entity.UserDataTransferGoal) {
		_, ok := masters.achievementTypeIDs[goal.AchievementType]
		add("achievement_type", goal.AchievementType, ok)
		if _, err := masters.internalizeGoalAttributes(goal.Attributes); err != nil {
			unresolved["goal_attributes:"+goal.Title] = struct{}{}
		}
	}
	for _, goal := range snapshot.Goals.Ungrouped {
		checkGoal(goal)
	}
	for _, group := range snapshot.Goals.Groups {
		for _, goal := range group.Goals {
			checkGoal(goal)
		}
	}
	result := make([]string, 0, len(unresolved))
	for value := range unresolved {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func checkTransferRecordMasters(unresolved map[string]struct{}, masters *transferMasterData, clear, combo, fullChain string) {
	if _, ok := masters.clearLampIDs[clear]; !ok {
		unresolved["clear_lamp:"+clear] = struct{}{}
	}
	if _, ok := masters.comboLampIDs[combo]; !ok {
		unresolved["combo_lamp:"+combo] = struct{}{}
	}
	if _, ok := masters.fullChainIDs[fullChain]; !ok {
		unresolved["full_chain:"+fullChain] = struct{}{}
	}
}

func (r *userDataTransferRepository) IsDestinationEmpty(ctx context.Context, userID int) (bool, error) {
	return r.isDestinationEmpty(ctx, r.db, userID, false)
}

func (r *userDataTransferRepository) isDestinationEmpty(ctx context.Context, exec domainrepo.Executor, userID int, lock bool) (bool, error) {
	query := `SELECT player_id,
		(SELECT COUNT(*) FROM players WHERE user_id = users.id) AS player_count,
		(SELECT COUNT(*) FROM goals WHERE user_id = users.id) AS goal_count,
		(SELECT COUNT(*) FROM goal_groups WHERE user_id = users.id) AS group_count,
		(SELECT COUNT(*) FROM record_filters WHERE user_id = users.id) AS filter_count
		FROM users WHERE id = ?`
	if lock && r.db.DriverName() != "sqlite" {
		query += ` FOR UPDATE`
	}
	var row struct {
		PlayerID    *int `db:"player_id"`
		PlayerCount int  `db:"player_count"`
		GoalCount   int  `db:"goal_count"`
		GroupCount  int  `db:"group_count"`
		FilterCount int  `db:"filter_count"`
	}
	if err := exec.GetContext(ctx, &row, query, userID); err != nil {
		return false, fmt.Errorf("failed to inspect transfer destination: %w", err)
	}
	return row.PlayerID == nil && row.PlayerCount == 0 && row.GoalCount == 0 && row.GroupCount == 0 && row.FilterCount == 0, nil
}

func (r *userDataTransferRepository) ImportSnapshot(ctx context.Context, userID int, snapshot *entity.UserDataTransferSnapshot) (playerID int, err error) {
	if snapshot == nil {
		return 0, usecase.ErrInternalError
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin data transfer import transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
			return
		}
		err = tx.Commit()
	}()
	empty, err := r.isDestinationEmpty(ctx, tx, userID, true)
	if err != nil {
		return 0, err
	}
	if !empty {
		return 0, usecase.ErrDataTransferDestinationNotEmpty
	}
	if unresolved, resolveErr := r.findUnresolvedReferences(ctx, tx, snapshot); resolveErr != nil {
		return 0, resolveErr
	} else if len(unresolved) > 0 {
		return 0, usecase.ErrDataTransferUnresolvedReference
	}
	return r.importSnapshot(ctx, tx, userID, snapshot)
}

func (r *userDataTransferRepository) importSnapshot(ctx context.Context, exec domainrepo.Executor, userID int, snapshot *entity.UserDataTransferSnapshot) (int, error) {
	if exec == nil || snapshot == nil {
		return 0, usecase.ErrInternalError
	}
	masters, err := loadTransferMasterData(ctx, exec)
	if err != nil {
		return 0, err
	}
	if unresolved := findTransferUnresolvedReferences(snapshot, masters); len(unresolved) > 0 {
		return 0, usecase.ErrDataTransferUnresolvedReference
	}
	calculatedRating, bestAverage, newAverage, overpowerValue := calculateTransferDerivedMetrics(snapshot, masters)
	classEmblemID, err := optionalTransferMasterID(snapshot.Player.ClassEmblemName, masters.classEmblemIDs)
	if err != nil {
		return 0, err
	}
	classEmblemBaseID, err := optionalTransferMasterID(snapshot.Player.ClassEmblemBaseName, masters.classEmblemBaseIDs)
	if err != nil {
		return 0, err
	}
	result, err := exec.ExecContext(ctx, `INSERT INTO players
		(user_id, player_name, player_level, official_player_rating, calculated_player_rating, new_average_rating, best_average_rating,
		 class_emblem_id, class_emblem_base_id, last_played_at, overpower_value, official_overpower, official_overpower_percent, data_collected_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		userID, snapshot.Player.Name.String(), snapshot.Player.Level, snapshot.Player.OfficialRating, calculatedRating, newAverage, bestAverage,
		classEmblemID, classEmblemBaseID, snapshot.Player.LastPlayedAt, overpowerValue, snapshot.Player.OfficialOverpower, snapshot.Player.OfficialOverpowerPercent, snapshot.Player.DataCollectedAt, snapshot.Player.CreatedAt)
	if err != nil {
		return 0, fmt.Errorf("failed to insert transfer player: %w", err)
	}
	playerID64, err := result.LastInsertId()
	if err != nil || playerID64 <= 0 {
		return 0, fmt.Errorf("failed to get transfer player ID: %w", err)
	}
	playerID := int(playerID64)
	if err := saveTransferRecords(ctx, exec, playerID, snapshot, masters); err != nil {
		return 0, err
	}
	if err := saveTransferAuxiliaryData(ctx, exec, playerID, snapshot, masters, &honorRepository{db: r.db}); err != nil {
		return 0, err
	}
	if err := saveTransferGoals(ctx, exec, userID, snapshot, masters); err != nil {
		return 0, err
	}
	if err := saveTransferRecordFilters(ctx, exec, userID, snapshot); err != nil {
		return 0, err
	}
	update, err := exec.ExecContext(ctx, `UPDATE users SET player_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND player_id IS NULL`, playerID, userID)
	if err != nil {
		return 0, err
	}
	affected, err := update.RowsAffected()
	if err != nil {
		return 0, err
	}
	if affected != 1 {
		return 0, usecase.ErrDataTransferDestinationNotEmpty
	}
	return playerID, nil
}

func calculateTransferDerivedMetrics(snapshot *entity.UserDataTransferSnapshot, masters *transferMasterData) (float64, float64, float64, float64) {
	best := make([]service.RatingSlotRecord, 0, 30)
	newest := make([]service.RatingSlotRecord, 0, 20)
	overpower := make([]service.OverpowerRecord, 0, len(snapshot.Records))
	locked := make(map[string]struct{}, len(snapshot.LockedSongs))
	for _, item := range snapshot.LockedSongs {
		locked[item.SongOfficialIdx+"\x00"+fmt.Sprintf("%t", item.IsUltima)] = struct{}{}
	}
	for _, record := range snapshot.Records {
		chart := masters.charts[transferChartKey(record.SongOfficialIdx, record.Difficulty)]
		rating := service.RatingSlotRecord{ChartID: chart.ID, Score: uint32(record.Score), ChartConst: chart.ChartConst, OfficialIndex: parseTransferOfficialIndex(record.SongOfficialIdx)}
		if !chart.IsDeleted {
			switch record.SlotName {
			case "best":
				best = append(best, rating)
			case "new":
				newest = append(newest, rating)
			}
		}
		_, songLocked := locked[record.SongOfficialIdx+"\x00false"]
		_, ultimaLocked := locked[record.SongOfficialIdx+"\x00true"]
		if !chart.IsDeleted && !songLocked && !(record.Difficulty == info.DifficultyNameUltima && ultimaLocked) {
			overpower = append(overpower, service.OverpowerRecord{SongID: chart.SongID, Score: uint32(record.Score), ChartConst: chart.ChartConst, ComboLampID: masters.comboLampIDs[record.ComboLampName]})
		}
	}
	stats := service.AggregateOfficialRating(best, newest)
	op, _ := service.CalcOverpowerSummary(overpower, 0)
	return stats.PlayerRating, stats.BestAverage, stats.NewAverage, op
}

func saveTransferRecords(ctx context.Context, exec domainrepo.Executor, playerID int, snapshot *entity.UserDataTransferSnapshot, masters *transferMasterData) error {
	type recordRow struct {
		PlayerID    int       `db:"player_id"`
		ChartID     int       `db:"chart_id"`
		Score       uint32    `db:"score"`
		ClearLampID int       `db:"clear_lamp_id"`
		ComboLampID int       `db:"combo_lamp_id"`
		FullChainID int       `db:"full_chain_id"`
		SlotID      int       `db:"slot_id"`
		SlotOrder   *int      `db:"slot_order"`
		UpdatedAt   time.Time `db:"updated_at"`
	}
	records := make([]recordRow, 0, len(snapshot.Records))
	for _, item := range snapshot.Records {
		records = append(records, recordRow{playerID, masters.charts[transferChartKey(item.SongOfficialIdx, item.Difficulty)].ID, uint32(item.Score), masters.clearLampIDs[item.ClearLampName], masters.comboLampIDs[item.ComboLampName], masters.fullChainIDs[item.FullChainName], masters.slotIDs[item.SlotName], item.SlotOrder, item.UpdatedAt})
	}
	if err := bulkTransferNamedExec(ctx, exec, `INSERT INTO player_records (player_id, chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, slot_id, slot_order, updated_at) VALUES (:player_id,:chart_id,:score,:clear_lamp_id,:combo_lamp_id,:full_chain_id,:slot_id,:slot_order,:updated_at)`, records); err != nil {
		return err
	}
	type historyRow struct {
		PlayerID    int       `db:"player_id"`
		ChartID     int       `db:"chart_id"`
		Score       uint32    `db:"score"`
		ClearLampID int       `db:"clear_lamp_id"`
		ComboLampID int       `db:"combo_lamp_id"`
		FullChainID int       `db:"full_chain_id"`
		UpdatedAt   time.Time `db:"updated_at"`
	}
	histories := make([]historyRow, 0, len(snapshot.RecordHistories))
	for _, item := range snapshot.RecordHistories {
		histories = append(histories, historyRow{playerID, masters.charts[transferChartKey(item.SongOfficialIdx, item.Difficulty)].ID, uint32(item.Score), masters.clearLampIDs[item.ClearLampName], masters.comboLampIDs[item.ComboLampName], masters.fullChainIDs[item.FullChainName], item.UpdatedAt})
	}
	if err := bulkTransferNamedExec(ctx, exec, `INSERT INTO player_record_histories (player_id, chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at) VALUES (:player_id,:chart_id,:score,:clear_lamp_id,:combo_lamp_id,:full_chain_id,:updated_at)`, histories); err != nil {
		return err
	}
	type worldsendRow struct {
		PlayerID    int       `db:"player_id"`
		ChartID     int       `db:"worldsend_chart_id"`
		Score       uint32    `db:"score"`
		ClearLampID int       `db:"clear_lamp_id"`
		ComboLampID int       `db:"combo_lamp_id"`
		FullChainID int       `db:"full_chain_id"`
		UpdatedAt   time.Time `db:"updated_at"`
	}
	worldsend := make([]worldsendRow, 0, len(snapshot.WorldsendRecords))
	for _, item := range snapshot.WorldsendRecords {
		worldsend = append(worldsend, worldsendRow{playerID, masters.worldsendChartIDs[item.SongOfficialIdx], uint32(item.Score), masters.clearLampIDs[item.ClearLampName], masters.comboLampIDs[item.ComboLampName], masters.fullChainIDs[item.FullChainName], item.UpdatedAt})
	}
	if err := bulkTransferNamedExec(ctx, exec, `INSERT INTO player_worldsend_records (player_id, worldsend_chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at) VALUES (:player_id,:worldsend_chart_id,:score,:clear_lamp_id,:combo_lamp_id,:full_chain_id,:updated_at)`, worldsend); err != nil {
		return err
	}
	worldsendHistories := make([]worldsendRow, 0, len(snapshot.WorldsendRecordHistories))
	for _, item := range snapshot.WorldsendRecordHistories {
		worldsendHistories = append(worldsendHistories, worldsendRow{playerID, masters.worldsendChartIDs[item.SongOfficialIdx], uint32(item.Score), masters.clearLampIDs[item.ClearLampName], masters.comboLampIDs[item.ComboLampName], masters.fullChainIDs[item.FullChainName], item.UpdatedAt})
	}
	return bulkTransferNamedExec(ctx, exec, `INSERT INTO player_worldsend_record_histories (player_id, worldsend_chart_id, score, clear_lamp_id, combo_lamp_id, full_chain_id, updated_at) VALUES (:player_id,:worldsend_chart_id,:score,:clear_lamp_id,:combo_lamp_id,:full_chain_id,:updated_at)`, worldsendHistories)
}

func saveTransferAuxiliaryData(ctx context.Context, exec domainrepo.Executor, playerID int, snapshot *entity.UserDataTransferSnapshot, masters *transferMasterData, honorRepo domainrepo.HonorRepository) error {
	type metricRow struct {
		PlayerID                 int       `db:"player_id"`
		OfficialRating           float64   `db:"official_rating"`
		OfficialOverpower        float64   `db:"official_overpower"`
		OfficialOverpowerPercent *float64  `db:"official_overpower_percent"`
		DataCollectedAt          time.Time `db:"data_collected_at"`
	}
	metrics := make([]metricRow, 0, len(snapshot.MetricHistories))
	for _, item := range snapshot.MetricHistories {
		metrics = append(metrics, metricRow{playerID, item.OfficialRating, item.OfficialOverpower, item.OfficialOverpowerPercent, item.DataCollectedAt})
	}
	if err := bulkTransferNamedExec(ctx, exec, `INSERT INTO player_metric_histories (player_id,official_rating,official_overpower,official_overpower_percent,data_collected_at) VALUES (:player_id,:official_rating,:official_overpower,:official_overpower_percent,:data_collected_at)`, metrics); err != nil {
		return err
	}
	type courseRow struct {
		PlayerID    int       `db:"player_id"`
		CourseID    int       `db:"course_id"`
		Score       uint32    `db:"score"`
		IsClear     bool      `db:"is_clear"`
		ComboLampID int       `db:"combo_lamp_id"`
		UpdatedAt   time.Time `db:"updated_at"`
	}
	courses := make([]courseRow, 0, len(snapshot.CourseRecords))
	for _, item := range snapshot.CourseRecords {
		courses = append(courses, courseRow{playerID, masters.courseIDs[item.CourseOfficialIdx], item.Score.Uint32(), item.IsClear, masters.comboLampIDs[item.ComboLampName], item.UpdatedAt})
	}
	if err := bulkTransferNamedExec(ctx, exec, `INSERT INTO player_course_records (player_id,course_id,score,is_clear,combo_lamp_id,updated_at) VALUES (:player_id,:course_id,:score,:is_clear,:combo_lamp_id,:updated_at)`, courses); err != nil {
		return err
	}
	type honorRow struct {
		PlayerID  int       `db:"player_id"`
		HonorID   int       `db:"honor_id"`
		Slot      int       `db:"slot"`
		CreatedAt time.Time `db:"created_at"`
	}
	honors := make([]honorRow, 0, len(snapshot.Honors))
	for _, item := range snapshot.Honors {
		id, exists := masters.resolveHonor(item.ImageURL, item.Name, item.TypeName)
		if !exists {
			result, err := honorRepo.EnsureHonor(ctx, exec, item.Name, masters.honorTypeIDs[item.TypeName], item.ImageURL)
			if err != nil {
				return err
			}
			id = result.ID
			masters.rememberHonor(id, item.ImageURL, item.Name, item.TypeName)
		}
		honors = append(honors, honorRow{playerID, id, item.Slot, item.EquippedAt})
	}
	if err := bulkTransferNamedExec(ctx, exec, `INSERT INTO player_honors (player_id,honor_id,slot,created_at) VALUES (:player_id,:honor_id,:slot,:created_at)`, honors); err != nil {
		return err
	}
	type favoriteRow struct {
		PlayerID  int       `db:"player_id"`
		SongID    int       `db:"song_id"`
		CreatedAt time.Time `db:"created_at"`
	}
	favorites := make([]favoriteRow, 0, len(snapshot.FavoriteSongs))
	for _, item := range snapshot.FavoriteSongs {
		favorites = append(favorites, favoriteRow{playerID, masters.songIDs[item.SongOfficialIdx], item.FavoritedAt})
	}
	if err := bulkTransferNamedExec(ctx, exec, `INSERT INTO player_favorite_songs (player_id,song_id,created_at) VALUES (:player_id,:song_id,:created_at)`, favorites); err != nil {
		return err
	}
	type lockedRow struct {
		PlayerID int  `db:"player_id"`
		SongID   int  `db:"song_id"`
		IsUltima bool `db:"is_ultima"`
	}
	locked := make([]lockedRow, 0, len(snapshot.LockedSongs))
	for _, item := range snapshot.LockedSongs {
		locked = append(locked, lockedRow{playerID, masters.songIDs[item.SongOfficialIdx], item.IsUltima})
	}
	return bulkTransferNamedExec(ctx, exec, `INSERT INTO player_locked_songs (player_id,song_id,is_ultima) VALUES (:player_id,:song_id,:is_ultima)`, locked)
}

func saveTransferGoals(ctx context.Context, exec domainrepo.Executor, userID int, snapshot *entity.UserDataTransferSnapshot, masters *transferMasterData) error {
	type goalRow struct {
		UserID            int       `db:"user_id"`
		GroupID           *uint32   `db:"group_id"`
		Title             string    `db:"title"`
		AchievementTypeID int       `db:"achievement_type_id"`
		AchievementParams []byte    `db:"achievement_params"`
		Attributes        []byte    `db:"attributes"`
		InvertValue       bool      `db:"invert_value"`
		InvertPercentage  bool      `db:"invert_percentage"`
		SortOrder         uint16    `db:"sort_order"`
		CreatedAt         time.Time `db:"created_at"`
	}
	rows := make([]goalRow, 0)
	appendGoal := func(item entity.UserDataTransferGoal, groupID *uint32) error {
		attrs, err := masters.internalizeGoalAttributes(item.Attributes)
		if err != nil {
			return err
		}
		rows = append(rows, goalRow{userID, groupID, item.Title, masters.achievementTypeIDs[item.AchievementType], item.AchievementParams, attrs, item.InvertValue, item.InvertPercentage, item.SortOrder, item.CreatedAt})
		return nil
	}
	for _, item := range snapshot.Goals.Ungrouped {
		if err := appendGoal(item, nil); err != nil {
			return err
		}
	}
	for _, group := range snapshot.Goals.Groups {
		result, err := exec.ExecContext(ctx, `INSERT INTO goal_groups (user_id,name,sort_order,created_at) VALUES (?,?,?,?)`, userID, group.Name.String(), group.SortOrder, group.CreatedAt)
		if err != nil {
			return err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		groupID, err := transferGoalGroupID(id)
		if err != nil {
			return err
		}
		for _, item := range group.Goals {
			if err := appendGoal(item, &groupID); err != nil {
				return err
			}
		}
	}
	return bulkTransferNamedExec(ctx, exec, `INSERT INTO goals (user_id,group_id,title,achievement_type_id,achievement_params,attributes,invert_value,invert_percentage,sort_order,created_at) VALUES (:user_id,:group_id,:title,:achievement_type_id,:achievement_params,:attributes,:invert_value,:invert_percentage,:sort_order,:created_at)`, rows)
}

// transferGoalGroupID はDBが返すIDを値域検証し、安全にドメインのID型へ変換します。
func transferGoalGroupID(id int64) (uint32, error) {
	if id < 0 || id > math.MaxUint32 {
		return 0, fmt.Errorf("goal_groups.id out of range: %d", id)
	}
	return uint32(id), nil
}

func saveTransferRecordFilters(ctx context.Context, exec domainrepo.Executor, userID int, snapshot *entity.UserDataTransferSnapshot) error {
	type row struct {
		ID          []byte    `db:"id"`
		UserID      int       `db:"user_id"`
		Name        string    `db:"name"`
		Value       []byte    `db:"filter_value_gzip"`
		IsWorldsend bool      `db:"is_worldsend"`
		CreatedAt   time.Time `db:"created_at"`
		UpdatedAt   time.Time `db:"updated_at"`
	}
	rows := make([]row, 0, len(snapshot.RecordFilters))
	for _, item := range snapshot.RecordFilters {
		id := uuid.NewV4()
		payload, err := json.Marshal(struct {
			SchemaVersion int             `json:"schema_version"`
			Filter        json.RawMessage `json:"filter"`
		}{item.SchemaVersion, item.Filter})
		if err != nil {
			return err
		}
		compressed, err := gzipTransferFilter(payload)
		if err != nil {
			return err
		}
		rows = append(rows, row{id[:], userID, item.Name, compressed, item.FilterType == usecase.RecordFilterTypeWorldsend, item.CreatedAt, item.UpdatedAt})
	}
	return bulkTransferNamedExec(ctx, exec, `INSERT INTO record_filters (id,user_id,name,filter_value_gzip,is_worldsend,created_at,updated_at) VALUES (:id,:user_id,:name,:filter_value_gzip,:is_worldsend,:created_at,:updated_at)`, rows)
}

func bulkTransferNamedExec[T any](ctx context.Context, exec domainrepo.Executor, query string, rows []T) error {
	for start := 0; start < len(rows); start += info.BulkInsertChunkSize {
		end := min(start+info.BulkInsertChunkSize, len(rows))
		if _, err := exec.NamedExecContext(ctx, query, rows[start:end]); err != nil {
			return err
		}
	}
	return nil
}
func optionalTransferMasterID(name *string, values map[string]int) (*int, error) {
	if name == nil {
		return nil, nil
	}
	id, ok := values[*name]
	if !ok {
		return nil, usecase.ErrDataTransferUnresolvedReference
	}
	return &id, nil
}
func gzipTransferFilter(value []byte) ([]byte, error) {
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write(value); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
func newTransferScore(value uint32) (score.Score, error) {
	parsed, err := score.NewScore(value)
	if err != nil {
		return 0, fmt.Errorf("invalid transfer score: %w", err)
	}
	return parsed, nil
}
func newTransferCourseScore(value uint32) (coursescore.CourseScore, error) {
	parsed, err := coursescore.New(value)
	if err != nil {
		return 0, fmt.Errorf("invalid transfer course score: %w", err)
	}
	return parsed, nil
}
