package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cobra"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

var (
	NODE_TYPE_SOURCE   = "node__source"
	NODE_TYPE_CONTENT  = "node__content"
	NODE_TYPE_DISPLAY  = "node__display"
	NODE_TYPE_VARIABLE = "node__variable"
	NODE_TYPE_FILTER   = "node__filter"
	NODE_TYPE_INCLUDE  = "node__include"
	NODE_TYPE_BLOCK    = "node__block"
	NODE_TYPE_YIELD    = "node__yield"
	NODE_TYPE_LOOP     = "node__loop"
	NODE_TYPE_ASSIGN   = "node__assign"
)

type ContextData map[string]interface{}

func (cd *ContextData) DecodeJSON(rawContextData []byte) error {
	err := json.Unmarshal(rawContextData, cd)
	return err
}

type FilterFunc func(context ContextData, nodes ...Node) (interface{}, error)

type Node struct {
	Type     string
	Value    string
	Children []Node
}

func (node Node) Evaluate(context ContextData, renderer *Renderer) (interface{}, error) {
	switch node.Type {
	case NODE_TYPE_CONTENT:
		return node.Value, nil
	case NODE_TYPE_VARIABLE:
		gotValue, varExists := context[node.Value]
		if !varExists {
			return nil, fmt.Errorf("variable `%s` does not exist", node.Value)
		}
		return gotValue, nil
	case NODE_TYPE_FILTER:
		gotFilter, filterExists := renderer.Filters[node.Value]
		if !filterExists {
			return nil, fmt.Errorf("filter `%s` does not exist", node.Value)
		}
		return gotFilter(context, node.Children...)
	default:
		return nil, fmt.Errorf("[eval] unsupported node type `%s`", node.Type)
	}
}

func (node Node) scanBlocks(parentBlockName string, context ContextData, renderer *Renderer) error {
	for _, cn := range node.Children {
		if cn.Type != NODE_TYPE_BLOCK {
			continue
		} else if cn.Value == parentBlockName {
			return fmt.Errorf("`%s` block should not be recursive", cn.Value)
		} else {
			cn.scanBlocks(cn.Value, context, renderer)
			renderer.Blocks[cn.Value] = cn.Children
		}
	}
	return nil
}

func (node Node) render(writer io.Writer, context ContextData, renderer *Renderer) error {
	switch node.Type {
	case NODE_TYPE_CONTENT:
		writer.Write([]byte(node.Value))
	case NODE_TYPE_DISPLAY:
		if len(node.Children) == 0 {
			return fmt.Errorf("nothing to print")
		}

		child := node.Children[0]
		if child.Type == NODE_TYPE_BLOCK {
			return child.render(writer, context, renderer)
		}

		gotValue, err := child.Evaluate(context, renderer)
		if err != nil {
			return err
		}

		writer.Write([]byte(fmt.Sprintf("%s", gotValue)))
	case NODE_TYPE_SOURCE:
		if err := node.scanBlocks("", context, renderer); err != nil {
			return err
		}

		for _, cn := range node.Children {
			if cn.Type == NODE_TYPE_BLOCK {
				continue
			}

			if err := cn.render(writer, context, renderer); err != nil {
				return err
			}
		}
	case NODE_TYPE_INCLUDE:
		gotTemplate, templateExists := renderer.Templates[node.Value]
		if !templateExists {
			return fmt.Errorf("template `%s` does not exist", gotTemplate.Name)
		}
		return gotTemplate.Render(writer, context, renderer)
	case NODE_TYPE_YIELD:
		gotBlock, blockExists := renderer.Blocks[node.Value]
		if !blockExists {
			// use default content
			for _, cn := range node.Children {
				if err := cn.render(writer, context, renderer); err != nil {
					return err
				}
			}
			return nil
		} else {
			for _, cn := range gotBlock {
				if err := cn.render(writer, context, renderer); err != nil {
					return err
				}
			}
		}
	case NODE_TYPE_LOOP:
		// for loop dissect
		// index 0 -

		// copy old context data
		oldContext := make(ContextData)
		for k, v := range context {
			oldContext[k] = v
		}

		// make a new special variable

		// context["$$i"] = len()

		newlyAssigned := make([]string, 10)
		for _, cn := range node.Children {
			if err := cn.render(writer, context, renderer); err != nil {
				return err
			}
		}

		for _, v := range newlyAssigned {
			delete(context, v)
		}

		for k, v := range oldContext {
			context[k] = v
		}
	default:
		return fmt.Errorf("[render] unsupported node: %s", node.Type)
	}
	return nil
}

