package source

import (
	"github.com/dharuncs/novel/internal/scraper"
	"os"
	"path/filepath"
	"strings"
)

func LoadDir(dir string, client *scraper.Client) (*Registry, error) {
	registry := NewRegistry()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".js") {
			continue
		}
		script, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		plugin, err := LoadScript(string(script), client)
		if err != nil {
			return nil, err
		}
		if err = registry.Add(plugin); err != nil {
			return nil, err
		}
	}
	return registry, nil
}
