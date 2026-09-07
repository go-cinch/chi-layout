package docs

import (
	"fmt"

	"github.com/knadh/koanf/parsers/yaml"
	"{{ .Computed.module_name_final }}/internal/common/config"
)

// Render an instance-local document at startup so a prebuilt binary uses its
// effective configuration rather than the server URLs embedded at build time.
func renderOpenAPI(data []byte, servers []config.HTTPDocsServersItemConfig) ([]byte, error) {
	parser := yaml.Parser()
	document, err := parser.Unmarshal(data)
	if err != nil {
		return nil, fmt.Errorf("decode openapi document: %w", err)
	}
	entries := make([]any, 0, len(servers))
	for _, item := range servers {
		entries = append(entries, map[string]any{"url": item.URL, "description": item.Description})
	}
	document["servers"] = entries
	output, err := parser.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("encode openapi document: %w", err)
	}
	return output, nil
}
