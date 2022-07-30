package main

import (
	"fmt"

	types "github.com/nedpals/hulma/node_types"
)

type Node struct {
	Type     types.NodeType
	Value    string
	Children []Node
}

func (node Node) evaluateExpression(tmpl TemplateData) (any, error) {
	exprType := types.ExpressionNodeType(node.Type)
	switch exprType {
	case types.NODE_TYPE_CONTENT:
		return node.Value, nil
	case types.NODE_TYPE_VARIABLE:
		gotValue, varExists := tmpl.Context.Data[node.Value]
		if !varExists {
			return nil, fmt.Errorf("variable `%s` does not exist", node.Value)
		}
		return gotValue, nil
	case types.NODE_TYPE_FILTER:
		filterFn, filterExists := tmpl.Filters[node.Value]
		if !filterExists {
			return nil, fmt.Errorf("filter `%s` does not exist", node.Value)
		}

		evaluatedValue, err := node.Children[0].evaluateExpression(tmpl)
		if err != nil {
			return "", err
		}

		return filterFn(evaluatedValue)
	case types.NODE_TYPE_FUNCTION:
		functionFn, functionExists := tmpl.Functions[node.Value]
		if !functionExists {
			filterFn, filterExists := tmpl.Filters[node.Value]
			if filterExists && len(node.Children) == 1 {
				functionFn = filterFn.ToFunction()
			} else {
				return nil, fmt.Errorf("function `%s` does not exist", node.Value)
			}
		}

		evaluatedValue, err := node.collectFunctionArguments(tmpl)
		if err != nil {
			return "", err
		}

		return functionFn(evaluatedValue)
	default:
		return nil, fmt.Errorf("invalid expression type: %s", exprType)
	}
}

func (node Node) collectFunctionArguments(tmpl TemplateData) (any, error) {
	if node.Type != types.NodeType(types.NODE_TYPE_FUNCTION) {
		return nil, fmt.Errorf("node is not a function call")
	} else if len(node.Children) == 0 {
		return nil, nil
	}

	j := 0
	shouldBeMap := false
	keys := []string{}
	values := []any{}

	for _, child := range node.Children {
		fType := types.FunctionNodeType(child.Type)

		switch fType {
		case types.NODE_TYPE_FUNCTION_PARAMETER:
			if shouldBeMap {
				shouldBeMap = true
			}
			keys = append(keys, child.Value)
			j++
		case types.NODE_TYPE_FUNCTION_ARGUMENT:
			if len(child.Children) != 0 && len(child.Value) != 0 {
				return nil, fmt.Errorf("argument value should not be a content or an expression node at the same time")
			}

			if len(keys) > len(values) {
				keys = append(keys, fmt.Sprintf("%d", j))
			}

			if len(child.Children) != 0 {
				evaluatedValue, err := node.Children[0].evaluateExpression(tmpl)
				if err != nil {
					return nil, err
				}

				values = append(values, evaluatedValue)
			} else {
				values = append(values, child.Value)
			}
		default:
			return nil, fmt.Errorf("invalid filter type: %s", fType)
		}
	}

	if shouldBeMap {
		parameters := map[string]any{}
		for i := range values {
			parameters[keys[i]] = values[i]
		}
		return parameters, nil
	} else if len(values) == 1 {
		return values[0], nil
	}

	return values, nil
}

func renderBool(value any) bool {
	if value == nil {
		return false
	} else if boolVal, ok := value.(bool); ok {
		return boolVal
	} else if strVal, ok := value.(string); ok {
		return len(strVal) != 0
	} else {
		// TODO: support for maps and arrays
		return false
	}
}

