package update

import (
	"github.com/dofusdude/ankabuffer"
)

func DownloadLanguages(hashJson *ankabuffer.Manifest) error {
	var deLangFile HashFile
	deLangFile.Filename = "data/i18n/i18n_de.d2i"
	deLangFile.FriendlyName = "data/tmp/lang_de.d2i"
	DownloadUnpackFiles(hashJson, "lang_de", []HashFile{deLangFile}, "data/languages", true)

	var enLangFile HashFile
	enLangFile.Filename = "data/i18n/i18n_en.d2i"
	enLangFile.FriendlyName = "data/tmp/lang_en.d2i"
	DownloadUnpackFiles(hashJson, "lang_en", []HashFile{enLangFile}, "data/languages", true)

	var esLangFile HashFile
	esLangFile.Filename = "data/i18n/i18n_es.d2i"
	esLangFile.FriendlyName = "data/tmp/lang_es.d2i"
	DownloadUnpackFiles(hashJson, "lang_es", []HashFile{esLangFile}, "data/languages", true)

	var frLangFile HashFile
	frLangFile.Filename = "data/i18n/i18n_fr.d2i"
	frLangFile.FriendlyName = "data/tmp/lang_fr.d2i"
	DownloadUnpackFiles(hashJson, "lang_fr", []HashFile{frLangFile}, "data/languages", true)

	var itLangFile HashFile
	itLangFile.Filename = "data/i18n/i18n_it.d2i"
	itLangFile.FriendlyName = "data/tmp/lang_it.d2i"
	DownloadUnpackFiles(hashJson, "lang_it", []HashFile{itLangFile}, "data/languages", true)

	var ptLangFile HashFile
	ptLangFile.Filename = "data/i18n/i18n_pt.d2i"
	ptLangFile.FriendlyName = "data/tmp/lang_pt.d2i"
	DownloadUnpackFiles(hashJson, "lang_pt", []HashFile{ptLangFile}, "data/languages", true)

	return nil
}
