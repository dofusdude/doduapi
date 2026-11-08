package apitest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/dofusdude/doduapi/query"
)

func (s *Suite) Run(t *testing.T) {
	for _, language := range s.options.Languages {
		t.Run(language, func(t *testing.T) {
			for _, collection := range s.data.Collections {
				t.Run(collection.Path, func(t *testing.T) { s.collection(t, language, collection) })
			}
			s.metadata(t, language)
			for _, endpoint := range query.Endpoints {
				path := strings.ReplaceAll(strings.ReplaceAll(endpoint.Path, "{game}", s.options.Game), "{language}", language)
				t.Run("QUERY"+path, func(t *testing.T) { s.queryEndpoint(t, language, path, endpoint) })
			}
		})
	}
	t.Run("filter-results", s.filters)
	t.Run("field-selection", s.fieldSelection)
	t.Run("recipe-data", s.recipes)
	t.Run("invalid-requests", s.invalid)
	t.Run("almanax-data", s.almanax)
	if s.options.Images {
		t.Run("image-redirect", func(t *testing.T) {
			client := *s.client
			client.CheckRedirect = func(r *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }
			path := "/" + s.options.Game + "/v1/img"
			response, err := client.Get(s.options.BaseURL + path)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != 301 {
				t.Fatalf("image-root redirect: %d", response.StatusCode)
			}
			s.requests["GET "+path] = true
		})
	}
	t.Run("all-router-methods-covered", s.routeCoverage)
	t.Run("query-route-coverage", func(t *testing.T) {
		for _, endpoint := range query.Endpoints {
			if !s.covered[endpoint.Path] {
				t.Errorf("QUERY endpoint never exercised successfully: %s", endpoint.Path)
			}
		}
	})
}

func (s *Suite) collection(t *testing.T, language string, c Collection) {
	base := s.prefix(language) + "/" + c.Path
	response := s.call(t, "GET", base+"?page[size]=100", nil, 200)
	got := rows(response.Value, c.Key)
	if len(got) != min(100, len(c.Records)) {
		t.Fatalf("missing collection data: got %d, expected %d", len(got), min(100, len(c.Records)))
	}
	for i, row := range got {
		object := Obj(row)
		if Number(object["ankama_id"]) != c.Records[i].ID || Text(object["name"]) != name(c.Records[i], language) {
			t.Fatalf("ID/name mismatch at %d: %#v", i, row)
		}
		if err := ValidatePublic(row); err != nil {
			t.Fatal(err)
		}
	}
	if language == s.options.Languages[0] {
		ids := ResponseIDs(response.Value, c.Key)
		seen := map[int]bool{}
		for _, id := range ids {
			seen[id] = true
		}
		next := Text(At(response.Value, "_links", "next"))
		pages := 1
		for next != "" {
			pages++
			if pages > len(c.Records)+1 {
				t.Fatal("pagination loop")
			}
			page := s.call(t, "GET", next, nil, 200)
			for _, id := range ResponseIDs(page.Value, c.Key) {
				if seen[id] {
					t.Fatalf("duplicate ID %d across pages", id)
				}
				seen[id] = true
				ids = append(ids, id)
			}
			next = Text(At(page.Value, "_links", "next"))
		}
		sameIDs(t, ids, IDs(c.Records, nil))
		all := s.call(t, "GET", base+"/all", nil, 200)
		sameIDs(t, ResponseIDs(all.Value, c.Key), IDs(c.Records, nil))
		if err := ValidatePublic(all.Value); err != nil {
			t.Fatal(err)
		}
	}
	anchors := 0
	for _, record := range c.Records {
		if record.Document == nil {
			continue
		}
		anchors++
		detail := s.call(t, "GET", fmt.Sprintf("%s/%d", base, record.ID), nil, 200)
		object := Obj(detail.Value)
		s.detailFields(t, c, record, object, language)
		if Number(object["ankama_id"]) != record.ID || Text(object["name"]) != name(record, language) {
			t.Fatalf("detail ID/name mismatch: %#v", object)
		}
		if err := ValidatePublic(detail.Value); err != nil {
			t.Fatal(err)
		}
		if c.Key == "items" {
			if Number(object["level"]) != record.Level || Number(At(object, "type", "id")) != record.TypeID {
				t.Fatal("item level/type does not match mapped source")
			}
			if record.Document["pods"] != nil && Number(object["pods"]) != Number(record.Document["pods"]) {
				t.Fatal("item pods missing or incorrect")
			}
		}
		if c.Path == "sets" && Number(object["highest_equipment_level"]) != record.Level {
			t.Fatal("set level does not match source")
		}
		if c.Path == "mounts" && Number(At(object, "family", "ankama_id")) != record.TypeID {
			t.Fatal("mount family does not match source")
		}
		if c.Mapped {
			for key, value := range record.Document {
				if key == "condition_expression" || key == "server_expression" || value == nil {
					continue
				}
				if object[key] == nil {
					t.Errorf("mapped %s field %s disappeared from API", c.Path, key)
				}
			}
			if c.Path == "spells" {
				s.spellGrades(t, base, record, object)
			}
			if c.Path == "monsters" {
				s.monsterSpells(t, base, record, object)
			}
		}
		if s.options.Images && (c.Key == "items" || c.Path == "mounts" || c.Path == "spells" || c.Path == "monsters" || c.Path == "classes") {
			s.image(t, detail.Value)
		}
	}
	if anchors == 0 {
		t.Fatal("no source anchors were retained")
	}
	missing := c.Records[len(c.Records)-1].ID + 100000000
	s.call(t, "GET", fmt.Sprintf("%s/%d", base, missing), nil, 404)
	s.call(t, "GET", base+"/not-an-id", nil, 400)
}

