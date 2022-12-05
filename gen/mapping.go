package gen

import (
	"fmt"
	"github.com/dofusdude/api/utils"
)

func MapSets(data JSONGameData, langs map[string]LangDict) []MappedMultilangSet {
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
			mappedSet.Name[lang] = (langs)[lang].Texts[set.NameId]
		}

		mappedSets = append(mappedSets, mappedSet)
	}

	if len(mappedSets) == 0 {
		return nil
	}

	return mappedSets
}

func MapRecipes(data JSONGameData) []MappedMultilangRecipe {
	var mappedRecipes []MappedMultilangRecipe
	for _, recipe := range data.Recipes {
		ingredientCount := len(recipe.IngredientIds)

		var mappedRecipe MappedMultilangRecipe
		mappedRecipe.ResultId = recipe.Id
		for i := 0; i < ingredientCount; i++ {
			var recipeEntry MappedMultilangRecipeEntry
			recipeEntry.ItemId = recipe.IngredientIds[i]
			recipeEntry.Quantity = recipe.Quantities[i]
			mappedRecipe.Entries = append(mappedRecipe.Entries, recipeEntry)
		}
		mappedRecipes = append(mappedRecipes, mappedRecipe)
	}

	if len(mappedRecipes) == 0 {
		return nil
	}

	return mappedRecipes
}

func MapMounts(data JSONGameData, langs map[string]LangDict) []MappedMultilangMount {
	var mappedMounts []MappedMultilangMount
	for _, mount := range data.Mounts {
		var mappedMount MappedMultilangMount
		mappedMount.AnkamaId = mount.Id
		mappedMount.FamilyId = mount.FamilyId
		mappedMount.Name = make(map[string]string)
		mappedMount.FamilyName = make(map[string]string)

		for _, lang := range utils.Languages {
			mappedMount.Name[lang] = (langs)[lang].Texts[mount.NameId]
			mappedMount.FamilyName[lang] = (langs)[lang].Texts[data.Mount_familys[mount.FamilyId].NameId]
		}

		allEffectResult := ParseEffects(data, [][]JSONGameItemPossibleEffect{mount.Effects}, langs)
		if allEffectResult != nil && len(allEffectResult) > 0 {
			mappedMount.Effects = allEffectResult[0]
		}

		mappedMounts = append(mappedMounts, mappedMount)
	}

	if len(mappedMounts) == 0 {
		return nil
	}

	return mappedMounts
}

func MapItems(data JSONGameData, langs map[string]LangDict) []MappedMultilangItem {
	var mappedItems []MappedMultilangItem
	for _, item := range data.Items {
		if (langs)["fr"].Texts[item.NameId] == "" || data.ItemTypes[item.TypeId].CategoryId == 4 || (langs)["de"].Texts[data.ItemTypes[item.TypeId].NameId] == "Hauptquesten" {
			continue // skip unnamed and hidden items
		}

		var mappedItem MappedMultilangItem
		mappedItem.AnkamaId = item.Id
		mappedItem.Level = item.Level
		mappedItem.Pods = item.Pods
		mappedItem.Image = fmt.Sprintf("https://static.ankama.com/dofus/www/game/items/200/%d.png", item.IconId)

		mappedItem.Name = make(map[string]string, len(utils.Languages))
		mappedItem.Description = make(map[string]string, len(utils.Languages))
		mappedItem.Type.Name = make(map[string]string, len(utils.Languages))
		mappedItem.IconId = item.IconId

		for _, lang := range utils.Languages {
			mappedItem.Name[lang] = (langs)[lang].Texts[item.NameId]
			mappedItem.Description[lang] = (langs)[lang].Texts[item.DescriptionId]
			mappedItem.Type.Name[lang] = (langs)[lang].Texts[data.ItemTypes[item.TypeId].NameId]
		}

		mappedItem.Type.Id = item.TypeId
		mappedItem.Type.SuperTypeId = data.ItemTypes[item.TypeId].SuperTypeId
		mappedItem.Type.CategoryId = data.ItemTypes[item.TypeId].CategoryId

		mappedItem.UsedInRecipes = item.RecipeIds
		allEffectResult := ParseEffects(data, [][]JSONGameItemPossibleEffect{item.PossibleEffects}, langs)
		if allEffectResult != nil && len(allEffectResult) > 0 {
			mappedItem.Effects = allEffectResult[0]
		}
		mappedItem.Range = item.Range
		mappedItem.MinRange = item.MinRange
		mappedItem.CriticalHitProbability = item.CriticalHitProbability
		mappedItem.CriticalHitBonus = item.CriticalHitBonus
		mappedItem.ApCost = item.ApCost
		mappedItem.TwoHanded = item.TwoHanded
		mappedItem.MaxCastPerTurn = item.MaxCastPerTurn
		mappedItem.HasParentSet = item.ItemSetId != -1
		if mappedItem.HasParentSet {
			mappedItem.ParentSet.Id = item.ItemSetId
			mappedItem.ParentSet.Name = make(map[string]string, len(utils.Languages))
			for _, lang := range utils.Languages {
				mappedItem.ParentSet.Name[lang] = (langs)[lang].Texts[data.Sets[item.ItemSetId].NameId]
			}
		}

		if len(item.Criteria) != 0 && mappedItem.Type.Name["de"] != "Verwendbarer Temporis-Gegenstand" { // TODO Temporis got some weird conditions, need to play to see the items, not in normal game
			mappedItem.Conditions = ParseCondition(item.Criteria, langs, data)
		}

		mappedItems = append(mappedItems, mappedItem)

		mappedItem.DropMonsterIds = item.DropMonsterIds
	}

	return mappedItems
}
