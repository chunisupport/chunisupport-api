package entity

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/chunisupport/chunisupport-api/internal/domain/constants"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/goalgroupname"
)

func validateUserDataTransferHonors(honors []UserDataTransferHonor) error {
	seenSlots := make(map[int]struct{}, len(honors))
	for i, honor := range honors {
		path := fmt.Sprintf("honors[%d]", i)
		if honor.Slot < 1 || honor.Slot > 3 {
			return invalidUserDataTransfer(path + ".slot is out of range")
		}
		if _, exists := seenSlots[honor.Slot]; exists {
			return invalidUserDataTransfer(path + ".slot is duplicated")
		}
		seenSlots[honor.Slot] = struct{}{}
		if strings.TrimSpace(honor.Name) == "" || strings.TrimSpace(honor.TypeName) == "" {
			return invalidUserDataTransfer(path + " has an empty master name")
		}
		if !validOptionalName(honor.ImageURL) {
			return invalidUserDataTransfer(path + ".image_url is invalid")
		}
		if err := validateUTCDateTime(path+".equipped_at", honor.EquippedAt); err != nil {
			return err
		}
	}
	return nil
}

func validateUserDataTransferFavoriteSongs(songs []UserDataTransferFavoriteSong) error {
	if len(songs) > constants.PlayerFavoriteSongMaxCount {
		return invalidUserDataTransfer("favorite_songs exceeds its limit")
	}
	seen := make(map[string]struct{}, len(songs))
	for i, song := range songs {
		path := fmt.Sprintf("favorite_songs[%d]", i)
		if err := validateSongOfficialIdx(path+".song_official_idx", song.SongOfficialIdx); err != nil {
			return err
		}
		if _, exists := seen[song.SongOfficialIdx]; exists {
			return invalidUserDataTransfer(path + " duplicates a song")
		}
		seen[song.SongOfficialIdx] = struct{}{}
		if err := validateUTCDateTime(path+".favorited_at", song.FavoritedAt); err != nil {
			return err
		}
	}
	return nil
}

