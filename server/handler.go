package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/dofusdude/api/gen"
	"github.com/dofusdude/api/utils"
	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-memdb"
	"github.com/meilisearch/meilisearch-go"
)

func paginate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		maxPageSize := 32

		pageNumStr := chi.URLParam(r, "pnum")
		var pageNum int

		pageSizeStr := chi.URLParam(r, "psize")
		var pageSize int
		if pageSizeStr == "" && pageNumStr != "" {
			pageSize = maxPageSize / 2
			pageNum, err := strconv.Atoi(pageNumStr)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if pageNum < 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}

		if pageSizeStr != "" {
			pageSize, err := strconv.Atoi(pageSizeStr)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if pageSize > maxPageSize || pageSize < 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}

		if pageSizeStr == "" && pageNumStr == "" {
			pageSize = maxPageSize / 2
			pageNum = 1
		}

		ctx := utils.WithValues(r.Context(), 
			"pageSize", pageSize,
			"pageNum", pageNum,
		)

		next.ServeHTTP(w, r.WithContext(ctx))

	})
}

func languageChecker(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := chi.URLParam(r, "lang")
		switch lang {
		case "en", "fr", "de", "es", "it", "pt":
			ctx := context.WithValue(r.Context(), "lang", lang)
			next.ServeHTTP(w, r.WithContext(ctx))
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	})
}

func ankamaIdExtractor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ankamaId, err := strconv.Atoi(chi.URLParam(r, "ankamaId"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), "ankamaId", ankamaId)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

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

func ListItems(itemType string, w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	pagination := utils.Pagination{
		PageSize: r.Context().Value("pageSize").(int),
		PageNumber:  r.Context().Value("pageNum").(int),
		BiggestPageSize: 32,
	}
	sortLevel := r.URL.Query().Get("slevel")

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
	links, _ := pagination.BuildLinks(total)
	paginatedItems := items[startIdx:endIdx]

	response := APIPageItem{
		Items: paginatedItems,
		Links: links,
	}

	json.NewEncoder(w).Encode(response)
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
			Limit: maxSearchResults,
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
	for _, hit := range searchResp.Hits {
		indexed := hit.(map[string]interface{})
		itemId := int(indexed["id"].(float64))

		raw, err := txn.First(fmt.Sprintf("%s-%s", utils.CurrentRedBlueVersionStr(Version.MemDb), itemType), "id", itemId)
		if err != nil || raw == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		item := raw.(*gen.MappedMultilangItem)
		items = append(items, RenderItemListEntry(item, lang))
	}

	json.NewEncoder(w).Encode(items)
}

func SearchAllItems(w http.ResponseWriter, r *http.Request) {
	SearchItems("", true, w, r)
}

func GetSingleConsumableHandler(w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	ankamaId := r.Context().Value("ankamaId").(int)

	txn := Db.Txn(false)
	defer txn.Abort()

	raw, err := txn.First(fmt.Sprintf("%s-consumables", utils.CurrentRedBlueVersionStr(Version.MemDb)), "id", ankamaId)
	if err != nil || raw == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	consumable := RenderConsumable(raw.(*gen.MappedMultilangItem), lang)
	recipe, exists := GetRecipeIfExists(ankamaId, txn)
	if exists {
		consumable.Recipe = RenderRecipe(recipe)
	}
	json.NewEncoder(w).Encode(consumable)
}

func GetSingleResourceHandler(w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	ankamaId := r.Context().Value("ankamaId").(int)

	txn := Db.Txn(false)
	defer txn.Abort()

	raw, err := txn.First(fmt.Sprintf("%s-resources", utils.CurrentRedBlueVersionStr(Version.MemDb)), "id", ankamaId)
	if err != nil || raw == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	resource := RenderResource(raw.(*gen.MappedMultilangItem), lang)
	recipe, exists := GetRecipeIfExists(ankamaId, txn)
	if exists {
		resource.Recipe = RenderRecipe(recipe)
	}
	json.NewEncoder(w).Encode(resource)
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
