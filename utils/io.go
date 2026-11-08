package utils

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

	"github.com/emirpasic/gods/maps/treebidimap"
	gutils "github.com/emirpasic/gods/utils"
	_ "github.com/joho/godotenv/autoload"
)

func Concat[T any](first []T, second []T) []T {
	n := len(first)
	return append(first[:n:n], second...)
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

func DownloadExtract(filename string, dockerMountDataPath string, releaseUrl string) error {
	absUrl := fmt.Sprintf("%s/%s.tar.gz", releaseUrl, filename)
	response, err := http.Get(absUrl)
	if err != nil {
		return err
	}
	err = ExtractTarGz(dockerMountDataPath, response.Body)
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

func DownloadImages(dockerMountDataPath string, releaseUrl string) error {
	var err error

	// -- items --
	err = DownloadExtract(fmt.Sprintf("items_images_64"), dockerMountDataPath, releaseUrl)
	if err != nil {
		return fmt.Errorf("could not download items_images: %v", err)
	}

	err = DownloadExtract(fmt.Sprintf("items_images_128"), dockerMountDataPath, releaseUrl)
	if err != nil {
		return fmt.Errorf("could not download items_images: %v", err)
	}

	oldPath1x := filepath.Join(dockerMountDataPath, "data", "img", "item", "1x")
	oldPath2x := filepath.Join(dockerMountDataPath, "data", "img", "item", "2x")
	newPath := filepath.Join(dockerMountDataPath, "data", "img", "item")

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

	return nil
}

func ImageUrls(iconId int, apiType string, resolutions []string, apiScheme string, doduapiMajorVersion int, apiHostname string, beta bool) []string {
	betaImage := ""
	if beta {
		betaImage = "beta"
	}
	baseUrl := fmt.Sprintf("%s://%s/dofus3%s/v%d/img/%s", apiScheme, apiHostname, betaImage, doduapiMajorVersion, apiType)
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

func LoadPersistedElements(elementsUrl string, typesUrl string) (PersistentStringKeysMap, PersistentStringKeysMap, error) {
	read := func(address string) ([]byte, error) {
		response, err := http.Get(address)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("download persistence: %s", response.Status)
		}
		return io.ReadAll(response.Body)
	}
	elementsBody, err := read(elementsUrl)
	if err != nil {
		return PersistentStringKeysMap{}, PersistentStringKeysMap{}, err
	}
	typesBody, err := read(typesUrl)
	if err != nil {
		return PersistentStringKeysMap{}, PersistentStringKeysMap{}, err
	}
	return parsePersistedElements(elementsBody, typesBody)
}

func LoadPersistedElementsFiles(elementsPath string, typesPath string) (PersistentStringKeysMap, PersistentStringKeysMap, error) {
	elementsBody, err := os.ReadFile(elementsPath)
	if err != nil {
		return PersistentStringKeysMap{}, PersistentStringKeysMap{}, err
	}
	typesBody, err := os.ReadFile(typesPath)
	if err != nil {
		return PersistentStringKeysMap{}, PersistentStringKeysMap{}, err
	}
	return parsePersistedElements(elementsBody, typesBody)
}

func parsePersistedElements(elementsBody []byte, typesBody []byte) (PersistentStringKeysMap, PersistentStringKeysMap, error) {
	var elements []string
	if err := json.Unmarshal(elementsBody, &elements); err != nil {
		return PersistentStringKeysMap{}, PersistentStringKeysMap{}, err
	}
	var types []string
	if err := json.Unmarshal(typesBody, &types); err != nil {
		return PersistentStringKeysMap{}, PersistentStringKeysMap{}, err
	}

	persistedElements := PersistentStringKeysMap{Entries: treebidimap.NewWith(gutils.IntComparator, gutils.StringComparator)}
	for _, entry := range elements {
		persistedElements.Entries.Put(persistedElements.NextId, entry)
		persistedElements.NextId++
	}
	persistedTypes := PersistentStringKeysMap{Entries: treebidimap.NewWith(gutils.IntComparator, gutils.StringComparator)}
	for _, entry := range types {
		persistedTypes.Entries.Put(persistedTypes.NextId, entry)
		persistedTypes.NextId++
	}
	return persistedElements, persistedTypes, nil
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
	if p.PageSize < -1 || p.PageSize == 0 {
		return -1
	}
	if (p.PageSize * p.PageNumber) >= listSize+p.PageSize {
		return 1
	}
	return 0
}

func (p *Pagination) BuildLinks(mainUrl url.URL, listSize int, apiScheme string, apiHostname string) (PaginationLinks, bool) {
	if p.PageSize <= 0 {
		return PaginationLinks{}, true
	}
	last := max(1, (listSize+p.PageSize-1)/p.PageSize)
	link := func(number int) *string {
		u := mainUrl
		u.Scheme, u.Host = apiScheme, apiHostname
		values := u.Query()
		values.Set("page[number]", strconv.Itoa(number))
		values.Set("page[size]", strconv.Itoa(p.PageSize))
		u.RawQuery = values.Encode()
		value := u.String()
		return &value
	}
	links := PaginationLinks{}
	if last > 1 {
		links.First, links.Last = link(1), link(last)
	}
	if p.PageNumber > 1 {
		links.Prev = link(p.PageNumber - 1)
	}
	if p.PageNumber < last {
		links.Next = link(p.PageNumber + 1)
	}
	return links, last == 1
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
