package update

import (
	"encoding/json"
	"fmt"
	"github.com/dofusdude/api/utils"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type GameVersions struct {
	main string
	beta string
}

type HashFile struct {
	hash         string
	filename     string
	friendlyName string
}

func DownloadUpdatesIfAvailable(force bool) bool {
	path, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	configPath := fmt.Sprintf("%s/config.json", path)

	config := utils.GetConfig(configPath)
	versions := GetVersion("https://launcher.cdn.ankama.com/cytrus.json")

	if !force && config.CurrentVersion == versions.main {
		return false
	}
	cleanUp()

	gameHashesUrl := fmt.Sprintf("https://launcher.cdn.ankama.com/dofus/releases/main/windows/%s.json", versions.main)
	hashResponse, err := http.Get(gameHashesUrl)
	if err != nil {
		log.Fatalln(err)
	}

	hashBody, err := io.ReadAll(hashResponse.Body)
	if err != nil {
		log.Fatalln(err)
	}

	var hashJson map[string]interface{}
	err = json.Unmarshal(hashBody, &hashJson)
	if err != nil {
		log.Fatalln(err)
	}

	var waitGrp sync.WaitGroup

	waitGrp.Add(1)
	go func() {
		defer waitGrp.Done()
		DownloadLanguages(hashJson)
	}()

	waitGrp.Add(1)
	go func() {
		defer waitGrp.Done()
		DownloadImagesLauncher(hashJson)
	}()

	waitGrp.Add(1)
	go func() {
		defer waitGrp.Done()
		DownloadItems(hashJson)
	}()

	waitGrp.Wait()

	os.RemoveAll("data/tmp")
	os.Mkdir("data/tmp", 0755)

	config.CurrentVersion = versions.main
	utils.SaveConfig(config, configPath)

	return true
}

func GetVersion(path string) GameVersions {
	versionResponse, err := http.Get(path)
	if err != nil {
		log.Fatalln(err)
	}

	versionBody, err := ioutil.ReadAll(versionResponse.Body)
	if err != nil {
		log.Fatalln(err)
	}

	var versionJson map[string]interface{}
	err = json.Unmarshal(versionBody, &versionJson)
	if err != nil {
		fmt.Println("error:", err)
	}

	games := versionJson["games"].(map[string]interface{})
	dofus := games["dofus"].(map[string]interface{})
	platform := dofus["platforms"].(map[string]interface{})
	windows := platform["windows"].(map[string]interface{})

	var gameVersions GameVersions
	gameVersions.beta = windows["beta"].(string)
	gameVersions.main = windows["main"].(string)

	return gameVersions
}

func DownloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func DownloadHashFile(file HashFile) {
	url := fmt.Sprintf("https://launcher.cdn.ankama.com/dofus/hashes/%s/%s", file.hash[:2], file.hash)
	DownloadFile(file.friendlyName, url)
}

func cleanUp() {
	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}

	files := []string{
		"data/effects.json",
		"data/items.json",
		"data/item_sets.json",
		"data/item_types.json",
		"data/bouses.json",
		"data/recipes.json",
		"data/spells.json",
		"data/spell_types.json",
		"data/areas.json",
		"data/monsters.json",
		"data/companion_spells.json",
		"data/companion_chars.json",
		"data/almanax.json",
		"data/idols.json",
		"data/companions.json",
		"data/mount_family.json",
		"data/npcs.json",
		"data/monsters.json",
		"data/server_game_types.json",
		"data/chars_categories.json",
		"data/create_bone_types.json",
		"data/create_bone_overrides.json",
		"data/evol_effects.json",
		"data/bonus_criterions.json",

		"data/MAPPED_ITEMS.json",
		"data/MAPPED_SETS.json",
		"data/MAPPED_RECIPES.json",
		"data/MAPPED_MOUNTS.json",
	}
	for _, lang := range utils.Languages {
		langJson := fmt.Sprintf("data/languages/lang_%s.json", lang)
		files = append(files, langJson)
	}

	for _, file := range files {
		absPath := fmt.Sprintf("%s/%s", path, file)
		os.Remove(absPath)
	}

	//os.RemoveAll("data/img") // keep old images, override with new ones, else they are unavaible while updating
	//os.Mkdir("data/img", 0755)

	meiliClient := utils.CreateMeiliClient()

	for _, lang := range utils.Languages {
		taskItemsDelete, err := meiliClient.DeleteIndex(fmt.Sprintf("all_items-%s", lang))
		if err != nil {
			log.Println(err)
		}
		taskSetsDelete, err := meiliClient.DeleteIndex(fmt.Sprintf("sets-%s", lang))
		if err != nil {
			log.Println(err)
		}
		taskMountsDelete, err := meiliClient.DeleteIndex(fmt.Sprintf("mounts-%s", lang))
		if err != nil {
			log.Println(err)
		}

		meiliClient.WaitForTask(taskItemsDelete.TaskUID)
		meiliClient.WaitForTask(taskSetsDelete.TaskUID)
		meiliClient.WaitForTask(taskMountsDelete.TaskUID)
	}

}

