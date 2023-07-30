package server

import (
	"encoding/json"
	"net/http"

	"github.com/dofusdude/api/gen"
	"github.com/dofusdude/api/utils"
)

func ListSearchAllTypes(w http.ResponseWriter, r *http.Request) {
	utils.WriteCacheHeader(&w)
	if err := json.NewEncoder(w).Encode(searchAllowedIndices); err != nil {
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
		effectElement := obj.(*gen.EffectConditionDbEntry)
		effects = append(effects, effectElement.Name)
	}

	utils.WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(effects)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
