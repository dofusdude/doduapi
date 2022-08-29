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
	MinInt       int     `json:"int_minimum"`
	MaxInt       int     `json:"int_maximum"`
	Type         ApiType `json:"type"`
	IgnoreMinInt bool    `json:"ignore_int_min"`
	IgnoreMaxInt bool    `json:"ignore_int_max"`
	Formatted    string  `json:"formatted"`
}

func RenderEffects(effects *[]gen.MappedMultilangEffect, lang string) []ApiEffect {
	var retEffects []ApiEffect
	for _, effect := range *effects {
		retEffects = append(retEffects, ApiEffect{
			MinInt:       effect.Min,
			MaxInt:       effect.Max,
			IgnoreMinInt: effect.MinMaxIrrelevant == -2,
			IgnoreMaxInt: effect.MinMaxIrrelevant <= -1,
			Type: ApiType{
				Name: effect.Type[lang],
			},
			Formatted: effect.Templated[lang],
		})
	}

	if len(retEffects) > 0 {
		return retEffects
	}

	return nil
}

type ApiElement struct {
	Name string `json:"name"`
}

type ApiCondition struct {
	Operator string     `json:"operator"`
	IntValue int        `json:"int_value"`
	Element  ApiElement `json:"element"`
}

type APIResource struct {
	Id          int            `json:"ankama_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Type        ApiType        `json:"type"`
	Level       int            `json:"level"`
	Pods        int            `json:"pods"`
	ImageUrls   ApiImageUrls   `json:"image_urls,omitempty"`
	Effects     []ApiEffect    `json:"effects"`
	Conditions  []ApiCondition `json:"conditions"`
	Recipe      []APIRecipe    `json:"recipe"`
}

func RenderResource(item *gen.MappedMultilangItem, lang string) APIResource {
	return APIResource{
		Id:   item.AnkamaId,
		Name: item.Name[lang],
		Type: ApiType{
			Name: item.Type.Name[lang],
		},
		Description: item.Description[lang],
		Level:       item.Level,
		Pods:        item.Pods,
		ImageUrls:   RenderImageUrls(utils.ImageUrls(item.IconId, "item")),
		Effects:     RenderEffects(&item.Effects, lang),
		Recipe:      nil,
	}
}

type APIEquipment struct {
	Id          int            `json:"ankama_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Type        ApiType        `json:"type"`
	IsWeapon    bool           `json:"is_weapon"`
	Level       int            `json:"level"`
	Pods        int            `json:"pods"`
	ImageUrls   ApiImageUrls   `json:"image_urls,omitempty"`
	Effects     []ApiEffect    `json:"effects"`
	Conditions  []ApiCondition `json:"conditions"`
	Recipe      []APIRecipe    `json:"recipe"`
}

func RenderEquipment(item *gen.MappedMultilangItem, lang string) APIEquipment {
	return APIEquipment{
		Id:   item.AnkamaId,
		Name: item.Name[lang],
		Type: ApiType{
			Name: item.Type.Name[lang],
		},
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
	Id                     int            `json:"ankama_id"`
	Name                   string         `json:"name"`
	Description            string         `json:"description"`
	Type                   ApiType        `json:"type"`
	IsWeapon               bool           `json:"is_weapon"`
	Level                  int            `json:"level"`
	Pods                   int            `json:"pods"`
	ImageUrls              ApiImageUrls   `json:"image_urls,omitempty"`
	Effects                []ApiEffect    `json:"effects"`
	Conditions             []ApiCondition `json:"conditions"`
	CriticalHitProbability int            `json:"critical_hit_probability"`
	CriticalHitBonus       int            `json:"critical_hit_bonus"`
	TwoHanded              bool           `json:"is_two_handed"`
	MaxCastPerTurn         int            `json:"max_cast_per_turn"`
	ApCost                 int            `json:"ap_cost"`
	Range                  int            `json:"range"`
	Recipe                 []APIRecipe    `json:"recipe"`
}

func RenderWeapon(item *gen.MappedMultilangItem, lang string) APIWeapon {
	return APIWeapon{
		Id:   item.AnkamaId,
		Name: item.Name[lang],
		Type: ApiType{
			Name: item.Type.Name[lang],
		},
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

func RenderConditions(conditions *[]gen.MappedMultiangCondition, lang string) []ApiCondition {
	var retConditions []ApiCondition
	for _, condition := range *conditions {
		retConditions = append(retConditions, ApiCondition{
			Operator: condition.Operator,
			IntValue: condition.Value,
			Element: ApiElement{
				Name: condition.Templated[lang],
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
	ItemIds  []int         `json:"equipment_ids"`
	Effects  [][]ApiEffect `json:"effects"`
	Level    int           `json:"highest_equipment_level"`
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
