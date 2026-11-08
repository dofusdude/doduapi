package apitest

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Independent, intentional minimum contracts: these fail even if a mapper and
// API accidentally drop the same field together. Source comparisons add all
// other mapped top-level fields and the complete expected ID sets.
var requiredFields = map[string][]string{
	"items":       {"ankama_id", "name", "description", "type", "level", "pods", "image_urls"},
	"mounts":      {"ankama_id", "name", "family", "image_urls"},
	"sets":        {"ankama_id", "name", "highest_equipment_level", "contains_cosmetics", "contains_cosmetics_only", "equipment_ids"},
	"classes":     {"ankama_id", "name", "description", "gameplay_description", "complexity", "looks", "roles", "characteristic_costs", "spells", "image_urls"},
	"spells":      {"ankama_id", "name", "description", "icon_id", "levels"},
	"monsters":    {"ankama_id", "name", "race", "grades", "spawn_locations", "drops", "global_drops"},
	"super_areas": {"ankama_id", "name", "world_map_id", "areas"},
	"areas":       {"ankama_id", "name", "super_area", "subareas"},
	"subareas":    {"ankama_id", "name", "level", "area", "super_area", "maps", "monsters"},
}

func RequiredFields(key string, object Object) error {
	for _, field := range requiredFields[key] {
		if _, ok := object[field]; !ok {
			return fmt.Errorf("required %s field %s is missing", key, field)
		}
	}
	return nil
}

func (s *Suite) detailFields(t *testing.T, c Collection, r Record, object Object, language string) {
	t.Helper()
	if err := RequiredFields(c.Key, object); err != nil {
		t.Fatal(err)
	}
	if c.Key == "items" {
		if Text(object["description"]) != Localized(r.Document["description"], language) {
			t.Fatal("item description differs from mapped source")
		}
		if len(List(r.Document["effects"])) > 0 && len(List(object["effects"])) != len(List(r.Document["effects"])) {
			t.Fatal("item effects disappeared")
		}
		if r.Document["conditions"] != nil && object["conditions"] == nil {
			t.Fatal("mapped item conditions disappeared")
		}
		if r.Document["weapon"] != nil && Text(At(object, "combat", "zone", "shape", "name")) == "" {
			t.Fatal("weapon effect-zone shape disappeared")
		}
	}
	if c.Path == "classes" {
		if Obj(At(object, "looks", "female")) == nil || Obj(At(object, "looks", "male")) == nil {
			t.Fatal("class appearance is not decoded")
		}
		if len(List(object["roles"])) == 0 || len(List(object["spells"])) == 0 {
			t.Fatal("class roles/spell book missing")
		}
		for _, spell := range List(object["spells"]) {
			if len(List(Obj(spell)["levels"])) == 0 {
				t.Fatal("class spell is not inlined")
			}
			if Text(At(spell, "image_urls", "icon")) == "" && Number(Obj(spell)["icon_id"]) > 0 {
				t.Fatal("class spell image link missing")
			}
		}
	}
	if c.Mapped {
		if err := CompareMapped(r.Document, object, language, c.Path); err != nil {
			t.Fatal(err)
		}
	}
	if c.Path == "monsters" && Text(At(object, "race", "name")) == "" {
		t.Fatal("monster race/family name missing")
	}
}

// Compare every source field on the retained anchors, independently of the
// API renderer. Only documented v1 localization, criterion wrapping and
// monster spell-link normalization are permitted transformations.
func CompareMapped(source, public any, language, path string) error {
	if source == nil {
		return nil
	}
	if object, ok := source.(map[string]any); ok {
		translated := false
		for _, lang := range []string{"en", "fr", "de", "es", "pt"} {
			if _, ok := object[lang]; ok {
				translated = true
				break
			}
		}
		if translated {
			if Text(public) != Localized(source, language) {
				return fmt.Errorf("%s: translated text changed or disappeared", path)
			}
			return nil
		}
		actual := Obj(public)
		if actual == nil {
			return fmt.Errorf("%s: expected object", path)
		}
		if operand, ok := object["is_operand"].(bool); ok {
			if actual["is_operand"] != operand {
				return fmt.Errorf("%s: condition structure changed", path)
			}
			if operand {
				value, condition := Obj(object["value"]), Obj(actual["condition"])
				if condition == nil {
					return fmt.Errorf("%s: interpreted condition missing", path)
				}
				for _, field := range []string{"kind", "unit", "operator", "quantity", "string_value", "value_label", "reference", "secondary_reference"} {
					if value[field] != nil {
						if err := CompareMapped(value[field], condition[field], language, path+"."+field); err != nil {
							return err
						}
					}
				}
				if Number(value["value"]) != Number(condition["int_value"]) {
					return fmt.Errorf("%s: condition threshold changed", path)
				}
				return nil
			}
		}
		for key, value := range object {
			if value == nil || key == "condition_expression" || key == "server_expression" {
				continue
			}
			if optionalName, ok := value.(map[string]any); key == "name" && ok && len(optionalName) == 0 {
				continue
			}
			if key == "spells" && strings.HasPrefix(path, "monsters.grades.") {
				continue
			}
			if _, exists := actual[key]; !exists {
				return fmt.Errorf("%s.%s: mapped field disappeared", path, key)
			}
			if err := CompareMapped(value, actual[key], language, path+"."+key); err != nil {
				return err
			}
		}
		return nil
	}
	if list, ok := source.([]any); ok {
		actual := List(public)
		if len(list) != len(actual) {
			return fmt.Errorf("%s: expected %d entries, got %d", path, len(list), len(actual))
		}
		for i, value := range list {
			if err := CompareMapped(value, actual[i], language, fmt.Sprintf("%s.%d", path, i)); err != nil {
				return err
			}
		}
		return nil
	}
	if !reflect.DeepEqual(source, public) {
		return fmt.Errorf("%s: expected %#v, got %#v", path, source, public)
	}
	return nil
}

func (s *Suite) recipes(t *testing.T) {
	checked := 0
	for _, recipe := range s.data.Recipes {
		id := Number(recipe["result_id"])
		path := ""
		for _, c := range s.data.Collections {
			if c.Key != "items" {
				continue
			}
			for _, r := range c.Records {
				if r.ID == id {
					path = c.Path
					break
				}
			}
		}
		if path == "" {
			continue
		}
		value := s.call(t, "GET", fmt.Sprintf("%s/%s/%d", s.prefix(s.options.Languages[0]), path, id), nil, 200).Value
		if len(List(Obj(value)["recipe"])) != len(List(recipe["entries"])) {
			t.Fatalf("recipe ingredients disappeared for %d", id)
		}
		checked++
		if checked == 3 {
			break
		}
	}
	if checked == 0 {
		t.Fatal("no mapped recipe could be verified against a public item")
	}
}
