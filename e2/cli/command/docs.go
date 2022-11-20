package command

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/suborbital/e2core/e2/cli/util"
)

const (
	supportExt     = "go"
	docRegexStart  = `<!-- DO NOT REMOVE: START -->`
	docRegexEnd    = `<!-- DO NOT REMOVE: END -->`
	actionTemplate = `<!-- {{ Snippet "[[.ExampleKey]]" }} -->`
	docTemplate    = `
{{- .Action }}
{{ .RegexStart }}
{{range $i, $e := .Examples}}
{{ $e.Code }}
{{ end }}
{{ .RegexEnd -}}`
	cleanActionRegex   = `<!--(\s)*{{(.[^}>]*)[}$]}(\s)*-->`
	cleanExtraNewLines = `(\n)*` + docRegexEnd + `(\n)*`
	oldDocsRegex       = docRegexStart + `((.|\s)[^>]*)` + docRegexEnd
	actionPrefixRegex  = `^<!--(\s)*{{`
	actionPrefix       = `<!-- {{`
	actionSuffixRegex  = `}}(\s)*-->$`
	actionSuffix       = `}} -->`
)

type codeData struct {
	Action     string
	Ext        string
	Package    string
	Function   string
	RegexStart string
	RegexEnd   string
	Examples   []*exampleData
}

type exampleData struct {
	Suffix string
	Doc    string
	Code   string
	Output string
}

// DocsBuildCmd returns the docs build command.
func DocsBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build [dir] [--output]",
		Short: "Build code and documentation with inserted code snippets",
		Long:  `Build code and documentation with inserted code snippets`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			outputDir, err := cmd.Flags().GetString("output")
			if err != nil {
				return errors.Wrap(err, "failed to GetString")
			} else if outputDir == "" {
				outputDir = "."
			}

			if err := generateDocs(dir, outputDir); err != nil {
				return errors.Wrap(err, "failed to getUpdatedDocs")
			}

			return nil
		},
	}
	cmd.Flags().String("output", "", "output directory for generated documentation")

	return cmd
}

// DocsTestCmd returns the docs test command.
func DocsTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test [dir]",
		Short: "Test code and snippets inserts",
		Long:  `Test code and snippets inserts without generating new documentation`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			if cache, err := util.CacheDir("e2"); err != nil {
				return errors.Wrap(err, "failed to CacheDir")
			} else if err := generateDocs(dir, filepath.Join(cache, "docs")); err != nil {
				return errors.Wrap(err, "failed to getUpdatedDocs")
			}

			return nil
		},
	}

	return cmd
}

