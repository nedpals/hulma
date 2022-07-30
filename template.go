package main

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type FilterFunc func(value any) (any, error)

func (filterFn FilterFunc) ToFunction() FunctionFunc {
	return func(arguments any) (any, error) {
		if args, ok := arguments.([]any); ok {
			return filterFn(args[0])
		} else if args, ok := arguments.(map[string]any); ok {
			for _, v := range args {
				return filterFn(v)
			}
		} else {
			return filterFn(arguments)
		}
		return "", nil
	}
}

type FunctionFunc func(arguments any) (any, error)

type Template struct {
	Name     string
	Version  string
	blocks   map[string][]Node `json:"-"`
	RootNode Node              `json:"root_node"`
}

func (tmpl *Template) scanBlocks() {
	for _, cn := range tmpl.RootNode.Children {
		_ = cn.scanBlock("", tmpl.blocks)
	}
}

type ContextData struct {
	Blocks map[string][]Node
	Data   map[string]any `json:"data"`
}

type TemplateData struct {
	Context   ContextData
	Filters   map[string]FilterFunc
	Functions map[string]FunctionFunc // funky
	Templates TemplateStore
}

type TemplateStore map[string]*Template

func (tmps TemplateStore) String() string {
	return ""
}

func (tmps TemplateStore) Render(name string, data TemplateData, renderer Renderer) error {
	selectedTemplate, templateExists := tmps[name]
	if !templateExists {
		return fmt.Errorf("template `%s` does not exist", name)
	}

	if data.Context.Blocks == nil {
		withBlocksInData := TemplateData{
			Context: ContextData{
				Blocks: selectedTemplate.blocks,
				Data:   data.Context.Data,
			},
			Filters:   data.Filters,
			Functions: data.Functions,
			Templates: data.Templates,
		}
		return selectedTemplate.RootNode.evaluate(withBlocksInData, renderer)
	} else {
		return selectedTemplate.RootNode.evaluate(data, renderer)
	}
}

func newTemplate() *Template {
	return &Template{
		blocks: make(map[string][]Node),
	}
}

func (tmps TemplateStore) Add(template *Template) error {
	template.scanBlocks()
	tmps[template.Name] = template
	return nil
}

func (tmps TemplateStore) Set(data string) error {
	decodedTemplate := newTemplate()

	// data should be JSON
	if err := json.UnmarshalFromString(data, &decodedTemplate); err != nil {
		return err
	}

	return tmps.Add(decodedTemplate)
}

func (tmps TemplateStore) Type() string {
	return "template_json"
}
