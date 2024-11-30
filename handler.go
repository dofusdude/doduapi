package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	mapping "github.com/dofusdude/dodumap"
	"github.com/hashicorp/go-memdb"
	"github.com/meilisearch/meilisearch-go"
	g "github.com/zyedidia/generic"
	"github.com/zyedidia/generic/set"
)

var (
	// Note: Never change the order of this list. Indices are guaranteed to be backwards compatible.
	searchAllowedIndices = []string{
		"items-consumables",
		"items-cosmetics",
		"items-resources",
		"items-equipment",
		"items-quest_items",
		"mounts",
		"sets",
	}

	searchAllItemAllowedExpandFields = []string{"type", "image_urls", "level"}

	mountAllowedExpandFields     = []string{"effects"}
	setAllowedExpandFields       = Concat(mountAllowedExpandFields, []string{"equipment_ids"})
	itemAllowedExpandFields      = Concat(mountAllowedExpandFields, []string{"recipe", "description", "conditions"})
	equipmentAllowedExpandFields = Concat(itemAllowedExpandFields, []string{"range", "parent_set", "is_weapon", "pods", "critical_hit_probability", "critical_hit_bonus", "max_cast_per_turn", "ap_cost"})
)

func GetRecipeIfExists(itemId int, txn *memdb.Txn) (mapping.MappedMultilangRecipe, bool) {
	var err error
	var raw interface{}
	if raw, err = txn.First(fmt.Sprintf("%s-recipes", CurrentRedBlueVersionStr(Version.MemDb)), "id", itemId); err != nil {
		log.Fatal(err)
	}

	if raw != nil {
		recipe := raw.(*mapping.MappedMultilangRecipe)
		return *recipe, true
	}

	return mapping.MappedMultilangRecipe{}, false
}

func MinMaxLevelInt(minLevel string, maxLevel string, indexFilterName string) (int, int, error) {
	minLevelInt := 0
	maxLevelInt := 0
	if minLevel != "" {
		var err error
		if minLevelInt, err = strconv.Atoi(minLevel); err != nil {
			return 0, 0, fmt.Errorf("filter[%s]", indexFilterName)
		}
	}
	if maxLevel != "" {
		var err error
		if maxLevelInt, err = strconv.Atoi(maxLevel); err != nil {
			return 0, 0, fmt.Errorf("filter[%s]", indexFilterName)
		}
	}
	return minLevelInt, maxLevelInt, nil
}

func MinMaxLevelMeiliFilterFromParams(filterMinLevel string, filterMaxLevel string, indexFilterName string) (string, error) {
	if filterMinLevel == "" && filterMaxLevel == "" {
		return "", nil
	}

	filterMinLevelInt, filterMaxLevelInt, err := MinMaxLevelInt(filterMinLevel, filterMaxLevel, indexFilterName)
	if err != nil {
		return "", err
	}

	filterMinLevelQuery := fmt.Sprintf("%s>=%d", indexFilterName, filterMinLevelInt)
	filterMaxLevelQuery := fmt.Sprintf("%s<=%d", indexFilterName, filterMaxLevelInt)
	filterString := ""
	if filterMinLevel != "" {
		filterString += filterMinLevelQuery
		if filterMaxLevel != "" {
			filterString += " AND " + filterMaxLevelQuery
		}
	} else {
		filterString += filterMaxLevelQuery
	}

	return filterString, nil
}

type UpdateMessage struct {
	Version string `json:"version"`
}

// update
func UpdateHandler(w http.ResponseWriter, r *http.Request) {
	var updateMessage UpdateMessage
	if err := json.NewDecoder(r.Body).Decode(&updateMessage); err != nil {
		writeInvalidJsonResponse(w, err.Error())
		return
	}

	var release string
	if IsBeta {
		release = "beta"
	} else {
		release = "main"
	}

	ReleaseUrl = fmt.Sprintf("https://github.com/dofusdude/dofus3-%s/releases/download/%s", release, updateMessage.Version)

	log.Info("Updating to version", updateMessage.Version)
	err := DownloadImages()
	if err != nil {
		writeServerErrorResponse(w, "Could not download images: "+err.Error())
		return
	}

	newVersion := GameVersion{
		Version:     updateMessage.Version,
		Release:     release,
		UpdateStamp: time.Now(),
	}

	UpdateChan <- newVersion
}

// listings

// all

func createAllQueryParams(fieldType string, allowedExpandFields []string, r *http.Request) {
	var currentQuery = r.URL.Query()
	currentQuery.Add(fmt.Sprintf("fields[%s]", fieldType), strings.Join(allowedExpandFields, ","))
	r.URL.RawQuery = currentQuery.Encode()
}

func ListAllMounts(w http.ResponseWriter, r *http.Request) {
	createAllQueryParams("mount", mountAllowedExpandFields, r)
	ListMounts(w, r)
}

func ListAllSets(w http.ResponseWriter, r *http.Request) {
	createAllQueryParams("set", setAllowedExpandFields, r)
	ListSets(w, r)
}

func ListAllConsumables(w http.ResponseWriter, r *http.Request) {
	createAllQueryParams("item", itemAllowedExpandFields, r)
	ListConsumables(w, r)
}

func ListAllEquipment(w http.ResponseWriter, r *http.Request) {
	createAllQueryParams("item", equipmentAllowedExpandFields, r)
	ListEquipment(w, r)
}

func ListAllResources(w http.ResponseWriter, r *http.Request) {
	createAllQueryParams("item", itemAllowedExpandFields, r)
	ListResources(w, r)
}

func ListAllQuestItems(w http.ResponseWriter, r *http.Request) {
	createAllQueryParams("item", itemAllowedExpandFields, r)
	ListQuestItems(w, r)
}

func ListAllCosmetics(w http.ResponseWriter, r *http.Request) {
	createAllQueryParams("item", itemAllowedExpandFields, r)
	ListCosmetics(w, r)
}

// paginated

