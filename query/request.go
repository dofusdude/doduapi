package query

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Value is decoded once, at the transport boundary. Handlers never parse URL
// strings or serialize a QUERY body into GET parameters.
type Value struct {
	Text    string
	Number  int
	Boolean bool
	List    []string
	Date    time.Time
}

type Request struct {
	Values   map[string]Value
	Location *time.Location
}

type requestKey struct{}

func WithRequest(r *http.Request, request *Request) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), requestKey{}, request))
}

func From(r *http.Request) *Request {
	request, _ := r.Context().Value(requestKey{}).(*Request)
	if request == nil {
		panic("query request decoder was not registered")
	}
	return request
}

func (r *Request) Has(key string) bool         { _, ok := r.Values[key]; return ok }
func (r *Request) String(key string) string    { return r.Values[key].Text }
func (r *Request) Int(key string) int          { return r.Values[key].Number }
func (r *Request) Bool(key string) bool        { return r.Values[key].Boolean }
func (r *Request) Strings(key string) []string { return r.Values[key].List }
func (r *Request) Date(key string) time.Time   { return r.Values[key].Date }
func (r *Request) IntDefault(key string, fallback int) int {
	if !r.Has(key) {
		return fallback
	}
	return r.Int(key)
}

// LevelFilter renders typed bounds for the search backend, not for another
// HTTP transport. The same typed bounds are used by in-memory list handlers.
func (r *Request) LevelFilter(min, max, field string) string {
	parts := []string{}
	if r.Has(min) {
		parts = append(parts, fmt.Sprintf("%s>=%d", field, r.Int(min)))
	}
	if r.Has(max) {
		parts = append(parts, fmt.Sprintf("%s<=%d", field, r.Int(max)))
	}
	return strings.Join(parts, " AND ")
}
