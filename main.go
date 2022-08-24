package main

import (
	"flag"
	"fmt"
	"github.com/dofusdude/api/gen"
	"github.com/dofusdude/api/server"
	"github.com/dofusdude/api/update"
	"github.com/dofusdude/api/utils"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/go-memdb"
)

func AutoUpdate(done chan bool, indexWaiterDone chan bool, indexed *bool, ticker *time.Ticker, updateDb chan *memdb.MemDB, updateSearchIndex chan map[string]gen.SearchIndexes) {
	for {
		select {
		case <-done:
			ticker.Stop()
			return
		case <-ticker.C:
			updated := update.DownloadUpdatesIfAvailable(false)
			if !updated {
				continue
			}
			gen.Parse()
			db, idx := gen.IndexApiData(indexWaiterDone, indexed)

			// send data to main thread
			updateDb <- db
			updateSearchIndex <- idx
		}
	}
}

func Hook(updaterRunning bool, updaterDone chan bool, indexWaiterCouldBeRunning bool, indexWaiterDone chan bool, updateDb chan *memdb.MemDB, updateSearchIndex chan map[string]gen.SearchIndexes) {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			select {
			case api.Db = <-updateDb: // override main memory with updated data
			case api.Indexes = <-updateSearchIndex:
			case sig := <-sigs:
				fmt.Println(sig)

				if updaterRunning {
					updaterDone <- true // signal update to stop
					fmt.Println("stopped update")
				}

				if indexWaiterCouldBeRunning && !*api.Indexed {
					indexWaiterDone <- true // signal index waiter to stop
					fmt.Println("stopped waiter")
				}

				err := server.Close()
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

var server *http.Server

func main() {
	parseFlag := flag.Bool("parseFlag", false, "Parse already existing files")
	updateFlag := flag.Bool("updateFlag", false, "Update the data")
	genFlag := flag.Bool("gen", false, "Generate API datastructure")
	serveFlag := flag.Bool("serveFlag", false, "No processing, just serveFlag")
	flag.Parse()

	all := !*parseFlag && !*updateFlag && !*genFlag && !*serveFlag

	isIndexed := false
	api.Indexed = &isIndexed

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
		gen.IndexApiData(indexWaiterDone, api.Indexed)
	}

	updateDb := make(chan *memdb.MemDB)
	updateSearchIndex := make(chan map[string]gen.SearchIndexes)
	if all || *serveFlag {

		if !all && !*genFlag {
			*api.Indexed = true
		}

		// start webserver async
		port, ok := os.LookupEnv("API_PORT")
		if !ok {
			port = "3000"
		}

		server = &http.Server{
			Addr:    fmt.Sprintf(":%s", port),
			Handler: api.Router(),
		}

		go func() {
			log.Printf("listen on port %s\n", port)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatal(err)
			}
		}()

		if all || *updateFlag {
			ticker := time.NewTicker(1 * time.Minute)
			go AutoUpdate(updaterDone, indexWaiterDone, api.Indexed, ticker, updateDb, updateSearchIndex)
		}
	}

	if all || *serveFlag {
		Hook(all || *updateFlag, updaterDone, all || *genFlag, indexWaiterDone, updateDb, updateSearchIndex) // block and wait for signal, handle db updates
	}

	if !*serveFlag && *genFlag {
		for {
			if !*api.Indexed {
				log.Println("waiting for index to finish. else there could be dataraces when starting the service again")
				time.Sleep(4 * time.Second)
			} else {
				break
			}
		}
	}
}