func ListMounts(w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	pagination := PageninationWithState(r.Context().Value("pagination").(string))

	filterFamilyName := r.URL.Query().Get("filter[family.name]")
	filterFamilyIdStr := r.URL.Query().Get("filter[family.id]")
	expansionsParam := strings.ToLower(r.URL.Query().Get("fields[mount]"))
	expansions := parseFields(expansionsParam)
	if !validateFields(expansions, mountAllowedExpandFields) {
		writeInvalidQueryResponse(w, "fields[mount] has invalid fields.")
		return
	}

	txn := Db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get(fmt.Sprintf("%s-%s", CurrentRedBlueVersionStr(Version.MemDb), "mounts"), "id")
	if err != nil || it == nil {
		writeNotFoundResponse(w, "No mounts found.")
		return
	}

	requestsTotal.Inc()
	requestsMountsList.Inc()

	var mounts []APIMount
	for obj := it.Next(); obj != nil; obj = it.Next() {
		p := obj.(*mapping.MappedMultilangMount)
		if filterFamilyName != "" {
			if !strings.EqualFold(p.FamilyName[lang], filterFamilyName) {
				continue
			}
		}
		if filterFamilyIdStr != "" {
			filterFamilyId, err := strconv.Atoi(filterFamilyIdStr)
			if err != nil {
				writeInvalidFilterResponse(w, "filter[family.id] is not a number.")
				return
			}
			if p.FamilyId != filterFamilyId {
				continue
			}
		}
		mount := RenderMountListEntry(p, lang)

		if expansions.Has("effects") {
			effects := RenderEffects(&p.Effects, lang)
			if len(effects) != 0 {
				mount.Effects = effects
			}
		}
		mounts = append(mounts, mount)
	}

	total := len(mounts)
	if total == 0 {
		writeNotFoundResponse(w, "No mounts left after filtering.")
		return
	}

	if pagination.ValidatePagination(total) != 0 {
		writeInvalidQueryResponse(w, "Invalid pagination parameters.")
		return
	}

	startIdx, endIdx := pagination.CalculateStartEndIndex(total)
	links, _ := pagination.BuildLinks(*r.URL, total)
	paginatedMounts := mounts[startIdx:endIdx]

	response := APIPageMount{
		Items: paginatedMounts,
		Links: links,
	}

	WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		writeServerErrorResponse(w, "Could not encode JSON: "+err.Error())
		return
	}
}

func ListSets(w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	pagination := PageninationWithState(r.Context().Value("pagination").(string))

	expansionsParam := strings.ToLower(r.URL.Query().Get("fields[set]"))
	expansions := parseFields(expansionsParam)
	if !validateFields(expansions, setAllowedExpandFields) {
		writeInvalidQueryResponse(w, "fields[set] has invalid fields.")
		return
	}
	sortLevel := strings.ToLower(r.URL.Query().Get("sort[level]"))
	filterMinLevel := strings.ToLower(r.URL.Query().Get("filter[min_highest_equipment_level]"))
	filterMaxLevel := strings.ToLower(r.URL.Query().Get("filter[max_highest_equipment_level]"))
	filterContainsCosmeticsStr := strings.ToLower(r.URL.Query().Get("filter[contains_cosmetics]"))
	filterContainsCosmeticsOnlyStr := strings.ToLower(r.URL.Query().Get("filter[contains_cosmetics_only]"))
	filterMinLevelInt, filterMaxLevelInt, err := MinMaxLevelInt(filterMinLevel, filterMaxLevel, "highest_equipment_level")
	if err != nil {
		writeInvalidFilterResponse(w, "filter[min_level] or filter[max_level] has invalid fields: "+err.Error())
		return
	}

	txn := Db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get(fmt.Sprintf("%s-%s", CurrentRedBlueVersionStr(Version.MemDb), "sets"), "id")
	if err != nil || it == nil {
		writeNotFoundResponse(w, "No sets found.")
		return
	}

	requestsTotal.Inc()
	requestsSetsList.Inc()

	var sets []APIListSet
	for obj := it.Next(); obj != nil; obj = it.Next() {
		p := obj.(*mapping.MappedMultilangSetUnity)

		if filterContainsCosmeticsOnlyStr != "" {
			filterIsCosmetic, err := strconv.ParseBool(filterContainsCosmeticsOnlyStr)
			if err != nil {
				writeInvalidFilterResponse(w, "filter[contains_cosmetics_only] is not a boolean.")
				return
			}

			if p.ContainsCosmeticsOnly != filterIsCosmetic {
				continue
			}
		}

		if filterContainsCosmeticsStr != "" {
			filterIsCosmetic, err := strconv.ParseBool(filterContainsCosmeticsStr)
			if err != nil {
				writeInvalidFilterResponse(w, "filter[contains_cosmetics] is not a boolean.")
				return
			}

			if p.ContainsCosmetics != filterIsCosmetic {
				continue
			}
		}

		if filterMinLevel != "" {
			if p.Level < filterMinLevelInt {
				continue
			}
		}

		if filterMaxLevel != "" {
			if p.Level > filterMaxLevelInt {
				continue
			}
		}

		set := RenderSetListEntry(p, lang)

		if expansions.Has("effects") {
			set.Effects = make(map[int][]ApiEffect, 0)
			for itemCombination, effect := range p.Effects {
				set.Effects[itemCombination] = RenderEffects(&effect, lang)
			}
		}

		if expansions.Has("equipment_ids") {
			set.ItemIds = p.ItemIds
		}

		sets = append(sets, set)
	}

	total := len(sets)
	if total == 0 {
		writeNotFoundResponse(w, "No sets left after filtering.")
		return
	}

	if sortLevel != "" {
		if sortLevel == "asc" {
			sort.Slice(sets, func(i, j int) bool {
				return sets[i].Level < sets[j].Level
			})
		}
		if sortLevel == "desc" {
			sort.Slice(sets, func(i, j int) bool {
				return sets[i].Level > sets[j].Level
			})
		}
	}

	if pagination.ValidatePagination(total) != 0 {
		writeInvalidQueryResponse(w, "Invalid pagination parameters.")
		return
	}

	startIdx, endIdx := pagination.CalculateStartEndIndex(total)
	links, _ := pagination.BuildLinks(*r.URL, total)
	paginatedSets := sets[startIdx:endIdx]

	response := APIPageSet{
		Items: paginatedSets,
		Links: links,
	}

	WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		writeServerErrorResponse(w, "Could not encode JSON: "+err.Error())
		return
	}
}

