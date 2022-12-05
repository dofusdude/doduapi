package gen

import (
	"fmt"
	"github.com/dofusdude/api/utils"
)

func MapSets(data *JSONGameData, langs *map[string]LangDict) []MappedMultilangSet {
	var mappedSets []MappedMultilangSet
	for _, set := range data.Sets {
		var mappedSet MappedMultilangSet
		mappedSet.AnkamaId = set.Id
		mappedSet.ItemIds = set.ItemIds
		mappedSet.Effects = ParseEffects(data, set.Effects, langs)

		highestLevel := 0
		for _, item := range set.ItemIds {
			if data.Items[item].Level > highestLevel {
				highestLevel = data.Items[item].Level
			}
		}
		mappedSet.Level = highestLevel

		mappedSet.Name = make(map[string]string)
		for _, lang := range utils.Languages {
			mappedSet.Name[lang] = (*langs)[lang].Texts[set.NameId]
		}

		mappedSets = append(mappedSets, mappedSet)
	}

	if len(mappedSets) == 0 {
		return nil
	}

	return mappedSets
}

func MapRecipes(data *JSONGameData) []MappedMultilangRecipe {
	mappedRecipes := make([]MappedMultilangRecipe, len(data.Recipes))

	for idx, recipe := range data.Recipes {
		ingredientCount := len(recipe.IngredientIds)

		mappedRecipes[idx].ResultId = recipe.Id
		mappedRecipes[idx].Entries = make([]MappedMultilangRecipeEntry, ingredientCount)
		for i := 0; i < ingredientCount; i++ {
			var recipeEntry MappedMultilangRecipeEntry
			recipeEntry.ItemId = recipe.IngredientIds[i]
			recipeEntry.Quantity = recipe.Quantities[i]
			mappedRecipes[idx].Entries[i] = recipeEntry
		}
	}

	if len(mappedRecipes) == 0 {
		return nil
	}

	return mappedRecipes
}

func MapMounts(data *JSONGameData, langs *map[string]LangDict) []MappedMultilangMount {
	mappedMounts := make([]MappedMultilangMount, len(data.Mounts))
	for idx, mount := range data.Mounts {
		mappedMounts[idx].AnkamaId = mount.Id
		mappedMounts[idx].FamilyId = mount.FamilyId
		mappedMounts[idx].Name = make(map[string]string)
		mappedMounts[idx].FamilyName = make(map[string]string)

		for _, lang := range utils.Languages {
			mappedMounts[idx].Name[lang] = (*langs)[lang].Texts[mount.NameId]
			mappedMounts[idx].FamilyName[lang] = (*langs)[lang].Texts[data.Mount_familys[mount.FamilyId].NameId]
		}

		allEffectResult := ParseEffects(data, [][]JSONGameItemPossibleEffect{mount.Effects}, langs)
		if allEffectResult != nil && len(allEffectResult) > 0 {
			mappedMounts[idx].Effects = allEffectResult[0]
		}
	}

	if len(mappedMounts) == 0 {
		return nil
	}

	return mappedMounts
}

func MapItems(data *JSONGameData, langs *map[string]LangDict) []MappedMultilangItem {
	mappedItems := make([]MappedMultilangItem, len(data.Items))

	for idx, item := range data.Items {
		if (*langs)["fr"].Texts[item.NameId] == "" || data.ItemTypes[item.TypeId].CategoryId == 4 || (*langs)["de"].Texts[data.ItemTypes[item.TypeId].NameId] == "Hauptquesten" {
			continue // skip unnamed and hidden items
		}

		mappedItems[idx].AnkamaId = item.Id
		mappedItems[idx].Level = item.Level
		mappedItems[idx].Pods = item.Pods
		mappedItems[idx].Image = fmt.Sprintf("https://static.ankama.com/dofus/www/game/items/200/%d.png", item.IconId)
		mappedItems[idx].Name = make(map[string]string, len(utils.Languages))
		mappedItems[idx].Description = make(map[string]string, len(utils.Languages))
		mappedItems[idx].Type.Name = make(map[string]string, len(utils.Languages))
		mappedItems[idx].IconId = item.IconId

		for _, lang := range utils.Languages {
			mappedItems[idx].Name[lang] = (*langs)[lang].Texts[item.NameId]
			mappedItems[idx].Description[lang] = (*langs)[lang].Texts[item.DescriptionId]
			mappedItems[idx].Type.Name[lang] = (*langs)[lang].Texts[data.ItemTypes[item.TypeId].NameId]
		}

		mappedItems[idx].Type.Id = item.TypeId
		mappedItems[idx].Type.SuperTypeId = data.ItemTypes[item.TypeId].SuperTypeId
		mappedItems[idx].Type.CategoryId = data.ItemTypes[item.TypeId].CategoryId

		mappedItems[idx].UsedInRecipes = item.RecipeIds
		allEffectResult := ParseEffects(data, [][]JSONGameItemPossibleEffect{item.PossibleEffects}, langs)
		if allEffectResult != nil && len(allEffectResult) > 0 {
			mappedItems[idx].Effects = allEffectResult[0]
		}
		mappedItems[idx].Range = item.Range
		mappedItems[idx].MinRange = item.MinRange
		mappedItems[idx].CriticalHitProbability = item.CriticalHitProbability
		mappedItems[idx].CriticalHitBonus = item.CriticalHitBonus
		mappedItems[idx].ApCost = item.ApCost
		mappedItems[idx].TwoHanded = item.TwoHanded
		mappedItems[idx].MaxCastPerTurn = item.MaxCastPerTurn
		mappedItems[idx].DropMonsterIds = item.DropMonsterIds
		mappedItems[idx].HasParentSet = item.ItemSetId != -1
		if mappedItems[idx].HasParentSet {
			mappedItems[idx].ParentSet.Id = item.ItemSetId
			mappedItems[idx].ParentSet.Name = make(map[string]string, len(utils.Languages))
			for _, lang := range utils.Languages {
				mappedItems[idx].ParentSet.Name[lang] = (*langs)[lang].Texts[data.Sets[item.ItemSetId].NameId]
			}
		}

		if len(item.Criteria) != 0 && mappedItems[idx].Type.Name["de"] != "Verwendbarer Temporis-Gegenstand" { // TODO Temporis got some weird conditions, need to play to see the items, not in normal game
			mappedItems[idx].Conditions = ParseCondition(item.Criteria, langs, data)
		}
	}

	return mappedItems
}
