package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/dofusdude/ankabuffer"
	"github.com/emirpasic/gods/maps/treebidimap"
	gutils "github.com/emirpasic/gods/utils"
	_ "github.com/joho/godotenv/autoload"
	"github.com/spf13/viper"
)

var (
	Languages           = []string{"de", "en", "es", "fr", "pt"}
	ItemImgResolutions  = []string{"64", "128"}
	MountImgResolutions = []string{"64", "256"}
	ApiHostName         string
	ApiPort             string
	ApiScheme           string
	DockerMountDataPath string
	FileHashes          ankabuffer.Manifest
	MeiliHost           string
	MeiliKey            string
	PrometheusEnabled   bool
	PublishFileServer   bool
	PersistedElements   PersistentStringKeysMap
	PersistedTypes      PersistentStringKeysMap
	IsBeta              bool
	LastUpdate          time.Time
	ElementsUrl         string
	TypesUrl            string
	ReleaseUrl          string
	UpdateHookToken     string
	DofusVersion        string
	CurrentVersion      GameVersion
	ApiVersion          string
)

var currentWd string

type VersionT struct {
	Search bool
	MemDb  bool
}

type GameVersion struct {
	Version     string    `json:"version"`
	Release     string    `json:"release"`
	UpdateStamp time.Time `json:"update_stamp"`
}

func Concat[T any](first []T, second []T) []T {
	n := len(first)
	return append(first[:n:n], second...)
}

func SetJsonHeader(w *http.ResponseWriter) {
	(*w).Header().Set("Content-Type", "application/json")
}

func WriteCacheHeader(w *http.ResponseWriter) {
	SetJsonHeader(w)
	(*w).Header().Set("Cache-Control", "max-age:300, public")
	(*w).Header().Set("Last-Modified", LastUpdate.Format(http.TimeFormat))
	(*w).Header().Set("Expires", time.Now().Add(time.Minute*5).Format(http.TimeFormat))
}

// from Armatorix https://codereview.stackexchange.com/questions/272457/decompress-tar-gz-file-in-go
func ExtractTarGz(baseDir string, gzipStream io.Reader) error {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(uncompressedStream)
	var header *tar.Header
	for header, err = tarReader.Next(); err == nil; header, err = tarReader.Next() {
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(filepath.Join(baseDir, header.Name), 0755); err != nil {
				return fmt.Errorf("ExtractTarGz: Mkdir() failed: %w", err)
			}
		case tar.TypeReg:
			fileDir := filepath.Dir(header.Name)
			if err := os.MkdirAll(filepath.Join(baseDir, fileDir), 0755); err != nil {
				return fmt.Errorf("ExtractTarGz: Mkdir() failed: %w", err)
			}
			outFile, err := os.Create(filepath.Join(baseDir, header.Name))
			if err != nil {
				return fmt.Errorf("ExtractTarGz: Create() failed: %w", err)
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("ExtractTarGz: Copy() failed: %w", err)
			}
			if err := outFile.Close(); err != nil {
				return fmt.Errorf("ExtractTarGz: Close() failed: %w", err)
			}
		default:
			return fmt.Errorf("ExtractTarGz: uknown type: %b in %s", header.Typeflag, header.Name)
		}
	}
	if err != io.EOF {
		return fmt.Errorf("ExtractTarGz: Next() failed: %w", err)
	}
	return nil
}

func DownloadExtract(filename string) error {
	absUrl := fmt.Sprintf("%s/%s.tar.gz", ReleaseUrl, filename)
	response, err := http.Get(absUrl)
	if err != nil {
		return err
	}
	err = ExtractTarGz(DockerMountDataPath, response.Body)
	if err != nil {
		return err
	}

	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	return err
}

