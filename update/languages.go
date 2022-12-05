package update

import (
	"github.com/dofusdude/ankabuffer"
	"log"
)

func DownloadLanguageFiles(hashJson ankabuffer.Manifest, lang string) error {
	var langFile HashFile
	langFile.Filename = "data/i18n/i18n_" + lang + ".d2i"
	langFile.FriendlyName = "data/tmp/lang_" + lang + ".d2i"
	if err := DownloadUnpackFiles(hashJson, "lang_"+lang, []HashFile{langFile}, "data/languages", true); err != nil {
		return err
	}
	return nil
}

func DownloadLanguages(hashJson ankabuffer.Manifest) error {
	langs := []string{"fr", "en", "es", "de", "it", "pt"}

	fail := make(chan error)
	for _, lang := range langs {
		go func(lang string, fail chan error) {
			fail <- DownloadLanguageFiles(hashJson, lang)
		}(lang, fail)
	}

	var someFail error
	log.Println("Downloading languages...")
	for _, lang := range langs {
		if err := <-fail; err != nil {
			someFail = err
		}
		log.Println("... " + lang)
	}

	return someFail
}
