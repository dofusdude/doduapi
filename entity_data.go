package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/dofusdude/doduapi/config"
	"github.com/dofusdude/doduapi/database"
	"github.com/dofusdude/doduapi/utils"
	mapping "github.com/dofusdude/dodumap"
)

type MappedEntityResource struct {
	Route       string
	Table       string
	Filename    string
	ResponseKey string
}

var mappedEntityResources = []MappedEntityResource{
	{Route: "classes", Table: "classes", Filename: "MAPPED_CLASSES.json", ResponseKey: "classes"},
	{Route: "spells", Table: "spells", Filename: "MAPPED_SPELLS.json", ResponseKey: "spells"},
	{Route: "monsters", Table: "monsters", Filename: "MAPPED_MONSTERS.json", ResponseKey: "monsters"},
	{Route: "super-areas", Table: "super_areas", Filename: "MAPPED_SUPER_AREAS.json", ResponseKey: "super_areas"},
	{Route: "areas", Table: "areas", Filename: "MAPPED_AREAS.json", ResponseKey: "areas"},
	{Route: "subareas", Table: "subareas", Filename: "MAPPED_SUBAREAS.json", ResponseKey: "subareas"},
}

type MappedEntityRecord struct {
	QueryProjection  *entityQueryProjection
	AnkamaId         int
	Name             map[string]string
	Document         json.RawMessage
	ImageID          int
	ImageType        string
	ImageResolutions []string
}

type APIEntityListEntry struct {
	AnkamaID  int           `json:"ankama_id"`
	Name      string        `json:"name"`
	ImageUrls *ApiImageUrls `json:"image_urls,omitempty"`
}

type SearchIndexedEntity struct {
	Id        int             `json:"id"`
	Name      string          `json:"name"`
	StuffType SearchStuffType `json:"stuff_type"`
}

const (
	monsterSpellsLinked    = ""
	monsterSpellsFull      = "spells"
	monsterSpellsGradeOnly = "spells.grade"
)

type MappedEntityRenderOptions struct {
	MonsterSpellInclude string
	SpellGrade          int
}

func readMappedFile(filename string) ([]byte, error) {
	if config.DevMode {
		return os.ReadFile(filepath.Join(config.DodudaOutputDir, filename))
	}

	response, err := http.Get(config.ReleaseUrl + "/" + filename)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: %s", filename, response.Status)
	}
	return io.ReadAll(response.Body)
}

func loadMappedEntityRecords(filename string) ([]*MappedEntityRecord, error) {
	body, err := readMappedFile(filename)
	if err != nil {
		return nil, err
	}

	var documents []json.RawMessage
	if err := json.Unmarshal(body, &documents); err != nil {
		return nil, fmt.Errorf("decode %s: %w", filename, err)
	}

	records := make([]*MappedEntityRecord, 0, len(documents))
	for _, document := range documents {
		var header struct {
			AnkamaID int               `json:"ankama_id"`
			Name     map[string]string `json:"name"`
			IconID   int               `json:"icon_id"`
			GfxID    int               `json:"gfx_id"`
		}
		if err := json.Unmarshal(document, &header); err != nil {
			return nil, fmt.Errorf("decode %s entity header: %w", filename, err)
		}
		if header.Name == nil {
			header.Name = map[string]string{}
		}
		record := &MappedEntityRecord{AnkamaId: header.AnkamaID, Name: header.Name, Document: document}
		if filename == "MAPPED_MONSTERS.json" || filename == "MAPPED_SPELLS.json" {
			record.QueryProjection, err = projectEntityQuery(document)
			if err != nil {
				return nil, fmt.Errorf("project %s: %w", filename, err)
			}
		}
		switch filename {
		case "MAPPED_SPELLS.json":
			record.ImageID, record.ImageType, record.ImageResolutions = header.IconID, "spell", []string{"48", "96"}
		case "MAPPED_MONSTERS.json":
			record.ImageID, record.ImageType, record.ImageResolutions = header.GfxID, "monster", []string{"64", "128"}
		case "MAPPED_CLASSES.json":
			record.ImageID, record.ImageType, record.ImageResolutions = header.AnkamaID, "class_head", []string{"32", "64"}
		}
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].AnkamaId < records[j].AnkamaId })
	return records, nil
}

func localizeMappedDocument(document json.RawMessage, language string) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(document))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return localizeMappedValue(value, language), nil
}

