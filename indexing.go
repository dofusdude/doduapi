package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/hashicorp/go-memdb"
	"github.com/meilisearch/meilisearch-go"

	mapping "github.com/dofusdude/dodumap"
)

type SearchIndexedItem struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SuperType   string `json:"super_type"`
	TypeName    string `json:"type_name"`
	Level       int    `json:"level"`
	StuffType   string `json:"stuff_type"`
}

type SearchIndexedMount struct {
	Id         int    `json:"id"`
	Name       string `json:"name"`
	FamilyName string `json:"family_name"`
	StuffType  string `json:"stuff_type"`
}

type SearchIndexedSet struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	Level     int    `json:"highest_equipment_level"`
	StuffType string `json:"stuff_type"`
}

type EffectConditionDbEntry struct {
	Id   int
	Name string
}

func IndexApiData(done chan bool, version *VersionT) (*memdb.MemDB, map[string]SearchIndexes) {
	var items []mapping.MappedMultilangItem
	var sets []mapping.MappedMultilangSet
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
	log.Debug("loaded ", len(items), " items")

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
	log.Debug("loaded ", len(sets), " sets")

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
	log.Debug("loaded ", len(recipes), " recipes")

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
	log.Debug("loaded ", len(mounts), " mounts")

	db, indexes := GenerateDatabase(&items, &sets, &recipes, &mounts, version, done)

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
	AllStuff *meilisearch.Index
	AllItems *meilisearch.Index
	Sets     *meilisearch.Index
	Mounts   *meilisearch.Index
}

