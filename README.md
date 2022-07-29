# hulma
A template compiler written in Go. 

## Project Status
Experimental at this moment. This is mostly done in spare time and not production ready.

## Why
The common IR-based approach to code compilation and analysis such as LLVM and LSP has been proven to accelerate innovation in the industry and thus has ushered an era of programming tools and languages that focused on maintainability without worrying on optimizations or performance-related factors.

On the other hand, sharing templates across multiple templating languages can be somewhat *complex* yet interesting idea to tackle in the age of microservices and multi-language web applications which in some cases can be very helpful when you're in the middle of migrating to a new templating system or just being lazy rewriting them and want to keep both templates.

With this in mind, Hulma strives to be a flexible template compiler by using an IR that can be used as an output format by templating languages and can be used by all programming languages without any dedicated module/plugin required.

## How
It requires the source templating language to emit an IR first before using. This IR is then cached to Hulma which can be used later by user-facing applications or by other templates. When an app requests for a render,  Hulma will then inject the contextual data that the app provided into the cached template which will output the final product (an HTML file, a JSON file, or etc.)

## IR
The immediate representation is just a simple JSON format which can be hand written and has a tree-like structure. 

### Template
The "template", which is the root JSON object you are seeing here, consists of a `name`, a `version`, and a `root_node` with a `source` node as the starting (or the root) node.

```json
{
    "name": "page",
    "version": "1.0",
    "root_node": {
        "type": "source",
        "children": [
            {
                "type": "content",
                "value": "This is a page!"
            }
        ]
    }
}
```

### Node
A node object consists of `type`, `value`, and `children` which is an array of nodes. The last two can be optional depending on the node type.
```json
{
    "type": "source",
    "children": [
        {
            "type": "content",
            "value": "This is a page!"
        }
    ]
}
```

### Node Types
Currently, there are eight node types that can be used.

|Type|Value|Children|Notes/Description|
|----|-----|--------|-----|
|`source`|❌|✅|The source node. Can be only used as a root node.|
|`content`|✅|❌|The content node. Used to display static plain text content.|
|`display`|❌|✅|The display node. Used to display/output expressions or identifiers such as variables.|
|`variable`|✅|❌|The variable node. Used to reference a variable from the given context data.|
|`filter`|✅|✅|The filter node. Applies a filter to the child.|
|`include`|✅|❌|The include node. Used to include other templates into the current template.|
|`block`|✅|✅|The block node. Used for inserting custom content into a specific content block. There must be an equivalent `yield` block in order to display the content.|
|`yield`|✅|✅|The yield node. Used for displaying a specific content block. If no custom content block was found, it can supply a default content as a fallback.|

## Context Data
The context data is still a JSON object in which the keys are the variables and the values are the contents of the variables.

```json
{
    "foo": "bar"
}
```

## Notes
- Loops and ~~conditionals~~ are not supported at this moment.
- Complex expressions such as index expressions, selectors, binary, and unary are also planned.
- There will be support for a client-server mode (in TCP) which will make Hulma utilized to it's full potential.
- Although my aim is to have stable support, adding tests are not my top priority right now.
- There are no reference implementations in the "front-end" side at this moment.
- The error tracking is limited and does not support position tracking because of the design limitations imposed to the IR.

## License
Licensed under MIT. See [LICENSE](LICENSE) for more details.

### (c) 2021-2022 Ned Palacios