type Template struct {
	Name     string
	Version  string
	RootNode Node `json:"root_node"`
}

func (tmp Template) Render(writer io.Writer, context ContextData, renderer *Renderer) error {
	return tmp.RootNode.render(writer, context, renderer)
}

type TemplateStore map[string]*Template

func (tmps TemplateStore) String() string {
	return ""
}

func (tmps TemplateStore) Set(data string) error {
	var decodedTemplate *Template

	// data should be JSON
	if err := json.Unmarshal([]byte(data), &decodedTemplate); err != nil {
		return err
	}

	tmps[decodedTemplate.Name] = decodedTemplate
	return nil
}

func (tmps TemplateStore) Type() string {
	return "template_store"
}

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
	return "file_template_loader"
}

type App struct {
	DefaultTemplateName string
	Templates           TemplateStore
	Filters             map[string]FilterFunc
	fileTemplateLoader  *FileTemplateLoader
}

func (app *App) Render(templateName string, contextData ContextData) (string, error) {
	selectedTemplate, templateExists := app.Templates[templateName]
	if !templateExists {
		return "", fmt.Errorf("template `%s` does not exist", templateName)
	}

	buf := &bytes.Buffer{}
	renderer := &Renderer{
		Templates: app.Templates,
		Filters:   app.Filters,
		Blocks:    make(map[string][]Node),
	}

	if err := selectedTemplate.Render(buf, contextData, renderer); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (rnd *App) RegisterTemplateJSON(rawTemplateData []byte) (*Template, error) {
	var decodedTemplate *Template
	if err := json.Unmarshal(rawTemplateData, &decodedTemplate); err != nil {
		return nil, err
	}
	rnd.Templates[decodedTemplate.Name] = decodedTemplate
	return decodedTemplate, nil
}

func (rnd *App) RegisterFilter(name string, filterFn FilterFunc) {
	rnd.Filters[name] = filterFn
}

var dataPath string
var app = &App{
	DefaultTemplateName: "default",
	Templates:           TemplateStore{},
	Filters:             map[string]FilterFunc{},
}

type Renderer struct {
	Templates TemplateStore
	Filters   map[string]FilterFunc
	Blocks    map[string][]Node
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

		var contextData ContextData
		if err := contextData.DecodeJSON(rawContextData); err != nil {
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
	app.fileTemplateLoader = &FileTemplateLoader{app.Templates}
	rootCmd.PersistentFlags().StringVar(&app.DefaultTemplateName, "name", "default", "Name of the template to be rendered.")
	rootCmd.PersistentFlags().Var(app.fileTemplateLoader, "template", "Path to the template.json file.")
	rootCmd.PersistentFlags().Var(&app.Templates, "templateData", "JSON data of the template.")
	rootCmd.PersistentFlags().StringVar(&dataPath, "data", "", "Path to the data.json file.")
}

func main() {
	app.RegisterFilter("upper", func(context ContextData, nodes ...Node) (interface{}, error) {
		if len(nodes) == 0 {
			return nil, fmt.Errorf("value is empty")
		}

		gotValue, err := nodes[0].Evaluate(context, nil)
		if err != nil {
			return nil, err
		}

		valueStr := fmt.Sprintf("%s", gotValue)
		return strings.ToUpper(valueStr), nil
	})

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
