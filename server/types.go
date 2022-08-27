package server

import (
	"github.com/dofusdude/api/gen"
	"github.com/dofusdude/api/utils"
)

type ApiImageUrls struct {
	Icon string `json:"icon"`
	Sd   string `json:"sd,omitempty"`
	Hq   string `json:"hq,omitempty"`
	Hd   string `json:"hd,omitempty"`
}

func RenderImageUrls(urls []string) ApiImageUrls {
	if len(urls) == 0 {
		return ApiImageUrls{}
	}

	var res ApiImageUrls
	res.Icon = urls[0]
	if len(urls) > 1 {
		res.Sd = urls[1]
	}
	if len(urls) > 2 {
		res.Hq = urls[2]
	}
	if len(urls) > 3 {
		res.Hd = urls[3]
	}

	return res
}

type ApiEffect struct {
	Min       int    `json:"min"`
	Max       int    `json:"max"`
	Type      string `json:"type"`
	IgnoreMin bool   `json:"ignore_min"`
	IgnoreMax bool   `json:"ignore_max"`
	Templated string `json:"templated"`
}

func RenderEffects(effects *[]gen.MappedMultilangEffect, lang string) []ApiEffect {
	var retEffects []ApiEffect
	for _, effect := range *effects {
		retEffects = append(retEffects, ApiEffect{
			Min:       effect.Min,
			Max:       effect.Max,
			IgnoreMin: effect.MinMaxIrrelevant == -2,
			IgnoreMax: effect.MinMaxIrrelevant <= -1,
			Type:      effect.Type[lang],
			Templated: effect.Templated[lang],
		})
	}

	if len(retEffects) > 0 {
		return retEffects
	}

	return nil
}

type APIResource struct {
	Id          int          `json:"ankama_id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Type        string       `json:"type"`
	Level       int          `json:"level"`
	Pods        int          `json:"pods"`
	ImageUrls   ApiImageUrls `json:"image_urls,omitempty"`
	Effects     []ApiEffect  `json:"effects"`
	Conditions  []string     `json:"conditions"`
	Recipe      []APIRecipe  `json:"recipe"`
}

func RenderResource(item *gen.MappedMultilangItem, lang string) APIResource {
	return APIResource{
		Id:          item.AnkamaId,
		Name:        item.Name[lang],
		Type:        item.Type.Name[lang],
		Description: item.Description[lang],
		Level:       item.Level,
		Pods:        item.Pods,
		ImageUrls:   RenderImageUrls(utils.ImageUrls(item.IconId, "item")),
		Effects:     RenderEffects(&item.Effects, lang),
		Recipe:      nil,
	}
}

type APIEquipment struct {
	Id          int          `json:"ankama_id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Type        string       `json:"type"`
	IsWeapon    bool         `json:"is_weapon"`
	Level       int          `json:"level"`
	Pods        int          `json:"pods"`
	ImageUrls   ApiImageUrls `json:"image_urls,omitempty"`
	Effects     []ApiEffect  `json:"effects"`
	Conditions  []string     `json:"conditions"`
	Recipe      []APIRecipe  `json:"recipe"`
}

func RenderEquipment(item *gen.MappedMultilangItem, lang string) APIEquipment {
	return APIEquipment{
		Id:          item.AnkamaId,
		Name:        item.Name[lang],
		Type:        item.Type.Name[lang],
		Description: item.Description[lang],
		Level:       item.Level,
		Pods:        item.Pods,
		ImageUrls:   RenderImageUrls(utils.ImageUrls(item.IconId, "item")),
		Effects:     RenderEffects(&item.Effects, lang),
		Conditions:  RenderConditions(&item.Conditions, lang),
		IsWeapon:    false,
		Recipe:      nil,
	}
}

type APIWeapon struct {
	Id                     int          `json:"ankama_id"`
	Name                   string       `json:"name"`
	Description            string       `json:"description"`
	Type                   string       `json:"type"`
	IsWeapon               bool         `json:"is_weapon"`
	Level                  int          `json:"level"`
	Pods                   int          `json:"pods"`
	ImageUrls              ApiImageUrls `json:"image_urls,omitempty"`
	Effects                []ApiEffect  `json:"effects"`
	Conditions             []string     `json:"conditions"`
	CriticalHitProbability int          `json:"critical_hit_probability"`
	CriticalHitBonus       int          `json:"critical_hit_bonus"`
	TwoHanded              bool         `json:"is_two_handed"`
	MaxCastPerTurn         int          `json:"max_cast_per_turn"`
	ApCost                 int          `json:"ap_cost"`
	Range                  int          `json:"range"`
	Recipe                 []APIRecipe  `json:"recipe"`
}

