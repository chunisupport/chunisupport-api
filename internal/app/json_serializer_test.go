package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type serializerChart struct {
	UpdatedAt *time.Time `json:"updated_at"`
}

type serializerOrderedCharts map[string]*serializerChart

func (c serializerOrderedCharts) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Master *serializerChart `json:"MASTER"`
	}{Master: c["MASTER"]})
}

func TestTimezoneJSONSerializer(t *testing.T) {
	location, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)
	e := echo.New()
	serializer := NewTimezoneJSONSerializer(location)
	e.JSONSerializer = serializer
	e.GET("/", func(c *echo.Context) error {
		value := time.Date(2026, 7, 11, 15, 30, 0, 123000000, time.UTC)
		return c.JSON(http.StatusOK, struct {
			CreatedAt time.Time  `json:"created_at"`
			UpdatedAt *time.Time `json:"updated_at"`
			Label     string     `json:"label"`
		}{CreatedAt: value, UpdatedAt: &value, Label: "2026-07-11T15:30:00Z"})
	})

	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, "2026-07-12T00:30:00.123+09:00", response["created_at"])
	assert.Equal(t, "2026-07-12T00:30:00.123+09:00", response["updated_at"])
	assert.Equal(t, "2026-07-11T15:30:00Z", response["label"])
}

func TestTimezoneJSONSerializer_入力は標準デシリアライズを維持する(t *testing.T) {
	location, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)
	e := echo.New()
	e.JSONSerializer = NewTimezoneJSONSerializer(location)
	e.POST("/", func(c *echo.Context) error {
		var request struct {
			OccurredAt time.Time `json:"occurred_at"`
		}
		if err := c.Bind(&request); err != nil {
			return err
		}
		return c.JSON(http.StatusOK, map[string]bool{"utc": request.OccurredAt.Location() == time.UTC})
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"occurred_at":"2026-07-11T15:30:00Z"}`))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	e.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"utc":true}`, recorder.Body.String())
}

func TestTimezoneJSONSerializer_nilはnullとして出力する(t *testing.T) {
	e := echo.New()
	e.JSONSerializer = NewTimezoneJSONSerializer(time.UTC)
	e.GET("/", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, nil)
	})

	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, "null", recorder.Body.String())
}

func TestTimezoneJSONSerializer_カスタムMap内の日時も変換する(t *testing.T) {
	location, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)
	e := echo.New()
	e.JSONSerializer = NewTimezoneJSONSerializer(location)
	e.GET("/", func(c *echo.Context) error {
		updatedAt := time.Date(2026, 7, 11, 15, 30, 0, 0, time.UTC)
		return c.JSON(http.StatusOK, struct {
			Charts serializerOrderedCharts `json:"charts"`
		}{Charts: serializerOrderedCharts{"MASTER": {UpdatedAt: &updatedAt}}})
	})

	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"updated_at":"2026-07-12T00:30:00+09:00"`)
}
