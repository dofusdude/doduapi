package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	e "github.com/dofusdude/doduapi/errmsg"
	"github.com/dofusdude/doduapi/query"
	"github.com/go-chi/chi/v5"
)

func readQueryBody(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	w.Header().Set("Cache-Control", "no-store")
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeEntityQueryError(w, 415, "Content-Type must be application/json.")
		return nil, false
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxEntityQueryBytes))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeEntityQueryError(w, 413, "Query body exceeds 64 KiB.")
		} else {
			e.WriteInvalidQueryResponse(w, "Could not read query body.")
		}
		return nil, false
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		e.WriteInvalidQueryResponse(w, "Query body must be a JSON object.")
		return nil, false
	}
	if err := validateEntityQueryJSON(body); err != nil {
		e.WriteInvalidQueryResponse(w, err.Error())
		return nil, false
	}
	return body, true
}

// registerQueryGET is the only registration path for filterable GET routes.
// Both decoders feed one validator, middleware chain, and handler.
func registerQueryGET(r chi.Router, path string, handler http.HandlerFunc, middleware ...func(http.Handler) http.Handler) {
	get := chi.Chain(middleware...).Handler(handler)
	shared := decodeQueryTransport(get)
	registerQueryHandlers(r, path, shared)
}

func registerQueryHandlers(r chi.Router, path string, shared http.Handler) {
	discover := func(handler http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, request *http.Request) {
			w.Header().Set("Accept-Query", "application/json")
			handler.ServeHTTP(w, request)
		}
	}
	r.Get(path, discover(shared))
	r.MethodFunc("QUERY", path, discover(shared))
	r.Options(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Accept-Query", "application/json")
		w.Header().Set("Allow", "GET, QUERY, OPTIONS")
		w.WriteHeader(http.StatusNoContent)
	})
}

func decodeQueryTransport(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpoint, ok := query.Match(r.URL.Path)
		if !ok && r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/all") {
			endpoint, ok = query.Match(strings.TrimSuffix(r.URL.Path, "/all"))
		}
		if !ok {
			e.WriteInvalidQueryResponse(w, "Unsupported QUERY route.")
			return
		}
		if endpoint.Mapped {
			decodeMappedTransport(next, w, r)
			return
		}
		var request *query.Request
		var err error
		if r.Method == "QUERY" {
			body, ok := readQueryBody(w, r)
			if !ok {
				return
			}
			request, err = decodeJSONQuery(endpoint, body, r.URL.Query())
		} else {
			request, err = decodeGETQuery(endpoint, r.URL.Query())
		}
		if err == nil {
			err = validateRequest(endpoint, request, r.Method == "QUERY")
		}
		if err != nil {
			e.WriteInvalidQueryResponse(w, err.Error())
			return
		}
		next.ServeHTTP(w, query.WithRequest(r, request))
	})
}