func RenderWeapon(item *gen.MappedMultilangItem, lang string) APIWeapon {
	return APIWeapon{
		Id:                     item.AnkamaId,
		Name:                   item.Name[lang],
		Type:                   item.Type.Name[lang],
		Description:            item.Description[lang],
		Level:                  item.Level,
		Pods:                   item.Pods,
		ImageUrls:              RenderImageUrls(utils.ImageUrls(item.IconId, "item")),
		Effects:                RenderEffects(&item.Effects, lang),
		Conditions:             RenderConditions(&item.Conditions, lang),
		Recipe:                 nil,
		CriticalHitBonus:       item.CriticalHitBonus,
		CriticalHitProbability: item.CriticalHitProbability,
		TwoHanded:              item.TwoHanded,
		MaxCastPerTurn:         item.MaxCastPerTurn,
		ApCost:                 item.ApCost,
		Range:                  item.Range,
		IsWeapon:               true,
	}
}

type MappedMultilangCondition struct {
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
	Id        int          `json:"ankama_id"`
	Name      string       `json:"name"`
	Type      string       `json:"type"`
	Level     int          `json:"level"`
	ImageUrls ApiImageUrls `json:"image_urls,omitempty"`
}

func RenderItemListEntry(item *gen.MappedMultilangItem, lang string) APIListItem {
	return APIListItem{
		Id:        item.AnkamaId,
		Name:      item.Name[lang],
		Type:      item.Type.Name[lang],
		Level:     item.Level,
		ImageUrls: RenderImageUrls(utils.ImageUrls(item.IconId, "item")),
	}
}

type APIListTypedItem struct {
	Id          int          `json:"ankama_id"`
	Name        string       `json:"name"`
	Type        string       `json:"type"`
	ItemSubtype string       `json:"item_subtype"`
	Level       int          `json:"level"`
	ImageUrls   ApiImageUrls `json:"image_urls,omitempty"`
}

func RenderTypedItemListEntry(item *gen.MappedMultilangItem, lang string) APIListTypedItem {
	return APIListTypedItem{
		Id:          item.AnkamaId,
		Name:        item.Name[lang],
		Type:        item.Type.Name[lang],
		ItemSubtype: utils.CategoryIdApiMapping(item.Type.CategoryId),
		Level:       item.Level,
		ImageUrls:   RenderImageUrls(utils.ImageUrls(item.IconId, "item")),
	}
}

type APIListMount struct {
	Id         int          `json:"ankama_id"`
	Name       string       `json:"name"`
	FamilyName string       `json:"family_name"`
	ImageUrls  ApiImageUrls `json:"image_urls,omitempty"`
}

func RenderMountListEntry(mount *gen.MappedMultilangMount, lang string) APIListMount {
	return APIListMount{
		Id:         mount.AnkamaId,
		Name:       mount.Name[lang],
		ImageUrls:  RenderImageUrls(utils.ImageUrls(mount.AnkamaId, "mount")),
		FamilyName: mount.FamilyName[lang],
	}
}

type APIRecipe struct {
	AnkamaId int `json:"item_ankama_id"`
	Quantity int `json:"quantity"`
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
	Links utils.PaginationLinks `json:"_links"`
	Items []APIListItem         `json:"items"`
}

type APIPageMount struct {
	Links utils.PaginationLinks `json:"_links"`
	Items []APIListMount        `json:"mounts"`
}

type APIPageSet struct {
	Links utils.PaginationLinks `json:"_links"`
	Items []APIListSet          `json:"sets"`
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

type APIMount struct {
	Id         int          `json:"ankama_id"`
	Name       string       `json:"name"`
	FamilyName string       `json:"family_name"`
	ImageUrls  ApiImageUrls `json:"image_urls,omitempty"`
	Effects    []ApiEffect  `json:"effects"`
}

func RenderMount(mount *gen.MappedMultilangMount, lang string) APIMount {
	return APIMount{
		Id:         mount.AnkamaId,
		Name:       mount.Name[lang],
		FamilyName: mount.FamilyName[lang],
		ImageUrls:  RenderImageUrls(utils.ImageUrls(mount.AnkamaId, "mount")),
		Effects:    RenderEffects(&mount.Effects, lang),
	}
}

type APIListSet struct {
	Id    int    `json:"ankama_id"`
	Name  string `json:"name"`
	Items int    `json:"items"`
	Level int    `json:"level"`
}

func RenderSetListEntry(set *gen.MappedMultilangSet, lang string) APIListSet {
	return APIListSet{
		Id:    set.AnkamaId,
		Name:  set.Name[lang],
		Items: len(set.ItemIds),
		Level: set.Level,
	}
}

type APISet struct {
	AnkamaId int           `json:"ankama_id"`
	Name     string        `json:"name"`
	ItemIds  []int         `json:"items"`
	Effects  [][]ApiEffect `json:"effects"`
	Level    int           `json:"level"`
}

func RenderSet(set *gen.MappedMultilangSet, lang string) APISet {
	var effects [][]ApiEffect
	for _, effect := range set.Effects {
		effects = append(effects, RenderEffects(&effect, lang))
	}

	return APISet{
		AnkamaId: set.AnkamaId,
		Name:     set.Name[lang],
		ItemIds:  set.ItemIds,
		Effects:  effects,
		Level:    set.Level,
	}
}

type ApiSearchResult struct {
	Id    int     `json:"ankama_id"`
	Score float64 `json:"score"`
}

func (item APIResource) SetRecipe(recipe []APIRecipe) {
	item.Recipe = recipe
}
