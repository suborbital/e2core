package builder

import (
	"fmt"
	"runtime"
)

// NativeBuildCommands returns the native build commands needed to build a Runnable of a particular language.
func NativeBuildCommands(lang string) ([]string, error) {
	os := runtime.GOOS

	cmds, exists := nativeCommandsForLang[os][lang]
	if !exists {
		return nil, fmt.Errorf("unable to build %s Runnables natively", lang)
	}

	return cmds, nil
}

var nativeCommandsForLang = map[string]map[string][]string{
	"darwin": {
		"rust": {
			"cargo vendor && cargo build --target wasm32-wasi --lib --release",
			"cp target/wasm32-wasi/release/{{ .UnderscoreName }}.wasm ./{{ .Name }}.wasm",
		},
		"swift": {
			"xcrun --toolchain swiftwasm swift build --triple wasm32-unknown-wasi -Xlinker --allow-undefined -Xlinker --export=allocate -Xlinker --export=deallocate -Xlinker --export=run_e -Xlinker --export=init",
			"cp .build/debug/{{ .Name }}.wasm .",
		},
		"assemblyscript": {
			"npm run asbuild",
		},
		"tinygo": {
			"go get -d",
			"go mod tidy",
			"tinygo build -o {{ .Name }}.wasm -target wasi .",
		},
		"grain": {
			"grain compile index.gr -I _lib -o {{ .Name }}.wasm",
		},
		"typescript": {
			"npm run build",
		},
		"javascript": {
			"npm run build",
		},
		"wat": {
			"wat2wasm lib.wat -o {{ .Name }}.wasm",
		},
	},
	"linux": {
		"rust": {
			"cargo vendor && cargo build --target wasm32-wasi --lib --release",
			"cp target/wasm32-wasi/release/{{ .UnderscoreName }}.wasm ./{{ .Name }}.wasm",
		},
		"swift": {
			"swift build --triple wasm32-unknown-wasi -Xlinker --allow-undefined -Xlinker --export=allocate -Xlinker --export=deallocate -Xlinker --export=run_e -Xlinker --export=init",
			"cp .build/debug/{{ .Name }}.wasm .",
		},
		"assemblyscript": {
			"chmod -R 777 ./",
			"chmod +x ./node_modules/assemblyscript/bin/asc",
			"./node_modules/assemblyscript/bin/asc src/index.ts --target release --use abort=src/index/abort {{ .CompilerFlags }}",
		},
		"tinygo": {
			"go get -d",
			"go mod tidy",
			"tinygo build -o {{ .Name }}.wasm -target wasi .",
		},
		"grain": {
			"grain compile index.gr -I _lib -o {{ .Name }}.wasm",
		},
		"typescript": {
			"npm run build",
		},
		"javascript": {
			"npm run build",
		},
		"wat": {
			"wat2wasm lib.wat -o {{ .Name }}.wasm",
		},
	},
}
