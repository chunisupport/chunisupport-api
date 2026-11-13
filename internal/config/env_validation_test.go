package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateEnv_設定ファイル名に使える文字だけを許可する(t *testing.T) {
	for _, value := range []string{"develop", "stg_2-a", "PROD9"} {
		t.Run(value, func(t *testing.T) { assert.NoError(t, validateEnv(value)) })
	}
	for _, value := range []string{"../prod", `..\prod`, "a/b", "a.b", "a b", "開発", "prod\n", "prod\x00"} {
		t.Run(value, func(t *testing.T) { assert.Error(t, validateEnv(value)) })
	}
}
