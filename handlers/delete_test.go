package handlers_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mtlynch/picoshare/v2/handlers"
	"github.com/mtlynch/picoshare/v2/store"
	"github.com/mtlynch/picoshare/v2/store/sqlite_test"
	"github.com/mtlynch/picoshare/v2/types"
)

func TestDeleteExistingFile(t *testing.T) {
	dataStore := sqlite_test.New()
	dataStore.InsertEntry(makeData("dummy data"),
		types.UploadMetadata{
			ID: types.EntryID("hR87apiUCjTV9E"),
		})

	s := handlers.New(mockAuthenticator{}, dataStore)

	req, err := http.NewRequest("DELETE", "/api/entry/hR87apiUCjTV9E", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Fatalf("DELETE /api/entry returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	_, err = dataStore.GetEntry(types.EntryID("hR87apiUCjTV9E"))
	if _, ok := err.(store.EntryNotFoundError); !ok {
		t.Fatalf("expected entry %v to be deleted", types.EntryID("hR87apiUCjTV9E"))
	}
}

func TestDeleteNonExistentFile(t *testing.T) {
	dataStore := sqlite_test.New()

	s := handlers.New(mockAuthenticator{}, dataStore)

	req, err := http.NewRequest("DELETE", "/api/entry/hR87apiUCjTV9E", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	// File doesn't exist, but there's no error for deleting a non-existent file.
	if status := w.Code; status != http.StatusOK {
		t.Fatalf("DELETE /api/entry returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func TestDeleteInvalidEntryID(t *testing.T) {
	dataStore := sqlite_test.New()

	s := handlers.New(mockAuthenticator{}, dataStore)

	req, err := http.NewRequest("DELETE", "/api/entry/invalid-entry-id", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, req)

	// File doesn't exist, but there's no error for deleting a non-existent file.
	if status := w.Code; status != http.StatusBadRequest {
		t.Fatalf("DELETE /api/entry returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}
}

func makeData(s string) io.Reader {
	return bytes.NewReader([]byte(s))
}
