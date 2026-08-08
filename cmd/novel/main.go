package main

import (
	"context"
	"fmt"
	"github.com/dharuncs/novel/internal/scraper"
	"github.com/dharuncs/novel/internal/source"
	"github.com/spf13/cobra"
	"os"
)

func main() {
	var url, sources string
	root := &cobra.Command{Use: "novel"}
	test := &cobra.Command{Use: "test novelfire", RunE: func(_ *cobra.Command, _ []string) error {
		registry, err := source.LoadDir(sources, scraper.NewClient())
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
	}}
	test.Flags().StringVar(&url, "url", "", "Novel URL")
	test.Flags().StringVar(&sources, "sources", "sources", "Plugin directory")
	test.MarkFlagRequired("url")
	sourceCmd := &cobra.Command{Use: "source"}
	sourceCmd.AddCommand(test)
	root.AddCommand(sourceCmd)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
