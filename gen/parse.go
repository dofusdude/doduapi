package gen

import (
	"encoding/json"
	"fmt"
	"github.com/dofusdude/api/update"
	"github.com/dofusdude/api/utils"
	"log"
	"os"
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
	go func() {
		defer wg.Done()
		DownloadMountsImages(gameData, utils.FileHashes, 6)
	}()

	languageData := ParseRawLanguages()
	log.Println("... completed parsing in", time.Since(startParsing))

	log.Println("mapping...")
	startMapping := time.Now()

	err := utils.LoadPersistedElements("db/elements.json")
	if err != nil {
		log.Fatal(err)
	}

	// ----
	mappedItems := MapItems(gameData, languageData)
	out, err := os.Create("data/MAPPED_ITEMS.json")
	if err != nil {
		fmt.Println(err)
	}
	defer out.Close()

	outBytes, err := json.MarshalIndent(*mappedItems, "", "    ")
	if err != nil {
		fmt.Println(err)
		return
	}

	out.Write(outBytes)

	// ----
	mappedMonsters := MapMonster(gameData, languageData)
	out, err = os.Create("data/MAPPED_MONSTER.json")
	if err != nil {
		fmt.Println(err)
	}
	defer out.Close()

	outBytes, err = json.MarshalIndent(*mappedMonsters, "", "    ")
	if err != nil {
		fmt.Println(err)
		return
	}

	out.Write(outBytes)

	// ----
	mappedMounts := MapMounts(gameData, languageData)
	out, err = os.Create("data/MAPPED_MOUNTS.json")
	if err != nil {
		fmt.Println(err)
	}
	defer out.Close()

	outBytes, err = json.MarshalIndent(*mappedMounts, "", "    ")
	if err != nil {
		fmt.Println(err)
		return
	}

	out.Write(outBytes)

	// ----
	mappedSets := MapSets(gameData, languageData)
	outSets, err := os.Create("data/MAPPED_SETS.json")
	if err != nil {
		fmt.Println(err)
	}
	defer outSets.Close()

	outSetsBytes, err := json.MarshalIndent(*mappedSets, "", "    ")
	if err != nil {
		fmt.Println(err)
		return
	}

	outSets.Write(outSetsBytes)

	// ----
	mappedRecipes := MapRecipes(gameData, languageData)
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

	err = utils.PersistElements("db/elements.json")
	if err != nil {
		log.Fatal(err)
	}

	wg.Wait() // mount images
	log.Println("... completed mapping in", time.Since(startMapping))
	mappedSets = nil
	mappedItems = nil
}

func DownloadMountImageWorker(files map[string]interface{}, workerSlice []JSONGameMount) {
	wg := sync.WaitGroup{}

	for _, mount := range workerSlice {
		wg.Add(1)
		go func(mountId int, wg *sync.WaitGroup) {
			defer wg.Done()
			var image update.HashFile
			image.Filename = fmt.Sprintf("content/gfx/mounts/%d.png", mountId)
			image.FriendlyName = fmt.Sprintf("data/img/mount/%d.png", mountId)
			err := update.DownloadHashImageFileInJson(files, image)
			if err != nil {
				fmt.Println(err)
			}
		}(mount.Id, &wg)

		wg.Add(1)
		go func(mountId int, wg *sync.WaitGroup) {
			defer wg.Done()
			var image update.HashFile
			image.Filename = fmt.Sprintf("content/gfx/mounts/%d.swf", mountId)
			image.FriendlyName = fmt.Sprintf("data/vector/mount/%d.swf", mountId)
			err := update.DownloadHashImageFileInJson(files, image)
			if err != nil {
				fmt.Println(err)
			}
		}(mount.Id, &wg)
	}

	wg.Wait()
}