func localizeMappedValue(value any, language string) any {
	switch typed := value.(type) {
	case []any:
		result := make([]any, len(typed))
		for index, child := range typed {
			result[index] = localizeMappedValue(child, language)
		}
		return result
	case map[string]any:
		// Combat eligibility uses the same mapped tree and v1 renderer as items.
		if _, node := typed["is_operand"]; node {
			if _, mapped := typed["value"]; mapped {
				body, err := json.Marshal(typed)
				if err == nil {
					var condition mapping.ConditionTreeNodeMapped
					if json.Unmarshal(body, &condition) == nil {
						return RenderConditionTree(&condition, language)
					}
				}
			}
		}
		if isLocalizedStringMap(typed) {
			if translated, ok := typed[language]; ok {
				return translated
			}
			if translated, ok := typed["en"]; ok {
				return translated
			}
			return ""
		}
		result := make(map[string]any, len(typed))
		for key, child := range typed {
			if key == "condition_expression" || key == "server_expression" {
				continue
			}
			result[key] = localizeMappedValue(child, language)
		}
		return result
	default:
		return value
	}
}

func isLocalizedStringMap(value map[string]any) bool {
	if len(value) == 0 {
		return false
	}
	for key, child := range value {
		switch key {
		case "de", "en", "es", "fr", "pt":
			if _, ok := child.(string); !ok {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func renderMappedEntity(record *MappedEntityRecord, language string, full bool) (any, error) {
	return renderMappedEntityWithOptions(record, language, full, MappedEntityRenderOptions{})
}

func renderMappedEntityWithOptions(record *MappedEntityRecord, language string, full bool, options MappedEntityRenderOptions) (any, error) {
	imageUrls := mappedEntityImageUrls(record)
	if !full {
		name := record.Name[language]
		if name == "" {
			name = record.Name["en"]
		}
		return APIEntityListEntry{AnkamaID: record.AnkamaId, Name: name, ImageUrls: imageUrls}, nil
	}
	localized, err := localizeMappedDocument(record.Document, language)
	if err != nil {
		return nil, err
	}
	if imageUrls != nil {
		if document, ok := localized.(map[string]any); ok {
			document["image_urls"] = imageUrls
		}
	}
	if document, ok := localized.(map[string]any); ok {
		addEmbeddedSpellImageUrls(document, record.ImageType)
		if record.ImageType == "spell" && options.SpellGrade > 0 && !filterSpellGrade(document, options.SpellGrade) {
			return nil, fmt.Errorf("spell %d has no grade %d", record.AnkamaId, options.SpellGrade)
		}
		if record.ImageType == "monster" {
			if err := normalizeMonsterGradeSpells(document, language, options.MonsterSpellInclude); err != nil {
				return nil, err
			}
		}
	}
	return localized, nil
}

func addEmbeddedSpellImageUrls(document map[string]any, entityType string) {
	if entityType == "monster" {
		for _, rawGrade := range valueList(document["grades"]) {
			grade, ok := rawGrade.(map[string]any)
			if !ok {
				continue
			}
			for _, rawSpell := range valueList(grade["spells"]) {
				spell, ok := rawSpell.(map[string]any)
				if ok {
					addSpellImageUrls(spell)
				}
			}
		}
		return
	}
	rawSpells, ok := document["spells"].([]any)
	if !ok {
		return
	}
	for _, rawEntry := range rawSpells {
		entry, ok := rawEntry.(map[string]any)
		if !ok {
			continue
		}
		addSpellImageUrls(entry)
	}
}

func addSpellImageUrls(spell map[string]any) {
	iconID, ok := numberAsInt(spell["icon_id"])
	if ok && iconID > 0 {
		spell["image_urls"] = spellImageUrls(iconID)
	}
}

func valueList(value any) []any {
	list, _ := value.([]any)
	return list
}

func filterSpellGrade(spell map[string]any, grade int) bool {
	levels := valueList(spell["levels"])
	filtered := make([]any, 0, 1)
	for _, rawLevel := range levels {
		level, ok := rawLevel.(map[string]any)
		if !ok {
			continue
		}
		mappedGrade, ok := numberAsInt(level["grade"])
		if ok && mappedGrade == grade {
			filtered = append(filtered, level)
		}
	}
	if len(filtered) == 0 {
		return false
	}
	spell["levels"] = filtered
	return true
}

func normalizeMonsterGradeSpells(document map[string]any, language, include string) error {
	for _, rawGrade := range valueList(document["grades"]) {
		grade, ok := rawGrade.(map[string]any)
		if !ok {
			continue
		}
		spells := valueList(grade["spells"])
		normalized := make([]any, 0, len(spells))
		for _, rawSpell := range spells {
			spell, ok := rawSpell.(map[string]any)
			if !ok {
				continue
			}
			spellID, idOK := numberAsInt(spell["ankama_id"])
			levels := valueList(spell["levels"])
			if !idOK || len(levels) != 1 {
				return fmt.Errorf("monster grade spell must have one resolved level")
			}
			level, levelOK := levels[0].(map[string]any)
			spellGrade, gradeOK := numberAsInt(level["grade"])
			if !levelOK || !gradeOK {
				return fmt.Errorf("monster grade spell %d has no resolved grade", spellID)
			}
			switch include {
			case monsterSpellsGradeOnly:
				normalized = append(normalized, spell)
			case monsterSpellsFull:
				fullSpell, err := loadRenderedSpell(spellID, language)
				if err != nil {
					return err
				}
				normalized = append(normalized, fullSpell)
			default:
				normalized = append(normalized, map[string]any{
					"ankama_id":  spellID,
					"name":       spell["name"],
					"grade":      spellGrade,
					"url":        spellGradeURL(language, spellID, spellGrade),
					"image_urls": spell["image_urls"],
				})
			}
		}
		grade["spells"] = normalized
	}
	return nil
}

func loadRenderedSpell(spellID int, language string) (map[string]any, error) {
	txn := database.Db.Txn(false)
	defer txn.Abort()
	raw, err := txn.First(fmt.Sprintf("%s-spells", utils.CurrentRedBlueVersionStr(database.Version.MemDb)), "id", spellID)
	if err != nil || raw == nil {
		return nil, fmt.Errorf("could not resolve spell %d", spellID)
	}
	rendered, err := renderMappedEntity(raw.(*MappedEntityRecord), language, true)
	if err != nil {
		return nil, err
	}
	spell, ok := rendered.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("could not render spell %d", spellID)
	}
	return spell, nil
}

func spellGradeURL(language string, spellID, grade int) string {
	beta := ""
	if config.IsBeta {
		beta = "beta"
	}
	return fmt.Sprintf("%s://%s/dofus3%s/v%d/%s/spells/%d?grade=%d", config.ApiScheme, config.ApiHostName, beta, config.MajorVersion, language, spellID, grade)
}

func numberAsInt(value any) (int, bool) {
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Int64()
		return int(parsed), err == nil
	case float64:
		return int(typed), true
	case int:
		return typed, true
	default:
		return 0, false
	}
}

func mappedEntityImageUrls(record *MappedEntityRecord) *ApiImageUrls {
	if record.ImageID <= 0 || record.ImageType == "" {
		return nil
	}
	if record.ImageType == "spell" {
		return spellImageUrls(record.ImageID)
	}
	if config.DevMode {
		return dodudaImageUrls(record.ImageID, record.ImageType, record.ImageResolutions)
	}
	rendered := RenderImageUrls(utils.ImageUrls(record.ImageID, record.ImageType, record.ImageResolutions, config.ApiScheme, config.MajorVersion, config.ApiHostName, config.IsBeta))
	return &rendered
}

func dodudaImageUrls(imageID int, imageType string, resolutions []string) *ApiImageUrls {
	beta := ""
	if config.IsBeta {
		beta = "beta"
	}
	base := fmt.Sprintf("%s://%s/dofus3%s/v%d/img/%s", config.ApiScheme, config.ApiHostName, beta, config.MajorVersion, imageType)
	urls := make([]string, 0, len(resolutions))
	for index, resolution := range resolutions {
		urls = append(urls, fmt.Sprintf("%s/%dx/%d-%s.png", base, index+1, imageID, resolution))
	}
	rendered := RenderImageUrls(urls)
	return &rendered
}

func spellImageUrls(iconID int) *ApiImageUrls {
	beta := ""
	if config.IsBeta {
		beta = "beta"
	}
	base := fmt.Sprintf("%s://%s/dofus3%s/v%d/img/spell", config.ApiScheme, config.ApiHostName, beta, config.MajorVersion)
	return &ApiImageUrls{
		Icon: fmt.Sprintf("%s/1x/sort_%d-48.png", base, iconID),
		Sd:   fmt.Sprintf("%s/2x/sort_%d-96.png", base, iconID),
	}
}
