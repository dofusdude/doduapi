package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/dofusdude/doduapi/database"
	e "github.com/dofusdude/doduapi/errmsg"
	requestquery "github.com/dofusdude/doduapi/query"
	"github.com/dofusdude/doduapi/utils"
	"github.com/meilisearch/meilisearch-go"
)

func includeFullMappedEntity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), "mapped_entity_full", true)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func mappedEntityFull(r *http.Request) bool {
	full, _ := r.Context().Value("mapped_entity_full").(bool)
	return full
}

func mappedEntityOptions(resource MappedEntityResource, r *http.Request) (MappedEntityRenderOptions, error) {
	options := MappedEntityRenderOptions{}
	include := strings.TrimSpace(r.URL.Query().Get("include"))
	if include != "" {
		if resource.Route != "monsters" {
			return options, fmt.Errorf("include is only supported for monsters")
		}
		if include != monsterSpellsFull && include != monsterSpellsGradeOnly {
			return options, fmt.Errorf("include must be %q or %q", monsterSpellsFull, monsterSpellsGradeOnly)
		}
		options.MonsterSpellInclude = include
	}
	gradeValue := strings.TrimSpace(r.URL.Query().Get("grade"))
	if gradeValue != "" {
		if resource.Route != "spells" {
			return options, fmt.Errorf("grade is only supported for a single spell")
		}
		grade, err := strconv.Atoi(gradeValue)
		if err != nil || grade <= 0 {
			return options, fmt.Errorf("grade must be a positive integer")
		}
		options.SpellGrade = grade
	}
	return options, nil
}

func ListMappedEntities(resource MappedEntityResource, w http.ResponseWriter, r *http.Request) {
	language := r.Context().Value("lang").(string)
	pagination := requestPagination(r)
	query, _ := r.Context().Value(entityQueryContextKey{}).(*EntityQuery)
	full := mappedEntityFull(r)
	options := MappedEntityRenderOptions{}
	if len(query.Include) > 0 {
		options.MonsterSpellInclude = query.Include[0]
	}
	if len(query.Fields[resource.Table]) > 0 {
		full = true
	}

	txn := database.Db.Txn(false)
	defer txn.Abort()
	iterator, err := txn.Get(fmt.Sprintf("%s-%s", utils.CurrentRedBlueVersionStr(database.Version.MemDb), resource.Table), "id")
	if err != nil || iterator == nil {
		e.WriteNotFoundResponse(w, fmt.Sprintf("No %s found.", resource.Route))
		return
	}

	records := make([]*MappedEntityRecord, 0)
	for object := iterator.Next(); object != nil; object = iterator.Next() {
		if err := r.Context().Err(); err != nil {
			return
		}
		record := object.(*MappedEntityRecord)
		if query.Filter.matches(record, language) {
			records = append(records, record)
		}
	}
	sortEntityQuery(records, query.Sort, language)
	if len(records) == 0 && !query.Active {
		e.WriteNotFoundResponse(w, fmt.Sprintf("No %s found.", resource.Route))
		return
	}
	if !(query.Active && len(records) == 0 && pagination.PageNumber == 1) && pagination.ValidatePagination(len(records)) != 0 {
		e.WriteInvalidQueryResponse(w, "Invalid pagination parameters.")
		return
	}

	start, end := pagination.CalculateStartEndIndex(len(records))
	links := collectionPaginationLinks(r, len(records), pagination)
	if end-start > 100 && links == (utils.PaginationLinks{}) {
		streamMappedEntities(resource, records[start:end], language, full, options, w, r)
		return
	}
	items := make([]any, 0, end-start)
	for _, record := range records[start:end] {
		rendered, renderErr := renderMappedEntityWithOptions(record, language, full, options)
		if renderErr != nil {
			e.WriteServerErrorResponse(w, "Could not render mapped entity: "+renderErr.Error())
			return
		}
		items = append(items, rendered)
	}
	response := map[string]any{resource.ResponseKey: items}
	if links != (utils.PaginationLinks{}) {
		response["_links"] = links
	}

	utils.RequestsTotal.Inc()
	utils.WriteCacheHeader(&w)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		e.WriteServerErrorResponse(w, "Could not encode JSON: "+err.Error())
	}
}

