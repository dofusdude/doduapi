package update

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/dofusdude/ankabuffer"
	"github.com/dofusdude/api/utils"
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
	go func(manifest *ankabuffer.Manifest) {
		defer waitGrp.Done()
		if err := DownloadLanguages(manifest); err != nil {
			log.Println(err)
		}
	}(&hashJson)

	waitGrp.Add(1)
	go func(manifest *ankabuffer.Manifest) {
		defer waitGrp.Done()
		if err := DownloadImagesLauncher(manifest); err != nil {
			log.Println(err)
		}
	}(&hashJson)

	waitGrp.Add(1)
	go func(manifest *ankabuffer.Manifest) {
		defer waitGrp.Done()
		if err := DownloadItems(manifest); err != nil {
			log.Println(err)
		}
	}(&hashJson)

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

func CleanUp() {
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
		absPath := fmt.Sprintf("%s/%s", utils.DockerMountDataPath, file)
		_ = os.Remove(absPath)
	}

	//os.RemoveAll("data/img") // keep old images, override with new ones, else they are unavailable while updating
	//os.Mkdir("data/img", 0755)

	meiliClient := utils.CreateMeiliClient()
	if meiliClient == nil {
		log.Fatal("meili could not be reached")
	}

	red_blue_versions := []string{"red", "blue"}
	for _, version := range red_blue_versions {
		for _, lang := range utils.Languages {
			taskStuffDelete, err := meiliClient.DeleteIndex(fmt.Sprintf("%s-all_stuff-%s", version, lang))
			if err != nil {
				log.Println(err)
			}
			taskItemsDelete, err := meiliClient.DeleteIndex(fmt.Sprintf("%s-all_items-%s", version, lang))
			if err != nil {
				log.Println(err)
			}
			taskSetsDelete, err := meiliClient.DeleteIndex(fmt.Sprintf("%s-sets-%s", version, lang))
			if err != nil {
				log.Println(err)
			}
			taskMountsDelete, err := meiliClient.DeleteIndex(fmt.Sprintf("%s-mounts-%s", version, lang))
			if err != nil {
				log.Println(err)
			}

			_, _ = meiliClient.WaitForTask(taskStuffDelete.TaskUID)
			_, _ = meiliClient.WaitForTask(taskItemsDelete.TaskUID)
			_, _ = meiliClient.WaitForTask(taskSetsDelete.TaskUID)
			_, _ = meiliClient.WaitForTask(taskMountsDelete.TaskUID)
		}
	}
}

func Unpack(filepath string, destDirRel string, fileType string) {
	var err error

	if fileType == "png" || fileType == "jpg" || fileType == "jpeg" {
		return // no need to unpack images files
	}

	absConvertCmd := fmt.Sprintf("%s/PyDofus/%s_unpack.py", utils.DockerMountDataPath, fileType)
	absFilePath := fmt.Sprintf("%s/%s", utils.DockerMountDataPath, filepath)
	absOutPath := strings.Replace(absFilePath, fileType, "json", 1)
	filenameParts := strings.Split(filepath, "/")
	filename := filenameParts[len(filenameParts)-1]
	outFile := strings.Replace(filename, fileType, "json", 1)
	finalOutPath := fmt.Sprintf("%s/%s/%s", utils.DockerMountDataPath, destDirRel, outFile)

	err = exec.Command(utils.PythonPath, absConvertCmd, absFilePath).Run()
	if err != nil {
		log.Fatalf("Unpacking failed: %s %s %s with Error %v", utils.PythonPath, absConvertCmd, absFilePath, err)
	}

	err = os.Rename(absOutPath, finalOutPath)
	if err != nil {
		log.Fatal(err)
	}
}

func DownloadUnpackFiles(manifest *ankabuffer.Manifest, fragment string, toDownload []HashFile, relDir string, unpack bool) error {
	var filesToDownload []ankabuffer.File
	for i, file := range toDownload {
		filesToDownload = append(filesToDownload, manifest.Fragments[fragment].Files[file.Filename])
		toDownload[i].Hash = manifest.Fragments[fragment].Files[file.Filename].Hash
	}

	bundles := ankabuffer.GetNeededBundles(filesToDownload)

	if len(bundles) == 0 && len(filesToDownload) > 0 {
		for _, file := range filesToDownload {
			log.Println("Missing bundle for", file.Name)
		}
	}

	if len(bundles) == 0 {
		return nil
	}

	bundlesMap := ankabuffer.GetBundleHashMap(manifest)

	type DownloadedBundle struct {
		BundleHash string
		Data       []byte
	}

	//bundleData := make(chan DownloadedBundle, len(bundles))
	bundlesBuffer := make(map[string]DownloadedBundle)

	for _, bundle := range bundles {
		//go func(bundleHash string, data chan DownloadedBundle) {
		bundleData, err := DownloadBundle(bundle)
		if err != nil {
			return fmt.Errorf("could not download bundle %s: %s", bundle, err)
		}
		res := DownloadedBundle{BundleHash: bundle, Data: bundleData}
		bundlesBuffer[bundle] = res
		//}(bundle, bundleData)
	}
	/*
		for i := 0; i < len(bundles); i++ {
			bundle := <-bundleData
			log.Println("Downloaded bundle", i, "of", len(bundles))
			bundlesBuffer[bundle.BundleHash] = <-bundleData
		}*/

	var wg sync.WaitGroup
	for i, file := range filesToDownload {
		if file.Name == "" {
			continue
		}
		wg.Add(1)
		go func(file ankabuffer.File, bundlesBuffer map[string]DownloadedBundle, relDir string, i int) {
			defer wg.Done()
			var fileData []byte

			if file.Chunks == nil || len(file.Chunks) == 0 { // file is not chunked
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
									err := fmt.Errorf("bundle data is too small. Bundle offset/size: %d/%d, BundleData length: %d, BundleHash: %s, BundleChunkHash: %s", bundleChunk.Offset, bundleChunk.Size, len(bundle.Data), bundle.BundleHash, bundleChunk.Hash)
									log.Fatal(err)
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
				//if len(chunksData) > 1 {
				//	log.Println("Chunks data", chunksData[0].Offset, chunksData[len(chunksData)-1].Offset)
				//}
				for _, chunk := range chunksData {
					fileData = append(fileData, chunk.Data...)
				}
			}

			if fileData == nil || len(fileData) == 0 {
				err := fmt.Errorf("file data is empty %s", file.Hash)
				log.Fatal(err)
			}

			fp, err := os.Create(toDownload[i].FriendlyName)
			if err != nil {
				log.Fatal(err)
			}
			defer fp.Close()
			_, err = fp.Write(fileData)
			if err != nil {
				log.Fatal(err)
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
	return nil
}
