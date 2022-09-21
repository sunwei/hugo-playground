package resource

import (
	"github.com/sunwei/hugo-playground/common/hugio"
	"github.com/sunwei/hugo-playground/common/maps"
	"github.com/sunwei/hugo-playground/langs"
	"github.com/sunwei/hugo-playground/media"
)

// Resource represents a linkable resource, i.e. a content page, image etc.
type Resource interface {
	ResourceTypeProvider
	ResourceLinksProvider
	ResourceMetaProvider
	ResourceParamsProvider
	ErrProvider
}

type ResourceTypeProvider interface {
	// ResourceType is the resource type. For most file types, this is the main
	// part of the MIME type, e.g. "image", "application", "text" etc.
	// For content pages, this value is "page".
	ResourceType() string
}

type MediaTypeProvider interface {
	// MediaType is this resource's MIME type.
	MediaType() media.Type
}

type ResourceLinksProvider interface {
	// Permalink represents the absolute link to this resource.
	Permalink() string

	// RelPermalink represents the host relative link to this resource.
	RelPermalink() string
}

type ResourceMetaProvider interface {
	// Name is the logical name of this resource. This can be set in the front matter
	// metadata for this resource. If not set, Hugo will assign a value.
	// This will in most cases be the base filename.
	// So, for the image "/some/path/sunset.jpg" this will be "sunset.jpg".
	// The value returned by this method will be used in the GetByPrefix and ByPrefix methods
	// on Resources.
	Name() string

	// Title returns the title if set in front matter. For content pages, this will be the expected value.
	Title() string
}

type ResourceParamsProvider interface {
	// Params set in front matter for this resource.
	Params() maps.Params
}

type ResourceDataProvider interface {
	// Resource specific data set by Hugo.
	// One example would be.Data.Digest for fingerprinted resources.
	Data() any
}

// ErrProvider provides an Err.
type ErrProvider interface {
	Err() ResourceError
}

// ResourceError is the error return from .Err in Resource in error situations.
type ResourceError interface {
	error
	ResourceDataProvider
}

// Cloner is for internal use.
type Cloner interface {
	Clone() Resource
}

// ContentProvider provides Content.
// This should be used with care, as it will read the file content into memory, but it
// should be cached as effectively as possible by the implementation.
type ContentProvider interface {
	// Content returns this resource's content. It will be equivalent to reading the content
	// that RelPermalink points to in the published folder.
	// The return type will be contextual, and should be what you would expect:
	// * Page: template.HTML
	// * JSON: String
	// * Etc.
	Content() (any, error)
}

// Identifier identifies a resource.
type Identifier interface {
	Key() string
}

// OriginProvider provides the original Resource if this is wrapped.
// This is an internal Hugo interface and not meant for use in the templates.
type OriginProvider interface {
	Origin() Resource
	GetFieldString(pattern string) (string, bool)
}

func NewResourceTypesProvider(mediaType media.Type, resourceType string) ResourceTypesProvider {
	return resourceTypesHolder{mediaType: mediaType, resourceType: resourceType}
}

type ResourceTypesProvider interface {
	ResourceTypeProvider
	MediaTypeProvider
}

type resourceTypesHolder struct {
	mediaType    media.Type
	resourceType string
}

func (r resourceTypesHolder) MediaType() media.Type {
	return r.mediaType
}

func (r resourceTypesHolder) ResourceType() string {
	return r.resourceType
}

// LanguageProvider is a Resource in a language.
type LanguageProvider interface {
	Language() *langs.Language
}

// UnmarshableResource represents a Resource that can be unmarshaled to some other format.
type UnmarshableResource interface {
	ReadSeekCloserResource
	Identifier
}

// ReadSeekCloserResource is a Resource that supports loading its content.
type ReadSeekCloserResource interface {
	MediaType() media.Type
	hugio.ReadSeekCloserProvider
}

// OpenReadSeekCloser allows setting some other way (than reading from a filesystem)
// to open or create a ReadSeekCloser.
type OpenReadSeekCloser func() (hugio.ReadSeekCloser, error)
