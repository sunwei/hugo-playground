package hugolib

import (
	"github.com/sunwei/hugo-playground/resources/page"
	"sync"
)

type pageData struct {
	*pageState

	dataInit sync.Once
	data     page.Data
}
