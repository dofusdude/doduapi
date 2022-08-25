package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dofusdude/api/gen"
	"github.com/dofusdude/api/server"
	"github.com/dofusdude/api/update"
	"github.com/dofusdude/api/utils"

	"github.com/hashicorp/go-memdb"
)

func AutoUpdate(done chan bool, indexed *bool, version *utils.VersionT, ticker *time.Ticker, updateDb chan *memdb.MemDB, updateSearchIndex chan map[string]gen.SearchIndexes) {
	indexWaiterDone := make(chan bool)
	for {
		select {
		case <-done:
			indexWaiterDone <- true
			ticker.Stop()
			return
		case <-ticker.C:
			updated := update.DownloadUpdatesIfAvailable(false)
			if !updated {
				continue
			}
			gen.Parse()
			db, idx := gen.IndexApiData(indexWaiterDone, indexed, version)

			// send data to main thread
			updateDb <- db

			nowOldItemsTable := fmt.Sprintf("%s-all_items", utils.CurrentRedBlueVersionStr(version.MemDb))
			nowOldSetsTable := fmt.Sprintf("%s-sets", utils.CurrentRedBlueVersionStr(version.MemDb))
			nowOldMountsTable := fmt.Sprintf("%s-mounts", utils.CurrentRedBlueVersionStr(version.MemDb))
			nowOldRecipesTable := fmt.Sprintf("%s-recipes", utils.CurrentRedBlueVersionStr(version.MemDb))

			version.MemDb = !version.MemDb // atomic version switch

			delOldTxn := db.Txn(true)
			_, err := delOldTxn.DeleteAll(nowOldItemsTable, "id")
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

			client := utils.CreateMeiliClient()
			nowOldRedBlueVersion := utils.CurrentRedBlueVersionStr(version.Search)

			version.Search = !version.Search // atomic version switch

			log.Println("-- updater: changed search version")
			for _, lang := range utils.Languages {
				nowOldItemIndexUid := fmt.Sprintf("%s-all_items-%s", nowOldRedBlueVersion, lang)
				nowOldSetIndexUid := fmt.Sprintf("%s-sets-%s", nowOldRedBlueVersion, lang)
				nowOldMountIndexUid := fmt.Sprintf("%s-mounts-%s", nowOldRedBlueVersion, lang)

				itemDeleteTask, err := client.DeleteIndex(nowOldItemIndexUid)
				_, err = client.WaitForTask(itemDeleteTask.TaskUID)
				if err != nil {
					log.Fatal(err)
				}

				setDeletionTask, err := client.DeleteIndex(nowOldSetIndexUid)
				_, err = client.WaitForTask(setDeletionTask.TaskUID)
				if err != nil {
					log.Fatal(err)
				}

				mountDeletionTask, err := client.DeleteIndex(nowOldMountIndexUid)
				_, err = client.WaitForTask(mountDeletionTask.TaskUID)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
}

func Hook(updaterRunning bool, updaterDone chan bool, updateDb chan *memdb.MemDB, updateSearchIndex chan map[string]gen.SearchIndexes) {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			select {
			case server.Db = <-updateDb: // override main memory with updated data

			case server.Indexes = <-updateSearchIndex:
			case sig := <-sigs:
				fmt.Println(sig)

				if updaterRunning {
					updaterDone <- true // signal update to stop
					fmt.Println("stopped update routine")
				}

				err := httpServer.Close()
				if err != nil {
					panic(err)
				} // close http connections and delete server
				done <- true
			}
		}
	}()

	<-done
	fmt.Println("Bye!")
}

var httpServer *http.Server

func main() {
	parseFlag := flag.Bool("parse", false, "Parse already existing files")
	updateFlag := flag.Bool("update", false, "Update the data")
	genFlag := flag.Bool("gen", false, "Generate API datastructure")
	serveFlag := flag.Bool("serve", false, "No processing, just serveFlag")
	flag.Parse()
	utils.ReadEnvs()

	all := !*parseFlag && !*updateFlag && !*genFlag && !*serveFlag

	server.Indexed = false

	updaterDone := make(chan bool)
	indexWaiterDone := make(chan bool)

	utils.CreateDataDirectoryStructure()

	if all || *updateFlag {
		startHashes := time.Now()
		update.DownloadUpdatesIfAvailable(true)
		log.Println("downloading game files took", time.Since(startHashes))
	}

	if all || *parseFlag {
		gen.Parse()
	}

	if all || *genFlag {
		server.Db, server.Indexes = gen.IndexApiData(indexWaiterDone, &server.Indexed, &server.Version)
		server.Version.Search = !server.Version.Search
		server.Version.MemDb = !server.Version.MemDb
	}

	updateDb := make(chan *memdb.MemDB)
	updateSearchIndex := make(chan map[string]gen.SearchIndexes)
	if all || *serveFlag {

		if !all && !*genFlag {
			server.Indexed = true
		}

		// start webserver async
		httpServer = &http.Server{
			Addr:    fmt.Sprintf(":%s", utils.ApiPort),
			Handler: server.Router(),
		}

		go func() {
			log.Printf("listen on port %s\n", utils.ApiPort)
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatal(err)
			}
		}()

		if all || *updateFlag {
			ticker := time.NewTicker(1 * time.Minute)
			go AutoUpdate(updaterDone, &server.Indexed, &server.Version, ticker, updateDb, updateSearchIndex)
		}
	}

	if all || *serveFlag {
		Hook(all || *updateFlag, updaterDone, updateDb, updateSearchIndex) // block and wait for signal, handle db updates
	}

	if !*serveFlag && *genFlag {
		for {
			if !server.Indexed {
				log.Println("waiting for index to finish. else there could be dataraces when starting the service again")
				time.Sleep(4 * time.Second)
			} else {
				break
			}
		}
	}
}
