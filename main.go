package main

import (
	"fmt"
	"github.com/sunwei/hugo-playground/helpers"
	"github.com/sunwei/hugo-playground/markup/converter"
	"github.com/sunwei/hugo-playground/markup/converter/hooks"
)

func main() {
	contentSpec, _ := helpers.NewContentSpec()

	provider := contentSpec.Converters.Get("goldmark")

	goldmarkConverter, _ := provider.New(
		converter.DocumentContext{
			Document:     nil,
			DocumentID:   "id",
			DocumentName: "path",
			Filename:     "filename",
		})

	r, _ := goldmarkConverter.Convert(
		converter.RenderContext{
			Src:       []byte("### first blog\nHello Blog\n### first section"),
			RenderTOC: true,
			GetRenderer: func(tp hooks.RendererType, id any) any {
				return nil
			},
		})

	fmt.Println(r)
}
