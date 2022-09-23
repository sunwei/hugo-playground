package hugolib

import (
	"github.com/sunwei/hugo-playground/config"
	"github.com/sunwei/hugo-playground/langs"
	"github.com/sunwei/hugo-playground/log"
)

func getLanguages(cfg config.Provider) langs.Languages {
	log.Process("NewLanguages", "create multiple languages, only 'en' in our case")
	return langs.Languages{langs.NewDefaultLanguage(cfg)}
}
