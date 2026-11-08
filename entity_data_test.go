package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dofusdude/doduapi/config"
)

func TestLocalizeMappedDocumentRecursively(t *testing.T) {
	document := json.RawMessage(`{
		"ankama_id": 42,
		"name": {"en": "English", "fr": "Français"},
		"effect": {
			"description": {"en": "Effect", "fr": "Effet"},
			"target_mask": {"raw": "a,A", "selectors": [{"kind": "allies"}]}
		}
	}`)

	localized, err := localizeMappedDocument(document, "fr")
	if err != nil {
		t.Fatal(err)
	}
	root := localized.(map[string]any)
	if root["name"] != "Français" {
		t.Fatalf("localized name = %#v", root["name"])
	}
	effect := root["effect"].(map[string]any)
	if effect["description"] != "Effet" {
		t.Fatalf("localized description = %#v", effect["description"])
	}
	mask := effect["target_mask"].(map[string]any)
	if mask["raw"] != "a,A" {
		t.Fatalf("non-localized map was changed: %#v", mask)
	}
}

func TestRenderMonsterPreservesEnrichedCombatAndDropData(t *testing.T) {
	record := &MappedEntityRecord{AnkamaId: 31, ImageType: "monster", Document: json.RawMessage(`{
		"ankama_id":31,"name":{"en":"Blue Larva"},
		"grades":[{"grade":1,"grade_xp":2112,"water_resistance":-9,"spells":[]}],
		"global_drops":[{"item":{"ankama_id":1,"name":{"en":"Drop"}},"min_percent":1,"max_percent":20,"has_conditions":true,"condition":null,"condition_expression":"HS=3383"}],
		"achievements":[{"ankama_id":203,"name":{"en":"Achievement"}}]
	}`)}
	rendered, err := renderMappedEntity(record, "en", true)
	if err != nil {
		t.Fatal(err)
	}
	document := rendered.(map[string]any)
	grade := valueList(document["grades"])[0].(map[string]any)
	if xp, _ := numberAsInt(grade["grade_xp"]); xp != 2112 {
		t.Fatalf("XP lost: %#v", grade)
	}
	if resistance, _ := numberAsInt(grade["water_resistance"]); resistance != -9 {
		t.Fatalf("resistance lost: %#v", grade)
	}
	drop := valueList(document["global_drops"])[0].(map[string]any)
	if drop["has_conditions"] != true || drop["condition_expression"] != nil {
		t.Fatalf("drop restriction lost: %#v", drop)
	}
	if item := drop["item"].(map[string]any); item["name"] != "Drop" {
		t.Fatalf("drop not localized: %#v", item)
	}
}

func TestLocalizeMappedValueFallsBackToEnglish(t *testing.T) {
	value := map[string]any{"en": "Fallback", "fr": "Secours"}
	if got := localizeMappedValue(value, "de"); got != "Fallback" {
		t.Fatalf("fallback = %#v", got)
	}
}

func TestCombatConditionsUseItemV1Renderer(t *testing.T) {
	document := json.RawMessage(`{"condition":{"is_operand":false,"relation":"or","value":null,"children":[{"is_operand":true,"value":{"element":"PO","element_id":-1,"operator":"<","value":42,"templated":{"en":"Inventory item"},"reference":{"ankama_id":42,"name":{"en":"Item"}},"quantity":10}},{"is_operand":true,"value":{"element":"server_requirement","element_id":-1,"operator":"","value":0,"templated":{"en":"Requirement evaluated by the game server"},"server_expression":"DD<30,21172"}}]}}`)
	localized, err := localizeMappedDocument(document, "en")
	if err != nil {
		t.Fatal(err)
	}
	tree := localized.(map[string]any)["condition"].(*ApiConditionNode)
	if tree.Relation == nil || *tree.Relation != "or" || len(tree.Children) != 2 {
		t.Fatal("logic lost")
	}
	item := tree.Children[0].Condition
	if item.Reference == nil || item.Reference.Name != "Item" || item.Quantity == nil || *item.Quantity != 10 || item.Element.Name != "Inventory item" {
		t.Fatalf("v1 reference lost: %#v", item)
	}
	public, err := json.Marshal(localized)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(public), "DD<30") || strings.Contains(string(public), "server_expression") {
		t.Fatalf("raw criterion leaked to API: %s", public)
	}
}

func TestLoadMappedEntityRecordsFromDevOutput(t *testing.T) {
	directory := t.TempDir()
	filename := "MAPPED_TEST.json"
	body := []byte(`[
		{"ankama_id": 9, "name": {"en": "Nine"}},
		{"ankama_id": 2, "name": {"en": "Two"}}
	]`)
	if err := os.WriteFile(filepath.Join(directory, filename), body, 0o600); err != nil {
		t.Fatal(err)
	}

	oldDevMode, oldOutputDir := config.DevMode, config.DodudaOutputDir
	config.DevMode, config.DodudaOutputDir = true, directory
	t.Cleanup(func() {
		config.DevMode, config.DodudaOutputDir = oldDevMode, oldOutputDir
	})

	records, err := loadMappedEntityRecords(filename)
	if err != nil {
		t.Fatal(err)
	}
	ids := []int{records[0].AnkamaId, records[1].AnkamaId}
	if !reflect.DeepEqual(ids, []int{2, 9}) {
		t.Fatalf("records not sorted: %v", ids)
	}
	if records[0].Name["en"] != "Two" {
		t.Fatalf("wrong header: %#v", records[0])
	}
}

