package builder

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/e2/cli/util"
	"github.com/suborbital/e2core/e2/project"
)

var dockerImageForLang = map[string]string{
	"rust":           "suborbital/builder-rs",
	"swift":          "suborbital/builder-swift",
	"assemblyscript": "suborbital/builder-as",
	"tinygo":         "suborbital/builder-tinygo",
	"grain":          "--platform linux/amd64 suborbital/builder-gr",
	"typescript":     "suborbital/builder-js",
	"javascript":     "suborbital/builder-js",
	"wat":            "suborbital/builder-wat",
}

// BuildConfig is the configuration for a Builder.
type BuildConfig struct {
	JsToolchain   string
	CommandRunner util.CommandRunner
}

// DefaultBuildConfig is the default build configuration.
var DefaultBuildConfig = BuildConfig{
	JsToolchain:   "npm",
	CommandRunner: util.Command,
}

// Builder is capable of building Wasm modules from source.
type Builder struct {
	Context *project.Context
	Config  *BuildConfig

	results []BuildResult

	log util.FriendlyLogger
}

// BuildResult is the results of a build including the built module and logs.
type BuildResult struct {
	Succeeded bool
	OutputLog string
}

type Toolchain string

const (
	ToolchainNative = Toolchain("native")
	ToolchainDocker = Toolchain("docker")
)

// ForDirectory creates a Builder bound to a particular directory.
func ForDirectory(logger util.FriendlyLogger, config *BuildConfig, dir string) (*Builder, error) {
	ctx, err := project.ForDirectory(dir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to project.ForDirectory")
	}

	b := &Builder{
		Context: ctx,
		Config:  config,
		results: []BuildResult{},
		log:     logger,
	}

	return b, nil
}

func (b *Builder) BuildWithToolchain(tcn Toolchain) error {
	var err error

	b.results = []BuildResult{}

	// When building in Docker mode, just collect the langs we need to build, and then
	// launch the associated builder images which will do the building.
	dockerLangs := map[string]bool{}

	for _, mod := range b.Context.Modules {
		if !b.Context.ShouldBuildLang(mod.Module.Lang) {
			continue
		}

		if tcn == ToolchainNative {
			b.log.LogStart(fmt.Sprintf("building runnable: %s (%s)", mod.Name, mod.Module.Lang))

			result := &BuildResult{}

			if err := b.checkAndRunPreReqs(mod, result); err != nil {
				return errors.Wrap(err, "🚫 failed to checkAndRunPreReqs")
			}

			if flags, err := b.analyzeForCompilerFlags(mod); err != nil {
				return errors.Wrap(err, "🚫 failed to analyzeForCompilerFlags")
			} else if flags != "" {
				mod.CompilerFlags = flags
			}

			err = b.doNativeBuildForModule(mod, result)

			// Even if there was a failure, load the result into the builder
			// since the logs of the failed build are useful.
			b.results = append(b.results, *result)

			if err != nil {
				return errors.Wrapf(err, "🚫 failed to build %s", mod.Name)
			}

			fullWasmFilepath := filepath.Join(mod.Fullpath, fmt.Sprintf("%s.wasm", mod.Name))
			b.log.LogDone(fmt.Sprintf("%s was built -> %s", mod.Name, fullWasmFilepath))

		} else {
			dockerLangs[mod.Module.Lang] = true
		}
	}

	if tcn == ToolchainDocker {
		for lang := range dockerLangs {
			result, err := b.dockerBuildForLang(lang)
			if err != nil {
				return errors.Wrap(err, "failed to dockerBuildForDirectory")
			}

			b.results = append(b.results, *result)
		}
	}

	return nil
}

// Results returns build results for all of the modules built by this builder
// returns os.ErrNotExists if none have been built yet.
func (b *Builder) Results() ([]BuildResult, error) {
	if b.results == nil || len(b.results) == 0 {
		return nil, os.ErrNotExist
	}

	return b.results, nil
}

