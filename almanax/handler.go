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
	e "github.com/dofusdude/doduapi/errmsg"
	"github.com/dofusdude/doduapi/utils"
	"github.com/meilisearch/meilisearch-go"
)

func GetAlmanaxSingle(w http.ResponseWriter, r *http.Request) {
	// TODO
	//
}

/*
*
per default the current day almanax in the requested language
timezone paris

filter[bonus_type] can be used seperately and does not have an effect on the other parameters.

range[from] changes the start date, everything else defaults to 6 following dates from this start date.

range[to] when used without anything else, it will use today as start date and this parameter as end. All ranges are inclusive.

range[from] + range[to] = inclusive range over the specified dates, should never be farther apart than 35 days.

range[from|to] + range[size] no need to specify the date, just following days with [from] (0 is today) or go backwards in time with only [to] and [size].

Not all combinations are listed but this should give you an idea how to they could work.

- timezone - timezone to use, default Europe/Paris
*/
func GetAlmanaxRange(w http.ResponseWriter, r *http.Request) {
	/*
		lang := r.Context().Value("lang").(string)
		from := r.URL.Query().Get("range[from]")
		to := r.URL.Query().Get("range[to]")
		size := r.URL.Query().Get("range[size]")
		var sizeNum int
		bonusType := r.URL.Query().Get("filter[bonus_type]")
		timezone := r.URL.Query().Get("timezone")

		if size == "" {
			sizeNum = -1
		} else {
			sizeNum, err := strconv.Atoi(size)
			if err != nil {
				e.WriteInvalidQueryResponse(w, "Invalid size value.")
				return
			}
		}

		defaultAhead := 6

		if timezone == "" {
			timezone = "Europe/Paris"
		}

		// TODO
		//  */
}

func UpdateAlmanaxBonusIndex(init bool, db *Repository) int {
	client := meilisearch.New(config.MeiliHost, meilisearch.WithAPIKey(config.MeiliKey))
	defer client.Close()

	added := 0

	for _, lang := range Languages {
		if lang == "pt" {
			continue // no portuguese almanax bonuses
		}
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

type UpdateAlmanaxRequest struct {
	// Load the mapped_almanax on startup, update with doduda API request, reload date => npc pairs from alm-dates repo.
	// alm-dates runs short.sh etc and manages files for each year. scripts update the <year>.json if something changes.
	// UpdateAlmanaxRequest then reads the files and updates the mapped_almanax.
}

// get webhook from github with secret and newest release tag, load the newest mapped almanax and iterate into the future, updating everything
// also run the update almanax bonus index if SearchEnabled is true
func UpdateAlmanax(w http.ResponseWriter, r *http.Request) {
	// TODO
}

func bonusListingsToBonusIdTranslated(bonuses []BonusType, lang string) []AlmanaxBonusListing {
	bonusesTranslated := make([]AlmanaxBonusListing, 0, len(bonuses))
	for _, bonus := range bonuses {
		var bonusTranslated AlmanaxBonusListing
		bonusTranslated.Id = bonus.NameID
		switch lang {
		case "en":
			bonusTranslated.Name = bonus.NameEn
		case "fr":
			bonusTranslated.Name = bonus.NameFr
		case "de":
			bonusTranslated.Name = bonus.NameDe
		case "es":
			bonusTranslated.Name = bonus.NameEs
		case "pt":
			bonusTranslated.Name = bonus.NamePt
		}
		bonusesTranslated = append(bonusesTranslated, bonusTranslated)
	}

	return bonusesTranslated
}

func ListBonuses(w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	db := NewDatabaseRepository(context.Background(), config.DbDir)

	bonuses, err := db.GetBonusTypes()
	if err != nil {
		e.WriteServerErrorResponse(w, "Could not get bonus types: "+err.Error())
		return
	}

	if len(bonuses) == 0 {
		e.WriteNotFoundResponse(w, "No bonuses found.")
		return
	}

	bonusesTranslated := bonusListingsToBonusIdTranslated(bonuses, lang)

	utils.WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(bonusesTranslated)
	if err != nil {
		e.WriteServerErrorResponse(w, "Could not encode JSON: "+err.Error())
		return
	}
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

func SetJsonHeader(w *http.ResponseWriter) {
	(*w).Header().Set("Content-Type", "application/json")
}

func SearchBonuses(w http.ResponseWriter, r *http.Request) {
	client := meilisearch.New(config.MeiliHost, meilisearch.WithAPIKey(config.MeiliKey))
	defer client.Close()

	query := r.URL.Query().Get("query")
	if query == "" {
		e.WriteInvalidQueryResponse(w, "Query parameter is required.")
		return
	}

	lang := r.Context().Value("lang").(string)

	if lang == "pt" {
		e.WriteInvalidQueryResponse(w, "Portuguese language is not translated for Almanax Bonuses.")
		return
	}

	var searchLimit int64
	var err error
	if searchLimit, err = getLimitInBoundary(r.URL.Query().Get("limit")); err != nil {
		e.WriteInvalidQueryResponse(w, "Invalid limit value: "+err.Error())
		return
	}

	index := client.Index(fmt.Sprintf("alm-bonuses-%s", lang))

	request := &meilisearch.SearchRequest{
		Limit: searchLimit,
	}

	var searchResp *meilisearch.SearchResponse
	if searchResp, err = index.Search(query, request); err != nil {
		e.WriteServerErrorResponse(w, "Could not search: "+err.Error())
		return
	}

	//requestsTotal.Inc()
	//requestsSearchTotal.Inc()

	if searchResp.EstimatedTotalHits == 0 {
		e.WriteNotFoundResponse(w, "No results found.")
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

	utils.WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(results)
	if err != nil {
		e.WriteServerErrorResponse(w, "Could not encode JSON: "+err.Error())
		return
	}
}
