package server

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsTotal",
		Help: "The total number of processed requests over all endpoints",
	})

	requestsSearchTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsSearchTotal",
		Help: "The total number of searches on the global /search endpoint",
	})

	requestsItemsSearch = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllItemsSearch",
		Help: "The total number of searched items requests",
	})

	requestsMountsSearch = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllMountsSearch",
		Help: "The total number of searched mounts requests",
	})

	requestsSetsSearch = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllSetsSearch",
		Help: "The total number of searched sets requests",
	})

	requestsItemsList = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllItemsList",
		Help: "The total number of list items requests",
	})

	requestsMountsList = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllMountsList",
		Help: "The total number of list mounts requests",
	})

	requestsSetsList = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllSetsList",
		Help: "The total number list sets requests",
	})

	requestsItemsSingle = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllItemsSingle",
		Help: "The total number of single item requests",
	})

	requestsMountsSingle = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllMountsSingle",
		Help: "The total number of single mount requests",
	})

	requestsSetsSingle = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllSetsSingle",
		Help: "The total number of single set requests",
	})
)
