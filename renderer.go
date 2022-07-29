package main

import (
	"fmt"
	"io"
)

type Renderer interface {
	Write(value any) error
}

func renderChildren(children []Node, tmpl TemplateData, renderer Renderer) error {
	for _, cn := range children {
		if err := cn.evaluate(tmpl, renderer); err != nil {
			return err
		}
	}
	return nil
}

type simpleRenderer struct {
	writer io.Writer
}

func (wr *simpleRenderer) Write(value any) error {
	_, err := wr.writer.Write([]byte(fmt.Sprintf("%s", value)))
	return err
}
