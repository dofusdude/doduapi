package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dofusdude/api/gen"
	"github.com/dofusdude/api/utils"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hashicorp/go-memdb"
)

var Db *memdb.MemDB
var Indexes map[string]gen.SearchIndexes

var Indexed bool

var Version utils.VersionT

func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
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

	workDir, _ := os.Getwd()

	var routePrefix string
	if utils.IsBeta {
		routePrefix = "/dofus2beta"
	} else {
		routePrefix = "/dofus2"
	}

	r.With(useCors).Route(routePrefix, func(r chi.Router) {

		if utils.FileServer {
			imagesDir := http.Dir(filepath.Join(workDir, "data", "img"))
			FileServer(r, "/img", imagesDir)
		}

		r.Route("/meta", func(r chi.Router) {
			r.Get("/elements", ListEffectConditionElements)
		})

		r.With(languageChecker).Route("/{lang}", func(r chi.Router) {
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

			r.Route("/monsters", func(r chi.Router) {
				r.With(paginate).Get("/", ListMonster)
				r.With(disablePaginate).Get("/all", ListMonster)
				r.With(ankamaIdExtractor).Get("/{ankamaId}", GetSingleMonsterHandler)
				r.Get("/search", SearchMonster)
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
