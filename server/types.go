package server

import (
	"github.com/dofusdude/api/gen"
	"github.com/dofusdude/api/utils"
)

type ApiEffect struct {
	Min              int    `json:"min"`
	Max              int    `json:"max"`
	Type             string `json:"type"`
	MinMaxIrrelevant int    `json:"min_max_irrelevant"`
	Templated        string `json:"templated"`
}

func RenderEffects(effects *[]gen.MappedMultilangEffect, lang string) []ApiEffect {
	var ret_effects []ApiEffect
	for _, effect := range *effects {
		ret_effects = append(ret_effects, ApiEffect{
			Min:              effect.Min,
			Max:              effect.Max,
			MinMaxIrrelevant: effect.MinMaxIrrelevant,
			Type:             effect.Type[lang],
			Templated:        effect.Templated[lang],
		})
	}

	if len(ret_effects) > 0 {
		return ret_effects
	}

	return nil
}

type APIConsumable struct {
	Id            int         `json:"ankama_id"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	Type          string      `json:"type"`
	Level         int         `json:"level"`
	Pods          int         `json:"pods"`
	ImageUrl      string      `json:"image_url"`
	Effects       []ApiEffect `json:"effects"`
	Conditions   []string    `json:"conditions"`
	Recipe        []APIRecipe  `json:"recipe"`
}

func RenderConsumable(item *gen.MappedMultilangItem, lang string) APIConsumable {
	
	return APIConsumable{
		Id:            item.AnkamaId,
		Name:          item.Name[lang],
		Type:          item.Type.Name[lang],
		Description:   item.Description[lang],
		Level:         item.Level,
		Pods:          item.Pods,
		ImageUrl:      utils.ItemImageUrl(item.AnkamaId),
		Effects:       RenderEffects(&item.Effects, lang),
		Conditions:   RenderConditions(&item.Conditions, lang),
		Recipe:        nil,
	}
}

type APIResource struct {
	Id            int         `json:"ankama_id"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	Type          string      `json:"type"`
	Level         int         `json:"level"`
	Pods          int         `json:"pods"`
	ImageUrl      string      `json:"image_url"`
	Effects       []ApiEffect `json:"effects"`
	Conditions   []string    `json:"conditions"`
	Recipe        []APIRecipe  `json:"recipe"`
}

func RenderResource(item *gen.MappedMultilangItem, lang string) APIResource {
	return APIResource{
		Id:            item.AnkamaId,
		Name:          item.Name[lang],
		Type:          item.Type.Name[lang],
		Description:   item.Description[lang],
		Level:         item.Level,
		Pods:          item.Pods,
		ImageUrl:      utils.ItemImageUrl(item.AnkamaId),
		Effects:       RenderEffects(&item.Effects, lang),
		Recipe:        nil,
	}
}

type APIEquipment struct {
	Id            int         `json:"ankama_id"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	Type          string      `json:"type"`
	IsWeapon 	bool        `json:"is_weapon"`
	Level         int         `json:"level"`
	Pods          int         `json:"pods"`
	ImageUrl      string      `json:"image_url"`
	Effects       []ApiEffect `json:"effects"`
	Conditions   []string    `json:"conditions"`
	Recipe        []APIRecipe  `json:"recipe"`
}

func RenderEquipment(item *gen.MappedMultilangItem, lang string) APIEquipment {
	return APIEquipment{
		Id:            item.AnkamaId,
		Name:          item.Name[lang],
		Type:          item.Type.Name[lang],
		Description:   item.Description[lang],
		Level:         item.Level,
		Pods:          item.Pods,
		ImageUrl:      utils.ItemImageUrl(item.AnkamaId),
		Effects:       RenderEffects(&item.Effects, lang),
		Conditions: RenderConditions(&item.Conditions, lang),
		IsWeapon: false,
		Recipe:        nil,
	}
}

type APIWeapon struct {
	Id            int         `json:"ankama_id"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	Type          string      `json:"type"`
	IsWeapon 	bool        `json:"is_weapon"`
	Level         int         `json:"level"`
	Pods          int         `json:"pods"`
	ImageUrl      string      `json:"image_url"`
	Effects       []ApiEffect `json:"effects"`
	Conditions   []string    `json:"conditions"`
	CriticalHitProbability int                             `json:"critical_hit_probability"`
	CriticalHitBonus       int                             `json:"critical_hit_bonus"`
	TwoHanded              bool                            `json:"is_two_handed"`
	MaxCastPerTurn         int                             `json:"max_cast_per_turn"`
	ApCost                 int                             `json:"ap_cost"`
	Range                  int                             `json:"range"`
	Recipe        []APIRecipe  `json:"recipe"`
}

