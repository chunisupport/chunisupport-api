package passwordhash

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPasswordHash_Scan(t *testing.T) {
	type args struct {
		src any
	}
	tests := []struct {
		name    string
		args    args
		want    PasswordHash
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "nil",
			args: args{
				src: nil,
			},
			want:    PasswordHash(""),
			wantErr: assert.NoError,
		},
		{
			name: "文字列",
			args: args{
				src: "test",
			},
			want:    PasswordHash("test"),
			wantErr: assert.NoError,
		},
		{
			name: "[]byte",
			args: args{
				src: []byte("test"),
			},
			want:    PasswordHash("test"),
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := new(PasswordHash)
			err := p.Scan(tt.args.src)
			if !tt.wantErr(t, err, fmt.Sprintf("Scan(%v)", tt.args.src)) {
				return
			}
			assert.Equalf(t, tt.want, *p, "Scan(%v)", tt.args.src)
		})
	}
}
