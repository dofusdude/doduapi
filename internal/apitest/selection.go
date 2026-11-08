package apitest

import (
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/dofusdude/doduapi/query"
)

func (s *Suite) fieldSelection(t *testing.T) {
	lang := s.options.Languages[0]
	for _, c := range s.data.Collections {
		path := s.prefix(lang) + "/" + c.Path
		if c.Mapped {
			id := c.Records[0].ID
			filter := Object{"ids": []int{id}}
			body := Object{"filter": filter, "fields": Object{c.Key: []string{"all"}}}
			parameters := url.Values{"filter": {encoded(filter)}, "fields[" + c.Key + "]": {"all"}}
			response := s.parity(t, path, c.Key, parameters, body, 200)
			if len(rows(response, c.Key)) != 1 {
				t.Fatal("field selection lost ID filter")
			}
			if c.Path == "monsters" {
				for _, include := range []string{"spells", "spells.grade"} {
					parameters.Set("include", include)
					s.parity(t, path, c.Key, parameters, Object{"filter": filter, "include": []string{include}}, 200)
				}
			}
			continue
		}
		endpoint, ok := query.Match(path)
		if !ok {
			t.Fatal("collection missing from query contract")
		}
		for _, field := range endpoint.FieldValues {
			t.Run(c.Path+"/"+field, func(t *testing.T) {
				s.parity(t, path, c.Key, url.Values{"fields[" + endpoint.Fields + "]": {field}, "page[size]": {"100"}}, Object{"fields": Object{endpoint.Fields: []string{field}}, "page": Object{"size": 100}}, 200)
			})
		}
	}
	// Cross-item and cross-resource filters have their own contracts too.
	var items []Record
	for _, c := range s.data.Collections {
		if c.Key == "items" {
			items = append(items, c.Records...)
		}
	}
	anchor := items[0]
	label := name(anchor, lang)
	low, high := anchor.Level, anchor.Level+20
	filter := Object{"min_level": low, "max_level": high, "type": Object{"name_id": []string{anchor.TypeName}}}
	parameters := url.Values{"query": {label}, "limit": {"100"}, "filter[min_level]": {strconv.Itoa(low)}, "filter[max_level]": {strconv.Itoa(high)}, "filter[type.name_id]": {anchor.TypeName}}
	response := s.parity(t, s.prefix(lang)+"/items/search", "", parameters, Object{"query": label, "limit": 100, "filter": filter}, 200)
	allowed := map[int]bool{}
	for _, item := range items {
		if item.Level >= low && item.Level <= high && item.TypeName == anchor.TypeName {
			allowed[item.ID] = true
		}
	}
	for _, id := range ResponseIDs(response, "") {
		if !allowed[id] {
			t.Fatalf("cross-item filter returned excluded ID %d", id)
		}
	}
	parameters = url.Values{"query": {label}, "filter[search_index]": {"items-equipment"}, "filter[type.name_id]": {anchor.TypeName}, "fields[item]": {"level,type,image_urls"}, "limit": {"100"}}
	response = s.parity(t, s.prefix(lang)+"/search", "", parameters, Object{"query": label, "filter": Object{"search_index": []string{"items-equipment"}, "type": Object{"name_id": []string{anchor.TypeName}}}, "fields": Object{"item": []string{"level", "type", "image_urls"}}, "limit": 100}, 200)
	for _, row := range List(response) {
		if Text(At(row, "type", "name_id")) != "items-equipment" {
			t.Fatal("global index filter ignored")
		}
		if Obj(row)["item_fields"] == nil {
			t.Fatal("global search expansion disappeared")
		}
	}
	for _, c := range s.data.Collections {
		index := strings.ReplaceAll(c.Path, "/", "-")
		if c.Path == "items/quest" {
			index = "items-quest_items"
		}
		label := name(c.Records[0], lang)
		response := s.parity(t, s.prefix(lang)+"/search", "", url.Values{"query": {label}, "filter[search_index]": {index}, "limit": {"100"}}, Object{"query": label, "filter": Object{"search_index": []string{index}}, "limit": 100}, 200)
		if !contains(ResponseIDs(response, ""), c.Records[0].ID) {
			t.Fatalf("global index %s lost its exact-name result", index)
		}
		for _, row := range List(response) {
			resultType := index
			// v1 mount search selects mount equipment; results intentionally retain
			// their item category instead of introducing a new response type.
			if index == "mounts" {
				resultType = "items-equipment"
			}
			if Text(At(row, "type", "name_id")) != resultType || !contains(IDs(c.Records, nil), Number(Obj(row)["ankama_id"])) {
				t.Fatalf("global index filter %s ignored", index)
			}
		}
	}
}
