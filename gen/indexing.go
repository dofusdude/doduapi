package gen

import (
	"encoding/json"
	"fmt"
	"github.com/dofusdude/api/update"
	"github.com/dofusdude/api/utils"
	"log"
	"os"
	"path"
	"sync"
	"time"

	"github.com/hashicorp/go-memdb"
	"github.com/meilisearch/meilisearch-go"
)

func IndexApiData(done chan bool, indexed *bool) (*memdb.MemDB, map[string]SearchIndexes) {
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
	db, indexes := GenerateDatabase(&items, &sets, &recipes, &mounts, indexed, done)
	log.Println("... completed indexing in", time.Since(startDatabaseIndex))

	return db, indexes
}

func GetMemDBSchema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"equipment": &memdb.TableSchema{
				Name: "equipment",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"resources": &memdb.TableSchema{
				Name: "resources",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"consumables": &memdb.TableSchema{
				Name: "consumables",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"quest_items": &memdb.TableSchema{
				Name: "quest_items",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"cosmetics": &memdb.TableSchema{
				Name: "cosmetics",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"sets": &memdb.TableSchema{
				Name: "sets",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"all_items": &memdb.TableSchema{
				Name: "all_items",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "AnkamaId"},
					},
				},
			},
			"recipes": &memdb.TableSchema{
				Name: "recipes",
				Indexes: map[string]*memdb.IndexSchema{
					"id": &memdb.IndexSchema{
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.IntFieldIndex{Field: "ResultId"},
					},
				},
			},
			"mounts": &memdb.TableSchema{
				Name: "mounts",
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

func DownloadImages(items *[]MappedMultilangItem) {
	for _, item := range *items {
		filename := path.Base(item.Image)
		err := update.DownloadFile(fmt.Sprintf("data/img/%s", filename), item.Image)
		if err != nil {
			log.Fatal(err)
		}
	}
	time.Sleep(time.Millisecond * 200)
}

func DownloadImagesAsync(items *[]MappedMultilangItem) {
	wg := sync.WaitGroup{}
	wg.Add(len(*items))
	for _, item := range *items {
		go func(item *MappedMultilangItem) {
			defer wg.Done()
			filename := path.Base(item.Image)
			err := update.DownloadFile(fmt.Sprintf("data/img/%s", filename), item.Image)
			if err != nil {
				log.Fatal(err)
			}
		}(&item)
	}
	wg.Wait()
}

func CategoryIdMapping(id int) string {
	switch id {
	case 0:
		return "equipment"
	case 1:
		return "consumables"
	case 2:
		return "resources"
	case 3:
		return "quest_items"
	case 5:
		return "cosmetics"
	}
	return ""
}

type SearchIndexes struct {
	AllItems *meilisearch.Index

	Sets   *meilisearch.Index
	Mounts *meilisearch.Index
}

func GenerateDatabase(items *[]MappedMultilangItem, sets *[]MappedMultilangSet, recipes *[]MappedMultilangRecipe, mounts *[]MappedMultilangMount, indexed *bool, done chan bool) (*memdb.MemDB, map[string]SearchIndexes) {
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

		// delete old indexes
		itemIndexUid := fmt.Sprintf("all_items-%s", lang)
		itemDeleteTask, err := client.DeleteIndex(itemIndexUid)

		// wait for deletion
		_, err = client.WaitForTask(itemDeleteTask.TaskUID)
		if err != nil {
			log.Fatal(err)
		}

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

		setIndexUid := fmt.Sprintf("sets-%s", lang)
		setDeletionTask, err := client.DeleteIndex(setIndexUid)

		// wait for deletion
		_, err = client.WaitForTask(setDeletionTask.TaskUID)
		if err != nil {
			log.Fatal(err)
		}

		createSetsIdxTask, err := client.CreateIndex(&meilisearch.IndexConfig{
			Uid:        itemIndexUid,
			PrimaryKey: "id",
		})
		if err != nil {
			log.Fatal(err)
		}
		_, err = client.WaitForTask(createSetsIdxTask.TaskUID)
		if err != nil {
			log.Fatal(err)
		}

		mountIndexUid := fmt.Sprintf("mounts-%s", lang)
		mountDeletionTask, err := client.DeleteIndex(mountIndexUid)

		// wait for deletion
		_, err = client.WaitForTask(mountDeletionTask.TaskUID)
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
		_, err = client.WaitForTask(createSetsIdxTask.TaskUID)
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
		allItemsIdx.UpdateFilterableAttributes(&[]string{
			"type",
		})

		mountsIdx := client.Index(mountIndexUid)
		mountsIdx.UpdateFilterableAttributes(&[]string{
			"family_name",
		})

		setsIdx := client.Index(setIndexUid)

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

	maxBatchSize := 100
	itemIndexBatch := make(map[string][]SearchIndexedItem)
	for _, item := range *items {
		itemCp := item
		var insertCategoryTable string
		if itemCp.Type.CategoryId == 4 {
			continue
		}
		insertCategoryTable = CategoryIdMapping(itemCp.Type.CategoryId)

		if err := txn.Insert(insertCategoryTable, &itemCp); err != nil {
			panic(err)
		}

		if err := txn.Insert("all_items", &itemCp); err != nil {
			panic(err)
		}

		for _, lang := range utils.Languages {
			object := SearchIndexedItem{
				Name:        itemCp.Name[lang],
				Id:          itemCp.AnkamaId,
				Description: itemCp.Description[lang],
				Type:        insertCategoryTable,
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
		if err := txn.Insert("sets", &setCp); err != nil {
			panic(err)
		}

		for _, lang := range utils.Languages {
			object := SearchIndexedSet{
				Name: setCp.Name[lang],
				Id:   setCp.AnkamaId,
			}

			setIndexBatch[lang] = append(setIndexBatch[lang], object)
			if len(setIndexBatch[lang]) >= maxBatchSize {
				task_info, err := multilangSearchIndexes[lang].Sets.AddDocuments(setIndexBatch[lang])
				if err != nil {
					log.Println(err)
				}
				indexTasks = append(indexTasks, task_info)
				setIndexBatch[lang] = nil
			}
		}
	}

	mountIndexBatch := make(map[string][]SearchIndexedMount)
	for _, mount := range *mounts {
		mountCp := mount
		if err := txn.Insert("mounts", &mountCp); err != nil {
			panic(err)
		}

		for _, lang := range utils.Languages {
			object := SearchIndexedMount{
				Name:       mountCp.Name[lang],
				Id:         mountCp.AnkamaId,
				FamilyName: mountCp.FamilyName[lang],
			}

			mountIndexBatch[lang] = append(mountIndexBatch[lang], object)
			if len(mountIndexBatch[lang]) >= maxBatchSize {
				task_info, err := multilangSearchIndexes[lang].Mounts.AddDocuments(mountIndexBatch[lang])
				if err != nil {
					log.Println(err)
				}
				indexTasks = append(indexTasks, task_info)
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
		ticker := time.NewTicker(5 * time.Second)
		go func(done chan bool, indexed *bool, client *meilisearch.Client, ticker *time.Ticker) {
			var awaited []bool
			firstRun := true
			for {
				select {
				case <-done:
					ticker.Stop()
					return
				case <-ticker.C:
					allTrue := true
					for i, task := range indexTasks {
						if !firstRun && awaited[i] { // save result from last run to save requests
							continue
						}

						waitingForSuccededOrFailed := true
						taskResp, err := client.GetTask(task.TaskUID)
						if err != nil {
							log.Println(err)
							return
						}
						waitingForSuccededOrFailed = taskResp.Status == meilisearch.TaskStatusSucceeded || taskResp.Status == meilisearch.TaskStatusFailed
						if !waitingForSuccededOrFailed {
							allTrue = false
						}
						if firstRun {
							awaited = append(awaited, waitingForSuccededOrFailed)
						} else {
							awaited[i] = waitingForSuccededOrFailed
						}
					}

					if allTrue {
						ticker.Stop()
						log.Println("-- waiter stopped itself")
						*indexed = true
						return
					}
				}
			}
		}(done, indexed, client, ticker)
	}

	return db, multilangSearchIndexes
}
