package update

import (
	"fmt"
	"github.com/dofusdude/api/utils"
	"io"
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
	Hash         string
	Filename     string
	FriendlyName string
}

func DownloadUpdatesIfAvailable(force bool) error {
	currentVersion := utils.GetCurrentVersion()
	version := utils.GetLatestLauncherVersion()

	if !force && currentVersion == version {
		return fmt.Errorf("no updates available")
	}
	CleanUp()
	utils.CreateDataDirectoryStructure()

	hashJson, err := utils.GetDofusFileHashesJson(version)
	if err != nil {
		return err
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

	err = utils.NewVersion(version)
	if err != nil {
		return err
	}

	return nil
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

func DownloadHashFile(file HashFile) error {
	url := fmt.Sprintf("https://launcher.cdn.ankama.com/dofus/hashes/%s/%s", file.Hash[:2], file.Hash)
	return DownloadFile(file.FriendlyName, url)
}

func CleanUp() {
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
		"data/mounts.json",
		"data/bonuses.json",
		"data/breeds.json",
		"data/creature_bone_types.json",

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
	if meiliClient == nil {
		log.Fatal("meili could not be reached")
	}

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

func Unpack(filepath string, destDirRel string, fileType string) {
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
	finalOutPath := fmt.Sprintf("%s/%s/%s", path, destDirRel, outFile)

	err = exec.Command("/usr/local/bin/python3", absConvertCmd, absFilePath).Run()
	if err != nil {
		log.Fatal(err)
	}

	err = os.Rename(absOutPath, finalOutPath)
	if err != nil {
		log.Fatal(err)
	}
}

func DownloadHashImageFileInJson(files map[string]interface{}, hashFile HashFile) error {
	file := files[hashFile.Filename].(map[string]interface{})
	hashFile.Hash = file["hash"].(string)
	err := DownloadHashFile(hashFile)
	return err
}

func DownloadHashFileInJson(files map[string]interface{}, hashFile HashFile, destDirRel string, fileType string) error {
	err := DownloadHashImageFileInJson(files, hashFile)
	if err != nil {
		return err
	}
	Unpack(hashFile.FriendlyName, destDirRel, fileType)

	return nil
}
