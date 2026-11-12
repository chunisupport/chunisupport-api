package api_internal_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/playername"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockUserUsecase は usecase.UserUsecase のモックです。
type mockUserUsecase struct {
	mock.Mock
}

func testUserPlayerOutput(t *testing.T, name string, level int, now time.Time) *usecase.UserPlayerOutput {
	t.Helper()
	value, err := playername.NewPlayerName(name)
	if err != nil {
		value, err = playername.NewPlayerName("テストプレイヤー")
	}
	assert.NoError(t, err)
	return &usecase.UserPlayerOutput{Player: &entity.Player{Name: value, Level: level, CreatedAt: now, UpdatedAt: now}, Honors: []*entity.PlayerHonor{}}
}

func testPlayerRecordOutput(id string) *usecase.PlayerRecordOutput {
	return &usecase.PlayerRecordOutput{PlayerRecord: &entity.PlayerRecord{Song: &entity.Song{DisplayID: id}}}
}

func testWorldsendRecordOutput(id string) *usecase.WorldsendRecordOutput {
	return &usecase.WorldsendRecordOutput{PlayerWorldsendRecord: &entity.PlayerWorldsendRecord{Song: &entity.Song{DisplayID: id}}, ID: id}
}

func (m *mockUserUsecase) GetUserProfileWithRecords(ctx context.Context, username string, requester *entity.User) (*usecase.UserProfileWithRecordsOutput, error) {
	args := m.Called(ctx, username, requester)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.UserProfileWithRecordsOutput), args.Error(1)
}

func (m *mockUserUsecase) GetUserUpdatedAt(ctx context.Context, username string, requester *entity.User) (*usecase.UserUpdatedAtOutput, error) {
	args := m.Called(ctx, username, requester)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.UserUpdatedAtOutput), args.Error(1)
}

func (m *mockUserUsecase) GetUserProfileRatingView(ctx context.Context, username string, requester *entity.User) (*usecase.UserProfileRatingViewOutput, error) {
	args := m.Called(ctx, username, requester)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.UserProfileRatingViewOutput), args.Error(1)
}

func (m *mockUserUsecase) GetUserProfileRecordView(ctx context.Context, username string, requester *entity.User) (*usecase.UserProfileRecordViewOutput, error) {
	args := m.Called(ctx, username, requester)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.UserProfileRecordViewOutput), args.Error(1)
}

func (m *mockUserUsecase) GetUserSongRecord(ctx context.Context, username string, requester *entity.User, displayID string, difficulty string) (*usecase.UserSongRecordOutput, error) {
	args := m.Called(ctx, username, requester, displayID, difficulty)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.UserSongRecordOutput), args.Error(1)
}

func (m *mockUserUsecase) GetUserWorldsendSongRecord(ctx context.Context, username string, requester *entity.User, displayID string) (*usecase.UserWorldsendSongRecordOutput, error) {
	args := m.Called(ctx, username, requester, displayID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.UserWorldsendSongRecordOutput), args.Error(1)
}

func (m *mockUserUsecase) GetAllUsersForAdmin(ctx context.Context, page int, limit int, name string) ([]usecase.AdminUserOutput, error) {
	args := m.Called(ctx, page, limit, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]usecase.AdminUserOutput), args.Error(1)
}

func (m *mockUserUsecase) DeleteUser(ctx context.Context, requester *entity.User, username string) error {
	args := m.Called(ctx, requester, username)
	return args.Error(0)
}

