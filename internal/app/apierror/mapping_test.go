package apierror

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromUsecaseError_スコア履歴時刻競合を409へ変換する(t *testing.T) {
	apiErr := FromUsecaseError(repository.ErrScoreHistoryTimestampConflict)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.HTTPStatus)
	assert.Equal(t, CodeConflict, apiErr.Code)
}

func TestFromUsecaseError_公式指標履歴時刻競合を409へ変換する(t *testing.T) {
	apiErr := FromUsecaseError(repository.ErrPlayerMetricHistoryTimestampConflict)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.HTTPStatus)
	assert.Equal(t, CodeConflict, apiErr.Code)
}

func TestFromUsecaseError_フレンド比較対象なしを404へ変換する(t *testing.T) {
	apiErr := FromUsecaseError(usecase.ErrFriendNotFound)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.HTTPStatus)
	assert.Equal(t, CodeFriendNotFound, apiErr.Code)
}

func TestFromUsecaseError_ユーザー一覧のpage範囲外を422へ変換する(t *testing.T) {
	apiErr := FromUsecaseError(usecase.ErrInvalidUserListPage)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
	assert.Equal(t, CodeValidationFailed, apiErr.Code)
	assert.ErrorIs(t, apiErr, usecase.ErrInvalidUserListPage)
}

func TestFromUsecaseError_フレンド比較不能を409へ変換する(t *testing.T) {
	apiErr := FromUsecaseError(usecase.ErrFriendScoreComparisonUnavailable)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.HTTPStatus)
	assert.Equal(t, CodeFriendScoreComparisonUnavailable, apiErr.Code)
}

func TestFromUsecaseError_公式指標履歴なしを404へ変換する(t *testing.T) {
	apiErr := FromUsecaseError(usecase.ErrPlayerMetricHistoryNotFound)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.HTTPStatus)
	assert.Equal(t, CodePlayerMetricHistoryNotFound, apiErr.Code)
}

func TestFromUsecaseError_プレイヤーデータ検証エラーを422へ変換する(t *testing.T) {
	validationErr := &usecase.PlayerDataValidationError{Field: "updated_at", Message: "must be RFC3339"}

	apiErr := FromUsecaseError(validationErr)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
	assert.Equal(t, CodeValidationFailed, apiErr.Code)
	assert.ErrorIs(t, apiErr, validationErr)
}

func TestFromUsecaseError_認証失敗は汎用認証エラーに丸める(t *testing.T) {
	got := FromUsecaseError(usecase.ErrInvalidCredentials)

	assert.Equal(t, CodeInvalidCredentials, got.Code)
	assert.Equal(t, ErrInvalidCredentials.HTTPStatus, got.HTTPStatus)
	assert.Equal(t, usecase.ErrInvalidCredentials, got.Internal)
}

func TestFromUsecaseError_メンテナンス中は専用503へ変換する(t *testing.T) {
	apiErr := FromUsecaseError(usecase.ErrMaintenanceMode)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusServiceUnavailable, apiErr.HTTPStatus)
	assert.Equal(t, CodeMaintenanceMode, apiErr.Code)
	assert.ErrorIs(t, apiErr.Internal, usecase.ErrMaintenanceMode)
}

func TestFromUsecaseError_不正なメンテナンスコメントは400へ変換する(t *testing.T) {
	apiErr := FromUsecaseError(usecase.ErrInvalidMaintenanceComment)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	assert.Equal(t, CodeBadRequest, apiErr.Code)
	assert.ErrorIs(t, apiErr.Internal, usecase.ErrInvalidMaintenanceComment)
}

func TestFromUsecaseError_禁止ユーザー名は登録失敗に丸める(t *testing.T) {
	got := FromUsecaseError(usecase.ErrUsernameForbidden)

	assert.Equal(t, CodeRegistrationFailed, got.Code)
	assert.Equal(t, http.StatusBadRequest, got.HTTPStatus)
}

