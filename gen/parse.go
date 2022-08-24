package gen

import (
	"encoding/json"
	"fmt"
	"github.com/dofusdude/api/utils"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func Parse() {
	log.Println("parsing...")
	startParsing := time.Now()
	gameData := ParseRawData()
	languageData := ParseRawLanguages()
	log.Println("... completed parsing in", time.Since(startParsing))

	log.Println("mapping...")
	startMapping := time.Now()

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

	log.Println("... completed mapping in", time.Since(startMapping))
	mappedSets = nil
	mappedItems = nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func ParseEffects(data *JSONGameData, allEffects [][]JSONGameItemPossibleEffect, langs *map[string]LangDict) [][]MappedMultilangEffect {
	var mappedAllEffects [][]MappedMultilangEffect
	for _, effects := range allEffects {
		var mappedEffects []MappedMultilangEffect
		for _, effect := range effects {

			var mappedEffect MappedMultilangEffect
			currentEffect := data.effects[effect.EffectId]
			//if !current_effect.Active {
			//	continue
			//}

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

				if lang == "en" && mappedEffect.Type[lang] == "" {
					break
				}

			}

			mappedEffect.MinMaxIrrelevant = minMaxRemove

			if mappedEffect.Type["en"] != "" {
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

func PrepareTextForRegex(input string) string {
	input = strings.ReplaceAll(input, "{~1~2 -}", "{~1~2 - }")
	input = strings.ReplaceAll(input, "{~1~2 to}level", "{~1~2 to } level") // {~1~2 to}level
	input = strings.ReplaceAll(input, "{~1~2 to}", "{~1~2 to }")
	input = strings.ReplaceAll(input, "\"\"", "")
	return input
}

func PrepareAndCreateRangeRegex(input string, extract bool) (string, *regexp.Regexp) {
	var regexStr string
	combiningWords := "(und|et|and|bis|to|a|à|-|auf)"
	if extract {
		regexStr = fmt.Sprintf("{~1~2 (%s [-,+]?)}", combiningWords)
	} else {
		regexStr = fmt.Sprintf("[-,+]?#1{~1~2 %s [-,+]?}#2", combiningWords)
	}

	concatRegex := regexp.MustCompile(regexStr)

	return PrepareTextForRegex(input), concatRegex
}

// returns info about min max with in. -1 "only_min", -2 "no_min_max"
func NumSpellFormatter(input string, lang string, gameData *JSONGameData, langs *map[string]LangDict, diceNum *int, diceSide *int, value *int, effectNameId int, numIsSpell bool, useDice bool) (string, int) {
	diceNumIsSpellId := *diceNum > 12000 || numIsSpell
	diceSideIsSpellId := *diceSide > 12000
	valueIsSpellId := *value > 12000

	onlyNoMinMax := 0

	// when + xp
	if !useDice && *diceNum == 0 && *value == 0 && *diceSide != 0 {
		*value = *diceSide
		*diceSide = 0
	}

	delValue := false

	input, concatRegex := PrepareAndCreateRangeRegex(input, true)
	numSigned, sideSigned := ParseSigness(input)
	concatEntries := concatRegex.FindAllStringSubmatch(input, -1)

	if *diceSide == 0 { // only replace #1 with dice_num
		for _, extracted := range concatEntries {
			input = strings.ReplaceAll(input, extracted[0], "")
		}
	} else {
		for _, extracted := range concatEntries {
			input = strings.ReplaceAll(input, extracted[0], fmt.Sprintf(" %s", extracted[1]))
		}
	}

	num1Regex := regexp.MustCompile("([-,+]?)#1")
	num1Entries := num1Regex.FindAllStringSubmatch(input, -1)
	for _, extracted := range num1Entries {
		var diceNumStr string
		if diceNumIsSpellId {
			diceNumStr = (*langs)[lang].Texts[gameData.spells[*diceNum].NameId]
		} else {
			diceNumStr = fmt.Sprint(*diceNum)
		}
		input = strings.ReplaceAll(input, extracted[0], fmt.Sprintf("%s%s", extracted[1], diceNumStr))
	}

	if *diceSide == 0 {
		input = strings.ReplaceAll(input, "#2", "")
	} else {
		var diceSideStr string
		if diceSideIsSpellId {
			diceSideStr = (*langs)[lang].Texts[gameData.spells[*diceSide].NameId]
			//del_dice_side = true
		} else {
			diceSideStr = fmt.Sprint(*diceSide)
		}
		input = strings.ReplaceAll(input, "#2", diceSideStr)
	}

	var valueStr string
	if valueIsSpellId {
		valueStr = (*langs)[lang].Texts[gameData.spells[*value].NameId]
		delValue = true
	} else {
		valueStr = fmt.Sprint(*value)
	}
	if effectNameId == 427090 { // go to <npc> for more info
		return "", -2
	}
	input = strings.ReplaceAll(input, "#3", valueStr)

	if delValue {
		*diceNum = min(*diceNum, *diceSide)
	}

	/*
		// reorder the values so diceNum is always the smallest, diceSide the bigger and value the spellId
		if dice_num_is_spell_id && !value_is_spell_id && !dice_side_is_spell_id {
			spell_id := *dice_num
			if *value == 0 {
				*dice_num = *dice_side
				*dice_side = 0
			} else {
				*dice_num = min(*value, *dice_side)
				*dice_side = max(*value, *dice_side)
			}
			*value = spell_id
		}

		if dice_side_is_spell_id && !value_is_spell_id && !dice_num_is_spell_id {
			spell_id := *dice_num
			if *value == 0 {
				*dice_side = 0
			} else {
				*dice_num = min(*value, *dice_num)
				*dice_side = max(*value, *dice_num)
			}
			*value = spell_id
		}
	*/
	if !useDice {
		// avoid min = 0, max > x
		if *diceNum == 0 && *diceSide != 0 {
			*diceNum = *diceSide
			*diceSide = 0
		}
	}

	if *diceNum == 0 && *diceSide == 0 {
		onlyNoMinMax = -2
	}

	if *diceNum != 0 && *diceSide == 0 {
		onlyNoMinMax = -1
	}

	input = strings.TrimSpace(input)

	if numSigned {
		*diceNum *= -1
	}

	if sideSigned {
		*diceSide *= -1
	}

	return input, onlyNoMinMax
}

func ParseSigness(input string) (bool, bool) {
	numSigness := false
	sideSigness := false

	regexNum := regexp.MustCompile("(([+,-])?#1)")
	entriesNum := regexNum.FindAllStringSubmatch(input, -1)
	for _, extracted := range entriesNum {
		for _, entry := range extracted {
			if entry == "-" {
				numSigness = true
			}
		}
	}

	regexSide := regexp.MustCompile("([+,-])?}?#2")
	entriesSide := regexSide.FindAllStringSubmatch(input, -1)
	for _, extracted := range entriesSide {
		for _, entry := range extracted {
			if entry == "-" {
				sideSigness = true
			}
		}
	}

	return numSigness, sideSigness
}

func DeleteDamageFormatter(input string) string {
	input, regex := PrepareAndCreateRangeRegex(input, false)
	if strings.Contains(input, "+#1{~1~2 to } level #2") {
		return "level"
	}

	input = strings.ReplaceAll(input, "#1{~1~2 -}#2", "#1{~1~2 - }#2") // bug from ankama
	input = regex.ReplaceAllString(input, "")

	numRegex := regexp.MustCompile(" ?#1")
	input = numRegex.ReplaceAllString(input, "")

	sideRegex := regexp.MustCompile(" ?#2")
	input = sideRegex.ReplaceAllString(input, "")

	valueRegex := regexp.MustCompile(" ?#3")
	input = valueRegex.ReplaceAllString(input, "")

	input = strings.TrimSpace(input)
	return input
}

func SingularPluralFormatter(input string, amount int, lang string) string {
	str := strings.ReplaceAll(input, "{~s}", "") // avoid only s without what to append
	str = strings.ReplaceAll(str, "{~p}", "")    // same

	// delete unknown z
	unknownZRegex := regexp.MustCompile("{~z[^}]*}")
	str = unknownZRegex.ReplaceAllString(str, "")

	var indicator rune

	if amount > 1 {
		indicator = 'p'
	} else {
		indicator = 's'
	}

	indicators := []rune{'s', 'p'}
	var regexps []*regexp.Regexp
	for _, indicatorIt := range indicators {
		regex := fmt.Sprintf("{~%c([^}]*)}", indicatorIt) // capturing with everything inside ()
		regexExtract := regexp.MustCompile(regex)
		regexps = append(regexps, regexExtract)

		//	if lang == "es" || lang == "pt" {
		if indicatorIt != indicator {
			continue
		}
		extractedEntries := regexExtract.FindAllStringSubmatch(str, -1)
		for _, extracted := range extractedEntries {
			str = strings.ReplaceAll(str, extracted[0], extracted[1])
		}
	}

	for _, regexIt := range regexps {
		str = regexIt.ReplaceAllString(str, "")
	}

	return str
}

func ElementFromCode(code string) int {
	code = strings.ToLower(code)

	switch code {
	case "cs":
		return 501945 // "Strength"
	case "ci":
		return 501944 // "Intelligence"
	case "cv":
		return 501947 // "Vitality"
	case "ca":
		return 501941 // "Agility"
	case "cc":
		return 501942 // "Chance"
	case "cw":
		return 501946 // "Wisdom"
	case "pk":
		return 422874 // "Set-Bonus"
	case "pl":
		return 837224 // "Mindestens Stufe %1"
	case "cm":
		return 67248 // "Bewegungsp. (BP)"
	case "cp":
		return 67755 // "Aktionsp. (AP)"
	case "po":
		return 335357 // Anderes Gebiet als: %1
	case "pf":
		return 644231 // Nicht ausgerüstetes %1-Reittier
	//case "": // Ps=1
	//	return 644230 // Ausgerüstetes %1-Reittier
	case "pa":
		return 66566 // Gesinunngsstufe
	//case "":
	//	return 637203 // Kein ausgerüstetes %1-Reittier haben
	case "of":
		return 637212 // Ein ausgerüstetes %1-Reittier haben
	case "pz":
		return 66351 // Abonniert sein
	}

	return -1
}

func ConditionWithOperator(input string, operator string, langs *map[string]LangDict, out *MappedMultiangCondition, data *JSONGameData) bool {
	partSplit := strings.Split(input, operator)
	rawElement := ElementFromCode(partSplit[0])
	if rawElement == -1 {
		return false
	}
	out.Element = strings.ToLower(partSplit[0])
	out.Value, _ = strconv.Atoi(partSplit[1])
	for _, lang := range utils.Languages {
		lang_str := (*langs)[lang].Texts[rawElement]

		switch rawElement {
		case 837224: // %1 replace
			int_val, _ := strconv.Atoi(partSplit[1])
			lang_str = strings.ReplaceAll(lang_str, "%1", fmt.Sprint(int_val+1))
			break
		case 335357: // anderes gebiet als %1
			lang_str = strings.ReplaceAll(lang_str, "%1", (*langs)[lang].Texts[data.areas[out.Value].NameId])
			break
		case 637212: // reittier %1
		case 644231:
			lang_str = strings.ReplaceAll(lang_str, "%1", (*langs)[lang].Texts[data.Mounts[out.Value].NameId])
			break
		}

		out.Templated[lang] = lang_str
	}
	out.Operator = operator
	return true
}

func ParseCondition(condition string, langs *map[string]LangDict, data *JSONGameData) []MappedMultiangCondition {
	if condition == "" || (!strings.Contains(condition, "&") && !strings.Contains(condition, "<") && !strings.Contains(condition, ">")) {
		return nil
	}

	condition = strings.ReplaceAll(condition, "\n", "")

	lower := strings.ToLower(condition)

	var outs []MappedMultiangCondition

	var parts []string
	if strings.Contains(lower, "&") {
		parts = strings.Split(lower, "&")
	} else {
		parts = []string{lower}
	}

	operators := []string{"<", ">", "=", "!"}

	for _, part := range parts {
		var out MappedMultiangCondition
		out.Templated = make(map[string]string)

		foundCond := false
		for _, operator := range operators { // try every known operator against it
			if strings.Contains(part, operator) {
				var outTmp MappedMultiangCondition
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

func cleanJSON(jsonStr string) string {
	jsonStr = strings.ReplaceAll(jsonStr, "NaN", "null")
	jsonStr = strings.ReplaceAll(jsonStr, "\"null\"", "null")
	jsonStr = strings.ReplaceAll(jsonStr, " ", " ")
	return jsonStr
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
	mountsChan := make(chan map[int]JSONGameMount)
	breedsChan := make(chan map[int]JSONGameBreed)
	mountFamilyChan := make(chan map[int]JSONGameMountFamily)
	npcsChan := make(chan map[int]JSONGameNPC)

	dataPath := fmt.Sprintf("%s/data", path)

	// npcs
	go func() {
		file, err := os.ReadFile(fmt.Sprintf("%s/%s", dataPath, "npcs.json"))
		if err != nil {
			fmt.Print(err)
		}
		fileStr := cleanJSON(string(file))
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
		fileStr := cleanJSON(string(file))
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
		fileStr := cleanJSON(string(file))
		var fileJson []JSONGameBreed
		err = json.Unmarshal([]byte(fileStr), &fileJson)
		if err != nil {
			fmt.Println(err)
		}
		items := make(map[int]JSONGameBreed)
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
		fileStr := cleanJSON(string(file))
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
		fileStr := cleanJSON(string(file))
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
		fileStr := cleanJSON(string(file))
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
		fileStr := cleanJSON(string(file))
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
		fileStr := cleanJSON(string(file))
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
		itemsFileStr := cleanJSON(string(itemsFile))
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
		itemTypesFileStr := cleanJSON(string(itemsTypeFile))
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
		itemSetsFileStr := cleanJSON(string(itemsSetsFile))
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
		bonusesFileStr := cleanJSON(string(bonusesFile))
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
		effectsFileStr := cleanJSON(string(effectsFile))
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

	data.Mount_familys = <-mountFamilyChan
	close(mountFamilyChan)

	data.npcs = <-npcsChan
	close(npcsChan)

	return &data
}

func ParseLangDict(lang_code string) LangDict {
	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}

	dataPath := fmt.Sprintf("%s/data/languages", path)
	var data LangDict
	data.IdText = make(map[int]int)
	data.Texts = make(map[int]string)
	data.NameText = make(map[string]int)

	langFile, err := os.ReadFile(fmt.Sprintf("%s/lang_%s.json", dataPath, lang_code))
	if err != nil {
		fmt.Print(err)
	}

	langFileStr := cleanJSON(string(langFile))
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