func RenderWeapon(item *gen.MappedMultilangItem, lang string) APIWeapon {
	return APIWeapon{
		Id:            item.AnkamaId,
		Name:          item.Name[lang],
		Type:          item.Type.Name[lang],
		Description:   item.Description[lang],
		Level:         item.Level,
		Pods:          item.Pods,
		ImageUrl:      utils.ItemImageUrl(item.AnkamaId),
		Effects:       RenderEffects(&item.Effects, lang),
		Conditions: RenderConditions(&item.Conditions, lang),
		Recipe:        nil,
		CriticalHitBonus: item.CriticalHitBonus,
		CriticalHitProbability: item.CriticalHitProbability,
		TwoHanded: item.TwoHanded,
		MaxCastPerTurn: item.MaxCastPerTurn,
		ApCost: item.ApCost,
		Range: item.Range,
		IsWeapon: true,
	}
}


type MappedMultiangCondition struct {
	Element   string            `json:"element"`
	Operator  string            `json:"operator"`
	Value     int               `json:"value"`
	Templated map[string]string `json:"templated"`
}

func RenderConditions(conditions *[]gen.MappedMultiangCondition, lang string) []string {
	var retConditions []string
	for _, condition := range *conditions {
		retConditions = append(retConditions, condition.Templated[lang])
	}

	if len(retConditions) > 0 {
		return retConditions
	}

	return nil
}




type APIListItem struct {
	Id            int         `json:"ankama_id"`
	Name          string      `json:"name"`
	Type          string      `json:"type"`
	Level         int         `json:"level"`
	ImageUrl      string      `json:"image_url"`
}

func RenderItemListEntry(item *gen.MappedMultilangItem, lang string) APIListItem {
	return APIListItem{
		Id:            item.AnkamaId,
		Name:          item.Name[lang],
		Type:          item.Type.Name[lang],
		Level:         item.Level,
		ImageUrl:      utils.ItemImageUrl(item.AnkamaId),
	}
}

type APIRecipe struct {
	AnkamaId   int    `json:"item_ankama_id"`
	Quantity int    `json:"quantity"`
}

func RenderRecipe(recipe gen.MappedMultilangRecipe) []APIRecipe {
	if len(recipe.Entries) == 0 {
		return nil
	}

	var apiRecipes []APIRecipe
	for _, entry := range recipe.Entries {
		apiRecipes = append(apiRecipes, APIRecipe{
			AnkamaId: entry.ItemId,
			Quantity: entry.Quantity,
		})
	}
	return apiRecipes
}

type APIPageItem struct {
	Links utils.PaginationLinks    `json:"_links"`
	Items []APIListItem `json:"items"`
}



type APICharacteristic struct {
	Value string `json:"value"`
	Name  string `json:"name"`
}

func RenderCharacteristics(characteristics *[]gen.MappedMultilangCharacteristic, lang string) []APICharacteristic {
	var retCharacteristics []APICharacteristic
	for _, characteristic := range *characteristics {
		retCharacteristics = append(retCharacteristics, APICharacteristic{
			Value: characteristic.Value[lang],
			Name:  characteristic.Name[lang],
		})
	}

	if len(retCharacteristics) > 0 {
		return retCharacteristics
	}

	return nil
}

type ApiSearchResult struct {
	Id    int     `json:"ankama_id"`
	Score float64 `json:"score"`
}