func Unpack(filepath string, dest_dir_rel string, fileType string) {
	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}

	absConvertCmd := fmt.Sprintf("%s/PyDofus/%s_unpack.py", path, fileType)
	absFilePath := fmt.Sprintf("%s/%s", path, filepath)
	absOutPath := strings.Replace(absFilePath, fileType, "json", 1)
	filenameParts := strings.Split(filepath, "/")
	filename := filenameParts[len(filenameParts)-1]
	outFile := strings.Replace(filename, fileType, "json", 1)
	finalOutPath := fmt.Sprintf("%s/%s/%s", path, dest_dir_rel, outFile)

	exec.Command("/usr/local/bin/python3", absConvertCmd, absFilePath).Run()
	err = os.Rename(absOutPath, finalOutPath)
	if err != nil {
		log.Println(err)
	}
}

func DownloadHashImageFileInJson(files map[string]interface{}, hashFile HashFile) {
	file := files[hashFile.filename].(map[string]interface{})
	hashFile.hash = file["hash"].(string)
	DownloadHashFile(hashFile)

}

func DownloadHashFileInJson(files map[string]interface{}, hashFile HashFile, destDirRel string, fileType string) {
	hashfile := files[hashFile.filename].(map[string]interface{})
	hashFile.hash = hashfile["hash"].(string)
	DownloadHashFile(hashFile)

	Unpack(hashFile.friendlyName, destDirRel, fileType)
}

