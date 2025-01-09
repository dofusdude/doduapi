package database

import (
	"github.com/hashicorp/go-memdb"
	"github.com/meilisearch/meilisearch-go"
)

type SearchIndexes struct {
	AllItems meilisearch.IndexManager
	Sets     meilisearch.IndexManager
	Mounts   meilisearch.IndexManager
}

var Db *memdb.MemDB
var Indexes map[string]SearchIndexes

type VersionT struct {
	Search bool
	MemDb  bool
}

var Version VersionT
