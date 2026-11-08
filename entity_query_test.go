package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/dofusdude/doduapi/database"
	"github.com/hashicorp/go-memdb"
)

func setupEntityQueryTest(t *testing.T) http.Handler {
	t.Helper()
	oldDB, oldVersion := database.Db, database.Version
	t.Cleanup(func() { database.Db, database.Version = oldDB, oldVersion })
	db, err := memdb.NewMemDB(GetMemDBSchema())
	if err != nil {
		t.Fatal(err)
	}
	database.Db = db
	database.Version.MemDb = true
	txn := db.Txn(true)
	defer txn.Abort()
	fixtures := map[string][]string{
		"monsters": {
			`{"ankama_id":1,"name":{"en":"Alpha","fr":"Premier"},"grades":[{"level":50,"spells":[{"ankama_id":10}]},{"level":150,"spells":[{"ankama_id":20}]}],"spawn_locations":[{"subarea":{"ankama_id":123}}]}`,
			`{"ankama_id":2,"name":{"en":"Beta","fr":"Deuxième"},"grades":[{"level":120,"spells":[{"ankama_id":20}]}],"spawn_locations":[{"subarea":{"ankama_id":456}}]}`,
			`{"ankama_id":3,"name":{"en":"Gamma"},"grades":[{"level":300,"spells":[]}],"spawn_locations":[]}`,
		},
		"spells": {
			`{"ankama_id":10,"name":{"en":"Fire"},"levels":[{"ap_cost":2,"min_player_level":1},{"ap_cost":4,"min_player_level":100}]}`,
			`{"ankama_id":20,"name":{"en":"Water"},"levels":[{"ap_cost":3,"min_player_level":50}]}`,
		},
	}
	for resource, documents := range fixtures {
		for _, document := range documents {
			var header struct {
				ID   int               `json:"ankama_id"`
				Name map[string]string `json:"name"`
			}
			if err := json.Unmarshal([]byte(document), &header); err != nil {
				t.Fatal(err)
			}
			projection, err := projectEntityQuery([]byte(document))
			if err != nil {
				t.Fatal(err)
			}
			record := &MappedEntityRecord{AnkamaId: header.ID, Name: header.Name, Document: []byte(document), QueryProjection: projection}
			if err := txn.Insert("red-"+resource, record); err != nil {
				t.Fatal(err)
			}
		}
	}
	txn.Commit()
	return Router()
}

func queryTestRequest(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w
}

