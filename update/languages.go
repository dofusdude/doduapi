package update

import (
	"log"
	"os"
	"sync"
)

func DownloadLanguages(hashJson map[string]interface{}) error {
	var wg sync.WaitGroup

	var deLangFile HashFile
	deLangFile.Filename = "data/i18n/i18n_de.d2i"
	deLangFile.FriendlyName = "data/tmp/lang_de.d2i"

	wg.Add(1)
	go func() {
		defer wg.Done()

		langDe := hashJson["lang_de"].(map[string]interface{})
		deFiles := langDe["files"].(map[string]interface{})
		deD2i := deFiles[deLangFile.Filename].(map[string]interface{})
		deLangFile.Hash = deD2i["hash"].(string)

		if err := DownloadHashFile(deLangFile); err != nil {
			log.Fatal(err)
		}

		Unpack(deLangFile.FriendlyName, "data/languages", "d2i")
		err := os.Remove(deLangFile.FriendlyName)
		if err != nil {
			log.Fatal(err)
		}
	}()

	var enLangFile HashFile
	enLangFile.Filename = "data/i18n/i18n_en.d2i"
	enLangFile.FriendlyName = "data/tmp/lang_en.d2i"

	wg.Add(1)
	go func() {
		defer wg.Done()

		langEn := hashJson["lang_en"].(map[string]interface{})
		enFiles := langEn["files"].(map[string]interface{})
		enD2i := enFiles[enLangFile.Filename].(map[string]interface{})
		enLangFile.Hash = enD2i["hash"].(string)
		if err := DownloadHashFile(enLangFile); err != nil {
			log.Fatal(err)
		}

		Unpack(enLangFile.FriendlyName, "data/languages", "d2i")
		if err := os.Remove(enLangFile.FriendlyName); err != nil {
			log.Fatal(err)
		}
	}()

	var esLangFile HashFile
	esLangFile.Filename = "data/i18n/i18n_es.d2i"
	esLangFile.FriendlyName = "data/tmp/lang_es.d2i"

	wg.Add(1)
	go func() {
		defer wg.Done()

		langEs := hashJson["lang_es"].(map[string]interface{})
		esFiles := langEs["files"].(map[string]interface{})
		esD2i := esFiles[esLangFile.Filename].(map[string]interface{})
		esLangFile.Hash = esD2i["hash"].(string)
		if err := DownloadHashFile(esLangFile); err != nil {
			log.Fatal(err)
		}

		Unpack(esLangFile.FriendlyName, "data/languages", "d2i")
		err := os.Remove(esLangFile.FriendlyName)
		if err != nil {
			log.Println(err)
		}
	}()

	var frLangFile HashFile
	frLangFile.Filename = "data/i18n/i18n_fr.d2i"
	frLangFile.FriendlyName = "data/tmp/lang_fr.d2i"

	wg.Add(1)
	go func() {
		defer wg.Done()

		langFr := hashJson["lang_fr"].(map[string]interface{})
		frFiles := langFr["files"].(map[string]interface{})
		frD2i := frFiles[frLangFile.Filename].(map[string]interface{})
		frLangFile.Hash = frD2i["hash"].(string)
		if err := DownloadHashFile(frLangFile); err != nil {
			log.Fatal(err)
		}

		Unpack(frLangFile.FriendlyName, "data/languages", "d2i")
		err := os.Remove(frLangFile.FriendlyName)
		if err != nil {
			log.Println(err)
		}
	}()

	var itLangFile HashFile
	itLangFile.Filename = "data/i18n/i18n_it.d2i"
	itLangFile.FriendlyName = "data/tmp/lang_it.d2i"

	wg.Add(1)
	go func() {
		defer wg.Done()

		langIt := hashJson["lang_it"].(map[string]interface{})
		itFiles := langIt["files"].(map[string]interface{})
		itD2i := itFiles[itLangFile.Filename].(map[string]interface{})
		itLangFile.Hash = itD2i["hash"].(string)
		if err := DownloadHashFile(itLangFile); err != nil {
			log.Fatal(err)
		}

		Unpack(itLangFile.FriendlyName, "data/languages", "d2i")
		err := os.Remove(itLangFile.FriendlyName)
		if err != nil {
			log.Println(err)
		}
	}()

	var ptLangFile HashFile
	ptLangFile.Filename = "data/i18n/i18n_pt.d2i"
	ptLangFile.FriendlyName = "data/tmp/lang_pt.d2i"

	wg.Add(1)
	go func() {
		defer wg.Done()

		langPt := hashJson["lang_pt"].(map[string]interface{})
		ptFiles := langPt["files"].(map[string]interface{})
		ptD2i := ptFiles[ptLangFile.Filename].(map[string]interface{})
		ptLangFile.Hash = ptD2i["hash"].(string)
		if err := DownloadHashFile(ptLangFile); err != nil {
			log.Fatal(err)
		}

		Unpack(ptLangFile.FriendlyName, "data/languages", "d2i")
		err := os.Remove(ptLangFile.FriendlyName)
		if err != nil {
			log.Println(err)
		}
	}()

	wg.Wait()
	return nil
}
