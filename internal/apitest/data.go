// Package apitest runs the same black-box contract against deterministic CI
// fixtures or a running API and its matching doduda output. No production
// filtering/rendering functions are used to calculate expected results.
package apitest

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Object = map[string]any
type Record struct {
	ID                       int
	Names                    map[string]string
	Level, TypeID            int
	TypeName                 string
	TypeNames                map[string]string
	Cosmetics, CosmeticsOnly bool
	Levels, Subareas, Spells []int
	SpellLevels              [][2]int
	Document                 Object // Only the first three anchors retain complete source data.
}
type Collection struct {
	Path, Key string
	Mapped    bool
	Records   []Record
}
type Data struct {
	Collections []Collection
	Almanax     []Object
	Recipes     []Object
	Calendar    map[string]Object
}

func Number(v any) int  { n, _ := v.(float64); return int(n) }
func Obj(v any) Object  { m, _ := v.(map[string]any); return m }
func List(v any) []any  { a, _ := v.([]any); return a }
func Text(v any) string { s, _ := v.(string); return s }
func At(v any, path ...string) any {
	for _, key := range path {
		v = Obj(v)[key]
	}
	return v
}
func Localized(v any, lang string) string {
	m := Obj(v)
	if s := Text(m[lang]); s != "" {
		return s
	}
	return Text(m["en"])
}

func readArray(filename string, visit func(Object) error) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	d := json.NewDecoder(f)
	token, err := d.Token()
	if err != nil || token != json.Delim('[') {
		return fmt.Errorf("%s: expected JSON array", filename)
	}
	for d.More() {
		var row Object
		if err := d.Decode(&row); err != nil {
			return fmt.Errorf("%s: %w", filename, err)
		}
		if row == nil {
			return fmt.Errorf("%s: null record", filename)
		}
		if err := visit(row); err != nil {
			return fmt.Errorf("%s: %w", filename, err)
		}
	}
	if _, err := d.Token(); err != nil {
		return err
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return fmt.Errorf("%s: trailing JSON", filename)
	}
	return nil
}

func Load(directory string) (*Data, error) {
	data := &Data{Calendar: map[string]Object{}}
	for _, entry := range []struct{ path, key string }{{"items/equipment", "items"}, {"items/consumables", "items"}, {"items/resources", "items"}, {"items/quest", "items"}, {"items/cosmetics", "items"}, {"mounts", "mounts"}, {"sets", "sets"}, {"classes", "classes"}, {"spells", "spells"}, {"monsters", "monsters"}, {"super-areas", "super_areas"}, {"areas", "areas"}, {"subareas", "subareas"}} {
		data.Collections = append(data.Collections, Collection{Path: entry.path, Key: entry.key, Mapped: entry.key != "items" && entry.path != "mounts" && entry.path != "sets"})
	}
	appendRecord := func(index int, row Object) error {
		r := Record{ID: Number(row["ankama_id"]), Names: map[string]string{}, Level: Number(row["level"])}
		for lang, name := range Obj(row["name"]) {
			r.Names[lang] = Text(name)
		}
		r.TypeID = Number(At(row, "type", "itemTypeId"))
		r.TypeName = strings.ToLower(strings.ReplaceAll(Localized(At(row, "type", "name"), "en"), " ", "-"))
		r.TypeNames = map[string]string{}
		for lang, value := range Obj(At(row, "type", "name")) {
			r.TypeNames[lang] = Text(value)
		}
		r.Cosmetics, _ = row["contains_cosmetics"].(bool)
		r.CosmeticsOnly, _ = row["contains_cosmetics_only"].(bool)
		for _, value := range List(row["grades"]) {
			grade := Obj(value)
			r.Levels = append(r.Levels, Number(grade["level"]))
			for _, spell := range List(grade["spells"]) {
				r.Spells = append(r.Spells, Number(Obj(spell)["ankama_id"]))
			}
			if spell := Obj(grade["starting_spell"]); spell["spell_id"] != nil {
				r.Spells = append(r.Spells, Number(spell["spell_id"]))
			}
		}
		for _, value := range List(row["spawn_locations"]) {
			r.Subareas = append(r.Subareas, Number(At(value, "subarea", "ankama_id")))
		}
		for _, value := range List(row["levels"]) {
			r.SpellLevels = append(r.SpellLevels, [2]int{Number(Obj(value)["ap_cost"]), Number(Obj(value)["min_player_level"])})
		}
		if len(data.Collections[index].Records) < 3 {
			r.Document = row
		}
		data.Collections[index].Records = append(data.Collections[index].Records, r)
		return nil
	}
	categories := map[int]int{0: 0, 1: 1, 2: 2, 3: 3, 5: 4}
	if err := readArray(filepath.Join(directory, "MAPPED_ITEMS.json"), func(row Object) error {
		category := Number(At(row, "type", "categoryId"))
		if index, ok := categories[category]; ok {
			if err := appendRecord(index, row); err != nil {
				return err
			}
		}
		kind := Number(At(row, "type", "itemTypeId"))
		if kind == 242 || kind == 245 || kind == 247 {
			return appendRecord(5, row)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	files := []string{"MAPPED_SETS.json", "MAPPED_CLASSES.json", "MAPPED_SPELLS.json", "MAPPED_MONSTERS.json", "MAPPED_SUPER_AREAS.json", "MAPPED_AREAS.json", "MAPPED_SUBAREAS.json"}
	for i, file := range files {
		index := i + 6
		if err := readArray(filepath.Join(directory, file), func(row Object) error { return appendRecord(index, row) }); err != nil {
			return nil, err
		}
	}
	if err := readArray(filepath.Join(directory, "MAPPED_RECIPES.json"), func(row Object) error { data.Recipes = append(data.Recipes, row); return nil }); err != nil {
		return nil, err
	}
	if err := readArray(filepath.Join(directory, "MAPPED_ALMANAX.json"), func(row Object) error { data.Almanax = append(data.Almanax, row); return nil }); err != nil {
		return nil, err
	}
	for i := range data.Collections {
		collection := &data.Collections[i]
		if len(collection.Records) == 0 {
			return nil, fmt.Errorf("expected nonempty %s source data", collection.Path)
		}
		seen := map[int]bool{}
		for _, r := range collection.Records {
			if seen[r.ID] {
				return nil, fmt.Errorf("duplicate ID %d in %s", r.ID, collection.Path)
			}
			seen[r.ID] = true
		}
		sort.Slice(collection.Records, func(i, j int) bool { return collection.Records[i].ID < collection.Records[j].ID })
	}
	if len(data.Almanax) == 0 {
		return nil, fmt.Errorf("expected nonempty Almanax source data")
	}
	for _, row := range data.Almanax {
		for _, raw := range List(row["days"]) {
			date := Text(raw)
			if _, err := time.Parse("2006-01-02", date); err != nil {
				return nil, fmt.Errorf("Almanax days must be resolved ISO dates, got %q", date)
			}
			if _, exists := data.Calendar[date]; exists {
				return nil, fmt.Errorf("duplicate mapped Almanax date %s", date)
			}
			data.Calendar[date] = row
		}
	}
	if len(data.Calendar) == 0 {
		return nil, fmt.Errorf("mapped Almanax has no concrete calendar dates")
	}
	return data, nil
}

func (data *Data) CalendarDates() []string {
	dates := make([]string, 0, len(data.Calendar))
	for date := range data.Calendar {
		dates = append(dates, date)
	}
	sort.Strings(dates)
	return dates
}
