package main

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/dofusdude/doduapi/config"
	e "github.com/dofusdude/doduapi/errmsg"
	"github.com/dofusdude/doduapi/utils"
)

const maxEntityQueryBytes = 64 * 1024

// Reject ambiguous duplicate keys and explicit nulls, which the typed request
// schemas do not allow (encoding/json would otherwise silently ignore them).
func validateEntityQueryJSON(body []byte) error {
	d := json.NewDecoder(bytes.NewReader(body))
	d.UseNumber()
	var value func(int) error
	value = func(depth int) error {
		if depth > 24 {
			return fmt.Errorf("JSON nesting is too deep")
		}
		token, err := d.Token()
		if err != nil {
			return err
		}
		if token == nil {
			return fmt.Errorf("null is not allowed; omit optional fields instead")
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok {
					return fmt.Errorf("expected object key")
				}
				if seen[name] {
					return fmt.Errorf("duplicate JSON key %q", name)
				}
				seen[name] = true
				if err := value(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for d.More() {
				if err := value(depth + 1); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("unexpected JSON delimiter")
		}
		_, err = d.Token()
		return err
	}
	if err := value(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("expected one JSON object")
	}
	return nil
}

type entityQueryContextKey struct{}

type EntityQuery struct {
	Active  bool                `json:"-"`
	Filter  *EntityFilter       `json:"filter,omitempty"`
	Sort    []string            `json:"sort,omitempty"`
	Page    *EntityQueryPage    `json:"page,omitempty"`
	Include []string            `json:"include,omitempty"`
	Fields  map[string][]string `json:"fields,omitempty"`
}

type EntityQueryPage struct {
	Number int `json:"number"`
	Size   int `json:"size"`
}

type EntityRange struct {
	Min *int `json:"min,omitempty"`
	Max *int `json:"max,omitempty"`
}

// Fields in a node are ANDed; values within an ID list are ORed.
type EntityFilter struct {
	And            []EntityFilter `json:"and,omitempty"`
	Or             []EntityFilter `json:"or,omitempty"`
	IDs            []int          `json:"ids,omitempty"`
	Name           *string        `json:"name,omitempty"`
	Level          *EntityRange   `json:"level,omitempty"`
	SubareaIDs     []int          `json:"subarea_ids,omitempty"`
	SpellIDs       []int          `json:"spell_ids,omitempty"`
	APCost         *EntityRange   `json:"ap_cost,omitempty"`
	MinPlayerLevel *EntityRange   `json:"min_player_level,omitempty"`
}

// Small projections are computed once when mapped files are loaded, rather
// than decoding hundreds of MB of embedded spells for every query.
type entityQueryProjection struct {
	Levels      []int
	SubareaIDs  []int
	SpellIDs    []int
	SpellLevels []struct {
		APCost         int `json:"ap_cost"`
		MinPlayerLevel int `json:"min_player_level"`
	}
}

func projectEntityQuery(document json.RawMessage) (*entityQueryProjection, error) {
	var raw struct {
		Grades []struct {
			Level  int `json:"level"`
			Spells []struct {
				ID int `json:"ankama_id"`
			} `json:"spells"`
			StartingSpell *struct {
				ID int `json:"spell_id"`
			} `json:"starting_spell"`
		} `json:"grades"`
		SpawnLocations []struct {
			Subarea struct {
				ID int `json:"ankama_id"`
			} `json:"subarea"`
		} `json:"spawn_locations"`
		Levels []struct {
			APCost         int `json:"ap_cost"`
			MinPlayerLevel int `json:"min_player_level"`
		} `json:"levels"`
	}
	if err := json.Unmarshal(document, &raw); err != nil {
		return nil, err
	}
	p := &entityQueryProjection{SpellLevels: raw.Levels}
	for _, grade := range raw.Grades {
		p.Levels = append(p.Levels, grade.Level)
		for _, spell := range grade.Spells {
			p.SpellIDs = append(p.SpellIDs, spell.ID)
		}
		if grade.StartingSpell != nil && grade.StartingSpell.ID > 0 {
			p.SpellIDs = append(p.SpellIDs, grade.StartingSpell.ID)
		}
	}
	for _, location := range raw.SpawnLocations {
		p.SubareaIDs = append(p.SubareaIDs, location.Subarea.ID)
	}
	return p, nil
}

func (f *EntityFilter) validate(resource string, depth int, nodes *int) error {
	if f == nil {
		return nil
	}
	*nodes = *nodes + 1
	if depth > 8 || *nodes > 64 {
		return fmt.Errorf("filter is limited to 8 levels and 64 nodes")
	}
	if f.And != nil && len(f.And) == 0 || f.Or != nil && len(f.Or) == 0 {
		return fmt.Errorf("and/or must not be empty")
	}
	for _, ids := range [][]int{f.IDs, f.SubareaIDs, f.SpellIDs} {
		if ids != nil && (len(ids) == 0 || len(ids) > 100) {
			return fmt.Errorf("ID lists must contain 1–100 values")
		}
		for _, id := range ids {
			if id < 0 {
				return fmt.Errorf("IDs must be non-negative")
			}
		}
	}
	if f.Name != nil && (strings.TrimSpace(*f.Name) == "" || len(*f.Name) > 200) {
		return fmt.Errorf("name must contain 1–200 bytes of text")
	}
	if resource != "spells" && (f.APCost != nil || f.MinPlayerLevel != nil) {
		return fmt.Errorf("ap_cost and min_player_level are spell filters")
	}
	if resource != "monsters" && (f.Level != nil || f.SubareaIDs != nil || f.SpellIDs != nil) {
		return fmt.Errorf("level, subarea_ids and spell_ids are monster filters")
	}
	for _, bounds := range []*EntityRange{f.Level, f.APCost, f.MinPlayerLevel} {
		if bounds == nil {
			continue
		}
		if bounds.Min == nil && bounds.Max == nil {
			return fmt.Errorf("ranges require min or max")
		}
		if bounds.Min != nil && *bounds.Min < 0 || bounds.Max != nil && *bounds.Max < 0 {
			return fmt.Errorf("range bounds must be non-negative")
		}
		if bounds.Min != nil && bounds.Max != nil && *bounds.Min > *bounds.Max {
			return fmt.Errorf("range min must not exceed max")
		}
	}
	for i := range f.And {
		if err := f.And[i].validate(resource, depth+1, nodes); err != nil {
			return err
		}
	}
	for i := range f.Or {
		if err := f.Or[i].validate(resource, depth+1, nodes); err != nil {
			return err
		}
	}
	return nil
}

func inEntityRange(value int, bounds *EntityRange) bool {
	return bounds == nil || (bounds.Min == nil || value >= *bounds.Min) && (bounds.Max == nil || value <= *bounds.Max)
}

func intersectsEntityIDs(wanted, actual []int) bool {
	if len(wanted) == 0 {
		return true
	}
	for _, a := range actual {
		for _, w := range wanted {
			if a == w {
				return true
			}
		}
	}
	return false
}

func (f *EntityFilter) matches(record *MappedEntityRecord, language string) bool {
	if f == nil {
		return true
	}
	if !intersectsEntityIDs(f.IDs, []int{record.AnkamaId}) {
		return false
	}
	if f.Name != nil && !strings.Contains(strings.ToLower(entityQueryName(record, language)), strings.ToLower(strings.TrimSpace(*f.Name))) {
		return false
	}
	for i := range f.And {
		if !f.And[i].matches(record, language) {
			return false
		}
	}
	if len(f.Or) > 0 {
		matched := false
		for i := range f.Or {
			if f.Or[i].matches(record, language) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	p := record.QueryProjection
	if p == nil {
		p = &entityQueryProjection{}
	}
	if !intersectsEntityIDs(f.SubareaIDs, p.SubareaIDs) || !intersectsEntityIDs(f.SpellIDs, p.SpellIDs) {
		return false
	}
	if f.Level != nil {
		matched := false
		for _, level := range p.Levels {
			if inEntityRange(level, f.Level) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if f.APCost != nil || f.MinPlayerLevel != nil {
		for _, level := range p.SpellLevels {
			if inEntityRange(level.APCost, f.APCost) && inEntityRange(level.MinPlayerLevel, f.MinPlayerLevel) {
				return true
			}
		}
		return false
	}
	return true
}

func entityQueryName(record *MappedEntityRecord, language string) string {
	if name := record.Name[language]; name != "" {
		return name
	}
	return record.Name["en"]
}

func validateEntityQuery(q *EntityQuery, resource string) error {
	return validateEntityRequest(q, resource, true)
}

func validateEntityRequest(q *EntityQuery, resource string, bounded bool) error {
	nodes := 0
	if err := q.Filter.validate(resource, 1, &nodes); err != nil {
		return err
	}
	if len(q.Sort) > 2 {
		return fmt.Errorf("sort accepts at most 2 fields")
	}
	seen := map[string]bool{}
	for _, field := range q.Sort {
		key := strings.TrimPrefix(field, "-")
		if key != "ankama_id" && key != "name" || seen[key] {
			return fmt.Errorf("sort fields must be unique: ankama_id, name (prefix '-' for descending)")
		}
		seen[key] = true
	}
	if len(q.Fields) > 1 {
		return fmt.Errorf("fields only supports this collection")
	}
	for name, fields := range q.Fields {
		if name != strings.ReplaceAll(resource, "-", "_") || len(fields) != 1 || fields[0] != "all" {
			return fmt.Errorf("fields[%s] only accepts ['all']", strings.ReplaceAll(resource, "-", "_"))
		}
	}
	if len(q.Include) > 1 || len(q.Include) == 1 && (resource != "monsters" || q.Include[0] != monsterSpellsFull && q.Include[0] != monsterSpellsGradeOnly) {
		return fmt.Errorf("include accepts spells or spells.grade for monsters only")
	}
	if q.Page == nil {
		q.Page = &EntityQueryPage{Number: 1, Size: 16}
	}
	if q.Page.Number < 1 || bounded && (q.Page.Number > 1000000 || q.Page.Size < 1 || q.Page.Size > 100) {
		return fmt.Errorf("page.number must be 1–1000000 and page.size 1–100")
	}
	return nil
}

func sortEntityQuery(records []*MappedEntityRecord, fields []string, language string) {
	// The database ID iterator is already ordered.
	if len(fields) == 0 {
		return
	}
	sort.Slice(records, func(i, j int) bool {
		for _, field := range fields {
			desc := strings.HasPrefix(field, "-")
			key := strings.TrimPrefix(field, "-")
			comparison := 0
			if key == "name" {
				comparison = strings.Compare(strings.ToLower(entityQueryName(records[i], language)), strings.ToLower(entityQueryName(records[j], language)))
			} else {
				comparison = cmp.Compare(records[i].AnkamaId, records[j].AnkamaId)
			}
			if comparison != 0 {
				if desc {
					return comparison > 0
				}
				return comparison < 0
			}
		}
		return records[i].AnkamaId < records[j].AnkamaId
	})
}

// Both transports decode an EntityQuery and enter this common validation path.
// No URL or body mutation is involved; the list handler sees only typed data.
func decodeMappedTransport(next http.Handler, w http.ResponseWriter, r *http.Request) {
	resource := strings.TrimSuffix(strings.TrimSuffix(r.URL.Path, "/"), "/all")
	resource = resource[strings.LastIndex(resource, "/")+1:]
	table := strings.ReplaceAll(resource, "-", "_")
	q := &EntityQuery{Page: &EntityQueryPage{Number: 1, Size: 16}}
	values := r.URL.Query()
	var err error
	if r.Method == "QUERY" {
		body, ok := readQueryBody(w, r)
		if !ok {
			return
		}
		err = decodeQueryObject(body, q)
		q.Active = true
		for key, items := range values {
			if len(items) != 1 || key != "page[number]" && key != "page[size]" {
				err = fmt.Errorf("QUERY only accepts pagination URL overrides")
			}
		}
	} else {
		q, err = decodeMappedGET(resource, r)
	}
	if q.Page == nil {
		q.Page = &EntityQueryPage{Number: 1, Size: 16}
	}
	for _, key := range []string{"page[number]", "page[size]"} {
		if raw := values.Get(key); raw != "" {
			n, parseErr := strconv.Atoi(raw)
			if parseErr != nil {
				err = fmt.Errorf("invalid page parameter")
			}
			if key == "page[number]" {
				q.Page.Number = n
			} else {
				q.Page.Size = n
			}
		}
	}
	if err == nil {
		// Plain GET preserves its original v1 pagination bounds.
		err = validateEntityRequest(q, resource, q.Active)
	}
	if err != nil {
		e.WriteInvalidQueryResponse(w, err.Error())
		return
	}
	if r.Method == "QUERY" && len(q.Include) > 0 {
		q.Fields = map[string][]string{table: {"all"}}
	}
	next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), entityQueryContextKey{}, q)))
}

func writeEntityQueryError(w http.ResponseWriter, status int, message string) {
	e.WriteErrorResponse(w, status, e.ERR_INVALID_QUERY_VALUE, http.StatusText(status), message)
}

func collectionPaginationLinks(r *http.Request, total int, page utils.Pagination) utils.PaginationLinks {
	u := *r.URL
	if r.Method == "QUERY" {
		u.RawQuery = ""
	} // The unchanged request body retains filters/fields.
	scheme, host := config.ApiScheme, config.ApiHostName
	if scheme == "" {
		scheme = "http"
	}
	if host == "" {
		host = r.Host
	}
	links, _ := page.BuildLinks(u, total, scheme, host)
	return links
}

// GET retains its original URL parameters. Structured filters and sorting
// belong to QUERY, while both transports still share the list handler.
func decodeMappedGET(resource string, r *http.Request) (*EntityQuery, error) {
	values := r.URL.Query()
	q := &EntityQuery{Page: &EntityQueryPage{Number: 1, Size: 16}}
	if values.Has("filter") || values.Has("sort") {
		return q, fmt.Errorf("structured filters and sorting require QUERY with a JSON body")
	}
	if include := strings.TrimSpace(values.Get("include")); include != "" {
		q.Include = []string{include}
	}
	table := strings.ReplaceAll(resource, "-", "_")
	if fields := strings.ToLower(values.Get("fields[" + table + "]")); fields != "" {
		q.Fields = map[string][]string{table: strings.Split(fields, ",")}
	}
	if values.Get("grade") != "" {
		return q, fmt.Errorf("grade is only supported for a single spell")
	}
	return q, nil
}
