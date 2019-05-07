sherpadoc - documentation for sherpa API's

Go package containing type defintions for sherpa documentation for encoding to and decoding from json.
Also contains the sherpadoc command reads Go code and writes sherpadoc JSON.

Use together with the sherpa library, github.com/mjl-/sherpa.
Read more about sherpa at https://www.ueber.net/who/mjl/sherpa/

# About

Written by Mechiel Lukkien, mechiel@ueber.net.
Bug fixes, patches, comments are welcome.
MIT-licensed, see LICENSE.
cmd/sherpadoc/gopath.go originates from the Go project, see LICENSE-go for its BSD-style license.

# todo

- raise error for ints & strings without constants defined.
- when reading types from other packages (imported packages), we only look at GOPATH. vendor and modules are not taking into account, but we should.
- better error messages and error handling, stricter parsing
- support plain iota enums? currently only simple literals are supported for enums.
- support complete expressions for enum consts?
- find out which go constructs people want to use that aren't yet implemented by sherpadoc
- error or warn when omitempty is set for non-pointer?
- write tests
