package api

import (
	"encoding/json"
	"fmt"
	"github.com/dofusdude/api/gen"
	"github.com/dofusdude/api/utils"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hashicorp/go-memdb"
	"github.com/meilisearch/meilisearch-go"
	"log"
	"net/http"
	"strconv"
	"time"
)

var Db *memdb.MemDB
var Indexes map[string]gen.SearchIndexes

var Indexed *bool

/* // TODO bonusesIds => real items? amounts are to high? how to compute the date?
r.Route("/almanax", func(r chi.Router) {
	r.Route("/bonuses", func(r chi.Router) {
		r.Get("/", SearchAll)
		r.Get("/{bonus_type}/next", SearchAll)
	})

	r.Route("/ahead/{days_ahead}", func(r chi.Router) {
		r.Get("/", SearchAll)
		r.Get("/bonus/{bonus_type}", SearchAll)
		r.Get("/items", SearchAll)
	})

	r.Get("/{date}", SearchAll)
})
*/

func Router() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Route("/dofus/{lang}", func(r chi.Router) {
		r.Route("/items", func(r chi.Router) {
			r.Route("/consumables", func(r chi.Router) {
				r.With(paginate).Get("/", ListConsumables)
				r.Get("/{ankamaId}", GetConsumableHandler)
				r.Get("/search", SearchAll)
			})

			r.Route("/resources", func(r chi.Router) {
				r.With(paginate).Get("/", ListConsumables)
				r.Get("/{ankamaId}", GetConsumableHandler)
				r.Get("/search", SearchAll)
			})

			r.Route("/equipment", func(r chi.Router) {
				r.With(paginate).Get("/", ListConsumables)
				r.Get("/{ankamaId}", GetConsumableHandler)
				r.Get("/search", SearchAll)
			})

			r.Route("/quest", func(r chi.Router) {
				r.With(paginate).Get("/", ListConsumables)
				r.Get("/{ankamaId}", GetConsumableHandler)
				r.Get("/search", SearchAll)
			})

			r.Route("/cosmetics", func(r chi.Router) {
				r.With(paginate).Get("/", ListConsumables)
				r.Get("/{ankamaId}", GetConsumableHandler)
				r.Get("/search", SearchAll)
			})

			r.Get("/search", SearchAll)

		})

		r.Route("/mounts", func(r chi.Router) {
			r.With(paginate).Get("/", ListConsumables)
			r.Get("/{ankamaId}", GetConsumableHandler)
			r.Get("/search", SearchAll)
		})

		r.Route("/sets", func(r chi.Router) {
			r.With(paginate).Get("/", ListConsumables)
			r.Get("/{ankamaId}", GetConsumableHandler)
			r.Get("/search", SearchAll)
		})
	})

	return r
}

// paginate is a stub, but very possible to implement middleware logic
// to handle the request params for handling a paginated request.
func paginate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// just a stub.. some ideas are to look at URL query params for something like
		// the page number, or the limit, and send a query cursor down the chain
		next.ServeHTTP(w, r)
	})
}

func ListConsumables(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hi"))
}

func SearchAll(w http.ResponseWriter, r *http.Request) {
	if !*Indexed {
		w.WriteHeader(http.StatusProcessing)
		w.Write([]byte("102 -- index building in progress, try again later"))
		return
	}

	client := utils.CreateMeiliClient()
	query := r.URL.Query().Get("q")
	lang := chi.URLParam(r, "lang")

	index := client.Index(fmt.Sprintf("all_items-%s", lang))
	searchResp, err := index.Search(query, &meilisearch.SearchRequest{})
	if err != nil {
		log.Println(err)
		w.Write([]byte("error"))
		return
	}

	log.Println(searchResp)
	if searchResp.EstimatedTotalHits == 0 {
		w.Write([]byte("no results"))
		return
	}

	rawRespJson, err := searchResp.MarshalJSON()
	if err != nil {
		log.Println(err)
		w.Write([]byte("error"))
		return
	}

	w.Write(rawRespJson)

	/*
	   raw_query := r.URL.Query().Get("q")
	   search_query := fmt.Sprintf("%s", raw_query)

	   var results []ApiSearchResult

	   	for _, hit := range searchResult.Hits {
	   		id_type_split := strings.Split(hit.ID, ":")
	   		id, _ := strconv.Atoi(id_type_split[0])
	   		results = append(results, ApiSearchResult{
	   			Id:    id,
	   			Score: hit.Score,
	   		})
	   	}

	   json.NewEncoder(w).Encode(results)
	*/
}

func GetConsumablesHandler(w http.ResponseWriter, r *http.Request) {
	ankamaId := chi.URLParam(r, "ankamaId")
	lang := chi.URLParam(r, "lang")
	log.Println("serving", ankamaId, "in", lang)
	w.Write([]byte("hi"))
}

func GetConsumableHandler(w http.ResponseWriter, r *http.Request) {
	//r.URL.Query().Get("q")
	lang := chi.URLParam(r, "lang")
	ankamaId, err := strconv.Atoi(chi.URLParam(r, "ankamaId"))
	if err != nil {
		panic(err)
	}

	txn := Db.Txn(false)
	defer txn.Abort()

	raw, err := txn.First("consumables", "id", ankamaId)
	if err != nil {
		panic(err)
	}

	consumable := RenderConsumable(raw.(*gen.MappedMultilangItem), lang)
	json.NewEncoder(w).Encode(consumable)
}
