package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dofusdude/ankabuffer"
	"github.com/emirpasic/gods/maps/treebidimap"
	gutils "github.com/emirpasic/gods/utils"
	"github.com/go-redis/redis/v9"
	_ "github.com/joho/godotenv/autoload"
	"github.com/meilisearch/meilisearch-go"
)

var (
	Languages           = []string{"de", "en", "es", "fr", "it", "pt"}
	ImgResolutions      = []string{"200", "400", "800"}
	ImgWithResExists    *Set
	ApiHostName         string
	ApiPort             string
	ApiScheme           string
	DockerMountDataPath string
	FileHashes          ankabuffer.Manifest
	MeiliHost           string
	MeiliKey            string
	PrometheusEnabled   bool
	FileServer          bool
	PersistedElements   PersistentStringKeysMap
	PersistedTypes      PersistentStringKeysMap
	IsBeta              bool
	LastUpdate          time.Time
	RedisHost           string
	RedisPassword       string
	PythonPath          string
)

var currentWd string

func GetReleaseManifest(version string) (ankabuffer.Manifest, error) {
	var gameVersionType string
	if IsBeta {
		gameVersionType = "beta"
	} else {
		gameVersionType = "main"
	}
	gameHashesUrl := fmt.Sprintf("https://cytrus.cdn.ankama.com/dofus/releases/%s/windows/%s.manifest", gameVersionType, version)
	hashResponse, err := http.Get(gameHashesUrl)
	if err != nil {
		log.Println(err)
		return ankabuffer.Manifest{}, err
	}

	hashBody, err := io.ReadAll(hashResponse.Body)
	if err != nil {
		log.Println(err)
		return ankabuffer.Manifest{}, err
	}

	FileHashes = *ankabuffer.ParseManifest(hashBody)

	marshalledBytes, _ := json.MarshalIndent(FileHashes, "", "  ")
	os.WriteFile("data/manifest.json", marshalledBytes, os.ModePerm)

	return FileHashes, nil
}

type VersionT struct {
	Search bool
	MemDb  bool
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

func ReadEnvs() {
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

	meiliPort, ok := os.LookupEnv("MEILI_PORT")
	if !ok {
		meiliPort = "7700"
	}

	meiliKey, ok := os.LookupEnv("MEILI_MASTER_KEY")
	if !ok {
		meiliKey = "masterKey"
	}

	MeiliKey = meiliKey

	meiliProtocol, ok := os.LookupEnv("MEILI_PROTOCOL")
	if !ok {
		meiliProtocol = "http"
	}

	meiliHost, ok := os.LookupEnv("MEILI_HOST")
	if !ok {
		meiliHost = "127.0.0.1"
	}

	MeiliHost = fmt.Sprintf("%s://%s:%s", meiliProtocol, meiliHost, meiliPort)

	promEnables, ok := os.LookupEnv("PROMETHEUS")
	if !ok {
		promEnables = ""
	}

	pythonPath, ok := os.LookupEnv("PYTHON_PATH")
	if !ok {
		pythonPath = "/usr/bin/python3"
	}

	PythonPath = pythonPath

	PrometheusEnabled = strings.ToLower(promEnables) == "true"

	fileServer, ok := os.LookupEnv("FILESERVER")
	if !ok {
		fileServer = "true"
	}

	FileServer = strings.ToLower(fileServer) == "true"

	isBeta, ok := os.LookupEnv("IS_BETA")
	if !ok {
		isBeta = "false"
	}

	IsBeta = strings.ToLower(isBeta) == "true"

	redisHost, ok := os.LookupEnv("REDIS_HOST")
	if !ok {
		redisHost = "127.0.0.1"
	}

	RedisHost = redisHost

	redisPassword, ok := os.LookupEnv("REDIS_PASSWORD")
	if !ok {
		redisPassword = "secret"
	}

	RedisPassword = redisPassword
}

func ImageUrls(iconId int, apiType string) []string {
	betaImage := ""
	if IsBeta {
		betaImage = "beta"
	}
	baseUrl := fmt.Sprintf("%s://%s/dofus2%s/img/%s", ApiScheme, ApiHostName, betaImage, apiType)
	var urls []string
	urls = append(urls, fmt.Sprintf("%s/%d.png", baseUrl, iconId))

	if ImgResolutions == nil || ImgWithResExists == nil {
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
	client := meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   MeiliHost,
		APIKey: MeiliKey,
	})

	return client
}

func touchFileIfNotExists(fileName string) error {
	_, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		file, err := os.Create(fileName)
		if err != nil {
			return err
		}
		defer func() {
			if err := file.Close(); err != nil {
				log.Println(err)
			}
		}()
	}

	return nil
}

