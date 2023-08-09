package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

func disablePaginate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), "pagination", "1,-1")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func paginate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageNumStr := r.URL.Query().Get("page[number]")
		var pageNum int

		pageSizeStr := r.URL.Query().Get("page[size]")
		var pageSize int

		var err error
		if pageNumStr != "" {
			pageNum, err = strconv.Atoi(pageNumStr)
			if err != nil || pageNum <= 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		} else {
			pageNum = 1
		}

		if pageSizeStr != "" {
			pageSize, err = strconv.Atoi(pageSizeStr)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		} else {
			pageSize = 16
		}

		ctx := context.WithValue(r.Context(), "pagination", fmt.Sprintf("%d,%d", pageNum, pageSize))

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func useCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		ctx := context.WithValue(r.Context(), "cors", true)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func languageChecker(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := strings.ToLower(chi.URLParam(r, "lang"))
		switch lang {
		case "en", "fr", "de", "es", "it", "pt":
			ctx := context.WithValue(r.Context(), "lang", lang)
			next.ServeHTTP(w, r.WithContext(ctx))
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	})
}

func ankamaIdExtractor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ankamaId, err := strconv.Atoi(chi.URLParam(r, "ankamaId"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), "ankamaId", ankamaId)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