func DownloadImagesLauncher(hashJson map[string]interface{}) {
	main := hashJson["main"].(map[string]interface{})
	files := main["files"].(map[string]interface{})

	log.Println("loading item images...")

	wg := sync.WaitGroup{}

	// item images 0
	var itemImages0 HashFile
	itemImages0.filename = "content/gfx/items/bitmap0.d2p"
	itemImages0.friendlyName = "data/tmp/bitmaps_0.d2p"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashImageFileInJson(files, itemImages0)
	}()

	var itemImages1 HashFile
	itemImages1.filename = "content/gfx/items/bitmap0_1.d2p"
	itemImages1.friendlyName = "data/tmp/bitmaps_1.d2p"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashImageFileInJson(files, itemImages1)
	}()

	var itemImages2 HashFile
	itemImages2.filename = "content/gfx/items/bitmap1.d2p"
	itemImages2.friendlyName = "data/tmp/bitmaps_2.d2p"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashImageFileInJson(files, itemImages2)
	}()

	var itemImages3 HashFile
	itemImages3.filename = "content/gfx/items/bitmap1_1.d2p"
	itemImages3.friendlyName = "data/tmp/bitmaps_3.d2p"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashImageFileInJson(files, itemImages3)
	}()

	var itemImages4 HashFile
	itemImages4.filename = "content/gfx/items/bitmap1_2.d2p"
	itemImages4.friendlyName = "data/tmp/bitmaps_4.d2p"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashImageFileInJson(files, itemImages4)
	}()

	/*
		// monsters images
		wg.Add(1)
		var images_1 HashFile
		images_1.filename = "content/gfx/monsters/monsters0.d2p"
		images_1.friendly_name = "data/tmp/monsters_0.d2p"

		wg.Add(1)
		go func() {
			defer wg.Done()
			DownloadHashImageFileInJson(files, images_1, "data", "d2p")
		}()

		// item images 2
		var images_2 HashFile
		images_2.filename = "content/gfx/monsters/monsters0_1.d2p"
		images_2.friendly_name = "data/tmp/monsters_1.d2p"

		wg.Add(1)
		go func() {
			defer wg.Done()
			DownloadHashImageFileInJson(files, images_2, "data", "d2p")
		}()

		// item images 3
		var images_3 HashFile
		images_3.filename = "content/gfx/monsters/monsters0_2.d2p"
		images_3.friendly_name = "data/tmp/monsters_2.d2p"

		wg.Add(1)
		go func() {
			defer wg.Done()
			DownloadHashImageFileInJson(files, images_3, "data", "d2p")
		}()
	*/

	wg.Wait()
	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}

	inPath := fmt.Sprintf("%s/data/tmp", path)
	outPath := fmt.Sprintf("%s/data/img/item", path)
	absConvertCmd := fmt.Sprintf("%s/PyDofus/%s_unpack.py", path, "d2p")
	exec.Command("/usr/local/bin/python3", absConvertCmd, inPath, outPath).Run()

	log.Println("... images complete")
}

