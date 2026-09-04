package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"{{ .Computed.module_name_final }}/internal/common/redact"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

const envPrefix = "SERVICE_"

type Override struct {
	Environment string
	Key         string
	Value       string
}

func LoadDir(dir string) (*Config, []Override, error) {
	values := koanf.New(".")
	paths, err := yamlFiles(dir)
	if err != nil {
		return nil, nil, err
	}
	for _, path := range paths {
		if err := values.Load(file.Provider(path), yaml.Parser()); err != nil {
			return nil, nil, fmt.Errorf("load config file %q: %w", path, err)
		}
	}
	envKeys := environmentKeys(values)
	policy := redact.New(values.Strings("redact.keys")...)
	overrides := environmentOverrides(envKeys, policy)
	if err := values.Load(env.Provider(envPrefix, ".", func(name string) string {
		return envKeys[name]
	}), nil); err != nil {
		return nil, nil, fmt.Errorf("load environment config: %w", err)
	}

	var cfg Config
	if err := values.Unmarshal("", &cfg); err != nil {
		return nil, nil, fmt.Errorf("decode configuration: %w", err)
	}
	return &cfg, overrides, nil
}

func LogOverrides(overrides []Override) {
	for _, override := range overrides {
		slog.Info("configuration overridden by environment",
			"env", override.Environment,
			"key", override.Key,
			"value", override.Value,
		)
	}
}

func yamlFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read config directory %q: %w", dir, err)
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext == ".yml" || ext == ".yaml" {
			paths = append(paths, filepath.Join(dir, entry.Name()))
		}
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, fmt.Errorf("no yaml configuration files found in %q", dir)
	}
	return paths, nil
}

func environmentKeys(values *koanf.Koanf) map[string]string {
	keys := make(map[string]string, len(values.Keys()))
	for _, key := range values.Keys() {
		name := envPrefix + strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
		keys[name] = key
	}
	return keys
}

func environmentOverrides(keys map[string]string, policy redact.Policy) []Override {
	overrides := make([]Override, 0)
	for _, entry := range os.Environ() {
		name, value, ok := strings.Cut(entry, "=")
		if !ok || !strings.HasPrefix(name, envPrefix) {
			continue
		}
		key := keys[name]
		if key == "" {
			continue
		}
		if policy.IsSensitive(key) {
			value = policy.Value(key, value)
		} else if policy.IsSensitive(name) {
			value = policy.Value(name, value)
		}
		overrides = append(overrides, Override{Environment: name, Key: key, Value: value})
	}
	sort.Slice(overrides, func(left, right int) bool {
		return overrides[left].Environment < overrides[right].Environment
	})
	return overrides
}
