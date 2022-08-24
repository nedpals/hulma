package engines

import (
	"bytes"
	"fmt"
	"strings"
	"text/scanner"
	"unicode"

	nodetypes "github.com/nedpals/hulma/node_types"
	"github.com/zyedidia/generic/stack"
)

type TwigNodeType int

const (
	TWIG_ROOT TwigNodeType = iota
	TWIG_DISPLAY
	TWIG_STMT
	TWIG_RAW
	TWIG_IDENT
	TWIG_STRING
	TWIG_SELECTOR
	TWIG_FILTER
	TWIG_CALL
	TWIG_COMMENT
	TWIG_ERROR
)

type TwigNode struct {
	node_type TwigNodeType
	value     string
	children  []TwigNode
}

func (node TwigNode) Type() nodetypes.NodeType {
	switch node.node_type {
	case TWIG_ROOT:
		return nodetypes.NODE_TYPE_SOURCE
	case TWIG_RAW:
		return nodetypes.NodeType(nodetypes.NODE_TYPE_CONTENT)
	case TWIG_DISPLAY:
		return nodetypes.NODE_TYPE_DISPLAY
	case TWIG_STMT:
		return nodetypes.NODE_TYPE_STATEMENT
	case TWIG_IDENT:
		return nodetypes.NodeType(nodetypes.NODE_TYPE_VARIABLE)
	case TWIG_STRING:
		return nodetypes.NodeType(nodetypes.NODE_TYPE_CONTENT)
	case TWIG_SELECTOR:
		return nodetypes.NodeType(nodetypes.NODE_TYPE_VARIABLE)
	case TWIG_FILTER:
		return nodetypes.NodeType(nodetypes.NODE_TYPE_FILTER)
	case TWIG_CALL:
		return nodetypes.NodeType(nodetypes.NODE_TYPE_FUNCTION)
	case TWIG_COMMENT:
		return nodetypes.NodeType(nodetypes.NODE_TYPE_COMMENT)
	default:
		return nodetypes.NodeType(nodetypes.NODE_TYPE_CONTENT) // ??
	}
}

func (node TwigNode) Value() string {
	return node.value
}

func (node TwigNode) Children() []Node {
	return ConvertChildren(node.children)
}

type Twig struct{}

func (engine Twig) FileFormats() []string {
	return []string{"*.twig", "*.html"}
}

func (engine Twig) Render(input []byte) (Node, error) {
	sc := TwigScanner{
		scanner: &scanner.Scanner{
			IsIdentRune: func(ch rune, i int) bool {
				return ch == '_' || unicode.IsLetter(ch) || unicode.IsDigit(ch) && i > 0
			},
		},
		tokenBuilder: &strings.Builder{},
		stack:        stack.New[TwigNodeType](),
	}

	sc.scanner.Init(bytes.NewBuffer(input))
	return sc.Scan()
}

func (engine Twig) RenderString(input string) (Node, error) { return engine.Render([]byte(input)) }

type TwigScanner struct {
	scanner      *scanner.Scanner
	tokenBuilder *strings.Builder
	stack        *stack.Stack[TwigNodeType]
}

func (sc TwigScanner) skipWhitespace(enabled bool) {
	if enabled {
		sc.scanner.Whitespace = scanner.GoWhitespace
	} else {
		sc.scanner.Whitespace = 0
	}
}

func (sc TwigScanner) Scan() (TwigNode, error) {
	sc.scanner.Mode = 0
	sc.skipWhitespace(false)

	root := TwigNode{
		node_type: TWIG_ROOT,
		children:  []TwigNode{},
	}

	for tok := sc.scanner.Scan(); tok != scanner.EOF; tok = sc.scanner.Scan() {
		if peek := sc.scanner.Peek(); tok == '{' && (peek == '{' || peek == '%' || peek == '#') {
			if sc.tokenBuilder.Len() != 0 {
				root.children = append(root.children, TwigNode{
					node_type: TWIG_RAW,
					value:     sc.tokenBuilder.String(),
				})

				sc.tokenBuilder.Reset()
			}

			sc.scanner.Next()
			sc.skipWhitespace(true)

			switch peek {
			case '{':
				// sc.stack.Push(TWIG_DISPLAY)

				expr, err := sc.scanExpression()
				if err != nil {
					return expr, err
				}

				finalExpr, err := sc.scanFilter(expr)
				if err != nil {
					return finalExpr, err
				}

				tok = sc.scanner.Next()
				if tok == '}' && sc.scanner.Peek() == '}' {
					sc.scanner.Next()
					root.children = append(root.children, TwigNode{
						node_type: TWIG_DISPLAY,
						children:  []TwigNode{finalExpr},
					})
				} else {
					return sc.error(fmt.Errorf("display tag not closed (tok = %c, peek = %c)", tok, sc.scanner.Peek()))
				}
			case '%':
				// sc.stack.Push(TWIG_STMT)
				return sc.error(fmt.Errorf("statement block tag not yet implemented"))
			case '#':
				commentNode, err := sc.scanComments()
				if err != nil {
					return sc.error(err)
				}

				root.children = append(root.children, commentNode)
			}

			sc.skipWhitespace(false)
		} else {
			sc.tokenBuilder.WriteRune(tok)
		}
	}

	if sc.tokenBuilder.Len() != 0 {
		root.children = append(root.children, TwigNode{
			node_type: TWIG_RAW,
			value:     sc.tokenBuilder.String(),
		})

		sc.tokenBuilder.Reset()
	}

	return root, nil
}