func DownloadItems(hashJson map[string]interface{}) {
	main := hashJson["main"].(map[string]interface{})
	files := main["files"].(map[string]interface{})

	log.Println("loading items...")

	var wg sync.WaitGroup

	var bonusCriterions HashFile
	bonusCriterions.filename = "data/common/BonusesCriterions.d2o"
	bonusCriterions.friendlyName = "data/tmp/bonus_criterions.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, bonusCriterions, "data", "d2o")
	}()

	var evolEffects HashFile
	evolEffects.filename = "data/common/EvolutiveEffects.d2o"
	evolEffects.friendlyName = "data/tmp/evol_effects.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, evolEffects, "data", "d2o")
	}()

	var createBoneOverrides HashFile
	createBoneOverrides.filename = "data/common/CreatureBonesOverrides.d2o"
	createBoneOverrides.friendlyName = "data/tmp/create_bone_overrides.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, createBoneOverrides, "data", "d2o")
	}()

	var creatureBoneTypes HashFile
	creatureBoneTypes.filename = "data/common/CreatureBonesTypes.d2o"
	creatureBoneTypes.friendlyName = "data/tmp/creature_bone_types.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, creatureBoneTypes, "data", "d2o")
	}()

	var charsCategories HashFile
	charsCategories.filename = "data/common/CharacteristicCategories.d2o"
	charsCategories.friendlyName = "data/tmp/chars_categories.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, charsCategories, "data", "d2o")
	}()

	var serverGameTypes HashFile
	serverGameTypes.filename = "data/common/ServerGameTypes.d2o"
	serverGameTypes.friendlyName = "data/tmp/server_game_types.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, serverGameTypes, "data", "d2o")
	}()

	var npcs HashFile
	npcs.filename = "data/common/Npcs.d2o"
	npcs.friendlyName = "data/tmp/npcs.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, npcs, "data", "d2o")
	}()

	var mountFamily HashFile
	mountFamily.filename = "data/common/MountFamily.d2o"
	mountFamily.friendlyName = "data/tmp/mount_family.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, mountFamily, "data", "d2o")
	}()

	var areas HashFile
	areas.filename = "data/common/Areas.d2o"
	areas.friendlyName = "data/tmp/areas.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, areas, "data", "d2o")
	}()

	var companions HashFile
	companions.filename = "data/common/Companions.d2o"
	companions.friendlyName = "data/tmp/companions.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, companions, "data", "d2o")
	}()

	var companionSpells HashFile
	companionSpells.filename = "data/common/CompanionSpells.d2o"
	companionSpells.friendlyName = "data/tmp/companion_spells.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, companionSpells, "data", "d2o")
	}()

	var companionChars HashFile
	companionChars.filename = "data/common/CompanionCharacteristics.d2o"
	companionChars.friendlyName = "data/tmp/companion_chars.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, companionChars, "data", "d2o")
	}()

	var monsters HashFile
	monsters.filename = "data/common/Monsters.d2o"
	monsters.friendlyName = "data/tmp/monsters.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, monsters, "data", "d2o")
	}()

	var almanax HashFile
	almanax.filename = "data/common/AlmanaxCalendars.d2o"
	almanax.friendlyName = "data/tmp/almanax.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, almanax, "data", "d2o")
	}()

	var idols HashFile
	idols.filename = "data/common/Idols.d2o"
	idols.friendlyName = "data/tmp/idols.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, idols, "data", "d2o")
	}()

	var mounts HashFile
	mounts.filename = "data/common/Mounts.d2o"
	mounts.friendlyName = "data/tmp/mounts.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, mounts, "data", "d2o")
	}()

	var breeds HashFile
	breeds.filename = "data/common/Breeds.d2o"
	breeds.friendlyName = "data/tmp/breeds.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, breeds, "data", "d2o")
	}()

	var spellTypes HashFile
	spellTypes.filename = "data/common/SpellTypes.d2o"
	spellTypes.friendlyName = "data/tmp/spell_types.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, spellTypes, "data", "d2o")
	}()

	var spells HashFile
	spells.filename = "data/common/Spells.d2o"
	spells.friendlyName = "data/tmp/spells.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, spells, "data", "d2o")
	}()

	var recipes HashFile
	recipes.filename = "data/common/Recipes.d2o"
	recipes.friendlyName = "data/tmp/recipes.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, recipes, "data", "d2o")
	}()

	var bonuses HashFile
	bonuses.filename = "data/common/Bonuses.d2o"
	bonuses.friendlyName = "data/tmp/bonuses.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, bonuses, "data", "d2o")
	}()

	var effects HashFile
	effects.filename = "data/common/Effects.d2o"
	effects.friendlyName = "data/tmp/effects.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, effects, "data", "d2o")
	}()

	// item types
	var itemTypes HashFile
	itemTypes.filename = "data/common/ItemTypes.d2o"
	itemTypes.friendlyName = "data/tmp/item_types.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, itemTypes, "data", "d2o")
	}()

	var itemSets HashFile
	itemSets.filename = "data/common/ItemSets.d2o"
	itemSets.friendlyName = "data/tmp/item_sets.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, itemSets, "data", "d2o")
	}()

	var items HashFile
	items.filename = "data/common/Items.d2o"
	items.friendlyName = "data/tmp/items.d2o"

	wg.Add(1)
	go func() {
		defer wg.Done()
		DownloadHashFileInJson(files, items, "data", "d2o")
	}()

	wg.Wait()
	log.Println("... items complete")

}

