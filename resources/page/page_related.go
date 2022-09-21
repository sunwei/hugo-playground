package page

import (
	"github.com/sunwei/hugo-playground/related"
	"sync"
)

type RelatedDocsHandler struct {
	cfg related.Config

	postingLists []*cachedPostingList
	mu           sync.RWMutex
}

type cachedPostingList struct {
	p Pages

	postingList *related.InvertedIndex
}
