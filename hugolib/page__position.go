package hugolib

import (
	"github.com/sunwei/hugo-playground/lazy"
	"github.com/sunwei/hugo-playground/resources/page"
)

type nextPrev struct {
	init     *lazy.Init
	prevPage page.Page
	nextPage page.Page
}

func (n *nextPrev) next() page.Page {
	n.init.Do()
	return n.nextPage
}

func (n *nextPrev) prev() page.Page {
	n.init.Do()
	return n.prevPage
}
