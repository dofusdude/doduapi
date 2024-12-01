package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/dofusdude/doduapi/ui"
	"github.com/hashicorp/go-memdb"
	"github.com/meilisearch/meilisearch-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

var (
	DoduapiMajor       = 1                                         // Major version also used for prefixing API routes.
	DoduapiVersion     = fmt.Sprintf("v%d.0.0-rc.8", DoduapiMajor) // change with every release
	DoduapiShort       = "doduapi - Open Dofus Encyclopedia API"
	DoduapiLong        = ""
	DoduapiVersionHelp = DoduapiShort + "\n" + DoduapiVersion + "\nhttps://github.com/dofusdude/doduapi"
	httpDataServer     *http.Server
	httpMetricsServer  *http.Server
	UpdateChan         chan GameVersion
)

func AutoUpdate(version *VersionT, updateHook chan GameVersion, updateDb chan *memdb.MemDB, updateSearchIndex chan map[string]SearchIndexes, almanaxBonusTicker *time.Ticker) {
	for {
		select {
		case <-almanaxBonusTicker.C:
			added := UpdateAlmanaxBonusIndex(false)
			log.Print("updated almanax bonus index", "added", added)
		case gameVersion, ok := <-updateHook:
			if !ok {
				log.Error("updateHook closed")
				return
			}
			var err error
			updateStart := time.Now()
			log.Print("Initialize update...")
			db, idx := IndexApiData(version)

			// send data to main thread
			updateDb <- db
			log.Info("updated db")

			nowOldItemsTable := fmt.Sprintf("%s-all_items", CurrentRedBlueVersionStr(version.MemDb))
			nowOldSetsTable := fmt.Sprintf("%s-sets", CurrentRedBlueVersionStr(version.MemDb))
			nowOldMountsTable := fmt.Sprintf("%s-mounts", CurrentRedBlueVersionStr(version.MemDb))
			nowOldRecipesTable := fmt.Sprintf("%s-recipes", CurrentRedBlueVersionStr(version.MemDb))

			version.MemDb = !version.MemDb // atomic version switch
			log.Info("updated db version")

			delOldTxn := db.Txn(true)
			_, err = delOldTxn.DeleteAll(nowOldItemsTable, "id")
			if err != nil {
				log.Fatal(err)
			}
			_, err = delOldTxn.DeleteAll(nowOldSetsTable, "id")
			if err != nil {
				log.Fatal(err)
			}
			_, err = delOldTxn.DeleteAll(nowOldMountsTable, "id")
			if err != nil {
				log.Fatal(err)
			}
			_, err = delOldTxn.DeleteAll(nowOldRecipesTable, "id")
			if err != nil {
				log.Fatal(err)
			}
			delOldTxn.Commit()

			// ----
			updateSearchIndex <- idx

			client := meilisearch.New(MeiliHost, meilisearch.WithAPIKey(MeiliKey))
			defer client.Close()

			nowOldRedBlueVersion := CurrentRedBlueVersionStr(version.Search)

			version.Search = !version.Search // atomic version switch

			log.Info("changed Meili index")
			for _, lang := range Languages {
				nowOldItemIndexUid := fmt.Sprintf("%s-all_items-%s", nowOldRedBlueVersion, lang)
				nowOldSetIndexUid := fmt.Sprintf("%s-sets-%s", nowOldRedBlueVersion, lang)
				nowOldMountIndexUid := fmt.Sprintf("%s-mounts-%s", nowOldRedBlueVersion, lang)

				itemDeleteTask, err := client.DeleteIndex(nowOldItemIndexUid)
				if err != nil {
					log.Error("Error while deleting old item index.", "err", err)
					return
				}
				task, err := client.WaitForTask(itemDeleteTask.TaskUID, 500*time.Millisecond)
				if err != nil {
					log.Error("Error while deleting old item index.", "err", err)
					return
				}

				if task.Status == "failed" {
					log.Error("Error while deleting old item index.", "err", task.Error)
					return
				}

				setDeletionTask, err := client.DeleteIndex(nowOldSetIndexUid)
				if err != nil {
					log.Error("Error while deleting old set index.", "err", err)
					return
				}
				task, err = client.WaitForTask(setDeletionTask.TaskUID, 500*time.Millisecond)
				if err != nil {
					log.Error("Error while deleting old set index.", "err", err)
					return
				}

				if task.Status == "failed" {
					log.Error("Error while deleting old set index.", "err", task.Error)
					return
				}

				mountDeletionTask, err := client.DeleteIndex(nowOldMountIndexUid)
				if err != nil {
					log.Error("Error while deleting old mount index.", "err", err)
					return
				}
				task, err = client.WaitForTask(mountDeletionTask.TaskUID, 500*time.Millisecond)
				if err != nil {
					log.Error("Error while deleting old mount index.", "err", err)
					return
				}

				if task.Status == "failed" {
					log.Error("Error while deleting old mount index.", "err", task.Error)
					return
				}

			}
			log.Info("deleted old in-memory data")
			log.Print("Updated", "s", time.Since(updateStart).Seconds())

			// update version info for API meta endpoint
			gameVersion.UpdateStamp = time.Now()
			CurrentVersion = gameVersion
		}
	}
}

