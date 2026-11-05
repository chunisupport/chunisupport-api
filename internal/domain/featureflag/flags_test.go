package featureflag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const testFeature Name = "test_feature"
const otherFeature Name = "other_feature"

func TestFlagsEnabled(t *testing.T) {
	tests := []struct {
		name  string
		flags Flags
		query Name
		want  bool
	}{
		{name: "ゼロ値は無効", query: testFeature},
		{name: "空の集合は無効", flags: New(), query: testFeature},
		{name: "渡した名前は有効", flags: New(testFeature), query: testFeature, want: true},
		{name: "集合に未登録の名前は無効", flags: New(testFeature), query: otherFeature},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: 各ケースのフラグ集合を使用する。
			flags := tt.flags
			// When
			got := flags.Enabled(tt.query)
			// Then
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewCopiesInput(t *testing.T) {
	// Given
	names := []Name{testFeature}
	// When
	flags := New(names...)
	names[0] = otherFeature
	// Then
	assert.True(t, flags.Enabled(testFeature))
	assert.False(t, flags.Enabled(otherFeature))
}

func TestRegisteredNamesReturnsCopy(t *testing.T) {
	// Given: 実運用の登録一覧は空でも、コピーの独立性を検証する。
	original := registeredNames
	registeredNames = []Name{testFeature}
	t.Cleanup(func() { registeredNames = original })
	// When
	names := RegisteredNames()
	names[0] = otherFeature
	// Then
	assert.Equal(t, []Name{testFeature}, RegisteredNames())
}