func TestFromUsecaseError_ユーザー名変更エラーを専用レスポンスへ変換する(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		code       string
		httpStatus int
	}{
		{name: "禁止語", err: usecase.ErrUsernameChangeForbidden, code: CodeUsernameForbidden, httpStatus: 422},
		{name: "使用済み", err: usecase.ErrUsernameChangeConflict, code: CodeUsernameTaken, httpStatus: 409},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromUsecaseError(tt.err)
			assert.Equal(t, tt.code, got.Code)
			assert.Equal(t, tt.httpStatus, got.HTTPStatus)
		})
	}
}

func TestFromUsecaseError_Firebaseアカウント未変更を400に変換する(t *testing.T) {
	apiErr := FromUsecaseError(usecase.ErrFirebaseAccountUnchanged)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	assert.Equal(t, CodeFirebaseAccountUnchanged, apiErr.Code)
}

func TestFromUsecaseError_お気に入り上限エラーを400に変換する(t *testing.T) {
	apiErr := FromUsecaseError(usecase.ErrPlayerFavoriteSongLimitExceeded)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	assert.Equal(t, CodeFavoriteSongLimitExceeded, apiErr.Code)
}

func TestFromUsecaseError_目標並び順不正を400に変換する(t *testing.T) {
	apiErr := FromUsecaseError(usecase.ErrInvalidGoalOrder)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	assert.Equal(t, CodeGoalInvalidOrder, apiErr.Code)
}

func TestFromUsecaseError_目標グループ名重複を409に変換する(t *testing.T) {
	apiErr := FromUsecaseError(usecase.ErrGoalGroupConflict)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.HTTPStatus)
	assert.Equal(t, CodeGoalGroupConflict, apiErr.Code)
}

func TestFromUsecaseError_auth_time欠落は詳細を伏せてrecent_sign_in_requiredに丸める(t *testing.T) {
	err := errors.Join(usecase.ErrRecentSignInAuthTimeMissing, errors.New("firebase token auth_time is empty"))

	got := FromUsecaseError(err)

	assert.Equal(t, CodeRecentSignInRequired, got.Code)
	assert.Equal(t, ErrRecentSignInRequired.HTTPStatus, got.HTTPStatus)
	assert.ErrorIs(t, got.Internal, usecase.ErrRecentSignInAuthTimeMissing)
}

func TestFromUsecaseError_不正なコース入力をバリデーションエラーへ変換する(t *testing.T) {
	apiErr := FromUsecaseError(usecase.ErrInvalidCourseInput)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
	assert.Equal(t, CodeValidationFailed, apiErr.Code)
}

func TestFromUsecaseError_コース未検出を404へ変換する(t *testing.T) {
	apiErr := FromUsecaseError(usecase.ErrCourseNotFound)

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.HTTPStatus)
}

func TestFromUsecaseError_APIトークン管理エラーを変換する(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "ID不正", err: usecase.ErrInvalidAPITokenID, wantStatus: http.StatusBadRequest, wantCode: CodeInvalidAPITokenID},
		{name: "名前不正", err: usecase.ErrInvalidAPITokenName, wantStatus: http.StatusBadRequest, wantCode: CodeInvalidAPITokenName},
		{name: "権限不正", err: usecase.ErrInvalidAPITokenPermission, wantStatus: http.StatusBadRequest, wantCode: CodeInvalidAPITokenPermission},
		{name: "書き込み権限不足", err: usecase.ErrAPITokenWritePermissionDenied, wantStatus: http.StatusForbidden, wantCode: CodeForbidden},
		{name: "上限超過", err: usecase.ErrAPITokenLimitExceeded, wantStatus: http.StatusBadRequest, wantCode: CodeAPITokenLimitExceeded},
		{name: "名前重複", err: usecase.ErrAPITokenNameConflict, wantStatus: http.StatusConflict, wantCode: CodeAPITokenNameConflict},
		{name: "未検出", err: usecase.ErrAPITokenNotFound, wantStatus: http.StatusNotFound, wantCode: CodeAPITokenNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := FromUsecaseError(tt.err)

			require.NotNil(t, apiErr)
			assert.Equal(t, tt.wantStatus, apiErr.HTTPStatus)
			assert.Equal(t, tt.wantCode, apiErr.Code)
		})
	}
}

