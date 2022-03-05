package sqlite_test

import (
	"bytes"
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/mtlynch/picoshare/v2/store/sqlite"
	"github.com/mtlynch/picoshare/v2/types"
)

func TestInsertDeleteSingleEntry(t *testing.T) {
	// TODO: Figure out why this breaks otherwise
	//db := sqlite.New("file::memory:?cache=shared")
	db := sqlite.New(":memory:")

	if err := db.InsertEntry(bytes.NewBufferString("hello, world!"), types.UploadMetadata{
		ID:       types.EntryID("dummy-id"),
		Filename: "dummy-file.txt",
		Expires:  mustParseExpirationTime("2040-01-01T00:00:00Z"),
	}); err != nil {
		t.Fatalf("failed to insert file into sqlite: %v", err)
	}

	entry, err := db.GetEntry(types.EntryID("dummy-id"))
	if err != nil {
		t.Fatalf("failed to get entry from DB: %v", err)
	}

	m1, err := db.GetEntriesMetadata()
	if err != nil {
		t.Fatalf("failed to get entry metadata: %v", err)
	} else {
		log.Printf("read entry metadata: %v", m1)
	}
	contents, err := ioutil.ReadAll(entry.Reader)
	if err != nil {
		t.Fatalf("failed to read entry contents: %v", err)
	}

	m, err := db.GetEntriesMetadata()
	if err != nil {
		t.Fatalf("failed to get entry metadata: %v", err)
	} else {
		log.Printf("read entry metadata: %v", m)
	}
	expected := "hello, world!"
	if string(contents) != expected {
		log.Fatalf("unexpected file contents: got %v, want %v", string(contents), expected)
	}

	meta, err := db.GetEntriesMetadata()
	if err != nil {
		t.Fatalf("failed to get entry metadata: %v", err)
	}

	if meta[0].Size != len(expected) {
		t.Fatalf("unexpected file size in entry metadata: got %v, want %v", meta[0].Size, len(expected))
	}
}

func mustParseExpirationTime(s string) types.ExpirationTime {
	et, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return types.ExpirationTime(et)
}