func TestMappedSpellImageUrlsMatchDodudaLayout(t *testing.T) {
	oldScheme, oldHost, oldMajor, oldBeta := config.ApiScheme, config.ApiHostName, config.MajorVersion, config.IsBeta
	config.ApiScheme, config.ApiHostName, config.MajorVersion, config.IsBeta = "http", "localhost:3000", 1, false
	t.Cleanup(func() {
		config.ApiScheme, config.ApiHostName, config.MajorVersion, config.IsBeta = oldScheme, oldHost, oldMajor, oldBeta
	})

	urls := mappedEntityImageUrls(&MappedEntityRecord{ImageID: 10075, ImageType: "spell"})
	if urls == nil {
		t.Fatal("expected spell image URLs")
	}
	if !strings.HasSuffix(urls.Icon, "/img/spell/1x/sort_10075-48.png") {
		t.Fatalf("unexpected icon URL: %s", urls.Icon)
	}
	if !strings.HasSuffix(urls.Sd, "/img/spell/2x/sort_10075-96.png") {
		t.Fatalf("unexpected SD URL: %s", urls.Sd)
	}
}

func TestMappedMonsterImageUrlsMatchDevDodudaLayout(t *testing.T) {
	oldScheme, oldHost, oldMajor, oldBeta, oldDevMode := config.ApiScheme, config.ApiHostName, config.MajorVersion, config.IsBeta, config.DevMode
	config.ApiScheme, config.ApiHostName, config.MajorVersion, config.IsBeta, config.DevMode = "http", "localhost:3000", 1, false, true
	t.Cleanup(func() {
		config.ApiScheme, config.ApiHostName, config.MajorVersion, config.IsBeta, config.DevMode = oldScheme, oldHost, oldMajor, oldBeta, oldDevMode
	})

	urls := mappedEntityImageUrls(&MappedEntityRecord{
		ImageID: 1, ImageType: "monster", ImageResolutions: []string{"64", "128"},
	})
	if urls == nil {
		t.Fatal("expected monster image URLs")
	}
	if !strings.HasSuffix(urls.Icon, "/img/monster/1x/1-64.png") {
		t.Fatalf("unexpected icon URL: %s", urls.Icon)
	}
	if !strings.HasSuffix(urls.Sd, "/img/monster/2x/1-128.png") {
		t.Fatalf("unexpected SD URL: %s", urls.Sd)
	}
}

func TestEmbeddedMonsterSpellsReceiveImageUrls(t *testing.T) {
	oldScheme, oldHost, oldMajor, oldBeta := config.ApiScheme, config.ApiHostName, config.MajorVersion, config.IsBeta
	config.ApiScheme, config.ApiHostName, config.MajorVersion, config.IsBeta = "http", "localhost:3000", 1, false
	t.Cleanup(func() {
		config.ApiScheme, config.ApiHostName, config.MajorVersion, config.IsBeta = oldScheme, oldHost, oldMajor, oldBeta
	})

	document := map[string]any{
		"grades": []any{
			map[string]any{"grade": json.Number("1"), "spells": []any{
				map[string]any{"icon_id": json.Number("10075")},
			}},
		},
	}
	addEmbeddedSpellImageUrls(document, "monster")
	spell := document["grades"].([]any)[0].(map[string]any)["spells"].([]any)[0].(map[string]any)
	urls, ok := spell["image_urls"].(*ApiImageUrls)
	if !ok || !strings.HasSuffix(urls.Icon, "/img/spell/1x/sort_10075-48.png") {
		t.Fatalf("embedded spell image URLs missing: %#v", spell)
	}
}

func TestNormalizeMonsterGradeSpellsCreatesGradeSpecificLinks(t *testing.T) {
	oldScheme, oldHost, oldMajor, oldBeta := config.ApiScheme, config.ApiHostName, config.MajorVersion, config.IsBeta
	config.ApiScheme, config.ApiHostName, config.MajorVersion, config.IsBeta = "http", "localhost:3000", 1, false
	t.Cleanup(func() {
		config.ApiScheme, config.ApiHostName, config.MajorVersion, config.IsBeta = oldScheme, oldHost, oldMajor, oldBeta
	})
	document := map[string]any{"grades": []any{map[string]any{
		"grade": json.Number("2"),
		"spells": []any{map[string]any{
			"ankama_id": json.Number("10198"), "name": "Larva Breath",
			"levels": []any{map[string]any{"grade": json.Number("1")}},
		}},
	}}}
	if err := normalizeMonsterGradeSpells(document, "en", monsterSpellsLinked); err != nil {
		t.Fatal(err)
	}
	spell := document["grades"].([]any)[0].(map[string]any)["spells"].([]any)[0].(map[string]any)
	if spell["grade"] != 1 {
		t.Fatalf("resolved spell grade = %#v", spell["grade"])
	}
	if got := spell["url"]; got != "http://localhost:3000/dofus3/v1/en/spells/10198?grade=1" {
		t.Fatalf("unexpected spell URL: %v", got)
	}
}

func TestFilterSpellGrade(t *testing.T) {
	spell := map[string]any{"levels": []any{
		map[string]any{"grade": json.Number("1")},
		map[string]any{"grade": json.Number("2")},
	}}
	if !filterSpellGrade(spell, 2) {
		t.Fatal("grade 2 should exist")
	}
	levels := spell["levels"].([]any)
	if len(levels) != 1 || levels[0].(map[string]any)["grade"] != json.Number("2") {
		t.Fatalf("unexpected filtered levels: %#v", levels)
	}
}
