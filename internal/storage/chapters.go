package storage

import (
	"database/sql"
	"time"
)

func (db *DB) CreateChapter(chapter Chapter) error {
	_, err := db.sql.Exec(`INSERT INTO chapters (id,novel_id,source_url,number,title,content,word_count,fetched_at,is_cached) VALUES (?,?,?,?,?,?,?,?,?)`, chapter.ID, chapter.NovelID, chapter.SourceURL, chapter.Number, chapter.Title, chapter.Content, chapter.WordCount, nullableTime(chapter.FetchedAt), chapter.IsCached)
	return err
}
func (db *DB) UpdateChapter(chapter Chapter) error {
	_, err := db.sql.Exec(`UPDATE chapters SET source_url=?,number=?,title=?,content=?,word_count=?,fetched_at=?,is_cached=? WHERE id=?`, chapter.SourceURL, chapter.Number, chapter.Title, chapter.Content, chapter.WordCount, nullableTime(chapter.FetchedAt), chapter.IsCached, chapter.ID)
	return err
}
func (db *DB) GetChapter(id string) (Chapter, error) {
	return scanChapter(db.sql.QueryRow(chapterQuery+" WHERE id=?", id))
}
func (db *DB) ListChapters(novelID string) ([]Chapter, error) {
	rows, err := db.sql.Query(chapterQuery+" WHERE novel_id=? ORDER BY number", novelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	chapters := []Chapter{}
	for rows.Next() {
		chapter, scanErr := scanChapter(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		chapters = append(chapters, chapter)
	}
	return chapters, rows.Err()
}
func (db *DB) DeleteChapter(id string) error {
	_, err := db.sql.Exec("DELETE FROM chapters WHERE id=?", id)
	return err
}

const chapterQuery = `SELECT id,novel_id,source_url,number,title,content,word_count,fetched_at,is_cached FROM chapters`

func scanChapter(row scanner) (Chapter, error) {
	var chapter Chapter
	var fetched sql.NullString
	err := row.Scan(&chapter.ID, &chapter.NovelID, &chapter.SourceURL, &chapter.Number, &chapter.Title, &chapter.Content, &chapter.WordCount, &fetched, &chapter.IsCached)
	if fetched.Valid {
		value, parseErr := time.Parse(time.RFC3339Nano, fetched.String)
		if parseErr != nil {
			return chapter, parseErr
		}
		chapter.FetchedAt = &value
	}
	return chapter, err
}
func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return timestamp(*value)
}
