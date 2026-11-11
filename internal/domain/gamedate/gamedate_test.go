package gamedate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestToday_日本時間7時で日付が切り替わる(t *testing.T) {
	jst := Location()
	tests := []struct {
		name string
		// Given
		now time.Time
		// Then
		want string
	}{
		{
			name: "06時59分59秒は前日",
			now:  time.Date(2026, 7, 6, 6, 59, 59, 0, jst),
			want: "2026-07-05",
		},
		{
			name: "07時00分00秒は当日",
			now:  time.Date(2026, 7, 6, 7, 0, 0, 0, jst),
			want: "2026-07-06",
		},
		{
			name: "UTCの前日21時59分59秒は前日",
			now:  time.Date(2026, 7, 5, 21, 59, 59, 0, time.UTC),
			want: "2026-07-05",
		},
		{
			name: "UTCの前日22時00分00秒は当日",
			now:  time.Date(2026, 7, 5, 22, 0, 0, 0, time.UTC),
			want: "2026-07-06",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got := Today(tt.now)

			// Then
			assert.Equal(t, tt.want, got.Format(time.DateOnly))
		})
	}
}

func TestIsReleased_稼働日の日本時間7時から現行になる(t *testing.T) {
	jst := Location()
	// DBのDATEは loc=UTC の0時として読み出される。壁時計の年月日が稼働日です。
	releasedAt := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		// Given
		now time.Time
		// Then
		want bool
	}{
		{
			name: "稼働日の0時00分00秒は未公開",
			now:  time.Date(2026, 7, 6, 0, 0, 0, 0, jst),
			want: false,
		},
		{
			name: "稼働日の06時59分59秒は未公開",
			now:  time.Date(2026, 7, 6, 6, 59, 59, 0, jst),
			want: false,
		},
		{
			name: "稼働日の07時00分00秒から公開",
			now:  time.Date(2026, 7, 6, 7, 0, 0, 0, jst),
			want: true,
		},
		{
			name: "稼働日前日の07時00分00秒は未公開",
			now:  time.Date(2026, 7, 5, 7, 0, 0, 0, jst),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got := IsReleased(releasedAt, tt.now)

			// Then
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStartsAt_日付の壁時計を稼働日として7時にする(t *testing.T) {
	jst := Location()
	want := time.Date(2026, 7, 6, 7, 0, 0, 0, jst)
	tests := []struct {
		name string
		// Given: 同じ暦日を別ロケーションの0時で保持した値
		releasedAt time.Time
	}{
		{
			name:       "UTCの0時",
			releasedAt: time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC),
		},
		{
			name:       "日本時間の0時",
			releasedAt: time.Date(2026, 7, 6, 0, 0, 0, 0, jst),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got := StartsAt(tt.releasedAt)

			// Then
			assert.True(t, want.Equal(got))
		})
	}
}
