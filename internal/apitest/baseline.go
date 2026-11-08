package apitest

import "fmt"

type CollectionBaseline struct {
	MinimumRecords int   `json:"minimum_records"`
	RequiredIDs    []int `json:"required_ids"`
}
type Baseline struct {
	Version             int                           `json:"version"`
	Collections         map[string]CollectionBaseline `json:"collections"`
	MinimumRecipes      int                           `json:"minimum_recipes"`
	MinimumAlmanax      int                           `json:"minimum_almanax"`
	MinimumCalendarDays int                           `json:"minimum_calendar_days"`
}

func Snapshot(data *Data) Baseline {
	result := Baseline{Version: 1, Collections: map[string]CollectionBaseline{}, MinimumRecipes: len(data.Recipes), MinimumAlmanax: len(data.Almanax)}
	result.MinimumCalendarDays = len(data.Calendar)
	for _, collection := range data.Collections {
		result.Collections[collection.Path] = CollectionBaseline{MinimumRecords: len(collection.Records), RequiredIDs: IDs(collection.Records[:min(3, len(collection.Records))], nil)}
	}
	return result
}

func (baseline Baseline) Check(data *Data) error {
	if baseline.Version != 1 {
		return fmt.Errorf("unsupported data-baseline version %d", baseline.Version)
	}
	for _, collection := range data.Collections {
		want, ok := baseline.Collections[collection.Path]
		if !ok {
			return fmt.Errorf("baseline missing collection %s", collection.Path)
		}
		if len(collection.Records) < want.MinimumRecords {
			return fmt.Errorf("%s lost data: expected at least %d records, got %d; review before updating the baseline", collection.Path, want.MinimumRecords, len(collection.Records))
		}
		ids := map[int]bool{}
		for _, r := range collection.Records {
			ids[r.ID] = true
		}
		for _, id := range want.RequiredIDs {
			if !ids[id] {
				return fmt.Errorf("%s lost required ID %d", collection.Path, id)
			}
		}
	}
	if len(data.Recipes) < baseline.MinimumRecipes || len(data.Almanax) < baseline.MinimumAlmanax || len(data.Calendar) < baseline.MinimumCalendarDays {
		return fmt.Errorf("mapped recipe/Almanax data shrank below its reviewed baseline")
	}
	return nil
}
