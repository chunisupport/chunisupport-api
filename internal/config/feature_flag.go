package config

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"

	"github.com/chunisupport/chunisupport-api/internal/domain/featureflag"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"go.yaml.in/yaml/v3"
)

//go:embed feature-flag.yml
var featureFlagYAML []byte

// parseFeatureFlags は全環境の定義と登録一覧を検証してから、指定環境の値を選びます。
// 登録一覧を引数にすることで、本番に不要なフラグを登録せずに検証できます。
func parseFeatureFlags(data []byte, env string, registered []featureflag.Name) (featureflag.Flags, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var document yaml.Node
	err := decoder.Decode(&document)
	if err != nil && !errors.Is(err, io.EOF) {
		return featureflag.Flags{}, fmt.Errorf("failed to decode feature-flag YAML: %w", err)
	}
	definitions := make(map[featureflag.Name]map[string]bool)
	if err == nil {
		var extra yaml.Node
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			if err != nil {
				return featureflag.Flags{}, fmt.Errorf("failed to decode subsequent feature-flag YAML document: %w", err)
			}
			return featureflag.Flags{}, fmt.Errorf("feature-flag YAML must not contain multiple documents")
		}

		root := document.Content[0]
		if root.Kind != yaml.MappingNode {
			return featureflag.Flags{}, fmt.Errorf("feature-flag YAML root must be a mapping")
		}
		for i := 0; i < len(root.Content); i += 2 {
			key, value := root.Content[i], root.Content[i+1]
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
				return featureflag.Flags{}, fmt.Errorf("feature-flag name must be a string")
			}
			name := featureflag.Name(key.Value)
			if _, exists := definitions[name]; exists {
				return featureflag.Flags{}, fmt.Errorf("duplicate feature-flag name %q", name)
			}
			values, err := parseFeatureFlagEnvironments(name, value)
			if err != nil {
				return featureflag.Flags{}, err
			}
			definitions[name] = values
		}
	}

	registeredSet := make(map[featureflag.Name]struct{}, len(registered))
	for _, name := range registered {
		registeredSet[name] = struct{}{}
		if _, exists := definitions[name]; !exists {
			return featureflag.Flags{}, fmt.Errorf("registered feature-flag %q is missing from YAML", name)
		}
	}
	var enabled []featureflag.Name
	for name, values := range definitions {
		if _, exists := registeredSet[name]; !exists {
			return featureflag.Flags{}, fmt.Errorf("feature-flag %q is missing from registered names", name)
		}
		// 環境キーは検証済みの3種類しか存在しないため、未知のAPP_ENVは必ず無効になります。
		if values[env] {
			enabled = append(enabled, name)
		}
	}
	return featureflag.New(enabled...), nil
}

func parseFeatureFlagEnvironments(name featureflag.Name, node *yaml.Node) (map[string]bool, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("feature-flag %q must be a mapping", name)
	}
	values := make(map[string]bool)
	for i := 0; i < len(node.Content); i += 2 {
		key, value := node.Content[i], node.Content[i+1]
		if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
			return nil, fmt.Errorf("feature-flag %q environment key must be a string", name)
		}
		switch key.Value {
		case info.AppEnvDevelop, info.AppEnvStaging, info.AppEnvProduction:
		default:
			return nil, fmt.Errorf("feature-flag %q environment key %q must be one of develop, staging, production", name, key.Value)
		}
		if _, exists := values[key.Value]; exists {
			return nil, fmt.Errorf("feature-flag %q has duplicate environment key %q", name, key.Value)
		}
		if value.Kind != yaml.ScalarNode || value.Tag != "!!bool" {
			return nil, fmt.Errorf("feature-flag %q value for environment %q must be a scalar with tag !!bool", name, key.Value)
		}
		// 種別とタグを確認した後だけ変換し、文字列やYAML 1.1のyesなどの暗黙変換を防ぎます。
		var enabled bool
		if err := value.Decode(&enabled); err != nil {
			return nil, fmt.Errorf("feature-flag %q value for environment %q must be a valid bool: %w", name, key.Value, err)
		}
		values[key.Value] = enabled
	}
	return values, nil
}
