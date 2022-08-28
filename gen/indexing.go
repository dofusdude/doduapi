package gen

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/dofusdude/api/utils"

	"github.com/hashicorp/go-memdb"
	"github.com/meilisearch/meilisearch-go"
)

func IndexApiData(done chan bool, indexed *bool, version *utils.VersionT) (*memdb.MemDB, map[string]SearchIndexes) {
	var items []MappedMultilangItem
	var sets []MappedMultilangSet
	var recipes []MappedMultilangRecipe
	var mounts []MappedMultilangMount

	log.Println("generating Database and search index ...")
	// --
	file, err := os.ReadFile("data/MAPPED_ITEMS.json")
	if err != nil {
		fmt.Print(err)
	}

	err = json.Unmarshal(file, &items)
	if err != nil {
		fmt.Println(err)
	}

	// --
	file, err = os.ReadFile("data/MAPPED_SETS.json")
	if err != nil {
		fmt.Print(err)
	}

	err = json.Unmarshal(file, &sets)
	if err != nil {
		fmt.Println(err)
	}

	// --
	file, err = os.ReadFile("data/MAPPED_RECIPES.json")
	if err != nil {
		fmt.Print(err)
	}

	err = json.Unmarshal(file, &recipes)
	if err != nil {
		fmt.Println(err)
	}

	// --
	file, err = os.ReadFile("data/MAPPED_MOUNTS.json")
	if err != nil {
		fmt.Print(err)
	}

	err = json.Unmarshal(file, &mounts)
	if err != nil {
		fmt.Println(err)
	}

	startDatabaseIndex := time.Now()
	db, indexes := GenerateDatabase(&items, &sets, &recipes, &mounts, indexed, version, done)
	log.Println("... completed indexing in", time.Since(startDatabaseIndex))

	return db, indexes
}

func GetMemDBSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"red-equipment": &memdb.TableSchema{
				Name: "red-equipment",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"blue-equipment": &memdb.TableSchema{
				Name: "blue-equipment",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"red-resources": &memdb.TableSchema{
				Name: "red-resources",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"blue-resources": &memdb.TableSchema{
				Name: "blue-resources",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"red-consumables": &memdb.TableSchema{
				Name: "red-consumables",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"blue-consumables": &memdb.TableSchema{
				Name: "blue-consumables",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"red-quest_items": &memdb.TableSchema{
				Name: "red-quest_items",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"blue-quest_items": &memdb.TableSchema{
				Name: "blue-quest_items",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"red-cosmetics": &memdb.TableSchema{
				Name: "red-cosmetics",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"blue-cosmetics": &memdb.TableSchema{
				Name: "blue-cosmetics",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"red-sets": &memdb.TableSchema{
				Name: "red-sets",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"blue-sets": &memdb.TableSchema{
				Name: "blue-sets",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"red-all_items": &memdb.TableSchema{
				Name: "red-all_items",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"blue-all_items": &memdb.TableSchema{
				Name: "blue-all_items",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"red-recipes": &memdb.TableSchema{
				Name: "red-recipes",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "ResultId"},
					},
				},
			},
			"blue-recipes": &memdb.TableSchema{
				Name: "blue-recipes",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "ResultId"},
					},
				},
			},
			"red-mounts": &memdb.TableSchema{
				Name: "red-mounts",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"blue-mounts": &memdb.TableSchema{
				Name: "blue-mounts",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
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
	AllItems *meilisearch.Index
	Sets     *meilisearch.Index
	Mounts   *meilisearch.Index
}

func GenerateDatabase(items *[]MappedMultilangItem, sets *[]MappedMultilangSet, recipes *[]MappedMultilangRecipe, mounts *[]MappedMultilangMount, indexed *bool, version *utils.VersionT, done chan bool) (*memdb.MemDB, map[string]SearchIndexes) {
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
	*indexed = false

	client := utils.CreateMeiliClient()

	for _, lang := range utils.Languages {
		itemIndexUid := fmt.Sprintf("%s-all_items-%s", utils.NextRedBlueVersionStr(version.Search), lang)
		setIndexUid := fmt.Sprintf("%s-sets-%s", utils.NextRedBlueVersionStr(version.Search), lang)
		mountIndexUid := fmt.Sprintf("%s-mounts-%s", utils.NextRedBlueVersionStr(version.Search), lang)

		// creation
		createItemsIdxTask, err := client.CreateIndex(&meilisearch.IndexConfig{
			Uid:        itemIndexUid,
			PrimaryKey: "id",
		})
		if err != nil {
			log.Fatal(err)
		}
		_, err = client.WaitForTask(createItemsIdxTask.TaskUID)
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
		_, err = client.WaitForTask(createSetsIdxTask.TaskUID)
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
		_, err = client.WaitForTask(createItemsIdxTask.TaskUID)
		if err != nil {
			log.Fatal(err)
		}
		_, err = client.WaitForTask(createSetsIdxTask.TaskUID)
		if err != nil {
			log.Fatal(err)
		}
		_, err = client.WaitForTask(createMountIdxTask.TaskUID)
		if err != nil {
			log.Fatal(err)
		}

		// add filters
		allItemsIdx := client.Index(itemIndexUid)
		_, err = allItemsIdx.UpdateFilterableAttributes(&[]string{
			"super_type",
			"type_name",
		})
		if err != nil {
			log.Println(err)
			return nil, nil
		}

		mountsIdx := client.Index(mountIndexUid)
		_, err = mountsIdx.UpdateFilterableAttributes(&[]string{
			"family_name",
		})
		if err != nil {
			log.Println(err)
			return nil, nil
		}

		setsIdx := client.Index(setIndexUid)
		_, err = setsIdx.UpdateFilterableAttributes(&[]string{
			"highest_equipment_level",
		})
		if err != nil {
			log.Println(err)
			return nil, nil
		}

		multilangSearchIndexes[lang] = SearchIndexes{
			AllItems: allItemsIdx,
			Sets:     setsIdx,
			Mounts:   mountsIdx,
		}
	}

	schema := GetMemDBSchema()

	db, err := memdb.NewMemDB(schema)
	if err != nil {
		panic(err)
	}

	txn := db.Txn(true)

	langItems := make(map[string]map[int][]SearchIndexedItem)
	for _, lang := range utils.Languages {
		langItems[lang] = make(map[int][]SearchIndexedItem)
	}

	maxBatchSize := 250
	itemIndexBatch := make(map[string][]SearchIndexedItem)
	itemsTable := fmt.Sprintf("%s-all_items", utils.NextRedBlueVersionStr(version.MemDb))
	setsTable := fmt.Sprintf("%s-sets", utils.NextRedBlueVersionStr(version.MemDb))
	mountsTable := fmt.Sprintf("%s-mounts", utils.NextRedBlueVersionStr(version.MemDb))
	recipesTable := fmt.Sprintf("%s-recipes", utils.NextRedBlueVersionStr(version.MemDb))

	for _, recipe := range *recipes {
		recipeCt := recipe
		if err := txn.Insert(recipesTable, &recipeCt); err != nil {
			panic(err)
		}
	}

	for _, item := range *items {
		itemCp := item
		var insertCategoryTable string
		if itemCp.Type.CategoryId == 4 {
			continue
		}
		insertCategoryTable = utils.CategoryIdMapping(itemCp.Type.CategoryId)

		if err := txn.Insert(fmt.Sprintf("%s-%s", utils.NextRedBlueVersionStr(version.MemDb), insertCategoryTable), &itemCp); err != nil {
			panic(err)
		}

		if err := txn.Insert(itemsTable, &itemCp); err != nil {
			panic(err)
		}

		for _, lang := range utils.Languages {
			object := SearchIndexedItem{
				Name:        itemCp.Name[lang],
				Id:          itemCp.AnkamaId,
				Description: itemCp.Description[lang],
				SuperType:   insertCategoryTable,
				TypeName:    strings.ToLower(itemCp.Type.Name[lang]),
			}

			itemIndexBatch[lang] = append(itemIndexBatch[lang], object)
			if len(itemIndexBatch[lang]) >= maxBatchSize {
				taskInfo, err := multilangSearchIndexes[lang].AllItems.AddDocuments(itemIndexBatch[lang])
				if err != nil {
					log.Println(err)
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
			panic(err)
		}

		for _, lang := range utils.Languages {
			object := SearchIndexedSet{
				Name:  setCp.Name[lang],
				Id:    setCp.AnkamaId,
				Level: setCp.Level,
			}

			setIndexBatch[lang] = append(setIndexBatch[lang], object)
			if len(setIndexBatch[lang]) >= maxBatchSize {
				taskInfo, err := multilangSearchIndexes[lang].Sets.AddDocuments(setIndexBatch[lang])
				if err != nil {
					log.Println(err)
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
			panic(err)
		}

		for _, lang := range utils.Languages {
			object := SearchIndexedMount{
				Name:       mountCp.Name[lang],
				Id:         mountCp.AnkamaId,
				FamilyName: strings.ToLower(mountCp.FamilyName[lang]),
			}

			mountIndexBatch[lang] = append(mountIndexBatch[lang], object)
			if len(mountIndexBatch[lang]) >= maxBatchSize {
				taskInfo, err := multilangSearchIndexes[lang].Mounts.AddDocuments(mountIndexBatch[lang])
				if err != nil {
					log.Println(err)
				}
				indexTasks = append(indexTasks, taskInfo)
				mountIndexBatch[lang] = nil
			}
		}
	}

	txn.Commit()

	// add everything not indexed because still under max batch size
	for _, lang := range utils.Languages {
		if len(itemIndexBatch[lang]) > 0 {
			taskInfo, err := multilangSearchIndexes[lang].AllItems.AddDocuments(itemIndexBatch[lang])
			if err != nil {
				log.Println(err)
			}
			indexTasks = append(indexTasks, taskInfo)
		}

		if len(setIndexBatch[lang]) > 0 {
			taskInfo, err := multilangSearchIndexes[lang].Sets.AddDocuments(setIndexBatch[lang])
			if err != nil {
				log.Println(err)
			}
			indexTasks = append(indexTasks, taskInfo)
		}
		if len(mountIndexBatch[lang]) > 0 {
			taskInfo, err := multilangSearchIndexes[lang].Mounts.AddDocuments(mountIndexBatch[lang])
			if err != nil {
				log.Println(err)
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
				break
			case <-ticker.C:
				allTrue := true
				for i, task := range indexTasks {
					if !firstRun && awaited[i] { // save result from last run to save requests
						continue
					}

					waitingForSucceededOrFailed := true
					taskResp, err := client.GetTask(task.TaskUID)
					if err != nil {
						log.Println(err)
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
					*indexed = true
					ticker.Stop()

					log.Println("-- waiter: indexes done, Bye!")
					staySelectLoop = false
					break
				}
			}
		}
		//}(done, indexed, client, ticker)
	} else {
		close(done)
	}

	return db, multilangSearchIndexes
}
