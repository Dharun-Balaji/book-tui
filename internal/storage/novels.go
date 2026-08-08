package storage

import (
	"database/sql"
	"time"
)

func (db *DB) UpsertSource(source Source) error {
	_, err := db.sql.Exec(`INSERT INTO sources (id,name,version,base_url,language,needs_js,rate_limit) VALUES (?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET name=excluded.name,version=excluded.version,base_url=excluded.base_url,language=excluded.language,needs_js=excluded.needs_js,rate_limit=excluded.rate_limit`, source.ID, source.Name, source.Version, source.BaseURL, source.Language, source.NeedsJS, source.RateLimit)
	return err
}

func (db *DB) CreateNovel(novel Novel) error { return db.saveNovel(novel, false) }
func (db *DB) UpdateNovel(novel Novel) error { return db.saveNovel(novel, true) }
func (db *DB) saveNovel(novel Novel, update bool) error {
	transaction, err := db.sql.Begin()
	if err != nil {
		return err
	}
	defer transaction.Rollback()
	if update {
		_, err = transaction.Exec(`UPDATE novels SET source_id=?,source_url=?,title=?,author=?,cover_url=?,description=?,status=?,total_chapters=?,in_library=?,updated_at=? WHERE id=?`, novel.SourceID, novel.SourceURL, novel.Title, novel.Author, novel.CoverURL, novel.Description, novel.Status, novel.TotalChapters, novel.InLibrary, timestamp(novel.UpdatedAt), novel.ID)
	} else {
		_, err = transaction.Exec(`INSERT INTO novels (id,source_id,source_url,title,author,cover_url,description,status,total_chapters,in_library,added_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, novel.ID, novel.SourceID, novel.SourceURL, novel.Title, novel.Author, novel.CoverURL, novel.Description, novel.Status, novel.TotalChapters, novel.InLibrary, timestamp(novel.AddedAt), timestamp(novel.UpdatedAt))
	}
	if err != nil {
		return err
	}
	if _, err = transaction.Exec("DELETE FROM novel_tags WHERE novel_id=?", novel.ID); err != nil {
		return err
	}
	for _, tag := range novel.Tags {
		if _, err = transaction.Exec("INSERT INTO novel_tags (novel_id,tag) VALUES (?,?)", novel.ID, tag); err != nil {
			return err
		}
	}
	return transaction.Commit()
}
func (db *DB) GetNovel(id string) (Novel, error) {
	return db.scanNovel(db.sql.QueryRow(novelQuery+" WHERE n.id=? GROUP BY n.id", id))
}
func (db *DB) ListNovels(inLibraryOnly bool) ([]Novel, error) {
	query := novelQuery
	args := []any{}
	if inLibraryOnly {
		query += " WHERE n.in_library=1"
	}
	query += " GROUP BY n.id ORDER BY n.title COLLATE NOCASE"
	rows, err := db.sql.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	novels := []Novel{}
	for rows.Next() {
		novel, scanErr := db.scanNovel(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		novels = append(novels, novel)
	}
	return novels, rows.Err()
}
func (db *DB) DeleteNovel(id string) error {
	_, err := db.sql.Exec("DELETE FROM novels WHERE id=?", id)
	return err
}

const novelQuery = `SELECT n.id,n.source_id,n.source_url,n.title,n.author,n.cover_url,n.description,n.status,n.total_chapters,n.in_library,n.added_at,n.updated_at,COALESCE(group_concat(t.tag, char(31)), '') FROM novels n LEFT JOIN novel_tags t ON t.novel_id=n.id`

type scanner interface{ Scan(...any) error }

func (db *DB) scanNovel(row scanner) (Novel, error) {
	var n Novel
	var tags, added, updated string
	err := row.Scan(&n.ID, &n.SourceID, &n.SourceURL, &n.Title, &n.Author, &n.CoverURL, &n.Description, &n.Status, &n.TotalChapters, &n.InLibrary, &added, &updated, &tags)
	if err != nil {
		return n, err
	}
	n.AddedAt, _ = time.Parse(time.RFC3339Nano, added)
	n.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	if tags != "" {
		for _, tag := range splitTags(tags) {
			n.Tags = append(n.Tags, tag)
		}
	}
	return n, nil
}
func timestamp(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
func splitTags(tags string) []string {
	result := []string{}
	start := 0
	for index := range tags {
		if tags[index] == 31 {
			result = append(result, tags[start:index])
			start = index + 1
		}
	}
	return append(result, tags[start:])
}

var _ = sql.ErrNoRows
