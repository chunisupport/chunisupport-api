package api_internal

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dataTransferUsecaseStub struct {
	exportOutput   *usecase.UserDataTransferExportOutput
	validateOutput *usecase.UserDataTransferValidationOutput
	importOutput   *usecase.UserDataTransferImportOutput
}

func (s *dataTransferUsecaseStub) Export(context.Context, int) (*usecase.UserDataTransferExportOutput, error) {
	return s.exportOutput, nil
}
func (s *dataTransferUsecaseStub) Validate(context.Context, int, []byte) (*usecase.UserDataTransferValidationOutput, error) {
	return s.validateOutput, nil
}
func (s *dataTransferUsecaseStub) Import(context.Context, int, []byte) (*usecase.UserDataTransferImportOutput, error) {
	return s.importOutput, nil
}

func TestDataTransferHandlerExportReturnsAttachment(t *testing.T) {
	handler := NewDataTransferHandler(&dataTransferUsecaseStub{exportOutput: &usecase.UserDataTransferExportOutput{File: []byte("{\"signed\":true}")}})
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/internal/me/data-transfer/export", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Set("userEntity", &entity.User{ID: 1})

	err := handler.Export(ctx)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get(echo.HeaderContentType))
	assert.Equal(t, "attachment; filename=\"chunisupport-transfer.json\"", rec.Header().Get(echo.HeaderContentDisposition))
	assert.JSONEq(t, "{\"signed\":true}", rec.Body.String())
}

func TestDataTransferHandlerValidatePassesEnvelope(t *testing.T) {
	output := &usecase.UserDataTransferValidationOutput{
		Importable:           true,
		PlayerName:           "テスト",
		Blockers:             []string{},
		UnresolvedReferences: []string{},
	}
	handler := NewDataTransferHandler(&dataTransferUsecaseStub{validateOutput: output})
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/internal/me/data-transfer/validate", bytes.NewBufferString("{\"signed\":true}"))
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Set("userEntity", &entity.User{ID: 1})

	err := handler.Validate(ctx)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	var response dto_internal.DataTransferValidationResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.True(t, response.Importable)
}

func TestDataTransferHandlerValidateReturnsEmptyArraysForNoValidationIssues(t *testing.T) {
	output := &usecase.UserDataTransferValidationOutput{
		Importable: true,
		PlayerName: "TEST",
	}
	handler := NewDataTransferHandler(&dataTransferUsecaseStub{validateOutput: output})
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/internal/me/data-transfer/validate", bytes.NewBufferString("{\"signed\":true}"))
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Set("userEntity", &entity.User{ID: 1})

	err := handler.Validate(ctx)

	require.NoError(t, err)
	assert.JSONEq(t, `{"importable":true,"player_name":"TEST","counts":{"records":0,"record_histories":0,"worldsend_records":0,"worldsend_record_histories":0,"metric_histories":0,"course_records":0,"honors":0,"favorite_songs":0,"locked_songs":0,"goal_groups":0,"goals":0,"record_filters":0},"blockers":[],"unresolved_references":[],"unresolved_reference_count":0}`, rec.Body.String())
}

func TestDataTransferHandlerValidateRejectsEmptyBodyAsInvalidFile(t *testing.T) {
	handler := NewDataTransferHandler(&dataTransferUsecaseStub{})
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/internal/me/data-transfer/validate", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Set("userEntity", &entity.User{ID: 1})

	err := handler.Validate(ctx)

	assert.Same(t, apierror.ErrDataTransferInvalidFile, err)
}

func TestDataTransferHandlerImportReturnsImportedPlayer(t *testing.T) {
	output := &usecase.UserDataTransferImportOutput{
		PlayerID: 42,
		Counts:   usecase.UserDataTransferCounts{Records: 3, Goals: 2},
	}
	handler := NewDataTransferHandler(&dataTransferUsecaseStub{importOutput: output})
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/internal/me/data-transfer/import", bytes.NewBufferString("{\"signed\":true}"))
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Set("userEntity", &entity.User{ID: 1})

	err := handler.Import(ctx)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	var response dto_internal.DataTransferImportResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, 42, response.PlayerID)
	assert.Equal(t, 3, response.Counts.Records)
	assert.Equal(t, 2, response.Counts.Goals)
}
