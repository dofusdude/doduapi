package main

import (
	"fmt"

	"github.com/charmbracelet/log"
	mapping "github.com/dofusdude/dodumap"
	"github.com/hashicorp/go-memdb"
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

type ApiAllSearchItem struct {
	Type      *ApiType      `json:"type,omitempty"`
	Level     *int          `json:"level,omitempty"`
	ImageUrls *ApiImageUrls `json:"image_urls,omitempty"`
}

type ApiAllSearchResult struct {
	Name       string            `json:"name"`
	Id         int               `json:"ankama_id"`
	Type       string            `json:"type"`
	ItemFields *ApiAllSearchItem `json:"item_fields,omitempty"`
}

type ApiEffect struct {
	MinInt       int           `json:"int_minimum"`
	MaxInt       int           `json:"int_maximum"`
	Type         ApiEffectType `json:"type"`
	IgnoreMinInt bool          `json:"ignore_int_min"`
	IgnoreMaxInt bool          `json:"ignore_int_max"`
	Formatted    string        `json:"formatted"`
}

func RenderEffects(effects *[]mapping.MappedMultilangEffect, lang string) []ApiEffect {
	var retEffects []ApiEffect
	for _, effect := range *effects {
		retEffects = append(retEffects, ApiEffect{
			MinInt:       effect.Min,
			MaxInt:       effect.Max,
			IgnoreMinInt: effect.IsMeta || effect.MinMaxIrrelevant == -2,
			IgnoreMaxInt: effect.IsMeta || effect.MinMaxIrrelevant <= -1,
			Type: ApiEffectType{
				Name:     effect.Type[lang],
				Id:       effect.ElementId,
				IsMeta:   effect.IsMeta,
				IsActive: effect.Active,
			},
			Formatted: effect.Templated[lang],
		})
	}

	if len(retEffects) > 0 {
		return retEffects
	}

	return nil
}

type ApiSetEffect struct {
	MinInt          int           `json:"int_minimum"`
	MaxInt          int           `json:"int_maximum"`
	Type            ApiEffectType `json:"type"`
	IgnoreMinInt    bool          `json:"ignore_int_min"`
	IgnoreMaxInt    bool          `json:"ignore_int_max"`
	Formatted       string        `json:"formatted"`
	ItemCombination uint          `json:"item_combination"`
}

func RenderSetEffects(effects *[]mapping.MappedMultilangSetEffect, lang string) []ApiSetEffect {
	var retEffects []ApiSetEffect
	for _, effect := range *effects {
		retEffects = append(retEffects, ApiSetEffect{
			MinInt:       effect.Min,
			MaxInt:       effect.Max,
			IgnoreMinInt: effect.IsMeta || effect.MinMaxIrrelevant == -2,
			IgnoreMaxInt: effect.IsMeta || effect.MinMaxIrrelevant <= -1,
			Type: ApiEffectType{
				Name:     effect.Type[lang],
				Id:       effect.ElementId,
				IsMeta:   effect.IsMeta,
				IsActive: effect.Active,
			},
			Formatted:       effect.Templated[lang],
			ItemCombination: effect.ItemCombination,
		})
	}

	if len(retEffects) > 0 {
		return retEffects
	}

	return nil
}

type ApiCondition struct {
	Operator string           `json:"operator"`
	IntValue int              `json:"int_value"`
	Element  ApiConditionType `json:"element"`
}

type ApiConditionNode struct {
	Condition *ApiCondition       `json:"condition,omitempty"`
	IsOperand bool                `json:"is_operand"`
	Relation  *string             `json:"relation,omitempty"` // "and" or "or"
	Children  []*ApiConditionNode `json:"children,omitempty"`
}

type APIResource struct {
	Id            int               `json:"ankama_id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Type          ApiType           `json:"type"`
	Level         int               `json:"level"`
	Pods          int               `json:"pods"`
	ImageUrls     ApiImageUrls      `json:"image_urls,omitempty"`
	Effects       []ApiEffect       `json:"effects,omitempty"`
	Conditions    []ApiCondition    `json:"conditions,omitempty"`
	ConditionTree *ApiConditionNode `json:"condition_tree,omitempty"`
	Recipe        []APIRecipe       `json:"recipe,omitempty"`
}

func RenderResource(item *mapping.MappedMultilangItem, lang string) APIResource {
	resource := APIResource{
		Id:   item.AnkamaId,
		Name: item.Name[lang],
		Type: ApiType{
			Name: item.Type.Name[lang],
			Id:   item.Type.ItemTypeId,
		},
		Description: item.Description[lang],
		Level:       item.Level,
		Pods:        item.Pods,
		ImageUrls:   RenderImageUrls(ImageUrls(item.IconId, "item")),
		Recipe:      nil,
	}

	resource.ConditionTree = RenderConditionTree(item.ConditionTree, lang)
	conditions := RenderConditions(&item.Conditions, lang)
	if len(conditions) == 0 {
		resource.Conditions = nil
	} else {
		resource.Conditions = conditions
	}

	effects := RenderEffects(&item.Effects, lang)
	if len(effects) == 0 {
		resource.Effects = nil
	} else {
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
	Effects       []ApiEffect        `json:"effects,omitempty"`
	Conditions    []ApiCondition     `json:"conditions,omitempty"`
	ConditionTree *ApiConditionNode  `json:"condition_tree,omitempty"`
	Recipe        []APIRecipe        `json:"recipe,omitempty"`
	ParentSet     *APISetReverseLink `json:"parent_set,omitempty"`
}

func RenderEquipment(item *mapping.MappedMultilangItem, lang string) APIEquipment {
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
			Id:   item.Type.ItemTypeId,
		},
		Description: item.Description[lang],
		Level:       item.Level,
		Pods:        item.Pods,
		ImageUrls:   RenderImageUrls(ImageUrls(item.IconId, "item")),
		IsWeapon:    false,
		Recipe:      nil,
		ParentSet:   setLink,
	}

	equip.ConditionTree = RenderConditionTree(item.ConditionTree, lang)
	conditions := RenderConditions(&item.Conditions, lang)
	if len(conditions) == 0 {
		equip.Conditions = nil
	} else {
		equip.Conditions = conditions
	}

	effects := RenderEffects(&item.Effects, lang)
	if len(effects) == 0 {
		equip.Effects = nil
	} else {
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
	Effects                []ApiEffect        `json:"effects,omitempty"`
	Conditions             []ApiCondition     `json:"conditions,omitempty"`
	ConditionTree          *ApiConditionNode  `json:"condition_tree,omitempty"`
	CriticalHitProbability int                `json:"critical_hit_probability"`
	CriticalHitBonus       int                `json:"critical_hit_bonus"`
	TwoHanded              bool               `json:"is_two_handed"`
	MaxCastPerTurn         int                `json:"max_cast_per_turn"`
	ApCost                 int                `json:"ap_cost"`
	Range                  APIRange           `json:"range"`
	Recipe                 []APIRecipe        `json:"recipe,omitempty"`
	ParentSet              *APISetReverseLink `json:"parent_set,omitempty"`
}

func RenderWeapon(item *mapping.MappedMultilangItem, lang string) APIWeapon {
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
			Id:   item.Type.ItemTypeId,
		},
		Description:            item.Description[lang],
		Level:                  item.Level,
		Pods:                   item.Pods,
		ImageUrls:              RenderImageUrls(ImageUrls(item.IconId, "item")),
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
		IsWeapon:  true,
		ParentSet: setLink,
	}

	weapon.ConditionTree = RenderConditionTree(item.ConditionTree, lang)
	conditions := RenderConditions(&item.Conditions, lang)
	if len(conditions) == 0 {
		weapon.Conditions = nil
	} else {
		weapon.Conditions = conditions
	}

	effects := RenderEffects(&item.Effects, lang)
	if len(effects) == 0 {
		weapon.Effects = nil
	} else {
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

func buildAPIConditionTree(out **ApiConditionNode, root *mapping.ConditionTreeNodeMapped, lang string) {
	if root == nil {
		return
	}

	if *out == nil {
		*out = new(ApiConditionNode)
	}

	(*out).IsOperand = root.IsOperand

	if root.IsOperand {
		(*out).Condition = &ApiCondition{
			Operator: root.Value.Operator,
			IntValue: root.Value.Value,
			Element: ApiConditionType{
				Name: root.Value.Templated[lang],
				Id:   root.Value.ElementId,
			},
		}
		return
	} else {
		(*out).Relation = root.Relation
		(*out).Children = make([]*ApiConditionNode, len(root.Children))
		for i, child := range root.Children {
			childOut := new(*ApiConditionNode)
			buildAPIConditionTree(childOut, child, lang)
			(*out).Children[i] = *childOut
		}
		return
	}
}

func RenderConditionTree(conditions *mapping.ConditionTreeNodeMapped, lang string) *ApiConditionNode {
	retConditionTree := new(*ApiConditionNode)
	buildAPIConditionTree(retConditionTree, conditions, lang)
	return *retConditionTree
}

func RenderConditions(conditions *[]mapping.MappedMultilangCondition, lang string) []ApiCondition {
	var retConditions []ApiCondition
	for _, condition := range *conditions {
		retConditions = append(retConditions, ApiCondition{
			Operator: condition.Operator,
			IntValue: condition.Value,
			Element: ApiConditionType{
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
	Id   int    `json:"id"`
}

type ApiConditionType struct {
	Name string `json:"name"`
	Id   int    `json:"id"`
}

type ApiEffectType struct {
	Name     string `json:"name"`
	Id       int    `json:"id"`
	IsMeta   bool   `json:"is_meta"`
	IsActive bool   `json:"is_active"`
}

type APIListItem struct {
	Id        int          `json:"ankama_id"`
	Name      string       `json:"name"`
	Type      ApiType      `json:"type"`
	Level     int          `json:"level"`
	ImageUrls ApiImageUrls `json:"image_urls,omitempty"`

	// extra fields
	Description   *string           `json:"description,omitempty"`
	Recipe        []APIRecipe       `json:"recipe,omitempty"`
	Conditions    []ApiCondition    `json:"conditions,omitempty"`
	ConditionTree *ApiConditionNode `json:"condition_tree,omitempty"`
	Effects       []ApiEffect       `json:"effects,omitempty"`

	// extra equipment
	IsWeapon  *bool              `json:"is_weapon,omitempty"`
	Pods      *int               `json:"pods,omitempty"`
	ParentSet *APISetReverseLink `json:"parent_set,omitempty"`

	// extra weapon
	CriticalHitProbability *int      `json:"critical_hit_probability,omitempty"`
	CriticalHitBonus       *int      `json:"critical_hit_bonus,omitempty"`
	TwoHanded              *bool     `json:"is_two_handed,omitempty"`
	MaxCastPerTurn         *int      `json:"max_cast_per_turn,omitempty"`
	ApCost                 *int      `json:"ap_cost,omitempty"`
	Range                  *APIRange `json:"range,omitempty"`
}

func RenderItemListEntry(item *mapping.MappedMultilangItem, lang string) APIListItem {
	return APIListItem{
		Id:   item.AnkamaId,
		Name: item.Name[lang],
		Type: ApiType{
			Name: item.Type.Name[lang],
			Id:   item.Type.ItemTypeId,
		},
		Level:     item.Level,
		ImageUrls: RenderImageUrls(ImageUrls(item.IconId, "item")),
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

func RenderTypedItemListEntry(item *mapping.MappedMultilangItem, lang string) APIListTypedItem {
	return APIListTypedItem{
		Id:   item.AnkamaId,
		Name: item.Name[lang],
		Type: ApiType{
			Name: item.Type.Name[lang],
			Id:   item.Type.ItemTypeId,
		},
		ItemSubtype: CategoryIdApiMapping(item.Type.CategoryId),
		Level:       item.Level,
		ImageUrls:   RenderImageUrls(ImageUrls(item.IconId, "item")),
	}
}

type APIListMount struct {
	Id         int          `json:"ankama_id"`
	Name       string       `json:"name"`
	FamilyName string       `json:"family_name"`
	ImageUrls  ApiImageUrls `json:"image_urls,omitempty"`

	// extra fields
	Effects []ApiEffect `json:"effects,omitempty"`
}

func RenderMountListEntry(mount *mapping.MappedMultilangMount, lang string) APIListMount {
	return APIListMount{
		Id:         mount.AnkamaId,
		Name:       mount.Name[lang],
		ImageUrls:  RenderImageUrls(ImageUrls(mount.AnkamaId, "mount")),
		FamilyName: mount.FamilyName[lang],
	}
}

type APIRecipe struct {
	AnkamaId int    `json:"item_ankama_id"`
	ItemType string `json:"item_subtype"`
	Quantity int    `json:"quantity"`
}

func RenderRecipe(recipe mapping.MappedMultilangRecipe, db *memdb.MemDB) []APIRecipe {
	if len(recipe.Entries) == 0 {
		return nil
	}

	txn := db.Txn(false)
	defer txn.Abort()

	var apiRecipes []APIRecipe
	for _, entry := range recipe.Entries {
		raw, err := txn.First(fmt.Sprintf("%s-%s", CurrentRedBlueVersionStr(Version.MemDb), "all_items"), "id", entry.ItemId)
		if err != nil {
			log.Error(err)
			return nil
		}
		item := raw.(*mapping.MappedMultilangItem)

		apiRecipes = append(apiRecipes, APIRecipe{
			AnkamaId: entry.ItemId,
			Quantity: entry.Quantity,
			ItemType: CategoryIdApiMapping(item.Type.CategoryId),
		})
	}
	return apiRecipes
}

type APIPageItem struct {
	Links PaginationLinks `json:"_links,omitempty"`
	Items []APIListItem   `json:"items"`
}

type APIPageMount struct {
	Links PaginationLinks `json:"_links,omitempty"`
	Items []APIListMount  `json:"mounts"`
}

type APIPageSet struct {
	Links PaginationLinks `json:"_links,omitempty"`
	Items []APIListSet    `json:"sets"`
}

type APIMount struct {
	Id         int          `json:"ankama_id"`
	Name       string       `json:"name"`
	FamilyName string       `json:"family_name"`
	ImageUrls  ApiImageUrls `json:"image_urls,omitempty"`
	Effects    []ApiEffect  `json:"effects,omitempty"`
}

func RenderMount(mount *mapping.MappedMultilangMount, lang string) APIMount {
	resMount := APIMount{
		Id:         mount.AnkamaId,
		Name:       mount.Name[lang],
		FamilyName: mount.FamilyName[lang],
		ImageUrls:  RenderImageUrls(ImageUrls(mount.AnkamaId, "mount")),
	}

	effects := RenderEffects(&mount.Effects, lang)
	if len(effects) == 0 {
		resMount.Effects = nil
	} else {
		resMount.Effects = effects
	}

	return resMount
}

type APIListSet struct {
	Id         int    `json:"ankama_id"`
	Name       string `json:"name"`
	Items      int    `json:"items"`
	Level      int    `json:"level"`
	IsCosmetic bool   `json:"is_cosmetic"`

	// extra fields
	Effects [][]ApiSetEffect `json:"effects,omitempty"`
	ItemIds []int            `json:"equipment_ids,omitempty"`
}

func RenderSetListEntry(set *mapping.MappedMultilangSet, lang string) APIListSet {
	return APIListSet{
		Id:         set.AnkamaId,
		Name:       set.Name[lang],
		Items:      len(set.ItemIds),
		Level:      set.Level,
		IsCosmetic: set.IsCosmetic,
	}
}

type APISet struct {
	AnkamaId   int              `json:"ankama_id"`
	Name       string           `json:"name"`
	ItemIds    []int            `json:"equipment_ids"`
	Effects    [][]ApiSetEffect `json:"effects,omitempty"`
	Level      int              `json:"highest_equipment_level"`
	IsCosmetic bool             `json:"is_cosmetic"`
}

func RenderSet(set *mapping.MappedMultilangSet, lang string) APISet {
	var effects [][]ApiSetEffect
	for _, effect := range set.Effects {
		effects = append(effects, RenderSetEffects(&effect, lang))
	}

	resSet := APISet{
		AnkamaId:   set.AnkamaId,
		Name:       set.Name[lang],
		ItemIds:    set.ItemIds,
		Effects:    effects,
		Level:      set.Level,
		IsCosmetic: set.IsCosmetic,
	}

	if len(effects) == 0 {
		resSet.Effects = nil
	} else {
		resSet.Effects = effects
	}

	return resSet
}
