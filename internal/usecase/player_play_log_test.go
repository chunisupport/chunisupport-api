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

func TestRegister_PlayLogIsNewUpdatedAt(t *testing.T) {
	collectedAt := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	playedAtJST := "2026/09/01 18:42"
	playedAtUTC := time.Date(2026, 9, 1, 9, 42, 0, 0, time.UTC)

	newFixtureWithCandidates := func(t *testing.T) (*registrationFixture, *playLogRegistrationRepository) {
		t.Helper()
		fx := newRegistrationFixture(t)
		fx.payload.UpdatedAt = collectedAt.Format(time.RFC3339)
		repo := &playLogRegistrationRepository{
			registrationDataRepository: fx.data,
			candidates: []entity.PlayLogChartCandidate{
				{ID: 101, Title: "Full Song", Jacket: "jacket", DifficultyID: 4},
				{ID: 201, Title: "World Song", Jacket: "we-jacket"},
			},
		}
		fx.uc.playerDataRepo = repo
		return fx, repo
	}

	t.Run("新規レコードはis_newかつスコア一致のplayed_atを更新日時に使う", func(t *testing.T) {
		fx, _ := newFixtureWithCandidates(t)
		fx.payload.Scores.Standard = []PlayerDataScoreEntry{
			{Idx: "full-song", Diff: "MASTER", Score: 1005000, Slot: new("best"), Order: new(1)},
		}
		fx.payload.Scores.Worldsend = []PlayerDataScoreEntry{
			{Idx: "world-song", Diff: "WE", Score: 995000},
		}
		stdLog := validPlayLogEntry()
		stdLog.Score = new(1005000)
		stdLog.IsNew = new(true)
		stdLog.PlayedAt = playedAtJST
		weLog := validPlayLogEntry()
		weLog.Title = "World Song"
		weLog.Diff = "WE"
		weLog.Jacket = new("we-jacket")
		weLog.Score = new(995000)
		weLog.IsNew = new(true)
		weLog.PlayedAt = playedAtJST
		fx.payload.ScoreHistory = []PlayerDataPlayLogEntry{stdLog, weLog}

		_, err := fx.uc.Register(context.Background(), fx.user, fx.payload, "play-log-is-new-insert")
		require.NoError(t, err)
		require.Len(t, fx.data.savedInput.FullRecordsToInsert, 1)
		assert.Equal(t, playedAtUTC, fx.data.savedInput.FullRecordsToInsert[0].State.UpdatedAt)
		require.Len(t, fx.data.savedInput.WorldsendRecordsToInsert, 1)
		assert.Equal(t, playedAtUTC, fx.data.savedInput.WorldsendRecordsToInsert[0].State.UpdatedAt)
	})

	t.Run("既存レコード更新時は前回更新日時より新しい場合のみplayed_atを使う", func(t *testing.T) {
		tests := []struct {
			name            string
			beforeUpdatedAt time.Time
			expected        time.Time
		}{
			{
				name:            "前回更新日時より新しいplayed_atは採用する",
				beforeUpdatedAt: time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC),
				expected:        playedAtUTC,
			},
			{
				name:            "前回更新日時と同一のplayed_atは収集日時にフォールバックする",
				beforeUpdatedAt: playedAtUTC,
				expected:        collectedAt,
			},
			{
				name:            "前回更新日時より古いplayed_atは収集日時にフォールバックする",
				beforeUpdatedAt: time.Date(2026, 9, 1, 9, 42, 30, 0, time.UTC),
				expected:        collectedAt,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				fx, _ := newFixtureWithCandidates(t)
				fx.data.fullBefore = map[int]repository.PlayerRecordState{
					101: {Score: 1000000, ClearLampID: 1, ComboLampID: 1, FullChainID: 1, SlotID: 2, SlotOrder: new(1), UpdatedAt: tt.beforeUpdatedAt},
				}
				fx.data.worldsendBefore = map[int]repository.WorldsendRecordState{
					201: {Score: 990000, ClearLampID: 1, ComboLampID: 1, FullChainID: 1, UpdatedAt: tt.beforeUpdatedAt},
				}
				fx.payload.Scores.Standard = []PlayerDataScoreEntry{
					{Idx: "full-song", Diff: "MASTER", Score: 1005000, Slot: new("best"), Order: new(1)},
				}
				fx.payload.Scores.Worldsend = []PlayerDataScoreEntry{
					{Idx: "world-song", Diff: "WE", Score: 995000},
				}
				stdLog := validPlayLogEntry()
				stdLog.Score = new(1005000)
				stdLog.IsNew = new(true)
				stdLog.PlayedAt = playedAtJST
				weLog := validPlayLogEntry()
				weLog.Title = "World Song"
				weLog.Diff = "WE"
				weLog.Jacket = new("we-jacket")
				weLog.Score = new(995000)
				weLog.IsNew = new(true)
				weLog.PlayedAt = playedAtJST
				fx.payload.ScoreHistory = []PlayerDataPlayLogEntry{stdLog, weLog}

				_, err := fx.uc.Register(context.Background(), fx.user, fx.payload, "play-log-is-new-update")
				require.NoError(t, err)
				require.Len(t, fx.data.savedInput.FullRecordsToUpdate, 1)
				assert.Equal(t, tt.expected, fx.data.savedInput.FullRecordsToUpdate[0].State.UpdatedAt)
				require.Len(t, fx.data.savedInput.WorldsendRecordsToUpdate, 1)
				assert.Equal(t, tt.expected, fx.data.savedInput.WorldsendRecordsToUpdate[0].State.UpdatedAt)
			})
		}
	})

	t.Run("成績変化のない既存レコードはis_newがあっても遡及更新しない", func(t *testing.T) {
		fx, _ := newFixtureWithCandidates(t)
		fx.data.fullBefore = map[int]repository.PlayerRecordState{
			101: {Score: 1005000, ClearLampID: 1, ComboLampID: 1, FullChainID: 1, SlotID: 2, SlotOrder: new(1), UpdatedAt: time.Date(2026, 9, 1, 9, 50, 0, 0, time.UTC)},
		}
		fx.payload.Scores.Standard = []PlayerDataScoreEntry{
			{Idx: "full-song", Diff: "MASTER", Score: 1005000, Slot: new("best"), Order: new(1)},
		}
		stdLog := validPlayLogEntry()
		stdLog.Score = new(1005000)
		stdLog.IsNew = new(true)
		stdLog.PlayedAt = playedAtJST
		fx.payload.ScoreHistory = []PlayerDataPlayLogEntry{stdLog}

		_, err := fx.uc.Register(context.Background(), fx.user, fx.payload, "play-log-no-retroactive")
		require.NoError(t, err)
		assert.Empty(t, fx.data.savedInput.FullRecordsToInsert)
		assert.Empty(t, fx.data.savedInput.FullRecordsToUpdate)
	})

	t.Run("フォールバック条件と複数履歴からの最新選択", func(t *testing.T) {
		tests := []struct {
			name     string
			logs     []PlayerDataPlayLogEntry
			expected time.Time
		}{
			{
				name: "is_newがfalseの場合は収集日時を使う",
				logs: []PlayerDataPlayLogEntry{{
					Title: "Full Song", Diff: "master", Jacket: new("jacket"),
					Score: new(1005000), ClearLamp: new("clear"), ComboLv: new(2), FullChain: new(2),
					IsNew: new(false), PlayedAt: playedAtJST,
				}},
				expected: collectedAt,
			},
			{
				name: "スコアが一致しないis_newは使わない",
				logs: []PlayerDataPlayLogEntry{{
					Title: "Full Song", Diff: "master", Jacket: new("jacket"),
					Score: new(1004000), ClearLamp: new("clear"), ComboLv: new(2), FullChain: new(2),
					IsNew: new(true), PlayedAt: playedAtJST,
				}},
				expected: collectedAt,
			},
			{
				name: "収集日時より未来のplayed_atは使わない",
				logs: []PlayerDataPlayLogEntry{{
					Title: "Full Song", Diff: "master", Jacket: new("jacket"),
					Score: new(1005000), ClearLamp: new("clear"), ComboLv: new(2), FullChain: new(2),
					IsNew: new(true), PlayedAt: "2026/09/01 19:01",
				}},
				expected: collectedAt,
			},
			{
				name: "TIMESTAMP下限未満のplayed_atは使わない",
				logs: []PlayerDataPlayLogEntry{{
					Title: "Full Song", Diff: "master", Jacket: new("jacket"),
					Score: new(1005000), ClearLamp: new("clear"), ComboLv: new(2), FullChain: new(2),
					IsNew: new(true), PlayedAt: "1969/12/31 23:59",
				}},
				expected: collectedAt,
			},
			{
				name: "同一譜面・同一スコアのis_newが複数ある場合は最新を選ぶ",
				logs: []PlayerDataPlayLogEntry{
					{
						Title: "Full Song", Diff: "master", Jacket: new("jacket"),
						Score: new(1005000), ClearLamp: new("clear"), ComboLv: new(2), FullChain: new(2),
						IsNew: new(true), PlayedAt: "2026/09/01 18:10",
					},
					{
						Title: "Full Song", Diff: "master", Jacket: new("jacket"),
						Score: new(1005000), ClearLamp: new("clear"), ComboLv: new(2), FullChain: new(2),
						IsNew: new(true), PlayedAt: playedAtJST,
					},
				},
				expected: playedAtUTC,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				fx, _ := newFixtureWithCandidates(t)
				fx.payload.Scores.Standard = []PlayerDataScoreEntry{
					{Idx: "full-song", Diff: "MASTER", Score: 1005000, Slot: new("best"), Order: new(1)},
				}
				fx.payload.ScoreHistory = tt.logs

				_, err := fx.uc.Register(context.Background(), fx.user, fx.payload, "play-log-fallback")
				require.NoError(t, err)
				require.Len(t, fx.data.savedInput.FullRecordsToInsert, 1)
				assert.Equal(t, tt.expected, fx.data.savedInput.FullRecordsToInsert[0].State.UpdatedAt)
			})
		}
	})

	t.Run("TIMESTAMP上限超過のplayed_atは使わない", func(t *testing.T) {
		futureCollectedAt := time.Date(2038, 1, 20, 0, 0, 0, 0, time.UTC)
		fx, _ := newFixtureWithCandidates(t)
		fx.payload.UpdatedAt = futureCollectedAt.Format(time.RFC3339)
		fx.payload.Scores.Standard = []PlayerDataScoreEntry{
			{Idx: "full-song", Diff: "MASTER", Score: 1005000, Slot: new("best"), Order: new(1)},
		}
		log := validPlayLogEntry()
		log.Score = new(1005000)
		log.IsNew = new(true)
		log.PlayedAt = "2038/01/19 12:15"
		fx.payload.ScoreHistory = []PlayerDataPlayLogEntry{log}

		_, err := fx.uc.Register(context.Background(), fx.user, fx.payload, "play-log-timestamp-upper-bound")
		require.NoError(t, err)
		require.Len(t, fx.data.savedInput.FullRecordsToInsert, 1)
		assert.Equal(t, futureCollectedAt, fx.data.savedInput.FullRecordsToInsert[0].State.UpdatedAt)
	})
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
