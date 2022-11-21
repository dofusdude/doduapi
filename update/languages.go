package update

import (
	"github.com/dofusdude/ankabuffer"
	"log"
	"os"
	"sync"
)

func DownloadLanguages(hashJson *ankabuffer.Manifest) error {
	var wg sync.WaitGroup

	var deLangFile HashFile
	deLangFile.Filename = "data/i18n/i18n_de.d2i"
	deLangFile.FriendlyName = "data/tmp/lang_de.d2i"
	deLangFile.Hash = hashJson.Fragments["lang_de"].Files[deLangFile.Filename].Hash

	wg.Add(1)
	go func() {
		defer wg.Done()

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
	enLangFile.Hash = hashJson.Fragments["lang_en"].Files[enLangFile.Filename].Hash

	wg.Add(1)
	go func() {
		defer wg.Done()

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
	esLangFile.Hash = hashJson.Fragments["lang_es"].Files[esLangFile.Filename].Hash

	wg.Add(1)
	go func() {
		defer wg.Done()

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
	frLangFile.Hash = hashJson.Fragments["lang_fr"].Files[frLangFile.Filename].Hash

	wg.Add(1)
	go func() {
		defer wg.Done()

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
	itLangFile.Hash = hashJson.Fragments["lang_it"].Files[itLangFile.Filename].Hash

	wg.Add(1)
	go func() {
		defer wg.Done()

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
	ptLangFile.Hash = hashJson.Fragments["lang_pt"].Files[ptLangFile.Filename].Hash

	wg.Add(1)
	go func() {
		defer wg.Done()

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