func TestUserHandler_GetUserSongRecord(t *testing.T) {
	// Given
	e := newTestEcho()
	mockUsecase := new(mockUserUsecase)
	h := api_internal.NewUserHandler(mockUsecase)
	// 本文の検証に使う固定日時と識別値（仕様から決めた固定値で、本番の式からは計算しない）。
	now := time.Date(2025, 6, 20, 10, 0, 0, 0, time.UTC)
	standardRecord := testPlayerRecordOutput("0000000000000001")
	standardRecord.PlayerRecord.UpdatedAt = now
	expected := &usecase.UserSongRecordOutput{
		Standard:  []*usecase.PlayerRecordOutput{standardRecord},
		UpdatedAt: &now,
		Meta:      &usecase.UserSongRecordMetaOutput{},
	}
	mockUsecase.On(
		"GetUserSongRecord",
		mock.Anything,
		"testuser",
		(*entity.User)(nil),
		"0000000000000001",
		"MASTER",
	).Return(expected, nil).Once()
	req := httptest.NewRequest(http.MethodGet, "/internal/users/testuser/record/songs/0000000000000001?difficulty=master", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{
		{Name: "username", Value: "testuser"},
		{Name: "id", Value: "0000000000000001"},
	})

	// When
	err := h.GetUserSongRecord(c)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	// 1件の標準レコードが識別値つきで返ることを確かめる（空配列との取り違えを検知する）。
	standard, ok := body["standard"].([]any)
	if assert.True(t, ok) && assert.Len(t, standard, 1) {
		first, ok := standard[0].(map[string]any)
		if assert.True(t, ok) {
			assert.Equal(t, "0000000000000001", first["id"])
			assert.Equal(t, now.Format(time.RFC3339), first["updated_at"])
			assert.Equal(t, true, first["is_played"])
		}
	}
	// メタの更新日時がレコードの更新日時と一致することを確かめる。
	meta, ok := body["meta"].(map[string]any)
	if assert.True(t, ok) {
		assert.Equal(t, now.Format(time.RFC3339), meta["updated_at"])
	}
	mockUsecase.AssertExpectations(t)
}

func TestUserHandler_GetUserWorldsendSongRecord_楽曲不存在(t *testing.T) {
	// Given
	e := newTestEcho()
	mockUsecase := new(mockUserUsecase)
	h := api_internal.NewUserHandler(mockUsecase)
	mockUsecase.On(
		"GetUserWorldsendSongRecord",
		mock.Anything,
		"testuser",
		(*entity.User)(nil),
		"0000000000000002",
	).Return((*usecase.UserWorldsendSongRecordOutput)(nil), repository.ErrSongNotFound).Once()
	req := httptest.NewRequest(http.MethodGet, "/internal/users/testuser/record/worldsend-songs/0000000000000002", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{
		{Name: "username", Value: "testuser"},
		{Name: "id", Value: "0000000000000002"},
	})

	// When
	err := h.GetUserWorldsendSongRecord(c)

	// Then
	apiErr, ok := err.(*apierror.APIError)
	assert.True(t, ok)
	assert.Equal(t, apierror.CodeSongNotFound, apiErr.Code)
	mockUsecase.AssertExpectations(t)
}

