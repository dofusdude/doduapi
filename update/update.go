package update

import (
	"fmt"
	"github.com/dofusdude/ankabuffer"
	"github.com/dofusdude/api/utils"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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

	hashJson, err := utils.GetReleaseManifest(version)
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

func DownloadBundle(bundleHash string) ([]byte, error) {
	url := fmt.Sprintf("https://cytrus.cdn.ankama.com/dofus/bundles/%s/%s", bundleHash[0:2], bundleHash)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bundle %s status %d", bundleHash, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
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
		"data/monster_races.json",

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

	err = exec.Command(utils.PythonPath, absConvertCmd, absFilePath).Run()
	if err != nil {
		log.Fatal(err)
	}

	err = os.Rename(absOutPath, finalOutPath)
	if err != nil {
		log.Fatal(err)
	}
}

func DownloadUnpackFiles(manifest *ankabuffer.Manifest, fragment string, toDownload []HashFile, relDir string, unpack bool) {
	var filesToDownload []ankabuffer.File
	for i, file := range toDownload {
		filesToDownload = append(filesToDownload, manifest.Fragments[fragment].Files[file.Filename])
		toDownload[i].Hash = manifest.Fragments[fragment].Files[file.Filename].Hash
	}

	bundles := ankabuffer.GetNeededBundles(filesToDownload)

	if len(bundles) == 0 {
		log.Println("No files to download")
		return
	}

	bundlesMap := ankabuffer.GetBundleHashMap(manifest)

	type DownloadedBundle struct {
		BundleHash string
		Data       []byte
	}

	bundleData := make(chan DownloadedBundle, len(bundles))

	for _, bundle := range bundles {
		go func(bundleHash string, data chan DownloadedBundle) {
			bundleData, err := DownloadBundle(bundleHash)
			if err != nil {
				log.Fatal(err)
				return
			}
			res := DownloadedBundle{BundleHash: bundleHash, Data: bundleData}
			data <- res
		}(bundle, bundleData)
	}

	bundlesBuffer := make(map[string]DownloadedBundle)

	for i := 0; i < len(bundles); i++ {
		bundle := <-bundleData
		bundlesBuffer[bundle.BundleHash] = <-bundleData
	}

	var wg sync.WaitGroup
	for i, file := range filesToDownload {
		wg.Add(1)
		go func(file ankabuffer.File, bundlesBuffer map[string]DownloadedBundle, relDir string, i int) {
			defer wg.Done()
			var fileData []byte

			if file.Chunks == nil { // file is not chunked
				for _, bundle := range bundlesBuffer {
					for _, chunk := range bundlesMap[bundle.BundleHash].Chunks {
						if chunk.Hash == file.Hash {
							fileData = bundle.Data[chunk.Offset : chunk.Offset+chunk.Size]
							break
						}
					}
					if fileData != nil {
						break
					}
				}
			} else { // file is chunked
				type ChunkData struct {
					Data   []byte
					Offset int64
					Size   int64
				}
				var chunksData []ChunkData
				for _, chunk := range file.Chunks { // all chunks of the file
					for _, bundle := range bundlesBuffer { // search in downloaded bundles for the chunk
						foundChunk := false
						for _, bundleChunk := range bundlesMap[bundle.BundleHash].Chunks { // each chunk of the searched bundle could be a chunk of the file
							if bundleChunk.Hash == chunk.Hash {
								foundChunk = true
								if len(bundle.Data) < int(bundleChunk.Offset+bundleChunk.Size) {
									log.Fatal("bundle data is too small", bundleChunk.Offset, bundleChunk.Size, len(bundle.Data), bundle.BundleHash, bundleChunk.Hash)
									//bundle data is too small 24842 21742063 243 6630b4be6b4bb75b5094153ad11e1db3047dc6 c59256166a885120574637785df120d2273a95c4
									//  bundle data is too small 67999 0 243 2d202014979e1b18f7eb328c3c90c025fc37cee 9e1930b6481e6064b059a1e821bb793ee1d2b9d2
									return
								}

								chunksData = append(chunksData, ChunkData{Data: bundle.Data[bundleChunk.Offset : bundleChunk.Offset+bundleChunk.Size], Offset: chunk.Offset, Size: chunk.Size})
							}
						}
						if foundChunk {
							break
						}
					}
				}
				sort.Slice(chunksData, func(i, j int) bool {
					return chunksData[i].Offset < chunksData[j].Offset
				})
				for _, chunk := range chunksData {
					fileData = append(fileData, chunk.Data...)
				}
			}

			if err := os.WriteFile(fmt.Sprintf("%s/%s", relDir, toDownload[i].Filename), fileData, 0644); err != nil {
				log.Println(err)
				return
			}

			if unpack {
				Unpack(toDownload[i].FriendlyName, relDir, filepath.Ext(toDownload[i].FriendlyName)[1:])
				err := os.Remove(toDownload[i].FriendlyName)
				if err != nil {
					log.Println(err)
				}
			}
		}(file, bundlesBuffer, relDir, i)
	}

	wg.Wait()
}
