package app

import (
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type serializerChart struct {
	UpdatedAt *time.Time `json:"updated_at"`
}

// serializerOrderedCharts は難易度順マップと同じく、渡されたエンコーダへ書き込む独自マーシャラを持つ型です。
type serializerOrderedCharts map[string]*serializerChart

func (c serializerOrderedCharts) MarshalJSONTo(enc *jsontext.Encoder) error {
	return jsonv2.MarshalEncode(enc, struct {
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

// SerializerEmbedded などの埋め込み用の型は、本番の EditorSongDTO が埋め込む *SongDTO と同じく公開型にしています。
type SerializerEmbedded struct {
	EmbeddedAt time.Time `json:"embedded_at"`
}

type SerializerEmbeddedPointer struct {
	PointerEmbeddedAt *time.Time `json:"pointer_embedded_at"`
}

type SerializerEmbeddedNil struct {
	NilEmbeddedAt time.Time `json:"nil_embedded_at"`
}

// serializerOpaqueTime は独自 MarshalJSON を持つため、内部の時刻は変換対象外であることを確認する型です。
type serializerOpaqueTime struct {
	At time.Time
}

func (o serializerOpaqueTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]time.Time{"opaque_at": o.At})
}

type serializerItem struct {
	At      *time.Time `json:"at"`
	Comment string     `json:"comment,omitempty"`
}

// serializerGoldenResponse はシリアライザの出力互換性をバイト単位で確認するためのレスポンスです。
type serializerGoldenResponse struct {
	SerializerEmbedded
	*SerializerEmbeddedPointer
	*SerializerEmbeddedNil
	CreatedAt       time.Time                           `json:"created_at"`
	WholeSecond     time.Time                           `json:"whole_second"`
	ZeroTime        time.Time                           `json:"zero_time"`
	ZeroOmitEmpty   time.Time                           `json:"zero_omit_empty,omitempty"`
	UpdatedAt       *time.Time                          `json:"updated_at"`
	DeletedAt       *time.Time                          `json:"deleted_at"`
	OmittedAt       *time.Time                          `json:"omitted_at,omitempty"`
	AnyTime         any                                 `json:"any_time"`
	AnyPointer      any                                 `json:"any_pointer"`
	AnyNil          any                                 `json:"any_nil"`
	TimeMap         map[string]time.Time                `json:"time_map"`
	AnyMap          map[string]any                      `json:"any_map"`
	NilMap          map[string]any                      `json:"nil_map"`
	EmptyMapOmit    map[string]any                      `json:"empty_map_omit,omitempty"`
	Items           []serializerItem                    `json:"items"`
	NilItems        []serializerItem                    `json:"nil_items"`
	EmptyItems      []serializerItem                    `json:"empty_items"`
	EmptyItemsOmit  []serializerItem                    `json:"empty_items_omit,omitempty"`
	TimeArray       [2]time.Time                        `json:"time_array"`
	Opaque          serializerOpaqueTime                `json:"opaque"`
	OrderedCharts   api_internal.EditorOrderedChartsMap `json:"ordered_charts"`
	NilCharts       api_internal.EditorOrderedChartsMap `json:"nil_charts"`
	Raw             json.RawMessage                     `json:"raw"`
	HTML            string                              `json:"html"`
	StringNumber    int                                 `json:"string_number,string"`
	Float           float64                             `json:"float"`
	LargeFloat      float64                             `json:"large_float"`
	Bytes           []byte                              `json:"bytes"`
	EmptyStringOmit string                              `json:"empty_string_omit,omitempty"`
	Ignored         time.Time                           `json:"-"`
	unexported      time.Time
}

func newSerializerGoldenResponse(t *testing.T) *serializerGoldenResponse {
	t.Helper()
	value := time.Date(2026, 7, 11, 15, 30, 0, 123456789, time.UTC)
	wholeSecond := time.Date(2026, 12, 31, 15, 0, 0, 0, time.UTC)
	chartConst, err := chartconstant.NewChartConstant(15.4)
	require.NoError(t, err)
	notes := 999
	return &serializerGoldenResponse{
		SerializerEmbedded:        SerializerEmbedded{EmbeddedAt: value},
		SerializerEmbeddedPointer: &SerializerEmbeddedPointer{PointerEmbeddedAt: &wholeSecond},
		CreatedAt:                 value,
		WholeSecond:               wholeSecond,
		UpdatedAt:                 &value,
		AnyTime:                   value,
		AnyPointer:                &wholeSecond,
		TimeMap:                   map[string]time.Time{"b": wholeSecond, "a": value},
		AnyMap:                    map[string]any{"at": value, "nested": map[string]any{"at": &wholeSecond}, "text": "x"},
		EmptyMapOmit:              map[string]any{},
		Items:                     []serializerItem{{At: &value, Comment: "c"}, {At: nil}},
		EmptyItems:                []serializerItem{},
		EmptyItemsOmit:            []serializerItem{},
		TimeArray:                 [2]time.Time{value, wholeSecond},
		Opaque:                    serializerOpaqueTime{At: value},
		OrderedCharts: api_internal.EditorOrderedChartsMap{
			"MASTER": {Const: chartConst, Notes: &notes, UpdatedAt: &value},
			"BASIC":  {Const: chartConst},
		},
		Raw:          json.RawMessage(`{"raw_at": "2026-07-11T15:30:00Z"}`),
		HTML:         "<a href=\"x\">&</a> ",
		StringNumber: 42,
		Float:        14.9,
		LargeFloat:   1e21,
		Bytes:        []byte("chunisupport"),
		Ignored:      value,
		unexported:   value,
	}
}

func TestTimezoneJSONSerializer_既存のJSON出力とバイト単位で一致する(t *testing.T) {
	location, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)

	tests := []struct {
		name string
		// Given: 出力インデント
		indent string
		// Then: 変更前のシリアライザ（レスポンス値を複製して時刻を変換する実装）の出力
		want string
	}{
		{
			name:   "インデントなし",
			indent: "",
			want:   `{"embedded_at":"2026-07-12T00:30:00.123456789+09:00","pointer_embedded_at":"2027-01-01T00:00:00+09:00","created_at":"2026-07-12T00:30:00.123456789+09:00","whole_second":"2027-01-01T00:00:00+09:00","zero_time":"0001-01-01T09:18:59+09:18","zero_omit_empty":"0001-01-01T09:18:59+09:18","updated_at":"2026-07-12T00:30:00.123456789+09:00","deleted_at":null,"any_time":"2026-07-12T00:30:00.123456789+09:00","any_pointer":"2027-01-01T00:00:00+09:00","any_nil":null,"time_map":{"a":"2026-07-12T00:30:00.123456789+09:00","b":"2027-01-01T00:00:00+09:00"},"any_map":{"at":"2026-07-12T00:30:00.123456789+09:00","nested":{"at":"2027-01-01T00:00:00+09:00"},"text":"x"},"nil_map":null,"items":[{"at":"2026-07-12T00:30:00.123456789+09:00","comment":"c"},{"at":null}],"nil_items":null,"empty_items":[],"time_array":["2026-07-12T00:30:00.123456789+09:00","2027-01-01T00:00:00+09:00"],"opaque":{"opaque_at":"2026-07-11T15:30:00.123456789Z"},"ordered_charts":{"BASIC":{"const":15.4,"is_const_unknown":false,"notes":null,"notes_designer":null,"updated_at":null},"ADVANCED":null,"EXPERT":null,"MASTER":{"const":15.4,"is_const_unknown":false,"notes":999,"notes_designer":null,"updated_at":"2026-07-12T00:30:00.123456789+09:00"},"ULTIMA":null},"nil_charts":{"BASIC":null,"ADVANCED":null,"EXPERT":null,"MASTER":null,"ULTIMA":null},"raw":{"raw_at":"2026-07-11T15:30:00Z"},"html":"\u003ca href=\"x\"\u003e\u0026\u003c/a\u003e\u2028","string_number":"42","float":14.9,"large_float":1e+21,"bytes":"Y2h1bmlzdXBwb3J0"}` + "\n",
		},
		{
			name:   "インデントあり",
			indent: "  ",
			want: `{
  "embedded_at": "2026-07-12T00:30:00.123456789+09:00",
  "pointer_embedded_at": "2027-01-01T00:00:00+09:00",
  "created_at": "2026-07-12T00:30:00.123456789+09:00",
  "whole_second": "2027-01-01T00:00:00+09:00",
  "zero_time": "0001-01-01T09:18:59+09:18",
  "zero_omit_empty": "0001-01-01T09:18:59+09:18",
  "updated_at": "2026-07-12T00:30:00.123456789+09:00",
  "deleted_at": null,
  "any_time": "2026-07-12T00:30:00.123456789+09:00",
  "any_pointer": "2027-01-01T00:00:00+09:00",
  "any_nil": null,
  "time_map": {
    "a": "2026-07-12T00:30:00.123456789+09:00",
    "b": "2027-01-01T00:00:00+09:00"
  },
  "any_map": {
    "at": "2026-07-12T00:30:00.123456789+09:00",
    "nested": {
      "at": "2027-01-01T00:00:00+09:00"
    },
    "text": "x"
  },
  "nil_map": null,
  "items": [
    {
      "at": "2026-07-12T00:30:00.123456789+09:00",
      "comment": "c"
    },
    {
      "at": null
    }
  ],
  "nil_items": null,
  "empty_items": [],
  "time_array": [
    "2026-07-12T00:30:00.123456789+09:00",
    "2027-01-01T00:00:00+09:00"
  ],
  "opaque": {
    "opaque_at": "2026-07-11T15:30:00.123456789Z"
  },
  "ordered_charts": {
    "BASIC": {
      "const": 15.4,
      "is_const_unknown": false,
      "notes": null,
      "notes_designer": null,
      "updated_at": null
    },
    "ADVANCED": null,
    "EXPERT": null,
    "MASTER": {
      "const": 15.4,
      "is_const_unknown": false,
      "notes": 999,
      "notes_designer": null,
      "updated_at": "2026-07-12T00:30:00.123456789+09:00"
    },
    "ULTIMA": null
  },
  "nil_charts": {
    "BASIC": null,
    "ADVANCED": null,
    "EXPERT": null,
    "MASTER": null,
    "ULTIMA": null
  },
  "raw": {
    "raw_at": "2026-07-11T15:30:00Z"
  },
  "html": "\u003ca href=\"x\"\u003e\u0026\u003c/a\u003e\u2028",
  "string_number": "42",
  "float": 14.9,
  "large_float": 1e+21,
  "bytes": "Y2h1bmlzdXBwb3J0"
}` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			response := newSerializerGoldenResponse(t)
			recorder := httptest.NewRecorder()
			c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), recorder)

			// When
			err := NewTimezoneJSONSerializer(location).Serialize(c, response, tt.indent)

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.want, recorder.Body.String())
			assert.Equal(t, time.UTC, response.CreatedAt.Location(), "レスポンス元の時刻は変更しない")
		})
	}
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

