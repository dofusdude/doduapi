package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/dofusdude/doduapi/almanax"
	"github.com/dofusdude/doduapi/config"
	"github.com/dofusdude/doduapi/database"
	"github.com/dofusdude/doduapi/ui"
	"github.com/dofusdude/doduapi/utils"
	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/sqlite3"
	"github.com/golang-migrate/migrate/source/file"
	"github.com/hashicorp/go-memdb"
	"github.com/meilisearch/meilisearch-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	DoduapiMajor       = 1                                          // Major version also used for prefixing API routes.
	DoduapiVersion     = fmt.Sprintf("v%d.0.0-rc.11", DoduapiMajor) // change with every release
	DoduapiShort       = "doduapi - Open Dofus Encyclopedia API"
	DoduapiLong        = ""
	DoduapiVersionHelp = DoduapiShort + "\n" + DoduapiVersion + "\nhttps://github.com/dofusdude/doduapi"
	httpDataServer     *http.Server
	httpMetricsServer  *http.Server
	UpdateChan         chan utils.GameVersion
)

var currentWd string

func ReadEnvs() {
	config.MajorVersion = DoduapiMajor

	viper.SetDefault("API_SCHEME", "http")
	viper.SetDefault("API_HOSTNAME", "localhost:3000")
	viper.SetDefault("API_PORT", "3000")
	viper.SetDefault("MEILI_PORT", "7700")
	viper.SetDefault("MEILI_MASTER_KEY", "masterKey")
	viper.SetDefault("MEILI_PROTOCOL", "http")
	viper.SetDefault("MEILI_HOST", "127.0.0.1")
	viper.SetDefault("PROMETHEUS", "false")
	viper.SetDefault("FILESERVER", "true")
	viper.SetDefault("ALMANAX_MAX_LOOKAHEAD_DAYS", 365)
	viper.SetDefault("ALMANAX_DEFAULT_LOOKAHEAD_DAYS", 6)
	viper.SetDefault("IS_BETA", "false")
	viper.SetDefault("UPDATE_HOOK_TOKEN", "")
	viper.SetDefault("DOFUS_VERSION", "")
	viper.SetDefault("LOG_LEVEL", "warn")

	var err error
	currentWd, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	viper.SetDefault("DIR", currentWd)

	viper.AutomaticEnv()

	config.IsBeta = viper.GetBool("IS_BETA")
	var betaStr string
	if config.IsBeta {
		betaStr = "beta"
	} else {
		betaStr = "main"
	}

	config.AlmanaxMaxLookAhead = viper.GetInt("ALMANAX_MAX_LOOKAHEAD_DAYS")
	config.AlmanaxDefaultLookAhead = viper.GetInt("ALMANAX_DEFAULT_LOOKAHEAD_DAYS")

	dofusVersion := viper.GetString("DOFUS_VERSION")
	if dofusVersion == "" {
		releaseApiResponse, err := http.Get(fmt.Sprintf("https://api.github.com/repos/dofusdude/dofus3-%s/releases/latest", betaStr))
		if err != nil {
			log.Fatal(err)
		}

		releaseApiResponseBody, err := io.ReadAll(releaseApiResponse.Body)
		if err != nil {
			log.Fatal(err)
		}

		var v map[string]interface{}
		err = json.Unmarshal(releaseApiResponseBody, &v)
		if err != nil {
			log.Fatal(err)
		}

		config.DofusVersion = v["name"].(string)
	} else {
		config.DofusVersion = dofusVersion
	}
	parsedLevel, err := log.ParseLevel(viper.GetString("LOG_LEVEL"))
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(parsedLevel)

	dofus3Prefix := ""
	if strings.HasPrefix(config.DofusVersion, "3") {
		dofus3Prefix = ".dofus3"
	}

	config.ElementsUrl = fmt.Sprintf("https://raw.githubusercontent.com/dofusdude/doduda/main/persistent/elements%s.%s.json", dofus3Prefix, betaStr)
	config.TypesUrl = fmt.Sprintf("https://raw.githubusercontent.com/dofusdude/doduda/main/persistent/item_types%s.%s.json", dofus3Prefix, betaStr)
	config.ReleaseUrl = fmt.Sprintf("https://github.com/dofusdude/dofus3-%s/releases/download/%s", betaStr, config.DofusVersion)

	config.ApiScheme = viper.GetString("API_SCHEME")
	config.ApiHostName = viper.GetString("API_HOSTNAME")
	config.ApiPort = viper.GetString("API_PORT")
	config.MeiliKey = viper.GetString("MEILI_MASTER_KEY")
	config.MeiliHost = fmt.Sprintf("%s://%s:%s", viper.GetString("MEILI_PROTOCOL"), viper.GetString("MEILI_HOST"), viper.GetString("MEILI_PORT"))
	config.PrometheusEnabled = viper.GetBool("PROMETHEUS")
	config.PublishFileServer = viper.GetBool("FILESERVER")
	config.UpdateHookToken = viper.GetString("UPDATE_HOOK_TOKEN")
	config.DockerMountDataPath = viper.GetString("DIR")
}

