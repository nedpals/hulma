package engines

import (
	"fmt"
	"path/filepath"
	"strings"

	nodetypes "github.com/nedpals/hulma/node_types"
)

type Engines []Engine

type Engine interface {
	FileFormats() []string
	Render(input []byte) (Node, error)
	RenderString(input string) (Node, error)
}

func (engs Engines) MatchEngine(rawFileName string) (Engine, string, error) {
	fileName := filepath.Base(rawFileName)

	for _, eng := range engs {
		for _, format := range eng.FileFormats() {
			if matched, err := filepath.Match(format, fileName); err == nil && matched {
				return eng, strings.TrimSuffix(fileName, filepath.Ext(fileName)), nil
			}
		}
	}

	return nil, "", fmt.Errorf("engine not found")
}

type Node interface {
	Type() nodetypes.NodeType
	Value() string
	Children() []Node
}
