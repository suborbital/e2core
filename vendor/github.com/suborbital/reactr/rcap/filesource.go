package rcap

import "github.com/pkg/errors"

var (
	ErrFileFuncNotSet = errors.New("file func not set")
)

// StaticFileFunc is a function that returns the contents of a requested file
type StaticFileFunc func(string) ([]byte, error)

// FileConfig is configuration for the File capability
type FileConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`

	FileFunc StaticFileFunc `json:"-" yaml:"-"`
}

// FileCapability gives runnables access to various kinds of files
type FileCapability interface {
	GetStatic(filename string) ([]byte, error)
}

// defaultFileSource grants access to files
type defaultFileSource struct {
	config         FileConfig
	staticFileFunc StaticFileFunc
}

func DefaultFileSource(config FileConfig) FileCapability {
	d := &defaultFileSource{
		config:         config,
		staticFileFunc: config.FileFunc,
	}

	return d
}

// GetStatic returns a static file
func (d *defaultFileSource) GetStatic(filename string) ([]byte, error) {
	if !d.config.Enabled {
		return nil, ErrCapabilityNotEnabled
	}

	if d.staticFileFunc == nil {
		return nil, ErrFileFuncNotSet
	}

	return d.staticFileFunc(filename)
}
