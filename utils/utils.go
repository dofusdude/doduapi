package utils

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/meilisearch/meilisearch-go"
)

var Languages = []string{"de", "en", "es", "fr", "it", "pt"}

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
	os.MkdirAll("data/tmp", 0755)
	os.MkdirAll("data/img/item", 0755)
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