func (b *Builder) dockerBuildForLang(lang string) (*BuildResult, error) {
	img, err := ImageForLang(lang, b.Context.BuilderTag)
	if err != nil {
		return nil, errors.Wrap(err, "failed to ImageForLang")
	}

	result := &BuildResult{}

	outputLog, err := b.Config.CommandRunner.Run(fmt.Sprintf("docker run --rm --mount type=bind,source=%s,target=/root/runnable %s e2 build %s --native --langs %s", b.Context.MountPath, img, b.Context.RelDockerPath, lang))

	result.OutputLog = outputLog

	if err != nil {
		result.Succeeded = false
		return nil, errors.Wrap(err, "failed to Run docker command")
	}

	result.Succeeded = true

	return result, nil
}

// results and resulting file are loaded into the BuildResult pointer.
func (b *Builder) doNativeBuildForModule(mod project.ModuleDir, result *BuildResult) error {
	cmds, err := NativeBuildCommands(mod.Module.Lang)
	if err != nil {
		return errors.Wrap(err, "failed to NativeBuildCommands")
	}

	for _, cmd := range cmds {
		cmdTmpl, err := template.New("cmd").Parse(cmd)
		if err != nil {
			return errors.Wrap(err, "failed to Parse command template")
		}

		fullCmd := &strings.Builder{}
		if err := cmdTmpl.Execute(fullCmd, mod); err != nil {
			return errors.Wrap(err, "failed to Execute command template")
		}

		cmdString := strings.TrimSpace(fullCmd.String())

		// Even if the command fails, still load the output into the result object.
		outputLog, err := b.Config.CommandRunner.RunInDir(cmdString, mod.Fullpath)

		result.OutputLog += outputLog + "\n"

		if err != nil {
			result.Succeeded = false
			return errors.Wrap(err, "failed to RunInDir")
		}

		result.Succeeded = true
	}

	return nil
}

// ImageForLang returns the Docker image:tag builder for the given language.
func ImageForLang(lang, tag string) (string, error) {
	img, ok := dockerImageForLang[lang]
	if !ok {
		return "", fmt.Errorf("%s is an unsupported language", lang)
	}

	return fmt.Sprintf("%s:%s", img, tag), nil
}

func (b *Builder) checkAndRunPreReqs(runnable project.ModuleDir, result *BuildResult) error {
	preReqLangs, ok := PreRequisiteCommands[runtime.GOOS]
	if !ok {
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	preReqs, ok := preReqLangs[runnable.Module.Lang]
	if !ok {
		return fmt.Errorf("unsupported language: %s", runnable.Module.Lang)
	}

	for _, p := range preReqs {

		filepathVar := filepath.Join(runnable.Fullpath, p.File)

		if _, err := os.Stat(filepathVar); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				b.log.LogStart(fmt.Sprintf("missing %s, fixing...", p.File))

				fullCmd, err := p.GetCommand(*b.Config, runnable)
				if err != nil {
					return errors.Wrap(err, "prereq.GetCommand")
				}

				outputLog, err := b.Config.CommandRunner.RunInDir(fullCmd, runnable.Fullpath)
				if err != nil {
					return errors.Wrapf(err, "commandRunner.RunInDir: %s", fullCmd)
				}

				result.OutputLog += outputLog + "\n"

				b.log.LogDone("fixed!")
			}
		}
	}

	return nil
}

// analyzeForCompilerFlags looks at the Runnable and determines if any additional compiler flags are needed
// this is initially added to support AS-JSON in AssemblyScript with its need for the --transform flag.
func (b *Builder) analyzeForCompilerFlags(md project.ModuleDir) (string, error) {
	if md.Module.Lang == "assemblyscript" {
		packageJSONBytes, err := ioutil.ReadFile(filepath.Join(md.Fullpath, "package.json"))
		if err != nil {
			return "", errors.Wrap(err, "failed to ReadFile package.json")
		}

		if strings.Contains(string(packageJSONBytes), "json-as") {
			return "--transform ./node_modules/json-as/transform", nil
		}
	}

	return "", nil
}
