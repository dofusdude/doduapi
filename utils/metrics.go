package utils

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RequestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsTotal",
		Help: "The total number of processed requests over all endpoints",
	})

	RequestsSearchTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsSearchTotal",
		Help: "The total number of searches on the global /search endpoint",
	})

	RequestsItemsSearch = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllItemsSearch",
		Help: "The total number of searched items requests",
	})

	RequestsMountsSearch = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllMountsSearch",
		Help: "The total number of searched mounts requests",
	})

	RequestsSetsSearch = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllSetsSearch",
		Help: "The total number of searched sets requests",
	})

	RequestsItemsList = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllItemsList",
		Help: "The total number of list items requests",
	})

	RequestsMountsList = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllMountsList",
		Help: "The total number of list mounts requests",
	})

	RequestsSetsList = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllSetsList",
		Help: "The total number list sets requests",
	})

	RequestsItemsSingle = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllItemsSingle",
		Help: "The total number of single item requests",
	})

	RequestsMountsSingle = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllMountsSingle",
		Help: "The total number of single mount requests",
	})

	RequestsSetsSingle = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllSetsSingle",
		Help: "The total number of single set requests",
	})

	RequestsAlmanaxSingle = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAlmanaxSingle",
		Help: "The total number of single almanax requests",
	})

	RequestsAlmanaxRange = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAlmanaxRange",
		Help: "The total number of almanax range requests",
	})
)
