package almanax

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/dofusdude/doduapi/config"
	"github.com/dofusdude/doduapi/database"
	"github.com/dofusdude/dodumap"
	mapping "github.com/dofusdude/dodumap"
	"github.com/google/go-github/v67/github"
	"github.com/meilisearch/meilisearch-go"
)

var (
	DataRepoOwner         = "dofusdude"
	DataRepoName          = "dofus3-main"
	MappedAlmanaxFileName = "MAPPED_ALMANAX.json"
)

func dateRange(from, to time.Time) ([]string, error) {
	layout := "2006-01-02"
	var dates []string

	for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d.Format(layout))
	}

	return dates, nil
}

func UpdateAlmanaxBonusIndex(init bool, db *database.Repository) int {
	client := meilisearch.New(config.MeiliHost, meilisearch.WithAPIKey(config.MeiliKey))
	defer client.Close()

	added := 0

	for _, lang := range config.Languages {
		bonusTypes, err := db.GetBonusTypes()
		if err != nil {
			log.Error(err, "lang", lang)
			return added
		}

		bonuses := bonusListingsToBonusIdTranslated(bonusTypes, lang)

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

func GatherAlmanaxData(initial bool, headless bool) error {
	db := database.NewDatabaseRepository(context.Background(), config.DbDir)
	defer db.Deinit()

	almanaxData, err := loadAlmanaxData(config.DofusVersion)
	if err != nil {
		return fmt.Errorf("could not load almanax data: %w", err)
	}

	yearLookup := make(map[string]dodumap.MappedMultilangNPCAlmanaxUnity)

	today := time.Now()
	yearFromNow := today.AddDate(1, 0, 0)
	dates, err := dateRange(today, yearFromNow)
	if err != nil {
		return fmt.Errorf("could not generate date range: %w", err)
	}

	datesNotFound := 0
	for _, date := range dates {
		for _, almanax := range almanaxData {
			if len(almanax.Days) == 0 {
				continue
			}

			for _, day := range almanax.Days {
				if day == date {
					yearLookup[day] = almanax
				}
			}
		}

		if _, ok := yearLookup[date]; !ok {
			datesNotFound++
		}
	}

	if datesNotFound > len(dates)/2 {
		return fmt.Errorf("could not find enough almanax data for the next year")
	}

	err = db.UpdateFuture(yearLookup)
	if err != nil {
		return err
	}

	if headless {
		log.Info("Next Almanax year updated successfully")
	}

	added := UpdateAlmanaxBonusIndex(initial, db)
	if headless {
		log.Info("Initial Almanax bonus index created", "count", added)
	}

	return nil
}

func loadAlmanaxData(version string) ([]mapping.MappedMultilangNPCAlmanaxUnity, error) {
	client := github.NewClient(nil)

	var repRel *github.RepositoryRelease
	var err error

	if version == "latest" {
		repRel, _, err = client.Repositories.GetLatestRelease(context.Background(), DataRepoOwner, DataRepoName)
	} else {
		repRel, _, err = client.Repositories.GetReleaseByTag(context.Background(), DataRepoOwner, DataRepoName, version)
	}
	if err != nil {
		return nil, fmt.Errorf("could not get release: %w", err)
	}

	// get the mapped almanax data
	var assetId int64
	assetId = -1
	for _, asset := range repRel.Assets {
		if asset.GetName() == MappedAlmanaxFileName {
			assetId = asset.GetID()
			break
		}
	}

	if assetId == -1 {
		return nil, fmt.Errorf("could not find asset with name %s", MappedAlmanaxFileName)
	}

	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Automatically follow all redirects
			return nil
		},
	}
	asset, redirectUrl, err := client.Repositories.DownloadReleaseAsset(context.Background(), DataRepoOwner, DataRepoName, assetId, httpClient)
	if err != nil {
		return nil, err
	}

	if asset == nil {
		return nil, fmt.Errorf("asset is nil, redirect url: %s", redirectUrl)
	}

	defer asset.Close()

	var almData []mapping.MappedMultilangNPCAlmanaxUnity
	dec := json.NewDecoder(asset)
	err = dec.Decode(&almData)
	if err != nil {
		return nil, fmt.Errorf("could not decode almanax data: %w", err)
	}

	return almData, nil
}
