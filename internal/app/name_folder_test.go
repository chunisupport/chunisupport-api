package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	internalhandler "github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/domain/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

type nameFolderMasterStub struct{ usecase.MasterDataUsecase }

func (nameFolderMasterStub) GetNameFolders(context.Context) []masterdata.NameFolder {
	return []masterdata.NameFolder{{ID: 99, Code: "KA", Name: "か行", SortOrder: 8}}
}

func (s nameFolderMasterStub) GetMasterData(ctx context.Context) *usecase.MasterDataOutput {
	return &usecase.MasterDataOutput{NameFolders: s.GetNameFolders(ctx)}
}

func TestRegisterRoutes_NameFolders(t *testing.T) {
	handlers := newAuthorizationTestHandlers()
	handlers.MasterData = internalhandler.NewMasterDataHandler(nameFolderMasterStub{})
	e := echo.New()
	auth := permissionAuthenticator{}
	registerRoutes(e, handlers, auth, auth, nil, stubMaintenanceUsecase{}, config.Config{})
	for _, path := range []string{"/internal/master/name-folders", "/internal/master"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Contains(t, rec.Body.String(), `"name_folders":[{"code":"KA","name":"か行","sort_order":8}]`)
			assert.NotContains(t, rec.Body.String(), `"id":99`)
		})
	}
}
