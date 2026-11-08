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

func init() { chi.RegisterMethod("QUERY") }

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
	r.Use(func(next http.Handler) http.Handler {
		standard := middleware.Timeout(10 * time.Second)(next)
		bulk := middleware.Timeout(2 * time.Minute)(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, resource := range mappedEntityResources {
				if strings.HasSuffix(r.URL.Path, "/"+resource.Route+"/all") {
					bulk.ServeHTTP(w, r)
					return
				}
			}
			standard.ServeHTTP(w, r)
		})
	})
	r.Use(useCors)

	var gameRelease string
	if config.IsBeta {
		gameRelease = "dofus3beta"
	} else {
		gameRelease = "dofus3"
	}

	r.Route(fmt.Sprintf("/%s/v%d", gameRelease, DoduapiMajor), func(r chi.Router) {

		if config.PublishFileServer {
			imagesPath := filepath.Join(config.DockerMountDataPath, "data", "img")
			if config.DevMode {
				imagesPath = filepath.Join(config.DodudaOutputDir, "img")
			}
			imagesDir := http.Dir(imagesPath)
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
				registerQueryGET(r, "/search", almanax.SearchBonuses)
			})
		})

		r.With(languageChecker).Route("/{lang}", func(r chi.Router) {
			r.Route("/search", func(r chi.Router) {
				registerQueryGET(r, "/", SearchAllIndices)
			})

			r.Route("/almanax", func(r chi.Router) {
				registerQueryGET(r, "/", almanax.GetAlmanaxRange)
				r.With(dateExtractor).Get("/{date}", almanax.GetAlmanaxSingle)
			})

			r.Route("/items", func(r chi.Router) {
				r.Route("/consumables", func(r chi.Router) {
					registerQueryGET(r, "/", ListConsumables, paginate)
					registerQueryGET(r, "/all", ListAllConsumables, disablePaginate)
					r.With(ankamaIdExtractor).Get("/{ankamaId}", GetSingleConsumableHandler)
					registerQueryGET(r, "/search", SearchConsumables)
				})

				r.Route("/resources", func(r chi.Router) {
					registerQueryGET(r, "/", ListResources, paginate)
					registerQueryGET(r, "/all", ListAllResources, disablePaginate)
					r.With(ankamaIdExtractor).Get("/{ankamaId}", GetSingleResourceHandler)
					registerQueryGET(r, "/search", SearchResources)
				})

				r.Route("/equipment", func(r chi.Router) {
					registerQueryGET(r, "/", ListEquipment, paginate)
					registerQueryGET(r, "/all", ListAllEquipment, disablePaginate)
					r.With(ankamaIdExtractor).Get("/{ankamaId}", GetSingleEquipmentHandler)
					registerQueryGET(r, "/search", SearchEquipment)
				})

				r.Route("/quest", func(r chi.Router) {
					registerQueryGET(r, "/", ListQuestItems, paginate)
					registerQueryGET(r, "/all", ListAllQuestItems, disablePaginate)
					r.With(ankamaIdExtractor).Get("/{ankamaId}", GetSingleQuestItemHandler)
					registerQueryGET(r, "/search", SearchQuestItems)
				})

				r.Route("/cosmetics", func(r chi.Router) {
					registerQueryGET(r, "/", ListCosmetics, paginate)
					registerQueryGET(r, "/all", ListAllCosmetics, disablePaginate)
					r.With(ankamaIdExtractor).Get("/{ankamaId}", GetSingleCosmeticHandler)
					registerQueryGET(r, "/search", SearchCosmetics)
				})

				registerQueryGET(r, "/search", SearchAllItems)

			})

			r.Route("/mounts", func(r chi.Router) {
				registerQueryGET(r, "/", ListMounts, paginate)
				registerQueryGET(r, "/all", ListAllMounts, disablePaginate)
				r.With(ankamaIdExtractor).Get("/{ankamaId}", GetSingleMountHandler)
				registerQueryGET(r, "/search", SearchMounts)
			})

			r.Route("/sets", func(r chi.Router) {
				registerQueryGET(r, "/", ListSets, paginate)
				registerQueryGET(r, "/all", ListAllSets, disablePaginate)
				r.With(ankamaIdExtractor).Get("/{ankamaId}", GetSingleSetHandler)
				registerQueryGET(r, "/search", SearchSets)
			})

			for _, resource := range mappedEntityResources {
				r.Route("/"+resource.Route, func(r chi.Router) {
					registerQueryGET(r, "/", func(w http.ResponseWriter, request *http.Request) {
						ListMappedEntities(resource, w, request)
					}, paginate)
					r.With(decodeQueryTransport, disablePaginate, includeFullMappedEntity).Get("/all", func(w http.ResponseWriter, request *http.Request) {
						ListMappedEntities(resource, w, request)
					})
					registerQueryGET(r, "/search", func(w http.ResponseWriter, request *http.Request) {
						SearchMappedEntities(resource, w, request)
					})
					r.With(ankamaIdExtractor).Get("/{ankamaId}", func(w http.ResponseWriter, request *http.Request) {
						GetSingleMappedEntity(resource, w, request)
					})
				})
			}
		})
	})

	return r
}