// generateDocs generates new docs with inserted example code snippets.
func generateDocs(dir string, outputDir string) error {
	files, err := getMarkdownCodeData(dir)
	if err != nil {
		return errors.Wrap(err, "failed to getMarkdownCodeData")
	}

	// Generate docs with inserted code snippets.
	for fileName, fileContent := range files["md"] {
		// Clean template actions.
		r := regexp.MustCompile(cleanActionRegex)
		fileContent = r.ReplaceAllStringFunc(fileContent,
			func(src string) string {
				// remove extra whitespaces from template actions.
				r := regexp.MustCompile(actionSuffixRegex)
				if r.MatchString(src) {
					src = r.ReplaceAllLiteralString(src, actionSuffix)
				}

				r = regexp.MustCompile(actionPrefixRegex)
				if r.MatchString(src) {
					src = r.ReplaceAllLiteralString(src, actionPrefix)
				}

				return src
			},
		)
		r = regexp.MustCompile(oldDocsRegex)
		// Remove old documentation examples.
		fileContent = r.ReplaceAllLiteralString(fileContent, "")
		// Generate new documentation examples.
		filePath := filepath.Join(outputDir, fileName)
		fileDir := filepath.Dir(filePath)
		if info, err := os.Stat(fileDir); os.IsNotExist(err) {
			if err := os.MkdirAll(fileDir, os.ModePerm); err != nil {
				return errors.Wrap(err, "failed to MkdirAll")
			}
		} else if err != nil {
			return errors.Wrap(err, "failed to Stat")
		} else if !info.IsDir() {
			return errors.New(fmt.Sprintf("%s is not a directory", fileDir))
		}

		file, err := os.Create(filePath)
		if err != nil {
			return errors.Wrap(err, "failed to Create")
		}

		// Main template that replaces user template actions `{ Snippet ... }` with their associated code snippets
		//
		// User template actions types:
		// `{{ Snippet "greetings" }}` => all package examples are inserted from package `greetings`
		// `{{ Snippet "greetings:doNotDoThis" }}` => only package example `doNotDoThis` is  inserted from package `greeting`
		// {{ Snippet "greetings/Hello" }} => all function  `Hello` examples are inserted from package `greetings`
		// {{ Snippet "greetings/Hello:doThis" }} => only function  `Hello` example `doThis` is  inserted from package `greetings`
		// Nonexistent examples for packages or functions will cause `doc` cmd to fail.
		tmpl, err := template.New(fileName).Funcs(template.FuncMap{
			"Snippet": func(exampleKey string) (string, error) {
				// Generate user template action for reinsertion.
				tmpl, err := template.New("NestedTemplateAction").Delims(`[[`, `]]`).Parse(actionTemplate)
				if err != nil {
					return "", err
				}

				var buffer bytes.Buffer
				if err := tmpl.Execute(&buffer, map[string]string{"ExampleKey": exampleKey}); err != nil {
					return "", err
				}

				// Check example key structure.
				var pkgName, funcName string
				keys := strings.Split(exampleKey, "/")
				if len(keys) == 1 {
					pkgName = keys[0]
				} else if len(keys) == 2 {
					pkgName, funcName = keys[0], keys[1]
				} else {
					return "", errors.New("`Snippet` expects a non-empty string key `packageName/funcName:Example` or 'packageName:Example', where 'funcName' and 'Example' are both optional")
				}

				// Check if supportExt files, `go` files, where found.
				examples, ok := files[supportExt]
				if !ok {
					return "", errors.New(fmt.Sprintf("Failed to `Snippet`, no files found with ext `%s`", supportExt))
				}

				// Check if example exists.
				example, ok := examples[exampleKey]
				if !ok {
					if pkgName == "" {
						return "", errors.New("`Snippet` expects a non-empty string package name")
					} else if funcName == "" {
						return "", errors.New(fmt.Sprintf("%s\nExamples for the package `%s` do not exist", buffer.String(), pkgName))
					}

					return "", errors.New(fmt.Sprintf("%s\nExamples for the function `%s` in the package `%s` do not exist", buffer.String(), funcName, pkgName))
				}

				return example, nil
			},
		}).Delims(actionPrefix, actionSuffix).Parse(fileContent)
		var buffer bytes.Buffer
		if err != nil {
			return errors.Wrap(err, "failed to Parse")
		} else if err = tmpl.Execute(&buffer, ""); err != nil {
			// Reset doc to its previous state.
			if errWrite := ioutil.WriteFile(filePath, []byte(fileContent), os.ModePerm); errWrite != nil {
				return errors.Wrap(errWrite, "failed to Write")
			}

			return errors.Wrap(err, "failed to Execute")
		}

		// weird behavior of text.templates, it adds new lines regardless of action delimiters
		// This ReplaceAll compensates for it.
		r = regexp.MustCompile(cleanExtraNewLines)
		fileContent = r.ReplaceAllLiteralString(buffer.String(), fmt.Sprintf("\n%s\n", docRegexEnd))
		_, err = file.WriteString(fileContent)
		if err != nil {
			return errors.Wrap(err, "failed to WriteString")
		}
	}

	return nil
}

// getMarkdownCodeData returns a mapping of markdown texts and go example code snippets.
func getMarkdownCodeData(dir string) (map[string]map[string]string, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, errors.Wrap(err, fmt.Sprintf("dir %s does not exist", dir))
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to Stat")
	}

	goSnippets, err := getGoSnippets(dir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to getGoSnippets")
	}

	mdTextss, err := getMarkdownTexts(dir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to getMarkdownTexts")
	}

	return map[string]map[string]string{"md": mdTextss, "go": goSnippets}, nil
}

// getMarkdownTexts returns discovered markdown texts.
func getMarkdownTexts(dir string) (map[string]string, error) {
	mdTexts := make(map[string]string)
	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.Wrap(err, "failed to Walk")
			} else if info.IsDir() {
				return nil
			} else if !strings.HasSuffix(path, ".md") {
				return nil
			}

			util.LogInfo(fmt.Sprintf("processing doc '%s'", path))
			pwd, err := os.Getwd()
			if err != nil {
				return errors.Wrap(err, "failed to Getwd")
			}

			data, err := ioutil.ReadFile(path)
			if err != nil {
				return errors.Wrap(err, "failed to ReadFile")
			}

			keyFile := filepath.Join(strings.Replace(filepath.Dir(path), pwd, "", 1), info.Name())
			mdTexts[keyFile] = string(data)

			return nil
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Walk")
	}

	return mdTexts, nil
}

