# Sherpa

Sherpa is a Go library for creating a [sherpa API](https://www.ueber.net/who/mjl/sherpa/).

This library makes it trivial to export Go functions as a sherpa API with an http.Handler.

Your API will automatically be documented: github.com/mjl-/sherpadoc reads your Go source, and exports function and type comments as API documentation.

See the [documentation](https://godoc.org/github.com/mjl-/sherpa).


## Examples

A public sherpa API: https://www.sherpadoc.org/#https://www.sherpadoc.org/example/

That web application is [sherpaweb](https://github.com/mjl-/sherpaweb). It shows documentation for any sherpa API but also includes an API called Example for demo purposes.

[Ding](https://github.com/irias/ding/) is a more elaborate web application built with this library.


# About

Written by Mechiel Lukkien, mechiel@ueber.net.
Bug fixes, patches, comments are welcome.
MIT-licensed, see LICENSE.


# todo

- sherpa: don't lowercase first char in function name by default, make it an option
- sherpa: add a toggle for enabling calls by GET request. turn off by default, people might be making requests with sensitive information in query strings...
- sherpa: in the collector, for FunctionCall, replace "serverError" with errorCode
- handler: write tests
- client: write tests
