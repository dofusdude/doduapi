package server

import (
	"fmt"
	"github.com/dofusdude/api/gen"
	"github.com/dofusdude/api/utils"
	"github.com/hashicorp/go-memdb"
	"log"
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
	MinInt       int                    `json:"int_minimum"`
	MaxInt       int                    `json:"int_maximum"`
	Type         ApiEffectConditionType `json:"type"`
	IgnoreMinInt bool                   `json:"ignore_int_min"`
	IgnoreMaxInt bool                   `json:"ignore_int_max"`
	Formatted    string                 `json:"formatted"`
}

func RenderEffects(effects *[]gen.MappedMultilangEffect, lang string) []ApiEffect {
	var retEffects []ApiEffect
	for _, effect := range *effects {
		retEffects = append(retEffects, ApiEffect{
			MinInt:       effect.Min,
			MaxInt:       effect.Max,
			IgnoreMinInt: effect.IsMeta || effect.MinMaxIrrelevant == -2,
			IgnoreMaxInt: effect.IsMeta || effect.MinMaxIrrelevant <= -1,
			Type: ApiEffectConditionType{
				Name:   effect.Type[lang],
				Id:     effect.ElementId,
				IsMeta: effect.IsMeta,
			},
			Formatted: effect.Templated[lang],
		})
	}

	if len(retEffects) > 0 {
		return retEffects
	}

	return nil
}

type ApiCondition struct {
	Operator string                 `json:"operator"`
	IntValue int                    `json:"int_value"`
	Element  ApiEffectConditionType `json:"element"`
}