func setFilter(in *set.Set[string], prefix string, exceptions *[]string) (set.Set[string], error) {
	txn := Db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get("item-type-ids", "id")
	if err != nil || it == nil {
		return set.NewHashset(0, g.Equals[string], g.HashString), err
	}

	typeIds := set.NewHashset(10, g.Equals[string], g.HashString)
	if exceptions != nil {
		for _, exception := range *exceptions {
			typeIds.Put(exception)
		}
	}

	for obj := it.Next(); obj != nil; obj = it.Next() {
		effectElement := obj.(*ItemTypeId)
		typeIds.Put(effectElement.EnName)
	}

	out := set.NewHashset(10, g.Equals[string], g.HashString)
	for _, str := range in.Keys() {
		noPrefix := str
		if prefix == "" {
			disallowedPrefixes := []string{"-", "+"}
			hasDisallowedPrefix := false
			for _, disallowedPrefix := range disallowedPrefixes {
				if strings.HasPrefix(str, disallowedPrefix) {
					hasDisallowedPrefix = true
					break
				}
			}
			if !hasDisallowedPrefix {
				if typeIds.Has(noPrefix) {
					out.Put(noPrefix)
				} else {
					return out, fmt.Errorf("unknown type: " + noPrefix)
				}
			}
		} else {
			if strings.HasPrefix(str, prefix) {
				noPrefix = strings.TrimPrefix(str, prefix)
				if typeIds.Has(noPrefix) {
					out.Put(noPrefix)
				} else {
					return out, fmt.Errorf("unknown type: " + noPrefix)
				}
			}
		}
	}

	return out, nil
}

func excludeTypes(all *set.Set[string], exceptions *[]string) (set.Set[string], error) {
	return setFilter(all, "-", exceptions)
}

func includeTypes(all *set.Set[string], exceptions *[]string) (set.Set[string], error) {
	explicitAdd, err := setFilter(all, "+", exceptions)
	if err != nil {
		return set.NewHashset(0, g.Equals[string], g.HashString), err
	}

	implicitAdd, err := setFilter(all, "", exceptions)
	if err != nil {
		return set.NewHashset(0, g.Equals[string], g.HashString), err
	}

	res := set.NewHashset(10, g.Equals[string], g.HashString)
	for _, str := range implicitAdd.Keys() {
		res.Put(str)
	}

	for _, str := range explicitAdd.Keys() {
		res.Put(str)
	}
	return res, nil
}

func parseFields(expansionsParam string) *set.Set[string] {
	expansions := set.NewHashset[string](10, g.Equals[string], g.HashString)
	expansionContainsDiv := strings.Contains(expansionsParam, ",")
	if len(expansionsParam) != 0 && expansionContainsDiv {
		expansionsArr := strings.Split(expansionsParam, ",")
		for _, expansion := range expansionsArr {
			expansions.Put(expansion)
		}
	}
	if len(expansionsParam) != 0 && !expansionContainsDiv {
		expansions.Put(expansionsParam)
	}

	return &expansions
}

func validateFields(expansions *set.Set[string], list []string) bool {
	allowedFields := set.NewHashset(10, g.Equals[string], g.HashString)
	for _, expansion := range list {
		allowedFields.Put(expansion)
	}

	return expansions.Difference(allowedFields).Size() == 0
}

