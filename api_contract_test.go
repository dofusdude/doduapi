package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/dofusdude/doduapi/almanax"
	"github.com/dofusdude/doduapi/config"
	"github.com/dofusdude/doduapi/database"
	"github.com/dofusdude/doduapi/internal/apitest"
	"github.com/dofusdude/doduapi/utils"
	mapping "github.com/dofusdude/dodumap"
	"github.com/go-chi/chi/v5"
)

// Opt-in because this test needs real search/storage. The CI contract job
// always enables it. External mode is read-only and never reindexes anything.
func TestAPIContract(t *testing.T) {
	if os.Getenv("DODUAPI_CONTRACT") != "1" {
		t.Skip("Run scripts/test-contract.sh, or set DODUAPI_CONTRACT=1 and API_TEST_BASE_URL/DODUDA_OUTPUT_DIR")
	}
	options := apitest.Options{BaseURL: os.Getenv("API_TEST_BASE_URL"), Game: os.Getenv("API_TEST_GAME")}
	if options.Game == "" {
		options.Game = "dofus3"
	}
	if options.Game != "dofus3" && options.Game != "dofus3beta" {
		t.Fatal("API_TEST_GAME must be dofus3 or dofus3beta")
	}
	directory := os.Getenv("DODUDA_OUTPUT_DIR")
	if options.BaseURL == "" {
		directory, options.BaseURL = contractFixtureServer(t, options.Game)
		options.Images = true
		options.UpdateToken = "contract-test"
		if err := chi.Walk(Router(), func(method, route string, handler http.Handler, middleware ...func(http.Handler) http.Handler) error {
			options.Routes = append(options.Routes, apitest.Route{Method: method, Pattern: route})
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	} else {
		if directory == "" {
			t.Fatal("External checks require DODUDA_OUTPUT_DIR with the exact mapped data served by the API")
		}
		options.Images = os.Getenv("API_TEST_IMAGES") == "true"
	}
	if languages := os.Getenv("API_TEST_LANGUAGES"); languages != "" {
		options.Languages = strings.Split(languages, ",")
	}
	data, err := apitest.Load(directory)
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("API_TEST_BASE_URL") != "" {
		baselineFile := os.Getenv("API_TEST_BASELINE")
		if baselineFile == "" {
			baselineFile = "testdata/release-baseline.json"
		}
		body, err := os.ReadFile(baselineFile)
		if err != nil {
			t.Fatalf("read reviewed data baseline: %v", err)
		}
		var baseline apitest.Baseline
		if err := json.Unmarshal(body, &baseline); err != nil {
			t.Fatal(err)
		}
		if err := baseline.Check(data); err != nil {
			t.Fatal(err)
		}
	}
	apitest.New(options, data).Run(t)
}

func writeContractJSON(t *testing.T, directory, name string, value any) {
	t.Helper()
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, name), body, 0600); err != nil {
		t.Fatal(err)
	}
}

func fixtureText(text string) map[string]string {
	result := map[string]string{}
	for _, lang := range []string{"en", "fr", "de", "es", "pt"} {
		result[lang] = text
		if lang != "en" {
			result[lang] = lang + " " + text
		}
	}
	return result
}