func DownloadLanguages(hashJson map[string]interface{}) {
	log.Println("loading languages...")
	var wg sync.WaitGroup
	wg.Add(6)

	var deLangFile HashFile
	deLangFile.filename = "data/i18n/i18n_de.d2i"
	deLangFile.friendlyName = "data/tmp/lang_de.d2i"

	go func() {
		defer wg.Done()

		langDe := hashJson["lang_de"].(map[string]interface{})
		deFiles := langDe["files"].(map[string]interface{})
		deD2i := deFiles[deLangFile.filename].(map[string]interface{})
		deLangFile.hash = deD2i["hash"].(string)
		DownloadHashFile(deLangFile)

		Unpack(deLangFile.friendlyName, "data/languages", "d2i")
		err := os.Remove(deLangFile.friendlyName)
		if err != nil {
			log.Println(err)
		}
	}()

	var enLangFile HashFile
	enLangFile.filename = "data/i18n/i18n_en.d2i"
	enLangFile.friendlyName = "data/tmp/lang_en.d2i"

	go func() {
		defer wg.Done()

		langEn := hashJson["lang_en"].(map[string]interface{})
		enFiles := langEn["files"].(map[string]interface{})
		enD2i := enFiles[enLangFile.filename].(map[string]interface{})
		enLangFile.hash = enD2i["hash"].(string)
		DownloadHashFile(enLangFile)

		Unpack(enLangFile.friendlyName, "data/languages", "d2i")
		err := os.Remove(enLangFile.friendlyName)
		if err != nil {
			log.Println(err)
		}
	}()

	var esLangFile HashFile
	esLangFile.filename = "data/i18n/i18n_es.d2i"
	esLangFile.friendlyName = "data/tmp/lang_es.d2i"

	go func() {
		defer wg.Done()

		langEs := hashJson["lang_es"].(map[string]interface{})
		esFiles := langEs["files"].(map[string]interface{})
		esD2i := esFiles[esLangFile.filename].(map[string]interface{})
		esLangFile.hash = esD2i["hash"].(string)
		DownloadHashFile(esLangFile)

		Unpack(esLangFile.friendlyName, "data/languages", "d2i")
		err := os.Remove(esLangFile.friendlyName)
		if err != nil {
			log.Println(err)
		}
	}()

	var frLangFile HashFile
	frLangFile.filename = "data/i18n/i18n_fr.d2i"
	frLangFile.friendlyName = "data/tmp/lang_fr.d2i"

	go func() {
		defer wg.Done()

		langFr := hashJson["lang_fr"].(map[string]interface{})
		frFiles := langFr["files"].(map[string]interface{})
		frD2i := frFiles[frLangFile.filename].(map[string]interface{})
		frLangFile.hash = frD2i["hash"].(string)
		DownloadHashFile(frLangFile)

		Unpack(frLangFile.friendlyName, "data/languages", "d2i")
		err := os.Remove(frLangFile.friendlyName)
		if err != nil {
			log.Println(err)
		}
	}()

	var itLangFile HashFile
	itLangFile.filename = "data/i18n/i18n_it.d2i"
	itLangFile.friendlyName = "data/tmp/lang_it.d2i"

	go func() {
		defer wg.Done()

		langIt := hashJson["lang_it"].(map[string]interface{})
		itFiles := langIt["files"].(map[string]interface{})
		itD2i := itFiles[itLangFile.filename].(map[string]interface{})
		itLangFile.hash = itD2i["hash"].(string)
		DownloadHashFile(itLangFile)

		Unpack(itLangFile.friendlyName, "data/languages", "d2i")
		err := os.Remove(itLangFile.friendlyName)
		if err != nil {
			log.Println(err)
		}
	}()

	var ptLangFile HashFile
	ptLangFile.filename = "data/i18n/i18n_pt.d2i"
	ptLangFile.friendlyName = "data/tmp/lang_pt.d2i"

	go func() {
		defer wg.Done()

		langPt := hashJson["lang_pt"].(map[string]interface{})
		ptFiles := langPt["files"].(map[string]interface{})
		ptD2i := ptFiles[ptLangFile.filename].(map[string]interface{})
		ptLangFile.hash = ptD2i["hash"].(string)
		DownloadHashFile(ptLangFile)

		Unpack(ptLangFile.friendlyName, "data/languages", "d2i")
		err := os.Remove(ptLangFile.friendlyName)
		if err != nil {
			log.Println(err)
		}
	}()

	wg.Wait()
	log.Println("... languages complete")
}
