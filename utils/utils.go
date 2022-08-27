package utils

import (
	"encoding/json"
	"fmt"
	"github.com/meilisearch/meilisearch-go"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

var Languages = []string{"de", "en", "es", "fr", "it", "pt"}
var ImgResolutions = []string{"200", "400", "800"}
var ImgWithResExists *Set

var ApiHostName string
var ApiPort string
var ApiScheme string
var DockerMountDataPath string
var FileHashes map[string]interface{}

var currentWd string

func GetFileHashesJson(version string) (map[string]interface{}, error) {
	gameHashesUrl := fmt.Sprintf("https://launcher.cdn.ankama.com/dofus/releases/main/windows/%s.json", version)
	hashResponse, err := http.Get(gameHashesUrl)
	if err != nil {
		log.Println(err)
		return map[string]interface{}{}, err
	}

	hashBody, err := io.ReadAll(hashResponse.Body)
	if err != nil {
		log.Println(err)
		return map[string]interface{}{}, err
	}

	var hashJson map[string]interface{}
	err = json.Unmarshal(hashBody, &hashJson)
	if err != nil {
		log.Println(err)
		return map[string]interface{}{}, err
	}

	FileHashes = hashJson
	return hashJson, nil
}

type VersionT struct {
	Search bool
	MemDb  bool
}

func ReadEnvs() (string, string) {
	apiScheme, ok := os.LookupEnv("API_SCHEME")
	if !ok {
		apiScheme = "http"
	}
	ApiScheme = apiScheme

	apiHostName, ok := os.LookupEnv("API_HOSTNAME")
	if !ok {
		apiHostName = "localhost"
	}
	ApiHostName = apiHostName

	apiPort, ok := os.LookupEnv("API_PORT")
	if !ok {
		apiPort = "3000"
	}
	ApiPort = apiPort

	path, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	currentWd = path

	dockerMountDataPath, ok := os.LookupEnv("DOCKER_MOUNT_DATA_PATH")
	if !ok {
		dockerMountDataPath = currentWd
	}

	DockerMountDataPath = dockerMountDataPath

	return ApiHostName, ApiPort
}

func ImageUrls(iconId int, apiType string) []string {
	baseUrl := fmt.Sprintf("%s://%s/dofus/img/%s", ApiScheme, ApiHostName, apiType)
	var urls []string
	urls = append(urls, fmt.Sprintf("%s/%d.png", baseUrl, iconId))

	if ImgResolutions == nil {
		return urls
	}

	for _, resolution := range ImgResolutions {
		finalImagePath := fmt.Sprintf("%s/data/img/%s/%d-%s.png", currentWd, apiType, iconId, resolution)
		resolutionUrl := fmt.Sprintf("%s/%d-%s.png", baseUrl, iconId, resolution)
		if ImgWithResExists.Has(finalImagePath) {
			urls = append(urls, resolutionUrl)
		}
	}
	return urls
}

func CreateMeiliClient() *meilisearch.Client {
	meiliPort, ok := os.LookupEnv("MEILISEARCH_PORT")
	if !ok {
		meiliPort = "7700"
	}

	meiliKey, ok := os.LookupEnv("MEILISEARCH_API_KEY")
	if !ok {
		meiliKey = "masterKey"
	}

	meiliHost, ok := os.LookupEnv("MEILISEARCH_HOST")
	if !ok {
		meiliHost = "http://127.0.0.1"
	}

	client := meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   fmt.Sprintf("%s:%s", meiliHost, meiliPort),
		APIKey: meiliKey,
	})

	return client
}

func CreateDataDirectoryStructure() {
	os.MkdirAll("data/tmp/vector", 0755)
	os.MkdirAll("data/img/item", 0755)
	os.MkdirAll("data/img/mount", 0755)

	os.MkdirAll("data/vector/item", 0755)
	os.MkdirAll("data/vector/mount", 0755)

	os.MkdirAll("data/languages", 0755)
}

type Config struct {
	CurrentVersion string `json:"currentDofusVersion"`
}

func GetConfig(path string) Config {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		fmt.Println(err)
	}

	return config
}

func SaveConfig(config Config, path string) error {
	configJson, err := json.Marshal(config)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, configJson, 0644)
	if err != nil {
		return err
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

	BiggestPageSize int
}

func PageninationWithState(paginationStr string) Pagination {
	vals := strings.Split(paginationStr, ",")
	num, _ := strconv.Atoi(vals[0])
	size, _ := strconv.Atoi(vals[1])
	max, _ := strconv.Atoi(vals[2])
	return Pagination{
		PageNumber:      num,
		PageSize:        size,
		BiggestPageSize: max,
	}
}

type PaginationLinks struct {
	First string `json:"first"`
	Prev  string `json:"prev"`
	Next  string `json:"next"`
	Last  string `json:"last"`
}

func (p Pagination) ValidatePagination(listSize int) int {
	if p.PageSize > p.BiggestPageSize {
		return -1
	}
	if p.BiggestPageSize*p.PageNumber > listSize+p.BiggestPageSize {
		return 1
	}
	return 0
}

func (p Pagination) BuildLinks(mainUrl url.URL, listSize int) (PaginationLinks, bool) {
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

	if p.PageNumber == firstPage {
		prevUrl = ""
	}
	if p.PageNumber == lastPage {
		nextUrl = ""
	}

	return PaginationLinks{
		First: firstUrl,
		Prev:  prevUrl,
		Next:  nextUrl,
		Last:  lastUrl,
	}, firstUrlStr == lastUrlStr
}

func (p Pagination) CalculateStartEndIndex(listSize int) (int, int) {
	startIndex := (p.PageNumber - 1) * p.PageSize
	endIndex := startIndex + p.PageSize
	if endIndex > listSize {
		endIndex = listSize
	}
	return startIndex, endIndex
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