func TestFromUsecaseError_データ移行エラーを専用HTTPエラーへ変換する(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantCode   string
		wantStatus int
	}{
		{name: "プレイヤーなし", err: usecase.ErrDataTransferPlayerNotFound, wantCode: CodeDataTransferPlayerNotFound, wantStatus: http.StatusBadRequest},
		{name: "ファイル不正", err: usecase.ErrDataTransferInvalidFile, wantCode: CodeDataTransferInvalidFile, wantStatus: http.StatusBadRequest},
		{name: "署名不正", err: usecase.ErrDataTransferInvalidSignature, wantCode: CodeDataTransferInvalidSignature, wantStatus: http.StatusBadRequest},
		{name: "スキーマ非対応", err: usecase.ErrDataTransferUnsupportedSchema, wantCode: CodeDataTransferUnsupportedSchema, wantStatus: http.StatusBadRequest},
		{name: "データ不正", err: usecase.ErrDataTransferInvalidData, wantCode: CodeDataTransferInvalidData, wantStatus: http.StatusBadRequest},
		{name: "参照未解決", err: usecase.ErrDataTransferUnresolvedReference, wantCode: CodeDataTransferUnresolvedReference, wantStatus: http.StatusBadRequest},
		{name: "移行先に既存データあり", err: usecase.ErrDataTransferDestinationNotEmpty, wantCode: CodeDataTransferDestinationNotEmpty, wantStatus: http.StatusConflict},
		{name: "サイズ超過", err: usecase.ErrDataTransferPayloadTooLarge, wantCode: CodePayloadTooLarge, wantStatus: http.StatusRequestEntityTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := FromUsecaseError(tt.err)

			require.NotNil(t, apiErr)
			assert.Equal(t, tt.wantCode, apiErr.Code)
			assert.Equal(t, tt.wantStatus, apiErr.HTTPStatus)
			assert.ErrorIs(t, apiErr, tt.err)
		})
	}
}

func TestFromUsecaseError_プレイヤーデータの独自エラーは原因エラーより優先する(t *testing.T) {
	tests := []struct {
		name string
		// Given: 対応表にある sentinel を原因エラーとして保持する独自エラー
		err error
		// Then: 独自エラーの分類で変換される
		wantStatus int
		wantCode   string
	}{
		{
			name:       "検証エラー",
			err:        &usecase.PlayerDataValidationError{Field: "name", Message: "invalid", Err: usecase.ErrInternalError},
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   CodeValidationFailed,
		},
		{
			name:       "競合エラー",
			err:        &usecase.PlayerDataConflictError{Reason: "conflict", Err: repository.ErrSongNotFound},
			wantStatus: http.StatusConflict,
			wantCode:   CodeConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			apiErr := FromUsecaseError(tt.err)

			// Then
			require.NotNil(t, apiErr)
			assert.Equal(t, tt.wantStatus, apiErr.HTTPStatus)
			assert.Equal(t, tt.wantCode, apiErr.Code)
		})
	}
}

func TestFromUsecaseError_Xアカウント連携のエラーを変換する(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "X未連携は409", err: usecase.ErrXAccountNotLinked, wantStatus: http.StatusConflict, wantCode: CodeXAccountNotLinked},
		{name: "Xのトークン拒否はセッション切れと区別して400", err: usecase.ErrInvalidXCredentials, wantStatus: http.StatusBadRequest, wantCode: CodeInvalidXCredentials},
		{name: "X APIの取得失敗は502", err: errors.Join(usecase.ErrXAPIUnavailable, errors.New("status 503")), wantStatus: http.StatusBadGateway, wantCode: CodeXAPIUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			apiErr := FromUsecaseError(tt.err)

			// Then
			require.NotNil(t, apiErr)
			assert.Equal(t, tt.wantStatus, apiErr.HTTPStatus)
			assert.Equal(t, tt.wantCode, apiErr.Code)
		})
	}
}

