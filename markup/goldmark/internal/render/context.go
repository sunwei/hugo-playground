package render

import (
	"bytes"
	"github.com/sunwei/hugo-playground/identity"
	"github.com/sunwei/hugo-playground/markup/converter"
	"math/bits"
)

type Context struct {
	*BufWriter
	positions []int
	ContextData
}

type BufWriter struct {
	*bytes.Buffer
}

const maxInt = 1<<(bits.UintSize-1) - 1

func (b *BufWriter) Available() int {
	return maxInt
}

func (b *BufWriter) Buffered() int {
	return b.Len()
}

func (b *BufWriter) Flush() error {
	return nil
}

type ContextData interface {
	RenderContext() converter.RenderContext
	DocumentContext() converter.DocumentContext
	AddIdentity(id identity.Provider)
}

func (ctx *Context) PushPos(n int) {
	ctx.positions = append(ctx.positions, n)
}

func (ctx *Context) PopPos() int {
	i := len(ctx.positions) - 1
	p := ctx.positions[i]
	ctx.positions = ctx.positions[:i]
	return p
}

type RenderContextDataHolder struct {
	Rctx converter.RenderContext
	Dctx converter.DocumentContext
	IDs  identity.Manager
}

func (ctx *RenderContextDataHolder) RenderContext() converter.RenderContext {
	return ctx.Rctx
}

func (ctx *RenderContextDataHolder) DocumentContext() converter.DocumentContext {
	return ctx.Dctx
}

func (ctx *RenderContextDataHolder) AddIdentity(id identity.Provider) {
	ctx.IDs.Add(id)
}
