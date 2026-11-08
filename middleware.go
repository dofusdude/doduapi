package main

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	e "github.com/dofusdude/doduapi/errmsg"
	"github.com/dofusdude/doduapi/query"
	"github.com/dofusdude/doduapi/utils"
	"github.com/go-chi/chi/v5"
)

type paginationKey struct{}

func requestPagination(r *http.Request) utils.Pagination {
	return r.Context().Value(paginationKey{}).(utils.Pagination)
}

func disablePaginate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), paginationKey{}, utils.Pagination{PageNumber: 1, PageSize: -1})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func paginate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := utils.Pagination{PageNumber: 1, PageSize: 16}
		if entity, ok := r.Context().Value(entityQueryContextKey{}).(*EntityQuery); ok {
			page.PageNumber, page.PageSize = entity.Page.Number, entity.Page.Size
		} else {
			request := query.From(r)
			page.PageNumber = request.IntDefault("page[number]", 1)
			page.PageSize = request.IntDefault("page[size]", 16)
		}
		ctx := context.WithValue(r.Context(), paginationKey{}, page)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func useCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Expose-Headers", "Accept-Query, Allow")
		if r.Method == http.MethodOptions && r.Header.Get("Origin") != "" && r.Header.Get("Access-Control-Request-Method") != "" {
			method := r.Header.Get("Access-Control-Request-Method")
			_, querySupported := query.Match(r.URL.Path)
			if method != "GET" && method != "HEAD" && (method != "QUERY" || !querySupported) {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			for _, header := range strings.Split(r.Header.Get("Access-Control-Request-Headers"), ",") {
				switch strings.ToLower(strings.TrimSpace(header)) {
				case "", "content-type", "accept":
				default:
					w.WriteHeader(http.StatusForbidden)
					return
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, QUERY, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
			w.Header().Set("Access-Control-Max-Age", "600")
			w.Header().Add("Vary", "Access-Control-Request-Method")
			w.Header().Add("Vary", "Access-Control-Request-Headers")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		ctx := context.WithValue(r.Context(), "cors", true)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func languageChecker(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := strings.ToLower(chi.URLParam(r, "lang"))
		switch lang {
		case "en", "fr", "de", "es", "pt":
			ctx := context.WithValue(r.Context(), "lang", lang)
			next.ServeHTTP(w, r.WithContext(ctx))
		default:
			e.WriteInvalidUrlResponse(w, "Invalid language: "+chi.URLParam(r, "lang"))
		}
	})
}

func ankamaIdExtractor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ankamaId, err := strconv.Atoi(chi.URLParam(r, "ankamaId"))
		if err != nil {
			e.WriteInvalidUrlResponse(w, "Invalid ankamaId: "+chi.URLParam(r, "ankamaId"))
			return
		}
		ctx := context.WithValue(r.Context(), "ankamaId", ankamaId)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func dateExtractor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parsedDate, err := time.Parse("2006-01-02", chi.URLParam(r, "date"))
		if err != nil {
			e.WriteInvalidUrlResponse(w, "Invalid date format: "+chi.URLParam(r, "date"))
			return
		}

		ctx := context.WithValue(r.Context(), "date", parsedDate)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
