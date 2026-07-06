package config

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/chunisupport/chunisupport-api/internal/domain/service"
)

// UsernamePolicy はユーザー名の登録禁止語設定です。
type UsernamePolicy struct {
	Exact    []string `json:"exact"`
	Contains []string `json:"contains"`
}

func loadUsernamePolicy(path string) (UsernamePolicy, error) {
	file, err := os.Open(path) // #nosec G304 設定ディレクトリ内の固定パスのみを使用
	if err != nil {
		return UsernamePolicy{}, fmt.Errorf("failed to open username forbidden words file: %w", err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	if prefix, _ := reader.Peek(3); bytes.Equal(prefix, []byte{0xEF, 0xBB, 0xBF}) {
		_, _ = reader.Discard(3)
	}

	var policy UsernamePolicy
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	var rawPolicy *UsernamePolicy
	if err := decoder.Decode(&rawPolicy); err != nil {
		return UsernamePolicy{}, fmt.Errorf("failed to decode username forbidden words file: %w", err)
	}
	if rawPolicy == nil {
		return UsernamePolicy{}, errors.New("username forbidden words file must contain a JSON object")
	}
	policy = *rawPolicy
	if err := ensureJSONEOF(decoder); err != nil {
		return UsernamePolicy{}, err
	}
	if _, err := service.NewForbiddenUsernamePolicy(policy.Exact, policy.Contains); err != nil {
		return UsernamePolicy{}, fmt.Errorf("failed to validate username forbidden words file: %w", err)
	}
	policy.Exact = uniqueStrings(policy.Exact)
	policy.Contains = uniqueStrings(policy.Contains)
	return policy, nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("username forbidden words file contains trailing JSON data")
		}
		return fmt.Errorf("failed to decode trailing username forbidden words data: %w", err)
	}
	return nil
}