func contractFixtureServer(t *testing.T, game string) (string, string) {
	t.Helper()
	meiliURL := os.Getenv("DODUAPI_TEST_MEILI_URL")
	if meiliURL == "" {
		t.Fatal("Fixture mode requires an isolated, EMPTY DODUAPI_TEST_MEILI_URL; use scripts/test-contract.sh")
	}
	key := os.Getenv("DODUAPI_TEST_MEILI_KEY")
	if key == "" {
		key = "contract-test-key"
	}
	// GenerateDatabase clears indexes. Refuse to run it against any nonempty
	// server, including the developer's normal Meilisearch instance.
	request, _ := http.NewRequest("GET", strings.TrimRight(meiliURL, "/")+"/indexes?limit=1", nil)
	request.Header.Set("Authorization", "Bearer "+key)
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	var indexes struct {
		Total int `json:"total"`
	}
	err = json.NewDecoder(response.Body).Decode(&indexes)
	response.Body.Close()
	if err != nil || response.StatusCode != 200 || indexes.Total != 0 {
		t.Fatalf("Refusing to reset a nonempty/unavailable test search server (status=%d indexes=%d error=%v)", response.StatusCode, indexes.Total, err)
	}

	oldDB, oldIndexes, oldVersion := database.Db, database.Indexes, database.Version
	oldBeta := config.IsBeta
	config.IsBeta = game == "dofus3beta"
	oldDev, oldOutput, oldFiles, oldMount := config.DevMode, config.DodudaOutputDir, config.PublishFileServer, config.DockerMountDataPath
	oldHost, oldScheme, oldMajor, oldToken := config.ApiHostName, config.ApiScheme, config.MajorVersion, config.UpdateHookToken
	oldMeili, oldKey, oldDir := config.MeiliHost, config.MeiliKey, config.DbDir
	oldElements, oldTypes := config.PersistedElements, config.PersistedTypes
	oldAlmMax, oldAlmDefault, oldCurrent := config.AlmanaxMaxLookAhead, config.AlmanaxDefaultLookAhead, config.CurrentVersion
	t.Cleanup(func() {
		config.IsBeta = oldBeta
		database.Db, database.Indexes, database.Version = oldDB, oldIndexes, oldVersion
		config.DevMode, config.DodudaOutputDir, config.PublishFileServer, config.DockerMountDataPath = oldDev, oldOutput, oldFiles, oldMount
		config.ApiHostName, config.ApiScheme, config.MajorVersion, config.UpdateHookToken = oldHost, oldScheme, oldMajor, oldToken
		config.MeiliHost, config.MeiliKey, config.DbDir = oldMeili, oldKey, oldDir
		config.PersistedElements, config.PersistedTypes = oldElements, oldTypes
		config.AlmanaxMaxLookAhead, config.AlmanaxDefaultLookAhead, config.CurrentVersion = oldAlmMax, oldAlmDefault, oldCurrent
	})
	directory := t.TempDir()
	config.DevMode = true
	config.PublishFileServer = true
	config.DodudaOutputDir = directory
	config.DockerMountDataPath = directory
	config.MeiliHost, config.MeiliKey = meiliURL, key
	config.DbDir = t.TempDir()
	config.MajorVersion = 1
	config.ApiScheme = "http"
	config.UpdateHookToken = "contract-test"
	config.AlmanaxMaxLookAhead, config.AlmanaxDefaultLookAhead = 365, 6
	config.CurrentVersion = utils.GameVersion{Version: "contract-fixture", Release: "main"}
	database.Version = database.VersionT{}
	writeContractFixtures(t, directory)
	config.PersistedElements, config.PersistedTypes, err = parseFixturePersistence(directory)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(Router())
	config.ApiHostName = server.Listener.Addr().String()
	database.Db, database.Indexes = IndexApiData(&database.Version)
	database.Version.MemDb = true
	database.Version.Search = true
	seedContractAlmanax(t, directory)
	server.Start()
	t.Cleanup(server.Close)
	return directory, server.URL
}

func parseFixturePersistence(directory string) (utils.PersistentStringKeysMap, utils.PersistentStringKeysMap, error) {
	return utils.LoadPersistedElementsFiles(filepath.Join(directory, "elements.json"), filepath.Join(directory, "types.json"))
}

