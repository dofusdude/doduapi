package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hashicorp/go-memdb"
)

var Db *memdb.MemDB
var Indexes map[string]SearchIndexes

var Version VersionT

func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		log.Fatal("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		routeCtx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(routeCtx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}

func Router() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(10 * time.Second))

	var routePrefix string
	if IsBeta {
		routePrefix = "/dofus2beta"
	} else {
		routePrefix = "/dofus2"
	}

	r.With(useCors).Route(routePrefix, func(r chi.Router) {

		if PublishFileServer {
			imagesDir := http.Dir(filepath.Join(DockerMountDataPath, "data", "img"))
			FileServer(r, "/img", imagesDir)
		}

		r.Route("/update", func(r chi.Router) {
			r.Post(fmt.Sprintf("/%s", UpdateHookToken), UpdateHandler)
		})

		r.Route("/meta", func(r chi.Router) {
			r.Get("/elements", ListEffectConditionElements)
			r.Get("/search/types", ListSearchAllTypes)
		})

		r.With(languageChecker).Route("/{lang}", func(r chi.Router) {
			r.Route("/search", func(r chi.Router) {
				r.Get("/", SearchAllIndices)
			})

			r.Route("/items", func(r chi.Router) {
				r.Route("/consumables", func(r chi.Router) {
					r.With(paginate).Get("/", ListConsumables)
					r.With(disablePaginate).Get("/all", ListAllConsumables)
					r.With(ankamaIdExtractor).Get("/{ankamaId}", GetSingleConsumableHandler)
					r.Get("/search", SearchConsumables)
				})

				r.Route("/resources", func(r chi.Router) {
					r.With(paginate).Get("/", ListResources)
					r.With(disablePaginate).Get("/all", ListAllResources)
					r.With(ankamaIdExtractor).Get("/{ankamaId}", GetSingleResourceHandler)
					r.Get("/search", SearchResources)
				})

				r.Route("/equipment", func(r chi.Router) {
					r.With(paginate).Get("/", ListEquipment)
					r.With(disablePaginate).Get("/all", ListAllEquipment)
					r.With(ankamaIdExtractor).Get("/{ankamaId}", GetSingleEquipmentHandler)
					r.Get("/search", SearchEquipment)
				})

				r.Route("/quest", func(r chi.Router) {
					r.With(paginate).Get("/", ListQuestItems)
					r.With(disablePaginate).Get("/all", ListAllQuestItems)
					r.With(ankamaIdExtractor).Get("/{ankamaId}", GetSingleQuestItemHandler)
					r.Get("/search", SearchQuestItems)
				})

				r.Route("/cosmetics", func(r chi.Router) {
					r.With(paginate).Get("/", ListCosmetics)
					r.With(disablePaginate).Get("/all", ListAllCosmetics)
					r.With(ankamaIdExtractor).Get("/{ankamaId}", GetSingleCosmeticHandler)
					r.Get("/search", SearchCosmetics)
				})

				r.Get("/search", SearchAllItems)

			})

			r.Route("/mounts", func(r chi.Router) {
				r.With(paginate).Get("/", ListMounts)
				r.With(disablePaginate).Get("/all", ListAllMounts)
				r.With(ankamaIdExtractor).Get("/{ankamaId}", GetSingleMountHandler)
				r.Get("/search", SearchMounts)
			})

			r.Route("/sets", func(r chi.Router) {
				r.With(paginate).Get("/", ListSets)
				r.With(disablePaginate).Get("/all", ListAllSets)
				r.With(ankamaIdExtractor).Get("/{ankamaId}", GetSingleSetHandler)
				r.Get("/search", SearchSets)
			})
		})
	})

	return r
}
