package utils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPersistenceHTTPAndFilesUseSameParser(t *testing.T) {
	files := map[string]string{"elements.json": `["Strength","Agility"]`, "types.json": `["hat","ring"]`}
	dir := t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, files[r.URL.Path[1:]]) }))
	defer server.Close()
	localElements, localTypes, err := LoadPersistedElementsFiles(filepath.Join(dir, "elements.json"), filepath.Join(dir, "types.json"))
	if err != nil {
		t.Fatal(err)
	}
	remoteElements, remoteTypes, err := LoadPersistedElements(server.URL+"/elements.json", server.URL+"/types.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]PersistentStringKeysMap{{localElements, remoteElements}, {localTypes, remoteTypes}} {
		if pair[0].NextId != 2 || pair[1].NextId != 2 || !reflect.DeepEqual(pair[0].Entries.Values(), pair[1].Entries.Values()) {
			t.Fatal("local and remote persistence disagree")
		}
		for id := 0; id < 2; id++ {
			local, _ := pair[0].Entries.Get(id)
			remote, _ := pair[1].Entries.Get(id)
			if local != remote {
				t.Fatalf("ID %d changed: %v != %v", id, local, remote)
			}
		}
	}
}

func TestPersistenceRejectsHTTPErrorAndMalformedJSON(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
	}{{404, `[]`}, {200, `{"not":"an array"}`}} {
		t.Run(fmt.Sprint(tc.status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); fmt.Fprint(w, tc.body) }))
			defer server.Close()
			if _, _, err := LoadPersistedElements(server.URL, server.URL); err == nil {
				t.Fatal("invalid persistence accepted")
			}
		})
	}
}
