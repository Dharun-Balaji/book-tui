package storage

import (
	"database/sql"
	"time"
)

func (db *DB) CreateHistory(entry HistoryEntry) error {
	_, err := db.sql.Exec(`INSERT INTO history (id,novel_id,chapter_id,opened_at,closed_at,session_sec) VALUES (?,?,?,?,?,?)`, entry.ID, entry.NovelID, entry.ChapterID, timestamp(entry.OpenedAt), nullableTime(entry.ClosedAt), entry.SessionSec)
	return err
}
func (db *DB) GetHistory(id string) (HistoryEntry, error) {
	return scanHistory(db.sql.QueryRow(`SELECT id,novel_id,chapter_id,opened_at,closed_at,session_sec FROM history WHERE id=?`, id))
}
func (db *DB) CloseHistory(id string, closedAt time.Time, sessionSec int64) error {
	_, err := db.sql.Exec("UPDATE history SET closed_at=?,session_sec=? WHERE id=?", timestamp(closedAt), sessionSec, id)
	return err
}
func (db *DB) ListHistory(novelID string) ([]HistoryEntry, error) {
	rows, err := db.sql.Query(`SELECT id,novel_id,chapter_id,opened_at,closed_at,session_sec FROM history WHERE novel_id=? ORDER BY opened_at DESC`, novelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []HistoryEntry{}
	for rows.Next() {
		entry, scanErr := scanHistory(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}
func scanHistory(row scanner) (HistoryEntry, error) {
	var entry HistoryEntry
	var opened string
	var closed sql.NullString
	err := row.Scan(&entry.ID, &entry.NovelID, &entry.ChapterID, &opened, &closed, &entry.SessionSec)
	if err != nil {
		return entry, err
	}
	entry.OpenedAt, err = time.Parse(time.RFC3339Nano, opened)
	if err != nil {
		return entry, err
	}
	if closed.Valid {
		value, parseErr := time.Parse(time.RFC3339Nano, closed.String)
		if parseErr != nil {
			return entry, parseErr
		}
		entry.ClosedAt = &value
	}
	return entry, nil
}