func (s *Suite) spellGrades(t *testing.T, base string, record Record, object Object) {
	t.Helper()
	levels := List(record.Document["levels"])
	if len(levels) == 0 {
		t.Fatal("spell source has no grades")
	}
	if len(List(object["levels"])) != len(levels) {
		t.Fatal("spell grades disappeared")
	}
	grade := Number(Obj(levels[0])["grade"])
	if grade < 1 {
		t.Fatal("spell grade must be positive")
	}
	response := s.call(t, "GET", fmt.Sprintf("%s/%d?grade=%d", base, record.ID, grade), nil, 200)
	actual := List(Obj(response.Value)["levels"])
	if len(actual) != 1 || Number(Obj(actual[0])["grade"]) != grade {
		t.Fatal("grade URL did not select the requested grade")
	}
	for _, level := range List(object["levels"]) {
		for _, effect := range append(append([]any{}, List(Obj(level)["effects"])...), List(Obj(level)["critical_effects"])...) {
			zone := Obj(Obj(effect)["zone"])
			if len(zone) > 0 && Text(At(zone, "shape", "name")) == "" {
				t.Fatal("effect zone has no human-readable shape")
			}
		}
	}
}

func (s *Suite) monsterSpells(t *testing.T, base string, record Record, object Object) {
	t.Helper()
	if len(List(object["grades"])) != len(List(record.Document["grades"])) {
		t.Fatal("monster grades disappeared")
	}
	for index, grade := range List(object["grades"]) {
		source := Obj(List(record.Document["grades"])[index])
		spells := List(Obj(grade)["spells"])
		wanted := List(source["spells"])
		if len(spells) != len(wanted) {
			t.Fatal("monster grade lost spell assignments")
		}
		for i, spell := range spells {
			levels := List(Obj(wanted[i])["levels"])
			if len(levels) != 1 || Number(Obj(spell)["ankama_id"]) != Number(Obj(wanted[i])["ankama_id"]) || Number(Obj(spell)["grade"]) != Number(Obj(levels[0])["grade"]) {
				t.Fatal("monster grade was linked to the wrong spell grade")
			}
		}
	}
	for _, mode := range []string{"spells.grade", "spells"} {
		response := s.call(t, "GET", fmt.Sprintf("%s/%d?include=%s", base, record.ID, mode), nil, 200)
		for _, grade := range List(Obj(response.Value)["grades"]) {
			for _, spell := range List(Obj(grade)["spells"]) {
				levels := List(Obj(spell)["levels"])
				if len(levels) == 0 || mode == "spells.grade" && len(levels) != 1 {
					t.Fatalf("%s did not inline the requested grade", mode)
				}
				if err := ValidatePublic(spell); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	for _, grade := range List(object["grades"]) {
		for _, spell := range List(Obj(grade)["spells"]) {
			address := Text(Obj(spell)["url"])
			if address == "" { // Existing v1 relationship key is uri.
				address = Text(Obj(spell)["uri"])
			}
			if address == "" {
				t.Fatal("linked monster spell has no URL")
			}
			response := s.call(t, "GET", address, nil, 200)
			levels := List(Obj(response.Value)["levels"])
			if len(levels) != 1 || Number(Obj(levels[0])["grade"]) != Number(Obj(spell)["grade"]) {
				t.Fatal("monster spell URL does not resolve its mapped grade")
			}
		}
	}
}

func (s *Suite) metadata(t *testing.T, language string) {
	for _, path := range []string{"version", "elements", "items/types", "search/types"} {
		response := s.call(t, "GET", "/"+s.options.Game+"/v1/meta/"+path, nil, 200)
		if response.Value == nil || response.Value == "" {
			t.Fatalf("empty metadata %s", path)
		}
		if path == "version" && Text(Obj(response.Value)["version"]) == "" {
			t.Fatal("game version missing")
		}
		if a, ok := response.Value.([]any); ok && len(a) == 0 {
			t.Fatalf("empty metadata %s", path)
		}
	}
	response := s.call(t, "GET", "/"+s.options.Game+"/v1/meta/"+language+"/almanax/bonuses", nil, 200)
	if len(List(response.Value)) == 0 {
		t.Fatal("missing Almanax bonuses")
	}
}

func (s *Suite) queryEndpoint(t *testing.T, language, path string, endpoint query.Endpoint) {
	body := Object{}
	parameters := url.Values{}
	key := ""
	for _, c := range s.data.Collections {
		if strings.HasSuffix(path, "/"+c.Path) || strings.HasSuffix(path, "/"+c.Path+"/all") {
			key = c.Key
		}
	}
	if strings.HasSuffix(path, "/search") {
		search := name(s.data.Collections[0].Records[0], language)
		for _, c := range s.data.Collections {
			if strings.HasSuffix(path, "/"+c.Path+"/search") {
				for _, r := range c.Records {
					if candidate := name(r, language); len(candidate) > 2 {
						search = candidate
						break
					}
				}
			}
		}
		if strings.Contains(path, "almanax/bonuses") {
			search = Localized(s.data.Calendar[s.data.CalendarDates()[0]]["bonusType"], language)
		}
		body["query"], body["limit"] = search, 100
		parameters.Set("query", search)
		parameters.Set("limit", "100")
	} else if strings.HasSuffix(path, "/almanax") {
		date := s.data.CalendarDates()[0]
		body["range"] = Object{"from": date, "to": date}
		parameters.Set("range[from]", date)
		parameters.Set("range[to]", date)
	}
	s.parity(t, path, key, parameters, body, 200)
	s.covered[endpoint.Path] = true
	if language == s.options.Languages[0] {
		options := s.call(t, "OPTIONS", path, nil, 204)
		if options.Headers.Get("Accept-Query") != "application/json" {
			t.Fatal("query discovery missing")
		}
		s.call(t, "QUERY", path, Object{"unsupported": true}, 400)
		s.call(t, "QUERY", path, nil, 415)
		if strings.HasSuffix(path, "/search") {
			for _, limit := range []int{-1, 0, 101} {
				bad := Object{}
				for key, value := range body {
					bad[key] = value
				}
				bad["limit"] = limit
				s.call(t, "QUERY", path, bad, 400)
			}
		}
		for _, method := range []string{"GET", "QUERY", "OPTIONS"} {
			r, _ := http.NewRequest("OPTIONS", s.options.BaseURL+path, nil)
			r.Header.Set("Origin", "http://localhost:5173")
			r.Header.Set("Access-Control-Request-Method", method)
			r.Header.Set("Access-Control-Request-Headers", "content-type")
			response, err := s.client.Do(r)
			if err != nil {
				t.Fatal(err)
			}
			response.Body.Close()
			if response.StatusCode != 204 || !strings.Contains(response.Header.Get("Access-Control-Allow-Methods"), "QUERY") {
				t.Fatal("browser preflight failed")
			}
		}
	}
}

func (s *Suite) almanax(t *testing.T) {
	dates := s.data.CalendarDates()
	date := dates[0]
	for _, lang := range s.options.Languages {
		response := s.call(t, "GET", s.prefix(lang)+"/almanax/"+date+"?level=200", nil, 200)
		if Text(Obj(response.Value)["date"]) != date || Obj(response.Value)["reward_xp"] == nil || Number(At(response.Value, "tribute", "quantity")) <= 0 || Text(At(response.Value, "bonus", "description")) == "" {
			t.Fatal("incomplete Almanax data")
		}
		bonus := Text(At(response.Value, "bonus", "type", "id"))
		params := url.Values{"range[from]": {date}, "range[to]": {date}, "level": {"200"}, "timezone": {"Europe/Paris"}, "filter[bonus_type]": {bonus}}
		s.parity(t, s.prefix(lang)+"/almanax", "", params, Object{"range": Object{"from": date, "to": date}, "level": 200, "timezone": "Europe/Paris", "filter": Object{"bonus_type": bonus}}, 200)
		if s.options.Images {
			s.image(t, Obj(At(response.Value, "tribute", "item")))
		}
		for start := 0; start < len(dates); start += 30 {
			wanted := dates[start:min(start+30, len(dates))]
			from, to := wanted[0], wanted[len(wanted)-1]
			value := s.parity(t, s.prefix(lang)+"/almanax", "", url.Values{"range[from]": {from}, "range[to]": {to}}, Object{"range": Object{"from": from, "to": to}}, 200)
			actual := List(value)
			if len(actual) != len(wanted) {
				t.Fatalf("Almanax range %s–%s: expected %d days, got %d", from, to, len(wanted), len(actual))
			}
			for i, row := range actual {
				source := s.data.Calendar[wanted[i]]
				if Text(Obj(row)["date"]) != wanted[i] || Number(At(row, "tribute", "item", "ankama_id")) != Number(At(source, "offering", "itemId")) || Number(At(row, "tribute", "quantity")) != Number(At(source, "offering", "quantity")) || Number(Obj(row)["reward_kamas"]) != Number(source["rewardKamas"]) {
					t.Fatalf("Almanax day %s differs from mapped data", wanted[i])
				}
			}
		}
	}
}

func (s *Suite) invalid(t *testing.T) {
	base := s.prefix(s.options.Languages[0])
	for _, suffix := range []string{"/items/equipment?page[number]=bad", "/spells?page[size]=0", "/spells/1?grade=0", "/almanax/not-a-date", "/almanax?level=201", "/items/equipment/search", "/sets/search", "/mounts/search"} {
		s.call(t, "GET", base+suffix, nil, 400)
	}
	s.call(t, "GET", "/"+s.options.Game+"/v1/zz/spells", nil, 400)
	s.call(t, "QUERY", base+"/spells", Object{"page": Object{"size": 101}}, 400)
	s.call(t, "QUERY", base+"/spells", Object{"filter": Object{"subarea_ids": []int{1}}}, 400)
	s.call(t, "QUERY", base+"/items/equipment", Object{"filter": Object{"min_level": "bad"}}, 400)
	s.call(t, "QUERY", base+"/items/equipment", Object{"fields": Object{"item": []string{"not_a_field"}}}, 400)
	for _, raw := range []string{`null`, `{"filter":null}`, `{"filter":{},"filter":{"ids":[1]}}`, `{} {}`, `{`} {
		s.call(t, "QUERY", base+"/spells", RawJSON(raw), 400)
	}
	s.call(t, "QUERY", base+"/spells", Object{"filter": Object{"name": strings.Repeat("x", 64*1024)}}, 413)
	if s.options.UpdateToken != "" { // Malformed JSON only: never invokes the update operation.
		s.call(t, "POST", "/"+s.options.Game+"/v1/update/"+s.options.UpdateToken, nil, 400)
	}
}

func sortedIDs(ids []int) []int { sort.Ints(ids); return ids }
func encoded(v any) string      { b, _ := json.Marshal(v); return string(b) }