func (node Node) evaluateStatement(tmpl TemplateData, renderer Renderer) error {
	stmtType := types.StatementNodeType(node.Type)
	switch stmtType {
	case types.NODE_TYPE_YIELD:
		if gotBlock, blockExists := tmpl.Context.Blocks[node.Value]; blockExists {
			return renderChildren(gotBlock, tmpl, renderer)
		} else {
			return renderChildren(node.Children, tmpl, renderer)
		}
	case types.NODE_TYPE_COND:
		if len(node.Children) < 2 || types.CondNodeType(node.Children[0].Type) != types.NODE_TYPE_COND_EXPR || len(node.Children[0].Children) != 1 {
			return fmt.Errorf("[1] invalid conditional node")
		} else if len(node.Children) == 3 && (types.StatementNodeType(node.Children[2].Type) != types.NODE_TYPE_COND && types.CondNodeType(node.Children[2].Type) != types.NODE_TYPE_COND_ALTER) {
			return fmt.Errorf("[2] invalid conditional node")
		}

		condExpr := node.Children[0]
		rawEvaluatedValue, err := condExpr.Children[0].evaluateExpression(tmpl)
		if err != nil {
			return err
		}

		evaluatedResult := renderBool(rawEvaluatedValue)
		if evaluatedResult {
			return renderChildren(node.Children[1].Children, tmpl, renderer)
		} else if len(node.Children) == 3 {
			// else-if or elif
			if types.StatementNodeType(node.Children[2].Type) == types.NODE_TYPE_COND {
				return node.Children[2].evaluateStatement(tmpl, renderer)
			} else {
				return renderChildren(node.Children[2].Children, tmpl, renderer)
			}
		} else if len(node.Children) == 4 {
			return renderChildren(node.Children[3].Children, tmpl, renderer)
		}
	case types.NODE_TYPE_LOOP:
		// for loop dissect
		// index 0 -

		// copy old context data
		oldContext := make(map[string]any)
		for k, v := range tmpl.Context.Data {
			oldContext[k] = v
		}

		// make a new special variable

		// context["$$i"] = len()

		// newlyAssigned := make([]string, 10)
		// for _, cn := range node.Children {
		// 	if err := cn.render(writer, context, renderer); err != nil {
		// 		return err
		// 	}
		// }

		// for k, v := range oldContext {
		// 	context[k] = v
		// }
	default:
		return fmt.Errorf("invalid expression type: %s", stmtType)
	}
	return nil
}

func (node Node) scanBlock(parentBlockName string, blocks map[string][]Node) error {
	if node.Type != types.NodeType(types.NODE_TYPE_BLOCK) {
		return fmt.Errorf("[scanBlocks] node not a block")
	} else if node.Value == parentBlockName {
		return fmt.Errorf("`%s` block should not be recursive", node.Value)
	} else {
		for _, cn := range node.Children {
			_ = cn.scanBlock(node.Value, blocks)
		}
		blocks[node.Value] = node.Children
		return nil
	}
}

func (node Node) evaluate(tmpl TemplateData, renderer Renderer) error {
	switch node.Type {
	case types.NODE_TYPE_SOURCE:
		for _, cn := range node.Children {
			if err := cn.evaluate(tmpl, renderer); err != nil {
				return err
			}
		}
	case types.NodeType(types.NODE_TYPE_CONTENT):
		return renderer.Write(node.Value)
	case types.NODE_TYPE_INCLUDE:
		return tmpl.Templates.Render(node.Value, tmpl, renderer)
	case types.NODE_TYPE_DISPLAY:
		if len(node.Children) != 1 {
			return fmt.Errorf("display node should have exactly one child")
		}
		gotValue, err := node.Children[0].evaluateExpression(tmpl)
		if err != nil {
			return err
		}
		return renderer.Write(gotValue)
	case types.NODE_TYPE_STATEMENT:
		if len(node.Children) != 1 {
			return fmt.Errorf("statement node should have exactly one child")
		}
		return node.Children[0].evaluateStatement(tmpl, renderer)
	case types.NODE_TYPE_BLOCK:
		return nil
	default:
		return fmt.Errorf("[evaluate] unsupported node: %s", node.Type)
	}
	return nil
}
