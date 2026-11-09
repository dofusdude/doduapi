package apitest

import (
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func contains(values []int, want int) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func (s *Suite) filterCase(t *testing.T, c Collection, filter Object, parameters url.Values, predicate func(Record) bool) {
	t.Helper()
	want := IDs(c.Records, predicate)
	status := 200
	if len(want) == 0 && !c.Mapped {
		status = 404
	}
	body := Object{"filter": filter, "page": Object{"size": 100}}
	parameters.Set("page[size]", "100")
	path := s.prefix(s.options.Languages[0]) + "/" + c.Path
	var value any
	if c.Mapped {
		value = s.call(t, "QUERY", path, body, status).Value
		s.call(t, "GET", path+"?filter="+url.QueryEscape(encoded(filter)), nil, 400)
		s.call(t, "GET", path+"?sort=name", nil, 400)
	} else {
		value = s.parity(t, path, c.Key, parameters, body, status)
	}
	if status != 200 {
		return
	}
	ids := ResponseIDs(value, c.Key)
	next := Text(At(value, "_links", "next"))
	pages := 1
	for next != "" {
		pages++
		if pages > len(c.Records)+1 {
			t.Fatal("filtered pagination loop")
		}
		response := s.call(t, "QUERY", next, body, 200)
		ids = append(ids, ResponseIDs(response.Value, c.Key)...)
		next = Text(At(response.Value, "_links", "next"))
	}
	sameIDs(t, sortedIDs(ids), sortedIDs(want))
}

func (s *Suite) filters(t *testing.T) {
	for _, c := range s.data.Collections {
		t.Run(c.Path, func(t *testing.T) {
			if c.Mapped {
				chosen := c.Records[:min(2, len(c.Records))]
				ids := IDs(chosen, nil)
				s.filterCase(t, c, Object{"ids": ids}, url.Values{}, func(r Record) bool { return contains(ids, r.ID) })
				first := chosen[0]
				label := strings.ToUpper(name(first, s.options.Languages[0]))
				s.filterCase(t, c, Object{"and": []any{Object{"name": label}, Object{"or": []any{Object{"ids": ids}, Object{"ids": []int{2147483647}}}}}}, url.Values{}, func(r Record) bool {
					return contains(ids, r.ID) && strings.Contains(strings.ToUpper(name(r, s.options.Languages[0])), label)
				})
				if c.Path == "monsters" {
					if len(first.Levels) == 0 {
						t.Fatal("expected monster grade levels")
					}
					level := first.Levels[0]
					s.filterCase(t, c, Object{"level": Object{"min": level, "max": level}}, url.Values{}, func(r Record) bool { return contains(r.Levels, level) })
					for _, attribute := range []string{"subarea_ids", "spell_ids"} {
						found := -1
						for _, r := range c.Records {
							values := r.Subareas
							if attribute == "spell_ids" {
								values = r.Spells
							}
							if len(values) > 0 {
								found = values[0]
								break
							}
						}
						if found < 0 {
							t.Fatalf("source is missing all monster %s", attribute)
						}
						s.filterCase(t, c, Object{attribute: []int{found}}, url.Values{}, func(r Record) bool {
							values := r.Subareas
							if attribute == "spell_ids" {
								values = r.Spells
							}
							return contains(values, found)
						})
					}
				}
				if c.Path == "spells" {
					s.filterCase(t, c, Object{"ap_cost": Object{"max": 2}, "min_player_level": Object{"min": 100}}, url.Values{}, func(r Record) bool {
						for _, level := range r.SpellLevels {
							if level[0] <= 2 && level[1] >= 100 {
								return true
							}
						}
						return false
					})
				}
				return
			}
			if c.Path == "mounts" {
				family := c.Records[0].TypeID
				familyName := c.Records[0].TypeNames[s.options.Languages[0]]
				s.filterCase(t, c, Object{"family": Object{"id": family}}, url.Values{"filter[family.id]": {strconv.Itoa(family)}}, func(r Record) bool { return r.TypeID == family })
				s.filterCase(t, c, Object{"family": Object{"name": familyName}}, url.Values{"filter[family.name]": {familyName}}, func(r Record) bool { return strings.EqualFold(r.TypeNames[s.options.Languages[0]], familyName) })
				s.searchPredicate(t, c, Object{"family": Object{"id": family}}, url.Values{"filter[family.id]": {strconv.Itoa(family)}}, func(r Record) bool { return r.TypeID == family })
				s.searchPredicate(t, c, Object{"family": Object{"name": familyName}}, url.Values{"filter[family.name]": {familyName}}, func(r Record) bool { return strings.EqualFold(r.TypeNames[s.options.Languages[0]], familyName) })
				return
			}
			low := c.Records[0].Level
			high := low + 20
			minKey, maxKey := "min_level", "max_level"
			if c.Path == "sets" {
				minKey, maxKey = "min_highest_equipment_level", "max_highest_equipment_level"
			}
			s.filterCase(t, c, Object{minKey: low, maxKey: high}, url.Values{"filter[" + minKey + "]": {strconv.Itoa(low)}, "filter[" + maxKey + "]": {strconv.Itoa(high)}}, func(r Record) bool { return r.Level >= low && r.Level <= high })
			s.filterCase(t, c, Object{minKey: low}, url.Values{"filter[" + minKey + "]": {strconv.Itoa(low)}}, func(r Record) bool { return r.Level >= low })
			s.filterCase(t, c, Object{maxKey: high}, url.Values{"filter[" + maxKey + "]": {strconv.Itoa(high)}}, func(r Record) bool { return r.Level <= high })
			if c.Path == "sets" {
				s.searchPredicate(t, c, Object{minKey: low, maxKey: high}, url.Values{"filter[" + minKey + "]": {strconv.Itoa(low)}, "filter[" + maxKey + "]": {strconv.Itoa(high)}}, func(r Record) bool { return r.Level >= low && r.Level <= high })
				for _, key := range []string{"contains_cosmetics", "contains_cosmetics_only"} {
					for _, flag := range []bool{true, false} {
						s.filterCase(t, c, Object{key: flag}, url.Values{"filter[" + key + "]": {strconv.FormatBool(flag)}}, func(r Record) bool {
							if key == "contains_cosmetics" {
								return r.Cosmetics == flag
							}
							return r.CosmeticsOnly == flag
						})
						s.searchPredicate(t, c, Object{key: flag}, url.Values{"filter[" + key + "]": {strconv.FormatBool(flag)}}, func(r Record) bool {
							if key == "contains_cosmetics" {
								return r.Cosmetics == flag
							}
							return r.CosmeticsOnly == flag
						})
					}
				}
			} else {
				kind := c.Records[0].TypeName
				for _, exclude := range []bool{false, true} {
					value := kind
					if exclude {
						value = "-" + kind
					}
					s.filterCase(t, c, Object{"type": Object{"name_id": []string{value}}}, url.Values{"filter[type.name_id]": {value}}, func(r Record) bool { return (r.TypeName == kind) != exclude })
					s.searchPredicate(t, c, Object{"type": Object{"name_id": []string{value}}}, url.Values{"filter[type.name_id]": {value}}, func(r Record) bool { return (r.TypeName == kind) != exclude })
				}
				s.searchPredicate(t, c, Object{minKey: low, maxKey: high}, url.Values{"filter[" + minKey + "]": {strconv.Itoa(low)}, "filter[" + maxKey + "]": {strconv.Itoa(high)}}, func(r Record) bool { return r.Level >= low && r.Level <= high })
			}
			for _, direction := range []string{"level", "-level"} {
				path := s.prefix(s.options.Languages[0]) + "/" + c.Path
				order := "asc"
				if direction[0] == '-' {
					order = "desc"
				}
				response := s.parity(t, path, c.Key, url.Values{"sort[level]": {order}, "page[size]": {"100"}}, Object{"sort": []string{direction}, "page": Object{"size": 100}}, 200)
				previous := -1
				if order == "desc" {
					previous = int(^uint(0) >> 1)
				}
				for _, row := range rows(response, c.Key) {
					key := "level"
					if c.Path == "sets" {
						key = "highest_equipment_level"
					}
					value := Number(Obj(row)[key])
					if order == "asc" && value < previous || order == "desc" && value > previous {
						t.Fatal("level sort order is wrong")
					}
					previous = value
				}
			}
		})
	}
}

func (s *Suite) searchPredicate(t *testing.T, c Collection, filter Object, params url.Values, predicate func(Record) bool) {
	t.Helper()
	anchor := -1
	for i, r := range c.Records {
		if predicate(r) {
			anchor = i
			break
		}
	}
	if anchor < 0 { // A real zero-match case must stay empty, not become an unfiltered success.
		params.Set("query", name(c.Records[0], s.options.Languages[0]))
		s.parity(t, s.prefix(s.options.Languages[0])+"/"+c.Path+"/search", "", params, Object{"query": params.Get("query"), "filter": filter}, 404)
		return
	}
	label := name(c.Records[anchor], s.options.Languages[0])
	params.Set("query", label)
	params.Set("limit", "100")
	value := s.parity(t, s.prefix(s.options.Languages[0])+"/"+c.Path+"/search", "", params, Object{"query": label, "limit": 100, "filter": filter}, 200)
	expected := map[int]bool{}
	for _, r := range c.Records {
		if predicate(r) {
			expected[r.ID] = true
		}
	}
	actual := ResponseIDs(value, "")
	if !contains(actual, c.Records[anchor].ID) {
		t.Fatalf("exact-name search lost expected ID %d (%s)", c.Records[anchor].ID, label)
	}
	for _, id := range actual {
		if !expected[id] {
			t.Fatalf("search filter returned excluded ID %d", id)
		}
	}
}