func GenerateDatabase(items *[]mapping.MappedMultilangItem, sets *[]mapping.MappedMultilangSet, recipes *[]mapping.MappedMultilangRecipe, mounts *[]mapping.MappedMultilangMount, version *VersionT, done chan bool) (*memdb.MemDB, map[string]SearchIndexes) {
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

	client := CreateMeiliClient()

	for _, lang := range Languages {
		allIndexUid := fmt.Sprintf("%s-all_stuff-%s", NextRedBlueVersionStr(version.Search), lang)
		itemIndexUid := fmt.Sprintf("%s-all_items-%s", NextRedBlueVersionStr(version.Search), lang)
		setIndexUid := fmt.Sprintf("%s-sets-%s", NextRedBlueVersionStr(version.Search), lang)
		mountIndexUid := fmt.Sprintf("%s-mounts-%s", NextRedBlueVersionStr(version.Search), lang)

		// creation
		createAllIdxTask, err := client.CreateIndex(&meilisearch.IndexConfig{
			Uid:        allIndexUid,
			PrimaryKey: "id",
		})
		if err != nil {
			log.Fatal(err)
		}

		createItemsIdxTask, err := client.CreateIndex(&meilisearch.IndexConfig{
			Uid:        itemIndexUid,
			PrimaryKey: "id",
		})
		if err != nil {
			log.Fatal(err)
		}

		createSetsIdxTask, err := client.CreateIndex(&meilisearch.IndexConfig{
			Uid:        setIndexUid,
			PrimaryKey: "id",
		})
		if err != nil {
			log.Fatal(err)
		}

		createMountIdxTask, err := client.CreateIndex(&meilisearch.IndexConfig{
			Uid:        mountIndexUid,
			PrimaryKey: "id",
		})
		if err != nil {
			log.Fatal(err)
		}

		// wait for creation end
		if _, err = client.WaitForTask(createAllIdxTask.TaskUID); err != nil {
			log.Fatal(err)
		}

		if _, err = client.WaitForTask(createItemsIdxTask.TaskUID); err != nil {
			log.Fatal(err)
		}

		if _, err = client.WaitForTask(createSetsIdxTask.TaskUID); err != nil {
			log.Fatal(err)
		}

		if _, err = client.WaitForTask(createMountIdxTask.TaskUID); err != nil {
			log.Fatal(err)
		}

		// add filters
		allStuffIdx := client.Index(allIndexUid)
		if _, err = allStuffIdx.UpdateFilterableAttributes(&[]string{
			"stuff_type",
		}); err != nil {
			log.Fatal(err)
		}

		allItemsIdx := client.Index(itemIndexUid)
		if _, err = allItemsIdx.UpdateFilterableAttributes(&[]string{
			"super_type",
			"type_name",
			"level",
		}); err != nil {
			log.Fatal(err)
		}

		mountsIdx := client.Index(mountIndexUid)
		if _, err = mountsIdx.UpdateFilterableAttributes(&[]string{
			"family_name",
		}); err != nil {
			log.Fatal(err)
		}

		setsIdx := client.Index(setIndexUid)
		if _, err = setsIdx.UpdateFilterableAttributes(&[]string{
			"highest_equipment_level",
		}); err != nil {
			log.Fatal(err)
		}

		multilangSearchIndexes[lang] = SearchIndexes{
			AllStuff: allStuffIdx,
			AllItems: allItemsIdx,
			Sets:     setsIdx,
			Mounts:   mountsIdx,
		}
	}

	schema := GetMemDBSchema()

	var err error
	var db *memdb.MemDB
	if db, err = memdb.NewMemDB(schema); err != nil {
		log.Fatal(err)
	}

	txn := db.Txn(true)

	persIt := PersistedElements.Entries.Iterator()
	for persIt.Next() {
		if err = txn.Insert("effect-condition-elements", &EffectConditionDbEntry{
			Id:   persIt.Key().(int),
			Name: persIt.Value().(string),
		}); err != nil {
			log.Fatal(err)
		}
	}

	langItems := make(map[string]map[int][]SearchIndexedItem)
	for _, lang := range Languages {
		langItems[lang] = make(map[int][]SearchIndexedItem)
	}

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
			object := SearchIndexedItem{
				Name:        itemCp.Name[lang],
				Id:          itemCp.AnkamaId,
				Description: itemCp.Description[lang],
				SuperType:   insertCategoryTable,
				TypeName:    strings.ToLower(itemCp.Type.Name[lang]),
				Level:       itemCp.Level,
				StuffType:   fmt.Sprintf("items-%s", insertCategoryTable),
			}

			itemIndexBatch[lang] = append(itemIndexBatch[lang], object)
			if len(itemIndexBatch[lang]) >= maxBatchSize {
				var taskInfo *meilisearch.TaskInfo
				if taskInfo, err = multilangSearchIndexes[lang].AllItems.AddDocuments(itemIndexBatch[lang]); err != nil {
					log.Fatal(err)
				}
				indexTasks = append(indexTasks, taskInfo)
				if taskInfo, err = multilangSearchIndexes[lang].AllStuff.AddDocuments(itemIndexBatch[lang]); err != nil {
					log.Fatal(err)
				}
				indexTasks = append(indexTasks, taskInfo)
				itemIndexBatch[lang] = nil
			}
		}
	}

	setIndexBatch := make(map[string][]SearchIndexedSet)
	for _, set := range *sets {
		setCp := set
		if err := txn.Insert(setsTable, &setCp); err != nil {
			log.Fatal(err)
		}

		for _, lang := range Languages {
			object := SearchIndexedSet{
				Name:      setCp.Name[lang],
				Id:        setCp.AnkamaId,
				Level:     setCp.Level,
				StuffType: "sets",
			}

			setIndexBatch[lang] = append(setIndexBatch[lang], object)
			if len(setIndexBatch[lang]) >= maxBatchSize {
				taskInfo, err := multilangSearchIndexes[lang].Sets.AddDocuments(setIndexBatch[lang])
				if err != nil {
					log.Warn(err)
				}
				indexTasks = append(indexTasks, taskInfo)
				taskInfo, err = multilangSearchIndexes[lang].AllStuff.AddDocuments(setIndexBatch[lang])
				if err != nil {
					log.Warn(err)
				}
				indexTasks = append(indexTasks, taskInfo)
				setIndexBatch[lang] = nil
			}
		}
	}

	mountIndexBatch := make(map[string][]SearchIndexedMount)
	for _, mount := range *mounts {
		mountCp := mount
		if err := txn.Insert(mountsTable, &mountCp); err != nil {
			log.Fatal(err)
		}

		for _, lang := range Languages {
			object := SearchIndexedMount{
				Name:       mountCp.Name[lang],
				Id:         mountCp.AnkamaId,
				FamilyName: strings.ToLower(mountCp.FamilyName[lang]),
				StuffType:  "mounts",
			}

			mountIndexBatch[lang] = append(mountIndexBatch[lang], object)
			if len(mountIndexBatch[lang]) >= maxBatchSize {
				taskInfo, err := multilangSearchIndexes[lang].Mounts.AddDocuments(mountIndexBatch[lang])
				if err != nil {
					log.Warn(err)
				}
				indexTasks = append(indexTasks, taskInfo)
				taskInfo, err = multilangSearchIndexes[lang].AllStuff.AddDocuments(mountIndexBatch[lang])
				if err != nil {
					log.Warn(err)
				}
				indexTasks = append(indexTasks, taskInfo)
				mountIndexBatch[lang] = nil
			}
		}
	}

	txn.Commit()

	// add everything not indexed because still under max batch size
	for _, lang := range Languages {
		if len(itemIndexBatch[lang]) > 0 {
			taskInfo, err := multilangSearchIndexes[lang].AllItems.AddDocuments(itemIndexBatch[lang])
			if err != nil {
				log.Warn(err)
			}
			indexTasks = append(indexTasks, taskInfo)
		}

		if len(setIndexBatch[lang]) > 0 {
			taskInfo, err := multilangSearchIndexes[lang].Sets.AddDocuments(setIndexBatch[lang])
			if err != nil {
				log.Warn(err)
			}
			indexTasks = append(indexTasks, taskInfo)
		}
		if len(mountIndexBatch[lang]) > 0 {
			taskInfo, err := multilangSearchIndexes[lang].Mounts.AddDocuments(mountIndexBatch[lang])
			if err != nil {
				log.Warn(err)
			}
			indexTasks = append(indexTasks, taskInfo)
		}
	}

	// wait for all indexing tasks to finish in the background
	if len(indexTasks) > 0 {
		ticker := time.NewTicker(3 * time.Second)
		//go func(done chan bool, indexed *bool, client *meilisearch.Client, ticker *time.Ticker) {
		var awaited []bool
		firstRun := true
		staySelectLoop := true
		for staySelectLoop {
			select {
			case <-done:
				ticker.Stop()
			case <-ticker.C:
				allTrue := true
				for i, task := range indexTasks {
					if !firstRun && awaited[i] { // save result from last run to save requests
						continue
					}

					waitingForSucceededOrFailed := true
					taskResp, err := client.GetTask(task.TaskUID)
					if err != nil {
						log.Error(err)
						break
					}
					waitingForSucceededOrFailed = taskResp.Status == meilisearch.TaskStatusSucceeded || taskResp.Status == meilisearch.TaskStatusFailed
					if !waitingForSucceededOrFailed {
						allTrue = false
					}
					if firstRun {
						awaited = append(awaited, waitingForSucceededOrFailed)
					} else {
						awaited[i] = waitingForSucceededOrFailed
					}
				}
				firstRun = false

				if allTrue {
					ticker.Stop()
					staySelectLoop = false
				}
			}
		}
		//}(done, indexed, client, ticker)
	} else {
		close(done)
	}

	return db, multilangSearchIndexes
}
