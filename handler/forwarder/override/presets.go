package override

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/go-logr/logr"
	"gopkg.in/yaml.v3"
)

var (
	//go:embed presets/*
	content embed.FS
	presets = make(map[string][]*Rule)

	errNotFound = errors.New("preset not found")
)

const presetDir = "presets"

func init() {
	err := fs.WalkDir(content, presetDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(d.Name(), ".yaml") {
			return nil
		}

		name := strings.TrimSuffix(strings.TrimPrefix(path, presetDir+"/"), ".yaml")

		data, err := content.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to load ruleset %q: %w", name, err)
		}

		var rules []*Rule
		if err := yaml.Unmarshal(data, &rules); err != nil {
			return fmt.Errorf("failed to unmarshal %q: %w", name, err)
		}
		presets[name] = rules

		s := &Set{
			Rules: rules,
		}

		if err := s.Validate(); err != nil {
			return fmt.Errorf("failed to validate preset %q: %w", name, err)
		}

		return nil
	})
	if err != nil {
		panic(err)
	}
}

func LoadPresets(logger logr.Logger, names ...string) (ss []*Set, err error) {
	for _, name := range names {
		rules, ok := presets[name]
		if !ok {
			return nil, errNotFound
		}

		ss = append(ss, &Set{
			Logger: logger.WithValues("set", name),
			Rules:  rules,
		})
	}
	return
}
