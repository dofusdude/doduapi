package main

import (
	"encoding/json"
	"net/http"
)

func ListSearchAllTypes(w http.ResponseWriter, r *http.Request) {
	WriteCacheHeader(&w)
	if err := json.NewEncoder(w).Encode(searchAllowedIndices); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func GetGameVersion(w http.ResponseWriter, r *http.Request) {
	WriteCacheHeader(&w)
	if err := json.NewEncoder(w).Encode(CurrentVersion); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func ListItemTypeIds(w http.ResponseWriter, r *http.Request) {
	txn := Db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get("item-type-ids", "id")
	if err != nil || it == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var typeIds []string
	for obj := it.Next(); obj != nil; obj = it.Next() {
		effectElement := obj.(*ItemTypeId)
		typeIds = append(typeIds, effectElement.EnName)
	}

	WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(typeIds)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func ListEffectConditionElements(w http.ResponseWriter, r *http.Request) {
	txn := Db.Txn(false)
	defer txn.Abort()

	it, err := txn.Get("effect-condition-elements", "id")
	if err != nil || it == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var effects []string
	for obj := it.Next(); obj != nil; obj = it.Next() {
		effectElement := obj.(*EffectConditionDbEntry)
		effects = append(effects, effectElement.Name)
	}

	WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(effects)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