func ListItems(itemType string, w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	pagination := PageninationWithState(r.Context().Value("pagination").(string))

	expansionsParam := strings.ToLower(r.URL.Query().Get("fields[item]"))
	var expansions *set.Set[string]
	if itemType == "equipment" || itemType == "cosmetics" {
		expansions = parseFields(expansionsParam)
		if !validateFields(expansions, equipmentAllowedExpandFields) {
			writeInvalidQueryResponse(w, "fields[item] has invalid fields.")
			return
		}
	} else {
		expansions = parseFields(expansionsParam)
		if !validateFields(expansions, itemAllowedExpandFields) {
			writeInvalidQueryResponse(w, "fields[item] has invalid fields.")
			return
		}
	}

	typeFiltering := strings.ToLower(r.URL.Query().Get("filter[type.name_id]"))
	filterset := parseFields(typeFiltering)
	additiveTypes, err := includeTypes(filterset, nil)
	if err != nil {
		writeInvalidQueryResponse(w, "filter[type.name_id] has invalid fields: "+err.Error())
		return
	}

	removedTypes, err := excludeTypes(filterset, nil)
	if err != nil {
		writeInvalidQueryResponse(w, "filter[type.name_id] has invalid fields: "+err.Error())
		return
	}

	sortLevel := strings.ToLower(r.URL.Query().Get("sort[level]"))
	filterMinLevel := strings.ToLower(r.URL.Query().Get("filter[min_level]"))
	filterMaxLevel := strings.ToLower(r.URL.Query().Get("filter[max_level]"))
	filterMinLevelInt, filterMaxLevelInt, err := MinMaxLevelInt(filterMinLevel, filterMaxLevel, "level")
	if err != nil {
		writeInvalidFilterResponse(w, "filter[min_level] or filter[max_level] has invalid fields: "+err.Error())
		return
	}

	txn := Db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get(fmt.Sprintf("%s-%s", CurrentRedBlueVersionStr(Version.MemDb), itemType), "id")
	if err != nil || it == nil {
		writeNotFoundResponse(w, "No items found.")
		return
	}

	requestsItemsList.Inc()
	requestsTotal.Inc()

	var items []APIListItem
	for obj := it.Next(); obj != nil; obj = it.Next() {
		p := obj.(*mapping.MappedMultilangItemUnity)

		enTypeName := strings.ToLower(p.Type.Name["en"])

		if removedTypes.Has(enTypeName) {
			continue
		}

		if additiveTypes.Size() > 0 && !additiveTypes.Has(enTypeName) {
			continue
		}

		if filterMinLevel != "" {
			if p.Level < filterMinLevelInt {
				continue
			}
		}

		if filterMaxLevel != "" {
			if p.Level > filterMaxLevelInt {
				continue
			}
		}

		item := RenderItemListEntry(p, lang)
		// items extra fields
		if expansions.Has("recipe") {
			recipe, exists := GetRecipeIfExists(item.Id, txn)
			if exists {
				item.Recipe = RenderRecipe(recipe, Db)
			} else {
				item.Recipe = nil
			}
		}

		if expansions.Has("description") {
			description := p.Description[lang]
			item.Description = &description
		}

		if expansions.Has("conditions") {
			if p.Conditions != nil {
				item.Conditions = RenderConditionTree(p.Conditions, lang)
			}
		}

		if expansions.Has("effects") {
			if p.Effects != nil {
				renderedEffects := RenderEffects(&p.Effects, lang)
				if len(renderedEffects) != 0 {
					item.Effects = renderedEffects
				}
			}
		}

		// equipment extra fields
		mIsWeapon := p.Type.SuperTypeId == 2 // is weapon
		if expansions.Has("is_weapon") {
			item.IsWeapon = &mIsWeapon
		}

		if expansions.Has("pods") {
			item.Pods = &p.Pods
		}

		if expansions.Has("parent_set") {
			if p.HasParentSet {
				item.ParentSet = &APISetReverseLink{
					Id:   p.ParentSet.Id,
					Name: p.ParentSet.Name[lang],
				}
			}
		}

		// weapon extra fields
		if mIsWeapon {
			if expansions.Has("critical_hit_probability") {
				item.CriticalHitProbability = &p.CriticalHitProbability
			}

			if expansions.Has("critical_hit_bonus") {
				item.CriticalHitBonus = &p.CriticalHitBonus
			}

			if expansions.Has("max_cast_per_turn") {
				item.MaxCastPerTurn = &p.MaxCastPerTurn
			}

			if expansions.Has("ap_cost") {
				item.ApCost = &p.ApCost
			}

			if expansions.Has("range") {
				item.Range = &APIRange{
					Min: p.MinRange,
					Max: p.Range,
				}
			}
		}

		items = append(items, item)
	}

	if len(items) == 0 {
		writeNotFoundResponse(w, "No items left after filtering.")
		return
	}

	if sortLevel != "" {
		if sortLevel == "asc" {
			sort.Slice(items, func(i, j int) bool {
				return items[i].Level < items[j].Level
			})
		}
		if sortLevel == "desc" {
			sort.Slice(items, func(i, j int) bool {
				return items[i].Level > items[j].Level
			})
		}
	}

	total := len(items)

	if pagination.ValidatePagination(total) != 0 {
		writeInvalidQueryResponse(w, "Invalid pagination parameters.")
		return
	}

	startIdx, endIdx := pagination.CalculateStartEndIndex(total)
	links, _ := pagination.BuildLinks(*r.URL, total)
	paginatedItems := items[startIdx:endIdx]

	response := APIPageItem{
		Items: paginatedItems,
		Links: links,
	}

	WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		writeServerErrorResponse(w, "Could not encode JSON: "+err.Error())
		return
	}
}

func ListConsumables(w http.ResponseWriter, r *http.Request) {
	ListItems("consumables", w, r)
}

func ListEquipment(w http.ResponseWriter, r *http.Request) {
	ListItems("equipment", w, r)
}

func ListResources(w http.ResponseWriter, r *http.Request) {
	ListItems("resources", w, r)
}

func ListQuestItems(w http.ResponseWriter, r *http.Request) {
	ListItems("quest_items", w, r)
}

func ListCosmetics(w http.ResponseWriter, r *http.Request) {
	ListItems("cosmetics", w, r)
}

func getLimitInBoundary(limitStr string) (int64, error) {
	if limitStr == "" {
		limitStr = "8"
	}
	var limit int
	var err error
	if limit, err = strconv.Atoi(limitStr); err != nil {
		return 0, fmt.Errorf("invalid limit value")
	}
	if limit > 100 {
		return 0, fmt.Errorf("limit value is too high")
	}

	return int64(limit), nil
}

// search

func SearchAlmanaxBonuses(w http.ResponseWriter, r *http.Request) {
	client := meilisearch.New(MeiliHost, meilisearch.WithAPIKey(MeiliKey))
	defer client.Close()

	query := r.URL.Query().Get("query")
	if query == "" {
		writeInvalidQueryResponse(w, "Query parameter is required.")
		return
	}

	lang := r.Context().Value("lang").(string)

	if lang == "pt" {
		writeInvalidQueryResponse(w, "Portuguese language is not translated for Almanax Bonuses.")
		return
	}

	var searchLimit int64
	var err error
	if searchLimit, err = getLimitInBoundary(r.URL.Query().Get("limit")); err != nil {
		writeInvalidQueryResponse(w, "Invalid limit value: "+err.Error())
		return
	}

	index := client.Index(fmt.Sprintf("alm-bonuses-%s", lang))

	request := &meilisearch.SearchRequest{
		Limit: searchLimit,
	}

	var searchResp *meilisearch.SearchResponse
	if searchResp, err = index.Search(query, request); err != nil {
		writeServerErrorResponse(w, "Could not search: "+err.Error())
		return
	}

	requestsTotal.Inc()
	requestsSearchTotal.Inc()

	if searchResp.EstimatedTotalHits == 0 {
		writeNotFoundResponse(w, "No results found.")
		return
	}

	var results []AlmanaxBonusListing
	for _, hit := range searchResp.Hits {
		almBonusJson := hit.(map[string]interface{})
		almBonus := AlmanaxBonusListing{
			Id:   almBonusJson["slug"].(string),
			Name: almBonusJson["name"].(string),
		}
		results = append(results, almBonus)
	}

	WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(results)
	if err != nil {
		writeServerErrorResponse(w, "Could not encode JSON: "+err.Error())
		return
	}
}

