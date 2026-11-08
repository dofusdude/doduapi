package apitest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type Options struct {
	BaseURL, Game string
	Languages     []string
	Images        bool
	UpdateToken   string
	Routes        []Route
}
type Route struct{ Method, Pattern string }
type Suite struct {
	options  Options
	client   *http.Client
	data     *Data
	covered  map[string]bool
	requests map[string]bool
}
type Response struct {
	Status  int
	Headers http.Header
	Value   any
}
type RawJSON string

func (s *Suite) call(t *testing.T, method, path string, body any, status int) Response {
	t.Helper()
	target := s.options.BaseURL + path
	if strings.HasPrefix(path, "http") {
		u, err := url.Parse(path)
		if err != nil {
			t.Fatal(err)
		}
		base, _ := url.Parse(s.options.BaseURL)
		if u.Host != base.Host || u.Scheme != base.Scheme {
			t.Fatalf("API link leaves test origin: %s", path)
		}
		target = path
	}
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if raw, ok := body.(RawJSON); ok {
			encoded, err = []byte(raw), nil
		}
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	}
	r, err := http.NewRequest(method, target, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	response, err := s.client.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	const maxBytes = 512 * 1024 * 1024
	encoded, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) > maxBytes {
		t.Fatalf("response exceeds 512 MiB: %s", path)
	}
	if response.StatusCode != status {
		t.Fatalf("%s %s: expected %d, got %d: %.1200s", method, path, status, response.StatusCode, encoded)
	}
	if status >= 200 && status < 300 || method == "POST" && status == 400 && s.options.UpdateToken != "" {
		s.requests[method+" "+r.URL.Path] = true
	}
	if status == 204 {
		return Response{Status: status, Headers: response.Header}
	}
	if !strings.Contains(response.Header.Get("Content-Type"), "application/json") {
		t.Fatalf("%s %s: missing JSON content type", method, path)
	}
	var value any
	if err := json.Unmarshal(encoded, &value); err != nil {
		t.Fatalf("%s %s: invalid JSON: %v", method, path, err)
	}
	if status >= 400 {
		object := Obj(value)
		for _, key := range []string{"status", "code", "message", "error"} {
			if object[key] == nil {
				t.Errorf("error response missing %s", key)
			}
		}
		if Number(object["status"]) != status {
			t.Error("error status field does not match HTTP status")
		}
	}
	return Response{Status: status, Headers: response.Header, Value: value}
}

func (s *Suite) prefix(language string) string { return "/" + s.options.Game + "/v1/" + language }
func rows(value any, key string) []any {
	if a, ok := value.([]any); ok {
		return a
	}
	return List(Obj(value)[key])
}
func IDs(records []Record, predicate func(Record) bool) []int {
	result := []int{}
	for _, record := range records {
		if predicate == nil || predicate(record) {
			result = append(result, record.ID)
		}
	}
	return result
}
func ResponseIDs(value any, key string) []int {
	result := []int{}
	for _, row := range rows(value, key) {
		result = append(result, Number(Obj(row)["ankama_id"]))
	}
	return result
}
func name(r Record, lang string) string {
	if r.Names[lang] != "" {
		return r.Names[lang]
	}
	return r.Names["en"]
}

func sameIDs(t *testing.T, got, want []int) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %d records, got %d (first IDs: %v)", len(want), len(got), got[:min(10, len(got))])
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("record %d: expected ID %d, got %d", i, want[i], got[i])
		}
	}
}

func (s *Suite) parity(t *testing.T, path, key string, parameters url.Values, body any, status int) any {
	t.Helper()
	get := s.call(t, "GET", path+"?"+parameters.Encode(), nil, status)
	query := s.call(t, "QUERY", path, body, status)
	if query.Headers.Get("Cache-Control") != "no-store" {
		t.Error("QUERY must not use URL-only caching")
	}
	a, b := get.Value, query.Value
	if key != "" {
		a, b = rows(a, key), rows(b, key)
	}
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	if !bytes.Equal(left, right) {
		t.Fatalf("GET/QUERY data mismatch for %s: %.500s != %.500s", path, left, right)
	}
	return query.Value
}

func New(options Options, data *Data) *Suite {
	options.BaseURL = strings.TrimRight(options.BaseURL, "/")
	if options.Game == "" {
		options.Game = "dofus3"
	}
	if len(options.Languages) == 0 {
		options.Languages = []string{"en", "fr", "de", "es", "pt"}
	}
	return &Suite{options: options, data: data, client: &http.Client{Timeout: 90 * time.Second}, covered: map[string]bool{}, requests: map[string]bool{}}
}

func (s *Suite) image(t *testing.T, value any) {
	t.Helper()
	address := Text(At(value, "image_urls", "icon"))
	if address == "" {
		t.Fatal("missing image_urls.icon")
	}
	u, err := url.Parse(address)
	if err != nil {
		t.Fatal(err)
	}
	base, _ := url.Parse(s.options.BaseURL)
	if u.Host != base.Host {
		t.Fatalf("image URL points outside this API: %s", address)
	}
	response, err := s.client.Get(address)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 || !strings.HasPrefix(response.Header.Get("Content-Type"), "image/") {
		t.Fatalf("broken image %s: HTTP %d", address, response.StatusCode)
	}
	s.requests["GET "+u.Path] = true
	if n, err := io.Copy(io.Discard, io.LimitReader(response.Body, 1024*1024)); err != nil || n == 0 {
		t.Fatalf("empty/broken image: %v", err)
	}
}

func (s *Suite) routeCoverage(t *testing.T) {
	for _, route := range s.options.Routes {
		found := false
		for request := range s.requests {
			pieces := strings.SplitN(request, " ", 2)
			if pieces[0] == route.Method && routeMatches(route.Pattern, pieces[1]) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Route has no successful contract case: %s %s (update hook is tested with malformed JSON only)", route.Method, route.Pattern)
		}
	}
}

func routeMatches(pattern, path string) bool {
	want, got := strings.Split(strings.Trim(pattern, "/"), "/"), strings.Split(strings.Trim(path, "/"), "/")
	for i, part := range want {
		if part == "*" {
			return true
		}
		if i >= len(got) {
			return false
		}
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			if got[i] == "" {
				return false
			}
			continue
		}
		if part != got[i] {
			return false
		}
	}
	return len(want) == len(got)
}

func ValidatePublic(value any) error {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			switch key {
			case "resolved", "known", "params", "server_expression", "condition_expression":
				return fmt.Errorf("unfriendly/internal field %q leaked", key)
			}
			if key == "name" && Obj(child) != nil {
				return fmt.Errorf("unlocalized name leaked")
			}
			if err := ValidatePublic(child); err != nil {
				return fmt.Errorf("%s: %w", key, err)
			}
		}
	case []any:
		for _, child := range v {
			if err := ValidatePublic(child); err != nil {
				return err
			}
		}
	}
	return nil
}
