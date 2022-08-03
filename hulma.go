package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/nedpals/hulma/engines"
	"github.com/spf13/cobra"
)

func EngineNodeToTemplate(engineNode engines.Node) (Node, error) {
	engineChildren := engineNode.Children()
	nodeChildren := make([]Node, 0, len(engineChildren))

	for _, cn := range engineChildren {
		nodeChild, err := EngineNodeToTemplate(cn)
		if err != nil {
			return Node{}, err
		}
		nodeChildren = append(nodeChildren, nodeChild)
	}

	return Node{
		Type:     engineNode.Type(),
		Value:    engineNode.Value(),
		Children: nodeChildren,
	}, nil
}

var defaultEngines = engines.Engines{
	engines.RawJson{},
}

type FileTemplateLoader struct {
	Engines engines.Engines
	Store   TemplateStore
}

func (ftl *FileTemplateLoader) LoadFromEngine(fileName string, input string) error {
	foundEngine, templateName, err := ftl.Engines.MatchEngine(fileName)
	if err != nil {
		return err
	} else if _, ok := foundEngine.(engines.RawJson); ok {
		return ftl.Store.Set(input)
	}

	parsedNode, err := foundEngine.RenderString(input)
	if err != nil {
		return err
	}

	rootNode, err := EngineNodeToTemplate(parsedNode)
	if err != nil {
		return err
	}

	return ftl.Store.Add(&Template{
		Name:     templateName,
		blocks:   make(map[string][]Node),
		Version:  "1",
		RootNode: rootNode,
	})
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
	return ftl.LoadFromEngine(fullTemplatePath, string(rawTemplateData))
}

func (*FileTemplateLoader) Type() string {
	return "template_path"
}

type App struct {
	DefaultTemplateName string
	OutputPath          string
	Templates           TemplateStore
	Filters             map[string]FilterFunc
	Functions           map[string]FunctionFunc
}

func (app *App) SaveOutput(data string) error {
	fileInfo, err := os.Stat(app.OutputPath)
	if fileInfo != nil && fileInfo.IsDir() {
		return fmt.Errorf("path is a directory")
	}

	var file *os.File
	if err != nil && errors.Is(err, fs.ErrExist) {
		file, err = os.Open(app.OutputPath)
	} else {
		file, err = os.Create(app.OutputPath)
	}

	if err != nil {
		return err
	}

	defer file.Close()
	if _, err := io.WriteString(file, data); err != nil {
		return err
	}

	return nil
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

		if app.OutputPath == "stdout" {
			fmt.Println(data)
		} else if err := app.SaveOutput(data); err != nil {
			return fmt.Errorf("cannot save to %s: %s", app.OutputPath, err.Error())
		} else {
			fmt.Printf("saved to %s\n", app.OutputPath)
		}

		return nil
	},
}

func init() {
	fileTemplateLoader := &FileTemplateLoader{Store: app.Templates, Engines: defaultEngines}
	rootCmd.PersistentFlags().StringVarP(&app.OutputPath, "output", "o", "stdout", "Location where the rendered output will be stored.")
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
		// fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