func SearchMounts(w http.ResponseWriter, r *http.Request) {
	client := meilisearch.New(MeiliHost, meilisearch.WithAPIKey(MeiliKey))
	defer client.Close()

	var err error
	query := r.URL.Query().Get("query")
	if query == "" {
		writeInvalidQueryResponse(w, "Query parameter is required.")
		return
	}

	var searchLimit int64
	if searchLimit, err = getLimitInBoundary(r.URL.Query().Get("limit")); err != nil {
		writeInvalidQueryResponse(w, "Invalid limit value: "+err.Error())
		return
	}

	filterFamilyName := r.URL.Query().Get("filter[family.name]")
	filterFamilyIdStr := r.URL.Query().Get("filter[family.id]")

	lang := r.Context().Value("lang").(string)

	index := client.Index(fmt.Sprintf("%s-mounts-%s", CurrentRedBlueVersionStr(Version.Search), lang))
	var request *meilisearch.SearchRequest
	filterString := ""
	if filterFamilyName != "" {
		filterString = fmt.Sprintf("family.name=%s", filterFamilyName)
	}

	if filterFamilyIdStr != "" {
		filterFamilyId, err := strconv.Atoi(filterFamilyIdStr)
		if err != nil {
			writeInvalidQueryResponse(w, "Family ID must be an integer.")
			return
		}

		if filterString != "" {
			filterString = fmt.Sprintf("%s AND family.id=%d", filterString, filterFamilyId)
		} else {
			filterString = fmt.Sprintf("family.id=%d", filterFamilyId)
		}
	}

	if filterString == "" {
		request = &meilisearch.SearchRequest{
			Limit: searchLimit,
		}
	} else {
		request = &meilisearch.SearchRequest{
			Limit:  searchLimit,
			Filter: filterString,
		}
	}

	searchResp, err := index.Search(query, request)
	if err != nil {
		writeServerErrorResponse(w, "Could not search: "+err.Error())
		return
	}

	requestsTotal.Inc()
	requestsMountsSearch.Inc()

	if searchResp.EstimatedTotalHits == 0 {
		writeNotFoundResponse(w, "No results found.")
		return
	}

	txn := Db.Txn(false)
	defer txn.Abort()

	var mounts []APIMount
	for _, hit := range searchResp.Hits {
		indexed := hit.(map[string]interface{})
		itemId := int(indexed["id"].(float64))

		raw, err := txn.First(fmt.Sprintf("%s-%s", CurrentRedBlueVersionStr(Version.MemDb), "mounts"), "id", itemId)
		if err != nil || raw == nil {
			writeServerErrorResponse(w, "Could not find mount in database: "+err.Error())
			return
		}

		item := raw.(*mapping.MappedMultilangMount)
		mounts = append(mounts, RenderMountListEntry(item, lang))
	}

	WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(mounts)
	if err != nil {
		writeServerErrorResponse(w, "Could not encode JSON: "+err.Error())
		return
	}
}

func SearchSets(w http.ResponseWriter, r *http.Request) {
	client := meilisearch.New(MeiliHost, meilisearch.WithAPIKey(MeiliKey))
	defer client.Close()

	query := r.URL.Query().Get("query")
	if query == "" {
		writeInvalidQueryResponse(w, "Query parameter is required.")
		return
	}

	lang := r.Context().Value("lang").(string)
	filterMinLevel := strings.ToLower(r.URL.Query().Get("filter[min_highest_equipment_level]"))
	filterMaxLevel := strings.ToLower(r.URL.Query().Get("filter[max_highest_equipment_level]"))
	filterContainsCosmeticsStr := strings.ToLower(r.URL.Query().Get("filter[contains_cosmetics]"))
	filterContainsCosmeticsOnlyStr := strings.ToLower(r.URL.Query().Get("filter[contains_cosmetics_only]"))
	filterString, err := MinMaxLevelMeiliFilterFromParams(filterMinLevel, filterMaxLevel, "highest_equipment_level")
	if err != nil {
		writeInvalidFilterResponse(w, "Min/Max level filter is invalid: "+err.Error())
		return
	}

	if filterContainsCosmeticsOnlyStr != "" {
		filterIsCosmetic, err := strconv.ParseBool(filterContainsCosmeticsOnlyStr)
		if err != nil {
			writeInvalidFilterResponse(w, "filter[contains_cosmetics_only] must be a boolean.")
			return
		}
		isCosmeticMeiliFilterBoolString := strconv.FormatBool(filterIsCosmetic)
		if filterString != "" {
			filterString += " AND "
		}
		filterString += fmt.Sprintf("contains_cosmetics_only=%s", isCosmeticMeiliFilterBoolString)
	}

	if filterContainsCosmeticsStr != "" {
		filterIsCosmetic, err := strconv.ParseBool(filterContainsCosmeticsStr)
		if err != nil {
			writeInvalidFilterResponse(w, "filter[contains_cosmetics] must be a boolean.")
			return
		}
		isCosmeticMeiliFilterBoolString := strconv.FormatBool(filterIsCosmetic)
		if filterString != "" {
			filterString += " AND "
		}
		filterString += fmt.Sprintf("contains_cosmetics=%s", isCosmeticMeiliFilterBoolString)
	}

	var searchLimit int64
	if searchLimit, err = getLimitInBoundary(r.URL.Query().Get("limit")); err != nil {
		writeInvalidQueryResponse(w, "Limit parameter is invalid: "+err.Error())
		return
	}

	index := client.Index(fmt.Sprintf("%s-sets-%s", CurrentRedBlueVersionStr(Version.Search), lang))
	var request *meilisearch.SearchRequest

	if filterString == "" {
		request = &meilisearch.SearchRequest{
			Limit: searchLimit,
		}
	} else {
		request = &meilisearch.SearchRequest{
			Limit:  searchLimit,
			Filter: filterString,
		}
	}

	searchResp, err := index.Search(query, request)
	if err != nil {
		writeServerErrorResponse(w, "Could not search: "+err.Error())
		return
	}

	requestsTotal.Inc()
	requestsSetsSearch.Inc()

	if searchResp.EstimatedTotalHits == 0 {
		writeNotFoundResponse(w, "No results found.")
		return
	}

	txn := Db.Txn(false)
	defer txn.Abort()

	var sets []APIListSet
	for _, hit := range searchResp.Hits {
		indexed := hit.(map[string]interface{})
		itemId := int(indexed["id"].(float64))

		raw, err := txn.First(fmt.Sprintf("%s-%s", CurrentRedBlueVersionStr(Version.MemDb), "sets"), "id", itemId)
		if err != nil || raw == nil {
			writeServerErrorResponse(w, "Could not find set in database: "+err.Error())
			return
		}

		item := raw.(*mapping.MappedMultilangSetUnity)
		sets = append(sets, RenderSetListEntry(item, lang))
	}

	WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(sets)
	if err != nil {
		writeServerErrorResponse(w, "Could not encode JSON: "+err.Error())
		return
	}
}

