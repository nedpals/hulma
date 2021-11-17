# hulma
A template compiler written in Go. 

## Project Status
Experimental at this moment. This is mostly done in spare time and not production ready at this moment.

## Why
The common IR-based approach to code compilation and analysis such as LLVM and LSP has been proven to accelerate innovation in the industry and thus has ushered an era of programming tools and languages that focused on maintainability without worrying on optimizations or performance-related factors.

In the age of microservices and multi-language web applications, sharing templates across multiple templating languages can be somewhat *complex* yet interesting idea to tackle and in some cases this can be very helpful when you're in the middle of migrating to a new templating system or just being lazy rewriting them and want to both templates.

With this in mind, Hulma strives to be a flexible template compiler by using an IR that can be used as an output format by templating languages and can be used by all programming languages without any dedicated module/plugin required.

## How
It requires the source templating language to emit an IR first before using. This IR is then cached to Hulma which can be used later by user-facing applications or by other templates. When an app requests for a render,  Hulma will then inject the contextual data that the app provided into the cached template which will output the final product (an HTML file, a JSON file, or etc.)

## License
Licensed under MIT. See [LICENSE](LICENSE) for more details.

### (c) 2021 Ned Palacios