func DownloadMountsImages(mounts *JSONGameData, hashJson map[string]interface{}, worker int) {
	main := hashJson["main"].(map[string]interface{})
	files := main["files"].(map[string]interface{})

	arr := utils.Values(mounts.Mounts)
	workerSlices := utils.PartitionSlice(arr, worker)

	wg := sync.WaitGroup{}
	for _, workerSlice := range workerSlices {
		wg.Add(1)
		go func(workerSlice []JSONGameMount) {
			defer wg.Done()
			DownloadMountImageWorker(files, workerSlice)
		}(workerSlice)
	}
	wg.Wait()
}

func ParseEffects(data *JSONGameData, allEffects [][]JSONGameItemPossibleEffect, langs *map[string]LangDict) [][]MappedMultilangEffect {
	var mappedAllEffects [][]MappedMultilangEffect
	for _, effects := range allEffects {
		var mappedEffects []MappedMultilangEffect
		for _, effect := range effects {

			var mappedEffect MappedMultilangEffect
			currentEffect := data.Effects[effect.EffectId]

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
					mappedEffect.Templated[lang] = (*langs)[lang].Texts[data.Spells[diceNum].DescriptionId]
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

				if lang != "en" {
					continue
				}

				if effectName == "()" {
					continue
				}

				key, foundKey := utils.PersistedElements.Entries.GetKey(effectName)
				if foundKey {
					mappedEffect.ElementId = key.(int)
				} else {
					utils.PersistedElements.Entries.Put(utils.PersistedElements.NextId, effectName)
					utils.PersistedElements.NextId++
				}
			}

			mappedEffect.MinMaxIrrelevant = minMaxRemove

			if mappedEffect.Type["en"] != "" && mappedEffect.Type["en"] != "()" {
				mappedEffects = append(mappedEffects, mappedEffect)
			}
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

func ParseRawData() *JSONGameData {
	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
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
	subAreasChan := make(chan map[int]JSONGameSubArea)
	superAreasChan := make(chan map[int]JSONGameSuperArea)
	mountsChan := make(chan map[int]JSONGameMount)
	breedsChan := make(chan map[int]JSONGameClass)
	mountFamilyChan := make(chan map[int]JSONGameMountFamily)
	npcsChan := make(chan map[int]JSONGameNPC)
	monstersChan := make(chan map[int]JSONGameMonster)
	monstersRacesChan := make(chan map[int]JSONGameMonsterRace)
	monstersSuperRacesChan := make(chan map[int]JSONGameMonsterSuperRace)

	dataPath := fmt.Sprintf("%s/data", path)

	// monsters
	go func() {
		file, err := os.ReadFile(fmt.Sprintf("%s/%s", dataPath, "monsters.json"))
		if err != nil {
			fmt.Print(err)
		}
		fileStr := utils.CleanJSON(string(file))
		var fileJson []JSONGameMonster
		err = json.Unmarshal([]byte(fileStr), &fileJson)
		if err != nil {
			fmt.Println(err)
		}
		items := make(map[int]JSONGameMonster)
		for _, item := range fileJson {
			items[item.Id] = item
		}
		monstersChan <- items
	}()

	// monster super races
	go func() {
		file, err := os.ReadFile(fmt.Sprintf("%s/%s", dataPath, "monster_super_races.json"))
		if err != nil {
			fmt.Print(err)
		}
		fileStr := utils.CleanJSON(string(file))
		var fileJson []JSONGameMonsterSuperRace
		err = json.Unmarshal([]byte(fileStr), &fileJson)
		if err != nil {
			fmt.Println(err)
		}
		items := make(map[int]JSONGameMonsterSuperRace)
		for _, item := range fileJson {
			items[item.Id] = item
		}
		monstersSuperRacesChan <- items
	}()

	// monster races
	go func() {
		file, err := os.ReadFile(fmt.Sprintf("%s/%s", dataPath, "monster_races.json"))
		if err != nil {
			fmt.Print(err)
		}
		fileStr := utils.CleanJSON(string(file))
		var fileJson []JSONGameMonsterRace
		err = json.Unmarshal([]byte(fileStr), &fileJson)
		if err != nil {
			fmt.Println(err)
		}
		items := make(map[int]JSONGameMonsterRace)
		for _, item := range fileJson {
			items[item.Id] = item
		}
		monstersRacesChan <- items
	}()

	// subareas
	go func() {
		file, err := os.ReadFile(fmt.Sprintf("%s/%s", dataPath, "sub_areas.json"))
		if err != nil {
			fmt.Print(err)
		}
		fileStr := utils.CleanJSON(string(file))
		var fileJson []JSONGameSubArea
		err = json.Unmarshal([]byte(fileStr), &fileJson)
		if err != nil {
			fmt.Println(err)
		}
		items := make(map[int]JSONGameSubArea)
		for _, item := range fileJson {
			items[item.Id] = item
		}
		subAreasChan <- items
	}()

	// super areas
	go func() {
		file, err := os.ReadFile(fmt.Sprintf("%s/%s", dataPath, "super_areas.json"))
		if err != nil {
			fmt.Print(err)
		}
		fileStr := utils.CleanJSON(string(file))
		var fileJson []JSONGameSuperArea
		err = json.Unmarshal([]byte(fileStr), &fileJson)
		if err != nil {
			fmt.Println(err)
		}
		items := make(map[int]JSONGameSuperArea)
		for _, item := range fileJson {
			items[item.Id] = item
		}
		superAreasChan <- items
	}()

	// npcs
	go func() {
		file, err := os.ReadFile(fmt.Sprintf("%s/%s", dataPath, "npcs.json"))
		if err != nil {
			fmt.Print(err)
		}
		fileStr := utils.CleanJSON(string(file))
		var fileJson []JSONGameNPC
		err = json.Unmarshal([]byte(fileStr), &fileJson)
		if err != nil {
			fmt.Println(err)
		}
		items := make(map[int]JSONGameNPC)
		for _, item := range fileJson {
			items[item.Id] = item
		}
		npcsChan <- items
	}()

	// mount family
	go func() {
		file, err := os.ReadFile(fmt.Sprintf("%s/%s", dataPath, "mount_family.json"))
		if err != nil {
			fmt.Print(err)
		}
		fileStr := utils.CleanJSON(string(file))
		var fileJson []JSONGameMountFamily
		err = json.Unmarshal([]byte(fileStr), &fileJson)
		if err != nil {
			fmt.Println(err)
		}
		items := make(map[int]JSONGameMountFamily)
		for _, item := range fileJson {
			items[item.Id] = item
		}
		mountFamilyChan <- items
	}()

	// breeds
	go func() {
		file, err := os.ReadFile(fmt.Sprintf("%s/%s", dataPath, "breeds.json"))
		if err != nil {
			fmt.Print(err)
		}
		fileStr := utils.CleanJSON(string(file))
		var fileJson []JSONGameClass
		err = json.Unmarshal([]byte(fileStr), &fileJson)
		if err != nil {
			fmt.Println(err)
		}
		items := make(map[int]JSONGameClass)
		for _, item := range fileJson {
			items[item.Id] = item
		}
		breedsChan <- items
	}()

	// mounts
	go func() {
		file, err := os.ReadFile(fmt.Sprintf("%s/%s", dataPath, "mounts.json"))
		if err != nil {
			fmt.Print(err)
		}
		fileStr := utils.CleanJSON(string(file))
		var fileJson []JSONGameMount
		err = json.Unmarshal([]byte(fileStr), &fileJson)
		if err != nil {
			fmt.Println(err)
		}
		items := make(map[int]JSONGameMount)
		for _, item := range fileJson {
			items[item.Id] = item
		}
		mountsChan <- items
	}()

	// areas
	go func() {
		file, err := os.ReadFile(fmt.Sprintf("%s/%s", dataPath, "areas.json"))
		if err != nil {
			fmt.Print(err)
		}
		fileStr := utils.CleanJSON(string(file))
		var fileJson []JSONGameArea
		err = json.Unmarshal([]byte(fileStr), &fileJson)
		if err != nil {
			fmt.Println(err)
		}
		items := make(map[int]JSONGameArea)
		for _, item := range fileJson {
			items[item.Id] = item
		}
		areasChan <- items
	}()

	// spells
	go func() {
		file, err := os.ReadFile(fmt.Sprintf("%s/%s", dataPath, "spells.json"))
		if err != nil {
			fmt.Print(err)
		}
		fileStr := utils.CleanJSON(string(file))
		var fileJson []JSONGameSpell
		err = json.Unmarshal([]byte(fileStr), &fileJson)
		if err != nil {
			fmt.Println(err)
		}
		items := make(map[int]JSONGameSpell)
		for _, item := range fileJson {
			items[item.Id] = item
		}
		spellsChan <- items
	}()

	// spell types
	go func() {
		file, err := os.ReadFile(fmt.Sprintf("%s/%s", dataPath, "spell_types.json"))
		if err != nil {
			fmt.Print(err)
		}
		fileStr := utils.CleanJSON(string(file))
		var fileJson []JSONGameSpellType
		err = json.Unmarshal([]byte(fileStr), &fileJson)
		if err != nil {
			fmt.Println(err)
		}
		items := make(map[int]JSONGameSpellType)
		for _, item := range fileJson {
			items[item.Id] = item
		}
		spellTypesChan <- items
	}()

	// recipes
	go func() {
		file, err := os.ReadFile(fmt.Sprintf("%s/%s", dataPath, "recipes.json"))
		if err != nil {
			fmt.Print(err)
		}
		fileStr := utils.CleanJSON(string(file))
		var fileJson []JSONGameRecipe
		err = json.Unmarshal([]byte(fileStr), &fileJson)
		if err != nil {
			fmt.Println(err)
		}
		items := make(map[int]JSONGameRecipe)
		for _, item := range fileJson {
			items[item.Id] = item
		}
		itemRecipesChang <- items
	}()

	// items
	go func() {
		itemsFile, err := os.ReadFile(fmt.Sprintf("%s/%s", dataPath, "items.json"))
		if err != nil {
			fmt.Print(err)
		}
		itemsFileStr := utils.CleanJSON(string(itemsFile))
		var itemsJson []JSONGameItem
		err = json.Unmarshal([]byte(itemsFileStr), &itemsJson)
		if err != nil {
			fmt.Println(err)
		}
		items := make(map[int]JSONGameItem)
		for _, item := range itemsJson {
			items[item.Id] = item
		}
		itemChan <- items
	}()

	// item_types
	go func() {
		itemsTypeFile, err := os.ReadFile(fmt.Sprintf("%s/%s", dataPath, "item_types.json"))
		if err != nil {
			fmt.Print(err)
		}
		itemTypesFileStr := utils.CleanJSON(string(itemsTypeFile))
		var itemTypesJson []JSONGameItemType
		err = json.Unmarshal([]byte(itemTypesFileStr), &itemTypesJson)
		if err != nil {
			fmt.Println(err)
		}
		itemTypes := make(map[int]JSONGameItemType)
		for _, itemType := range itemTypesJson {
			itemTypes[itemType.Id] = itemType
		}
		itemTypeChan <- itemTypes
	}()

	// item_sets
	go func() {
		itemsSetsFile, err := os.ReadFile(fmt.Sprintf("%s/%s", dataPath, "item_sets.json"))
		if err != nil {
			fmt.Print(err)
		}
		itemSetsFileStr := utils.CleanJSON(string(itemsSetsFile))
		var setsJson []JSONGameSet
		err = json.Unmarshal([]byte(itemSetsFileStr), &setsJson)
		if err != nil {
			fmt.Println(err)
		}
		sets := make(map[int]JSONGameSet)
		for _, set := range setsJson {
			sets[set.Id] = set
		}

		itemSetsChan <- sets
	}()

	// bonuses
	go func() {
		bonusesFile, err := os.ReadFile(fmt.Sprintf("%s/%s", dataPath, "bonuses.json"))
		if err != nil {
			fmt.Print(err)
		}
		bonusesFileStr := utils.CleanJSON(string(bonusesFile))
		var bonusesJson []JSONGameBonus
		err = json.Unmarshal([]byte(bonusesFileStr), &bonusesJson)
		if err != nil {
			fmt.Println(err)
		}
		bonuses := make(map[int]JSONGameBonus)
		for _, bonus := range bonusesJson {
			bonuses[bonus.Id] = bonus
		}
		itemBonusesChan <- bonuses
	}()

	// effects
	go func() {
		effectsFile, err := os.ReadFile(fmt.Sprintf("%s/%s", dataPath, "effects.json"))
		if err != nil {
			fmt.Print(err)
		}
		effectsFileStr := utils.CleanJSON(string(effectsFile))
		var effectsJson []JSONGameEffect
		err = json.Unmarshal([]byte(effectsFileStr), &effectsJson)
		if err != nil {
			fmt.Println(err)
		}
		effects := make(map[int]JSONGameEffect)
		for _, effect := range effectsJson {
			effects[effect.Id] = effect
		}
		itemEffectsChan <- effects
	}()

	data.Monsters = <-monstersChan
	close(monstersChan)

	data.ItemMonsterDrops = make(map[int][]int)
	for _, monster := range data.Monsters {
		for _, drop := range monster.Drops {
			if _, ok := data.ItemMonsterDrops[drop.ObjectId]; !ok {
				data.ItemMonsterDrops[drop.ObjectId] = []int{}
			}
			data.ItemMonsterDrops[drop.ObjectId] = append(data.ItemMonsterDrops[drop.ObjectId], monster.Id)
		}
	}

	data.Items = <-itemChan
	close(itemChan)

	data.Bonuses = <-itemBonusesChan
	close(itemBonusesChan)

	data.Effects = <-itemEffectsChan
	close(itemEffectsChan)

	data.ItemTypes = <-itemTypeChan
	close(itemTypeChan)

	data.Sets = <-itemSetsChan
	close(itemSetsChan)

	data.Recipes = <-itemRecipesChang
	close(itemRecipesChang)

	data.Spells = <-spellsChan
	close(spellsChan)

	data.SpellTypes = <-spellTypesChan
	close(spellTypesChan)

	data.Areas = <-areasChan
	close(areasChan)

	data.Mounts = <-mountsChan
	close(mountsChan)

	data.Classes = <-breedsChan
	close(breedsChan)

	data.MountFamilys = <-mountFamilyChan
	close(mountFamilyChan)

	data.Npcs = <-npcsChan
	close(npcsChan)

	data.MonsterRaces = <-monstersRacesChan
	close(monstersRacesChan)

	data.MonsterSuperRaces = <-monstersSuperRacesChan
	close(monstersSuperRacesChan)

	data.SubAreas = <-subAreasChan
	close(subAreasChan)

	data.SuperAreas = <-superAreasChan
	close(superAreasChan)

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

func ParseRawLanguages() *map[string]LangDict {
	data := make(map[string]LangDict)

	chanDe := make(chan LangDict)
	go func() {
		chanDe <- ParseLangDict("de")
	}()

	chanEn := make(chan LangDict)
	go func() {
		chanEn <- ParseLangDict("en")
	}()

	chanFr := make(chan LangDict)
	go func() {
		chanFr <- ParseLangDict("fr")
	}()

	chanEs := make(chan LangDict)
	go func() {
		chanEs <- ParseLangDict("es")
	}()

	chanPt := make(chan LangDict)
	go func() {
		chanPt <- ParseLangDict("pt")
	}()

	chanIt := make(chan LangDict)
	go func() {
		chanIt <- ParseLangDict("it")
	}()

	data["de"] = <-chanDe
	close(chanDe)

	data["en"] = <-chanEn
	close(chanEn)

	data["fr"] = <-chanFr
	close(chanFr)

	data["es"] = <-chanEs
	close(chanEs)

	data["pt"] = <-chanPt
	close(chanPt)

	data["it"] = <-chanIt
	close(chanIt)

	return &data
}
