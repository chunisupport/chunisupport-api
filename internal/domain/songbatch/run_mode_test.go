package songbatch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunRequestLockConflictIsError(t *testing.T) {
	t.Parallel()

	if NewRunRequest(false, false).LockConflictIsError() {
		t.Fatal("normal run should skip on lock conflict")
	}
	if NewRunRequest(false, true).LockConflictIsError() {
		t.Fatal("fill-missing-release-date should skip on lock conflict")
	}
	if !NewRunRequest(true, false).LockConflictIsError() {
		t.Fatal("major update should error on lock conflict")
	}
}

func TestTargetAndRequiredDatasourceTypes(t *testing.T) {
	t.Parallel()

	if got := len(RunModeMajorUpdate.TargetSources()); got != 2 {
		t.Fatalf("major targets=%d", got)
	}
	if got := len(RunModeMajorUpdate.RequiredSources()); got != 2 {
		t.Fatalf("major required=%d", got)
	}
	if got := len(RunModeNormal.RequiredSources()); got != 3 {
		t.Fatalf("normal required=%d", got)
	}
}

func TestParseRunMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		want    RunMode
		wantErr bool
	}{
		{name: "NORMALは通常実行になる", value: "NORMAL", want: RunModeNormal},
		{name: "MAJOR_UPDATEは大型更新になる", value: "MAJOR_UPDATE", want: RunModeMajorUpdate},
		{name: "小文字は受け付けない", value: "normal", wantErr: true},
		{name: "空文字は受け付けない", value: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseRunMode(tt.value)

			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidRunMode)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
