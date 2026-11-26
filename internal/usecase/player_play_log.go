package usecase

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
)

func validatePlayLogCount(entries []PlayerDataPlayLogEntry) error {
	if len(entries) == 0 || len(entries) > info.MaxPlayerPlayLogs {
		return &PlayerDataValidationError{
			Field: "score_history", Message: fmt.Sprintf("must contain 1 to %d entries", info.MaxPlayerPlayLogs),
		}
	}
	return nil
}

type validatedPlayLog struct {
	entity.PlayerPlayLog
	title string
}

func buildPlayLogs(entries []PlayerDataPlayLogEntry, masters *playerDataMaster) ([]validatedPlayLog, error) {
	logs := make([]validatedPlayLog, len(entries))
	for i, entry := range entries {
		log, err := buildPlayLog(i, entry, masters)
		if err != nil {
			return nil, err
		}
		logs[i] = log
	}
	return logs, nil
}

func buildPlayLog(index int, entry PlayerDataPlayLogEntry, masters *playerDataMaster) (validatedPlayLog, error) {
	invalid := func(field, message string, cause error) (validatedPlayLog, error) {
		return validatedPlayLog{}, &PlayerDataValidationError{
			Field: fmt.Sprintf("score_history[%d].%s", index, field), Message: message, Err: cause,
		}
	}

	title := strings.TrimSpace(entry.Title)
	if title == "" || utf8.RuneCountInString(title) > info.MaxPlayLogTitleLength {
		return invalid("title", fmt.Sprintf("must contain 1 to %d characters", info.MaxPlayLogTitleLength), nil)
	}
	var jacket *string
	if entry.Jacket != nil {
		value := strings.TrimSpace(*entry.Jacket)
		if utf8.RuneCountInString(value) > info.MaxPlayLogJacketLength {
			return invalid("jacket_img_id", fmt.Sprintf("must contain at most %d characters", info.MaxPlayLogJacketLength), nil)
		}
		if value != "" {
			jacket = &value
		}
	}

	rawDiff := strings.ToUpper(strings.TrimSpace(entry.Diff))
	difficulty, ok := difficultyCodeToName[rawDiff]
	switch rawDiff {
	case "WE", "WORLDSEND", info.StatsDifficultyWorldsend:
		difficulty, ok = info.StatsDifficultyWorldsend, true
	}
	if !ok {
		return invalid("diff", "unknown difficulty", nil)
	}
	var difficultyID *int
	if difficulty != info.StatsDifficultyWorldsend {
		master, exists := masters.Difficulties[difficulty]
		if !exists {
			return validatedPlayLog{}, fmt.Errorf("play log difficulty master not found: %s", difficulty)
		}
		difficultyID = new(master.ID)
	}
	if entry.Score == nil || *entry.Score < minScoreValue || *entry.Score > maxScoreValue {
		return invalid("score", fmt.Sprintf("must be between %d and %d", minScoreValue, maxScoreValue), nil)
	}
	if entry.IsNew == nil {
		return invalid("is_new", "is required", nil)
	}
	if entry.ComboLv == nil {
		return invalid("cmb_lv", "is required", nil)
	}
	if entry.FullChain == nil {
		return invalid("fch_lv", "is required", nil)
	}

	clearID, err := resolveClearLampID(entry.ClearLamp, masters)
	if err != nil {
		return invalid("clear_lamp", err.Error(), err)
	}
	comboID, err := resolveComboLampID(entry.ComboLv, masters)
	if err != nil {
		return invalid("cmb_lv", err.Error(), err)
	}
	chainID, err := resolveFullChainID(entry.FullChain, masters)
	if err != nil {
		return invalid("fch_lv", err.Error(), err)
	}
	score := PlayerDataScoreEntry{Score: *entry.Score, Idx: title, ComboLv: entry.ComboLv, FullChain: entry.FullChain}
	if err := validateScoreEntry(&score, "score_history", index); err != nil {
		return invalid("score", err.Error(), err)
	}

	playedAt, err := parsePlayerDataJST(entry.PlayedAt)
	if err != nil || playedAt.Year() < 1000 {
		return invalid("played_at", fmt.Sprintf("must be a valid JST date and time in %s format", tokyoLayout), err)
	}

	return validatedPlayLog{
		PlayerPlayLog: entity.PlayerPlayLog{
			Jacket: jacket, DifficultyID: difficultyID, Score: *entry.Score,
			ClearLampID: clearID, ComboLampID: comboID, FullChainID: chainID, PlayedAt: playedAt,
		},
		title: title,
	}, nil
}

func (us *playerDataUsecase) replacePlayLogs(ctx context.Context, tx repository.Executor, playerID int, logs []validatedPlayLog) error {
	var jackets []string
	for _, log := range logs {
		if log.Jacket != nil {
			jackets = append(jackets, *log.Jacket)
		}
	}
	slices.Sort(jackets)
	jackets = slices.Compact(jackets)
	byJacket := make(map[string][]entity.PlayLogChartCandidate, len(jackets))
	if len(jackets) > 0 {
		candidates, err := us.playerDataRepo.FindPlayLogChartCandidates(ctx, tx, jackets)
		if err != nil {
			return err
		}
		for _, candidate := range candidates {
			byJacket[candidate.Jacket] = append(byJacket[candidate.Jacket], candidate)
		}
	}
	resolved := make([]entity.PlayerPlayLog, len(logs))
	for i, log := range logs {
		var candidates []entity.PlayLogChartCandidate
		if log.Jacket != nil {
			candidates = byJacket[*log.Jacket]
		}
		if chartID, ok := resolvePlayLogChart(log, candidates); ok {
			if log.DifficultyID == nil {
				log.WorldsendChartID = new(chartID)
			} else {
				log.ChartID = new(chartID)
			}
		}
		resolved[i] = log.PlayerPlayLog
	}
	return us.playerDataRepo.ReplacePlayLogs(ctx, tx, playerID, resolved)
}

func resolvePlayLogChart(log validatedPlayLog, byJacket []entity.PlayLogChartCandidate) (int, bool) {
	var difficultyID int
	if log.DifficultyID != nil {
		difficultyID = *log.DifficultyID
	}
	if len(byJacket) == 1 && byJacket[0].DifficultyID == difficultyID {
		return byJacket[0].ID, true
	}

	var chartID int
	found := false
	for _, candidate := range byJacket {
		if candidate.DifficultyID != difficultyID || candidate.Title != log.title {
			continue
		}
		if found {
			return 0, false
		}
		chartID = candidate.ID
		found = true
	}
	return chartID, found
}