// serializerHiddenEmbedded は、APIトークン発行レスポンスのように非公開型として埋め込まれる構造体です。
type serializerHiddenEmbedded struct {
	CreatedAt time.Time `json:"created_at"`
}

func TestTimezoneJSONSerializer_非公開型の埋め込み構造体の日時も変換する(t *testing.T) {
	// Given: 非公開型を埋め込んだレスポンス
	e := echo.New()
	recorder := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), recorder)
	createdAt := time.Date(2026, 7, 22, 3, 34, 56, 0, time.UTC)
	response := struct {
		*serializerHiddenEmbedded
		Token string `json:"token"`
	}{serializerHiddenEmbedded: &serializerHiddenEmbedded{CreatedAt: createdAt}, Token: "token"}

	// When
	err := NewTimezoneJSONSerializer(time.FixedZone("Asia/Tokyo", 9*60*60)).Serialize(c, response, "")

	// Then: 昇格したフィールドの日時も出力タイムゾーンへ変換する
	require.NoError(t, err)
	assert.Equal(t, `{"created_at":"2026-07-22T12:34:56+09:00","token":"token"}`+"\n", recorder.Body.String())
}

type benchmarkSerializerRecord struct {
	SongID    string     `json:"song_id"`
	Score     int        `json:"score"`
	Comment   string     `json:"comment,omitempty"`
	PlayedAt  *time.Time `json:"played_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// BenchmarkTimezoneJSONSerializer_Serialize は、時刻を多く含む大きなレスポンスと
// 難易度順の譜面マップを含む楽曲一覧のシリアライズ性能を計測します。
func BenchmarkTimezoneJSONSerializer_Serialize(b *testing.B) {
	location, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(b, err)
	playedAt := time.Date(2026, 7, 11, 15, 30, 0, 123000000, time.UTC)
	chartConst, err := chartconstant.NewChartConstant(14.9)
	require.NoError(b, err)

	records := make([]benchmarkSerializerRecord, 4000)
	for i := range records {
		records[i] = benchmarkSerializerRecord{SongID: "0123456789abcdef", Score: 1009000 + i, PlayedAt: &playedAt, UpdatedAt: playedAt}
	}
	songs := make([]*api_internal.EditorSongDTO, 1000)
	for i := range songs {
		songs[i] = &api_internal.EditorSongDTO{
			SongDTO:   &api_internal.SongDTO{DisplayID: "0123456789abcdef", Title: "楽曲"},
			UpdatedAt: &playedAt,
			Charts: api_internal.EditorOrderedChartsMap{
				"BASIC":  {Const: chartConst, UpdatedAt: &playedAt},
				"EXPERT": {Const: chartConst, UpdatedAt: &playedAt},
				"MASTER": {Const: chartConst, UpdatedAt: &playedAt},
			},
		}
	}

	benchmarks := []struct {
		name   string
		target any
	}{
		{name: "時刻を含む記録4000件", target: map[string]any{"records": records}},
		{name: "編集者向け楽曲一覧1000件", target: &api_internal.EditorSongsResponse{Songs: songs}},
	}
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			serializer := NewTimezoneJSONSerializer(location)
			recorder := httptest.NewRecorder()
			c := echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/", nil), recorder)
			b.ReportAllocs()
			for b.Loop() {
				recorder.Body.Reset()
				require.NoError(b, serializer.Serialize(c, bm.target, ""))
			}
		})
	}
}