func TestEntityQueryFiltersAndGETParity(t *testing.T) {
	router := setupEntityQueryTest(t)
	filter := `{"and":[{"level":{"min":100,"max":200}},{"or":[{"subarea_ids":[123]},{"ids":[2]}]},{"spell_ids":[20]}]}`
	body := `{"filter":` + filter + `,"sort":["-name"],"page":{"number":1,"size":1}}`
	w := queryTestRequest(router, "QUERY", "/dofus3/v1/en/monsters", body)
	if w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	var result struct {
		Monsters []APIEntityListEntry `json:"monsters"`
		Links    struct {
			Next string `json:"next"`
		} `json:"_links"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Monsters) != 1 || result.Monsters[0].AnkamaID != 2 || result.Links.Next == "" {
		t.Fatalf("bad result: %s", w.Body)
	}
	if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Accept-Query") != "application/json" {
		t.Fatal("QUERY headers missing")
	}
	next := queryTestRequest(router, "QUERY", result.Links.Next, body)
	if next.Code != 200 || !strings.Contains(next.Body.String(), `"ankama_id":1`) {
		t.Fatalf("pagination lost body: %d %s", next.Code, next.Body)
	}
	get := queryTestRequest(router, "GET", "/dofus3/v1/en/monsters?filter="+url.QueryEscape(filter)+"&sort=-name&page[number]=1&page[size]=1", "")
	if get.Code != 200 {
		t.Fatalf("GET %d %s", get.Code, get.Body)
	}
	var getResult struct {
		Monsters []APIEntityListEntry `json:"monsters"`
	}
	json.Unmarshal(get.Body.Bytes(), &getResult)
	if len(getResult.Monsters) != 1 || getResult.Monsters[0].AnkamaID != result.Monsters[0].AnkamaID {
		t.Fatalf("GET/QUERY mismatch: %s", get.Body)
	}
}

func TestEntityQuerySpellLevelsMustMatchTogether(t *testing.T) {
	router := setupEntityQueryTest(t)
	w := queryTestRequest(router, "QUERY", "/dofus3/v1/en/spells/", `{"filter":{"ap_cost":{"max":2},"min_player_level":{"min":100}}}`)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"spells":[]`) {
		t.Fatalf("cross-level false positive: %d %s", w.Code, w.Body)
	}
	w = queryTestRequest(router, "QUERY", "/dofus3/v1/en/spells", `{"filter":{"ap_cost":{"min":3,"max":3}},"fields":{"spells":["all"]}}`)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"levels"`) || strings.Contains(w.Body.String(), `"ankama_id":10`) {
		t.Fatalf("bad spell filtering: %d %s", w.Code, w.Body)
	}
}

func TestEntityQueryInvalidRequests(t *testing.T) {
	router := setupEntityQueryTest(t)
	for _, body := range []string{
		`null`, `[]`, `{}`, // {} is tested separately below as valid.
		`{"unknown":1}`, `{"filter":{"unknown":1}}`, `{"filter":{"ids":[]}}`,
		`{"filter":{"level":{"min":200,"max":100}}}`, `{"filter":{"level":{}}}`,
		`{"filter":{"ap_cost":{"max":3}}}`, `{"filter":{"and":[]}}`,
		`{"page":{"size":0}}`, `{"page":{"number":-1}}`, `{"page":{"size":101}}`,
		`{"sort":["unknown"]}`, `{"sort":["name","-name"]}`, `{"include":["unknown"]}`,
		`{"filter":{"ids":[-1]}}`, `{} {}`, `{"filter":{"name":" "}}`,
		`{"filter":null}`, `{"page":null}`, `{"filter":{"ids":null}}`, `{"filter":{},"filter":{"ids":[1]}}`,
	} {
		w := queryTestRequest(router, "QUERY", "/dofus3/v1/en/monsters", body)
		if body == `{}` {
			if w.Code != 200 {
				t.Fatalf("empty query should browse: %d %s", w.Code, w.Body)
			}
			continue
		}
		if w.Code != 400 {
			t.Errorf("%s: expected 400, got %d %s", body, w.Code, w.Body)
		}
	}
	w := queryTestRequest(router, "QUERY", "/dofus3/v1/en/monsters", `{"padding":"`+strings.Repeat("x", maxEntityQueryBytes)+`"}`)
	if w.Code != 413 {
		t.Fatalf("size limit: %d", w.Code)
	}
	r := httptest.NewRequest("QUERY", "/dofus3/v1/en/monsters", strings.NewReader(`{}`))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != 415 {
		t.Fatalf("media type: %d", w.Code)
	}
	deep := `{}`
	for i := 0; i < 9; i++ {
		deep = `{"and":[` + deep + `]}`
	}
	w = queryTestRequest(router, "QUERY", "/dofus3/v1/en/monsters", `{"filter":`+deep+`}`)
	if w.Code != 400 {
		t.Fatal("depth limit missing")
	}
	wide := strings.TrimSuffix(strings.Repeat(`{},`, 64), ",")
	w = queryTestRequest(router, "QUERY", "/dofus3/v1/en/monsters", `{"filter":{"and":[`+wide+`]}}`)
	if w.Code != 400 {
		t.Fatal("node limit missing")
	}
	ids := strings.TrimSuffix(strings.Repeat(`1,`, 101), ",")
	w = queryTestRequest(router, "QUERY", "/dofus3/v1/en/monsters", `{"filter":{"ids":[`+ids+`]}}`)
	if w.Code != 400 {
		t.Fatal("ID list limit missing")
	}
	w = queryTestRequest(router, "QUERY", "/dofus3/v1/meta/version", `{}`)
	if w.Code != 405 {
		t.Fatalf("unimplemented collection: %d", w.Code)
	}
}

func TestEntityQueryPreflightAndDiscovery(t *testing.T) {
	router := setupEntityQueryTest(t)
	r := httptest.NewRequest("OPTIONS", "/dofus3/v1/en/monsters", nil)
	r.Header.Set("Origin", "http://localhost:5173")
	r.Header.Set("Access-Control-Request-Method", "QUERY")
	r.Header.Set("Access-Control-Request-Headers", "content-type")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != 204 || !strings.Contains(w.Header().Get("Access-Control-Allow-Methods"), "QUERY") {
		t.Fatalf("preflight: %d %v", w.Code, w.Header())
	}
	w = queryTestRequest(router, "OPTIONS", "/dofus3/v1/en/spells", "")
	if w.Code != 204 || w.Header().Get("Accept-Query") != "application/json" {
		t.Fatalf("discovery: %d %v", w.Code, w.Header())
	}
	r = httptest.NewRequest("OPTIONS", "/dofus3/v1/meta/version", nil)
	r.Header.Set("Origin", "http://localhost:5173")
	r.Header.Set("Access-Control-Request-Method", "QUERY")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != 405 {
		t.Fatalf("unsupported QUERY preflight: %d", w.Code)
	}
}

func TestEntityQueryMonsterIncludesUseExistingGradeRenderer(t *testing.T) {
	router := setupEntityQueryTest(t)
	document := json.RawMessage(`{"ankama_id":9,"name":{"en":"Included"},"grades":[{"grade":1,"level":10,"spells":[{"ankama_id":10,"name":{"en":"Fire"},"levels":[{"grade":2,"ap_cost":3}]}]}]}`)
	projection, err := projectEntityQuery(document)
	if err != nil {
		t.Fatal(err)
	}
	txn := database.Db.Txn(true)
	if err := txn.Insert("red-monsters", &MappedEntityRecord{AnkamaId: 9, Name: map[string]string{"en": "Included"}, Document: document, QueryProjection: projection}); err != nil {
		txn.Abort()
		t.Fatal(err)
	}
	txn.Commit()
	w := queryTestRequest(router, "QUERY", "/dofus3/v1/en/monsters", `{"filter":{"ids":[9]},"include":["spells.grade"]}`)
	if w.Code != 200 {
		t.Fatalf("include: %d %s", w.Code, w.Body)
	}
	var result struct {
		Monsters []struct {
			Grades []struct {
				Spells []struct {
					Levels []struct {
						Grade int `json:"grade"`
					} `json:"levels"`
				} `json:"spells"`
			} `json:"grades"`
		} `json:"monsters"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Monsters) != 1 || len(result.Monsters[0].Grades) != 1 || len(result.Monsters[0].Grades[0].Spells) != 1 || len(result.Monsters[0].Grades[0].Spells[0].Levels) != 1 || result.Monsters[0].Grades[0].Spells[0].Levels[0].Grade != 2 {
		t.Fatalf("grade include lost: %s", w.Body)
	}
}