func decodeJSONQuery(endpoint query.Endpoint, body []byte, overrides url.Values) (*query.Request, error) {
	var document map[string]json.RawMessage
	if err := json.Unmarshal(body, &document); err != nil {
		return nil, err
	}
	if document == nil {
		return nil, fmt.Errorf("QUERY body must be a JSON object")
	}
	request := &query.Request{Values: map[string]query.Value{}}
	values := request.Values
	allowed := map[string]query.Parameter{}
	groups := map[string]bool{}
	for _, parameter := range endpoint.Parameters {
		allowed[parameter.Name] = parameter
		key := parameter.Group
		if key == "" {
			key = parameter.Key
		}
		groups[key] = true
	}
	set := func(name string, raw json.RawMessage) error {
		parameter, ok := allowed[name]
		if !ok {
			return fmt.Errorf("unsupported parameter %s for this endpoint", name)
		}
		var result query.Value
		switch parameter.Type {
		case "integer":
			var value int
			if err := json.Unmarshal(raw, &value); err != nil {
				return fmt.Errorf("%s must be an integer", name)
			}
			result.Number = value
		case "boolean":
			var value bool
			if err := json.Unmarshal(raw, &value); err != nil {
				return fmt.Errorf("%s must be a boolean", name)
			}
			result.Boolean = value
		case "strings":
			var items []string
			if err := json.Unmarshal(raw, &items); err != nil || len(items) == 0 || len(items) > 100 {
				return fmt.Errorf("%s must be an array of 1–100 strings", name)
			}
			for _, item := range items {
				if item == "" || strings.Contains(item, ",") {
					return fmt.Errorf("%s contains an empty or comma-separated array entry", name)
				}
			}
			result.List = items
		default:
			if err := json.Unmarshal(raw, &result.Text); err != nil {
				return fmt.Errorf("%s must be a string", name)
			}
		}
		values[name] = result
		return nil
	}
	// JSON paths are objects, not literal dotted property names. The registry's
	// existing GET names remain the internal lookup keys only.
	var decodeGroup func(string, string, json.RawMessage) error
	decodeGroup = func(group, prefix string, raw json.RawMessage) error {
		var members map[string]json.RawMessage
		if err := json.Unmarshal(raw, &members); err != nil || members == nil {
			return fmt.Errorf("%s must be an object", group+"."+prefix)
		}
		for member, value := range members {
			if strings.Contains(member, ".") {
				return fmt.Errorf("use nested objects instead of dotted key %q", member)
			}
			path := member
			if prefix != "" {
				path = prefix + "." + member
			}
			name := group + "[" + path + "]"
			if _, leaf := allowed[name]; leaf {
				if err := set(name, value); err != nil {
					return err
				}
				continue
			}
			branch := false
			for _, parameter := range endpoint.Parameters {
				if parameter.Group == group && strings.HasPrefix(parameter.Key, path+".") {
					branch = true
					break
				}
			}
			if !branch {
				return fmt.Errorf("unsupported parameter %s", name)
			}
			if err := decodeGroup(group, path, value); err != nil {
				return err
			}
		}
		return nil
	}
	for key, raw := range document {
		switch key {
		case "sort":
			var fields []string
			if err := json.Unmarshal(raw, &fields); err != nil || len(fields) > len(endpoint.Sort) {
				return nil, fmt.Errorf("unsupported sorting for this endpoint")
			}
			seen := map[string]bool{}
			for _, field := range fields {
				name := strings.TrimPrefix(field, "-")
				found := false
				for _, candidate := range endpoint.Sort {
					if candidate == name {
						found = true
					}
				}
				if !found || seen[name] {
					return nil, fmt.Errorf("unsupported or duplicate sort field %s", field)
				}
				seen[name] = true
				direction := "asc"
				if strings.HasPrefix(field, "-") {
					direction = "desc"
				}
				values["sort["+name+"]"] = query.Value{Text: direction}
			}
		case "fields":
			if endpoint.Fields == "" {
				return nil, fmt.Errorf("fields are not supported for this endpoint")
			}
			var fields map[string][]string
			if err := json.Unmarshal(raw, &fields); err != nil {
				return nil, fmt.Errorf("fields must map a resource name to an array of fields")
			}
			for name, items := range fields {
				if name != endpoint.Fields || len(items) > 100 {
					return nil, fmt.Errorf("unsupported fields resource %s", name)
				}
				for _, item := range items {
					if item == "" || strings.Contains(item, ",") {
						return nil, fmt.Errorf("invalid fields entry")
					}
				}
				values["fields["+name+"]"] = query.Value{List: items}
			}
		default:
			if !groups[key] {
				return nil, fmt.Errorf("unsupported QUERY field %s", key)
			}
			if key == "filter" || key == "page" || key == "range" {
				if err := decodeGroup(key, "", raw); err != nil {
					return nil, err
				}
			} else if err := set(key, raw); err != nil {
				return nil, err
			}
		}
	}
	for key, items := range overrides {
		if len(items) != 1 || key != "page[number]" && key != "page[size]" {
			return nil, fmt.Errorf("only pagination URL overrides are supported with QUERY")
		}
		if _, ok := allowed[key]; !ok {
			return nil, fmt.Errorf("pagination is not supported for this endpoint")
		}
		n, err := strconv.Atoi(items[0])
		if err != nil {
			return nil, fmt.Errorf("%s must be an integer", key)
		}
		values[key] = query.Value{Number: n}
	}
	return request, nil
}