func SearchAllIndices(w http.ResponseWriter, r *http.Request) {
	client := meilisearch.New(MeiliHost, meilisearch.WithAPIKey(MeiliKey))
	defer client.Close()

	query := r.URL.Query().Get("query")
	if query == "" {
		writeInvalidQueryResponse(w, "Query parameter is required.")
		return
	}

	filterSearchIndex := strings.ToLower(r.URL.Query().Get("filter[search_index]"))
	if filterSearchIndex == "" {
		filterSearchIndex = strings.Join(searchAllowedIndices, ",")
	}

	parsedIndices := parseFields(filterSearchIndex)
	if !validateFields(parsedIndices, searchAllowedIndices) {
		writeInvalidFilterResponse(w, "filter[type] has invalid fields.")
		return
	}

	itemExpansionsParam := strings.ToLower(r.URL.Query().Get("fields[item]"))
	itemExpansions := parseFields(itemExpansionsParam)
	if !validateFields(itemExpansions, searchAllItemAllowedExpandFields) {
		writeInvalidQueryResponse(w, "fields[item] has invalid fields.")
		return
	}

	lang := r.Context().Value("lang").(string)

	var searchLimit int64
	var err error
	if searchLimit, err = getLimitInBoundary(r.URL.Query().Get("limit")); err != nil {
		writeInvalidQueryResponse(w, "Limit parameter is invalid: "+err.Error())
		return
	}

	typeFiltering := strings.ToLower(r.URL.Query().Get("filter[type.name_id]"))
	exceptions := []string{"mount", "set"}
	filterset := parseFields(typeFiltering)
	additiveTypes, err := includeTypes(filterset, &exceptions)
	if err != nil {
		writeInvalidFilterResponse(w, "filter[type.name_id] is invalid: "+err.Error())
		return
	}

	removedTypes, err := excludeTypes(filterset, &exceptions)
	if err != nil {
		writeInvalidFilterResponse(w, "filter[type.name_id] is invalid: "+err.Error())
		return
	}

	indexName := fmt.Sprintf("%s-all_stuff-%s", CurrentRedBlueVersionStr(Version.Search), lang)
	index := client.Index(indexName)

	exceptionSet := set.NewHashset(uint64(len(exceptions)), g.Equals[string], g.HashString)
	for _, exception := range exceptions {
		exceptionSet.Put(exception)
	}

	additiveTypesExceptions := exceptionSet.Intersection(additiveTypes)
	removedTypesExceptions := exceptionSet.Intersection(removedTypes)

	additiveTypes = additiveTypes.Difference(exceptionSet)
	removedTypes = removedTypes.Difference(exceptionSet)

	filterString := ""
	if additiveTypes.Size() > 0 || additiveTypesExceptions.Size() > 0 {
		if filterString != "" {
			filterString += " AND "
		}
		// A single type is singular but the category is plural.
		plural := []string{}
		for _, elem := range additiveTypesExceptions.Keys() {
			plural = append(plural, elem+"s")
		}

		addTypesArr := additiveTypes.Keys()

		filterString += "("
		if len(addTypesArr) > 0 {
			filterString += "type.name_id=" + strings.Join(addTypesArr, " OR type.name_id=")
		}

		if len(plural) > 0 && len(addTypesArr) > 0 {
			filterString += " OR "
		}

		if len(plural) > 0 {
			filterString += "stuff_type.name_id=" + strings.Join(plural, " OR stuff_type.name_id=")
		}

		filterString += ")"
	}

	if removedTypes.Size() > 0 || removedTypesExceptions.Size() > 0 {
		if filterString != "" {
			filterString += " AND "
		}
		plural := []string{}
		for _, elem := range removedTypesExceptions.Keys() {
			plural = append(plural, elem+"s")
		}

		remTypesArr := removedTypes.Keys()

		filterString += "("
		if len(remTypesArr) > 0 {
			filterString += "NOT type.name_id=" + strings.Join(remTypesArr, " AND NOT type.name_id=")
		}

		if len(plural) > 0 && len(remTypesArr) > 0 {
			filterString += " AND "
		}

		if len(plural) > 0 {
			filterString += "NOT stuff_type.name_id=" + strings.Join(plural, " AND NOT stuff_type.name_id=")
		}

		filterString += ")"
	}

	if parsedIndices.Size() > 0 || additiveTypesExceptions.Size() > 0 || removedTypesExceptions.Size() > 0 {
		if parsedIndices.Size() > 0 {
			if filterString != "" {
				filterString += " AND "
			}

			filterString += "(stuff_type.name_id=" + strings.Join(parsedIndices.Keys(), " OR stuff_type.name_id=") + ")"
		}
	}

	request := &meilisearch.SearchRequest{
		Limit:  searchLimit,
		Filter: filterString,
	}

	var searchResp *meilisearch.SearchResponse
	if searchResp, err = index.Search(query, request); err != nil {
		writeServerErrorResponse(w, "Failed to search for query: "+err.Error())
		return
	}

	requestsTotal.Inc()
	requestsSearchTotal.Inc()

	if searchResp.EstimatedTotalHits == 0 {
		writeNotFoundResponse(w, "No results found.")
		return
	}

	txn := Db.Txn(false)
	defer txn.Abort()

	var stuffs []ApiAllSearchResult
	for _, hit := range searchResp.Hits {
		indexed := hit.(map[string]interface{})

		stuffType := indexed["stuff_type"].(map[string]interface{})["name_id"].(string)

		isItem := strings.HasPrefix(stuffType, "items-")
		ankamaId := int(indexed["id"].(float64))

		var itemInclude *ApiAllSearchItem
		if isItem {
			var itemType string
			switch stuffType {
			case "items-consumables":
				itemType = "consumables"
			case "items-cosmetics":
				itemType = "cosmetics"
			case "items-resources":
				itemType = "resources"
			case "items-equipment":
				itemType = "equipment"
			case "items-quest_items":
				itemType = "quest_items"
			default:
				writeServerErrorResponse(w, "Unknown stuff type: "+stuffType)
				return
			}

			txn := Db.Txn(false)
			defer txn.Abort()

			raw, err := txn.First(fmt.Sprintf("%s-%s", CurrentRedBlueVersionStr(Version.MemDb), itemType), "id", ankamaId)
			if err != nil {
				writeServerErrorResponse(w, "Could not find entity in database: "+err.Error())
				return
			}

			if raw == nil {
				log.Warn("Could not find item in memdb", "id", ankamaId)
				continue
			}

			item := raw.(*mapping.MappedMultilangItemUnity)
			itemFields := RenderItemListEntry(item, lang)

			itemInclude = &ApiAllSearchItem{}

			if itemExpansions.Has("type") {
				itemInclude.Type = &itemFields.Type
			}

			if itemExpansions.Has("level") {
				itemInclude.Level = &itemFields.Level
			}

			if itemExpansions.Has("image_urls") {
				itemInclude.ImageUrls = &itemFields.ImageUrls
			}

			if itemInclude.ImageUrls == nil && itemInclude.Type == nil && itemInclude.Level == nil {
				itemInclude = nil
			}
		}

		result := ApiAllSearchResult{
			Id:   ankamaId,
			Name: indexed["name"].(string),
			Type: ApiAllSearchResultType{
				NameId: stuffType,
			},
			ItemFields: itemInclude,
		}

		stuffs = append(stuffs, result)
	}

	WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(stuffs)
	if err != nil {
		writeServerErrorResponse(w, "Could not encode JSON: "+err.Error())
		return
	}
}

