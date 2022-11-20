package project

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/suborbital/e2core/e2/cli/release"
	"github.com/suborbital/e2core/e2/cli/util"
	"github.com/suborbital/systemspec/tenant"
)

// validLangs are the available languages.
var validLangs = map[string]struct{}{
	"rust":           {},
	"swift":          {},
	"assemblyscript": {},
	"tinygo":         {},
	"grain":          {},
	"typescript":     {},
	"javascript":     {},
	"wat":            {},
}

// Context describes the context under which the tool is being run.
type Context struct {
	Cwd            string
	CwdIsRunnable  bool
	Modules        []ModuleDir
	Bundle         BundleRef
	TenantConfig   *tenant.Config
	RuntimeVersion string
	Langs          []string
	MountPath      string
	RelDockerPath  string
	BuilderTag     string
}

// ModuleDir represents a directory containing a Runnable.
type ModuleDir struct {
	Name           string
	UnderscoreName string
	Fullpath       string
	Module         *tenant.Module
	CompilerFlags  string
}

// BundleRef contains information about a bundle in the current context.
type BundleRef struct {
	Exists   bool
	Fullpath string
}

// ForDirectory returns the build context for the provided working directory.
func ForDirectory(dir string) (*Context, error) {
	fullDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get Abs path")
	}

	modules, cwdIsRunnable, err := getModuleDirs(fullDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to getRunnableDirs")
	}

	bundle, err := bundleTargetPath(fullDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to bundleIfExists")
	}

	config, err := readTenantConfig(fullDir)
	if err != nil {
		if !os.IsNotExist(errors.Cause(err)) {
			return nil, errors.Wrap(err, "failed to readDirectiveFile")
		}
	}

	queries, err := readQueriesFile(dir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to readQueriesFile")
	} else if len(queries) > 0 {
		config.DefaultNamespace.Queries = queries
	}

	connections, err := readConnectionsFile(dir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to readConnectionsFile")
	} else if len(connections) > 0 {
		config.DefaultNamespace.Connections = connections
	}

	bctx := &Context{
		Cwd:           fullDir,
		CwdIsRunnable: cwdIsRunnable,
		Modules:       modules,
		Bundle:        *bundle,
		TenantConfig:  config,
		Langs:         []string{},
		MountPath:     fullDir,
		RelDockerPath: ".",
		BuilderTag:    fmt.Sprintf("v%s", release.E2CLIDotVersion),
	}

	return bctx, nil
}

// ModuleExists returns true if the context contains a module with name <name>.
func (b *Context) ModuleExists(name string) bool {
	for _, r := range b.Modules {
		if r.Name == name {
			return true
		}
	}

	return false
}

// ShouldBuildLang returns true if the provided language is safe-listed for building.
func (b *Context) ShouldBuildLang(lang string) bool {
	if len(b.Langs) == 0 {
		return true
	}

	for _, l := range b.Langs {
		if l == lang {
			return true
		}
	}

	return false
}

func (b *Context) ModuleFiles() ([]os.File, error) {
	modules := []os.File{}

	for _, r := range b.Modules {
		wasmPath := filepath.Join(r.Fullpath, fmt.Sprintf("%s.wasm", r.Name))

		file, err := os.Open(wasmPath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to Open module file %s", wasmPath)
		}

		modules = append(modules, *file)
	}

	return modules, nil
}

// HasDockerfile returns a nil error if the project's Dockerfile exists.
func (b *Context) HasDockerfile() error {
	dockerfilePath := filepath.Join(b.Cwd, "Dockerfile")

	if _, err := os.Stat(dockerfilePath); err != nil {
		return errors.Wrap(err, "failed to Stat Dockerfile")
	}

	return nil
}

// WasmFile returns a file object for the .wasm file. It is the caller's responsibility to close the file.
func (m *ModuleDir) WasmFile() (io.ReadCloser, error) {
	modulePath := filepath.Join(m.Fullpath, fmt.Sprintf("%s.wasm", m.Name))

	wasmFile, err := os.Open(modulePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to Open %s", modulePath)
	}

	return wasmFile, nil
}

// HasWasmFile returns a nil error if the module's .wasm file exists.
func (m *ModuleDir) HasWasmFile() error {
	modulePath := filepath.Join(m.Fullpath, fmt.Sprintf("%s.wasm", m.Name))

	if _, err := os.Stat(modulePath); err != nil {
		return errors.Wrapf(err, "failed to Stat %s", modulePath)
	}

	return nil
}

func getModuleDirs(cwd string) ([]ModuleDir, bool, error) {
	modules := []ModuleDir{}

	// Go through all of the dirs in the current dir.
	topLvlFiles, err := ioutil.ReadDir(cwd)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to list directory")
	}

	// Check to see if we're running from within a Runnable directory
	// and return true if so.
	moduleDir, err := getModuleFromFiles(cwd, topLvlFiles)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to getRunnableFromFiles")
	} else if moduleDir != nil {
		modules = append(modules, *moduleDir)
		return modules, true, nil
	}

	for _, tf := range topLvlFiles {
		if !tf.IsDir() {
			continue
		}

		dirPath := filepath.Join(cwd, tf.Name())

		// Determine if a .module file exists in that dir.
		innerFiles, err := ioutil.ReadDir(dirPath)
		if err != nil {
			util.LogWarn(fmt.Sprintf("couldn't read files in %v", dirPath))
			continue
		}

		moduleDir, err := getModuleFromFiles(dirPath, innerFiles)
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to getRunnableFromFiles")
		} else if moduleDir == nil {
			continue
		}

		modules = append(modules, *moduleDir)
	}

	return modules, false, nil
}

// ContainsModuleYaml finds any .module file in a list of files.
func ContainsModuleYaml(files []os.FileInfo) (string, bool) {
	for _, f := range files {
		if strings.HasPrefix(f.Name(), ".module.") {
			return f.Name(), true
		}
	}

	return "", false
}

// IsValidLang returns true if a language is valid.
func IsValidLang(lang string) bool {
	_, exists := validLangs[lang]

	return exists
}

func getModuleFromFiles(wd string, files []os.FileInfo) (*ModuleDir, error) {
	filename, exists := ContainsModuleYaml(files)
	if !exists {
		return nil, nil
	}

	moduleBytes, err := ioutil.ReadFile(filepath.Join(wd, filename))
	if err != nil {
		return nil, errors.Wrap(err, "failed to ReadFile .module yaml")
	}

	module := &tenant.Module{}
	if err := yaml.Unmarshal(moduleBytes, &module); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal .module yaml")
	}

	if module.Name == "" {
		module.Name = filepath.Base(wd)
	}

	if module.Namespace == "" {
		module.Namespace = "default"
	}

	if ok := IsValidLang(module.Lang); !ok {
		return nil, fmt.Errorf("(%s) %s is not a valid lang", module.Name, module.Lang)
	}

	absolutePath, err := filepath.Abs(wd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get Abs filepath")
	}

	moduleDir := &ModuleDir{
		Name:           module.Name,
		UnderscoreName: strings.Replace(module.Name, "-", "_", -1),
		Fullpath:       absolutePath,
		Module:         module,
	}

	return moduleDir, nil
}

func bundleTargetPath(cwd string) (*BundleRef, error) {
	path := filepath.Join(cwd, "modules.wasm.zip")

	b := &BundleRef{
		Fullpath: path,
		Exists:   false,
	}

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return b, nil
		} else {
			return nil, err
		}
	}

	b.Exists = true

	return b, nil
}
