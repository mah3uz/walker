package modules

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
)

type Themes struct {
	general config.GeneralModule
	entries []util.Entry
}

func (t *Themes) General() *config.GeneralModule {
	return &t.general
}

func (t Themes) Cleanup() {}

func (t *Themes) Setup(cfg *config.Config) bool {
	t.general = cfg.Builtins.Themes.GeneralModule

	return true
}

func (t *Themes) SetupData(cfg *config.Config, ctx context.Context) {
	t.entries = []util.Entry{}

	t.addLocalThemes()
}

func (t *Themes) addLocalThemes() {
	themeDir := util.ThemeDir()

	filepath.WalkDir(themeDir, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)

		if ext != ".css" {
			return nil
		}

		t.entries = append(t.entries, util.Entry{
			Label:            strings.TrimSuffix(filepath.Base(path), ext),
			Sub:              "Themes",
			Categories:       []string{"themes"},
			Class:            "themes",
			Matching:         util.Fuzzy,
			RecalculateScore: true,
		})

		return nil
	})
}

func (t Themes) Entries(ctx context.Context, term string) []util.Entry {
	return t.entries
}

func (t *Themes) Refresh() {
	t.general.IsSetup = false
}