func TestFromUsecaseError_プロフィールコメントの内容判定エラーを変換する(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "不適切と判定されたコメントは400", err: usecase.ErrCommentInappropriate, wantStatus: http.StatusBadRequest, wantCode: CodeCommentInappropriate},
		{name: "判定できない場合は502", err: fmt.Errorf("%w: %w", usecase.ErrCommentModerationUnavailable, errors.New("status 503")), wantStatus: http.StatusBadGateway, wantCode: CodeCommentModerationUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			apiErr := FromUsecaseError(tt.err)

			// Then
			require.NotNil(t, apiErr)
			assert.Equal(t, tt.wantStatus, apiErr.HTTPStatus)
			assert.Equal(t, tt.wantCode, apiErr.Code)
			assert.ErrorIs(t, apiErr, tt.err)
		})
	}
}

func TestFromUsecaseError_非公開ユーザーは存在を隠すため404へ丸める(t *testing.T) {
	// Given: 非公開ユーザーを表すエラー（ユーザー列挙を避けるため存在しない場合と同じ応答にする）
	err := usecase.ErrUserPrivate

	// When
	apiErr := FromUsecaseError(err)

	// Then: 存在しない場合と同じ 404 user_not_found になり、原因エラーを保持する
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.HTTPStatus)
	assert.Equal(t, CodeUserNotFound, apiErr.Code)
	assert.ErrorIs(t, apiErr, usecase.ErrUserPrivate)
}

func TestFromUsecaseError_使用済みユーザー名は登録失敗に丸める(t *testing.T) {
	// Given: ユーザー名が既に使われていることを表すエラー（存在を悟らせないため詳細を隠す）
	err := usecase.ErrUsernameTaken

	// When
	apiErr := FromUsecaseError(err)

	// Then: 400 registration_failed に丸められ、原因エラーを保持する
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	assert.Equal(t, CodeRegistrationFailed, apiErr.Code)
	assert.ErrorIs(t, apiErr, usecase.ErrUsernameTaken)
}

func TestFromUsecaseError_未知のエラーは500へ丸める(t *testing.T) {
	// Given: 対応表にない未知のエラー
	err := errors.New("unknown error for mapping test")

	// When
	apiErr := FromUsecaseError(err)

	// Then: 500 internal_error になり、原因エラーを errors.Is で辿れる
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusInternalServerError, apiErr.HTTPStatus)
	assert.Equal(t, CodeInternalError, apiErr.Code)
	assert.ErrorIs(t, apiErr, err)
}

func TestFromUsecaseError_nilはnilを返す(t *testing.T) {
	// Given: nil エラー

	// When
	apiErr := FromUsecaseError(nil)

	// Then: 変換せず nil を返す
	assert.Nil(t, apiErr)
}

func TestFromUsecaseError_プレイヤーデータ未検出を400へ変換する(t *testing.T) {
	// Given: マスタ未登録などを表す見つからない系の独自エラー
	err := &usecase.PlayerDataNotFoundError{Resource: "song", Key: "99999"}

	// When
	apiErr := FromUsecaseError(err)

	// Then: 400 resource_not_found になり、原因エラーを保持する
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	assert.Equal(t, CodeResourceNotFound, apiErr.Code)
	assert.ErrorIs(t, apiErr, err)
}

