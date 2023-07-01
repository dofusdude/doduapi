package gen

import (
	"encoding/json"
	"fmt"
	"github.com/dofusdude/ankabuffer"
	"github.com/dofusdude/api/update"
	"github.com/dofusdude/api/utils"
	"log"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

func Parse() {
	log.Println("parsing...")
	startParsing := time.Now()
	gameData := ParseRawData()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func(data *JSONGameData) {
		defer wg.Done()
		DownloadMountsImages(data, &utils.FileHashes, 6)
		log.Println("... downloaded mount images")
	}(gameData)

	languageData := ParseRawLanguages()
	log.Println("... completed parsing in", time.Since(startParsing))

	log.Println("mapping...")
	startMapping := time.Now()

	err := utils.LoadPersistedElements("db/elements.json", "db/item_types.json")
	if err != nil {
		log.Fatal(err)
	}

	// ----
	log.Println("mapping items...")

	runtime.GC()

	mappedItems := MapItems(gameData, &languageData)
	log.Println("saving items...")
	out, err := os.Create("data/MAPPED_ITEMS.json")
	if err != nil {
		fmt.Println(err)
	}
	defer out.Close()

	outBytes, err := json.MarshalIndent(mappedItems, "", "    ")
	if err != nil {
		fmt.Println(err)
		return
	}

	out.Write(outBytes)

	// ----
	log.Println("mapping mounts...")
	mappedMounts := MapMounts(gameData, &languageData)
	log.Println("saving mounts...")
	out, err = os.Create("data/MAPPED_MOUNTS.json")
	if err != nil {
		fmt.Println(err)
	}
	defer out.Close()

	outBytes, err = json.MarshalIndent(mappedMounts, "", "    ")
	if err != nil {
		fmt.Println(err)
		return
	}

	out.Write(outBytes)

	// ----
	log.Println("mapping sets...")
	mappedSets := MapSets(gameData, &languageData)
	log.Println("saving sets...")
	outSets, err := os.Create("data/MAPPED_SETS.json")
	if err != nil {
		fmt.Println(err)
	}
	defer outSets.Close()

	outSetsBytes, err := json.MarshalIndent(mappedSets, "", "    ")
	if err != nil {
		fmt.Println(err)
		return
	}

	outSets.Write(outSetsBytes)

	// ----
	log.Println("mapping recipes...")
	mappedRecipes := MapRecipes(gameData)
	log.Println("saving recipes...")
	outRecipes, err := os.Create("data/MAPPED_RECIPES.json")
	if err != nil {
		fmt.Println(err)
	}
	defer outRecipes.Close()

	outRecipeBytes, err := json.MarshalIndent(mappedRecipes, "", "    ")
	if err != nil {
		fmt.Println(err)
		return
	}

	outRecipes.Write(outRecipeBytes)

	err = utils.PersistElements("db/elements.json", "db/item_types.json")
	if err != nil {
		log.Fatal(err)
	}

	wg.Wait() // mount images
	log.Println("... completed mapping in", time.Since(startMapping))
	mappedSets = nil
	mappedItems = nil
}

func DownloadMountImageWorker(manifest *ankabuffer.Manifest, fragment string, workerSlice []JSONGameMount) {
	wg := sync.WaitGroup{}

	for _, mount := range workerSlice {
		wg.Add(1)
		go func(mountId int, wg *sync.WaitGroup) {
			defer wg.Done()
			var image update.HashFile
			image.Filename = fmt.Sprintf("content/gfx/mounts/%d.png", mountId)
			image.FriendlyName = fmt.Sprintf("data/img/mount/%d.png", mountId)
			_ = update.DownloadUnpackFiles(manifest, fragment, []update.HashFile{image}, "data/img/mount", true)
		}(mount.Id, &wg)

		//  Missing bundle for content/gfx/mounts/162.swf
		wg.Add(1)
		go func(mountId int, wg *sync.WaitGroup) {
			defer wg.Done()
			var image update.HashFile
			image.Filename = fmt.Sprintf("content/gfx/mounts/%d.swf", mountId)
			image.FriendlyName = fmt.Sprintf("data/vector/mount/%d.swf", mountId)
			_ = update.DownloadUnpackFiles(manifest, fragment, []update.HashFile{image}, "data/vector/mount", false)
		}(mount.Id, &wg)
	}

	wg.Wait()
}

func DownloadMountsImages(mounts *JSONGameData, hashJson *ankabuffer.Manifest, worker int) {
	arr := utils.Values(mounts.Mounts)
	workerSlices := utils.PartitionSlice(arr, worker)

	wg := sync.WaitGroup{}
	for _, workerSlice := range workerSlices {
		wg.Add(1)
		go func(workerSlice []JSONGameMount) {
			defer wg.Done()
			DownloadMountImageWorker(hashJson, "main", workerSlice)
		}(workerSlice)
	}
	wg.Wait()
}

func isActiveEffect(name map[string]string) bool {
	regex := regexp.MustCompile(`^\(.*\)$`)
	if regex.Match([]byte(name["en"])) {
		return true
	}
	if strings.Contains(name["de"], "(Ziel)") {
		return true
	}
	return false
}

func ParseEffects(data *JSONGameData, allEffects [][]JSONGameItemPossibleEffect, langs *map[string]LangDict) [][]MappedMultilangEffect {
	var mappedAllEffects [][]MappedMultilangEffect
	for _, effects := range allEffects {
		var mappedEffects []MappedMultilangEffect
		for _, effect := range effects {

			var mappedEffect MappedMultilangEffect
			currentEffect := data.effects[effect.EffectId]

			numIsSpell := false
			if strings.Contains((*langs)["de"].Texts[currentEffect.DescriptionId], "Zauberspruchs #1") || strings.Contains((*langs)["de"].Texts[currentEffect.DescriptionId], "Zaubers #1") {
				numIsSpell = true
			}

			mappedEffect.Type = make(map[string]string)
			mappedEffect.Templated = make(map[string]string)
			var minMaxRemove int
			for _, lang := range utils.Languages {
				var diceNum int
				var diceSide int
				var value int

				diceNum = effect.MinimumValue

				diceSide = effect.MaximumValue

				value = effect.Value

				effectName := (*langs)[lang].Texts[currentEffect.DescriptionId]
				if lang == "de" {
					effectName = strings.ReplaceAll(effectName, "{~ps}{~zs}", "") // german has error in template
				}

				if effectName == "#1" { // is spell description from dicenum 1
					effectName = "-special spell-"
					mappedEffect.Min = 0
					mappedEffect.Max = 0
					mappedEffect.Type[lang] = effectName
					mappedEffect.Templated[lang] = (*langs)[lang].Texts[data.spells[diceNum].DescriptionId]
					mappedEffect.IsMeta = true
				} else {
					templatedName := effectName
					templatedName, minMaxRemove = NumSpellFormatter(templatedName, lang, data, langs, &diceNum, &diceSide, &value, currentEffect.DescriptionId, numIsSpell, currentEffect.UseDice)
					if templatedName == "" { // found effect that should be discarded for now
						break
					}
					templatedName = SingularPluralFormatter(templatedName, effect.MinimumValue, lang)

					effectName = DeleteDamageFormatter(effectName)
					effectName = SingularPluralFormatter(effectName, effect.MinimumValue, lang)

					mappedEffect.Min = diceNum
					mappedEffect.Max = diceSide
					mappedEffect.Type[lang] = effectName
					mappedEffect.Templated[lang] = templatedName
					mappedEffect.IsMeta = false
				}

				if lang == "en" && mappedEffect.Type[lang] == "" {
					break
				}
			}

			if mappedEffect.Type["en"] == "()" || mappedEffect.Type["en"] == "" {
				continue
			}

			mappedEffect.Active = isActiveEffect(mappedEffect.Type)
			searchTypeEn := mappedEffect.Type["en"]
			if mappedEffect.Active {
				searchTypeEn += " (Active)"
			}
			key, foundKey := utils.PersistedElements.Entries.GetKey(searchTypeEn)
			if foundKey {
				mappedEffect.ElementId = key.(int)
			} else {
				mappedEffect.ElementId = utils.PersistedElements.NextId
				utils.PersistedElements.Entries.Put(utils.PersistedElements.NextId, searchTypeEn)
				utils.PersistedElements.NextId++
			}

			mappedEffect.MinMaxIrrelevant = minMaxRemove

			mappedEffects = append(mappedEffects, mappedEffect)
		}
		if len(mappedEffects) > 0 {
			mappedAllEffects = append(mappedAllEffects, mappedEffects)
		}
	}
	if len(mappedAllEffects) == 0 {
		return nil
	}
	return mappedAllEffects
}

func ParseCondition(condition string, langs *map[string]LangDict, data *JSONGameData) []MappedMultilangCondition {
	if condition == "" || (!strings.Contains(condition, "&") && !strings.Contains(condition, "<") && !strings.Contains(condition, ">")) {
		return nil
	}

	condition = strings.ReplaceAll(condition, "\n", "")

	lower := strings.ToLower(condition)

	var outs []MappedMultilangCondition

	var parts []string
	if strings.Contains(lower, "&") {
		parts = strings.Split(lower, "&")
	} else {
		parts = []string{lower}
	}

	operators := []string{"<", ">", "=", "!"}

	for _, part := range parts {
		var out MappedMultilangCondition
		out.Templated = make(map[string]string)

		foundCond := false
		for _, operator := range operators { // try every known operator against it
			if strings.Contains(part, operator) {
				var outTmp MappedMultilangCondition
				outTmp.Templated = make(map[string]string)
				foundConditionElement := ConditionWithOperator(part, operator, langs, &out, data)
				if foundConditionElement {
					foundCond = true
				}
			}
		}

		if foundCond {
			outs = append(outs, out)
		}
	}

	if len(outs) == 0 {
		return nil
	}

	return outs
}

type HasId interface {
	GetID() int
}

func ParseRawDataPart[T HasId](fileSource string, result chan map[int]T) {
	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	dataPath := fmt.Sprintf("%s/data", path)

	file, err := os.ReadFile(fmt.Sprintf("%s/%s", dataPath, fileSource))
	if err != nil {
		fmt.Print(err)
	}
	fileStr := utils.CleanJSON(string(file))
	var fileJson []T
	err = json.Unmarshal([]byte(fileStr), &fileJson)
	if err != nil {
		fmt.Println(err)
	}
	items := make(map[int]T)
	for _, item := range fileJson {
		items[item.GetID()] = item
	}
	result <- items
}

func ParseRawData() *JSONGameData {
	var data JSONGameData
	itemChan := make(chan map[int]JSONGameItem)
	itemTypeChan := make(chan map[int]JSONGameItemType)
	itemSetsChan := make(chan map[int]JSONGameSet)
	itemEffectsChan := make(chan map[int]JSONGameEffect)
	itemBonusesChan := make(chan map[int]JSONGameBonus)
	itemRecipesChang := make(chan map[int]JSONGameRecipe)
	spellsChan := make(chan map[int]JSONGameSpell)
	spellTypesChan := make(chan map[int]JSONGameSpellType)
	areasChan := make(chan map[int]JSONGameArea)
	mountsChan := make(chan map[int]JSONGameMount)
	breedsChan := make(chan map[int]JSONGameBreed)
	mountFamilyChan := make(chan map[int]JSONGameMountFamily)
	npcsChan := make(chan map[int]JSONGameNPC)

	go func() {
		ParseRawDataPart("npcs.json", npcsChan)
	}()
	go func() {
		ParseRawDataPart("mount_family.json", mountFamilyChan)
	}()
	go func() {
		ParseRawDataPart("breeds.json", breedsChan)
	}()
	go func() {
		ParseRawDataPart("mounts.json", mountsChan)
	}()
	go func() {
		ParseRawDataPart("areas.json", areasChan)
	}()
	go func() {
		ParseRawDataPart("spell_types.json", spellTypesChan)
	}()
	go func() {
		ParseRawDataPart("spells.json", spellsChan)
	}()
	go func() {
		ParseRawDataPart("recipes.json", itemRecipesChang)
	}()
	go func() {
		ParseRawDataPart("items.json", itemChan)
	}()
	go func() {
		ParseRawDataPart("item_types.json", itemTypeChan)
	}()
	go func() {
		ParseRawDataPart("item_sets.json", itemSetsChan)
	}()
	go func() {
		ParseRawDataPart("bonuses.json", itemBonusesChan)
	}()
	go func() {
		ParseRawDataPart("effects.json", itemEffectsChan)
	}()

	data.Items = <-itemChan
	close(itemChan)

	data.bonuses = <-itemBonusesChan
	close(itemBonusesChan)

	data.effects = <-itemEffectsChan
	close(itemEffectsChan)

	data.ItemTypes = <-itemTypeChan
	close(itemTypeChan)

	data.Sets = <-itemSetsChan
	close(itemSetsChan)

	data.Recipes = <-itemRecipesChang
	close(itemRecipesChang)

	data.spells = <-spellsChan
	close(spellsChan)

	data.spellTypes = <-spellTypesChan
	close(spellTypesChan)

	data.areas = <-areasChan
	close(areasChan)

	data.Mounts = <-mountsChan
	close(mountsChan)

	data.classes = <-breedsChan
	close(breedsChan)

	data.MountFamilys = <-mountFamilyChan
	close(mountFamilyChan)

	data.npcs = <-npcsChan
	close(npcsChan)

	return &data
}

func ParseLangDict(langCode string) LangDict {
	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}

	dataPath := fmt.Sprintf("%s/data/languages", path)
	var data LangDict
	data.IdText = make(map[int]int)
	data.Texts = make(map[int]string)
	data.NameText = make(map[string]int)

	langFile, err := os.ReadFile(fmt.Sprintf("%s/lang_%s.json", dataPath, langCode))
	if err != nil {
		fmt.Print(err)
	}

	langFileStr := utils.CleanJSON(string(langFile))
	var langJson JSONLangDict
	err = json.Unmarshal([]byte(langFileStr), &langJson)
	if err != nil {
		fmt.Println(err)
	}

	for key, value := range langJson.IdText {
		keyParsed, err := strconv.Atoi(key)
		if err != nil {
			fmt.Println(err)
		}
		data.IdText[keyParsed] = value
	}

	for key, value := range langJson.Texts {
		keyParsed, err := strconv.Atoi(key)
		if err != nil {
			fmt.Println(err)
		}
		data.Texts[keyParsed] = value
	}
	data.NameText = langJson.NameText
	return data
}

func ParseRawLanguages() map[string]LangDict {
	data := make(map[string]LangDict)
	for _, lang := range utils.Languages {
		data[lang] = ParseLangDict(lang)
	}
	return data
}
