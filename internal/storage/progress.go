package storage

import (
	"database/sql"
	"time"
)

func (db *DB) SaveProgress(progress ReadingProgress) error {
	_, err := db.sql.Exec(`INSERT INTO reading_progress (novel_id,chapter_id,chapter_num,paragraph_idx,scroll_offset,progress_pct,chapters_read,total_read_sec,last_read_at) VALUES (?,?,?,?,?,?,?,?,?) ON CONFLICT(novel_id) DO UPDATE SET chapter_id=excluded.chapter_id,chapter_num=excluded.chapter_num,paragraph_idx=excluded.paragraph_idx,scroll_offset=excluded.scroll_offset,progress_pct=excluded.progress_pct,chapters_read=excluded.chapters_read,total_read_sec=excluded.total_read_sec,last_read_at=excluded.last_read_at`, progress.NovelID, progress.ChapterID, progress.ChapterNum, progress.ParagraphIdx, progress.ScrollOffset, progress.ProgressPct, progress.ChaptersRead, progress.TotalReadSec, timestamp(progress.LastReadAt))
	return err
}
func (db *DB) GetProgress(novelID string) (*ReadingProgress, error) {
	var p ReadingProgress
	var last string
	err := db.sql.QueryRow(`SELECT novel_id,chapter_id,chapter_num,paragraph_idx,scroll_offset,progress_pct,chapters_read,total_read_sec,last_read_at FROM reading_progress WHERE novel_id=?`, novelID).Scan(&p.NovelID, &p.ChapterID, &p.ChapterNum, &p.ParagraphIdx, &p.ScrollOffset, &p.ProgressPct, &p.ChaptersRead, &p.TotalReadSec, &last)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.LastReadAt, err = time.Parse(time.RFC3339Nano, last)
	return &p, err
}
