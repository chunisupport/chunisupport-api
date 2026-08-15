package entity

import (
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
	"github.com/stretchr/testify/assert"
)

func TestCourseValidate_DisplayIDの形式を検証する(t *testing.T) {
	tests := []struct {
		name      string
		displayID string
		wantErr   bool
	}{
		{name: "16文字の小文字16進数は有効", displayID: "0123456789abcdef", wantErr: false},
		{name: "空文字は無効", displayID: "", wantErr: true},
		{name: "大文字を含む場合は無効", displayID: "0123456789abcdeF", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			course := &Course{DisplayID: displayid.DisplayID(tt.displayID), OfficialIdx: "50020", Name: "コース", CourseClassID: 1}

			err := course.Validate()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