func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		switch entry.Type() {
		case os.ModeDir:
			if _, err := os.Stat(dstPath); os.IsNotExist(err) {
				if err := os.Mkdir(dstPath, 0755); err != nil {
					return err
				}
			}
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		default:
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func DownloadImages() error {
	var err error

	// -- items --
	err = DownloadExtract(fmt.Sprintf("items_images_64"))
	if err != nil {
		return fmt.Errorf("could not download items_images: %v", err)
	}

	err = DownloadExtract(fmt.Sprintf("items_images_128"))
	if err != nil {
		return fmt.Errorf("could not download items_images: %v", err)
	}

	oldPath1x := filepath.Join(DockerMountDataPath, "data", "img", "item", "1x")
	oldPath2x := filepath.Join(DockerMountDataPath, "data", "img", "item", "2x")
	newPath := filepath.Join(DockerMountDataPath, "data", "img", "item")

	err = copyDir(oldPath1x, newPath)
	if err != nil {
		return fmt.Errorf("could not copy images to path: %v", err)
	}

	err = copyDir(oldPath2x, newPath)
	if err != nil {
		return fmt.Errorf("could not copy images to path: %v", err)
	}

	err = os.RemoveAll(oldPath1x)
	if err != nil {
		return fmt.Errorf("could not remove old images dir: %v", err)
	}

	err = os.RemoveAll(oldPath2x)
	if err != nil {
		return fmt.Errorf("could not remove old images dir: %v", err)
	}

	// -- mounts --
	err = DownloadExtract(fmt.Sprintf("mounts_images_64"))
	if err != nil {
		return fmt.Errorf("could not download items_images: %v", err)
	}

	err = DownloadExtract(fmt.Sprintf("mounts_images_256"))
	if err != nil {
		return fmt.Errorf("could not download items_images: %v", err)
	}

	oldPathSmall := filepath.Join(DockerMountDataPath, "data", "img", "mount", "small")
	oldPathBig := filepath.Join(DockerMountDataPath, "data", "img", "mount", "big")
	newPathMounts := filepath.Join(DockerMountDataPath, "data", "img", "mount")

	err = copyDir(oldPathSmall, newPathMounts)
	if err != nil {
		return fmt.Errorf("could not copy images to path: %v", err)
	}

	err = copyDir(oldPathBig, newPathMounts)
	if err != nil {
		return fmt.Errorf("could not copy images to path: %v", err)
	}

	err = os.RemoveAll(oldPathSmall)
	if err != nil {
		return fmt.Errorf("could not remove old images dir: %v", err)
	}

	err = os.RemoveAll(oldPathBig)
	if err != nil {
		return fmt.Errorf("could not remove old images dir: %v", err)
	}

	return nil
}

func ReadEnvs() {
	viper.SetDefault("API_SCHEME", "http")
	viper.SetDefault("API_HOSTNAME", "localhost")
	viper.SetDefault("API_PORT", "3000")
	viper.SetDefault("MEILI_PORT", "7700")
	viper.SetDefault("MEILI_MASTER_KEY", "masterKey")
	viper.SetDefault("MEILI_PROTOCOL", "http")
	viper.SetDefault("MEILI_HOST", "127.0.0.1")
	viper.SetDefault("PROMETHEUS", "false")
	viper.SetDefault("FILESERVER", "true")
	viper.SetDefault("IS_BETA", "false")
	viper.SetDefault("UPDATE_HOOK_TOKEN", "")
	viper.SetDefault("DOFUS_VERSION", "")
	viper.SetDefault("LOG_LEVEL", "warn")

	var err error
	currentWd, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	viper.SetDefault("DIR", currentWd)

	viper.AutomaticEnv()

	IsBeta = viper.GetBool("IS_BETA")
	var betaStr string
	if IsBeta {
		betaStr = "beta"
	} else {
		betaStr = "main"
	}

	dofusVersion := viper.GetString("DOFUS_VERSION")
	if dofusVersion == "" {
		releaseApiResponse, err := http.Get(fmt.Sprintf("https://api.github.com/repos/dofusdude/dofus3-%s/releases/latest", betaStr))
		if err != nil {
			log.Fatal(err)
		}

		releaseApiResponseBody, err := io.ReadAll(releaseApiResponse.Body)
		if err != nil {
			log.Fatal(err)
		}

		var v map[string]interface{}
		err = json.Unmarshal(releaseApiResponseBody, &v)
		if err != nil {
			log.Fatal(err)
		}

		DofusVersion = v["name"].(string)
	} else {
		DofusVersion = dofusVersion
	}
	parsedLevel, err := log.ParseLevel(viper.GetString("LOG_LEVEL"))
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(parsedLevel)

	dofus3Prefix := ""
	if strings.HasPrefix(DofusVersion, "3") {
		log.Print("Dofus 3 detected") // Todo: remove
		dofus3Prefix = ".dofus3"
	}

	ElementsUrl = fmt.Sprintf("https://raw.githubusercontent.com/dofusdude/doduda/main/persistent/elements%s.%s.json", dofus3Prefix, betaStr)
	TypesUrl = fmt.Sprintf("https://raw.githubusercontent.com/dofusdude/doduda/main/persistent/item_types%s.%s.json", dofus3Prefix, betaStr)
	ReleaseUrl = fmt.Sprintf("https://github.com/dofusdude/dofus3-%s/releases/download/%s", betaStr, DofusVersion)

	ApiScheme = viper.GetString("API_SCHEME")
	ApiHostName = viper.GetString("API_HOSTNAME")
	ApiPort = viper.GetString("API_PORT")
	MeiliKey = viper.GetString("MEILI_MASTER_KEY")
	MeiliHost = fmt.Sprintf("%s://%s:%s", viper.GetString("MEILI_PROTOCOL"), viper.GetString("MEILI_HOST"), viper.GetString("MEILI_PORT"))
	PrometheusEnabled = viper.GetBool("PROMETHEUS")
	PublishFileServer = viper.GetBool("FILESERVER")
	UpdateHookToken = viper.GetString("UPDATE_HOOK_TOKEN")
	DockerMountDataPath = viper.GetString("DIR")
}

func ImageUrls(iconId int, apiType string, resolutions []string) []string {
	betaImage := ""
	if IsBeta {
		betaImage = "beta"
	}
	baseUrl := fmt.Sprintf("%s://%s/dofus3%s/v%d/img/%s", ApiScheme, ApiHostName, betaImage, DoduapiMajor, apiType)
	var urls []string

	for _, resolution := range resolutions {
		resolutionUrl := fmt.Sprintf("%s/%d-%s.png", baseUrl, iconId, resolution)
		urls = append(urls, resolutionUrl)
	}
	return urls
}

type PersistentStringKeysMap struct {
	Entries *treebidimap.Map `json:"entries"`
	NextId  int              `json:"next_id"`
}

func LoadPersistedElements() error {
	elementsResponse, err := http.Get(ElementsUrl)
	if err != nil {
		return err
	}

	elementsBody, err := io.ReadAll(elementsResponse.Body)
	if err != nil {
		return err
	}

	var elements []string
	err = json.Unmarshal(elementsBody, &elements)
	if err != nil {
		return err
	}

	PersistedElements = PersistentStringKeysMap{
		Entries: treebidimap.NewWith(gutils.IntComparator, gutils.StringComparator),
		NextId:  0,
	}

	for _, entry := range elements {
		PersistedElements.Entries.Put(PersistedElements.NextId, entry)
		PersistedElements.NextId++
	}

	typesResponse, err := http.Get(TypesUrl)
	if err != nil {
		return err
	}

	typesBody, err := io.ReadAll(typesResponse.Body)
	if err != nil {
		return err
	}

	var types []string
	err = json.Unmarshal(typesBody, &types)
	if err != nil {
		return err
	}

	PersistedTypes = PersistentStringKeysMap{
		Entries: treebidimap.NewWith(gutils.IntComparator, gutils.StringComparator),
		NextId:  0,
	}

	for _, entry := range types {
		PersistedTypes.Entries.Put(PersistedTypes.NextId, entry)
		PersistedTypes.NextId++
	}

	return nil
}

func CurrentRedBlueVersionStr(redBlueValue bool) string {
	if redBlueValue {
		return "red"
	}
	return "blue"
}

func NextRedBlueVersionStr(redBlueValue bool) string {
	if redBlueValue {
		return "blue"
	}
	return "red"
}

type Pagination struct {
	PageNumber int
	PageSize   int
}

func PageninationWithState(paginationStr string) Pagination {
	vals := strings.Split(paginationStr, ",")
	num, _ := strconv.Atoi(vals[0])
	size, _ := strconv.Atoi(vals[1])
	return Pagination{
		PageNumber: num,
		PageSize:   size,
	}
}

type PaginationLinks struct {
	First *string `json:"first"`
	Prev  *string `json:"prev"`
	Next  *string `json:"next"`
	Last  *string `json:"last"`
}

func (p *Pagination) ValidatePagination(listSize int) int {
	if p.PageSize == -1 {
		p.PageSize = listSize
	}
	if p.PageSize > listSize || p.PageSize < -1 || p.PageSize == 0 {
		return -1
	}
	if (p.PageSize * p.PageNumber) >= listSize+p.PageSize {
		return 1
	}
	return 0
}

func (p *Pagination) BuildLinks(mainUrl url.URL, listSize int) (PaginationLinks, bool) {
	firstPage := 1
	var lastPage int

	lastPageSize := listSize % p.PageSize
	if lastPageSize == 0 {
		lastPage = listSize / p.PageSize
	} else {
		lastPage = (listSize / p.PageSize) + 1
	}

	baseUrl, _ := url.JoinPath(fmt.Sprintf("%s://%s", ApiScheme, ApiHostName), mainUrl.Path)

	firstUrlQuery := fmt.Sprintf("page[number]=%d&page[size]=%d", firstPage, p.PageSize)
	prevUrlQuery := fmt.Sprintf("page[number]=%d&page[size]=%d", p.PageNumber-1, p.PageSize)
	nextUrlQuery := fmt.Sprintf("page[number]=%d&page[size]=%d", p.PageNumber+1, p.PageSize)
	lastUrlQuery := fmt.Sprintf("page[number]=%d&page[size]=%d", lastPage, p.PageSize)

	firstUrlStr := url.QueryEscape(firstUrlQuery)
	prevUrlStr := url.QueryEscape(prevUrlQuery)
	nextUrlStr := url.QueryEscape(nextUrlQuery)
	lastUrlStr := url.QueryEscape(lastUrlQuery)

	firstUrl := fmt.Sprintf("%s?%s", baseUrl, firstUrlStr)
	prevUrl := fmt.Sprintf("%s?%s", baseUrl, prevUrlStr)
	nextUrl := fmt.Sprintf("%s?%s", baseUrl, nextUrlStr)
	lastUrl := fmt.Sprintf("%s?%s", baseUrl, lastUrlStr)

	finalFirstUrl := &firstUrl
	finalPrevUrl := &prevUrl
	finalNextUrl := &nextUrl
	finalLastUrl := &lastUrl

	if p.PageNumber == firstPage {
		finalPrevUrl = nil
	}

	if p.PageNumber == lastPage {
		finalNextUrl = nil
	}

	if lastPage == firstPage {
		finalLastUrl = nil
		finalFirstUrl = nil
	}

	return PaginationLinks{
		First: finalFirstUrl,
		Prev:  finalPrevUrl,
		Next:  finalNextUrl,
		Last:  finalLastUrl,
	}, firstUrlStr == lastUrlStr
}

func (p *Pagination) CalculateStartEndIndex(listSize int) (int, int) {
	startIndex := (p.PageNumber * p.PageSize) - p.PageSize
	endIndex := startIndex + p.PageSize

	if endIndex > listSize {
		endIndex = listSize
	}
	return Max(startIndex, 0), Max(endIndex, 0)
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func PartitionSlice[T any](items []T, parts int) (chunks [][]T) {
	var divided [][]T

	chunkSize := (len(items) + parts - 1) / parts

	for i := 0; i < len(items); i += chunkSize {
		end := i + chunkSize

		if end > len(items) {
			end = len(items)
		}

		divided = append(divided, items[i:end])
	}

	return divided
}

// https://stackoverflow.com/questions/13422578/in-go-how-to-get-a-slice-of-values-from-a-map
func Values[M ~map[K]V, K comparable, V any](m M) []V {
	r := make([]V, 0, len(m))
	for _, v := range m {
		r = append(r, v)
	}
	return r
}

func CleanJSON(jsonStr string) string {
	jsonStr = strings.ReplaceAll(jsonStr, "NaN", "null")
	jsonStr = strings.ReplaceAll(jsonStr, "\"null\"", "null")
	jsonStr = strings.ReplaceAll(jsonStr, " ", " ")
	return jsonStr
}

func CategoryIdMapping(id int) string {
	switch id {
	case 0:
		return "equipment"
	case 1:
		return "consumables"
	case 2:
		return "resources"
	case 3:
		return "quest_items"
	case 5:
		return "cosmetics"
	}
	return ""
}

func CategoryIdApiMapping(id int) string {
	switch id {
	case 0:
		return "equipment"
	case 1:
		return "consumables"
	case 2:
		return "resources"
	case 3:
		return "quest"
	case 5:
		return "cosmetics"
	}
	return ""
}