func decodeGETQuery(endpoint query.Endpoint, values url.Values) (*query.Request, error) {
	request := &query.Request{Values: map[string]query.Value{}}
	for _, parameter := range endpoint.Parameters {
		raw := values.Get(parameter.Name)
		if raw == "" {
			continue
		}
		value := query.Value{}
		var err error
		switch parameter.Type {
		case "integer":
			value.Number, err = strconv.Atoi(raw)
		case "boolean":
			value.Boolean, err = strconv.ParseBool(strings.ToLower(raw))
		case "strings":
			value.List = strings.Split(raw, ",")
		default:
			value.Text = raw
		}
		if err != nil {
			return nil, fmt.Errorf("invalid %s", parameter.Name)
		}
		request.Values[parameter.Name] = value
	}
	for _, field := range endpoint.Sort {
		if raw := values.Get("sort[" + field + "]"); raw != "" {
			request.Values["sort["+field+"]"] = query.Value{Text: strings.ToLower(raw)}
		}
	}
	key := "fields[" + endpoint.Fields + "]"
	if endpoint.Fields != "" && values.Has(key) {
		fields := []string{}
		if raw := values.Get(key); raw != "" {
			fields = strings.Split(raw, ",")
		}
		request.Values[key] = query.Value{List: fields}
	}
	return request, nil
}

// Transport-independent validation and defaults. QUERY pagination is bounded;
// GET retains the established v1 unlimited-page option.
func validateRequest(endpoint query.Endpoint, request *query.Request, bounded bool) error {
	if strings.HasSuffix(endpoint.Path, "/almanax") {
		for _, key := range []string{"range[from]", "range[to]"} {
			if !request.Has(key) {
				continue
			}
			value := request.Values[key]
			date, err := time.Parse("2006-01-02", value.Text)
			if err != nil {
				return fmt.Errorf("invalid date format")
			}
			value.Date = date
			request.Values[key] = value
		}
		zone := request.String("timezone")
		if zone == "" {
			zone = "Europe/Paris"
		}
		loc, err := time.LoadLocation(zone)
		if err != nil {
			return fmt.Errorf("Invalid timezone.")
		}
		request.Location = loc
		if request.Int("range[size]") > 0 && request.Has("range[from]") && request.Has("range[to]") {
			return fmt.Errorf("Cannot use range[size] with range[from] and range[to].")
		}
	}
	for _, key := range []string{"page[number]", "page[size]", "limit", "level"} {
		if !request.Has(key) {
			continue
		}
		n := request.Int(key)
		switch key {
		case "page[number]":
			if n < 1 || bounded && n > 1000000 {
				return fmt.Errorf("invalid page number")
			}
		case "page[size]":
			if bounded && (n < 1 || n > 100) {
				return fmt.Errorf("invalid page size")
			}
		case "limit":
			// Preserve v1 GET handling; the new JSON transport has strict bounds.
			if n > 100 || bounded && n < 1 {
				return fmt.Errorf("invalid limit value")
			}
		case "level":
			if n < 1 || n > 200 {
				return fmt.Errorf("Level value out of bounds.")
			}
		}
	}
	for _, parameter := range endpoint.Parameters {
		if parameter.Type == "strings" && request.Has(parameter.Name) {
			value := request.Values[parameter.Name]
			for i := range value.List {
				value.List[i] = strings.ToLower(value.List[i])
			}
			request.Values[parameter.Name] = value
		}
	}
	if endpoint.Fields != "" {
		key := "fields[" + endpoint.Fields + "]"
		value := request.Values[key]
		for i, field := range value.List {
			field = strings.ToLower(field)
			valid := false
			for _, allowed := range endpoint.FieldValues {
				if field == allowed {
					valid = true
				}
			}
			if !valid {
				return fmt.Errorf("%s has invalid fields", key)
			}
			value.List[i] = field
		}
		if request.Has(key) {
			request.Values[key] = value
		}
	}
	return nil
}

// Decode through the common strict JSON validator before typed normalization.
func decodeQueryObject(body []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}
