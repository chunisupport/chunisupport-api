package score

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   uint32
		want    Score
		wantErr bool
	}{
		{
			name:  "valid score",
			value: 1000000,
			want:  Score(1000000),
		},
		{
			name:    "exceeds max",
			value:   1010001,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewScore(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestScore_Value(t *testing.T) {
	t.Parallel()

	s, err := NewScore(500000)
	assert.NoError(t, err)

	v, err := s.Value()
	assert.NoError(t, err)
	assert.Equal(t, int64(500000), v)
}

func TestScore_Scan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		src     any
		want    Score
		wantErr bool
	}{
		{
			name: "int64 value",
			src:  int64(750000),
			want: Score(750000),
		},
		{
			name: "byte slice",
			src:  []byte("900000"),
			want: Score(900000),
		},
		{
			name: "string",
			src:  "1000000",
			want: Score(1000000),
		},
		{
			name:    "unsupported type",
			src:     12.34,
			wantErr: true,
		},
		{
			name:    "over max",
			src:     int64(1010001),
			wantErr: true,
		},
		{
			name:    "negative",
			src:     int64(-1),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var s Score
			err := s.Scan(tt.src)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, s)
		})
	}
}

func TestScore_ScanNil(t *testing.T) {
	t.Parallel()

	var s Score
	err := s.Scan(nil)
	assert.NoError(t, err)
	assert.Equal(t, Score(0), s)
}
