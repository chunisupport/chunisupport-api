package api_internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDateOnly_UnmarshalJSON(t *testing.T) {
	var date DateOnly

	err := date.UnmarshalJSON([]byte(`"2026-08-30"`))

	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC), date.Time)
}

func TestDateOnly_UnmarshalJSON_不正な入力(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "日付形式ではない", input: `"2026/08/30"`},
		{name: "存在しない日付", input: `"2026-02-30"`},
		{name: "JSON文字列ではない", input: `123`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var date DateOnly
			assert.Error(t, date.UnmarshalJSON([]byte(tt.input)))
		})
	}
}

func TestDateOnly_TimePtr(t *testing.T) {
	expected := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	date := &DateOnly{Time: expected}

	actual := date.TimePtr()

	require.NotNil(t, actual)
	assert.Equal(t, expected, *actual)
	assert.NotSame(t, &date.Time, actual)
	assert.Nil(t, (*DateOnly)(nil).TimePtr())
}