func TestFromUsecaseError_譜面統計バッチ関連エラー(t *testing.T) {
	tests := []struct {
		name string
		// Given: 対応表にある譜面統計バッチ系のエラー
		err error
		// Then: 期待する対応付け
		wantStatus int
		wantCode   string
	}{
		{
			name:       "停止処理中は503",
			err:        usecase.ErrChartStatsBatchUnavailable,
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   CodeServiceUnavailable,
		},
		{
			name:       "実行中は409",
			err:        usecase.ErrChartStatsBatchAlreadyRunning,
			wantStatus: http.StatusConflict,
			wantCode:   CodeChartStatsBatchAlreadyRunning,
		},
		{
			name:       "ジョブID不正は400",
			err:        usecase.ErrInvalidChartStatsBatchJobID,
			wantStatus: http.StatusBadRequest,
			wantCode:   CodeInvalidChartStatsBatchJobID,
		},
		{
			name:       "ジョブ未検出は404",
			err:        repository.ErrChartStatsBatchJobNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   CodeChartStatsBatchJobNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			apiErr := FromUsecaseError(tt.err)

			// Then
			require.NotNil(t, apiErr)
			assert.Equal(t, tt.wantStatus, apiErr.HTTPStatus)
			assert.Equal(t, tt.wantCode, apiErr.Code)
			assert.ErrorIs(t, apiErr, tt.err)
		})
	}
}

