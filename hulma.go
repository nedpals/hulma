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
		return nil, fmt.Errorf("unsupported node type `%s`", node.Type)
	}
}

func (node Node) render(writer io.Writer, context ContextData, renderer *Renderer) error {
	switch node.Type {
	case NODE_TYPE_CONTENT:
		writer.Write([]byte(node.Value))
	case NODE_TYPE_DISPLAY:
		if len(node.Children) == 0 {
			return fmt.Errorf("nothing to print")
		}

		gotValue, err := node.Children[0].Evaluate(context, renderer)
		if err != nil {
			return err
		}

		writer.Write([]byte(fmt.Sprintf("%s", gotValue)))
	case NODE_TYPE_SOURCE:
		for _, cn := range node.Children {
			if err := cn.render(writer, context, renderer); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported node: %s", node.Type)
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

type Renderer struct {
	DefaultTemplateName string
	Templates           map[string]*Template
	Filters             map[string]FilterFunc
}

func (rnd *Renderer) Render(templateName string, contextData ContextData) (string, error) {
	selectedTemplate, templateExists := rnd.Templates[templateName]
	if !templateExists {
		return "", fmt.Errorf("template `%s` does not exist", templateName)
	}

	buf := &bytes.Buffer{}
	if err := selectedTemplate.Render(buf, contextData, rnd); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (rnd *Renderer) RegisterTemplateJSON(rawTemplateData []byte) (*Template, error) {
	var decodedTemplate *Template
	if err := json.Unmarshal(rawTemplateData, &decodedTemplate); err != nil {
		return nil, err
	}
	rnd.Templates[decodedTemplate.Name] = decodedTemplate
	return decodedTemplate, nil
}

func (rnd *Renderer) RegisterFilter(name string, filterFn FilterFunc) {
	rnd.Filters[name] = filterFn
}

var templatePath string
var dataPath string
var renderer = &Renderer{
	DefaultTemplateName: "default",
	Templates:           map[string]*Template{},
	Filters:             map[string]FilterFunc{},
}

var rootCmd = &cobra.Command{
	Use:   "hulma",
	Short: "Hulma is an experimental template compiler.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fullTemplatePath, _ := filepath.Abs(templatePath)
		rawTemplateData, err := os.ReadFile(fullTemplatePath)
		if err != nil {
			return err
		}

		if _, err := renderer.RegisterTemplateJSON(rawTemplateData); err != nil {
			return err
		}

		fullDataPath, _ := filepath.Abs(dataPath)
		rawContextData, err := os.ReadFile(fullDataPath)
		if err != nil {
			return err
		}

		var contextData ContextData
		if err := contextData.DecodeJSON(rawContextData); err != nil {
			return err
		}

		data, err := renderer.Render(renderer.DefaultTemplateName, contextData)
		if err != nil {
			return err
		}

		fmt.Println(data)
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&renderer.DefaultTemplateName, "name", "default", "Name of the template to be rendered.")
	rootCmd.PersistentFlags().StringVar(&templatePath, "template", "", "Path to the template.json file.")
	rootCmd.PersistentFlags().StringVar(&dataPath, "data", "", "Path to the data.json file.")
}

func main() {
	renderer.RegisterFilter("upper", func(context ContextData, nodes ...Node) (interface{}, error) {
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
