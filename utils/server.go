package utils

import (
	"net/http"
	"time"
)

func SetJsonHeader(w *http.ResponseWriter) {
	(*w).Header().Set("Content-Type", "application/json")
}

func WriteCacheHeader(w *http.ResponseWriter) {
	SetJsonHeader(w)
	//(*w).Header().Set("Cache-Control", "max-age:300, public")
	//(*w).Header().Set("Last-Modified", LastUpdate.Format(http.TimeFormat))
	//(*w).Header().Set("Expires", time.Now().Add(time.Minute*5).Format(http.TimeFormat))
}

type GameVersion struct {
	Version     string    `json:"version"`
	Release     string    `json:"release"`
	UpdateStamp time.Time `json:"update_stamp"`
}
