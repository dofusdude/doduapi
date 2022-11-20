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

	requestsItemsSearch = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllItemsSearch",
		Help: "The total number of searched items requests",
	})

	requestsMountsSearch = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllMountsSearch",
		Help: "The total number of searched mounts requests",
	})

	requestsMonsterSearch = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllMonsterSearch",
		Help: "The total number of searched monster requests",
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

	requestsMonsterList = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllMonsterList",
		Help: "The total number of list monster requests",
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

	requestsMonsterSingle = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllMonsterSingle",
		Help: "The total number of single monster requests",
	})

	requestsSetsSingle = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dofus_requestsAllSetsSingle",
		Help: "The total number of single set requests",
	})
)
