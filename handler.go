package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

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
	equipmentAllowedExpandFields = Concat(itemAllowedExpandFields, []string{"range", "parent_set", "is_weapon", "pods", "critical_hit_probability", "critical_hit_bonus", "is_two_handed", "max_cast_per_turn", "ap_cost"})
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
		log.Error(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var release string
	if IsBeta {
		release = "beta"
	} else {
		release = "main"
	}

	ReleaseUrl = fmt.Sprintf("https://github.com/dofusdude/dofus2-%s/releases/download/%s", release, updateMessage.Version)

	log.Info("Updating to version", updateMessage.Version)
	err := DownloadImages()
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	UpdateChan <- true
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

	filterFamilyName := r.URL.Query().Get("filter[family_name]")
	expansionsParam := strings.ToLower(r.URL.Query().Get("fields[mount]"))
	expansions := parseFields(expansionsParam)
	if !validateFields(expansions, mountAllowedExpandFields) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	txn := Db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get(fmt.Sprintf("%s-%s", CurrentRedBlueVersionStr(Version.MemDb), "mounts"), "id")
	if err != nil || it == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	requestsTotal.Inc()
	requestsMountsList.Inc()

	var mounts []APIListMount
	for obj := it.Next(); obj != nil; obj = it.Next() {
		p := obj.(*mapping.MappedMultilangMount)
		if filterFamilyName != "" {
			if !strings.EqualFold(p.FamilyName[lang], filterFamilyName) {
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
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if pagination.ValidatePagination(total) != 0 {
		w.WriteHeader(http.StatusBadRequest)
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
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func ListSets(w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	pagination := PageninationWithState(r.Context().Value("pagination").(string))

	expansionsParam := strings.ToLower(r.URL.Query().Get("fields[set]"))
	expansions := parseFields(expansionsParam)
	if !validateFields(expansions, setAllowedExpandFields) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sortLevel := strings.ToLower(r.URL.Query().Get("sort[level]"))
	filterMinLevel := strings.ToLower(r.URL.Query().Get("filter[min_highest_equipment_level]"))
	filterMaxLevel := strings.ToLower(r.URL.Query().Get("filter[max_highest_equipment_level]"))
	filterMinLevelInt, filterMaxLevelInt, err := MinMaxLevelInt(filterMinLevel, filterMaxLevel, "highest_equipment_level")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	txn := Db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get(fmt.Sprintf("%s-%s", CurrentRedBlueVersionStr(Version.MemDb), "sets"), "id")
	if err != nil || it == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	requestsTotal.Inc()
	requestsSetsList.Inc()

	var sets []APIListSet
	for obj := it.Next(); obj != nil; obj = it.Next() {
		p := obj.(*mapping.MappedMultilangSet)

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
			for _, effect := range p.Effects {
				set.Effects = append(set.Effects, RenderSetEffects(&effect, lang))
			}
		}

		if expansions.Has("equipment_ids") {
			set.ItemIds = p.ItemIds
		}

		sets = append(sets, set)
	}

	total := len(sets)
	if total == 0 {
		w.WriteHeader(http.StatusNotFound)
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
		w.WriteHeader(http.StatusBadRequest)
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
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func isValidType(typeid string) bool {
	// TODO
	return true
}

func setFilter(in *set.Set[string], prefix string) (*set.Set[string], error) {
	out := set.NewHashset[string](10, g.Equals[string], g.HashString)
	for _, str := range in.Keys() {
		noPrefix := str
		if prefix != "" {
			if strings.HasPrefix(str, prefix) {
				noPrefix = strings.TrimPrefix(str, prefix)
			}
		}
		if isValidType(noPrefix) {
			out.Put(noPrefix)
		} else {
			return &out, fmt.Errorf("unknown type: " + noPrefix)
		}
	}

	return &out, nil
}

func excludeTypes(all *set.Set[string]) (*set.Set[string], error) {
	return setFilter(all, "-")
}

func includeTypes(all *set.Set[string]) (*set.Set[string], error) {
	explicitAdd, err := setFilter(all, "+")
	if err != nil {
		return nil, err
	}

	implicitAdd, err := setFilter(all, "")
	if err != nil {
		return nil, err
	}

	res := explicitAdd.Union(implicitAdd)
	return &res, nil
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
	allowedFields := set.NewHashset[string](10, g.Equals[string], g.HashString)
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
	if itemType == "equipment" {
		expansions = parseFields(expansionsParam)
		if !validateFields(expansions, equipmentAllowedExpandFields) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	} else {
		expansions = parseFields(expansionsParam)
		if !validateFields(expansions, itemAllowedExpandFields) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	typeFiltering := strings.ToLower(r.URL.Query().Get("filter[type]"))
	filterset := parseFields(typeFiltering)
	additiveTypes, err := includeTypes(filterset)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	removedTypes, err := excludeTypes(filterset)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// TODO validate fields
	//if !validateFields(expansions, equipmentAllowedExpandFields) {
	//	w.WriteHeader(http.StatusBadRequest)
	//	return
	//}

	sortLevel := strings.ToLower(r.URL.Query().Get("sort[level]"))
	filterTypeName := strings.ToLower(r.URL.Query().Get("filter[type_name]"))
	filterMinLevel := strings.ToLower(r.URL.Query().Get("filter[min_level]"))
	filterMaxLevel := strings.ToLower(r.URL.Query().Get("filter[max_level]"))
	filterMinLevelInt, filterMaxLevelInt, err := MinMaxLevelInt(filterMinLevel, filterMaxLevel, "level")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	txn := Db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get(fmt.Sprintf("%s-%s", CurrentRedBlueVersionStr(Version.MemDb), itemType), "id")
	if err != nil || it == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	requestsItemsList.Inc()
	requestsTotal.Inc()

	var items []APIListItem
	for obj := it.Next(); obj != nil; obj = it.Next() {
		p := obj.(*mapping.MappedMultilangItem)

		enTypeName := strings.ToLower(p.Type.Name["en"])

		if removedTypes.Has(enTypeName) {
			continue
		}

		if additiveTypes.Size() > 0 && !additiveTypes.Has(enTypeName) {
			continue
		}

		if filterTypeName != "" {
			if strings.ToLower(p.Type.Name[lang]) != filterTypeName {
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
				item.Conditions = RenderConditions(&p.Conditions, lang)
			}

			if p.ConditionTree != nil {
				item.ConditionTree = RenderConditionTree(p.ConditionTree, lang)
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

			if expansions.Has("is_two_handed") {
				item.TwoHanded = &p.TwoHanded
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
		w.WriteHeader(http.StatusNotFound)
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
		w.WriteHeader(http.StatusBadRequest)
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
		w.WriteHeader(http.StatusInternalServerError)
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
	client := CreateMeiliClient()
	query := r.URL.Query().Get("query")
	if query == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	lang := r.Context().Value("lang").(string)

	if lang == "pt" {
		log.Info("SearchAlmanaxBonuses: pt is not supported")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var searchLimit int64
	var err error
	if searchLimit, err = getLimitInBoundary(r.URL.Query().Get("limit")); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	indexName := fmt.Sprintf("alm-bonuses-%s", lang)
	index := client.Index(indexName)

	request := &meilisearch.SearchRequest{
		Limit: searchLimit,
	}

	var searchResp *meilisearch.SearchResponse
	if searchResp, err = index.Search(query, request); err != nil {
		log.Warn("SearchAlmanaxBonuses: index not found", "err", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	requestsTotal.Inc()
	requestsSearchTotal.Inc()

	if searchResp.EstimatedTotalHits == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var results []AlmanaxBonusListing
	for _, hit := range searchResp.Hits {
		almBonusJson := hit.(map[string]interface{})
		almBonus := AlmanaxBonusListing{
			Id:   almBonusJson["id"].(string),
			Name: almBonusJson["name"].(string),
		}
		results = append(results, almBonus)
	}

	WriteCacheHeader(&w)
	if json.NewEncoder(w).Encode(results) != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func SearchMounts(w http.ResponseWriter, r *http.Request) {
	var err error
	client := CreateMeiliClient()
	query := r.URL.Query().Get("query")
	if query == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var searchLimit int64
	if searchLimit, err = getLimitInBoundary(r.URL.Query().Get("limit")); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	familyName := strings.ToLower(r.URL.Query().Get("filter[family_name]"))

	lang := r.Context().Value("lang").(string)

	index := client.Index(fmt.Sprintf("%s-mounts-%s", CurrentRedBlueVersionStr(Version.Search), lang))
	var request *meilisearch.SearchRequest
	filterString := ""
	if familyName != "" {
		filterString = fmt.Sprintf("family_name=%s", familyName)
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
		w.WriteHeader(http.StatusNotFound)
		return
	}

	requestsTotal.Inc()
	requestsMountsSearch.Inc()

	if searchResp.EstimatedTotalHits == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	txn := Db.Txn(false)
	defer txn.Abort()

	var mounts []APIListMount
	for _, hit := range searchResp.Hits {
		indexed := hit.(map[string]interface{})
		itemId := int(indexed["id"].(float64))

		raw, err := txn.First(fmt.Sprintf("%s-%s", CurrentRedBlueVersionStr(Version.MemDb), "mounts"), "id", itemId)
		if err != nil || raw == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		item := raw.(*mapping.MappedMultilangMount)
		mounts = append(mounts, RenderMountListEntry(item, lang))
	}

	WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(mounts)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func SearchSets(w http.ResponseWriter, r *http.Request) {
	client := CreateMeiliClient()
	query := r.URL.Query().Get("query")
	if query == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	lang := r.Context().Value("lang").(string)
	filterMinLevel := strings.ToLower(r.URL.Query().Get("filter[min_highest_equipment_level]"))
	filterMaxLevel := strings.ToLower(r.URL.Query().Get("filter[max_highest_equipment_level]"))
	filterString, err := MinMaxLevelMeiliFilterFromParams(filterMinLevel, filterMaxLevel, "highest_equipment_level")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var searchLimit int64
	if searchLimit, err = getLimitInBoundary(r.URL.Query().Get("limit")); err != nil {
		w.WriteHeader(http.StatusBadRequest)
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

	request = &meilisearch.SearchRequest{
		Limit: searchLimit,
	}

	searchResp, err := index.Search(query, request)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	requestsTotal.Inc()
	requestsSetsSearch.Inc()

	if searchResp.EstimatedTotalHits == 0 {
		w.WriteHeader(http.StatusNotFound)
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
			w.WriteHeader(http.StatusNotFound)
			return
		}

		item := raw.(*mapping.MappedMultilangSet)
		sets = append(sets, RenderSetListEntry(item, lang))
	}

	WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(sets)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func SearchAllIndices(w http.ResponseWriter, r *http.Request) {
	client := CreateMeiliClient()
	query := r.URL.Query().Get("query")
	if query == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filterSearchIndex := strings.ToLower(r.URL.Query().Get("filter[type]"))
	if filterSearchIndex == "" {
		filterSearchIndex = strings.Join(searchAllowedIndices, ",")
	}

	parsedIndices := parseFields(filterSearchIndex)
	if !validateFields(parsedIndices, searchAllowedIndices) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	itemExpansionsParam := strings.ToLower(r.URL.Query().Get("fields[item]"))
	itemExpansions := parseFields(itemExpansionsParam)
	if !validateFields(itemExpansions, searchAllItemAllowedExpandFields) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	lang := r.Context().Value("lang").(string)

	var searchLimit int64
	var err error
	if searchLimit, err = getLimitInBoundary(r.URL.Query().Get("limit")); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	typeFiltering := strings.ToLower(r.URL.Query().Get("filter[type]"))
	filterset := parseFields(typeFiltering)
	additiveTypes, err := includeTypes(filterset)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	removedTypes, err := excludeTypes(filterset)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	indexName := fmt.Sprintf("%s-all_stuff-%s", CurrentRedBlueVersionStr(Version.Search), lang)
	index := client.Index(indexName)
	filterString := "stuff_type=" + strings.Join(parsedIndices.Keys(), " OR stuff_type=")

	if additiveTypes.Size() > 0 {
		filterString += " OR type_id=" + strings.Join(additiveTypes.Keys(), " OR type_id=")
	}

	if removedTypes.Size() > 0 {
		filterString += " AND NOT type_id=" + strings.Join(removedTypes.Keys(), " AND NOT type_id=")
	}

	request := &meilisearch.SearchRequest{
		Limit:  searchLimit,
		Filter: filterString,
	}

	log.Info(filterString)

	var searchResp *meilisearch.SearchResponse
	if searchResp, err = index.Search(query, request); err != nil {
		log.Warn("SearchAllIndices: index not found: ", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	requestsTotal.Inc()
	requestsSearchTotal.Inc()

	if searchResp.EstimatedTotalHits == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	txn := Db.Txn(false)
	defer txn.Abort()

	var stuffs []ApiAllSearchResult
	for _, hit := range searchResp.Hits {
		indexed := hit.(map[string]interface{})

		isItem := strings.HasPrefix(indexed["stuff_type"].(string), "items-")
		ankamaId := int(indexed["id"].(float64))
		stuffType := indexed["stuff_type"].(string)

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
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			txn := Db.Txn(false)
			defer txn.Abort()

			raw, err := txn.First(fmt.Sprintf("%s-%s", CurrentRedBlueVersionStr(Version.MemDb), itemType), "id", ankamaId)
			if err != nil || raw == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			item := raw.(*mapping.MappedMultilangItem)
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
			Id:         ankamaId,
			Name:       indexed["name"].(string),
			Type:       stuffType,
			ItemFields: itemInclude,
		}

		stuffs = append(stuffs, result)
	}

	WriteCacheHeader(&w)
	if json.NewEncoder(w).Encode(stuffs) != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func SearchItems(itemType string, all bool, w http.ResponseWriter, r *http.Request) {
	client := CreateMeiliClient()
	query := r.URL.Query().Get("query")
	if query == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filterTypeName := strings.ToLower(r.URL.Query().Get("filter[type_name]"))
	filterMinLevel := strings.ToLower(r.URL.Query().Get("filter[min_level]"))
	filterMaxLevel := strings.ToLower(r.URL.Query().Get("filter[max_level]"))
	filterString, err := MinMaxLevelMeiliFilterFromParams(filterMinLevel, filterMaxLevel, "level")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	lang := r.Context().Value("lang").(string)

	var searchLimit int64
	if searchLimit, err = getLimitInBoundary(r.URL.Query().Get("limit")); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	index := client.Index(fmt.Sprintf("%s-all_items-%s", CurrentRedBlueVersionStr(Version.Search), lang))
	var request *meilisearch.SearchRequest
	if all {
		if filterTypeName != "" {
			if filterString == "" {
				filterString = fmt.Sprintf("type_name=%s", filterTypeName)
			} else {
				filterString = fmt.Sprintf("%s AND type_name=%s", filterString, filterTypeName)
			}
		}
	} else {
		if filterTypeName != "" {
			if filterString == "" { // not already set with MinMaxLevels
				filterString = fmt.Sprintf("type_name=%s", filterTypeName)
			} else {
				filterString = fmt.Sprintf("%s AND type_name=%s", filterString, filterTypeName)
			}
		}

		if filterString == "" {
			filterString = fmt.Sprintf("super_type=%s", itemType)
		} else {
			filterString = fmt.Sprintf("%s AND super_type=%s", filterString, itemType)
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
		w.WriteHeader(http.StatusNotFound)
		return
	}

	requestsTotal.Inc()
	requestsItemsSearch.Inc()

	if searchResp.EstimatedTotalHits == 0 {
		w.WriteHeader(http.StatusNotFound)
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
		if err != nil || raw == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		item := raw.(*mapping.MappedMultilangItem)
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
		w.WriteHeader(http.StatusInternalServerError)
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
	fmt.Println("GetSingleSetHandler")
	lang := r.Context().Value("lang").(string)
	ankamaId := r.Context().Value("ankamaId").(int)

	txn := Db.Txn(false)
	defer txn.Abort()

	raw, err := txn.First(fmt.Sprintf("%s-%s", CurrentRedBlueVersionStr(Version.MemDb), "sets"), "id", ankamaId)
	if err != nil || raw == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	requestsTotal.Inc()
	requestsSetsSingle.Inc()

	set := RenderSet(raw.(*mapping.MappedMultilangSet), lang)
	WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(set)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
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
		w.WriteHeader(http.StatusNotFound)
		return
	}

	requestsTotal.Inc()
	requestsMountsSingle.Inc()

	mount := RenderMount(raw.(*mapping.MappedMultilangMount), lang)
	WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(mount)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
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
		w.WriteHeader(http.StatusNotFound)
		return
	}

	requestsTotal.Inc()
	requestsItemsSingle.Inc()

	resource := RenderResource(raw.(*mapping.MappedMultilangItem), lang)
	recipe, exists := GetRecipeIfExists(ankamaId, txn)
	if exists {
		resource.Recipe = RenderRecipe(recipe, Db)
	}
	WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(resource)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
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
	GetSingleItemWithOptionalRecipeHandler("cosmetics", w, r)
}

func GetSingleEquipmentHandler(w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	ankamaId := r.Context().Value("ankamaId").(int)

	txn := Db.Txn(false)
	defer txn.Abort()

	raw, err := txn.First(fmt.Sprintf("%s-equipment", CurrentRedBlueVersionStr(Version.MemDb)), "id", ankamaId)
	if err != nil || raw == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	requestsTotal.Inc()
	requestsItemsSingle.Inc()

	item := raw.(*mapping.MappedMultilangItem)
	if item.Type.SuperTypeId == 2 { // is weapon
		weapon := RenderWeapon(item, lang)
		recipe, exists := GetRecipeIfExists(ankamaId, txn)
		if exists {
			weapon.Recipe = RenderRecipe(recipe, Db)
		}
		WriteCacheHeader(&w)
		err = json.NewEncoder(w).Encode(weapon)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
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
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	}
}