func CreateDataDirectoryStructure() {
	os.MkdirAll("data/tmp/vector", os.ModePerm)
	os.MkdirAll("data/img/item", os.ModePerm)
	os.MkdirAll("data/img/mount", os.ModePerm)

	os.MkdirAll("data/vector/item", os.ModePerm)
	os.MkdirAll("data/vector/mount", os.ModePerm)

	os.MkdirAll("data/languages", os.ModePerm)

	err := touchFileIfNotExists("data/img/index.html")
	if err != nil {
		log.Println(err)
	}
	err = touchFileIfNotExists("data/img/item/index.html")
	if err != nil {
		log.Println(err)
	}
	err = touchFileIfNotExists("data/img/mount/index.html")
	if err != nil {
		log.Println(err)
	}

}

func DeleteReplacer(input string) string {
	replacer := []string{
		"#",
		"%",
	}

	for i := 1; i < 6; i++ {
		for _, replace := range replacer {
			numRegex := regexp.MustCompile(fmt.Sprintf(" ?%s%d", replace, i))
			input = numRegex.ReplaceAllString(input, "")
		}
	}
	return input
}

type PersistentStringKeysMap struct {
	Entries *treebidimap.Map `json:"entries"`
	NextId  int              `json:"next_id"`
}

func LoadPersistedElements() error {
	log.Println("loading persisted elements...")

	var element_path string
	var item_type_path string

	element_path = fmt.Sprintf("%s/db/elements.json", DockerMountDataPath)
	item_type_path = fmt.Sprintf("%s/db/item_types.json", DockerMountDataPath)

	data, err := os.ReadFile(element_path)
	if err != nil {
		return err
	}

	var elements []string
	err = json.Unmarshal(data, &elements)
	if err != nil {
		fmt.Println(err)
	}

	PersistedElements = PersistentStringKeysMap{
		Entries: treebidimap.NewWith(gutils.IntComparator, gutils.StringComparator),
		NextId:  0,
	}

	for _, entry := range elements {
		PersistedElements.Entries.Put(PersistedElements.NextId, entry)
		PersistedElements.NextId++
	}

	data, err = os.ReadFile(item_type_path)
	if err != nil {
		return err
	}

	var types []string
	err = json.Unmarshal(data, &types)
	if err != nil {
		fmt.Println(err)
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

func PersistElements(element_path string, item_type_path string) error {
	elements := make([]string, PersistedElements.NextId)
	it := PersistedElements.Entries.Iterator()
	for it.Next() {
		elements[it.Key().(int)] = it.Value().(string)
	}

	elementsJson, err := json.MarshalIndent(elements, "", "    ")
	if err != nil {
		return err
	}
	err = os.WriteFile(element_path, elementsJson, 0644)
	if err != nil {
		return err
	}

	types := make([]string, PersistedTypes.NextId)
	it = PersistedTypes.Entries.Iterator()
	for it.Next() {
		types[it.Key().(int)] = it.Value().(string)
	}

	typesJson, err := json.MarshalIndent(types, "", "    ")
	if err != nil {
		return err
	}
	err = os.WriteFile(item_type_path, typesJson, 0644)
	if err != nil {
		return err
	}
	return nil
}

func GetLatestLauncherVersion() string {
	versionResponse, err := http.Get("https://cytrus.cdn.ankama.com/cytrus.json")
	if err != nil {
		log.Fatalln(err)
	}

	versionBody, err := io.ReadAll(versionResponse.Body)
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

	if IsBeta {
		return windows["beta"].(string)
	} else {
		return windows["main"].(string)
	}
}

func NewVersion(version string) error {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:6379", RedisHost),
		Password: RedisPassword,
		DB:       0, // use default DB
	})

	var ctx = context.Background()

	var versionPrefix string
	if IsBeta {
		versionPrefix = "Beta"
	} else {
		versionPrefix = "Main"
	}

	err := rdb.Set(ctx, fmt.Sprintf("dofus2%sVersion", versionPrefix), version, 0).Err()
	if err != nil {
		return err
	}

	err = rdb.Set(ctx, fmt.Sprintf("dofus2%sVersionUpdated", versionPrefix), time.Now().Format(http.TimeFormat), 0).Err()
	if err != nil {
		return err
	}

	return nil
}

func GetCurrentVersion() string {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:6379", RedisHost),
		Password: RedisPassword,
		DB:       0, // use default DB
	})

	var ctx = context.Background()

	var versionPrefix string
	if IsBeta {
		versionPrefix = "Beta"
	} else {
		versionPrefix = "Main"
	}

	val, err := rdb.Get(ctx, fmt.Sprintf("dofus2%sVersion", versionPrefix)).Result()
	if err != nil {
		return ""
	}

	changedTime, err := rdb.Get(ctx, fmt.Sprintf("dofus2%sVersionUpdated", versionPrefix)).Result()
	if err != nil {
		return ""
	}

	LastUpdate, err = http.ParseTime(changedTime)
	if err != nil {
		return ""
	}

	return val
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
