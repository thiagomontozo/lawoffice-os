package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type ObjectStorage interface {
	Put(context.Context, string, io.Reader) error
	Open(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
	Health(context.Context) error
}
type Local struct{ root string }

func NewLocal(root string) (*Local, error) {
	abs, e := filepath.Abs(root)
	if e != nil {
		return nil, e
	}
	if e = os.MkdirAll(abs, 0o750); e != nil {
		return nil, e
	}
	return &Local{root: filepath.Clean(abs)}, nil
}
func (s *Local) path(key string) (string, error) {
	if key == "" || filepath.IsAbs(key) || strings.ContainsAny(key, `/\`) || filepath.Base(key) != key {
		return "", errors.New("invalid storage key")
	}
	p := filepath.Join(s.root, key)
	rel, e := filepath.Rel(s.root, p)
	if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("storage key escapes root")
	}
	return p, nil
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (c contextReader) Read(p []byte) (int, error) {
	select {
	case <-c.ctx.Done():
		return 0, c.ctx.Err()
	default:
		return c.r.Read(p)
	}
}
func (s *Local) Put(ctx context.Context, key string, r io.Reader) error {
	p, e := s.path(key)
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(s.root, ".upload-*")
	if e != nil {
		return e
	}
	temp := f.Name()
	defer os.Remove(temp)
	if _, e = io.Copy(f, contextReader{ctx, r}); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(temp, p)
}
func (s *Local) Open(_ context.Context, key string) (io.ReadCloser, error) {
	p, e := s.path(key)
	if e != nil {
		return nil, e
	}
	return os.Open(p)
}
func (s *Local) Delete(_ context.Context, key string) error {
	p, e := s.path(key)
	if e != nil {
		return e
	}
	e = os.Remove(p)
	if errors.Is(e, os.ErrNotExist) {
		return nil
	}
	return e
}
func (s *Local) Health(_ context.Context) error {
	i, e := os.Stat(s.root)
	if e != nil {
		return e
	}
	if !i.IsDir() {
		return errors.New("storage root is not a directory")
	}
	return nil
}
