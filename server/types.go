package api

import (
	"fmt"
	"github.com/dofusdude/api/gen"
	"os"
)

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

func RenderConsumable(item *gen.MappedMultilangItem, lang string) APIConsumable {
	hostname, ok := os.LookupEnv("API_HOSTNAME")
	if !ok {
		hostname = "http://localhost"
	}

	port, ok := os.LookupEnv("API_PORT")
	if !ok {
		port = "3000"
	}

	return APIConsumable{
		Id:            item.AnkamaId,
		Name:          item.Name[lang],
		Type:          item.Type.Name[lang],
		Description:   item.Description[lang],
		Level:         item.Level,
		Pods:          item.Pods,
		ImageUrl:      item.Image,
		ImageUrlLocal: fmt.Sprintf("%s:%s/static/item/%d.png", hostname, port, item.AnkamaId),
		UsedInRecipes: item.UsedInRecipes,
		Effects:       RenderEffects(&item.Effects, lang),
	}
}

type ApiEffect struct {
	Min              int    `json:"min"`
	Max              int    `json:"max"`
	Type             string `json:"type"`
	MinMaxIrrelevant int    `json:"min_max_irrelevant"`
	Templated        string `json:"templated"`
}

type APIConsumable struct { // TODO add recipe
	Id            int         `json:"ankama_id"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	Type          string      `json:"type"`
	Level         int         `json:"level"`
	Pods          int         `json:"pods"`
	ImageUrl      string      `json:"image_url"`
	ImageUrlLocal string      `json:"image_url_local"`
	UsedInRecipes []int       `json:"used_in_recipes"`
	Effects       []ApiEffect `json:"effects"`
}

type APIRecipe struct {
	ItemId   int    `json:"item_id"`
	Quantity int    `json:"quantity"`
	ItemType string `json:"item_type"`
}

type APICharacteristic struct {
	Value string `json:"value"`
	Name  string `json:"name"`
}

type ApiSearchResult struct {
	Id    int     `json:"ankama_id"`
	Score float64 `json:"score"`
}
