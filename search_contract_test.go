package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dofusdude/doduapi/config"
	"github.com/dofusdude/doduapi/errmsg"
)

func TestGlobalSearchBackendFailuresReturnOneV1Error(t *testing.T) {
	var requests atomic.Int32
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		fmt.Fprint(w, `{"message":"test backend failure","code":"invalid_search_filter","type":"invalid_request"}`)
	}))
	defer backend.Close()
	oldHost, oldKey := config.MeiliHost, config.MeiliKey
	config.MeiliHost, config.MeiliKey = backend.URL, ""
	t.Cleanup(func() { config.MeiliHost, config.MeiliKey = oldHost, oldKey })
	router := setupEntityQueryTest(t)
	for _, entry := range []struct{ method, path, body string }{
		{"GET", "/dofus3/v1/en/search?query=fixture&filter[search_index]=items-equipment,sets", ""},
		{"QUERY", "/dofus3/v1/en/search", `{"query":"fixture","filter":{"search_index":["items-equipment","sets"]}}`},
	} {
		result := make(chan *httptest.ResponseRecorder, 1)
		go func() { result <- queryTestRequest(router, entry.method, entry.path, entry.body) }()
		select {
		case w := <-result:
			var apiErr errmsg.ApiError
			if w.Code != 500 || json.Unmarshal(w.Body.Bytes(), &apiErr) != nil || apiErr.Status != 500 || apiErr.Code != errmsg.ERR_SERVER_ERROR {
				t.Fatalf("%s: expected one v1 error, got %d %s", entry.method, w.Code, w.Body)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("global search did not finish after backend failure")
		}
	}
	if requests.Load() != 4 {
		t.Fatalf("both requested indexes must be queried, got %d requests", requests.Load())
	}
}
