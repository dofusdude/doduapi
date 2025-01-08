package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/dofusdude/doduapi/almanax"
	"github.com/dofusdude/doduapi/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

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

	var gameRelease string
	if config.IsBeta {
		gameRelease = "dofus3beta"
	} else {
		gameRelease = "dofus3"
	}

	r.With(useCors).Route(fmt.Sprintf("/%s/v%d", gameRelease, DoduapiMajor), func(r chi.Router) {

		if config.PublishFileServer {
			imagesDir := http.Dir(filepath.Join(config.DockerMountDataPath, "data", "img"))
			FileServer(r, "/img", imagesDir)
		}

		r.Route("/update", func(r chi.Router) {
			r.Post(fmt.Sprintf("/%s", config.UpdateHookToken), UpdateHandler)
		})

		r.Route("/meta", func(r chi.Router) {
			r.Get("/version", GetGameVersion)
			r.Get("/elements", ListEffectConditionElements)
			r.Get("/items/types", ListItemTypeIds)
			r.Get("/search/types", ListSearchAllTypes)

			r.With(languageChecker).Route("/{lang}/almanax/bonuses", func(r chi.Router) {
				r.Get("/", almanax.ListBonuses)
				r.Get("/search", almanax.SearchBonuses)
			})
		})

		r.With(languageChecker).Route("/{lang}", func(r chi.Router) {
			r.Route("/search", func(r chi.Router) {
				r.Get("/", SearchAllIndices)
			})

			r.Route("/almanax", func(r chi.Router) {
				r.Get("/", almanax.GetAlmanaxRange)
				r.With(dateExtractor).Get("/{date}", almanax.GetAlmanaxSingle)
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