func writeContractFixtures(t *testing.T, directory string) {
	t.Helper()
	type object = map[string]any
	text := fixtureText
	condition := object{"is_operand": true, "value": object{"kind": "characteristic", "unit": "points", "element": "CS", "element_id": 0, "operator": ">", "value": 10, "templated": text("Strength > 10")}}
	effect := object{"min": 1, "max": 5, "type": text("Strength"), "templated": text("1 to 5 Strength"), "element_id": 0, "active": true}
	zone := object{"shape": object{"code": "C", "name": "circle"}, "size": 2, "min_size": 0, "cell_ids": []int{}, "damage_decrease_step_percent": 10}
	items := []object{}
	for category := 0; category <= 5; category++ {
		if category == 4 {
			continue
		}
		for i, level := range []int{50, 120, 180} {
			id := (category+1)*1000 + i + 1
			kind := 10 + category
			kindName := fmt.Sprintf("Type%d", category)
			if category == 0 {
				kindName = "Sword"
				kind = 1
				if i > 0 {
					kindName = "Hat"
					kind = 16
				}
			}
			row := object{"ankama_id": id, "name": text(fmt.Sprintf("Contract Category%d Item%d", category, i)), "description": text("Contract description"), "level": level, "pods": 10 + i, "iconId": id, "type": object{"itemTypeId": kind, "name": text(kindName), "categoryId": category, "superTypeId": 1}, "effects": []object{effect}, "conditions": condition}
			if category == 0 && i == 0 {
				row["type"].(object)["superTypeId"] = 2
				row["weapon"] = object{"zone": zone, "requires_line_of_sight": true}
				row["apCost"] = 3
				row["range"] = 2
				row["minRange"] = 1
			}
			items = append(items, row)
		}
	}
	for i, kind := range []int{242, 245} {
		id := 1101 + i
		items = append(items, object{"ankama_id": id, "name": text(fmt.Sprintf("Contract Mount%d", i)), "description": text("Mount description"), "level": 1, "pods": 10, "iconId": id, "type": object{"itemTypeId": kind, "name": text(fmt.Sprintf("MountFamily%d", i)), "categoryId": 0, "superTypeId": 1}, "effects": []object{effect}, "conditions": condition})
	}
	items = append(items, object{"ankama_id": 99999, "name": text("Internal Contract Hidden"), "type": object{"categoryId": 4, "itemTypeId": 16, "name": text("Hat")}})
	writeContractJSON(t, directory, "MAPPED_ITEMS.json", items)
	sets := []object{}
	for i, level := range []int{50, 120, 180} {
		sets = append(sets, object{"ankama_id": 7001 + i, "name": text(fmt.Sprintf("Contract Set%d", i)), "items": []int{1001, 1002}, "level": level, "contains_cosmetics": i > 0, "contains_cosmetics_only": i == 2, "effects": object{"2": []object{effect}}})
	}
	writeContractJSON(t, directory, "MAPPED_SETS.json", sets)
	writeContractJSON(t, directory, "MAPPED_RECIPES.json", []object{{"result_id": 1001, "entries": []object{{"item_id": 3001, "quantity": 2}}}})
	spells := []object{}
	for i := 0; i < 3; i++ {
		levels := []object{}
		for grade := 1; grade <= 2; grade++ {
			ap, unlock := 2+i, 1+i*20
			if grade == 2 {
				ap, unlock = 5+i, 100+i*20
			}
			levels = append(levels, object{"ankama_id": 9100 + i*10 + grade, "grade": grade, "ap_cost": ap, "min_range": 1, "max_range": 4, "min_player_level": unlock, "effects": []object{{"min": 1, "max": 5, "type": text("Damage"), "templated": text("1 to 5 Damage"), "element_id": 0, "zone": zone, "target_mask": object{"selectors": []object{{"kind": "enemies"}}}, "triggers": object{"events": []object{{"kind": "cast"}}}}}, "critical_effects": []object{}, "states_criterion": condition})
		}
		spells = append(spells, object{"ankama_id": 9001 + i, "name": text(fmt.Sprintf("Contract Spell%d", i)), "description": text("Spell description"), "icon_id": 9001 + i, "levels": levels, "classes": []object{}, "monsters": []object{}})
	}
	writeContractJSON(t, directory, "MAPPED_SPELLS.json", spells)
	monsters := []object{}
	classes := []object{}
	for i := 0; i < 3; i++ {
		spell := object{}
		for key, value := range spells[i] {
			spell[key] = value
		}
		spell["levels"] = []object{spells[i]["levels"].([]object)[i%2]}
		grades := []object{}
		for g := 1; g <= 2; g++ {
			grades = append(grades, object{"grade": g, "level": 50 + i*50 + g, "spells": []object{spell}, "life_points": 1000, "action_points": 6, "movement_points": 3})
		}
		location := object{"subarea": object{"ankama_id": 8201 + i, "name": text(fmt.Sprintf("Contract Subarea%d", i))}, "area": object{"ankama_id": 8101 + i, "name": text(fmt.Sprintf("Contract Area%d", i))}, "super_area": object{"ankama_id": 8001 + i, "name": text(fmt.Sprintf("Contract SuperArea%d", i))}, "subarea_maps": []object{{"ankama_id": 8301 + i, "position_x": i, "position_y": i}}, "preferred": true}
		monsters = append(monsters, object{"ankama_id": 9201 + i, "name": text(fmt.Sprintf("Contract Monster%d", i)), "gfx_id": 9201 + i, "grades": grades, "race": object{"ankama_id": 1, "name": text("Contract Race"), "super_race": object{"ankama_id": 2, "name": text("Contract Family")}}, "spawn_locations": []object{location}, "drops": []object{{"item": object{"ankama_id": 3001, "name": text("Contract Category2 Item0")}, "rates_percent": []object{{"grade": 1, "percent": 25}, {"grade": 2, "percent": 30}}, "condition": condition, "has_conditions": true}}, "global_drops": []object{}})
		look := object{"bones_id": 1, "skin_ids": []int{2}, "scale_percent": object{"x": 100, "y": 100}, "default_colors": []int{0x123456}}
		classes = append(classes, object{"ankama_id": 9301 + i, "name": text(fmt.Sprintf("Contract Class%d", i)), "description": text("Class description"), "gameplay_description": text("Class gameplay"), "complexity": 2, "characteristic_costs": object{"strength": []object{{"from": 0, "cost": 1}}}, "spells": []object{spells[i]}, "roles": []object{{"name": text("Damage"), "value": 3}}, "looks": object{"female": look, "male": look}})
	}
	writeContractJSON(t, directory, "MAPPED_MONSTERS.json", monsters)
	writeContractJSON(t, directory, "MAPPED_CLASSES.json", classes)
	for _, world := range []struct {
		file, label string
		start       int
	}{{"MAPPED_SUPER_AREAS.json", "SuperArea", 8001}, {"MAPPED_AREAS.json", "Area", 8101}, {"MAPPED_SUBAREAS.json", "Subarea", 8201}} {
		records := []object{}
		for i := 0; i < 3; i++ {
			row := object{"ankama_id": world.start + i, "name": text(fmt.Sprintf("Contract %s%d", world.label, i))}
			super := object{"ankama_id": 8001 + i, "name": text(fmt.Sprintf("Contract SuperArea%d", i))}
			area := object{"ankama_id": 8101 + i, "name": text(fmt.Sprintf("Contract Area%d", i))}
			subarea := object{"ankama_id": 8201 + i, "name": text(fmt.Sprintf("Contract Subarea%d", i))}
			switch world.label {
			case "SuperArea":
				row["world_map_id"], row["areas"] = 1, []object{area}
			case "Area":
				row["super_area"], row["subareas"] = super, []object{subarea}
			case "Subarea":
				row["level"], row["area"], row["super_area"] = 50, area, super
				row["maps"] = []object{{"ankama_id": 8301 + i, "position_x": i, "position_y": i, "world_map_id": 1}}
				row["monsters"] = []object{{"ankama_id": 9201 + i, "name": text(fmt.Sprintf("Contract Monster%d", i))}}
			}
			records = append(records, row)
		}
		writeContractJSON(t, directory, world.file, records)
	}
	writeContractJSON(t, directory, "MAPPED_ALMANAX.json", []object{{"days": []string{"2030-01-01", "2030-01-02"}, "bonusType": text("Contract Bonus"), "bonus": text("Contract offering bonus"), "offering": object{"itemId": 3001, "itemCategoryId": 2, "itemName": text("Contract Category2 Item0"), "quantity": 2}, "rewardKamas": 1000, "experienceRatio": 1, "optimalLevel": 100, "duration": 1}})
	writeContractJSON(t, directory, "elements.json", []string{"Strength"})
	writeContractJSON(t, directory, "types.json", []string{"Sword", "Hat", "Type1", "Type2", "Type3", "Type5", "MountFamily0", "MountFamily1"})
	var pixel bytes.Buffer
	canvas := image.NewRGBA(image.Rect(0, 0, 1, 1))
	canvas.Set(0, 0, color.RGBA{R: 100, G: 200, B: 50, A: 255})
	if err := png.Encode(&pixel, canvas); err != nil {
		t.Fatal(err)
	}
	writeImage := func(relative string) {
		filename := filepath.Join(directory, "img", relative)
		if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, pixel.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, item := range items {
		if item["iconId"] == nil {
			continue
		}
		for _, size := range []int{64, 128} {
			writeImage(fmt.Sprintf("item/%d-%d.png", item["iconId"], size))
		}
	}
	for i := 0; i < 3; i++ {
		for scale := 1; scale <= 2; scale++ {
			writeImage(fmt.Sprintf("spell/%dx/sort_%d-%d.png", scale, 9001+i, 48*scale))
			writeImage(fmt.Sprintf("monster/%dx/%d-%d.png", scale, 9201+i, 64*scale))
			writeImage(fmt.Sprintf("class_head/%dx/%d-%d.png", scale, 9301+i, 32*scale))
		}
	}
}

func seedContractAlmanax(t *testing.T, directory string) {
	t.Helper()
	repository := database.NewDatabaseRepository(context.Background(), config.DbDir)
	defer repository.Deinit()
	files, err := filepath.Glob("migrations/*.up.sql")
	if err != nil || len(files) == 0 {
		t.Fatal("missing production migrations")
	}
	sort.Strings(files)
	for _, file := range files {
		sql, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := repository.Db.Exec(string(sql)); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
	}
	body, err := os.ReadFile(filepath.Join(directory, "MAPPED_ALMANAX.json"))
	if err != nil {
		t.Fatal(err)
	}
	var entries []mapping.MappedMultilangNPCAlmanaxUnity
	if err := json.Unmarshal(body, &entries); err != nil {
		t.Fatal(err)
	}
	for i := range entries {
		for _, date := range entries[i].Days {
			if _, err := repository.CreateOrUpdate(date, &entries[i]); err != nil {
				t.Fatal(err)
			}
		}
	}
	if added := almanax.UpdateAlmanaxBonusIndex(true, repository); added != len(config.Languages) {
		t.Fatalf("Almanax search indexing failed: %d", added)
	}
}
