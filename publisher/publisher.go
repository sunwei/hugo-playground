package publisher

import (
	"errors"
	"fmt"
	"github.com/spf13/afero"
	bp "github.com/sunwei/hugo-playground/bufferpool"
	"github.com/sunwei/hugo-playground/helpers"
	"github.com/sunwei/hugo-playground/media"
	"github.com/sunwei/hugo-playground/minifiers"
	"github.com/sunwei/hugo-playground/output"
	"github.com/sunwei/hugo-playground/resources"
	"github.com/sunwei/hugo-playground/transform"
	"github.com/sunwei/hugo-playground/transform/livereloadinject"
	"github.com/sunwei/hugo-playground/transform/metainject"
	"github.com/sunwei/hugo-playground/transform/urlreplacers"
	"io"
	"net/url"
)

// Publisher publishes a result file.
type Publisher interface {
	Publish(d Descriptor) error
}

// Descriptor describes the needed publishing chain for an item.
type Descriptor struct {
	// The content to publish.
	Src io.Reader

	// The OutputFormat of the this content.
	OutputFormat output.Format

	// Where to publish this content. This is a filesystem-relative path.
	TargetPath string

	// Configuration that trigger pre-processing.
	// LiveReload script will be injected if this is != nil
	LiveReloadBaseURL *url.URL

	// Enable to inject the Hugo generated tag in the header. Is currently only
	// injected on the home page for HTML type of output formats.
	AddHugoGeneratorTag bool

	// If set, will replace all relative URLs with this one.
	AbsURLPath string

	// Enable to minify the output using the OutputFormat defined above to
	// pick the correct minifier configuration.
	Minify bool
}

// NewDestinationPublisher creates a new DestinationPublisher.
func NewDestinationPublisher(rs *resources.Spec, outputFormats output.Formats, mediaTypes media.Types) (pub DestinationPublisher, err error) {
	fs := rs.BaseFs.PublishFs
	cfg := rs.Cfg

	pub = DestinationPublisher{fs: fs}
	pub.min, err = minifiers.New(mediaTypes, outputFormats, cfg)
	return
}

// DestinationPublisher is the default and currently only publisher in Hugo. This
// publisher prepares and publishes an item to the defined destination, e.g. /public.
type DestinationPublisher struct {
	fs  afero.Fs
	min minifiers.Client
}

// Publish applies any relevant transformations and writes the file
// to its destination, e.g. /public.
func (p DestinationPublisher) Publish(d Descriptor) error {
	if d.TargetPath == "" {
		return errors.New("publish: must provide a TargetPath")
	}

	src := d.Src

	transformers := p.createTransformerChain(d)

	if len(transformers) != 0 {
		b := bp.GetBuffer()
		defer bp.PutBuffer(b)

		if err := transformers.Apply(b, d.Src); err != nil {
			return fmt.Errorf("failed to process %q: %w", d.TargetPath, err)
		}

		// This is now what we write to disk.
		src = b
	}

	f, err := helpers.OpenFileForWriting(p.fs, d.TargetPath)
	if err != nil {
		return err
	}
	defer f.Close()

	var w io.Writer = f

	_, err = io.Copy(w, src)

	return err
}

// XML transformer := transform.New(urlreplacers.NewAbsURLInXMLTransformer(path))
func (p DestinationPublisher) createTransformerChain(f Descriptor) transform.Chain {
	transformers := transform.NewEmpty()

	isHTML := f.OutputFormat.IsHTML

	if f.AbsURLPath != "" {
		if isHTML {
			transformers = append(transformers, urlreplacers.NewAbsURLTransformer(f.AbsURLPath))
		} else {
			// Assume XML.
			transformers = append(transformers, urlreplacers.NewAbsURLInXMLTransformer(f.AbsURLPath))
		}
	}

	if isHTML {
		if f.LiveReloadBaseURL != nil {
			transformers = append(transformers, livereloadinject.New(*f.LiveReloadBaseURL))
		}

		// This is only injected on the home page.
		if f.AddHugoGeneratorTag {
			transformers = append(transformers, metainject.HugoGenerator)
		}

	}

	return transformers
}
