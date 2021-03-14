<p align="center">
  <a href="https://fnrun.dev/" target="_blank" rel="noopener noreferrer">
    <img src="https://fnrun.dev/fnrun.png" width="400">
  </a>
</p>

# Welcome to fnrun
[![PkgGoDev](https://pkg.go.dev/badge/github.com/fnrun/fnrun)](https://pkg.go.dev/github.com/fnrun/fnrun)
[![Go Report Card](https://goreportcard.com/badge/fnrun/fnrun)](https://goreportcard.com/report/fnrun/fnrun)
[![codecov](https://codecov.io/gh/fnrun/fnrun/branch/main/graph/badge.svg)](https://codecov.io/gh/fnrun/fnrun)
[![Releases](https://img.shields.io/github/v/tag/fnrun/fnrun?include_prereleases&sort=semver)](https://github.com/fnrun/fnrun/releases)
[![LICENSE](https://img.shields.io/github/license/fnrun/fnrun.svg)](https://github.com/fnrun/fnrun/blob/main/LICENSE)

## What is fnrun?
The fnrun project provides a set of tools for building and running business
functions. It contains four main concepts: sources, middleware, fns, and 
runners.

A source is a component that provides inputs to a business function and will 
handle the outputs. Some common sources include a web server that will receive 
HTTP requests and return HTTP responses and a queue client that will read 
messages from a queue and mark the messages as handled only if the business 
function does not return an error.

Middleware are components that process inputs received from sources before the 
input is sent to a business function. They also have an opportunity to 
manipulate the output or errors from the business function before being returned
to the source. Middleware can be composed into a middleware pipeline, where data
is passed through each middleware in order until the end of the pipeline is 
reached.

Fns are components that represent business functions. They can actually _be_ 
business functions written in Go, but they are more commonly components that 
interact with an external business function. As an example, the CLI fn runs a 
business function as a CLI application and communicates with it over std 
streams.

A runner is a combination of a source, middleware, and an fn. The fnrun project 
provides a runner called fnrunner that provides common sources, middleware, and 
fns. However, it is also possible to create custom runners that include new 
components designed to meet the specific needs of your environment.

## License
fnrun is released under the [MIT License](LICENSE).