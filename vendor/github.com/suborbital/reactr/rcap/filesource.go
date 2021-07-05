package rcap

import "github.com/pkg/errors"

var (
	ErrFileFuncNotSet = errors.New("file func not set")
)

// StaticFileFunc is a function that returns the contents of a requested file
type StaticFileFunc func(string) ([]byte, error)

// FileSource gives runnables access to various kinds of files
type FileSource interface {
	GetStatic(filename string) ([]byte, error)
}

// defaultFileSource grants access to files
type defaultFileSource struct {
	staticFileFunc StaticFileFunc
}

func DefaultFileSource(staticFileFunc StaticFileFunc) FileSource {
	d := &defaultFileSource{
		staticFileFunc: staticFileFunc,
	}

	return d
}

// GetStatic returns a static file
func (d *defaultFileSource) GetStatic(filename string) ([]byte, error) {
	if d.staticFileFunc == nil {
		return nil, ErrFileFuncNotSet
	}

	return d.staticFileFunc(filename)
}
