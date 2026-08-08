package storage

func (db *DB) GetSettings() (UserSettings, error) {
	var settings UserSettings
	err := db.sql.QueryRow(`SELECT line_width,scroll_mode,theme,vim_keys,auto_save,auto_save_every FROM user_settings WHERE id=1`).Scan(&settings.LineWidth, &settings.ScrollMode, &settings.Theme, &settings.VimKeys, &settings.AutoSave, &settings.AutoSaveEvery)
	return settings, err
}

func (db *DB) UpdateSettings(settings UserSettings) error {
	if settings.ScrollMode == "" {
		settings.ScrollMode = "line"
	}
	if settings.Theme == "" {
		settings.Theme = "dark"
	}
	if settings.LineWidth <= 0 {
		settings.LineWidth = 80
	}
	if settings.AutoSaveEvery <= 0 {
		settings.AutoSaveEvery = 5
	}
	_, err := db.sql.Exec(`UPDATE user_settings SET line_width=?,scroll_mode=?,theme=?,vim_keys=?,auto_save=?,auto_save_every=? WHERE id=1`, settings.LineWidth, settings.ScrollMode, settings.Theme, settings.VimKeys, settings.AutoSave, settings.AutoSaveEvery)
	return err
}
