package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cobra"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

var (
	NODE_TYPE_SOURCE  = "node__source"
	NODE_TYPE_CONTENT = "node__content"
	NODE_TYPE_DISPLAY = "node__display"
)

type Node struct {
	Type     string
	Value    string
	Children []Node
}

func (node Node) render(writer io.Writer, context map[string]interface{}) error {
	switch node.Type {
	case NODE_TYPE_CONTENT:
		writer.Write([]byte(node.Value))
	case NODE_TYPE_DISPLAY:
		gotValue, varExists := context[node.Value]
		if !varExists {
			return fmt.Errorf("variable `%s` does not exist", node.Value)
		}
		writer.Write([]byte(fmt.Sprintf("%s", gotValue)))
	case NODE_TYPE_SOURCE:
	default:
		return fmt.Errorf("unsupported node: %s", node.Type)
	}
	for _, cn := range node.Children {
		if err := cn.render(writer, context); err != nil {
			return err
		}
	}
	return nil
}

type Template struct {
	Name     string
	Version  string
	RootNode Node `json:"root_node"`
}

func (tmp Template) Render(writer io.Writer, context map[string]interface{}) error {
	return tmp.RootNode.render(writer, context)
}

var templatePath string
var dataPath string

var rootCmd = &cobra.Command{
	Use:   "hulma",
	Short: "Hulma is an experimental template compiler.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fullTemplatePath, _ := filepath.Abs(templatePath)
		rawTemplateData, err := os.ReadFile(fullTemplatePath)
		if err != nil {
			return err
		}

		fullDataPath, _ := filepath.Abs(dataPath)
		rawContextData, err := os.ReadFile(fullDataPath)
		if err != nil {
			return err
		}

		var contextData map[string]interface{}
		if err := json.Unmarshal(rawContextData, &contextData); err != nil {
			return err
		}

		var decodedTemplate Template
		if err := json.Unmarshal(rawTemplateData, &decodedTemplate); err != nil {
			return err
		}

		buf := &bytes.Buffer{}
		if err := decodedTemplate.Render(buf, contextData); err != nil {
			return err
		}

		fmt.Println(buf.String())
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&templatePath, "template", "", "Path to the template.json file.")
	rootCmd.PersistentFlags().StringVar(&dataPath, "data", "", "Path to the data.json file.")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