func isChannelClosed[T any](ch chan T) bool {
	select {
	case _, ok := <-ch:
		if !ok {
			return true
		}
	default:
	}

	return false
}

var (
	rootCmd = &cobra.Command{
		Use:   "doduapi",
		Short: DoduapiShort,
		Long:  DoduapiLong,
		Run:   rootCommand,
	}
)

func main() {
	rootCmd.PersistentFlags().Bool("headless", false, "Run without a TUI.")
	rootCmd.PersistentFlags().Bool("version", false, "Print API version.")
	rootCmd.PersistentFlags().Bool("skip-images", false, "Do not load (re)load images from the web.")
	rootCmd.PersistentFlags().Int32("alm-bonus-interval", 12, "Almanax bonuses search index interval in hours.")

	err := rootCmd.Execute()
	if err != nil && err.Error() != "" {
		fmt.Fprintln(os.Stderr, err)
	}
}

func rootCommand(ccmd *cobra.Command, args []string) {
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	var err error

	printVersion, err := ccmd.Flags().GetBool("version")
	if err != nil {
		log.Fatal(err)
	}

	skipImages, err := ccmd.Flags().GetBool("skip-images")
	if err != nil {
		log.Fatal(err)
	}

	if printVersion {
		fmt.Println(DoduapiVersionHelp)
		return
	}

	headless, err := ccmd.Flags().GetBool("headless")
	if err != nil {
		log.Fatal(err)
	}

	almBonusInterval, err := ccmd.Flags().GetInt32("alm-bonus-interval")
	if err != nil {
		log.Fatal(err)
	}

	ReadEnvs()

	feedbackChan := make(chan string, 5)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ui.Spinner("Init", feedbackChan, false, headless)
		if !isChannelClosed(feedbackChan) {
			close(feedbackChan)
		}
	}()

	if !skipImages {
		if isChannelClosed(feedbackChan) {
			os.Exit(1)
		}

		feedbackChan <- "Images"
		err = DownloadImages()
		if err != nil {
			log.Fatal(err)
		}
	}

	if isChannelClosed(feedbackChan) {
		os.Exit(1)
	}
	feedbackChan <- "Persistence"
	err = LoadPersistedElements()
	if err != nil {
		log.Fatal(err)
	}

	if isChannelClosed(feedbackChan) {
		os.Exit(1)
	}
	feedbackChan <- "Database"
	Db, Indexes = IndexApiData(&Version)
	Version.Search = !Version.Search
	Version.MemDb = !Version.MemDb

	updateDb := make(chan *memdb.MemDB)
	updateSearchIndex := make(chan map[string]SearchIndexes)
	UpdateChan = make(chan GameVersion)

	if isChannelClosed(feedbackChan) {
		os.Exit(1)
	}
	feedbackChan <- "Servers"

	httpDataServer = &http.Server{
		Addr:    fmt.Sprintf(":%s", ApiPort),
		Handler: Router(),
	}

	apiPort, _ := strconv.Atoi(ApiPort)

	if PrometheusEnabled {
		httpMetricsServer = &http.Server{
			Addr:    fmt.Sprintf(":%d", apiPort+1),
			Handler: promhttp.Handler(),
		}

		go func() {
			if err := httpMetricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatal(err)
			}
		}()
	}

	go func() {
		if err := httpDataServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	almanaxBonusSearchTicker := time.NewTicker(time.Duration(almBonusInterval) * time.Hour)

	go AutoUpdate(&Version, UpdateChan, updateDb, updateSearchIndex, almanaxBonusSearchTicker)

	added := UpdateAlmanaxBonusIndex(true)

	if !isChannelClosed(feedbackChan) {
		close(feedbackChan)
	}
	wg.Wait()

	log.Print("updated almanax bonus index", "added", added)

	var releaseLog string
	if IsBeta {
		releaseLog = "beta"
	} else {
		releaseLog = "main"
	}

	CurrentVersion.Version = DofusVersion
	CurrentVersion.UpdateStamp = time.Now()
	CurrentVersion.Release = releaseLog

	if PrometheusEnabled {
		log.Print("Listening...", "port", apiPort, "metrics", apiPort+1, "release", releaseLog)
	} else {
		log.Print("Listening...", "port", apiPort, "release", releaseLog)
	}

	go func() {
		for {
			select {
			case Db = <-updateDb: // override main memory with updated data
			case Indexes = <-updateSearchIndex:
			}
		}
	}()

	<-sigint
	fmt.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpDataServer.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
	if PrometheusEnabled {
		if err := httpMetricsServer.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}
}
