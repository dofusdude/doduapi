package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/dofusdude/api/gen"
	"github.com/dofusdude/api/utils"
	"github.com/hashicorp/go-memdb"
	"github.com/meilisearch/meilisearch-go"
)

var (
	mountAllowedExpandFields     = []string{"effects"}
	setAllowedExpandFields       = utils.Concat(mountAllowedExpandFields, []string{"equipment_ids"})
	itemAllowedExpandFields      = utils.Concat(mountAllowedExpandFields, []string{"recipe", "description", "conditions"})
	equipmentAllowedExpandFields = utils.Concat(itemAllowedExpandFields, []string{"range", "parent_set", "is_weapon", "pods", "critical_hit_probability", "critical_hit_bonus", "is_two_handed", "max_cast_per_turn", "ap_cost"})
)

func GetRecipeIfExists(itemId int, txn *memdb.Txn) (gen.MappedMultilangRecipe, bool) {
	raw, err := txn.First(fmt.Sprintf("%s-recipes", utils.CurrentRedBlueVersionStr(Version.MemDb)), "id", itemId)
	if err != nil {
		panic(err)
	}

	if raw != nil {
		recipe := raw.(*gen.MappedMultilangRecipe)
		return *recipe, true
	}

	return gen.MappedMultilangRecipe{}, false
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

func ListMonster(w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	pagination := utils.PageninationWithState(r.Context().Value("pagination").(string))

	filterRace := r.URL.Query().Get("filter[race]")
	filterRace = strings.ToLower(filterRace)
	filterSuperRace := r.URL.Query().Get("filter[super_race]")
	filterSuperRace = strings.ToLower(filterSuperRace)
	filterIsBossStr := r.URL.Query().Get("filter[is_boss]")
	filterIsBossStr = strings.ToLower(filterIsBossStr)
	if filterIsBossStr != "" {
		if filterIsBossStr != "true" && filterIsBossStr != "false" {
			http.Error(w, "filter[is_boss] must be true or false", http.StatusBadRequest)
			return
		}
	}
	filterIsBoss := filterIsBossStr == "true"

	txn := Db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get(fmt.Sprintf("%s-%s", utils.CurrentRedBlueVersionStr(Version.MemDb), "monster"), "id")
	if err != nil || it == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	requestsTotal.Inc()
	requestsMonsterList.Inc()

	var monster []APIMonster
	for obj := it.Next(); obj != nil; obj = it.Next() {
		p := obj.(*gen.MappedMultilangMonster)
		if filterIsBossStr != "" {
			if p.IsBoss != filterIsBoss {
				continue
			}
		}
		if filterRace != "" {
			if strings.ToLower(p.Race.Name[lang]) != filterRace {
				continue
			}
		}
		if filterSuperRace != "" {
			if strings.ToLower(p.Race.SuperRace.Name[lang]) != strings.ToLower(filterSuperRace) {
				continue
			}
		}
		mount := RenderMonster(p, lang)
		monster = append(monster, mount)
	}

	total := len(monster)
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
	paginatedMonsters := monster[startIdx:endIdx]

	response := APIPageMonster{
		Items: paginatedMonsters,
		Links: links,
	}

	utils.WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func ListMounts(w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	pagination := utils.PageninationWithState(r.Context().Value("pagination").(string))

	filterFamilyName := r.URL.Query().Get("filter[family_name]")
	expansionsParam := strings.ToLower(r.URL.Query().Get("fields[mount]"))
	var expansions *utils.Set
	expansions = parseFields(expansionsParam)
	if !validateFields(expansions, mountAllowedExpandFields) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	txn := Db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get(fmt.Sprintf("%s-%s", utils.CurrentRedBlueVersionStr(Version.MemDb), "mounts"), "id")
	if err != nil || it == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	requestsTotal.Inc()
	requestsMountsList.Inc()

	var mounts []APIListMount
	for obj := it.Next(); obj != nil; obj = it.Next() {
		p := obj.(*gen.MappedMultilangMount)
		if filterFamilyName != "" {
			if strings.ToLower(p.FamilyName[lang]) != strings.ToLower(filterFamilyName) {
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

	utils.WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func ListSets(w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	pagination := utils.PageninationWithState(r.Context().Value("pagination").(string))

	expansionsParam := strings.ToLower(r.URL.Query().Get("fields[set]"))
	var expansions *utils.Set
	expansions = parseFields(expansionsParam)
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

	it, err := txn.Get(fmt.Sprintf("%s-%s", utils.CurrentRedBlueVersionStr(Version.MemDb), "sets"), "id")
	if err != nil || it == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	requestsTotal.Inc()
	requestsSetsList.Inc()

	var sets []APIListSet
	for obj := it.Next(); obj != nil; obj = it.Next() {
		p := obj.(*gen.MappedMultilangSet)

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
				set.Effects = append(set.Effects, RenderEffects(&effect, lang))
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

	utils.WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func parseFields(expansionsParam string) *utils.Set {
	expansions := utils.NewSet()
	expansionContainsDiv := strings.Contains(expansionsParam, ",")
	if len(expansionsParam) != 0 && expansionContainsDiv {
		expansionsArr := strings.Split(expansionsParam, ",")
		for _, expansion := range expansionsArr {
			expansions.Add(expansion)
		}
	}
	if len(expansionsParam) != 0 && !expansionContainsDiv {
		expansions.Add(expansionsParam)
	}

	return expansions
}

func validateFields(expansions *utils.Set, list []string) bool {
	allowedFields := utils.NewSet()
	for _, expansion := range list {
		allowedFields.Add(expansion)
	}

	return expansions.Difference(allowedFields).Size() == 0
}

func ListItems(itemType string, w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	pagination := utils.PageninationWithState(r.Context().Value("pagination").(string))

	expansionsParam := strings.ToLower(r.URL.Query().Get("fields[item]"))
	var expansions *utils.Set
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

	it, err := txn.Get(fmt.Sprintf("%s-%s", utils.CurrentRedBlueVersionStr(Version.MemDb), itemType), "id")
	if err != nil || it == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	requestsItemsList.Inc()
	requestsTotal.Inc()

	var items []APIListItem
	for obj := it.Next(); obj != nil; obj = it.Next() {
		p := obj.(*gen.MappedMultilangItem)

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

	utils.WriteCacheHeader(&w)
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

// search

func SearchMonster(w http.ResponseWriter, r *http.Request) {
	var err error
	client := utils.CreateMeiliClient()
	query := r.URL.Query().Get("query")
	if query == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	raceName := strings.ToLower(r.URL.Query().Get("filter[race]"))
	superRaceName := strings.ToLower(r.URL.Query().Get("filter[super_race]"))
	isBoss := r.URL.Query().Get("filter[is_boss]")
	isBoss = strings.ToLower(isBoss)
	if isBoss != "" {
		if isBoss != "true" && isBoss != "false" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	lang := r.Context().Value("lang").(string)
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		limitStr = "8"
	}
	var limit int
	if limit, err = strconv.Atoi(limitStr); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if limit > 100 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var maxSearchResults int64 = int64(limit)

	index := client.Index(fmt.Sprintf("%s-monster-%s", utils.CurrentRedBlueVersionStr(Version.Search), lang))
	var request *meilisearch.SearchRequest
	filterString := ""
	if isBoss == "" {
		if raceName != "" && superRaceName == "" {
			filterString = fmt.Sprintf("race=%s", raceName)
		}
		if raceName == "" && superRaceName != "" {
			filterString = fmt.Sprintf("super_race=%s", superRaceName)
		}
		if raceName != "" && superRaceName != "" {
			filterString = fmt.Sprintf("race=%s and super_race=%s", raceName, superRaceName)
		}
	} else {
		if raceName != "" && superRaceName == "" {
			filterString = fmt.Sprintf("race=%s and is_boss=%s", raceName, isBoss)
		}
		if raceName == "" && superRaceName != "" {
			filterString = fmt.Sprintf("super_race=%s and is_boss=%s", superRaceName, isBoss)
		}
		if raceName != "" && superRaceName != "" {
			filterString = fmt.Sprintf("race=%s and super_race=%s and is_boss=%s", raceName, superRaceName, isBoss)
		}
		if filterString == "" {
			filterString = fmt.Sprintf("is_boss=%s", isBoss)
		}
	}

	if filterString == "" {
		request = &meilisearch.SearchRequest{
			Limit: maxSearchResults,
		}
	} else {
		request = &meilisearch.SearchRequest{
			Limit:  maxSearchResults,
			Filter: filterString,
		}
	}

	searchResp, err := index.Search(query, request)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	requestsTotal.Inc()
	requestsMonsterSearch.Inc()

	if searchResp.EstimatedTotalHits == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	txn := Db.Txn(false)
	defer txn.Abort()

	var monster []APIMonster
	for _, hit := range searchResp.Hits {
		indexed := hit.(map[string]interface{})
		itemId := int(indexed["id"].(float64))

		raw, err := txn.First(fmt.Sprintf("%s-%s", utils.CurrentRedBlueVersionStr(Version.MemDb), "monster"), "id", itemId)
		if err != nil || raw == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		item := raw.(*gen.MappedMultilangMonster)
		monster = append(monster, RenderMonster(item, lang))
	}

	utils.WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(monster)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func SearchMounts(w http.ResponseWriter, r *http.Request) {
	var err error
	client := utils.CreateMeiliClient()
	query := r.URL.Query().Get("query")
	if query == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		limitStr = "8"
	}
	var limit int
	if limit, err = strconv.Atoi(limitStr); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if limit > 100 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	familyName := strings.ToLower(r.URL.Query().Get("filter[family_name]"))

	lang := r.Context().Value("lang").(string)
	var maxSearchResults int64 = int64(limit)

	index := client.Index(fmt.Sprintf("%s-mounts-%s", utils.CurrentRedBlueVersionStr(Version.Search), lang))
	var request *meilisearch.SearchRequest
	filterString := ""
	if familyName != "" {
		filterString = fmt.Sprintf("family_name=%s", familyName)
	}

	if filterString == "" {
		request = &meilisearch.SearchRequest{
			Limit: maxSearchResults,
		}
	} else {
		request = &meilisearch.SearchRequest{
			Limit:  maxSearchResults,
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

		raw, err := txn.First(fmt.Sprintf("%s-%s", utils.CurrentRedBlueVersionStr(Version.MemDb), "mounts"), "id", itemId)
		if err != nil || raw == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		item := raw.(*gen.MappedMultilangMount)
		mounts = append(mounts, RenderMountListEntry(item, lang))
	}

	utils.WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(mounts)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func SearchSets(w http.ResponseWriter, r *http.Request) {
	client := utils.CreateMeiliClient()
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

	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		limitStr = "8"
	}
	var limit int
	if limit, err = strconv.Atoi(limitStr); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if limit > 100 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var maxSearchResults int64 = int64(limit)

	index := client.Index(fmt.Sprintf("%s-sets-%s", utils.CurrentRedBlueVersionStr(Version.Search), lang))
	var request *meilisearch.SearchRequest

	if filterString == "" {
		request = &meilisearch.SearchRequest{
			Limit: maxSearchResults,
		}
	} else {
		request = &meilisearch.SearchRequest{
			Limit:  maxSearchResults,
			Filter: filterString,
		}
	}

	request = &meilisearch.SearchRequest{
		Limit: maxSearchResults,
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

		raw, err := txn.First(fmt.Sprintf("%s-%s", utils.CurrentRedBlueVersionStr(Version.MemDb), "sets"), "id", itemId)
		if err != nil || raw == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		item := raw.(*gen.MappedMultilangSet)
		sets = append(sets, RenderSetListEntry(item, lang))
	}

	utils.WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(sets)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func SearchItems(itemType string, all bool, w http.ResponseWriter, r *http.Request) {
	client := utils.CreateMeiliClient()
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

	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		limitStr = "8"
	}
	var limit int
	if limit, err = strconv.Atoi(limitStr); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if limit > 100 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var maxSearchResults int64 = int64(limit)

	index := client.Index(fmt.Sprintf("%s-all_items-%s", utils.CurrentRedBlueVersionStr(Version.Search), lang))
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
			Limit: maxSearchResults,
		}
	} else {
		request = &meilisearch.SearchRequest{
			Limit:  maxSearchResults,
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
			raw, err = txn.First(fmt.Sprintf("%s-%s", utils.CurrentRedBlueVersionStr(Version.MemDb), "all_items"), "id", itemId)
		} else {
			raw, err = txn.First(fmt.Sprintf("%s-%s", utils.CurrentRedBlueVersionStr(Version.MemDb), itemType), "id", itemId)
		}
		if err != nil || raw == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		item := raw.(*gen.MappedMultilangItem)
		if all {
			typedItems = append(typedItems, RenderTypedItemListEntry(item, lang))
		} else {
			items = append(items, RenderItemListEntry(item, lang))
		}
	}

	utils.WriteCacheHeader(&w)
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

func GetSingleMonsterHandler(w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	ankamaId := r.Context().Value("ankamaId").(int)

	txn := Db.Txn(false)
	defer txn.Abort()

	raw, err := txn.First(fmt.Sprintf("%s-%s", utils.CurrentRedBlueVersionStr(Version.MemDb), "monster"), "id", ankamaId)
	if err != nil || raw == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	requestsTotal.Inc()
	requestsMonsterSingle.Inc()

	monster := RenderMonster(raw.(*gen.MappedMultilangMonster), lang)
	utils.WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(monster)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func GetSingleSetHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("GetSingleSetHandler")
	lang := r.Context().Value("lang").(string)
	ankamaId := r.Context().Value("ankamaId").(int)

	txn := Db.Txn(false)
	defer txn.Abort()

	raw, err := txn.First(fmt.Sprintf("%s-%s", utils.CurrentRedBlueVersionStr(Version.MemDb), "sets"), "id", ankamaId)
	if err != nil || raw == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	requestsTotal.Inc()
	requestsSetsSingle.Inc()

	set := RenderSet(raw.(*gen.MappedMultilangSet), lang)
	utils.WriteCacheHeader(&w)
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

	raw, err := txn.First(fmt.Sprintf("%s-%s", utils.CurrentRedBlueVersionStr(Version.MemDb), "mounts"), "id", ankamaId)
	if err != nil || raw == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	requestsTotal.Inc()
	requestsMountsSingle.Inc()

	mount := RenderMount(raw.(*gen.MappedMultilangMount), lang)
	utils.WriteCacheHeader(&w)
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

	raw, err := txn.First(fmt.Sprintf("%s-%s", utils.CurrentRedBlueVersionStr(Version.MemDb), itemType), "id", ankamaId)
	if err != nil || raw == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	requestsTotal.Inc()
	requestsItemsSingle.Inc()

	resource := RenderResource(raw.(*gen.MappedMultilangItem), lang)
	recipe, exists := GetRecipeIfExists(ankamaId, txn)
	if exists {
		resource.Recipe = RenderRecipe(recipe, Db)
	}
	utils.WriteCacheHeader(&w)
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

	raw, err := txn.First(fmt.Sprintf("%s-equipment", utils.CurrentRedBlueVersionStr(Version.MemDb)), "id", ankamaId)
	if err != nil || raw == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	requestsTotal.Inc()
	requestsItemsSingle.Inc()

	item := raw.(*gen.MappedMultilangItem)
	if item.Type.SuperTypeId == 2 { // is weapon
		weapon := RenderWeapon(item, lang)
		recipe, exists := GetRecipeIfExists(ankamaId, txn)
		if exists {
			weapon.Recipe = RenderRecipe(recipe, Db)
		}
		utils.WriteCacheHeader(&w)
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
		utils.WriteCacheHeader(&w)
		err = json.NewEncoder(w).Encode(equipment)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	}
}
