package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/dofusdude/api/gen"
	"github.com/dofusdude/api/utils"
	"github.com/hashicorp/go-memdb"
	"github.com/meilisearch/meilisearch-go"
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

// listing
func ListMounts(w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	pagination := utils.PageninationWithState(r.Context().Value("pagination").(string))

	filterFamily := r.URL.Query().Get("filter[family]")

	txn := Db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get(fmt.Sprintf("%s-%s", utils.CurrentRedBlueVersionStr(Version.MemDb), "mounts"), "id")
	if err != nil || it == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var mounts []APIListMount
	for obj := it.Next(); obj != nil; obj = it.Next() {
		p := obj.(*gen.MappedMultilangMount)
		if filterFamily != "" {
			if strings.ToLower(p.FamilyName["en"]) != strings.ToLower(filterFamily) {
				continue
			}
		}
		mount := RenderMountListEntry(p, lang)
		mounts = append(mounts, mount)
	}

	total := len(mounts)

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

	json.NewEncoder(w).Encode(response)
}

func ListSets(w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	pagination := utils.PageninationWithState(r.Context().Value("pagination").(string))

	sortLevel := r.URL.Query().Get("sort[level]")

	txn := Db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get(fmt.Sprintf("%s-%s", utils.CurrentRedBlueVersionStr(Version.MemDb), "sets"), "id")
	if err != nil || it == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var sets []APIListSet
	for obj := it.Next(); obj != nil; obj = it.Next() {
		p := obj.(*gen.MappedMultilangSet)
		mount := RenderSetListEntry(p, lang)
		sets = append(sets, mount)
	}

	total := len(sets)

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

	json.NewEncoder(w).Encode(response)
}

func ListItems(itemType string, w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	pagination := utils.PageninationWithState(r.Context().Value("pagination").(string))

	sortLevel := r.URL.Query().Get("sort[level]")

	txn := Db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get(fmt.Sprintf("%s-%s", utils.CurrentRedBlueVersionStr(Version.MemDb), itemType), "id")
	if err != nil || it == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var items []APIListItem
	for obj := it.Next(); obj != nil; obj = it.Next() {
		p := obj.(*gen.MappedMultilangItem)
		consumable := RenderItemListEntry(p, lang)
		items = append(items, consumable)
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

	json.NewEncoder(w).Encode(response)
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
func SearchMounts(w http.ResponseWriter, r *http.Request) {
	client := utils.CreateMeiliClient()
	query := r.URL.Query().Get("q")
	if query == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	familyName := r.URL.Query().Get("filter[family]")

	lang := r.Context().Value("lang").(string)
	var maxSearchResults int64 = 8

	index := client.Index(fmt.Sprintf("%s-mounts-%s", utils.CurrentRedBlueVersionStr(Version.Search), lang))
	var request *meilisearch.SearchRequest
	if familyName == "" {
		request = &meilisearch.SearchRequest{
			Limit: maxSearchResults,
		}
	} else {
		request = &meilisearch.SearchRequest{
			Limit:  maxSearchResults,
			Filter: fmt.Sprintf("family_name=%s", familyName),
		}
	}
	searchResp, err := index.Search(query, request)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

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

	json.NewEncoder(w).Encode(mounts)
}

func SearchSets(w http.ResponseWriter, r *http.Request) {
	client := utils.CreateMeiliClient()
	query := r.URL.Query().Get("q")
	if query == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	lang := r.Context().Value("lang").(string)
	var maxSearchResults int64 = 8

	index := client.Index(fmt.Sprintf("%s-sets-%s", utils.CurrentRedBlueVersionStr(Version.Search), lang))
	var request *meilisearch.SearchRequest
	request = &meilisearch.SearchRequest{
		Limit: maxSearchResults,
	}

	searchResp, err := index.Search(query, request)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

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

	json.NewEncoder(w).Encode(sets)
}

func SearchItems(itemType string, all bool, w http.ResponseWriter, r *http.Request) {
	client := utils.CreateMeiliClient()
	query := r.URL.Query().Get("q")
	if query == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	lang := r.Context().Value("lang").(string)
	var maxSearchResults int64 = 8

	index := client.Index(fmt.Sprintf("%s-all_items-%s", utils.CurrentRedBlueVersionStr(Version.Search), lang))
	var request *meilisearch.SearchRequest
	if all {
		request = &meilisearch.SearchRequest{
			Limit: maxSearchResults,
		}
	} else {
		request = &meilisearch.SearchRequest{
			Limit:  maxSearchResults,
			Filter: fmt.Sprintf("type=%s", itemType),
		}
	}
	searchResp, err := index.Search(query, request)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

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

	if all {
		json.NewEncoder(w).Encode(typedItems)
	} else {
		json.NewEncoder(w).Encode(items)
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

	raw, err := txn.First(fmt.Sprintf("%s-%s", utils.CurrentRedBlueVersionStr(Version.MemDb), "sets"), "id", ankamaId)
	if err != nil || raw == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	set := RenderSet(raw.(*gen.MappedMultilangSet), lang)
	json.NewEncoder(w).Encode(set)
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

	mount := RenderMount(raw.(*gen.MappedMultilangMount), lang)
	json.NewEncoder(w).Encode(mount)
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

	resource := RenderResource(raw.(*gen.MappedMultilangItem), lang)
	recipe, exists := GetRecipeIfExists(ankamaId, txn)
	if exists {
		resource.SetRecipe(RenderRecipe(recipe))
	}
	json.NewEncoder(w).Encode(resource)
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

	item := raw.(*gen.MappedMultilangItem)
	if item.Type.SuperTypeId == 2 { // is weapon
		weapon := RenderWeapon(item, lang)
		recipe, exists := GetRecipeIfExists(ankamaId, txn)
		if exists {
			weapon.Recipe = RenderRecipe(recipe)
		}
		json.NewEncoder(w).Encode(weapon)
	} else {
		equipment := RenderEquipment(item, lang)
		recipe, exists := GetRecipeIfExists(ankamaId, txn)
		if exists {
			equipment.Recipe = RenderRecipe(recipe)
		}
		json.NewEncoder(w).Encode(equipment)

	}
}
