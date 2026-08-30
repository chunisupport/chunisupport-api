package entity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestoreRecordFilter(t *testing.T) {
	createdAt := time.Date(2026, 8, 29, 1, 2, 3, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	id := []byte{1, 2, 3}
	filterValue := []byte(`{"difficulty":"MASTER"}`)

	filter, err := RestoreRecordFilter(id, 10, "高難易度", filterValue, true, createdAt, updatedAt)

	require.NoError(t, err)
	assert.Equal(t, id, filter.ID())
	assert.Equal(t, 10, filter.UserID())
	assert.Equal(t, "高難易度", filter.Name())
	assert.Equal(t, filterValue, filter.FilterValueGzip())
	assert.True(t, filter.IsWorldsend())
	assert.Equal(t, createdAt, filter.CreatedAt())
	assert.Equal(t, updatedAt, filter.UpdatedAt())
}

func TestRestoreRecordFilter_必須項目の検証(t *testing.T) {
	tests := []struct {
		name        string
		id          []byte
		userID      int
		filterName  string
		filterValue []byte
		expected    error
	}{
		{name: "IDが空", userID: 1, filterName: "名前", filterValue: []byte{1}, expected: ErrRecordFilterIDRequired},
		{name: "ユーザーIDが0", id: []byte{1}, filterName: "名前", filterValue: []byte{1}, expected: ErrRecordFilterUserIDInvalid},
		{name: "ユーザーIDが負数", id: []byte{1}, userID: -1, filterName: "名前", filterValue: []byte{1}, expected: ErrRecordFilterUserIDInvalid},
		{name: "名前が空", id: []byte{1}, userID: 1, filterValue: []byte{1}, expected: ErrRecordFilterNameRequired},
		{name: "フィルタ値が空", id: []byte{1}, userID: 1, filterName: "名前", expected: ErrRecordFilterFilterValueGzipRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := RestoreRecordFilter(tt.id, tt.userID, tt.filterName, tt.filterValue, false, time.Time{}, time.Time{})

			assert.Nil(t, filter)
			assert.ErrorIs(t, err, tt.expected)
		})
	}
}

func TestRecordFilter_バイト列を外部変更から保護する(t *testing.T) {
	id := []byte{1, 2}
	value := []byte{3, 4}
	filter, err := NewRecordFilter(id, 1, "名前", value, false)
	require.NoError(t, err)

	id[0], value[0] = 9, 9
	returnedID := filter.ID()
	returnedValue := filter.FilterValueGzip()
	returnedID[0], returnedValue[0] = 8, 8

	assert.Equal(t, []byte{1, 2}, filter.ID())
	assert.Equal(t, []byte{3, 4}, filter.FilterValueGzip())
}

func TestRecordFilter_変更操作(t *testing.T) {
	filter, err := NewRecordFilter([]byte{1}, 1, "変更前", []byte{2}, false)
	require.NoError(t, err)

	assert.ErrorIs(t, filter.ChangeName(""), ErrRecordFilterNameRequired)
	assert.Equal(t, "変更前", filter.Name())
	require.NoError(t, filter.ChangeName("変更後"))
	assert.Equal(t, "変更後", filter.Name())

	assert.ErrorIs(t, filter.ChangeFilterValueGzip(nil), ErrRecordFilterFilterValueGzipRequired)
	assert.Equal(t, []byte{2}, filter.FilterValueGzip())
	replacement := []byte{3}
	require.NoError(t, filter.ChangeFilterValueGzip(replacement))
	replacement[0] = 9
	assert.Equal(t, []byte{3}, filter.FilterValueGzip())

	filter.ChangeWorldsend(true)
	assert.True(t, filter.IsWorldsend())
}
