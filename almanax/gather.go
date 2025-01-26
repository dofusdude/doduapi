package almanax

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
	"github.com/dofusdude/doduapi/config"
	"github.com/dofusdude/doduapi/database"
	"github.com/dofusdude/dodumap"
	mapping "github.com/dofusdude/dodumap"
	"github.com/google/go-github/v67/github"
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
			return err
		}
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