// Bulk exports must not retain every expanded monster/spell in memory before
// sending the first byte. Keep the same collection schema, one record at a time.
func streamMappedEntities(resource MappedEntityResource, records []*MappedEntityRecord, language string, full bool, options MappedEntityRenderOptions, w http.ResponseWriter, r *http.Request) {
	utils.RequestsTotal.Inc()
	utils.WriteCacheHeader(&w)
	w.Header().Set("Content-Type", "application/json")
	buffer := bufio.NewWriterSize(w, 64*1024)
	fmt.Fprintf(buffer, "{%q:[", resource.ResponseKey)
	encoder := json.NewEncoder(buffer)
	for index, record := range records {
		if r.Context().Err() != nil {
			return
		}
		rendered, err := renderMappedEntityWithOptions(record, language, full, options)
		if err != nil {
			log.Error("Could not render bulk entity", "resource", resource.Route, "id", record.AnkamaId, "err", err)
			return // Headers may already be sent; never append an error object to JSON.
		}
		if index > 0 {
			buffer.WriteByte(',')
		}
		if encoder.Encode(rendered) != nil {
			return
		}
	}
	buffer.WriteString("]}\n")
	buffer.Flush()
}

func GetSingleMappedEntity(resource MappedEntityResource, w http.ResponseWriter, r *http.Request) {
	language := r.Context().Value("lang").(string)
	ankamaID := r.Context().Value("ankamaId").(int)
	options, optionsErr := mappedEntityOptions(resource, r)
	if optionsErr != nil {
		e.WriteInvalidQueryResponse(w, optionsErr.Error())
		return
	}
	txn := database.Db.Txn(false)
	defer txn.Abort()
	raw, err := txn.First(fmt.Sprintf("%s-%s", utils.CurrentRedBlueVersionStr(database.Version.MemDb), resource.Table), "id", ankamaID)
	if err != nil {
		e.WriteServerErrorResponse(w, "Could not read database: "+err.Error())
		return
	}
	if raw == nil {
		e.WriteNotFoundResponse(w, fmt.Sprintf("Could not find %s with ID %d.", resource.Route, ankamaID))
		return
	}
	rendered, err := renderMappedEntityWithOptions(raw.(*MappedEntityRecord), language, true, options)
	if err != nil {
		if resource.Route == "spells" && options.SpellGrade > 0 {
			e.WriteNotFoundResponse(w, err.Error())
			return
		}
		e.WriteServerErrorResponse(w, "Could not render mapped entity: "+err.Error())
		return
	}
	utils.RequestsTotal.Inc()
	utils.WriteCacheHeader(&w)
	if err := json.NewEncoder(w).Encode(rendered); err != nil {
		e.WriteServerErrorResponse(w, "Could not encode JSON: "+err.Error())
	}
}

func SearchMappedEntities(resource MappedEntityResource, w http.ResponseWriter, r *http.Request) {
	query := requestquery.From(r).String("query")
	if query == "" {
		e.WriteInvalidQueryResponse(w, "Query parameter is required.")
		return
	}
	limit := int64(requestquery.From(r).IntDefault("limit", 8))
	language := r.Context().Value("lang").(string)
	indexes, ok := database.Indexes[language]
	if !ok || indexes.Entities == nil || indexes.Entities[resource.Table] == nil {
		e.WriteServerErrorResponse(w, "Search index is unavailable.")
		return
	}
	response, err := indexes.Entities[resource.Table].SearchWithContext(r.Context(), query, &meilisearch.SearchRequest{Limit: limit})
	if err != nil {
		e.WriteServerErrorResponse(w, "Could not search: "+err.Error())
		return
	}
	if response.EstimatedTotalHits == 0 {
		e.WriteNotFoundResponse(w, "No results found.")
		return
	}

	txn := database.Db.Txn(false)
	defer txn.Abort()
	results := make([]APIEntityListEntry, 0, len(response.Hits))
	for _, hit := range response.Hits {
		var indexed SearchIndexedEntity
		if err := hit.DecodeInto(&indexed); err != nil {
			e.WriteServerErrorResponse(w, "Could not decode search result: "+err.Error())
			return
		}
		raw, err := txn.First(fmt.Sprintf("%s-%s", utils.CurrentRedBlueVersionStr(database.Version.MemDb), resource.Table), "id", indexed.Id)
		if err != nil || raw == nil {
			continue
		}
		rendered, _ := renderMappedEntity(raw.(*MappedEntityRecord), language, false)
		results = append(results, rendered.(APIEntityListEntry))
	}
	utils.RequestsTotal.Inc()
	utils.WriteCacheHeader(&w)
	if err := json.NewEncoder(w).Encode(results); err != nil {
		e.WriteServerErrorResponse(w, "Could not encode JSON: "+err.Error())
	}
}
