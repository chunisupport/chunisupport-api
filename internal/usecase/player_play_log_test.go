package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validPlayLogEntry() PlayerDataPlayLogEntry {
	return PlayerDataPlayLogEntry{
		Title: "Full Song", Diff: "master", Jacket: new("jacket"),
		Score: new(1000000), ClearLamp: new("clear"), ComboLv: new(2), FullChain: new(2),
		IsNew: new(false), PlayedAt: "2026/11/26 00:10",
	}
}

func TestRegister_PlayLog(t *testing.T) {
	t.Run("受信順で保存し未解決行も残す", func(t *testing.T) {
		fx := newRegistrationFixture(t)
		repo := &playLogRegistrationRepository{
			registrationDataRepository: fx.data,
			candidates: []entity.PlayLogChartCandidate{
				{ID: 102, Title: "別の曲", Jacket: "jacket", DifficultyID: 4},
				{ID: 101, Title: "Full Song", Jacket: "jacket", DifficultyID: 4},
				{ID: 101, Title: "Full Song", Jacket: "jacket"},
			},
		}
		fx.uc.playerDataRepo = repo
		standard := validPlayLogEntry()
		we := standard
		we.Diff = "worldsend"
		unresolved := standard
		unresolved.Title, unresolved.Jacket, unresolved.ClearLamp = "未登録曲", new("unknown-jacket"), nil
		fx.payload.ScoreHistory = []PlayerDataPlayLogEntry{standard, we, standard, unresolved}

		_, err := fx.uc.Register(context.Background(), fx.user, fx.payload, "play-log")
		require.NoError(t, err)
		require.Len(t, repo.saved, 4)
		assert.Same(t, fx.tx, repo.saveExec)
		assert.Equal(t, []string{"jacket", "unknown-jacket"}, repo.jackets)
		assert.Equal(t, 77, repo.playerID)
		assert.Equal(t, new(101), repo.saved[0].ChartID)
		assert.Equal(t, new(4), repo.saved[0].DifficultyID)
		assert.Nil(t, repo.saved[0].WorldsendChartID)
		assert.Equal(t, new(101), repo.saved[1].WorldsendChartID)
		assert.Nil(t, repo.saved[1].ChartID)
		assert.Nil(t, repo.saved[1].DifficultyID)
		assert.Equal(t, repo.saved[0], repo.saved[2])
		assert.Nil(t, repo.saved[3].ChartID)
		assert.Nil(t, repo.saved[3].WorldsendChartID)
		assert.Equal(t, new("unknown-jacket"), repo.saved[3].Jacket)
		assert.Equal(t, new(4), repo.saved[3].DifficultyID)
		assert.Equal(t, 1, repo.saved[3].ClearLampID)
		assert.Equal(t, time.Date(2026, 11, 25, 15, 10, 0, 0, time.UTC), repo.saved[0].PlayedAt)
		assert.Equal(t, 2, repo.saved[0].ClearLampID)
		assert.Equal(t, 2, repo.saved[0].ComboLampID)
		assert.Equal(t, 3, repo.saved[0].FullChainID)
	})

	t.Run("同名の候補が残る場合はNEW RECORDでも未紐付けで保存する", func(t *testing.T) {
		fx := newRegistrationFixture(t)
		repo := &playLogRegistrationRepository{
			registrationDataRepository: fx.data,
			candidates: []entity.PlayLogChartCandidate{
				{ID: 102, Title: "Full Song", Jacket: "jacket", DifficultyID: 4},
				{ID: 101, Title: "Full Song", Jacket: "jacket", DifficultyID: 4},
			},
		}
		fx.uc.playerDataRepo = repo
		fx.payload.Scores.Standard = []PlayerDataScoreEntry{{Idx: "full-song", Diff: "MASTER", Score: 1000000, Slot: new("best"), Order: new(1)}}
		entry := validPlayLogEntry()
		entry.IsNew = new(true)
		fx.payload.ScoreHistory = []PlayerDataPlayLogEntry{entry}

		_, err := fx.uc.Register(context.Background(), fx.user, fx.payload, "play-log-best")
		require.NoError(t, err)
		require.Len(t, repo.saved, 1)
		assert.Nil(t, repo.saved[0].ChartID)
		assert.Nil(t, repo.saved[0].WorldsendChartID)
		assert.Equal(t, new("jacket"), repo.saved[0].Jacket)
		assert.Equal(t, new(4), repo.saved[0].DifficultyID)
	})

	t.Run("候補に曲名と難易度の組み合わせがない場合や画像IDがない場合は未紐付けで保存する", func(t *testing.T) {
		fx := newRegistrationFixture(t)
		repo := &playLogRegistrationRepository{
			registrationDataRepository: fx.data,
			candidates: []entity.PlayLogChartCandidate{
				{ID: 102, Title: "別の曲", Jacket: "jacket", DifficultyID: 4},
				{ID: 103, Title: "Full Song", Jacket: "jacket", DifficultyID: 3},
			},
		}
		fx.uc.playerDataRepo = repo
		fx.payload.Scores.Standard = []PlayerDataScoreEntry{{Idx: "full-song", Diff: "MASTER", Score: 1000000, Slot: new("best"), Order: new(1)}}
		entry := validPlayLogEntry()
		entry.IsNew = new(true)
		noJacket := entry
		noJacket.Jacket = nil
		fx.payload.ScoreHistory = []PlayerDataPlayLogEntry{entry, noJacket}

		_, err := fx.uc.Register(context.Background(), fx.user, fx.payload, "play-log-known-jacket")
		require.NoError(t, err)
		require.Len(t, repo.saved, 2)
		assert.Nil(t, repo.saved[0].ChartID)
		assert.Nil(t, repo.saved[1].ChartID)
		assert.Nil(t, repo.saved[1].WorldsendChartID)
		assert.Nil(t, repo.saved[1].Jacket)
		assert.Equal(t, new(4), repo.saved[1].DifficultyID)
	})

	t.Run("同一本文の再送では保存済みの譜面の紐付けを維持する", func(t *testing.T) {
		fx := newRegistrationFixture(t)
		updatedAt, err := time.Parse(time.RFC3339, fx.payload.UpdatedAt)
		require.NoError(t, err)
		fx.data.latestUpdate, err = entity.NewPlayerLatestUpdate(77, 1, []byte("gzip"), updatedAt, updatedAt.Add(time.Minute), "replay")
		require.NoError(t, err)
		fx.data.latestUpdateErr = nil
		repo := &playLogRegistrationRepository{registrationDataRepository: fx.data}
		fx.uc.playerDataRepo = repo

		_, err = fx.uc.Register(context.Background(), fx.user, fx.payload, "replay")
		require.NoError(t, err)
		assert.Nil(t, repo.saveExec)
	})

	t.Run("履歴の保存エラーをトランザクションへ返す", func(t *testing.T) {
		fx := newRegistrationFixture(t)
		failure := errors.New("play log insert failed")
		repo := &playLogRegistrationRepository{registrationDataRepository: fx.data, saveErr: failure}
		fx.uc.playerDataRepo = repo

		_, err := fx.uc.Register(context.Background(), fx.user, fx.payload, "play-log-error")
		require.ErrorIs(t, err, failure)
		assert.Same(t, fx.tx, repo.saveExec)
		assert.Equal(t, 1, fx.data.saveCalls)
		assert.Zero(t, fx.data.latestUpdateSaveCalls)
	})

	t.Run("50件まで同一内容のプレイを保持する", func(t *testing.T) {
		fx := newRegistrationFixture(t)
		repo := &playLogRegistrationRepository{registrationDataRepository: fx.data}
		fx.uc.playerDataRepo = repo
		fx.payload.ScoreHistory = make([]PlayerDataPlayLogEntry, info.MaxPlayerPlayLogs)
		for i := range fx.payload.ScoreHistory {
			fx.payload.ScoreHistory[i] = validPlayLogEntry()
		}

		_, err := fx.uc.Register(context.Background(), fx.user, fx.payload, "play-log-50")
		require.NoError(t, err)
		assert.Len(t, repo.saved, info.MaxPlayerPlayLogs)
	})

	badDate := validPlayLogEntry()
	badDate.PlayedAt = "2026/02/30 00:10"
	badDifficulty := validPlayLogEntry()
	badDifficulty.Diff = "INVALID"
	badCombo := validPlayLogEntry()
	badCombo.ComboLv = new(0)
	badChain := validPlayLogEntry()
	badChain.ComboLv = new(1)
	badScore := validPlayLogEntry()
	badScore.Score = new(maxScoreValue + 1)
	for _, tt := range []struct {
		name  string
		logs  []PlayerDataPlayLogEntry
		field string
	}{
		{"欠落", nil, "score_history"},
		{"空配列", []PlayerDataPlayLogEntry{}, "score_history"},
		{"51件", make([]PlayerDataPlayLogEntry, 51), "score_history"},
		{"スコアなし", []PlayerDataPlayLogEntry{{Title: "Full Song", Diff: "MASTER"}}, "score_history[0].score"},
		{"要素がnull", []PlayerDataPlayLogEntry{{}}, "score_history[0].title"},
		{"存在しない日付", []PlayerDataPlayLogEntry{badDate}, "score_history[0].played_at"},
		{"対象外の難易度", []PlayerDataPlayLogEntry{badDifficulty}, "score_history[0].diff"},
		{"範囲外のコンボランプ", []PlayerDataPlayLogEntry{badCombo}, "score_history[0].cmb_lv"},
		{"フルコンボなしのフルチェイン", []PlayerDataPlayLogEntry{badChain}, "score_history[0].score"},
		{"範囲外のスコア", []PlayerDataPlayLogEntry{badScore}, "score_history[0].score"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fx := newRegistrationFixture(t)
			fx.payload.ScoreHistory = tt.logs
			_, err := fx.uc.Register(context.Background(), fx.user, fx.payload, "invalid-play-log")
			var validationErr *PlayerDataValidationError
			require.ErrorAs(t, err, &validationErr)
			assert.Equal(t, tt.field, validationErr.Field)
			assert.Zero(t, fx.data.saveCalls)
		})
	}
}

type playLogRegistrationRepository struct {
	*registrationDataRepository
	candidates []entity.PlayLogChartCandidate
	jackets    []string
	saved      []entity.PlayerPlayLog
	saveExec   repository.Executor
	playerID   int
	saveErr    error
}

func (r *playLogRegistrationRepository) FindPlayLogChartCandidates(_ context.Context, exec repository.Executor, jackets []string) ([]entity.PlayLogChartCandidate, error) {
	assert.Same(r.t, r.tx, exec)
	r.jackets = jackets
	return r.candidates, nil
}

func (r *playLogRegistrationRepository) ReplacePlayLogs(_ context.Context, exec repository.Executor, playerID int, logs []entity.PlayerPlayLog) error {
	r.saveExec, r.playerID, r.saved = exec, playerID, logs
	return r.saveErr
}
