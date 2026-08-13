package api_internal

import (
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

type DataTransferHandler struct {
	usecase usecase.UserDataTransferUsecase
}

func NewDataTransferHandler(transferUsecase ...usecase.UserDataTransferUsecase) *DataTransferHandler {
	var selected usecase.UserDataTransferUsecase
	if len(transferUsecase) > 0 {
		selected = transferUsecase[0]
	}
	return &DataTransferHandler{usecase: selected}
}

func (handler *DataTransferHandler) Export(context *echo.Context) error {
	user, err := getUserEntityFromContext(context)
	if err != nil {
		return err
	}
	if handler.usecase == nil {
		return apierror.ErrInternalError
	}
	started := time.Now()
	output, err := handler.usecase.Export(context.Request().Context(), user.ID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	context.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="chunisupport-transfer.json"`)
	context.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	slog.Info("data transfer exported", "event", "data_transfer_exported", "user_id", user.ID, "duration_ms", time.Since(started).Milliseconds())
	return context.Blob(http.StatusOK, echo.MIMEApplicationJSON, output.File)
}

func (handler *DataTransferHandler) Validate(context *echo.Context) error {
	user, err := getUserEntityFromContext(context)
	if err != nil {
		return err
	}
	encoded, err := readDataTransferBody(context)
	if err != nil {
		return err
	}
	if handler.usecase == nil {
		return apierror.ErrInternalError
	}
	output, err := handler.usecase.Validate(context.Request().Context(), user.ID, encoded)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	context.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	return context.JSON(http.StatusOK, toDataTransferValidationResponse(output))
}

func (handler *DataTransferHandler) Import(context *echo.Context) error {
	user, err := getUserEntityFromContext(context)
	if err != nil {
		return err
	}
	encoded, err := readDataTransferBody(context)
	if err != nil {
		return err
	}
	if handler.usecase == nil {
		return apierror.ErrInternalError
	}
	started := time.Now()
	output, err := handler.usecase.Import(context.Request().Context(), user.ID, encoded)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	slog.Info("data transfer imported", "event", "data_transfer_imported", "user_id", user.ID, "player_id", output.PlayerID, "duration_ms", time.Since(started).Milliseconds())
	context.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	return context.JSON(http.StatusOK, &dto_internal.DataTransferImportResponse{PlayerID: output.PlayerID, Counts: toDataTransferCountsResponse(output.Counts)})
}

func readDataTransferBody(context *echo.Context) ([]byte, error) {
	reader := io.LimitReader(context.Request().Body, int64(info.DataTransferEnvelopeMaxBytes)+1)
	encoded, err := io.ReadAll(reader)
	if err != nil {
		return nil, apierror.ErrDataTransferInvalidFile.WithInternal(err)
	}
	if len(encoded) == 0 {
		return nil, apierror.ErrDataTransferInvalidFile
	}
	if len(encoded) > info.DataTransferEnvelopeMaxBytes {
		return nil, apierror.ErrPayloadTooLarge
	}
	return encoded, nil
}

func toDataTransferValidationResponse(output *usecase.UserDataTransferValidationOutput) *dto_internal.DataTransferValidationResponse {
	return &dto_internal.DataTransferValidationResponse{Importable: output.Importable, PlayerName: output.PlayerName, Counts: toDataTransferCountsResponse(output.Counts), Blockers: output.Blockers, UnresolvedReferences: output.UnresolvedReferences, UnresolvedReferenceCount: output.UnresolvedReferenceCount}
}

func toDataTransferCountsResponse(counts usecase.UserDataTransferCounts) dto_internal.DataTransferCountsResponse {
	return dto_internal.DataTransferCountsResponse{Records: counts.Records, RecordHistories: counts.RecordHistories, WorldsendRecords: counts.WorldsendRecords, WorldsendRecordHistories: counts.WorldsendRecordHistories, MetricHistories: counts.MetricHistories, CourseRecords: counts.CourseRecords, Honors: counts.Honors, FavoriteSongs: counts.FavoriteSongs, LockedSongs: counts.LockedSongs, GoalGroups: counts.GoalGroups, Goals: counts.Goals, RecordFilters: counts.RecordFilters}
}
