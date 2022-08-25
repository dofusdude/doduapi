package server

import (
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

	r.With(languageChecker).Route("/dofus/{lang}", func(r chi.Router) {
		r.Route("/items", func(r chi.Router) {
			r.Route("/consumables", func(r chi.Router) {
				r.With(paginate).Get("/", ListConsumables)
				r.With(ankamaIdExtractor).Get("/{ankamaId}", GetSingleConsumableHandler)
				r.Get("/search", SearchConsumables)
			})

			r.Route("/resources", func(r chi.Router) {
				r.With(paginate).Get("/", ListResources)
				r.Get("/{ankamaId}", ListConsumables)
				r.Get("/search", SearchResources)
			})

			r.Route("/equipment", func(r chi.Router) {
				r.With(paginate).Get("/", ListEquipment)
				r.Get("/{ankamaId}", ListConsumables)
				r.Get("/search", SearchEquipment)
			})

			r.Route("/quest", func(r chi.Router) {
				r.With(paginate).Get("/", ListQuestItems)
				r.Get("/{ankamaId}", ListConsumables)
				r.Get("/search", SearchQuestItems)
			})

			r.Route("/cosmetics", func(r chi.Router) {
				r.With(paginate).Get("/", ListCosmetics)
				r.Get("/{ankamaId}", ListConsumables)
				r.Get("/search", SearchCosmetics)
			})

			r.Get("/search", SearchAllItems)

		})

		r.Route("/mounts", func(r chi.Router) {
			r.With(paginate).Get("/", ListConsumables)
			r.Get("/{ankamaId}", ListConsumables)
			r.Get("/search", SearchAllItems)
		})

		r.Route("/sets", func(r chi.Router) {
			r.With(paginate).Get("/", ListConsumables)
			r.Get("/{ankamaId}", ListConsumables)
			r.Get("/search", SearchAllItems)
		})
	})

	return r
}