type APIResource struct {
	Id            int            `json:"ankama_id"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Type          ApiType        `json:"type"`
	Level         int            `json:"level"`
	Pods          int            `json:"pods"`
	ImageUrls     ApiImageUrls   `json:"image_urls,omitempty"`
	HasEffects    bool           `json:"has_effects"`
	Effects       []ApiEffect    `json:"effects,omitempty"`
	HasConditions bool           `json:"has_conditions"`
	Conditions    []ApiCondition `json:"conditions,omitempty"`
	HasRecipe     bool           `json:"has_recipe"`
	Recipe        []APIRecipe    `json:"recipe,omitempty"`
}

func RenderResource(item *gen.MappedMultilangItem, lang string) APIResource {
	resource := APIResource{
		Id:   item.AnkamaId,
		Name: item.Name[lang],
		Type: ApiType{
			Name: item.Type.Name[lang],
		},
		Description: item.Description[lang],
		Level:       item.Level,
		Pods:        item.Pods,
		ImageUrls:   RenderImageUrls(utils.ImageUrls(item.IconId, "item")),
		Recipe:      nil,
		HasRecipe:   false,
	}

	conditions := RenderConditions(&item.Conditions, lang)
	if len(conditions) == 0 {
		resource.HasConditions = false
		resource.Conditions = nil
	} else {
		resource.HasConditions = true
		resource.Conditions = conditions
	}

	effects := RenderEffects(&item.Effects, lang)
	if len(effects) == 0 {
		resource.HasEffects = false
		resource.Effects = nil
	} else {
		resource.HasEffects = true
		resource.Effects = effects
	}

	return resource
}

type APIEquipment struct {
	Id            int                `json:"ankama_id"`
	Name          string             `json:"name"`
	Description   string             `json:"description"`
	Type          ApiType            `json:"type"`
	IsWeapon      bool               `json:"is_weapon"`
	Level         int                `json:"level"`
	Pods          int                `json:"pods"`
	ImageUrls     ApiImageUrls       `json:"image_urls,omitempty"`
	HasEffects    bool               `json:"has_effects"`
	Effects       []ApiEffect        `json:"effects,omitempty"`
	HasConditions bool               `json:"has_conditions"`
	Conditions    []ApiCondition     `json:"conditions,omitempty"`
	HasRecipe     bool               `json:"has_recipe"`
	Recipe        []APIRecipe        `json:"recipe,omitempty"`
	HasParentSet  bool               `json:"has_parent_set"`
	ParentSet     *APISetReverseLink `json:"parent_set,omitempty"`
}

func RenderEquipment(item *gen.MappedMultilangItem, lang string) APIEquipment {
	var setLink *APISetReverseLink = nil
	if item.HasParentSet {
		setLink = &APISetReverseLink{
			Id:   item.ParentSet.Id,
			Name: item.ParentSet.Name[lang],
		}
	}

	equip := APIEquipment{
		Id:   item.AnkamaId,
		Name: item.Name[lang],
		Type: ApiType{
			Name: item.Type.Name[lang],
		},
		Description:  item.Description[lang],
		Level:        item.Level,
		Pods:         item.Pods,
		ImageUrls:    RenderImageUrls(utils.ImageUrls(item.IconId, "item")),
		IsWeapon:     false,
		HasRecipe:    false,
		Recipe:       nil,
		HasParentSet: item.HasParentSet,
		ParentSet:    setLink,
	}

	conditions := RenderConditions(&item.Conditions, lang)
	if len(conditions) == 0 {
		equip.HasConditions = false
		equip.Conditions = nil
	} else {
		equip.HasConditions = true
		equip.Conditions = conditions
	}

	effects := RenderEffects(&item.Effects, lang)
	if len(effects) == 0 {
		equip.HasEffects = false
		equip.Effects = nil
	} else {
		equip.HasEffects = true
		equip.Effects = effects
	}

	return equip
}

type APIRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type APISetReverseLink struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type APIWeapon struct {
	Id                     int                `json:"ankama_id"`
	Name                   string             `json:"name"`
	Description            string             `json:"description"`
	Type                   ApiType            `json:"type"`
	IsWeapon               bool               `json:"is_weapon"`
	Level                  int                `json:"level"`
	Pods                   int                `json:"pods"`
	ImageUrls              ApiImageUrls       `json:"image_urls,omitempty"`
	HasEffects             bool               `json:"has_effects"`
	Effects                []ApiEffect        `json:"effects,omitempty"`
	HasConditions          bool               `json:"has_conditions"`
	Conditions             []ApiCondition     `json:"conditions,omitempty"`
	CriticalHitProbability int                `json:"critical_hit_probability"`
	CriticalHitBonus       int                `json:"critical_hit_bonus"`
	TwoHanded              bool               `json:"is_two_handed"`
	MaxCastPerTurn         int                `json:"max_cast_per_turn"`
	ApCost                 int                `json:"ap_cost"`
	Range                  APIRange           `json:"range"`
	HasRecipe              bool               `json:"has_recipe"`
	Recipe                 []APIRecipe        `json:"recipe,omitempty"`
	HasParentSet           bool               `json:"has_parent_set"`
	ParentSet              *APISetReverseLink `json:"parent_set,omitempty"`
}

func RenderWeapon(item *gen.MappedMultilangItem, lang string) APIWeapon {
	var setLink *APISetReverseLink = nil
	if item.HasParentSet {
		setLink = &APISetReverseLink{
			Id:   item.ParentSet.Id,
			Name: item.ParentSet.Name[lang],
		}
	}

	weapon := APIWeapon{
		Id:   item.AnkamaId,
		Name: item.Name[lang],
		Type: ApiType{
			Name: item.Type.Name[lang],
		},
		Description:            item.Description[lang],
		Level:                  item.Level,
		Pods:                   item.Pods,
		ImageUrls:              RenderImageUrls(utils.ImageUrls(item.IconId, "item")),
		Recipe:                 nil,
		CriticalHitBonus:       item.CriticalHitBonus,
		CriticalHitProbability: item.CriticalHitProbability,
		TwoHanded:              item.TwoHanded,
		MaxCastPerTurn:         item.MaxCastPerTurn,
		ApCost:                 item.ApCost,
		Range: APIRange{
			Min: item.MinRange,
			Max: item.Range,
		},
		IsWeapon:     true,
		HasParentSet: item.HasParentSet,
		ParentSet:    setLink,
		HasRecipe:    false,
	}

	conditions := RenderConditions(&item.Conditions, lang)
	if len(conditions) == 0 {
		weapon.HasConditions = false
		weapon.Conditions = nil
	} else {
		weapon.HasConditions = true
		weapon.Conditions = conditions
	}

	effects := RenderEffects(&item.Effects, lang)
	if len(effects) == 0 {
		weapon.HasEffects = false
		weapon.Effects = nil
	} else {
		weapon.HasEffects = true
		weapon.Effects = effects
	}

	return weapon
}

type MappedMultilangCondition struct {
	Element   string            `json:"element"`
	Operator  string            `json:"operator"`
	Value     int               `json:"value"`
	Templated map[string]string `json:"templated"`
}

func RenderConditions(conditions *[]gen.MappedMultiangCondition, lang string) []ApiCondition {
	var retConditions []ApiCondition
	for _, condition := range *conditions {
		retConditions = append(retConditions, ApiCondition{
			Operator: condition.Operator,
			IntValue: condition.Value,
			Element: ApiEffectConditionType{
				Name: condition.Templated[lang],
				Id:   condition.ElementId,
			},
		})
	}

	if len(retConditions) > 0 {
		return retConditions
	}

	return nil
}

type ApiType struct {
	Name string `json:"name"`
}

type ApiEffectConditionType struct {
	Name   string `json:"name"`
	Id     int    `json:"id"`
	IsMeta bool   `json:"is_meta"`
}

type APIListItem struct {
	Id        int          `json:"ankama_id"`
	Name      string       `json:"name"`
	Type      ApiType      `json:"type"`
	Level     int          `json:"level"`
	ImageUrls ApiImageUrls `json:"image_urls,omitempty"`
}

func RenderItemListEntry(item *gen.MappedMultilangItem, lang string) APIListItem {
	return APIListItem{
		Id:   item.AnkamaId,
		Name: item.Name[lang],
		Type: ApiType{
			Name: item.Type.Name[lang],
		},
		Level:     item.Level,
		ImageUrls: RenderImageUrls(utils.ImageUrls(item.IconId, "item")),
	}
}

type APIListTypedItem struct {
	Id          int          `json:"ankama_id"`
	Name        string       `json:"name"`
	Type        ApiType      `json:"type"`
	ItemSubtype string       `json:"item_subtype"`
	Level       int          `json:"level"`
	ImageUrls   ApiImageUrls `json:"image_urls,omitempty"`
}

func RenderTypedItemListEntry(item *gen.MappedMultilangItem, lang string) APIListTypedItem {
	return APIListTypedItem{
		Id:   item.AnkamaId,
		Name: item.Name[lang],
		Type: ApiType{
			Name: item.Type.Name[lang],
		},
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
	AnkamaId int    `json:"item_ankama_id"`
	ItemType string `json:"item_subtype"`
	Quantity int    `json:"quantity"`
}

func RenderRecipe(recipe gen.MappedMultilangRecipe, db *memdb.MemDB) []APIRecipe {
	if len(recipe.Entries) == 0 {
		return nil
	}

	txn := Db.Txn(false)
	defer txn.Abort()

	var apiRecipes []APIRecipe
	for _, entry := range recipe.Entries {
		raw, err := txn.First(fmt.Sprintf("%s-%s", utils.CurrentRedBlueVersionStr(Version.MemDb), "all_items"), "id", entry.ItemId)
		if err != nil {
			log.Println(err)
			return nil
		}
		item := raw.(*gen.MappedMultilangItem)

		apiRecipes = append(apiRecipes, APIRecipe{
			AnkamaId: entry.ItemId,
			Quantity: entry.Quantity,
			ItemType: utils.CategoryIdApiMapping(item.Type.CategoryId),
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

type APIMount struct {
	Id         int          `json:"ankama_id"`
	Name       string       `json:"name"`
	FamilyName string       `json:"family_name"`
	ImageUrls  ApiImageUrls `json:"image_urls,omitempty"`
	HasEffects bool         `json:"has_effects"`
	Effects    []ApiEffect  `json:"effects,omitempty"`
}

func RenderMount(mount *gen.MappedMultilangMount, lang string) APIMount {
	resMount := APIMount{
		Id:         mount.AnkamaId,
		Name:       mount.Name[lang],
		FamilyName: mount.FamilyName[lang],
		ImageUrls:  RenderImageUrls(utils.ImageUrls(mount.AnkamaId, "mount")),
	}

	effects := RenderEffects(&mount.Effects, lang)
	if len(effects) == 0 {
		resMount.HasEffects = false
		resMount.Effects = nil
	} else {
		resMount.HasEffects = true
		resMount.Effects = effects
	}

	return resMount
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
	AnkamaId   int           `json:"ankama_id"`
	Name       string        `json:"name"`
	ItemIds    []int         `json:"equipment_ids"`
	HasEffects bool          `json:"has_effects"`
	Effects    [][]ApiEffect `json:"effects,omitempty"`
	Level      int           `json:"highest_equipment_level"`
}

func RenderSet(set *gen.MappedMultilangSet, lang string) APISet {
	var effects [][]ApiEffect
	for _, effect := range set.Effects {
		effects = append(effects, RenderEffects(&effect, lang))
	}

	resSet := APISet{
		AnkamaId: set.AnkamaId,
		Name:     set.Name[lang],
		ItemIds:  set.ItemIds,
		Effects:  effects,
		Level:    set.Level,
	}

	if len(effects) == 0 {
		resSet.HasEffects = false
		resSet.Effects = nil
	} else {
		resSet.HasEffects = true
		resSet.Effects = effects
	}

	return resSet
}
