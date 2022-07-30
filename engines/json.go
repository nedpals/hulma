package engines

import "fmt"

type RawJson struct{}

func (engine RawJson) FileFormats() []string {
	return []string{"*.json"}
}

func (engine RawJson) Render(input []byte) (Node, error) {
	return nil, fmt.Errorf("stub only. this should be delegated to TemplateStore.set")
}

func (engine RawJson) RenderString(input string) (Node, error) {
	return nil, fmt.Errorf("stub only. this should be delegated to TemplateStore.set")
}
