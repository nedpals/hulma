package nodetypes

type NodeType string

const (
	NODE_TYPE_SOURCE    NodeType = "source"
	NODE_TYPE_DISPLAY   NodeType = "display"
	NODE_TYPE_STATEMENT NodeType = "statement"
	NODE_TYPE_INCLUDE   NodeType = "include"
	NODE_TYPE_BLOCK     NodeType = "block"
	NODE_TYPE_COMMENT   NodeType = "comment"
)

type ExpressionNodeType NodeType

const (
	NODE_TYPE_VARIABLE ExpressionNodeType = "variable"
	NODE_TYPE_FILTER   ExpressionNodeType = "filter"
	NODE_TYPE_CONTENT  ExpressionNodeType = "content"
	NODE_TYPE_FUNCTION ExpressionNodeType = "function"
)

type StatementNodeType NodeType

const (
	NODE_TYPE_COND   StatementNodeType = "cond"
	NODE_TYPE_YIELD  StatementNodeType = "yield"
	NODE_TYPE_LOOP   StatementNodeType = "loop"
	NODE_TYPE_ASSIGN StatementNodeType = "assign"
)

type FunctionNodeType NodeType

const (
	NODE_TYPE_FUNCTION_PARAMETER FunctionNodeType = "filter_parameter"
	NODE_TYPE_FUNCTION_ARGUMENT  FunctionNodeType = "filter_argument"
)

type CondNodeType NodeType

const (
	NODE_TYPE_COND_EXPR   CondNodeType = "cond_expression"
	NODE_TYPE_COND_CONSEQ CondNodeType = "cond_consequence"
	NODE_TYPE_COND_ALTER  CondNodeType = "cond_alternative"
)