func SearchItems(itemType string, all bool, w http.ResponseWriter, r *http.Request) {
	client := meilisearch.New(MeiliHost, meilisearch.WithAPIKey(MeiliKey))
	defer client.Close()

	query := r.URL.Query().Get("query")
	if query == "" {
		writeInvalidQueryResponse(w, "Query parameter is required.")
		return
	}

	filterMinLevel := strings.ToLower(r.URL.Query().Get("filter[min_level]"))
	filterMaxLevel := strings.ToLower(r.URL.Query().Get("filter[max_level]"))
	filterString, err := MinMaxLevelMeiliFilterFromParams(filterMinLevel, filterMaxLevel, "level")
	if err != nil {
		writeInvalidFilterResponse(w, "Min/Max level filter is invalid: "+err.Error())
		return
	}

	lang := r.Context().Value("lang").(string)

	var searchLimit int64
	if searchLimit, err = getLimitInBoundary(r.URL.Query().Get("limit")); err != nil {
		writeInvalidQueryResponse(w, "Limit parameter is invalid: "+err.Error())
		return
	}

	typeFiltering := strings.ToLower(r.URL.Query().Get("filter[type.name_id]"))
	filterset := parseFields(typeFiltering)
	additiveTypes, err := includeTypes(filterset, nil)
	if err != nil {
		writeInvalidFilterResponse(w, "filter[type.name_id] is invalid: "+err.Error())
		return
	}

	removedTypes, err := excludeTypes(filterset, nil)
	if err != nil {
		writeInvalidFilterResponse(w, "filter[type.name_id] is invalid: "+err.Error())
		return
	}

	if additiveTypes.Size() > 0 {
		if filterString != "" {
			filterString += " AND "
		}
		filterString += "(type.name_id=" + strings.Join(additiveTypes.Keys(), " OR type.name_id=") + ")"
	}

	if removedTypes.Size() > 0 {
		if filterString != "" {
			filterString += " AND "
		}

		filterString += "(NOT type.name_id=" + strings.Join(removedTypes.Keys(), " AND NOT type.name_id=") + ")"
	}

	index := client.Index(fmt.Sprintf("%s-all_items-%s", CurrentRedBlueVersionStr(Version.Search), lang))
	var request *meilisearch.SearchRequest
	if !all {
		if filterString == "" {
			filterString += fmt.Sprintf("super_type.name_id=%s", itemType)
		} else {
			filterString += fmt.Sprintf(" AND super_type.name_id=%s", itemType)
		}
	}

	if filterString == "" {
		request = &meilisearch.SearchRequest{
			Limit: searchLimit,
		}
	} else {
		request = &meilisearch.SearchRequest{
			Limit:  searchLimit,
			Filter: filterString,
		}
	}

	searchResp, err := index.Search(query, request)
	if err != nil {
		writeServerErrorResponse(w, "Could not search: "+err.Error())
		return
	}

	requestsTotal.Inc()
	requestsItemsSearch.Inc()

	if searchResp.EstimatedTotalHits == 0 {
		writeNotFoundResponse(w, "No results found.")
		return
	}

	txn := Db.Txn(false)
	defer txn.Abort()

	var items []APIListItem
	var typedItems []APIListTypedItem
	for _, hit := range searchResp.Hits {
		indexed := hit.(map[string]interface{})
		itemId := int(indexed["id"].(float64))

		var raw interface{}
		if all {
			raw, err = txn.First(fmt.Sprintf("%s-%s", CurrentRedBlueVersionStr(Version.MemDb), "all_items"), "id", itemId)
		} else {
			raw, err = txn.First(fmt.Sprintf("%s-%s", CurrentRedBlueVersionStr(Version.MemDb), itemType), "id", itemId)
		}

		if err != nil {
			writeServerErrorResponse(w, "Could not find item in database: "+err.Error())
			return
		}

		if raw == nil {
			log.Warn("Item not found in memdb.", "id", itemId)
			continue
		}

		item := raw.(*mapping.MappedMultilangItemUnity)
		if all {
			typedItems = append(typedItems, RenderTypedItemListEntry(item, lang))
		} else {
			items = append(items, RenderItemListEntry(item, lang))
		}
	}

	WriteCacheHeader(&w)
	var encodeErr error
	if all {
		encodeErr = json.NewEncoder(w).Encode(typedItems)
	} else {
		encodeErr = json.NewEncoder(w).Encode(items)
	}
	if encodeErr != nil {
		writeServerErrorResponse(w, "Could not encode JSON: "+err.Error())
		return
	}
}

