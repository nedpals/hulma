package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type FileTemplateLoader struct {
	Store TemplateStore
}

func (*FileTemplateLoader) String() string {
	return ""
}

func (ftl *FileTemplateLoader) Set(templatePath string) error {
	fullTemplatePath, _ := filepath.Abs(templatePath)
	rawTemplateData, err := os.ReadFile(fullTemplatePath)
	if err != nil {
		return err
	}
	return ftl.Store.Set(string(rawTemplateData))
}

func (*FileTemplateLoader) Type() string {
	return "template_path"
}

type App struct {
	DefaultTemplateName string
	Templates           TemplateStore
	Filters             map[string]FilterFunc
	Functions           map[string]FunctionFunc
}

func (app *App) Render(templateName string, varData map[string]any) (string, error) {
	data := TemplateData{
		Context: ContextData{
			Data: varData,
		},
		Filters:   app.Filters,
		Functions: app.Functions,
		Templates: app.Templates,
	}
	writer := &bytes.Buffer{}
	if err := app.Templates.Render(templateName, data, &simpleRenderer{writer: writer}); err != nil {
		return "", err
	}
	return writer.String(), nil
}

func (rnd *App) RegisterFilter(name string, filterFn FilterFunc) {
	rnd.Filters[name] = filterFn
}

func (rnd *App) RegisterFunction(name string, fnFn FunctionFunc) {
	rnd.Functions[name] = fnFn
}

var dataPath string
var app = &App{
	DefaultTemplateName: "default",
	Templates:           TemplateStore{},
	Filters:             map[string]FilterFunc{},
	Functions:           map[string]FunctionFunc{},
}

var rootCmd = &cobra.Command{
	Use:   "hulma",
	Short: "Hulma is an experimental template compiler.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fullDataPath, _ := filepath.Abs(dataPath)
		rawContextData, err := os.ReadFile(fullDataPath)
		if err != nil {
			return err
		}

		contextData := make(map[string]any)
		if err := json.Unmarshal(rawContextData, &contextData); err != nil {
			return err
		}

		data, err := app.Render(app.DefaultTemplateName, contextData)
		if err != nil {
			return err
		}

		fmt.Println(data)
		return nil
	},
}

func init() {
	fileTemplateLoader := &FileTemplateLoader{app.Templates}
	rootCmd.PersistentFlags().StringVar(&app.DefaultTemplateName, "name", "default", "Name of the template to be rendered.")
	rootCmd.PersistentFlags().Var(fileTemplateLoader, "template", "Path to the template.json file.")
	rootCmd.PersistentFlags().Var(&app.Templates, "templateData", "JSON data of the template.")
	rootCmd.PersistentFlags().StringVar(&dataPath, "data", "", "Path to the data.json file.")
}

func main() {
	app.RegisterFilter("upper", func(value any) (any, error) {
		if valueStr, ok := value.(string); ok {
			return strings.ToUpper(valueStr), nil
		}
		return value, nil
	})

	app.RegisterFunction("foo", func(arguments any) (any, error) {
		return "foo", nil
	})

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
