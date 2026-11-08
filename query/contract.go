// Package query defines the read-only query transport contract shared by the
// router, browser discovery, documentation generator, and request decoders.
package query

import "strings"

type Parameter struct {
	Group string `json:"group,omitempty"`
	Key   string `json:"key"`
	Name  string `json:"name"`
	Type  string `json:"type"`
}

type Endpoint struct {
	Path        string      `json:"path"`
	Mapped      bool        `json:"mapped,omitempty"`
	Parameters  []Parameter `json:"parameters,omitempty"`
	Sort        []string    `json:"sort,omitempty"`
	Fields      string      `json:"fields,omitempty"`
	FieldValues []string    `json:"field_values,omitempty"`
}

var (
	MountFields      = []string{"effects"}
	SetFields        = []string{"effects", "equipment_ids"}
	ItemFields       = []string{"effects", "recipe", "description", "conditions", "pods"}
	EquipmentFields  = append(append([]string{}, ItemFields...), "range", "parent_set", "is_weapon", "critical_hit_probability", "critical_hit_bonus", "max_cast_per_turn", "ap_cost", "combat")
	SearchItemFields = []string{"type", "image_urls", "level"}
)

func param(group, key, kind string) Parameter {
	name := key
	if group != "" {
		name = group + "[" + key + "]"
	}
	return Parameter{Group: group, Key: key, Name: name, Type: kind}
}

var Endpoints = buildEndpoints()

func buildEndpoints() []Endpoint {
	page := []Parameter{param("page", "number", "integer"), param("page", "size", "integer")}
	search := []Parameter{param("", "query", "string"), param("", "limit", "integer")}
	itemFilters := []Parameter{param("filter", "min_level", "integer"), param("filter", "max_level", "integer"), param("filter", "type.name_id", "strings")}
	setFilters := []Parameter{param("filter", "min_highest_equipment_level", "integer"), param("filter", "max_highest_equipment_level", "integer"), param("filter", "contains_cosmetics", "boolean"), param("filter", "contains_cosmetics_only", "boolean")}
	mountFilters := []Parameter{param("filter", "family.name", "string"), param("filter", "family.id", "integer")}
	join := func(groups ...[]Parameter) []Parameter {
		result := []Parameter{}
		for _, group := range groups {
			result = append(result, group...)
		}
		return result
	}
	result := []Endpoint{}
	for _, category := range []string{"equipment", "consumables", "resources", "quest", "cosmetics"} {
		base := "/{game}/v1/{language}/items/" + category
		for _, suffix := range []string{"", "/all"} {
			parameters := itemFilters
			if suffix == "" {
				parameters = join(itemFilters, page)
			}
			result = append(result, Endpoint{Path: base + suffix, Parameters: parameters, Sort: []string{"level"}, Fields: "item"})
		}
		result = append(result, Endpoint{Path: base + "/search", Parameters: join(itemFilters, search)})
	}
	for _, collection := range []struct {
		name    string
		filters []Parameter
		field   string
		sort    []string
	}{
		{"mounts", mountFilters, "mount", nil}, {"sets", setFilters, "set", []string{"level"}},
	} {
		base := "/{game}/v1/{language}/" + collection.name
		result = append(result, Endpoint{Path: base, Parameters: join(collection.filters, page), Fields: collection.field, Sort: collection.sort}, Endpoint{Path: base + "/all", Parameters: collection.filters, Fields: collection.field, Sort: collection.sort}, Endpoint{Path: base + "/search", Parameters: join(collection.filters, search)})
	}
	result = append(result,
		Endpoint{Path: "/{game}/v1/{language}/items/search", Parameters: join(itemFilters, search)},
		Endpoint{Path: "/{game}/v1/{language}/search", Parameters: join(search, []Parameter{param("filter", "search_index", "strings"), param("filter", "type.name_id", "strings")}), Fields: "item"},
		Endpoint{Path: "/{game}/v1/{language}/almanax", Parameters: []Parameter{param("range", "from", "string"), param("range", "to", "string"), param("range", "size", "integer"), param("filter", "bonus_type", "string"), param("", "timezone", "string"), param("", "level", "integer")}},
		Endpoint{Path: "/{game}/v1/meta/{language}/almanax/bonuses/search", Parameters: search},
	)
	for _, resource := range []string{"classes", "monsters", "spells", "super-areas", "areas", "subareas"} {
		base := "/{game}/v1/{language}/" + resource
		result = append(result, Endpoint{Path: base, Mapped: true}, Endpoint{Path: base + "/search", Parameters: search})
	}
	for i := range result {
		endpoint := &result[i]
		switch endpoint.Fields {
		case "mount":
			endpoint.FieldValues = MountFields
		case "set":
			endpoint.FieldValues = SetFields
		case "item":
			endpoint.FieldValues = ItemFields
			if endpoint.Path == "/{game}/v1/{language}/search" {
				endpoint.FieldValues = SearchItemFields
			} else if strings.Contains(endpoint.Path, "/equipment") || strings.Contains(endpoint.Path, "/cosmetics") {
				endpoint.FieldValues = EquipmentFields
			}
		}
	}
	return result
}

func Match(path string) (Endpoint, bool) {
	// Deliberately matches only known, filterable routes, not arbitrary suffixes.
	for _, endpoint := range Endpoints {
		if pathMatches(endpoint.Path, path) {
			return endpoint, true
		}
	}
	return Endpoint{}, false
}