// getGoSnippets returns discovered go example code snippets.
func getGoSnippets(dir string) (map[string]string, error) {
	goSnippets := make(map[string]string)
	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.Wrap(err, "failed to Walk")
			} else if !info.IsDir() {
				return nil
			}

			path, err = filepath.Abs(path)
			if err != nil {
				return errors.Wrap(err, "failed to Abs")
			}

			fset := token.NewFileSet()
			pkgs, err := parser.ParseDir(fset, path, nil, parser.ParseComments)
			if err != nil {
				return errors.Wrap(err, "failed to ParseDir")
			}

			// Get and structure package and function examples from ast to text.
			for pkgName, pkg := range pkgs {
				files := []*ast.File{}
				for _, file := range pkg.Files {
					files = append(files, file)
				}

				// Get special package and function asts that contain example metadata.
				pkgDoc, err := doc.NewFromFiles(fset, files, filepath.Join(path, pkgName), doc.AllDecls)
				if err != nil {
					return errors.Wrap(err, "failed to NewFromFiles")
				}

				// Process examples associated with the package.
				pkgData := codeData{
					Ext:        "go",
					RegexStart: docRegexStart,
					RegexEnd:   docRegexEnd,
					Package:    pkgName,
				}
				for _, example := range pkgDoc.Examples {
					exampleKey := getExampleKey(pkgName, "", example.Suffix)
					text, err := pkgData.getCodeText(exampleKey, []*doc.Example{example}, fset)
					if err != nil {
						return errors.Wrap(err, "failed to getCodeText")
					}

					goSnippets[exampleKey] = text
				}

				// Add all examples option for package.
				exampleKey := getExampleKey(pkgName, "", "")
				text, err := pkgData.getCodeText(exampleKey, pkgDoc.Examples, fset)
				if err != nil {
					return errors.Wrap(err, "failed to getCodeText")
				}

				goSnippets[exampleKey] = text
				// Process examples associated with this function or method.
				for _, funcNode := range pkgDoc.Funcs {
					if len(funcNode.Examples) == 0 {
						continue
					}

					funcData := codeData{
						Ext:        "go",
						RegexStart: docRegexStart,
						RegexEnd:   docRegexEnd,
						Package:    pkgName,
						Function:   funcNode.Name,
					}
					for _, example := range funcNode.Examples {
						exampleKey := getExampleKey(pkgName, example.Name, example.Suffix)
						text, err := funcData.getCodeText(exampleKey, []*doc.Example{example}, fset)
						if err != nil {
							return errors.Wrap(err, "failed to getCodeText")
						}

						goSnippets[exampleKey] = text
					}

					// Add all examples option for function.
					exampleKey := getExampleKey(pkgName, funcNode.Name, "")
					text, err := funcData.getCodeText(exampleKey, funcNode.Examples, fset)
					if err != nil {
						return errors.Wrap(err, "failed to getCodeText")
					}

					goSnippets[exampleKey] = text
				}
			}

			return nil
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to Walk")
	}

	return goSnippets, nil
}

// getExampleKey returns keys associated to examples.
func getExampleKey(pkgName, funcName, exampleSuffix string) string {
	key := pkgName
	if funcName != "" {
		key = fmt.Sprintf("%s/%s", pkgName, funcName)
		if exampleSuffix != "" {
			key = strings.Replace(key, fmt.Sprintf("_%s", exampleSuffix), fmt.Sprintf(":%s", exampleSuffix), 1)
		}
	} else if exampleSuffix != "" {
		key = fmt.Sprintf("%s:%s", key, exampleSuffix)
	}

	return key
}

// getCodeText returns code snippets generated from example asts.
func (c *codeData) getCodeText(exampleKey string, examples []*doc.Example, fset *token.FileSet) (string, error) {
	err := c.getExampleData(examples, fset)
	if err != nil {
		return "", errors.Wrap(err, "failed to getExampleData")
	}

	// Generate user template action for reinsertion.
	tmpl, err := template.New("NestedTemplateAction").Delims(`[[`, `]]`).Parse(actionTemplate)
	if err != nil {
		return "", errors.Wrap(err, "failed to Parse")
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, map[string]string{"ExampleKey": exampleKey}); err != nil {
		return "", errors.Wrap(err, "failed to Execute")
	}

	c.Action = buffer.String()
	// Generate code snippets based on `docTemplate` templates.
	tmpl, err = template.New("CodeText").Parse(docTemplate)
	if err != nil {
		return "", errors.Wrap(err, "failed to Parse")
	}

	buffer.Reset()
	if err := tmpl.Execute(&buffer, c); err != nil {
		return "", errors.Wrap(err, "failed to Execute")
	}

	return buffer.String(), nil
}

// getExampleData loads example asts to codeData.
func (c *codeData) getExampleData(examples []*doc.Example, fset *token.FileSet) error {
	c.Examples = nil
	for i, example := range examples {
		// Get example code snippets ast.
		var buffer bytes.Buffer
		switch n := example.Code.(type) {
		case *ast.BlockStmt:
			for _, n := range n.List {
				if err := printer.Fprint(&buffer, fset, n); err != nil {
					return errors.Wrap(err, "failed to Fprint")
				}

				fmt.Fprint(&buffer, "\n")
			}
		}

		// Get example code snippets metadata.
		c.Examples = append(c.Examples,
			&exampleData{
				Suffix: example.Suffix,
				Doc:    example.Doc,
				Code:   fmt.Sprintf("```go\n%s```", buffer.String()),
			},
		)
		if example.Output != "" {
			c.Examples[i].Output = fmt.Sprintf("`%s`", strings.TrimSpace(example.Output))
		}
	}

	return nil
}
