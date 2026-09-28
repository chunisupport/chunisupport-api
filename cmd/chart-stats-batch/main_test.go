package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{name: "引数なしは受け付ける", args: nil},
		{name: "廃止したdry-runは実際に書き込んでしまわないようエラーにする", args: []string{"--dry-run"}, wantErr: true},
		{name: "位置引数はエラー", args: []string{"extra"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			err := validateArgs(tt.args)

			// Then
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
