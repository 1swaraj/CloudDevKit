package memblob

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/swaraj1802/CloudDevKit/genericstorage"
	"github.com/swaraj1802/CloudDevKit/genericstorage/driver"
	"github.com/swaraj1802/CloudDevKit/gcerrors"
)

const defaultPageSize = 1000

var (
	errNotFound       = errors.New("genericstorage not found")
	errNotImplemented = errors.New("not implemented")
)

func init() {
	genericstorage.DefaultURLMux().RegisterBucket(Scheme, &URLOpener{})
}

const Scheme = "mem"

type URLOpener struct{}

func (*URLOpener) OpenBucketURL(ctx context.Context, u *url.URL) (*genericstorage.Bucket, error) {
	for param := range u.Query() {
		return nil, fmt.Errorf("open bucket %v: invalid query parameter %q", u, param)
	}
	return OpenBucket(nil), nil
}

type Options struct{}

type blobEntry struct {
	Content    []byte
	Attributes *driver.Attributes
}

type bucket struct {
	mu    sync.Mutex
	blobs map[string]*blobEntry
}

func openBucket(_ *Options) driver.Bucket {
	return &bucket{
		blobs: map[string]*blobEntry{},
	}
}

func OpenBucket(opts *Options) *genericstorage.Bucket {
	return genericstorage.NewBucket(openBucket(opts))
}

func (b *bucket) Close() error {
	return nil
}

func (b *bucket) ErrorCode(err error) gcerrors.ErrorCode {
	switch err {
	case errNotFound:
		return gcerrors.NotFound
	case errNotImplemented:
		return gcerrors.Unimplemented
	default:
		return gcerrors.Unknown
	}
}

func (b *bucket) ListPaged(ctx context.Context, opts *driver.ListOptions) (*driver.ListPage, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var pageToken string
	if len(opts.PageToken) > 0 {
		pageToken = string(opts.PageToken)
	}
	pageSize := opts.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}

	var keys []string
	for key := range b.blobs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var lastPrefix string
	var result driver.ListPage
	for _, key := range keys {

		if !strings.HasPrefix(key, opts.Prefix) {
			continue
		}

		entry := b.blobs[key]
		obj := &driver.ListObject{
			Key:     key,
			ModTime: entry.Attributes.ModTime,
			Size:    entry.Attributes.Size,
			MD5:     entry.Attributes.MD5,
		}

		if opts.Delimiter != "" {

			keyWithoutPrefix := key[len(opts.Prefix):]

			if idx := strings.Index(keyWithoutPrefix, opts.Delimiter); idx != -1 {
				prefix := opts.Prefix + keyWithoutPrefix[0:idx+len(opts.Delimiter)]

				if prefix == lastPrefix {
					continue
				}

				obj = &driver.ListObject{
					Key:   prefix,
					IsDir: true,
				}
				lastPrefix = prefix
			}
		}

		if pageToken != "" && obj.Key <= pageToken {
			continue
		}

		if len(result.Objects) == pageSize {
			result.NextPageToken = []byte(result.Objects[pageSize-1].Key)
			return &result, nil
		}
		result.Objects = append(result.Objects, obj)
	}
	return &result, nil
}

func (b *bucket) As(i interface{}) bool { return false }

func (b *bucket) ErrorAs(err error, i interface{}) bool { return false }

func (b *bucket) Attributes(ctx context.Context, key string) (*driver.Attributes, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	entry, found := b.blobs[key]
	if !found {
		return nil, errNotFound
	}
	return entry.Attributes, nil
}

func (b *bucket) NewRangeReader(ctx context.Context, key string, offset, length int64, opts *driver.ReaderOptions) (driver.Reader, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	entry, found := b.blobs[key]
	if !found {
		return nil, errNotFound
	}

	if opts.BeforeRead != nil {
		if err := opts.BeforeRead(func(interface{}) bool { return false }); err != nil {
			return nil, err
		}
	}
	r := bytes.NewReader(entry.Content)
	if offset > 0 {
		if _, err := r.Seek(offset, io.SeekStart); err != nil {
			return nil, err
		}
	}
	var ior io.Reader = r
	if length >= 0 {
		ior = io.LimitReader(r, length)
	}
	return &reader{
		r: ior,
		attrs: driver.ReaderAttributes{
			ContentType: entry.Attributes.ContentType,
			ModTime:     entry.Attributes.ModTime,
			Size:        entry.Attributes.Size,
		},
	}, nil
}

type reader struct {
	r     io.Reader
	attrs driver.ReaderAttributes
}

func (r *reader) Read(p []byte) (int, error) {
	return r.r.Read(p)
}

func (r *reader) Close() error {
	return nil
}

func (r *reader) Attributes() *driver.ReaderAttributes {
	return &r.attrs
}

func (r *reader) As(i interface{}) bool { return false }

func (b *bucket) NewTypedWriter(ctx context.Context, key string, contentType string, opts *driver.WriterOptions) (driver.Writer, error) {
	if key == "" {
		return nil, errors.New("invalid key (empty string)")
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	if opts.BeforeWrite != nil {
		if err := opts.BeforeWrite(func(interface{}) bool { return false }); err != nil {
			return nil, err
		}
	}
	md := map[string]string{}
	for k, v := range opts.Metadata {
		md[k] = v
	}
	return &writer{
		ctx:         ctx,
		b:           b,
		key:         key,
		contentType: contentType,
		metadata:    md,
		opts:        opts,
		md5hash:     md5.New(),
	}, nil
}

type writer struct {
	ctx         context.Context
	b           *bucket
	key         string
	contentType string
	metadata    map[string]string
	opts        *driver.WriterOptions
	buf         bytes.Buffer

	md5hash hash.Hash
}

func (w *writer) Write(p []byte) (n int, err error) {
	if _, err := w.md5hash.Write(p); err != nil {
		return 0, err
	}
	return w.buf.Write(p)
}

func (w *writer) Close() error {

	if err := w.ctx.Err(); err != nil {
		return err
	}

	md5sum := w.md5hash.Sum(nil)
	content := w.buf.Bytes()
	now := time.Now()
	entry := &blobEntry{
		Content: content,
		Attributes: &driver.Attributes{
			CacheControl:       w.opts.CacheControl,
			ContentDisposition: w.opts.ContentDisposition,
			ContentEncoding:    w.opts.ContentEncoding,
			ContentLanguage:    w.opts.ContentLanguage,
			ContentType:        w.contentType,
			Metadata:           w.metadata,
			Size:               int64(len(content)),
			CreateTime:         now,
			ModTime:            now,
			MD5:                md5sum,
			ETag:               fmt.Sprintf("\"%x-%x\"", now.UnixNano(), len(content)),
		},
	}
	w.b.mu.Lock()
	defer w.b.mu.Unlock()
	if prev := w.b.blobs[w.key]; prev != nil {
		entry.Attributes.CreateTime = prev.Attributes.CreateTime
	}
	w.b.blobs[w.key] = entry
	return nil
}

func (b *bucket) Copy(ctx context.Context, dstKey, srcKey string, opts *driver.CopyOptions) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if opts.BeforeCopy != nil {
		return opts.BeforeCopy(func(interface{}) bool { return false })
	}
	v := b.blobs[srcKey]
	if v == nil {
		return errNotFound
	}
	b.blobs[dstKey] = v
	return nil
}

func (b *bucket) Delete(ctx context.Context, key string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.blobs[key] == nil {
		return errNotFound
	}
	delete(b.blobs, key)
	return nil
}

func (b *bucket) SignedURL(ctx context.Context, key string, opts *driver.SignedURLOptions) (string, error) {
	return "", errNotImplemented
}
