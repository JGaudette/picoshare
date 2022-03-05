package sqlite

import (
	"context"
	"database/sql"
	"io"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/mtlynch/picoshare/v2/store"
	"github.com/mtlynch/picoshare/v2/store/sqlite/file"
	"github.com/mtlynch/picoshare/v2/types"
)

const (
	timeFormat = time.RFC3339
)

type db struct {
	ctx *sql.DB
}

func (d db) Prepare(query string) (*sql.Stmt, error) {
	return d.ctx.Prepare(query)
}

func (d db) Exec(query string, args ...interface{}) (sql.Result, error) {
	return d.ctx.Exec(query, args)
}

func New(path string) store.Store {
	log.Printf("reading DB from %s", path)
	ctx, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatalln(err)
	}

	initStmts := []string{
		`CREATE TABLE IF NOT EXISTS entries (
			id TEXT PRIMARY KEY,
			filename TEXT,
			upload_time TEXT,
			expiration_time TEXT
			)`,
		`CREATE TABLE IF NOT EXISTS entries_data (
			id TEXT,
			chunk_index INTEGER,
			chunk BLOB,
			FOREIGN KEY(id) REFERENCES entries(id)
			)`,
	}
	for _, stmt := range initStmts {
		_, err = ctx.Exec(stmt)
		if err != nil {
			log.Fatalln(err)
		}
	}

	return &db{
		ctx: ctx,
	}
}

func (d db) GetEntriesMetadata() ([]types.UploadMetadata, error) {
	rows, err := d.ctx.Query(`
	SELECT
		id,
		filename,
		upload_time,
		expiration_time
	FROM
		entries`)
	if err != nil {
		return []types.UploadMetadata{}, err
	}

	ee := []types.UploadMetadata{}
	for rows.Next() {
		var id string
		var filename string
		var uploadTimeRaw string
		var expirationTimeRaw string
		err = rows.Scan(&id, &filename, &uploadTimeRaw, &expirationTimeRaw)
		if err != nil {
			return []types.UploadMetadata{}, err
		}

		ut, err := parseDatetime(uploadTimeRaw)
		if err != nil {
			return []types.UploadMetadata{}, err
		}

		et, err := parseDatetime(expirationTimeRaw)
		if err != nil {
			return []types.UploadMetadata{}, err
		}

		ee = append(ee, types.UploadMetadata{
			ID:       types.EntryID(id),
			Filename: types.Filename(filename),
			Uploaded: ut,
			Expires:  types.ExpirationTime(et),
			Size:     0, // TODO: Replace
		})
	}

	return ee, nil
}

func (d db) GetEntry(id types.EntryID) (types.UploadEntry, error) {
	stmt, err := d.ctx.Prepare(`
		SELECT
			filename,
			upload_time,
			expiration_time
		FROM
			entries
		WHERE
			id=? AND
			-- TODO: Purge expired records instead of filtering them here.
			expiration_time >= strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
			`)
	if err != nil {
		return types.UploadEntry{}, err
	}
	defer stmt.Close()

	var filename string
	var uploadTimeRaw string
	var expirationTimeRaw string
	err = stmt.QueryRow(id).Scan(&filename, &uploadTimeRaw, &expirationTimeRaw)
	if err == sql.ErrNoRows {
		return types.UploadEntry{}, store.EntryNotFoundError{ID: id}
	} else if err != nil {
		return types.UploadEntry{}, err
	}

	ut, err := parseDatetime(uploadTimeRaw)
	if err != nil {
		return types.UploadEntry{}, err
	}

	et, err := parseDatetime(expirationTimeRaw)
	if err != nil {
		return types.UploadEntry{}, err
	}

	r, err := file.NewReader(d, id)
	if err != nil {
		return types.UploadEntry{}, err
	}

	return types.UploadEntry{
		UploadMetadata: types.UploadMetadata{
			ID:       id,
			Filename: types.Filename(filename),
			Uploaded: ut,
			Expires:  types.ExpirationTime(et),
		},
		Reader: r,
	}, nil
}

func (d db) InsertEntry(reader io.Reader, metadata types.UploadMetadata) error {
	log.Printf("saving new entry %s", metadata.ID)

	writeFileData := func() error {
		chunkSize := 32 << 20
		w := file.NewWriter(d.ctx, metadata.ID, chunkSize)
		_, err := io.Copy(w, reader)
		if err != nil {
			return err
		}
		return w.Close()
	}
	if err := writeFileData(); err != nil {
		return err
	}

	_, err := d.ctx.Exec(`
	INSERT INTO
		entries
	(
		id,
		filename,
		upload_time,
		expiration_time
	)
	VALUES(?,?,?,?)`, metadata.ID, metadata.Filename, formatTime(metadata.Uploaded), formatTime(time.Time(metadata.Expires)))
	if err != nil {
		return err
	}

	return nil
}

func (d db) DeleteEntry(id types.EntryID) error {
	log.Printf("deleting entry %v", id)

	tx, err := d.ctx.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
	DELETE FROM
		entries
	WHERE
		id=?`, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
	DELETE FROM
		entries_data
	WHERE
		id=?`, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func formatTime(t time.Time) string {
	return t.UTC().Format(timeFormat)
}

func parseDatetime(s string) (time.Time, error) {
	return time.Parse(timeFormat, s)
}
