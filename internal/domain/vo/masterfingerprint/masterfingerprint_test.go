package masterfingerprint

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFingerprint(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "16進数小文字64文字は受け付ける", input: strings.Repeat("0123456789abcdef", 4), wantErr: false},
		{name: "大文字を含む場合はエラー", input: strings.Repeat("0123456789ABCDEF", 4), wantErr: true},
		{name: "63文字の場合はエラー", input: strings.Repeat("a", 63), wantErr: true},
		{name: "65文字の場合はエラー", input: strings.Repeat("a", 65), wantErr: true},
		{name: "16進数以外の文字を含む場合はエラー", input: strings.Repeat("g", 64), wantErr: true},
		{name: "空文字はエラー", input: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got, err := NewFingerprint(tt.input)

			// Then
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidFingerprint)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.input, got.String())
		})
	}
}

func TestCompute(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "空データのSHA-256を16進数小文字で返す",
			data:     []byte{},
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "abcのSHA-256を16進数小文字で返す",
			data:     []byte("abc"),
			expected: "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			got := Compute(tt.data)

			// Then
			assert.Equal(t, tt.expected, got.String())
		})
	}
}

func TestFingerprint_ValueとScanで往復できる(t *testing.T) {
	// Given
	original := Compute([]byte("abc"))

	// When
	value, err := original.Value()
	require.NoError(t, err)
	var restored Fingerprint
	err = restored.Scan([]byte(value.(string)))

	// Then
	require.NoError(t, err)
	assert.Equal(t, original, restored)
}

func TestFingerprint_Scan_不正な値を拒否する(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{name: "NULLはエラー", input: nil},
		{name: "形式不正な文字列はエラー", input: "invalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			var fingerprint Fingerprint

			// When
			err := fingerprint.Scan(tt.input)

			// Then
			assert.ErrorIs(t, err, ErrInvalidFingerprint)
		})
	}
}

func TestFingerprint_Value_ゼロ値はエラー(t *testing.T) {
	// Given
	var fingerprint Fingerprint

	// When
	_, err := fingerprint.Value()

	// Then
	assert.ErrorIs(t, err, ErrInvalidFingerprint)
}