func TestUserHandler_GetUserUpdatedAt(t *testing.T) {
	e := newTestEcho()
	mockUsecase := new(mockUserUsecase)
	h := api_internal.NewUserHandler(mockUsecase)
	now := time.Date(2026, 4, 18, 12, 34, 56, 0, time.UTC)

	t.Run("正常系: updated_at を返す", func(t *testing.T) {
		mockUsecase.On("GetUserUpdatedAt", mock.Anything, "testuser", (*entity.User)(nil)).Return(&usecase.UserUpdatedAtOutput{
			UpdatedAt: &now,
		}, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/testuser/updated-at", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

		err := h.GetUserUpdatedAt(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		var body dto_internal.UserUpdatedAtDTO
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		if assert.NotNil(t, body.UpdatedAt) {
			assert.True(t, now.Equal(*body.UpdatedAt))
		}
		mockUsecase.AssertExpectations(t)
	})

	t.Run("プレイヤー未連携時は null を返す", func(t *testing.T) {
		mockUsecase.On("GetUserUpdatedAt", mock.Anything, "testuser", (*entity.User)(nil)).Return(&usecase.UserUpdatedAtOutput{
			UpdatedAt: nil,
		}, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/testuser/updated-at", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

		err := h.GetUserUpdatedAt(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		// キーの存在と null をまとめて固定する（omitempty 付与やキー欠落の退行を検知する）。
		assert.JSONEq(t, `{"updated_at":null}`, rec.Body.String())
		mockUsecase.AssertExpectations(t)
	})

	t.Run("異常系: ユースケースエラーを変換する", func(t *testing.T) {
		mockUsecase.On("GetUserUpdatedAt", mock.Anything, "testuser", (*entity.User)(nil)).Return((*usecase.UserUpdatedAtOutput)(nil), usecase.ErrUserNotFound).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/testuser/updated-at", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

		err := h.GetUserUpdatedAt(c)

		assert.ErrorIs(t, err, apierror.ErrUserNotFound)
		mockUsecase.AssertExpectations(t)
	})
}

func TestUserHandler_GetUserProfileWithRecords(t *testing.T) {
	e := newTestEcho()
	mockUsecase := new(mockUserUsecase)
	h := api_internal.NewUserHandler(mockUsecase)
	now := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	player := testUserPlayerOutput(t, "player", 10, now)
	records := &usecase.UserRecordOutput{
		UpdatedAt:     now,
		Best:          []*usecase.PlayerRecordOutput{testPlayerRecordOutput("best1")},
		BestCandidate: []*usecase.PlayerRecordOutput{testPlayerRecordOutput("best_candidate1")},
		New:           []*usecase.PlayerRecordOutput{testPlayerRecordOutput("new1")},
		NewCandidate:  []*usecase.PlayerRecordOutput{testPlayerRecordOutput("new_candidate1")},
		All:           []*usecase.PlayerRecordOutput{testPlayerRecordOutput("standard1")},
		WorldsEnd:     []*usecase.WorldsendRecordOutput{testWorldsendRecordOutput("we1")},
	}
	result := &usecase.UserProfileWithRecordsOutput{
		Username:  "testuser",
		Player:    player,
		Records:   records,
		UpdatedAt: &now,
	}
	ratingRecords := &usecase.UserRatingRecordOutput{
		UpdatedAt:     now,
		Best:          []*usecase.PlayerRecordOutput{testPlayerRecordOutput("best1")},
		BestCandidate: []*usecase.PlayerRecordOutput{testPlayerRecordOutput("best_candidate1")},
		New:           []*usecase.PlayerRecordOutput{testPlayerRecordOutput("new1")},
		NewCandidate:  []*usecase.PlayerRecordOutput{testPlayerRecordOutput("new_candidate1")},
	}
	ratingResult := &usecase.UserProfileRatingViewOutput{
		Username:  "testuser",
		Player:    player,
		Records:   ratingRecords,
		UpdatedAt: &now,
	}

	t.Run("viewなしは全レコードを返す", func(t *testing.T) {
		mockUsecase.On("GetUserProfileWithRecords", mock.Anything, "testuser", (*entity.User)(nil)).Return(result, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/testuser", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

		err := h.GetUserProfileWithRecords(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		var body map[string]any
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		recordsBody, ok := body["records"].(map[string]any)
		assert.True(t, ok)
		_, hasStandard := recordsBody["standard"]
		_, hasWorldsend := recordsBody["worldsend"]
		assert.True(t, hasStandard)
		assert.True(t, hasWorldsend)
		// 各枠の配列に対応する識別値が入っていることを確かめる（best/new の代入入れ替えを検知する）。
		assertRecordID := func(key, expectedID string) {
			t.Helper()
			arr, ok := recordsBody[key].([]any)
			if assert.True(t, ok) && assert.Len(t, arr, 1) {
				first, ok := arr[0].(map[string]any)
				if assert.True(t, ok) {
					assert.Equal(t, expectedID, first["id"])
				}
			}
		}
		assertRecordID("best", "best1")
		assertRecordID("best_candidate", "best_candidate1")
		assertRecordID("new", "new1")
		assertRecordID("new_candidate", "new_candidate1")
		assertRecordID("standard", "standard1")
		assertRecordID("worldsend", "we1")
		// 連携済みでコース補完がない場合は course が空配列になることを確かめる（null やキー欠落との区別）。
		course, ok := recordsBody["course"].([]any)
		if assert.True(t, ok) {
			assert.Len(t, course, 0)
		}
		mockUsecase.AssertExpectations(t)
	})

	t.Run("viewなしでプレイヤー未連携なら player と records は null を返す", func(t *testing.T) {
		noPlayerResult := &usecase.UserProfileWithRecordsOutput{
			Username:  "testuser",
			Player:    nil,
			Records:   nil,
			UpdatedAt: nil,
		}
		mockUsecase.On("GetUserProfileWithRecords", mock.Anything, "testuser", (*entity.User)(nil)).Return(noPlayerResult, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/testuser", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

		err := h.GetUserProfileWithRecords(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		// キーの存在と null をまとめて固定する（omitempty 付与やキー欠落の退行を検知する）。
		assert.JSONEq(t, `{"username":"testuser","player":null,"records":null,"updated_at":null}`, rec.Body.String())
		mockUsecase.AssertExpectations(t)
	})

	t.Run("view=ratingはレーティング枠のみ返す", func(t *testing.T) {
		mockUsecase.On("GetUserProfileRatingView", mock.Anything, "testuser", (*entity.User)(nil)).Return(ratingResult, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/testuser?view=rating", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

		err := h.GetUserProfileWithRecords(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		var body map[string]any
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		recordsBody, ok := body["records"].(map[string]any)
		assert.True(t, ok)
		_, hasStandard := recordsBody["standard"]
		_, hasWorldsend := recordsBody["worldsend"]
		assert.False(t, hasStandard)
		assert.False(t, hasWorldsend)
		// レーティング枠の配列に対応する識別値が入っていることを確かめる（best/new の代入入れ替えを検知する）。
		assertRecordID := func(key, expectedID string) {
			t.Helper()
			arr, ok := recordsBody[key].([]any)
			if assert.True(t, ok) && assert.Len(t, arr, 1) {
				first, ok := arr[0].(map[string]any)
				if assert.True(t, ok) {
					assert.Equal(t, expectedID, first["id"])
				}
			}
		}
		assertRecordID("best", "best1")
		assertRecordID("best_candidate", "best_candidate1")
		assertRecordID("new", "new1")
		assertRecordID("new_candidate", "new_candidate1")
		mockUsecase.AssertExpectations(t)
	})

	t.Run("view=ratingの異常系", func(t *testing.T) {
		testCases := []struct {
			name          string
			usecaseError  error
			expectedError error
		}{
			{
				name:          "ユーザーが存在しない",
				usecaseError:  usecase.ErrUserNotFound,
				expectedError: apierror.ErrUserNotFound,
			},
			{
				name:          "ユーザーが非公開",
				usecaseError:  usecase.ErrUserPrivate,
				expectedError: apierror.ErrUserNotFound,
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				mockUsecase.On("GetUserProfileRatingView", mock.Anything, "testuser", (*entity.User)(nil)).Return((*usecase.UserProfileRatingViewOutput)(nil), testCase.usecaseError).Once()

				req := httptest.NewRequest(http.MethodGet, "/users/testuser?view=rating", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

				err := h.GetUserProfileWithRecords(c)

				assert.ErrorIs(t, err, testCase.expectedError)
				mockUsecase.AssertExpectations(t)
			})
		}
	})
}

func TestUserHandler_GetUserRating(t *testing.T) {
	e := newTestEcho()
	mockUsecase := new(mockUserUsecase)
	h := api_internal.NewUserHandler(mockUsecase)
	now := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	calculatedRating := 17.1234
	bestAverageRating := 17.2345
	newAverageRating := 16.9567
	player := testUserPlayerOutput(t, "player", 10, now)
	player.CalculatedRating, player.BestAverageRating, player.NewAverageRating = &calculatedRating, &bestAverageRating, &newAverageRating
	ratingResult := &usecase.UserProfileRatingViewOutput{
		Username: "testuser",
		Player:   player,
		Records: &usecase.UserRatingRecordOutput{
			UpdatedAt:     now,
			Best:          []*usecase.PlayerRecordOutput{testPlayerRecordOutput("best1")},
			BestCandidate: []*usecase.PlayerRecordOutput{testPlayerRecordOutput("best_candidate1")},
			New:           []*usecase.PlayerRecordOutput{testPlayerRecordOutput("new1")},
			NewCandidate:  []*usecase.PlayerRecordOutput{testPlayerRecordOutput("new_candidate1")},
		},
		UpdatedAt: &now,
	}

	t.Run("正常系: レーティング枠とメタ情報を返す", func(t *testing.T) {
		mockUsecase.On("GetUserProfileRatingView", mock.Anything, "testuser", (*entity.User)(nil)).Return(ratingResult, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/testuser/rating", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

		err := h.GetUserRating(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		var body map[string]any
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Len(t, body["best"], 1)
		assert.Len(t, body["best_candidate"], 1)
		assert.Len(t, body["new"], 1)
		assert.Len(t, body["new_candidate"], 1)
		assert.Equal(t, calculatedRating, body["rating"])
		assert.Equal(t, bestAverageRating, body["best_average"])
		assert.Equal(t, newAverageRating, body["new_average"])
		meta, ok := body["meta"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, now.Format(time.RFC3339), meta["updated_at"])
		_, hasUsername := body["username"]
		assert.False(t, hasUsername)
		_, hasPlayer := body["player"]
		assert.False(t, hasPlayer)
		// 各枠の配列に対応する識別値が入っていることを確かめる（best/new の代入入れ替えを検知する）。
		assertRecordID := func(key, expectedID string) {
			t.Helper()
			arr, ok := body[key].([]any)
			if assert.True(t, ok) && assert.Len(t, arr, 1) {
				first, ok := arr[0].(map[string]any)
				if assert.True(t, ok) {
					assert.Equal(t, expectedID, first["id"])
				}
			}
		}
		assertRecordID("best", "best1")
		assertRecordID("best_candidate", "best_candidate1")
		assertRecordID("new", "new1")
		assertRecordID("new_candidate", "new_candidate1")
		mockUsecase.AssertExpectations(t)
	})

	t.Run("プレイヤー未連携時は空配列とnullのupdated_atを返す", func(t *testing.T) {
		mockUsecase.On("GetUserProfileRatingView", mock.Anything, "testuser", (*entity.User)(nil)).Return(&usecase.UserProfileRatingViewOutput{
			Username:  "testuser",
			Player:    nil,
			Records:   nil,
			UpdatedAt: nil,
		}, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/testuser/rating", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

		err := h.GetUserRating(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		// キーの存在、`[]` と `null` の区別をまとめて固定する（空スライス初期化の削除や omitempty 付与の退行を検知する）。
		assert.JSONEq(t, `{"rating":null,"best_average":null,"new_average":null,"best":[],"best_candidate":[],"new":[],"new_candidate":[],"meta":{"updated_at":null}}`, rec.Body.String())
		mockUsecase.AssertExpectations(t)
	})

	t.Run("異常系: ユースケースエラーを変換する", func(t *testing.T) {
		mockUsecase.On("GetUserProfileRatingView", mock.Anything, "testuser", (*entity.User)(nil)).Return((*usecase.UserProfileRatingViewOutput)(nil), usecase.ErrUserNotFound).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/testuser/rating", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

		err := h.GetUserRating(c)

		assert.ErrorIs(t, err, apierror.ErrUserNotFound)
		mockUsecase.AssertExpectations(t)
	})
}

func TestUserHandler_GetUserRecord(t *testing.T) {
	e := newTestEcho()
	mockUsecase := new(mockUserUsecase)
	h := api_internal.NewUserHandler(mockUsecase)
	now := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	recordResult := &usecase.UserProfileRecordViewOutput{
		Username: "testuser",
		Player:   testUserPlayerOutput(t, "player", 10, now),
		Records: &usecase.UserRecordViewOutput{
			UpdatedAt: now,
			All:       []*usecase.PlayerRecordOutput{testPlayerRecordOutput("standard1")},
			Worldsend: []*usecase.WorldsendRecordOutput{testWorldsendRecordOutput("we1")},
		},
		UpdatedAt: &now,
	}

	t.Run("正常系: レコード枠とメタ情報を返す", func(t *testing.T) {
		mockUsecase.On("GetUserProfileRecordView", mock.Anything, "testuser", (*entity.User)(nil)).Return(recordResult, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/testuser/record", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

		err := h.GetUserRecord(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		var body map[string]any
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Len(t, body["standard"], 1)
		assert.Len(t, body["worldsend"], 1)
		meta, ok := body["meta"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, now.Format(time.RFC3339), meta["updated_at"])
		_, hasUsername := body["username"]
		assert.False(t, hasUsername)
		_, hasPlayer := body["player"]
		assert.False(t, hasPlayer)
		// 各枠の配列に対応する識別値が入っていることを確かめる（standard/worldsend の取り違えを検知する）。
		standard, ok := body["standard"].([]any)
		if assert.True(t, ok) && assert.Len(t, standard, 1) {
			first, ok := standard[0].(map[string]any)
			if assert.True(t, ok) {
				assert.Equal(t, "standard1", first["id"])
			}
		}
		worldsend, ok := body["worldsend"].([]any)
		if assert.True(t, ok) && assert.Len(t, worldsend, 1) {
			first, ok := worldsend[0].(map[string]any)
			if assert.True(t, ok) {
				assert.Equal(t, "we1", first["id"])
			}
		}
		mockUsecase.AssertExpectations(t)
	})

	t.Run("プレイヤー未連携時は空配列とnullのupdated_atを返す", func(t *testing.T) {
		mockUsecase.On("GetUserProfileRecordView", mock.Anything, "testuser", (*entity.User)(nil)).Return(&usecase.UserProfileRecordViewOutput{
			Username:  "testuser",
			Player:    nil,
			Records:   nil,
			UpdatedAt: nil,
		}, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/testuser/record", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

		err := h.GetUserRecord(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		// キーの存在と `[]`・`null` の区別を項目ごとに固定する（空スライス初期化の削除や omitempty 付与の退行を検知する）。
		// course は未連携時に null を返すか [] を返すかの仕様が未確定のため、ここでは固定しない。
		var body map[string]json.RawMessage
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.JSONEq(t, `[]`, string(body["standard"]))
		assert.JSONEq(t, `[]`, string(body["worldsend"]))
		assert.JSONEq(t, `{"updated_at":null}`, string(body["meta"]))
		mockUsecase.AssertExpectations(t)
	})

	t.Run("異常系: ユースケースエラーを変換する", func(t *testing.T) {
		mockUsecase.On("GetUserProfileRecordView", mock.Anything, "testuser", (*entity.User)(nil)).Return((*usecase.UserProfileRecordViewOutput)(nil), usecase.ErrUserNotFound).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/testuser/record", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

		err := h.GetUserRecord(c)

		assert.ErrorIs(t, err, apierror.ErrUserNotFound)
		mockUsecase.AssertExpectations(t)
	})
}

func TestUserHandler_GetUserProfileWithRecordView(t *testing.T) {
	e := newTestEcho()
	mockUsecase := new(mockUserUsecase)
	h := api_internal.NewUserHandler(mockUsecase)
	now := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	player := testUserPlayerOutput(t, "player", 10, now)
	recordViewResult := &usecase.UserProfileRecordViewOutput{
		Username: "testuser",
		Player:   player,
		Records: &usecase.UserRecordViewOutput{
			UpdatedAt: now,
			All:       []*usecase.PlayerRecordOutput{testPlayerRecordOutput("standard1")},
			Worldsend: []*usecase.WorldsendRecordOutput{testWorldsendRecordOutput("we1")},
		},
		UpdatedAt: &now,
	}

	mockUsecase.On("GetUserProfileRecordView", mock.Anything, "testuser", (*entity.User)(nil)).Return(recordViewResult, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/users/testuser?view=record", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

	err := h.GetUserProfileWithRecords(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	recordsBody, ok := body["records"].(map[string]any)
	assert.True(t, ok)
	_, hasStandard := recordsBody["standard"]
	_, hasWorldsend := recordsBody["worldsend"]
	_, hasBest := recordsBody["best"]
	_, hasNew := recordsBody["new"]
	assert.True(t, hasStandard)
	assert.True(t, hasWorldsend)
	assert.False(t, hasBest)
	assert.False(t, hasNew)
	// 各枠の配列に対応する識別値が入っていることを確かめる（standard/worldsend の取り違えを検知する）。
	standard, ok := recordsBody["standard"].([]any)
	if assert.True(t, ok) && assert.Len(t, standard, 1) {
		first, ok := standard[0].(map[string]any)
		if assert.True(t, ok) {
			assert.Equal(t, "standard1", first["id"])
		}
	}
	worldsend, ok := recordsBody["worldsend"].([]any)
	if assert.True(t, ok) && assert.Len(t, worldsend, 1) {
		first, ok := worldsend[0].(map[string]any)
		if assert.True(t, ok) {
			assert.Equal(t, "we1", first["id"])
		}
	}
	mockUsecase.AssertExpectations(t)
}

func TestUserHandler_GetUserProfileWithRecords_不正なviewは検証エラー(t *testing.T) {
	// Given
	e := newTestEcho()
	mockUsecase := new(mockUserUsecase)
	h := api_internal.NewUserHandler(mockUsecase)
	req := httptest.NewRequest(http.MethodGet, "/users/testuser?view=all", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

	// When
	err := h.GetUserProfileWithRecords(c)

	// Then: 全件表示へフォールバックせず、取得処理を呼ばない
	var apiErr *apierror.APIError
	if assert.ErrorAs(t, err, &apiErr) {
		assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
	}
	mockUsecase.AssertNotCalled(t, "GetUserProfileWithRecords", mock.Anything, mock.Anything, mock.Anything)
}

func TestAdminUserHandler_GetAllUsers(t *testing.T) {
	e := newTestEcho()
	createdAt := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	updatedAt := createdAt.Add(2 * time.Hour)

	expected := []usecase.AdminUserOutput{
		{
			UserName:       "user1",
			AccountType:    "EDITOR",
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
			PlayerName:     new("player1"),
			Rating:         new(17.25),
			OverPowerValue: new(float64(9500)),
			IsSuspicious:   true,
			IsPrivate:      false,
		},
		{
			UserName:       "user2",
			AccountType:    "PLAYER",
			CreatedAt:      createdAt.Add(time.Hour),
			UpdatedAt:      updatedAt.Add(time.Hour),
			PlayerName:     nil,
			Rating:         nil,
			OverPowerValue: nil,
			IsSuspicious:   false,
			IsPrivate:      true,
		},
	}

	// page クエリの解釈（未指定・変換不可・0以下は 1 に補正）を表形式で検証する。
	tests := []struct {
		name string
		// Given: リクエストターゲット
		target string
		// Then: usecase へ渡されることを期待する page の固定値
		expectedPage int
	}{
		{
			name:         "page未指定は1で呼ぶ",
			target:       "/internal/users?name=user",
			expectedPage: 1,
		},
		{
			name:         "page=0は1で呼ぶ",
			target:       "/internal/users?page=0&name=user",
			expectedPage: 1,
		},
		{
			name:         "page=-1は1で呼ぶ",
			target:       "/internal/users?page=-1&name=user",
			expectedPage: 1,
		},
		{
			name:         "pageが数値でないときは1で呼ぶ",
			target:       "/internal/users?page=abc&name=user",
			expectedPage: 1,
		},
		{
			name:         "page=1は1で呼ぶ",
			target:       "/internal/users?page=1&name=user",
			expectedPage: 1,
		},
		{
			name:         "page=2は2で呼ぶ",
			target:       "/internal/users?page=2&name=user",
			expectedPage: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			mockUsecase := new(mockUserUsecase)
			h := api_internal.NewAdminUserHandler(mockUsecase)
			// page は具体的な値で照合し、補正なしの値が渡されたらテストが落ちるようにする。
			mockUsecase.On("GetAllUsersForAdmin", mock.Anything, tt.expectedPage, info.DefaultUserListLimit, "user").Return(expected, nil).Once()
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// When
			err := h.GetAllUsers(c)

			// Then: page の値によらず同じ一覧をそのまま返す
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)
			mockUsecase.AssertExpectations(t)

			var body []map[string]any
			assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Len(t, body, 2)
			assert.Equal(t, "user1", body[0]["username"])
			assert.Equal(t, "EDITOR", body[0]["account_type"])
			assert.Equal(t, createdAt.Format(time.RFC3339), body[0]["created_at"])
			assert.Equal(t, updatedAt.Format(time.RFC3339), body[0]["updated_at"])
			assert.Equal(t, "player1", body[0]["player_name"])
			assert.Equal(t, 17.25, body[0]["rating"])
			assert.Equal(t, 9500.0, body[0]["overpower_value"])
			assert.Equal(t, true, body[0]["is_suspicious"])
			assert.Equal(t, false, body[0]["is_private"])
			assert.NotContains(t, body[0], "firebase_uid")
			assert.NotContains(t, body[0], "is_deleted")
			assert.NotContains(t, body[0], "email")
			assert.Equal(t, "user2", body[1]["username"])
			assert.Equal(t, "PLAYER", body[1]["account_type"])
			assert.Equal(t, createdAt.Add(time.Hour).Format(time.RFC3339), body[1]["created_at"])
			assert.Equal(t, updatedAt.Add(time.Hour).Format(time.RFC3339), body[1]["updated_at"])
			assert.Nil(t, body[1]["player_name"])
			assert.Nil(t, body[1]["rating"])
			assert.Nil(t, body[1]["overpower_value"])
			assert.Equal(t, false, body[1]["is_suspicious"])
			assert.Equal(t, true, body[1]["is_private"])
			assert.NotContains(t, body[1], "firebase_uid")
			assert.NotContains(t, body[1], "email")
		})
	}
}

func TestAdminUserHandler_GetAllUsers_クライアント切断は原因エラーを保持する(t *testing.T) {
	// Given
	e := newTestEcho()
	mockUsecase := new(mockUserUsecase)
	h := api_internal.NewAdminUserHandler(mockUsecase)
	mockUsecase.On("GetAllUsersForAdmin", mock.Anything, 1, info.DefaultUserListLimit, "").Return(nil, context.Canceled).Once()
	req := httptest.NewRequest(http.MethodGet, "/internal/users", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// When
	err := h.GetAllUsers(c)

	// Then
	assert.ErrorIs(t, err, apierror.ErrInternalError)
	assert.ErrorIs(t, err, context.Canceled)
	mockUsecase.AssertExpectations(t)
}

func TestUserHandler_DeleteUser(t *testing.T) {
	e := newTestEcho()
	mockUsecase := new(mockUserUsecase)
	h := api_internal.NewUserHandler(mockUsecase)
	adminUser := &entity.User{ID: 99, AccountTypeID: 3}

	t.Run("正常系: ユーザー削除", func(t *testing.T) {
		mockUsecase.On("DeleteUser", mock.Anything, adminUser, "testuser").Return(nil).Once()

		req := httptest.NewRequest(http.MethodDelete, "/users/testuser", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})
		c.Set("userEntity", adminUser)

		err := h.DeleteUser(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		mockUsecase.AssertExpectations(t)
	})

	t.Run("異常系: ユーザーが存在しない", func(t *testing.T) {
		mockUsecase.On("DeleteUser", mock.Anything, adminUser, "nonexistent").Return(usecase.ErrUserNotFound).Once()

		req := httptest.NewRequest(http.MethodDelete, "/users/nonexistent", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "nonexistent"}})
		c.Set("userEntity", adminUser)

		err := h.DeleteUser(c)

		var apiErr *apierror.APIError
		if assert.ErrorAs(t, err, &apiErr) {
			assert.Equal(t, apierror.CodeUserNotFound, apiErr.Code)
			assert.Equal(t, http.StatusNotFound, apiErr.HTTPStatus)
		}
		mockUsecase.AssertExpectations(t)
	})

	t.Run("異常系: ADMIN権限がない", func(t *testing.T) {
		normalUser := &entity.User{ID: 1, AccountTypeID: 1}
		mockUsecase.On("DeleteUser", mock.Anything, normalUser, "testuser").Return(usecase.ErrAdminRequired).Once()

		req := httptest.NewRequest(http.MethodDelete, "/users/testuser", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})
		c.Set("userEntity", normalUser)

		err := h.DeleteUser(c)

		var apiErr *apierror.APIError
		if assert.ErrorAs(t, err, &apiErr) {
			assert.Equal(t, apierror.CodeForbidden, apiErr.Code)
			assert.Equal(t, http.StatusForbidden, apiErr.HTTPStatus)
		}
		mockUsecase.AssertExpectations(t)
	})

	t.Run("異常系: 自分自身は削除できない", func(t *testing.T) {
		mockUsecase.On("DeleteUser", mock.Anything, adminUser, "adminuser").Return(usecase.ErrCannotDeleteSelf).Once()

		req := httptest.NewRequest(http.MethodDelete, "/users/adminuser", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "adminuser"}})
		c.Set("userEntity", adminUser)

		err := h.DeleteUser(c)

		var apiErr *apierror.APIError
		if assert.ErrorAs(t, err, &apiErr) {
			assert.Equal(t, apierror.CodeForbidden, apiErr.Code)
			assert.Equal(t, http.StatusForbidden, apiErr.HTTPStatus)
		}
		mockUsecase.AssertExpectations(t)
	})
}
