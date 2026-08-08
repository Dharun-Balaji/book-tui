package main

import (
	"fmt"

	"github.com/dharuncs/novel/internal/storage"
	"github.com/dharuncs/novel/internal/tui/settings"
)

func main() {
	// 1. Render Settings View (Dark Theme)
	fmt.Println("=== SETTINGS VIEW (DARK THEME) ===")
	darkSet := settings.New().SetSettings(storage.UserSettings{
		Theme:         "dark",
		LineWidth:     80,
		AutoSaveEvery: 5,
	}).SetSize(80, 10)
	fmt.Println(darkSet.View())

	// 2. Render Settings View (Light Theme)
	fmt.Println("\n=== SETTINGS VIEW (LIGHT THEME) ===")
	lightSet := settings.New().SetSettings(storage.UserSettings{
		Theme:         "light",
		LineWidth:     100,
		AutoSaveEvery: 10,
	}).SetSize(80, 10)
	fmt.Println(lightSet.View())
}
