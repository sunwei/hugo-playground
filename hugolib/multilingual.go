package hugolib

import (
	"github.com/sunwei/hugo-playground/config"
	"github.com/sunwei/hugo-playground/langs"
)

func getLanguages(cfg config.Provider) langs.Languages {
	return langs.Languages{langs.NewDefaultLanguage(cfg)}
}
