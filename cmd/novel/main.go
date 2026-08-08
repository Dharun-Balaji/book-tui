package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dharuncs/novel/internal/core"
	"github.com/dharuncs/novel/internal/scraper"
	"github.com/dharuncs/novel/internal/source"
	"github.com/dharuncs/novel/internal/storage"
	"github.com/dharuncs/novel/internal/tui"
	"github.com/spf13/cobra"
)

func main() {
	var url, sources string
	var dbPath string

	root := &cobra.Command{
		Use:   "novel",
		Short: "Novel TUI reader",
		RunE: func(_ *cobra.Command, _ []string) error {
			if dbPath == "" {
				home, _ := os.UserHomeDir()
				dir := filepath.Join(home, ".config", "novel")
				os.MkdirAll(dir, 0755)
				dbPath = filepath.Join(dir, "novel.db")
			}

			db, err := storage.Open(dbPath)
			if err != nil {
				return fmt.Errorf("open storage: %w", err)
			}
			defer db.Close()

			client := scraper.NewClient()
			registry, err := source.LoadDir(sources, client)
			if err != nil {
				return fmt.Errorf("load sources: %w", err)
			}

			lm := core.NewLibraryManager(db)
			pm := core.NewProgressManager(db)
			sm := core.NewStatsManager(db)

			app := tui.NewAppModel(lm, pm, sm, registry, db)
			p := tea.NewProgram(app, tea.WithAltScreen())
			_, err = p.Run()
			return err
		},
	}

	root.PersistentFlags().StringVar(&dbPath, "db", "", "Path to SQLite database")
	root.PersistentFlags().StringVar(&sources, "sources", "sources", "Plugin directory")

	test := &cobra.Command{
		Use: "test novelfire",
		RunE: func(_ *cobra.Command, _ []string) error {
			client := scraper.NewClient()
			registry, err := source.LoadDir(sources, client)
			if err != nil {
				return err
			}
			plugin, ok := registry.Get("novelfire")
			if !ok {
				return fmt.Errorf("novelfire source not found")
			}
			novel, err := plugin.NovelInfo(context.Background(), url)
			if err != nil {
				return err
			}
			chapters, err := plugin.ChapterList(context.Background(), url)
			if err != nil {
				return err
			}
			fmt.Printf("Title: %s\nAuthor: %s\nChapters: %d\n", novel.Title, novel.Author, len(chapters))
			for i, chapter := range chapters {
				if i == 5 {
					break
				}
				fmt.Printf("%g: %s — %s\n", chapter.Number, chapter.Title, chapter.URL)
			}
			return nil
		},
	}

	test.Flags().StringVar(&url, "url", "", "Novel URL")
	test.MarkFlagRequired("url")

	sourceCmd := &cobra.Command{Use: "source"}
	sourceCmd.AddCommand(test)
	root.AddCommand(sourceCmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