func validateUserDataTransferLockedSongs(songs []UserDataTransferLockedSong) error {
	seen := make(map[string]struct{}, len(songs))
	for i, song := range songs {
		path := fmt.Sprintf("locked_songs[%d]", i)
		if err := validateSongOfficialIdx(path+".song_official_idx", song.SongOfficialIdx); err != nil {
			return err
		}
		key := song.SongOfficialIdx + "\x00" + strconv.FormatBool(song.IsUltima)
		if _, exists := seen[key]; exists {
			return invalidUserDataTransfer(path + " duplicates a locked-song state")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateUserDataTransferGoals(goals UserDataTransferGoals) error {
	if len(goals.Groups) > constants.GoalGroupMaxPerUser {
		return invalidUserDataTransfer("goals.groups exceeds its limit")
	}
	goalCount := len(goals.Ungrouped)
	seenNames := make(map[string]struct{}, len(goals.Groups))
	for i, group := range goals.Groups {
		path := fmt.Sprintf("goals.groups[%d]", i)
		name, err := goalgroupname.NewGoalGroupName(group.Name.String())
		if err != nil || name.String() != group.Name.String() {
			return invalidUserDataTransfer(path + ".name is invalid")
		}
		if _, exists := seenNames[group.Name.String()]; exists {
			return invalidUserDataTransfer(path + ".name is duplicated")
		}
		seenNames[group.Name.String()] = struct{}{}
		if group.SortOrder != uint16(i+1) {
			return invalidUserDataTransfer(path + ".sort_order is not continuous")
		}
		if err := validateUTCDateTime(path+".created_at", group.CreatedAt); err != nil {
			return err
		}
		if err := validateUserDataTransferGoalList(path+".goals", group.Goals); err != nil {
			return err
		}
		goalCount += len(group.Goals)
	}
	if goalCount > constants.GoalMaxPerUser {
		return invalidUserDataTransfer("goals exceeds its limit")
	}
	return validateUserDataTransferGoalList("goals.ungrouped", goals.Ungrouped)
}

func validateUserDataTransferGoalList(path string, goals []UserDataTransferGoal) error {
	for i, goal := range goals {
		goalPath := fmt.Sprintf("%s[%d]", path, i)
		if goal.SortOrder != uint16(i+1) {
			return invalidUserDataTransfer(goalPath + ".sort_order is not continuous")
		}
		title := strings.TrimSpace(goal.Title)
		if title == "" || title != goal.Title || len([]rune(title)) > 30 || hasControlCharacter(title) {
			return invalidUserDataTransfer(goalPath + ".title is invalid")
		}
		if strings.TrimSpace(goal.AchievementType) == "" {
			return invalidUserDataTransfer(goalPath + ".achievement_type is required")
		}
		if !isJSONObject(goal.AchievementParams) {
			return invalidUserDataTransfer(goalPath + ".achievement_params must be an object")
		}
		if !isJSONObject(goal.Attributes) {
			return invalidUserDataTransfer(goalPath + ".attributes must be an object")
		}
		if err := validateUTCDateTime(goalPath+".created_at", goal.CreatedAt); err != nil {
			return err
		}
	}
	return nil
}

func validateUserDataTransferRecordFilters(filters []UserDataTransferRecordFilter) error {
	if len(filters) > constants.RecordFilterMaxPerUser {
		return invalidUserDataTransfer("record_filters exceeds its limit")
	}
	for i, filter := range filters {
		path := fmt.Sprintf("record_filters[%d]", i)
		name := strings.TrimSpace(filter.Name)
		if name == "" || name != filter.Name || len([]rune(name)) > constants.RecordFilterNameMaxLength || hasControlCharacter(name) {
			return invalidUserDataTransfer(path + ".name is invalid")
		}
		if filter.FilterType != "standard" && filter.FilterType != "worldsend" {
			return invalidUserDataTransfer(path + ".filter_type is invalid")
		}
		if filter.SchemaVersion <= 0 {
			return invalidUserDataTransfer(path + ".schema_version is invalid")
		}
		compactFilter, err := compactJSONObject(filter.Filter)
		if err != nil {
			return invalidUserDataTransfer(path + ".filter must be a JSON object")
		}
		payloadLength := len(`{"schema_version":`) + len(strconv.Itoa(filter.SchemaVersion)) + len(`,"filter":`) + len(compactFilter) + 1
		if payloadLength > constants.RecordFilterMaxPayloadBytes {
			return invalidUserDataTransfer(path + " exceeds the payload limit")
		}
		if err := validateUTCDateTime(path+".created_at", filter.CreatedAt); err != nil {
			return err
		}
		if err := validateUTCDateTime(path+".updated_at", filter.UpdatedAt); err != nil {
			return err
		}
		if filter.UpdatedAt.Before(filter.CreatedAt) {
			return invalidUserDataTransfer(path + ".updated_at precedes created_at")
		}
	}
	return nil
}

func validateSongOfficialIdx(path, value string) error {
	if strings.TrimSpace(value) != value || value == "" || len(value) > 10 {
		return invalidUserDataTransfer(path + " is invalid")
	}
	return nil
}

func validDifficulty(value string) bool {
	switch value {
	case "BASIC", "ADVANCED", "EXPERT", "MASTER", "ULTIMA":
		return true
	default:
		return false
	}
}

func userDataTransferSlotLimit(name string) (int, bool) {
	switch name {
	case "best":
		return constants.BestSlotMaxCount, true
	case "new":
		return constants.NewSlotMaxCount, true
	case "best_candidate", "new_candidate":
		return constants.CandidateSlotMaxCount, true
	default:
		return 0, false
	}
}

func validOptionalName(value *string) bool {
	return value == nil || *value != "" && strings.TrimSpace(*value) == *value
}

func validateUTCDateTime(path string, value time.Time) error {
	if value.IsZero() {
		return invalidUserDataTransfer(path + " is required")
	}
	_, offset := value.Zone()
	if offset != 0 {
		return invalidUserDataTransfer(path + " must be UTC")
	}
	return nil
}

func validateOptionalUTCDateTime(path string, value *time.Time) error {
	if value == nil {
		return nil
	}
	return validateUTCDateTime(path, *value)
}

func databaseSecond(value time.Time) time.Time {
	return value.UTC().Truncate(time.Second)
}

func isJSONObject(value json.RawMessage) bool {
	_, err := compactJSONObject(value)
	return err == nil
}

func compactJSONObject(value json.RawMessage) ([]byte, error) {
	trimmed := bytes.TrimSpace(value)
	if len(trimmed) < 2 || trimmed[0] != '{' || trimmed[len(trimmed)-1] != '}' || !json.Valid(trimmed) {
		return nil, errors.New("not a JSON object")
	}
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, trimmed); err != nil {
		return nil, err
	}
	return compacted.Bytes(), nil
}

func hasControlCharacter(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func invalidUserDataTransfer(message string) error {
	return fmt.Errorf("%w: %s", ErrInvalidUserDataTransfer, message)
}