func TestFromUsecaseError_エラーコードとステータスの対応(t *testing.T) {
	// エラーコードとHTTPステータスはAPIの契約なので、他のテストで個別に扱っていない分岐を1つの表で固定する。
	// 期待値（ステータス・コード）は docs/API.md と docs/error_code_reason_codes.md から求めた固定値で書く。
	// usecase.ErrInvalidWorldsendInput は、POST の docs（400）と本番（422）が食い違っているため、仕様が決まるまで固定しない。
	tests := []struct {
		name string
		// Given: FromUsecaseError への直接入力
		err error
		// Then: 期待するHTTPステータスとエラーコード
		wantStatus int
		wantCode   string
	}{
		// リポジトリ層のエラー
		{name: "楽曲未検出はsong_not_foundの404", err: repository.ErrSongNotFound, wantStatus: 404, wantCode: "song_not_found"},
		{name: "公式インデックス重複はduplicate_official_idxの409", err: repository.ErrDuplicateOfficialIdx, wantStatus: 409, wantCode: "duplicate_official_idx"},
		{name: "リポジトリ側のコース未検出も汎用not_foundの404へ丸める", err: repository.ErrCourseNotFound, wantStatus: 404, wantCode: "not_found"},
		{name: "未知のコースクラスはvalidation_failedの422へ丸める", err: repository.ErrCourseClassNotFound, wantStatus: 422, wantCode: "validation_failed"},
		{name: "称号未検出は汎用not_foundの404へ丸める", err: repository.ErrHonorNotFound, wantStatus: 404, wantCode: "not_found"},
		{name: "称号競合は汎用conflictの409へ丸める", err: repository.ErrHonorConflict, wantStatus: 409, wantCode: "conflict"},
		{name: "バージョン未検出はversion_not_foundの404", err: repository.ErrVersionNotFound, wantStatus: 404, wantCode: "version_not_found"},
		{name: "バージョン名重複はversion_name_conflictの409へ丸める", err: repository.ErrVersionConflict, wantStatus: 409, wantCode: "version_name_conflict"},
		// 認証・認可系
		{name: "IDトークン不正はinvalid_tokenの401へ丸める", err: usecase.ErrInvalidIDToken, wantStatus: 401, wantCode: "invalid_token"},
		{name: "APIトークン不正もinvalid_tokenの401へ丸める", err: usecase.ErrInvalidAPIToken, wantStatus: 401, wantCode: "invalid_token"},
		{name: "Turnstileトークン不正はinvalid_turnstile_tokenの401", err: usecase.ErrInvalidTurnstileToken, wantStatus: 401, wantCode: "invalid_turnstile_token"},
		{name: "連携済みFirebase UIDはfirebase_uid_already_linkedの409", err: usecase.ErrFirebaseUIDAlreadyLinked, wantStatus: 409, wantCode: "firebase_uid_already_linked"},
		{name: "再認証要求自体もrecent_sign_in_requiredの401", err: usecase.ErrRecentSignInRequired, wantStatus: 401, wantCode: "recent_sign_in_required"},
		{name: "管理者権限不足はforbiddenの403へ丸める", err: usecase.ErrAdminRequired, wantStatus: 403, wantCode: "forbidden"},
		{name: "自分自身の削除もforbiddenの403へ丸める", err: usecase.ErrCannotDeleteSelf, wantStatus: 403, wantCode: "forbidden"},
		// ユーザー・プレイヤー系
		{name: "ユーザー未検出はuser_not_foundの404", err: usecase.ErrUserNotFound, wantStatus: 404, wantCode: "user_not_found"},
		{name: "プレイヤー未連携はplayer_not_linkedの404", err: usecase.ErrPlayerNotLinked, wantStatus: 404, wantCode: "player_not_linked"},
		{name: "操作失敗はoperation_failedの400", err: usecase.ErrOperationFailed, wantStatus: 400, wantCode: "operation_failed"},
		{name: "内部エラーはinternal_errorの500", err: usecase.ErrInternalError, wantStatus: 500, wantCode: "internal_error"},
		// 楽曲・譜面系
		{name: "難易度不正はinvalid_difficultyの400", err: usecase.ErrInvalidDifficulty, wantStatus: 400, wantCode: "invalid_difficulty"},
		{name: "譜面未検出はchart_not_foundの404", err: usecase.ErrChartNotFound, wantStatus: 404, wantCode: "chart_not_found"},
		{name: "スコア履歴なしはscore_history_not_foundの404", err: usecase.ErrScoreHistoryNotFound, wantStatus: 404, wantCode: "score_history_not_found"},
		{name: "対象外難易度の履歴参照はscore_history_unsupported_difficultyの400", err: usecase.ErrScoreHistoryUnsupportedDifficulty, wantStatus: 400, wantCode: "score_history_unsupported_difficulty"},
		{name: "レート帯不正はvalidation_failedの422へ丸める", err: usecase.ErrInvalidRatingBand, wantStatus: 422, wantCode: "validation_failed"},
		{name: "ベスト枠カーソル不正はvalidation_failedの422へ丸める", err: usecase.ErrInvalidBestSlotRankingCursor, wantStatus: 422, wantCode: "validation_failed"},
		{name: "称号入力不正はvalidation_failedの422へ丸める", err: usecase.ErrInvalidHonorInput, wantStatus: 422, wantCode: "validation_failed"},
		// バージョン系
		{name: "バージョン入力不正はinvalid_version_inputの422", err: usecase.ErrInvalidVersionInput, wantStatus: 422, wantCode: "invalid_version_input"},
		{name: "最新版でないバージョン指定はversion_not_latestの409", err: usecase.ErrVersionNotLatest, wantStatus: 409, wantCode: "version_not_latest"},
		{name: "使用中バージョンの変更はversion_in_useの409", err: usecase.ErrVersionInUse, wantStatus: 409, wantCode: "version_in_use"},
		// プロフィールコメント・ユーザー名系
		{name: "長すぎるコメントはcomment_too_longの400", err: usecase.ErrCommentTooLong, wantStatus: 400, wantCode: "comment_too_long"},
		{name: "不正文字を含むコメントはcomment_invalid_charの400", err: usecase.ErrCommentInvalidChar, wantStatus: 400, wantCode: "comment_invalid_char"},
		{name: "空のユーザー名はusername_emptyの400", err: usecase.ErrUsernameEmpty, wantStatus: 400, wantCode: "username_empty"},
		{name: "短すぎるユーザー名はusername_too_shortの400", err: usecase.ErrUsernameTooShort, wantStatus: 400, wantCode: "username_too_short"},
		{name: "長すぎるユーザー名はusername_too_longの400", err: usecase.ErrUsernameTooLong, wantStatus: 400, wantCode: "username_too_long"},
		{name: "不正文字のユーザー名はusername_invalid_charの400", err: usecase.ErrUsernameInvalidChar, wantStatus: 400, wantCode: "username_invalid_char"},
		// 目標系
		{name: "目標未検出はgoal_not_foundの404", err: usecase.ErrGoalNotFound, wantStatus: 404, wantCode: "goal_not_found"},
		{name: "目標上限超過はgoal_limit_exceededの400", err: usecase.ErrGoalLimitExceeded, wantStatus: 400, wantCode: "goal_limit_exceeded"},
		{name: "目標入力不正はinvalid_goal_inputの400", err: usecase.ErrInvalidGoalInput, wantStatus: 400, wantCode: "invalid_goal_input"},
		{name: "目標タイトル不正はgoal_invalid_titleの400へ丸める", err: usecase.ErrInvalidGoalTitle, wantStatus: 400, wantCode: "goal_invalid_title"},
		{name: "達成種別不正はgoal_invalid_achievement_typeの400へ丸める", err: usecase.ErrInvalidAchievementType, wantStatus: 400, wantCode: "goal_invalid_achievement_type"},
		{name: "達成条件不正はgoal_invalid_achievement_paramsの400へ丸める", err: usecase.ErrInvalidAchievementParam, wantStatus: 400, wantCode: "goal_invalid_achievement_params"},
		{name: "目標属性不正はgoal_invalid_attributesの400へ丸める", err: usecase.ErrInvalidGoalAttributes, wantStatus: 400, wantCode: "goal_invalid_attributes"},
		{name: "ゲージ開始位置不正はgoal_invalid_gauge_startの400へ丸める", err: usecase.ErrInvalidGoalGaugeStart, wantStatus: 400, wantCode: "goal_invalid_gauge_start"},
		{name: "目標グループ未検出はgoal_group_not_foundの404", err: usecase.ErrGoalGroupNotFound, wantStatus: 404, wantCode: "goal_group_not_found"},
		{name: "目標グループ上限超過はgoal_group_limit_exceededの400", err: usecase.ErrGoalGroupLimitExceeded, wantStatus: 400, wantCode: "goal_group_limit_exceeded"},
		{name: "目標グループ名不正はgoal_group_invalid_nameの400へ丸める", err: usecase.ErrInvalidGoalGroupName, wantStatus: 400, wantCode: "goal_group_invalid_name"},
		{name: "目標グループ並び順不正はgoal_group_invalid_orderの400へ丸める", err: usecase.ErrInvalidGoalGroupOrder, wantStatus: 400, wantCode: "goal_group_invalid_order"},
		// 保存済みフィルタ系
		{name: "フィルタ未検出はrecord_filter_not_foundの404", err: usecase.ErrRecordFilterNotFound, wantStatus: 404, wantCode: "record_filter_not_found"},
		{name: "フィルタ上限超過はrecord_filter_limit_exceededの400", err: usecase.ErrRecordFilterLimitExceeded, wantStatus: 400, wantCode: "record_filter_limit_exceeded"},
		{name: "フィルタ入力不正はinvalid_record_filter_inputの400", err: usecase.ErrInvalidRecordFilterInput, wantStatus: 400, wantCode: "invalid_record_filter_input"},
		{name: "フィルタID不正はinvalid_record_filter_idの400", err: usecase.ErrInvalidRecordFilterID, wantStatus: 400, wantCode: "invalid_record_filter_id"},
		// フレンド系
		{name: "フレンド上限超過はfriendship_limit_exceededの400", err: usecase.ErrFriendshipLimitExceeded, wantStatus: 400, wantCode: "friendship_limit_exceeded"},
		{name: "既存の友情関係はfriendship_conflictの409へ丸める", err: usecase.ErrFriendshipAlreadyExists, wantStatus: 409, wantCode: "friendship_conflict"},
		{name: "フレンド申請未検出はfriend_request_not_foundの404", err: usecase.ErrFriendRequestNotFound, wantStatus: 404, wantCode: "friend_request_not_found"},
		{name: "不正なフレンド申請は400のvalidation_failedへ丸める", err: usecase.ErrInvalidFriendRequest, wantStatus: 400, wantCode: "validation_failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			apiErr := FromUsecaseError(tt.err)

			// Then
			require.NotNil(t, apiErr)
			assert.Equal(t, tt.wantStatus, apiErr.HTTPStatus)
			assert.Equal(t, tt.wantCode, apiErr.Code)
			assert.ErrorIs(t, apiErr, tt.err)
		})
	}
}
