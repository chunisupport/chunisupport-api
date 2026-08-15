package main

import (
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignalName(t *testing.T) {
	tests := []struct {
		name string
		// Given: 受信した終了シグナル
		signal os.Signal
		// Then: ログに出力するシグナル名
		expected string
	}{
		{
			name:     "SIGINTは大文字表記で出力する",
			signal:   os.Interrupt,
			expected: "SIGINT",
		},
		{
			name:     "SIGTERMは大文字表記で出力する",
			signal:   syscall.SIGTERM,
			expected: "SIGTERM",
		},
		{
			name:     "nilは空文字列を返す",
			signal:   nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			actual := signalName(tt.signal)

			// Then
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestPendingSignalName(t *testing.T) {
	t.Run("保留中のシグナル名を返す", func(t *testing.T) {
		// Given
		ch := make(chan os.Signal, 1)
		ch <- syscall.SIGTERM

		// When
		actual := pendingSignalName(ch)

		// Then
		assert.Equal(t, "SIGTERM", actual)
	})

	t.Run("保留中のシグナルがなければ空文字列を返す", func(t *testing.T) {
		// Given
		ch := make(chan os.Signal, 1)

		// When
		actual := pendingSignalName(ch)

		// Then
		assert.Equal(t, "", actual)
	})
}
