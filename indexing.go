package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/hashicorp/go-memdb"
	"github.com/meilisearch/meilisearch-go"
	g "github.com/zyedidia/generic"
	"github.com/zyedidia/generic/set"

	mapping "github.com/dofusdude/dodumap"
)

type SearchStuffType struct {
	NameId string `json:"name_id"`
}

type SearchType struct {
	Name   string `json:"name"`    // old "type_name"
	NameId string `json:"name_id"` // old "type_id"
}

type SearchIndexedItem struct {
	Id          int             `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	SuperType   SearchStuffType `json:"super_type"`
	Type        SearchType      `json:"type"`
	Level       int             `json:"level"`
	StuffType   SearchStuffType `json:"stuff_type"`
}

type SearchIndexedMount struct {
	Id        int             `json:"id"`
	Name      string          `json:"name"`
	Family    ApiType         `json:"family"` // family_name before, now with id and translated name
	StuffType SearchStuffType `json:"stuff_type"`
}

type SearchIndexedSet struct {
	Id                    int             `json:"id"`
	Name                  string          `json:"name"`
	Level                 int             `json:"highest_equipment_level"`
	ContainsCosmetics     bool            `json:"contains_cosmetics"`
	ContainsCosmeticsOnly bool            `json:"contains_cosmetics_only"`
	StuffType             SearchStuffType `json:"stuff_type"`
}

type EffectConditionDbEntry struct {
	Id   int
	Name string
}

type ItemTypeId struct {
	Id     int
	EnName string
}

func IndexApiData(version *VersionT) (*memdb.MemDB, map[string]SearchIndexes) {
	var items []mapping.MappedMultilangItemUnity
	var sets []mapping.MappedMultilangSetUnity
	var recipes []mapping.MappedMultilangRecipe
	var mounts []mapping.MappedMultilangMount

	// --
	itemsResponse, err := http.Get(ReleaseUrl + "/MAPPED_ITEMS.json")
	if err != nil {
		log.Fatal(err)
	}

	itemsBody, err := io.ReadAll(itemsResponse.Body)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(itemsBody, &items)
	if err != nil {
		log.Fatal(err)
	}

	// --
	setsResponse, err := http.Get(ReleaseUrl + "/MAPPED_SETS.json")
	if err != nil {
		log.Fatal(err)
	}

	setsBody, err := io.ReadAll(setsResponse.Body)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(setsBody, &sets)
	if err != nil {
		log.Fatal(err)
	}

	// --
	recipesResponse, err := http.Get(ReleaseUrl + "/MAPPED_RECIPES.json")
	if err != nil {
		log.Fatal(err)
	}

	recipesBody, err := io.ReadAll(recipesResponse.Body)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(recipesBody, &recipes)
	if err != nil {
		log.Fatal(err)
	}

	// --
	mountsResponse, err := http.Get(ReleaseUrl + "/MAPPED_MOUNTS.json")
	if err != nil {
		log.Fatal(err)
	}

	mountsBody, err := io.ReadAll(mountsResponse.Body)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(mountsBody, &mounts)
	if err != nil {
		log.Fatal(err)
	}
	log.Debug("loaded", "mounts", len(mounts), "items", len(items), "sets", len(sets), "recipes", len(recipes))

	db, indexes := GenerateDatabase(&items, &sets, &recipes, &mounts, version)

	return db, indexes
}

func GetMemDBSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"red-equipment": {
				Name: "red-equipment",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"blue-equipment": {
				Name: "blue-equipment",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"red-resources": {
				Name: "red-resources",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"blue-resources": {
				Name: "blue-resources",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"red-consumables": {
				Name: "red-consumables",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"blue-consumables": {
				Name: "blue-consumables",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"red-quest_items": {
				Name: "red-quest_items",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"blue-quest_items": {
				Name: "blue-quest_items",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"red-cosmetics": {
				Name: "red-cosmetics",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"blue-cosmetics": {
				Name: "blue-cosmetics",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"red-sets": {
				Name: "red-sets",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"blue-sets": {
				Name: "blue-sets",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"red-all_items": {
				Name: "red-all_items",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"blue-all_items": {
				Name: "blue-all_items",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"red-recipes": {
				Name: "red-recipes",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "ResultId"},
					},
				},
			},
			"blue-recipes": {
				Name: "blue-recipes",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "ResultId"},
					},
				},
			},
			"red-mounts": {
				Name: "red-mounts",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"blue-mounts": {
				Name: "blue-mounts",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			// Maybe add red/blue staging here.
			"effect-condition-elements": {
				Name: "effect-condition-elements",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "Id"},
					},
				},
			},
			// Maybe add red/blue staging here.
			"item-type-ids": {
				Name: "item-type-ids",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "Id"},
					},
				},
			},
		},
	}
}

func GetItemSuperType(id int) int {
	switch id {
	case 1:
		return 66230 // Amulett
		//case 2: return 66231 // Ring
		//case 3: return 66239 // Waffe
		//case 4: return 66240 // Zweihandwaffe

	}
	return 0
}

type SearchIndexes struct {
	AllItems meilisearch.IndexManager
	Sets     meilisearch.IndexManager
	Mounts   meilisearch.IndexManager
}

type AlmanaxBonusListing struct {
	Id   string `json:"id"`   // english-id
	Name string `json:"name"` // translated text
}

type AlmanaxBonusListingMeili struct {
	Id   string `json:"id"`   // meili specific id without utf8 guarantees
	Slug string `json:"slug"` // english-id
	Name string `json:"name"` // translated text
}

func UpdateAlmanaxBonusIndex(init bool) int {
	client := meilisearch.New(MeiliHost, meilisearch.WithAPIKey(MeiliKey))
	defer client.Close()

	added := 0

	for _, lang := range Languages {
		if lang == "pt" {
			continue // no portuguese almanax bonuses
		}
		url := fmt.Sprintf("https://api.dofusdu.de/dofus2/meta/%s/almanax/bonuses", lang)
		resp, err := http.Get(url)
		if err != nil {
			log.Warn(err, "lang", lang)
			return added
		}

		var bonuses []AlmanaxBonusListing
		err = json.NewDecoder(resp.Body).Decode(&bonuses)
		if err != nil {
			log.Error(err, "lang", lang)
			return added
		}

		var bonusesMeili []AlmanaxBonusListingMeili
		var counter int = 0
		for i := range bonuses {
			bonusesMeili = append(bonusesMeili, AlmanaxBonusListingMeili{
				Id:   strconv.Itoa(counter),
				Slug: bonuses[i].Id,
				Name: bonuses[i].Name,
			})
			counter++
		}

		indexName := fmt.Sprintf("alm-bonuses-%s", lang)
		_, err = client.GetIndex(indexName)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				log.Info("alm bonuses index does not exist yet, creating now", "index", indexName)
				almTaskInfo, err := client.CreateIndex(&meilisearch.IndexConfig{
					Uid:        indexName,
					PrimaryKey: "id",
				})

				if err != nil {
					log.Error("Error while creating alm bonus index in meili", "err", err)
					return added
				}

				task, err := client.WaitForTask(almTaskInfo.TaskUID, 500*time.Millisecond)
				if err != nil {
					log.Error("Error while waiting alm bonus index creation at meili", "err", err)
					return added
				}

				if task.Status == "failed" && !strings.Contains(task.Error.Message, "already exists") {
					log.Error("alm bonuses creation failed.", "err", task.Error)
					return added
				}

			} else {
				log.Error("Error while getting alm bonus index in meili", "err", err)
				return added
			}
		}

		almBonusIndex := client.Index(indexName)

		if init { // clean index, add all
			cleanTask, err := almBonusIndex.DeleteAllDocuments()
			if err != nil {
				log.Error("Error while cleaning alm bonuses in meili.", "err", err)
				return added
			}

			task, err := client.WaitForTask(cleanTask.TaskUID, 100*time.Millisecond)
			if err != nil {
				log.Error("Error while waiting for meili to clean alm bonuses.", "err", err)
				return added
			}

			if task.Status == "failed" {
				log.Error("clean alm bonuses task failed.", "err", task.Error)
				return added
			}

			var documentsAddTask *meilisearch.TaskInfo
			if documentsAddTask, err = almBonusIndex.AddDocuments(bonusesMeili); err != nil {
				log.Error("Error while adding alm bonuses to meili.", "err", err)
				return added
			}

			task, err = client.WaitForTask(documentsAddTask.TaskUID, 500*time.Millisecond)
			if err != nil {
				log.Error("Error while waiting for meili to add alm bonuses.", "err", err)
				return added
			}

			if task.Status == "failed" {
				log.Error("alm bonuses add docs task failed.", "err", task.Error)
				return added
			}

			added += len(bonuses)
		} else { // search the item exact matches before adding it
			for _, bonus := range bonusesMeili {
				request := &meilisearch.SearchRequest{
					Limit: 1,
				}

				var searchResp *meilisearch.SearchResponse
				if searchResp, err = almBonusIndex.Search(bonus.Name, request); err != nil {
					log.Error("SearchAlmanaxBonuses: index not found: ", "err", err)
					return added
				}

				foundIdentical := false
				if len(searchResp.Hits) > 0 {
					var item = searchResp.Hits[0].(map[string]interface{})
					if item["name"] == bonus.Name {
						foundIdentical = true
					}
				}

				if !foundIdentical { // add only if not found
					log.Info("adding", "bonus", bonus.Name, "bonus", bonus, "lang", lang, "hits", searchResp.Hits)
					documentsAddTask, err := almBonusIndex.AddDocuments([]AlmanaxBonusListingMeili{bonus})
					if err != nil {
						log.Error("Error while adding alm bonuses to meili.", "err", err)
						return added
					}

					task, err := client.WaitForTask(documentsAddTask.TaskUID, 500*time.Millisecond)
					if err != nil {
						log.Error("Error while waiting for meili to add alm bonuses.", "err", err)
						return added
					}

					if task.Status == "failed" {
						log.Error("alm bonuses adding failed.", "err", task.Error)
						return added
					}

					added += 1
				}
			}
		}
	}

	return added
}

func GenerateDatabase(items *[]mapping.MappedMultilangItemUnity, sets *[]mapping.MappedMultilangSetUnity, recipes *[]mapping.MappedMultilangRecipe, mounts *[]mapping.MappedMultilangMount, version *VersionT) (*memdb.MemDB, map[string]SearchIndexes) {
	/*
		item_category_mapping := hashbidimap.New()
		item_category_Put(0, 862817) // Ausrüstung
		item_category_Put(1, 748369) // Komsumgüter
		item_category_Put(2, 67146)  // Ressourcen
		item_category_Put(3, 67303)  // Questgegenstände
		item_category_Put(4, 67303)  // Questgegenstände -- skipped because internal (hidden without translations)
		item_category_Put(5, 764933) // Ausschmückungen
	*/

	multilangSearchIndexes := make(map[string]SearchIndexes)
	var indexTasks []*meilisearch.TaskInfo

	client := meilisearch.New(MeiliHost, meilisearch.WithAPIKey(MeiliKey))
	defer client.Close()

	// generate all indexes with %version-%lang
	//meiliPullInterval := 100 * time.Millisecond
	updateTasks := make([]*meilisearch.TaskInfo, 0)

	for _, lang := range Languages {
		itemIndexUid := fmt.Sprintf("%s-all_items-%s", NextRedBlueVersionStr(version.Search), lang)
		setIndexUid := fmt.Sprintf("%s-sets-%s", NextRedBlueVersionStr(version.Search), lang)
		mountIndexUid := fmt.Sprintf("%s-mounts-%s", NextRedBlueVersionStr(version.Search), lang)

		createClearIndices([]string{
			itemIndexUid,
			setIndexUid,
			mountIndexUid,
		}, client)

		// add filters and searchable attributes
		// -- all items --
		allItemsIdx := client.Index(itemIndexUid)
		allItemsFilterTask, err := allItemsIdx.UpdateFilterableAttributes(&[]string{
			"super_type.name_id",
			"type.name_id",
			"level",
		})
		if err != nil {
			log.Fatal(err)
		}
		updateTasks = append(updateTasks, allItemsFilterTask)

		allItemsSearchableTask, err := allItemsIdx.UpdateSearchableAttributes(&[]string{
			"name",
			"type.name",
			"description",
		})
		if err != nil {
			log.Fatal(err)
		}
		updateTasks = append(updateTasks, allItemsSearchableTask)

		// -- mounts --
		mountsIdx := client.Index(mountIndexUid)
		mountFilterTask, err := mountsIdx.UpdateFilterableAttributes(&[]string{
			"family.name",
			"family.id",
		})
		if err != nil {
			log.Fatal(err)
		}
		updateTasks = append(updateTasks, mountFilterTask)

		mountSearchableTask, err := mountsIdx.UpdateSearchableAttributes(&[]string{
			"name",
			"family.name",
		})
		if err != nil {
			log.Fatal(err)
		}
		updateTasks = append(updateTasks, mountSearchableTask)

		// -- sets --
		setsIdx := client.Index(setIndexUid)
		setFilterUpdateTask, err := setsIdx.UpdateFilterableAttributes(&[]string{
			"highest_equipment_level",
			"constains_cosmetics",
			"constains_cosmetics_only",
		})
		if err != nil {
			log.Fatal(err)
		}
		updateTasks = append(updateTasks, setFilterUpdateTask)

		setSearchableTask, err := setsIdx.UpdateSearchableAttributes(&[]string{
			"name",
		})
		if err != nil {
			log.Fatal(err)
		}
		updateTasks = append(updateTasks, setSearchableTask)

		multilangSearchIndexes[lang] = SearchIndexes{
			AllItems: allItemsIdx,
			Sets:     setsIdx,
			Mounts:   mountsIdx,
		}
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func(tasks []*meilisearch.TaskInfo, client meilisearch.ServiceManager) {
		defer wg.Done()
		waitForTasks(tasks, client, false)
	}(updateTasks, client)
	log.Info("waiting for all indexes to be updated")
	wg.Wait()

	// create in-memory db
	schema := GetMemDBSchema()

	var err error
	var db *memdb.MemDB
	if db, err = memdb.NewMemDB(schema); err != nil {
		log.Fatal(err)
	}

	txn := db.Txn(true)

	// persistent elements are also in db. TODO does this update automatically?
	persIt := PersistedElements.Entries.Iterator()
	for persIt.Next() {
		if err = txn.Insert("effect-condition-elements", &EffectConditionDbEntry{
			Id:   persIt.Key().(int),
			Name: persIt.Value().(string),
		}); err != nil {
			log.Fatal(err)
		}
	}

	// db prepare insertions
	maxBatchSize := 250
	itemIndexBatch := make(map[string][]SearchIndexedItem)
	itemsTable := fmt.Sprintf("%s-all_items", NextRedBlueVersionStr(version.MemDb))
	setsTable := fmt.Sprintf("%s-sets", NextRedBlueVersionStr(version.MemDb))
	mountsTable := fmt.Sprintf("%s-mounts", NextRedBlueVersionStr(version.MemDb))
	recipesTable := fmt.Sprintf("%s-recipes", NextRedBlueVersionStr(version.MemDb))

	for _, recipe := range *recipes {
		recipeCt := recipe
		if err = txn.Insert(recipesTable, &recipeCt); err != nil {
			log.Fatal(err)
		}
	}

	itemTypeIds := set.NewHashset[string](10, g.Equals[string], g.HashString)

	// all items search
	for _, item := range *items {
		itemCp := item
		var insertCategoryTable string
		if itemCp.Type.CategoryId == 4 {
			continue
		}
		insertCategoryTable = CategoryIdMapping(itemCp.Type.CategoryId)

		if err = txn.Insert(fmt.Sprintf("%s-%s", NextRedBlueVersionStr(version.MemDb), insertCategoryTable), &itemCp); err != nil {
			log.Fatal(err)
		}

		if err = txn.Insert(itemsTable, &itemCp); err != nil {
			log.Fatal(err)
		}

		for _, lang := range Languages {
			enTypeId := strings.ToLower(strings.ReplaceAll(itemCp.Type.Name["en"], " ", "-"))
			object := SearchIndexedItem{
				Name:        itemCp.Name[lang],
				Id:          itemCp.AnkamaId,
				Description: itemCp.Description[lang],
				SuperType: SearchStuffType{
					NameId: insertCategoryTable,
				},
				Type: SearchType{
					Name:   strings.ToLower(itemCp.Type.Name[lang]),
					NameId: enTypeId,
				},
				Level: itemCp.Level,
				StuffType: SearchStuffType{
					NameId: fmt.Sprintf("items-%s", insertCategoryTable),
				},
			}

			itemTypeIds.Put(enTypeId)

			itemIndexBatch[lang] = append(itemIndexBatch[lang], object)
			if len(itemIndexBatch[lang]) >= maxBatchSize {
				var taskInfo *meilisearch.TaskInfo
				if taskInfo, err = multilangSearchIndexes[lang].AllItems.AddDocuments(itemIndexBatch[lang]); err != nil {
					log.Fatal(err)
				}
				indexTasks = append(indexTasks, taskInfo)
				itemIndexBatch[lang] = make([]SearchIndexedItem, 0)
			}
		}
	}

	// leftover items
	for _, lang := range Languages {
		if len(itemIndexBatch[lang]) > 0 {
			var taskInfo *meilisearch.TaskInfo
			if taskInfo, err = multilangSearchIndexes[lang].AllItems.AddDocuments(itemIndexBatch[lang]); err != nil {
				log.Fatal(err)
			}
			indexTasks = append(indexTasks, taskInfo)
			itemIndexBatch[lang] = make([]SearchIndexedItem, 0)
		}
	}

	for id, itemTypeId := range itemTypeIds.Keys() {
		if err = txn.Insert("item-type-ids", &ItemTypeId{
			Id:     id,
			EnName: itemTypeId,
		}); err != nil {
			log.Fatal(err)
		}
	}

	// sets
	setIndexBatch := make(map[string][]SearchIndexedSet)
	for _, set := range *sets {
		setCp := set
		if err := txn.Insert(setsTable, &setCp); err != nil {
			log.Fatal(err)
		}

		for _, lang := range Languages {
			object := SearchIndexedSet{
				Name:                  setCp.Name[lang],
				Id:                    setCp.AnkamaId,
				Level:                 setCp.Level,
				ContainsCosmetics:     setCp.ContainsCosmetics,
				ContainsCosmeticsOnly: setCp.ContainsCosmeticsOnly,
				StuffType: SearchStuffType{
					NameId: "sets",
				},
			}

			setIndexBatch[lang] = append(setIndexBatch[lang], object)
			if len(setIndexBatch[lang]) >= maxBatchSize {
				taskInfo, err := multilangSearchIndexes[lang].Sets.AddDocuments(setIndexBatch[lang])
				if err != nil {
					log.Fatal(err)
				}
				indexTasks = append(indexTasks, taskInfo)
				setIndexBatch[lang] = nil
			}
		}
	}

	// leftover sets
	for _, lang := range Languages {
		if len(setIndexBatch[lang]) > 0 {
			var taskInfo *meilisearch.TaskInfo
			if taskInfo, err = multilangSearchIndexes[lang].AllItems.AddDocuments(setIndexBatch[lang]); err != nil {
				log.Fatal(err)
			}
			indexTasks = append(indexTasks, taskInfo)
			setIndexBatch[lang] = make([]SearchIndexedSet, 0)
		}
	}

	// mounts
	mountIndexBatch := make(map[string][]SearchIndexedMount)
	for _, mount := range *mounts {
		mountCp := mount
		if err := txn.Insert(mountsTable, &mountCp); err != nil {
			log.Fatal(err)
		}

		for _, lang := range Languages {
			object := SearchIndexedMount{
				Name: mountCp.Name[lang],
				Id:   mountCp.AnkamaId,
				Family: ApiType{
					Name: strings.ToLower(mountCp.FamilyName[lang]),
					Id:   mountCp.FamilyId,
				},
				StuffType: SearchStuffType{
					NameId: "mounts",
				},
			}

			mountIndexBatch[lang] = append(mountIndexBatch[lang], object)
			if len(mountIndexBatch[lang]) >= maxBatchSize {
				taskInfo, err := multilangSearchIndexes[lang].Mounts.AddDocuments(mountIndexBatch[lang])
				if err != nil {
					log.Fatal(err)
				}
				indexTasks = append(indexTasks, taskInfo)
				mountIndexBatch[lang] = nil
			}
		}
	}

	// leftover mounts
	for _, lang := range Languages {
		if len(mountIndexBatch[lang]) > 0 {
			var taskInfo *meilisearch.TaskInfo
			if taskInfo, err = multilangSearchIndexes[lang].AllItems.AddDocuments(mountIndexBatch[lang]); err != nil {
				log.Fatal(err)
			}
			indexTasks = append(indexTasks, taskInfo)
			mountIndexBatch[lang] = make([]SearchIndexedMount, 0)
		}
	}

	txn.Commit()

	// wait for all indexing tasks to finish
	wg.Add(1)
	go func(tasks []*meilisearch.TaskInfo, client meilisearch.ServiceManager) {
		defer wg.Done()
		waitForTasks(tasks, client, false)
	}(indexTasks, client)
	log.Info("waiting for all documents to be indexed")
	wg.Wait()

	return db, multilangSearchIndexes
}

func createClearIndices(indexNames []string, client meilisearch.ServiceManager) {
	for _, indexName := range indexNames {
		index, err := client.GetIndex(indexName)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				log.Info("index does not exist yet, creating now", "index", indexName)
				taskInfo, err := client.CreateIndex(&meilisearch.IndexConfig{
					Uid:        indexName,
					PrimaryKey: "id",
				})
				if err != nil {
					log.Error("Error while creating index in meili", "err", err)
					return
				}

				task, err := client.WaitForTask(taskInfo.TaskUID, 100*time.Millisecond)
				if err != nil {
					log.Error("Error while waiting index creation at meili", "err", err)
					return
				}

				if task.Status != meilisearch.TaskStatusSucceeded {
					log.Error("Meili", "status", task.Status, "message", task.Error.Message)
				}
			} else {
				log.Error("Error while getting index in meili", "err", err)
				return
			}
		} else { // clear index and start over
			log.Info("index exists, clearing", "index", indexName)
			delTask, err := index.DeleteAllDocuments()
			task, err := client.WaitForTask(delTask.TaskUID, 100*time.Millisecond)
			if err != nil {
				log.Error("Error while waiting index creation at meili", "err", err)
				return
			}

			if task.Status != meilisearch.TaskStatusSucceeded {
				log.Error("Meili", "status", task.Status, "message", task.Error.Message)
			}
		}
	}
}

func waitForTasks(tasks []*meilisearch.TaskInfo, client meilisearch.ServiceManager, ignoreExists bool) {
	if len(tasks) == 0 {
		return
	}
	wg := sync.WaitGroup{}
	semap := make(chan struct{}, runtime.NumCPU()*2)
	for _, task := range tasks {
		wg.Add(1)
		go func(taskInfo *meilisearch.TaskInfo, client meilisearch.ServiceManager) {
			defer wg.Done()

			semap <- struct{}{}
			defer func() {
				<-semap
			}()

			task, err := client.WaitForTask(taskInfo.TaskUID, 100*time.Millisecond)
			if err != nil {
				log.Fatal(err)
			}

			if ignoreExists && task.Status == meilisearch.TaskStatusFailed && !strings.Contains(task.Error.Message, "already exists") {
				return
			}

			if task.Status != meilisearch.TaskStatusSucceeded {
				log.Error("Meili", "status", task.Status, "message", task.Error.Message)
			}
		}(task, client)
	}
	wg.Wait()
}
