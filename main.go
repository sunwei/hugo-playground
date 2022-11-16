package main

import (
	"fmt"
	"github.com/sunwei/hugo-playground/helpers"
	"github.com/sunwei/hugo-playground/markup/converter"
	"github.com/sunwei/hugo-playground/markup/converter/hooks"
)

func main() {
	contentSpec, _ := helpers.NewContentSpec()

	cp := contentSpec.Converters.Get("goldmark")

	cpp, _ := cp.New(
		converter.DocumentContext{
			Document:     nil,
			DocumentID:   "id",
			DocumentName: "path",
			Filename:     "filename",
		})

	src := "### first blog\nHello Blog"
	r, _ := cpp.Convert(
		converter.RenderContext{
			Src:       []byte(src),
			RenderTOC: false,
			GetRenderer: func(tp hooks.RendererType, id any) any {
				return nil
			},
		})

	fmt.Println(r)

}
