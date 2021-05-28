package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/suborbital/atmo/bundle"
	"github.com/suborbital/atmo/directive"
)

func main() {
	files := []os.File{}
	for _, filename := range []string{"fetch/fetch.wasm", "log/log.wasm", "hello-echo/hello-echo.wasm"} {
		path := filepath.Join("./", "rwasm", "testdata", filename)

		file, err := os.Open(path)
		if err != nil {
			log.Fatal("failed to open file", err)
		}

		files = append(files, *file)
	}

	staticFiles := map[string]os.File{}
	for _, filename := range []string{"go.mod", "go.sum", "Makefile"} {
		path := filepath.Join("./", filename)

		file, err := os.Open(path)
		if err != nil {
			log.Fatal("failed to open file", err)
		}

		staticFiles[path] = *file
	}

	directive := &directive.Directive{
		Identifier:  "dev.suborbital.appname",
		AppVersion:  "v0.1.1",
		AtmoVersion: "v0.0.6",
		Runnables: []directive.Runnable{
			{
				Name:      "fetch",
				Namespace: "default",
			},
			{
				Name:      "log",
				Namespace: "default",
			},
			{
				Name:      "hello-echo",
				Namespace: "default",
			},
		},
		Handlers: []directive.Handler{
			{
				Input: directive.Input{
					Type:     directive.InputTypeRequest,
					Method:   "GET",
					Resource: "/api/v1/user",
				},
				Steps: []directive.Executable{
					{
						Group: []directive.CallableFn{
							{
								Fn: "fetch",
								As: "ghData",
							},
							{
								Fn: "log",
								OnErr: &directive.FnOnErr{
									Code: map[int]string{
										404: "continue",
									},
									Other: "return",
								},
							},
						},
					},
					{
						CallableFn: directive.CallableFn{
							Fn: "hello-echo",
							With: map[string]string{
								"data": "ghData",
							},
							OnErr: &directive.FnOnErr{
								Any: "return",
							},
						},
					},
				},
				Response: "ghData",
			},
		},
		Schedules: []directive.Schedule{
			{
				Name: "user-purger",
				Every: directive.ScheduleEvery{
					Minutes: 5,
				},
				Steps: []directive.Executable{
					{
						CallableFn: directive.CallableFn{
							Fn: "hello-echo",
						},
					},
				},
			},
		},
	}

	if err := directive.Validate(); err != nil {
		log.Fatal("failed to validate directive: ", err)
	}

	directiveBytes, err := directive.Marshal()
	if err != nil {
		log.Fatal("failed to Marshal directive")
	}

	if err := bundle.Write(directiveBytes, files, staticFiles, "./runnables.wasm.zip"); err != nil {
		log.Fatal("failed to WriteBundle", err)
	}

	bdl, err := bundle.Read("./runnables.wasm.zip")
	if err != nil {
		log.Fatal("failed to re-read bundle:", err)
	}

	file, err := bdl.StaticFile("go.mod")
	if err != nil {
		log.Fatal("failed to StaticFile:", err)
	}

	fmt.Println(string(file))

	fmt.Println("done ✨")
}
