package registry

import (
	"io"
)

type RepositoryInterface interface {
	ListTag() ([]string, error)
	ManifestExist(reference string) (digest string, exist bool, err error)
	PullManifest(reference string, acceptMediaTypes []string) (digest, mediaType string, payload []byte, err error)
	PushManifest(reference, mediaType string, payload []byte) (digest string, err error)
	DeleteManifest(digest string) error
	DeleteTag(tag string) error
	BlobExist(digest string) (bool, error)
	PullBlob(digest string) (size int64, data io.ReadCloser, err error)
	PushBlob(digest string, size int64, data io.Reader) error
	DeleteBlob(digest string) error
}