func (sc TwigScanner) scanComments() (TwigNode, error) {
	defer sc.tokenBuilder.Reset()
	sc.skipWhitespace(false)

	for {
		tok := sc.scanner.Scan()
		if tok == '#' && sc.scanner.Peek() == '}' {
			sc.scanner.Next()
			break
		}
		sc.tokenBuilder.WriteRune(tok)
	}

	return TwigNode{
		node_type: TWIG_COMMENT,
		value:     sc.tokenBuilder.String(),
	}, nil
}

func (sc TwigScanner) scanExpression() (TwigNode, error) {
	tok := sc.scanner.Scan()
	if sc.scanner.IsIdentRune(tok, 0) {
		sc.tokenBuilder.WriteRune(tok)
		return sc.scanExpressionFromType(sc.scanIdent(0))
	} else {
		switch tok {
		case '"', '\'':
			expected := tok
			sc.skipWhitespace(false)
			defer sc.tokenBuilder.Reset()
			defer sc.skipWhitespace(true)

			for {
				tok = sc.scanner.Next()
				if tok == expected {
					break
				} else if tok == scanner.EOF {
					return sc.error(fmt.Errorf("reached eof"))
				}
				sc.tokenBuilder.WriteRune(tok)
			}

			return TwigNode{
				node_type: TWIG_STRING,
				value:     sc.tokenBuilder.String(),
			}, nil
		default:
			return sc.error(fmt.Errorf("unknown token: %c", tok))
		}
	}
}

func (sc TwigScanner) scanIdent(idx int) TwigNode {
	tok := sc.scanner.Next()
	defer sc.tokenBuilder.Reset()

	for i := idx + 1; sc.scanner.IsIdentRune(tok, i); i++ {
		sc.tokenBuilder.WriteRune(tok)
		tok = sc.scanner.Next()
	}

	return TwigNode{
		node_type: TWIG_IDENT,
		value:     sc.tokenBuilder.String(),
	}
}

func (sc TwigScanner) error(err error) (TwigNode, error) {
	return TwigNode{node_type: TWIG_ERROR}, err
}

func (sc TwigScanner) scanFilter(node TwigNode) (TwigNode, error) {
	tok := sc.scanner.Peek()

	if tok == '|' {
		sc.scanner.Next()
		identOrExpr, err := sc.scanExpression()
		if err != nil {
			return identOrExpr, err
		}

		return sc.scanFilter(TwigNode{
			node_type: TWIG_FILTER,
			value:     identOrExpr.value,
			children:  []TwigNode{node},
		})
	}
	return node, nil
}

func (sc TwigScanner) scanExpressionFromType(node TwigNode) (TwigNode, error) {
	tok := sc.scanner.Peek()
	if node.node_type == TWIG_IDENT {
		switch tok {
		case '(':
			sc.scanner.Next()
			expr, err := sc.scanExpression()
			if err != nil {
				return sc.error(err)
			}

			if tok = sc.scanner.Scan(); tok != '(' {
				return sc.error(fmt.Errorf("unknown token: %c", tok))
			}

			return TwigNode{
				node_type: TWIG_CALL,
				value:     node.value,
				children:  []TwigNode{expr},
			}, nil
		case '.':
			identOrExpr, err := sc.scanExpressionFromType(sc.scanIdent(0))
			if err != nil {
				return sc.error(err)
			}

			return TwigNode{
				node_type: TWIG_SELECTOR,
				children: []TwigNode{
					node,
					identOrExpr,
				},
			}, nil
		}
	}

	return node, nil
}