func SearchConsumables(w http.ResponseWriter, r *http.Request) {
	SearchItems("consumables", false, w, r)
}

func SearchCosmetics(w http.ResponseWriter, r *http.Request) {
	SearchItems("cosmetics", false, w, r)
}

func SearchResources(w http.ResponseWriter, r *http.Request) {
	SearchItems("resources", false, w, r)
}

func SearchEquipment(w http.ResponseWriter, r *http.Request) {
	SearchItems("equipment", false, w, r)
}

func SearchQuestItems(w http.ResponseWriter, r *http.Request) {
	SearchItems("quest_items", false, w, r)
}

func SearchAllItems(w http.ResponseWriter, r *http.Request) {
	SearchItems("", true, w, r)
}

// single

func GetSingleSetHandler(w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	ankamaId := r.Context().Value("ankamaId").(int)

	txn := Db.Txn(false)
	defer txn.Abort()

	raw, err := txn.First(fmt.Sprintf("%s-%s", CurrentRedBlueVersionStr(Version.MemDb), "sets"), "id", ankamaId)
	if err != nil || raw == nil {
		writeServerErrorResponse(w, "Could not find set in database: "+err.Error())
		return
	}

	requestsTotal.Inc()
	requestsSetsSingle.Inc()

	set := RenderSet(raw.(*mapping.MappedMultilangSetUnity), lang)
	WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(set)
	if err != nil {
		writeServerErrorResponse(w, "Could not encode JSON: "+err.Error())
		return
	}
}

func GetSingleMountHandler(w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	ankamaId := r.Context().Value("ankamaId").(int)

	txn := Db.Txn(false)
	defer txn.Abort()

	raw, err := txn.First(fmt.Sprintf("%s-%s", CurrentRedBlueVersionStr(Version.MemDb), "mounts"), "id", ankamaId)
	if err != nil || raw == nil {
		writeServerErrorResponse(w, "Could not find mount in database: "+err.Error())
		return
	}

	requestsTotal.Inc()
	requestsMountsSingle.Inc()

	mount := RenderMount(raw.(*mapping.MappedMultilangMount), lang)
	WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(mount)
	if err != nil {
		writeServerErrorResponse(w, "Could not encode JSON: "+err.Error())
		return
	}
}

func GetSingleItemWithOptionalRecipeHandler(itemType string, w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	ankamaId := r.Context().Value("ankamaId").(int)

	txn := Db.Txn(false)
	defer txn.Abort()

	raw, err := txn.First(fmt.Sprintf("%s-%s", CurrentRedBlueVersionStr(Version.MemDb), itemType), "id", ankamaId)
	if err != nil || raw == nil {
		writeServerErrorResponse(w, "Could not find item in database: "+err.Error())
		return
	}

	requestsTotal.Inc()
	requestsItemsSingle.Inc()

	resource := RenderResource(raw.(*mapping.MappedMultilangItemUnity), lang)
	recipe, exists := GetRecipeIfExists(ankamaId, txn)
	if exists {
		resource.Recipe = RenderRecipe(recipe, Db)
	}
	WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(resource)
	if err != nil {
		writeServerErrorResponse(w, "Could not encode JSON: "+err.Error())
		return
	}
}

func GetSingleConsumableHandler(w http.ResponseWriter, r *http.Request) {
	GetSingleItemWithOptionalRecipeHandler("consumables", w, r)
}

func GetSingleResourceHandler(w http.ResponseWriter, r *http.Request) {
	GetSingleItemWithOptionalRecipeHandler("resources", w, r)
}

func GetSingleQuestItemHandler(w http.ResponseWriter, r *http.Request) {
	GetSingleItemWithOptionalRecipeHandler("quest_items", w, r)
}

func GetSingleCosmeticHandler(w http.ResponseWriter, r *http.Request) {
	GetSingleEquipmentLikeHandler(true, w, r)
}

func GetSingleEquipmentHandler(w http.ResponseWriter, r *http.Request) {
	GetSingleEquipmentLikeHandler(false, w, r)
}

func GetSingleEquipmentLikeHandler(cosmetic bool, w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	ankamaId := r.Context().Value("ankamaId").(int)

	txn := Db.Txn(false)
	defer txn.Abort()

	dbType := ""
	if cosmetic {
		dbType = "cosmetics"
	} else {
		dbType = "equipment"
	}

	raw, err := txn.First(fmt.Sprintf("%s-%s", CurrentRedBlueVersionStr(Version.MemDb), dbType), "id", ankamaId)
	if err != nil || raw == nil {
		writeNotFoundResponse(w, "Item not found: "+strconv.Itoa(ankamaId))
		return
	}

	requestsTotal.Inc()
	requestsItemsSingle.Inc()

	item := raw.(*mapping.MappedMultilangItemUnity)
	if item.Type.SuperTypeId == 2 { // is weapon
		weapon := RenderWeapon(item, lang)
		recipe, exists := GetRecipeIfExists(ankamaId, txn)
		if exists {
			weapon.Recipe = RenderRecipe(recipe, Db)
		}
		WriteCacheHeader(&w)
		err = json.NewEncoder(w).Encode(weapon)
		if err != nil {
			writeServerErrorResponse(w, "Could not encode JSON: "+err.Error())
			return
		}
	} else {
		equipment := RenderEquipment(item, lang)
		recipe, exists := GetRecipeIfExists(ankamaId, txn)
		if exists {
			equipment.Recipe = RenderRecipe(recipe, Db)
		}
		WriteCacheHeader(&w)
		err = json.NewEncoder(w).Encode(equipment)
		if err != nil {
			writeServerErrorResponse(w, "Could not encode JSON: "+err.Error())
			return
		}

	}
}