func AutoUpdate(version *database.VersionT, updateHook chan utils.GameVersion, updateDb chan *memdb.MemDB, updateSearchIndex chan map[string]database.SearchIndexes) {
	for {
		select {
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

			nowOldItemsTable := fmt.Sprintf("%s-all_items", utils.CurrentRedBlueVersionStr(version.MemDb))
			nowOldSetsTable := fmt.Sprintf("%s-sets", utils.CurrentRedBlueVersionStr(version.MemDb))
			nowOldMountsTable := fmt.Sprintf("%s-mounts", utils.CurrentRedBlueVersionStr(version.MemDb))
			nowOldRecipesTable := fmt.Sprintf("%s-recipes", utils.CurrentRedBlueVersionStr(version.MemDb))

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

			err = almanax.GatherAlmanaxData(false, true) // headless true since we want the log output
			if err != nil {
				log.Fatal(err) // TODO notify on error, not just hard exit since we want high availability
			}

			nowOldRedBlueVersion := utils.CurrentRedBlueVersionStr(version.Search)

			log.Info("atomic version switch")
			version.Search = !version.Search

			client := meilisearch.New(config.MeiliHost, meilisearch.WithAPIKey(config.MeiliKey))
			defer client.Close()

			for _, lang := range config.Languages {
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
			config.CurrentVersion = gameVersion
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

	migrateCmd = &cobra.Command{
		Use:           "migrate",
		Short:         "Run migrations on the database.",
		SilenceErrors: true,
		SilenceUsage:  false,
	}

	migrateDownCmd = &cobra.Command{
		Use:   "down",
		Short: "run migrations for downgrading",
		Long:  `Command to downgrade database`,
		Run:   migrateDown,
	}

	migrateUpCmd = &cobra.Command{
		Use:   "up",
		Short: "run migrations for upgrading",
		Long:  `Command to upgrade database`,
		Run:   migrateUp,
	}
)

func migrateUp(cmd *cobra.Command, args []string) {
	dbdir, err := cmd.Flags().GetString("persistent-dir")
	if err != nil {
		log.Fatal(err)
	}
	config.DbDir = dbdir

	if _, err := os.Stat(dbdir); os.IsNotExist(err) {
		os.Mkdir(dbdir, 0755)
	}

	database := database.NewDatabaseRepository(context.Background(), dbdir)

	dbDriver, err := sqlite3.WithInstance(database.Db, &sqlite3.Config{})
	if err != nil {
		log.Fatalf("instance error: %v \n", err)
	}

	fileSource, err := (&file.File{}).Open("file://migrations")
	if err != nil {
		log.Fatalf("opening file error: %v \n", err)
	}

	m, err := migrate.NewWithInstance("file", fileSource, "myDB", dbDriver)
	if err != nil {
		log.Fatalf("migrate error: %v \n", err)
	}

	if err = m.Up(); err != nil {
		log.Fatalf("migrate up error: %v \n", err)
	}

	log.Print("Migrate up done with success")
}

func migrateDown(cmd *cobra.Command, args []string) {
	dbdir, err := cmd.Flags().GetString("persistent-dir")
	if err != nil {
		log.Fatal(err)
	}
	config.DbDir = dbdir

	database := database.NewDatabaseRepository(context.Background(), dbdir)

	dbDriver, err := sqlite3.WithInstance(database.Db, &sqlite3.Config{})
	if err != nil {
		log.Fatalf("instance error: %v \n", err)
	}

	fileSource, err := (&file.File{}).Open("file://migrations")
	if err != nil {
		log.Fatalf("opening file error: %v \n", err)
	}

	m, err := migrate.NewWithInstance("file", fileSource, "myDB", dbDriver)
	if err != nil {
		log.Fatalf("migrate error: %v \n", err)
	}

	if err = m.Down(); err != nil {
		log.Fatalf("migrate down error: %v \n", err)
	}

	log.Print("Migrate down done with success")
}

func main() {
	rootCmd.PersistentFlags().Bool("headless", false, "Run without a TUI.")
	rootCmd.PersistentFlags().Bool("version", false, "Print API version.")
	rootCmd.PersistentFlags().Bool("skip-images", false, "Do not load (re)load images from the web.")
	rootCmd.PersistentFlags().String("persistent-dir", ".", "Directory for persistent data like databases.")

	migrateCmd.AddCommand(migrateDownCmd)
	migrateCmd.AddCommand(migrateUpCmd)
	rootCmd.AddCommand(migrateCmd)

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

	dbdir, err := ccmd.Flags().GetString("persistent-dir")
	if err != nil {
		log.Fatal(err)
	}
	config.DbDir = dbdir

	// populate env vars
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
		err = utils.DownloadImages(config.DockerMountDataPath, config.ReleaseUrl)
		if err != nil {
			log.Fatal(err)
		}
	}

	if isChannelClosed(feedbackChan) {
		os.Exit(1)
	}
	feedbackChan <- "Persistence"
	config.PersistedElements, config.PersistedTypes, err = utils.LoadPersistedElements(config.ElementsUrl, config.TypesUrl)
	if err != nil {
		log.Fatal(err)
	}

	if isChannelClosed(feedbackChan) {
		os.Exit(1)
	}
	feedbackChan <- "Almanax"
	err = almanax.GatherAlmanaxData(true, headless)
	if err != nil {
		log.Fatal(err)
	}

	if isChannelClosed(feedbackChan) {
		os.Exit(1)
	}
	feedbackChan <- "Database"
	database.Db, database.Indexes = IndexApiData(&database.Version)
	database.Version.Search = !database.Version.Search
	database.Version.MemDb = !database.Version.MemDb

	updateDb := make(chan *memdb.MemDB)
	updateSearchIndex := make(chan map[string]database.SearchIndexes)
	UpdateChan = make(chan utils.GameVersion)

	if isChannelClosed(feedbackChan) {
		os.Exit(1)
	}
	feedbackChan <- "Servers"

	httpDataServer = &http.Server{
		Addr:    fmt.Sprintf(":%s", config.ApiPort),
		Handler: Router(),
	}

	apiPort, _ := strconv.Atoi(config.ApiPort)

	if config.PrometheusEnabled {
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

	go AutoUpdate(&database.Version, UpdateChan, updateDb, updateSearchIndex)

	_ = UpdateAlmanaxBonusIndex(true)

	if !isChannelClosed(feedbackChan) {
		close(feedbackChan)
	}
	wg.Wait()

	var releaseLog string
	if config.IsBeta {
		releaseLog = "beta"
	} else {
		releaseLog = "main"
	}

	config.CurrentVersion.Version = config.DofusVersion
	config.CurrentVersion.UpdateStamp = time.Now()
	config.CurrentVersion.Release = releaseLog

	if config.PrometheusEnabled {
		log.Print("Listening...", "port", apiPort, "metrics", apiPort+1, "release", releaseLog)
	} else {
		log.Print("Listening...", "port", apiPort, "release", releaseLog)
	}

	go func() {
		for {
			select {
			case database.Db = <-updateDb: // override main memory with updated data
			case database.Indexes = <-updateSearchIndex:
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
	if config.PrometheusEnabled {
		if err := httpMetricsServer.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}
}
