package vo

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToString(t *testing.T) {
	type args struct {
		v any
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "文字列",
			args: args{
				v: "test",
			},
			want:    "test",
			wantErr: assert.NoError,
		},
		{
			name: "[]byte",
			args: args{
				v: []byte("test"),
			},
			want:    "test",
			wantErr: assert.NoError,
		},
		{
			name: "整数",
			args: args{
				v: 1,
			},
			want:    "1",
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToString(tt.args.v)
			if !tt.wantErr(t, err, fmt.Sprintf("ToString(%v)", tt.args.v)) {
				return
			}
			assert.Equalf(t, tt.want, got, "ToString(%v)", tt.args.v)
		})
	}
}
